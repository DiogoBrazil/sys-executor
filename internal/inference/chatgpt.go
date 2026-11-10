package inference

// MotorChatGPT: inferência pela Responses API pública da OpenAI com o plano
// ChatGPT do usuário (Sign in with ChatGPT; V4 §48.7, TRV-IA-015).
//
// O corpo é montado à mão, com só os campos aceitos nesse modo: model,
// instructions, input, store=false, stream=true, text.format (json_schema
// strict, aceito com os schemas do app na PoC de 09/11/2026) e, se
// configurado, reasoning. Campos que o modo recusa ou manda omitir
// (max_output_tokens, temperature, top_p, metadata, truncation, user,
// previous_response_id…) nunca existem no tipo do corpo. A resposta só vale
// com o evento response.completed.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// OrigemAPIOpenAI é o único destino da inferência do ChatGPT (TRV-SEG-012).
const OrigemAPIOpenAI = "https://api.openai.com"

const (
	caminhoRespostas = "/v1/responses"
	caminhoModelos   = "/v1/models"
	// inatividadePadrao: prazo máximo sem nenhum byte do fluxo.
	inatividadePadrao = 3 * time.Minute
	tentativasChatGPT = 3
)

// FonteToken entrega o token de acesso da conta conectada (só em memória).
// Renovar troca o token depois de um 401 (uma vez por requisição).
type FonteToken interface {
	Token(ctx context.Context) (string, error)
	Renovar(ctx context.Context) (string, error)
}

// ConfigChatGPT configura o motor.
type ConfigChatGPT struct {
	Modelo   string // slug da lista da conta
	Tokens   FonteToken
	Contexto int    // janela de trabalho; 0 = ContextoPadrao (limitada pelo modelo)
	Esforco  string // reasoning.effort ("" = padrão do modelo)
	// Inspecionar: barreira de privacidade, camada 2 (TRV-SEG-013).
	Inspecionar func([]byte) error
	// Inatividade: prazo sem bytes do fluxo; 0 = inatividadePadrao.
	Inatividade time.Duration

	// Somente testes do pacote: servidor falso e espera instantânea.
	origem string
	espera func(context.Context, time.Duration) error
}

// MotorChatGPT implementa Motor.
type MotorChatGPT struct {
	cfg      ConfigChatGPT
	cli      *http.Client
	contexto int

	mu  sync.Mutex
	uso Uso
}

var reSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9.\-]{0,99}$`)

// NovoMotorChatGPT prepara o motor (sem rede: a conta e o modelo já foram
// conferidos pela fonte).
func NovoMotorChatGPT(cfg ConfigChatGPT) (*MotorChatGPT, error) {
	if !reSlug.MatchString(cfg.Modelo) {
		return nil, erros.Com(erros.IAModeloIndisponivel, errors.New("identificador de modelo inválido"))
	}
	if cfg.Tokens == nil {
		return nil, erros.Novo(erros.IAContaDesconectada)
	}
	origem := cfg.origem
	if origem == "" {
		origem = OrigemAPIOpenAI
	}
	cli, err := NovoClienteRestrito(origem, cfg.Inspecionar)
	if err != nil {
		return nil, erros.Com(erros.InternoInesperado, err)
	}
	ctxt := cfg.Contexto
	if ctxt <= 0 {
		ctxt = ContextoPadrao
	}
	return &MotorChatGPT{cfg: cfg, cli: cli, contexto: ctxt,
		uso: Uso{Modelo: cfg.Modelo}}, nil
}

func (m *MotorChatGPT) origem() string {
	if m.cfg.origem != "" {
		return m.cfg.origem
	}
	return OrigemAPIOpenAI
}

// Contexto devolve a janela de trabalho.
func (m *MotorChatGPT) Contexto() int { return m.contexto }

// ContarTokens estima localmente (sem rede).
func (m *MotorChatGPT) ContarTokens(_ context.Context, t string) (int, error) {
	return EstimarTokens(t), nil
}

// Uso acumulado da tarefa (só memória).
func (m *MotorChatGPT) Uso() Uso {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.uso
}

// Fechar libera as conexões ociosas.
func (m *MotorChatGPT) Fechar() { m.cli.CloseIdleConnections() }

// corpoChatGPT é o corpo da requisição: só os campos permitidos.
type corpoChatGPT struct {
	Model        string         `json:"model"`
	Instructions string         `json:"instructions,omitempty"`
	Input        []itemEntrada  `json:"input"`
	Store        bool           `json:"store"`
	Stream       bool           `json:"stream"`
	Text         *textoFormato  `json:"text,omitempty"`
	Reasoning    *raciocinioCfg `json:"reasoning,omitempty"`
}

type itemEntrada struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type parteSaida struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type textoFormato struct {
	Format formatoSchema `json:"format"`
}

type formatoSchema struct {
	Type   string          `json:"type"`
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict"`
}

type raciocinioCfg struct {
	Effort string `json:"effort"`
}

var reNomeSchema = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func (m *MotorChatGPT) montar(p Pedido) ([]byte, error) {
	c := corpoChatGPT{Model: m.cfg.Modelo, Instructions: p.Sistema, Store: false, Stream: true}
	c.Input = append(c.Input, itemEntrada{Role: "user", Content: p.Usuario})
	for _, h := range p.Historico {
		conteudo := h.Conteudo
		if strings.TrimSpace(conteudo) == "" {
			conteudo = respostaVazia
		}
		switch h.Papel {
		case "assistant", "model":
			c.Input = append(c.Input, itemEntrada{Role: "assistant", Content: []parteSaida{{Type: "output_text", Text: conteudo}}})
		default:
			c.Input = append(c.Input, itemEntrada{Role: "user", Content: conteudo})
		}
	}
	if len(p.Schema) > 0 {
		nome := reNomeSchema.ReplaceAllString(p.NomeSchema, "_")
		if nome == "" {
			nome = "resposta"
		}
		if len(nome) > 64 {
			nome = nome[:64]
		}
		c.Text = &textoFormato{Format: formatoSchema{Type: "json_schema", Name: nome, Schema: p.Schema, Strict: true}}
	}
	if m.cfg.Esforco != "" {
		c.Reasoning = &raciocinioCfg{Effort: m.cfg.Esforco}
	}
	return json.Marshal(c)
}

// Gerar envia o pedido e consome o fluxo até response.completed.
func (m *MotorChatGPT) Gerar(ctx context.Context, p Pedido) (Resposta, error) {
	corpo, err := m.montar(p)
	if err != nil {
		return Resposta{}, erros.Com(erros.InternoInesperado, err)
	}
	ini := time.Now()
	var r Resposta
	renovou := false
	for tentativa := 0; ; tentativa++ {
		r, err = m.umaVez(ctx, corpo)
		if err == nil {
			break
		}
		var f falhaChatGPT
		if !errors.As(err, &f) {
			m.anotarFalha()
			return Resposta{}, err
		}
		if f.renovar && !renovou {
			renovou = true
			if _, e := m.cfg.Tokens.Renovar(ctx); e != nil {
				m.anotarFalha()
				return Resposta{}, e
			}
			continue
		}
		if !f.repetir || tentativa+1 >= tentativasChatGPT {
			m.anotarFalha()
			return Resposta{}, f.erro
		}
		if e := m.esperar(ctx, tentativa); e != nil {
			m.anotarFalha()
			return Resposta{}, f.erro
		}
	}
	r.Duracao = time.Since(ini)
	m.mu.Lock()
	m.uso.Chamadas++
	m.uso.TokensEntrada += r.TokensEntrada
	m.uso.TokensSaida += r.TokensSaida
	m.uso.TokensPensamento += r.TokensPensamento
	m.uso.Duracao += r.Duracao
	m.mu.Unlock()
	return r, nil
}

func (m *MotorChatGPT) anotarFalha() {
	m.mu.Lock()
	m.uso.Chamadas++
	m.uso.Falhas++
	m.mu.Unlock()
}

func (m *MotorChatGPT) esperar(ctx context.Context, tentativa int) error {
	d := time.Duration(1<<tentativa)*time.Second + time.Duration(rand.Int64N(int64(time.Second)))
	if m.cfg.espera != nil {
		return m.cfg.espera(ctx, d)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// umaVez faz uma requisição e lê o fluxo SSE.
func (m *MotorChatGPT) umaVez(ctx context.Context, corpo []byte) (Resposta, error) {
	tok, err := m.cfg.Tokens.Token(ctx)
	if err != nil {
		return Resposta{}, err
	}
	ctx, cancelar := context.WithCancel(ctx)
	defer cancelar()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.origem()+caminhoRespostas, bytes.NewReader(corpo))
	if err != nil {
		return Resposta{}, erros.Com(erros.InternoInesperado, err)
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	inat := m.cfg.Inatividade
	if inat <= 0 {
		inat = inatividadePadrao
	}
	vivo := make(chan struct{}, 1)
	porInatividade := make(chan struct{})
	go vigiar(ctx, cancelar, vivo, inat, porInatividade)

	resp, err := m.cli.Do(req)
	if err != nil {
		return Resposta{}, classificarRede(ctx, err, porInatividade)
	}
	defer resp.Body.Close()
	reqID := idRequisicao(resp.Header)
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return Resposta{}, classificarHTTP(resp.StatusCode, b, reqID)
	}
	return lerFluxo(ctx, resp.Body, vivo, porInatividade, reqID)
}

func vigiar(ctx context.Context, cancelar func(), vivo <-chan struct{}, inat time.Duration, disparou chan<- struct{}) {
	t := time.NewTimer(inat)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-vivo:
			if !t.Stop() {
				select {
				case <-t.C:
				default:
				}
			}
			t.Reset(inat)
		case <-t.C:
			close(disparou)
			cancelar()
			return
		}
	}
}

func idRequisicao(h http.Header) string {
	for _, k := range []string{"x-oai-request-id", "x-request-id"} {
		if v := h.Get(k); v != "" {
			return v
		}
	}
	return ""
}

// lerFluxo consome o SSE. Só response.completed vale como sucesso; o texto
// é o dos deltas (conferido com o montado a partir do evento final).
func lerFluxo(ctx context.Context, corpo io.Reader, vivo chan<- struct{}, porInatividade <-chan struct{}, reqID string) (Resposta, error) {
	sc := bufio.NewScanner(corpo)
	sc.Buffer(make([]byte, 0, 64<<10), 32<<20)
	var dados strings.Builder
	var texto strings.Builder
	var r Resposta
	concluido := false
	var falha error
	tratar := func(d string) {
		if d == "" || d == "[DONE]" || falha != nil || concluido {
			return
		}
		var ev eventoSSE
		if json.Unmarshal([]byte(d), &ev) != nil {
			return
		}
		switch ev.Type {
		case "response.output_text.delta":
			texto.WriteString(ev.Delta)
		case "response.completed":
			concluido = true
			r.TokensEntrada, r.TokensSaida, r.TokensPensamento = ev.Response.uso()
			if final := ev.Response.textoFinal(); final != "" && texto.Len() == 0 {
				texto.WriteString(final)
			}
		case "response.incomplete":
			r.TokensEntrada, r.TokensSaida, r.TokensPensamento = ev.Response.uso()
			motivo := ev.Response.IncompleteDetails.Reason
			if motivo == "content_filter" {
				falha = erros.Com(erros.IARespostaBloqueada, fmt.Errorf("chatgpt: resposta incompleta (%s) req=%s", motivo, reqID))
				return
			}
			// Corte por tamanho: o validador pede concisão (como o MAX_TOKENS do Gemini).
			r.Truncada = true
			concluido = true
		case "response.failed":
			falha = classificarCodigo(ev.Response.Error.Code, ev.Response.Error.Message, "", reqID, true)
		case "error":
			falha = classificarCodigo(ev.codigo(), ev.Message, ev.Param, reqID, true)
		}
	}
	for sc.Scan() {
		select {
		case vivo <- struct{}{}:
		default:
		}
		linha := sc.Text()
		switch {
		case linha == "":
			tratar(dados.String())
			dados.Reset()
		case strings.HasPrefix(linha, "data:"):
			if dados.Len() > 0 {
				dados.WriteByte('\n')
			}
			dados.WriteString(strings.TrimPrefix(strings.TrimPrefix(linha, "data:"), " "))
		}
	}
	tratar(dados.String())
	if falha != nil {
		return Resposta{}, falha
	}
	if err := sc.Err(); err != nil && !concluido {
		return Resposta{}, classificarRede(ctx, err, porInatividade)
	}
	if !concluido {
		return Resposta{}, falhaChatGPT{erro: erros.Com(erros.IARespostaIncompleta,
			fmt.Errorf("chatgpt: fluxo encerrado sem response.completed req=%s", reqID)), repetir: true}
	}
	r.Conteudo = []byte(texto.String())
	if !r.Truncada && len(bytes.TrimSpace(r.Conteudo)) == 0 {
		return Resposta{}, erros.Com(erros.IARespostaIncompleta, fmt.Errorf("chatgpt: resposta vazia req=%s", reqID))
	}
	return r, nil
}

type eventoSSE struct {
	Type     string        `json:"type"`
	Delta    string        `json:"delta"`
	Code     any           `json:"code"`
	Message  string        `json:"message"`
	Param    string        `json:"param"`
	Response respostaFinal `json:"response"`
}

func (e eventoSSE) codigo() string {
	if e.Code == nil {
		return ""
	}
	return fmt.Sprint(e.Code)
}

type respostaFinal struct {
	Status            string `json:"status"`
	IncompleteDetails struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Usage struct {
		InputTokens         int `json:"input_tokens"`
		OutputTokens        int `json:"output_tokens"`
		OutputTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"output_tokens_details"`
	} `json:"usage"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func (r respostaFinal) uso() (entrada, saida, pensamento int) {
	return r.Usage.InputTokens, r.Usage.OutputTokens, r.Usage.OutputTokensDetails.ReasoningTokens
}

func (r respostaFinal) textoFinal() string {
	var b strings.Builder
	for _, o := range r.Output {
		if o.Type != "message" {
			continue
		}
		for _, c := range o.Content {
			if c.Type == "output_text" {
				b.WriteString(c.Text)
			}
		}
	}
	return b.String()
}

package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// Parâmetros técnicos (V4 §47.5: calibrados no benchmark sintético).
const (
	// ContextoPadrao é a janela de trabalho usada no orçamento do pedido e no
	// fatiamento. Os modelos aceitam muito mais, mas "contexto maior não
	// substitui recuperação documental estruturada" (V4 §38.3, mantido) e a
	// extração devolve no máximo 25 pontos por bloco.
	ContextoPadrao = 32768
	// TimeoutPadrao por requisição (o cancelamento da tarefa interrompe antes).
	TimeoutPadrao = 5 * time.Minute
	// folgaPensamento: tokens de saída somados à reserva da resposta quando o
	// modelo raciocina e o limite do modelo não é conhecido. Com o limite
	// conhecido, usa-se o limite do modelo: o raciocínio consome a mesma
	// cota e, esgotada, a resposta volta cortada e sem texto. Só os tokens
	// efetivamente gerados são cobrados.
	folgaPensamento = 24576
	// tentativasTransporte: total de tentativas para falhas transitórias
	// (429 por minuto, 5xx, rede). Distintas dos reenvios do validador.
	tentativasTransporte = 3
)

var reModelo = regexp.MustCompile(`^[a-z0-9][a-z0-9.\-]{0,99}$`)

// ConfigGemini configura o cliente. A chave é usada só para criar o cliente
// do SDK e nunca é registrada nem devolvida em erros.
type ConfigGemini struct {
	Chave  []byte
	Modelo string // ex.: "gemini-3.6-flash" (sem o prefixo "models/")
	// Contexto: janela de trabalho; 0 = ContextoPadrao (limitada pelo modelo).
	Contexto int
	// Timeout por requisição; 0 = TimeoutPadrao.
	Timeout time.Duration
	// Pensamento: nível de raciocínio ("MINIMAL", "LOW", "MEDIUM", "HIGH");
	// vazio = padrão do modelo (o suporte varia por modelo).
	Pensamento string

	// EsperaMax limita cada espera entre tentativas (0 = sem limite além do
	// prazo do contexto). As operações da tela de configurações usam um valor
	// curto: o usuário vê o erro em vez de esperar.
	EsperaMax time.Duration

	// Inspecionar recebe o corpo exato de cada requisição antes de sair
	// (barreira de privacidade, camada 2; TRV-SEG-013). Erro = não envia.
	Inspecionar func(corpo []byte) error

	// Somente testes do pacote: servidor falso e espera instantânea.
	baseURL string
	espera  func(context.Context, time.Duration) error
}

func (c ConfigGemini) base() string {
	if c.baseURL != "" {
		return c.baseURL
	}
	return baseURLGemini
}

// novoCliente cria o cliente do SDK com a chave e a base explícitas e o
// transporte restrito (o SDK leria GEMINI_API_KEY/GOOGLE_GEMINI_BASE_URL do
// ambiente se não fossem informados).
func novoCliente(ctx context.Context, c ConfigGemini) (*genai.Client, *http.Client, error) {
	if len(c.Chave) == 0 {
		return nil, nil, erros.Novo(erros.IAChaveAusente)
	}
	hc, err := novoClienteHTTP(c.base(), c.Inspecionar)
	if err != nil {
		return nil, nil, erros.Com(erros.InternoInesperado, err)
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = TimeoutPadrao
	}
	cli, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:     string(c.Chave),
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: hc,
		HTTPOptions: genai.HTTPOptions{
			BaseURL: c.base(),
			Timeout: &timeout,
		},
	})
	if err != nil {
		return nil, nil, erros.Com(erros.InternoInesperado, sanear(err, string(c.Chave)))
	}
	return cli, hc, nil
}

// Uso acumulado de uma tarefa (somente memória; nunca gravado — V4 §33).
type Uso struct {
	Modelo           string
	Chamadas         int
	Falhas           int
	TokensEntrada    int
	TokensSaida      int
	TokensPensamento int
	Duracao          time.Duration
}

// MotorGemini implementa Motor com a Gemini API (TRV-IA-011, TRV-IA-012,
// TRV-SEG-012): sem ferramentas, store=false, resposta íntegra ou erro.
type MotorGemini struct {
	cli        *genai.Client
	hc         *http.Client
	chave      string
	modelo     string
	contexto   int
	saidaMax   int
	pensa      bool
	pensamento string
	espera     func(context.Context, time.Duration) error
	esperaMax  time.Duration

	mu  sync.Mutex
	uso Uso
}

// NovoMotorGemini cria o motor e confere o modelo no provedor (Models.Get,
// sem gerar tokens): modelo inexistente → IA-004; chave recusada → IA-002.
func NovoMotorGemini(ctx context.Context, c ConfigGemini) (*MotorGemini, error) {
	if !reModelo.MatchString(c.Modelo) {
		return nil, erros.Com(erros.IAModeloIndisponivel, fmt.Errorf("identificador de modelo inválido: %q", c.Modelo))
	}
	cli, hc, err := novoCliente(ctx, c)
	if err != nil {
		return nil, err
	}
	m := &MotorGemini{cli: cli, hc: hc, chave: string(c.Chave), modelo: c.Modelo, pensamento: c.Pensamento, espera: c.espera, esperaMax: c.EsperaMax}
	if m.espera == nil {
		m.espera = esperar
	}
	m.uso.Modelo = c.Modelo
	var info *genai.Model
	err = m.repetir(ctx, func(ctx context.Context) error {
		var e error
		info, e = cli.Models.Get(ctx, c.Modelo, nil)
		return e
	})
	if err != nil {
		m.Fechar()
		return nil, err
	}
	m.contexto = c.Contexto
	if m.contexto <= 0 {
		m.contexto = ContextoPadrao
	}
	if info.InputTokenLimit > 0 && int(info.InputTokenLimit) < m.contexto {
		m.contexto = int(info.InputTokenLimit)
	}
	m.saidaMax = int(info.OutputTokenLimit)
	m.pensa = info.Thinking
	return m, nil
}

// Contexto é a janela de trabalho (orçamento e fatiamento).
func (m *MotorGemini) Contexto() int { return m.contexto }

// ContarTokens: estimativa local, sem rede.
func (m *MotorGemini) ContarTokens(_ context.Context, texto string) (int, error) {
	return EstimarTokens(texto), nil
}

// Uso devolve o consumo acumulado (cópia).
func (m *MotorGemini) Uso() Uso {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.uso
}

// Fechar descarta a chave e as conexões ociosas.
func (m *MotorGemini) Fechar() {
	m.chave = ""
	m.cli = nil
	if m.hc != nil {
		m.hc.CloseIdleConnections()
	}
}

// configuracao monta o pedido ao provedor. Temperatura: os modelos Gemini 3
// devem ficar no padrão (a documentação oficial alerta para laços e perda de
// qualidade abaixo de 1,0), então Pedido.Temperatura não é enviada.
func (m *MotorGemini) configuracao(p Pedido) (*genai.GenerateContentConfig, error) {
	falso := false
	cfg := &genai.GenerateContentConfig{
		CandidateCount:        1,
		Store:                 &falso, // sem registro de requisições no provedor (V4 §47.3)
		AutomaticContinuation: &falso, // cada continuação seria cobrada; resposta parcial = truncada
		SafetySettings:        segurancaDesligada(),
	}
	if p.Sistema != "" {
		cfg.SystemInstruction = genai.NewContentFromText(p.Sistema, genai.RoleUser)
	}
	saida := p.MaxTokens
	if saida <= 0 {
		saida = 2048
	}
	if m.pensa {
		saida += folgaPensamento
		if m.saidaMax > saida {
			saida = m.saidaMax
		}
		if m.pensamento != "" {
			cfg.ThinkingConfig = &genai.ThinkingConfig{ThinkingLevel: genai.ThinkingLevel(m.pensamento)}
		}
	}
	if m.saidaMax > 0 && saida > m.saidaMax {
		saida = m.saidaMax
	}
	cfg.MaxOutputTokens = int32(saida)
	if len(p.Schema) > 0 {
		var sch any
		if err := json.Unmarshal(p.Schema, &sch); err != nil {
			return nil, erros.Com(erros.InternoInesperado, fmt.Errorf("schema inválido: %w", err))
		}
		cfg.ResponseMIMEType = "application/json"
		cfg.ResponseJsonSchema = semLimitesDeLista(sch)
	}
	return cfg, nil
}

// semLimitesDeLista remove minItems/maxItems do schema enviado à Gemini API:
// em schemas maiores (extração, análise) ela recusa a requisição inteira com
// 400 INVALID_ARGUMENT, sem apontar campo (verificado em 09/11/2026 com os
// modelos 3.5 e 3.6 Flash). Os limites continuam valendo: a resposta é
// conferida no Go contra o schema completo (analise.ConferirSchema).
func semLimitesDeLista(s any) any {
	switch x := s.(type) {
	case map[string]any:
		m := make(map[string]any, len(x))
		for k, v := range x {
			if k == "minItems" || k == "maxItems" {
				continue
			}
			m[k] = semLimitesDeLista(v)
		}
		return m
	case []any:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = semLimitesDeLista(v)
		}
		return out
	}
	return s
}

// segurancaDesligada: os autos descrevem violência, assédio e crimes; com os
// filtros ligados o provedor recusaria peças legítimas. Uma recusa que ainda
// ocorra vira IA-009, nunca silêncio.
func segurancaDesligada() []*genai.SafetySetting {
	var out []*genai.SafetySetting
	for _, c := range []genai.HarmCategory{genai.HarmCategoryHarassment, genai.HarmCategoryHateSpeech,
		genai.HarmCategorySexuallyExplicit, genai.HarmCategoryDangerousContent} {
		out = append(out, &genai.SafetySetting{Category: c, Threshold: genai.HarmBlockThresholdOff})
	}
	return out
}

// respostaVazia substitui, no histórico do reenvio, uma resposta anterior sem
// texto (ex.: cortada durante o raciocínio): a API recusa parte vazia (400).
const respostaVazia = "(a resposta anterior veio vazia ou foi cortada antes do texto)"

func conteudos(p Pedido) []*genai.Content {
	cs := []*genai.Content{genai.NewContentFromText(p.Usuario, genai.RoleUser)}
	for _, h := range p.Historico {
		papel := genai.Role(genai.RoleUser)
		if h.Papel == "assistant" || h.Papel == "model" {
			papel = genai.RoleModel
		}
		t := h.Conteudo
		if strings.TrimSpace(t) == "" {
			t = respostaVazia
		}
		cs = append(cs, genai.NewContentFromText(t, papel))
	}
	return cs
}

// Gerar faz a chamada e só devolve resposta íntegra (TRV-IA-012):
// FinishReason STOP com texto; MAX_TOKENS/continuação → Truncada (o
// validador pede resposta mais concisa); recusa → IA-009; vazio → IA-010.
func (m *MotorGemini) Gerar(ctx context.Context, p Pedido) (Resposta, error) {
	if m.cli == nil {
		return Resposta{}, erros.Com(erros.IAIndisponivel, errors.New("motor fechado"))
	}
	cfg, err := m.configuracao(p)
	if err != nil {
		return Resposta{}, err
	}
	ini := time.Now()
	var resp *genai.GenerateContentResponse
	err = m.repetir(ctx, func(ctx context.Context) error {
		var e error
		resp, e = m.cli.Models.GenerateContent(ctx, m.modelo, conteudos(p), cfg)
		return e
	})
	dur := time.Since(ini)
	if err != nil {
		m.registrar(nil, dur, true)
		return Resposta{}, anotar(err, p)
	}
	m.registrar(resp, dur, false)
	return interpretar(resp, dur)
}

func interpretar(resp *genai.GenerateContentResponse, dur time.Duration) (Resposta, error) {
	r := Resposta{Duracao: dur}
	if u := resp.UsageMetadata; u != nil {
		r.TokensEntrada = int(u.PromptTokenCount)
		r.TokensSaida = int(u.CandidatesTokenCount)
		r.TokensPensamento = int(u.ThoughtsTokenCount)
		if s := dur.Seconds(); s > 0 {
			r.TokensPorSeg = float64(r.TokensSaida+r.TokensPensamento) / s
		}
	}
	if pf := resp.PromptFeedback; pf != nil && pf.BlockReason != "" && pf.BlockReason != genai.BlockedReasonUnspecified {
		return r, erros.Com(erros.IARespostaBloqueada, fmt.Errorf("pedido bloqueado: %s", pf.BlockReason))
	}
	if len(resp.Candidates) != 1 || resp.Candidates[0] == nil {
		return r, erros.Com(erros.IARespostaIncompleta, fmt.Errorf("%d candidatos", len(resp.Candidates)))
	}
	c := resp.Candidates[0]
	var b strings.Builder
	if c.Content != nil {
		for _, part := range c.Content.Parts {
			if part != nil && !part.Thought && part.Text != "" {
				b.WriteString(part.Text)
			}
		}
	}
	r.Conteudo = []byte(b.String())
	switch c.FinishReason {
	case genai.FinishReasonStop:
		if strings.TrimSpace(b.String()) == "" {
			return Resposta{Duracao: dur}, erros.Com(erros.IARespostaIncompleta, errors.New("resposta vazia"))
		}
		return r, nil
	case genai.FinishReasonMaxTokens, genai.FinishReasonContinuation:
		r.Truncada = true
		return r, nil
	case genai.FinishReasonSafety, genai.FinishReasonRecitation, genai.FinishReasonBlocklist,
		genai.FinishReasonProhibitedContent, genai.FinishReasonSPII, genai.FinishReasonLanguage,
		genai.FinishReasonOther:
		return Resposta{Duracao: dur}, erros.Com(erros.IARespostaBloqueada, fmt.Errorf("finish_reason %s", c.FinishReason))
	}
	return Resposta{Duracao: dur}, erros.Com(erros.IARespostaIncompleta, fmt.Errorf("finish_reason %q", c.FinishReason))
}

func (m *MotorGemini) registrar(resp *genai.GenerateContentResponse, dur time.Duration, falhou bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uso.Chamadas++
	m.uso.Duracao += dur
	if falhou {
		m.uso.Falhas++
	}
	if resp != nil && resp.UsageMetadata != nil {
		m.uso.TokensEntrada += int(resp.UsageMetadata.PromptTokenCount)
		m.uso.TokensSaida += int(resp.UsageMetadata.CandidatesTokenCount)
		m.uso.TokensPensamento += int(resp.UsageMetadata.ThoughtsTokenCount)
	}
}

// repetir executa f com reenvio limitado das falhas transitórias (429 por
// minuto, 5xx, rede), com backoff exponencial e jitter, respeitando o
// RetryInfo do provedor. Erros definitivos voltam na primeira vez.
func (m *MotorGemini) repetir(ctx context.Context, f func(context.Context) error) error {
	var ultimo falhaGemini
	for tentativa := 0; tentativa < tentativasTransporte; tentativa++ {
		err := f(ctx)
		if err == nil {
			return nil
		}
		ultimo = classificar(ctx, err, m.chave)
		if ultimo.cancelado || !ultimo.repetir || tentativa == tentativasTransporte-1 {
			break
		}
		espera := ultimo.aguardar
		if espera <= 0 {
			espera = time.Second<<tentativa + time.Duration(rand.Int64N(int64(time.Second)))
		}
		if m.esperaMax > 0 && espera > m.esperaMax {
			break // espera longa demais: o erro real vai ao usuário
		}
		if prazo, ok := ctx.Deadline(); ok && time.Until(prazo) < espera+time.Second {
			break // a espera não cabe no prazo: não adianta começar
		}
		if err := m.espera(ctx, espera); err != nil {
			if errors.Is(err, context.Canceled) {
				return context.Canceled // cancelamento pelo usuário
			}
			break // prazo esgotado durante a espera: vale o erro da última tentativa
		}
	}
	return ultimo.erro
}

func esperar(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// anotar acrescenta à causa técnica (só diagnóstico em memória) a etapa e o
// formato do pedido; nenhum conteúdo dos autos entra.
func anotar(err error, p Pedido) error {
	var e *erros.Erro
	if !errors.As(err, &e) || e.Causa == nil {
		return err
	}
	vazios := 0
	for _, h := range p.Historico {
		if strings.TrimSpace(h.Conteudo) == "" {
			vazios++
		}
	}
	c := *e
	c.Causa = fmt.Errorf("%w [schema=%s historico=%d vazios=%d schema_bytes=%d]", e.Causa, p.NomeSchema, len(p.Historico), vazios, len(p.Schema))
	return &c
}

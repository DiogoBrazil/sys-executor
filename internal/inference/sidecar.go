package inference

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// Pedido é uma chamada de geração estruturada (sem ferramentas; plano §5.6).
type Pedido struct {
	Sistema     string
	Usuario     string
	NomeSchema  string
	Schema      json.RawMessage // JSON Schema imposto pela gramática do servidor
	MaxTokens   int
	Temperatura float64
	// Historico permite reenviar com a mensagem do validador (§5.7).
	Historico []Mensagem
}

// Mensagem de chat.
type Mensagem struct {
	Papel    string `json:"role"`
	Conteudo string `json:"content"`
}

// Resposta da geração.
type Resposta struct {
	Conteudo      []byte
	Truncada      bool // parou por limite de tokens
	TokensEntrada int
	TokensSaida   int
	TokensPorSeg  float64
	Duracao       time.Duration
}

// Motor é a porta de inferência (TRV-IA-001): o orquestrador só conhece isto.
type Motor interface {
	Gerar(ctx context.Context, p Pedido) (Resposta, error)
	ContarTokens(ctx context.Context, texto string) (int, error)
	Contexto() int
}

// ConfigServidor descreve uma instância do llama-server.
type ConfigServidor struct {
	Executavel    string
	Modelo        string
	Contexto      int
	Threads       int
	CamadasGPU    int // 0 = só CPU (padrão; D-08)
	Embeddings    bool
	LimiteMemoria uint64 // limite do Job (Windows); 0 = sem limite
	TimeoutCarga  time.Duration
	TimeoutPedido time.Duration
}

// Servidor é um llama-server em execução: só loopback, porta e chave
// aleatórias, sem web UI, sem log em disco (saída guardada em memória).
type Servidor struct {
	cfg   ConfigServidor
	base  string
	chave string
	cmd   *exec.Cmd
	prot  protecao
	saida *anel
	http  *http.Client

	morto   chan struct{}
	errFim  error
	fechar  sync.Once
	fechado bool
	mu      sync.Mutex
}

// LocalizarServidor procura o llama-server: SYSEXEC_LLAMA_SERVER ou
// <pasta do executável>/llama/cpu/llama-server[.exe].
func LocalizarServidor() (string, error) {
	if p := os.Getenv("SYSEXEC_LLAMA_SERVER"); p != "" {
		return p, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	nome := "llama-server"
	if runtime.GOOS == "windows" {
		nome += ".exe"
	}
	p := filepath.Join(filepath.Dir(exe), "llama", "cpu", nome)
	if _, err := os.Stat(p); err != nil {
		return "", erros.Com(erros.ModeloIncompativel, fmt.Errorf("llama-server não encontrado em %s", p))
	}
	return p, nil
}

func portaLivre() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func argumentos(cfg ConfigServidor, porta int, chave string) []string {
	a := []string{"-m", cfg.Modelo, "--host", "127.0.0.1", "--port", strconv.Itoa(porta), "--api-key", chave,
		"--no-webui", "--no-slots", "--offline", "--log-disable",
		"-c", strconv.Itoa(cfg.Contexto), "-ngl", strconv.Itoa(cfg.CamadasGPU), "-np", "1"}
	if cfg.Threads > 0 {
		a = append(a, "-t", strconv.Itoa(cfg.Threads))
	}
	if cfg.Embeddings {
		a = append(a, "--embeddings", "--pooling", "last")
	} else {
		a = append(a, "--reasoning", "off") // D-09.11: sem thinking em CPU
	}
	return a
}

// IniciarServidor sobe o sidecar e espera o modelo carregar.
func IniciarServidor(ctx context.Context, cfg ConfigServidor) (*Servidor, error) {
	if cfg.TimeoutCarga == 0 {
		cfg.TimeoutCarga = 5 * time.Minute
	}
	if cfg.TimeoutPedido == 0 {
		cfg.TimeoutPedido = 15 * time.Minute
	}
	porta, err := portaLivre()
	if err != nil {
		return nil, erros.Com(erros.ModeloTravado, err)
	}
	kb := make([]byte, 24)
	if _, err := rand.Read(kb); err != nil {
		return nil, erros.Com(erros.InternoInesperado, err)
	}
	s := &Servidor{
		cfg: cfg, base: fmt.Sprintf("http://127.0.0.1:%d", porta), chave: hex.EncodeToString(kb),
		saida: &anel{max: 64 << 10}, morto: make(chan struct{}),
		http: &http.Client{Transport: &http.Transport{Proxy: nil, DisableCompression: true}},
	}
	s.cmd = exec.Command(cfg.Executavel, argumentos(cfg, porta, s.chave)...)
	s.cmd.Stdout, s.cmd.Stderr = s.saida, s.saida
	s.cmd.Env = ambienteSidecar()
	prepararProcesso(s.cmd)
	if err := s.cmd.Start(); err != nil {
		return nil, erros.Com(erros.ModeloIncompativel, err)
	}
	if s.prot, err = proteger(s.cmd, cfg.LimiteMemoria); err != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
		return nil, erros.Com(erros.ModeloTravado, err)
	}
	go func() {
		err := s.cmd.Wait()
		s.mu.Lock()
		s.errFim = err
		s.mu.Unlock()
		close(s.morto)
	}()
	if err := s.esperarCarga(ctx); err != nil {
		s.Fechar()
		return nil, err
	}
	return s, nil
}

// ambienteSidecar não repassa variáveis de proxy (o sidecar só fala com o
// loopback e roda com --offline).
func ambienteSidecar() []string {
	var out []string
	for _, kv := range os.Environ() {
		k, _, _ := cutIgual(kv)
		switch k {
		case "HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy", "ALL_PROXY", "all_proxy":
			continue
		}
		out = append(out, kv)
	}
	return out
}

func cutIgual(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

func (s *Servidor) esperarCarga(ctx context.Context) error {
	limite := time.NewTimer(s.cfg.TimeoutCarga)
	defer limite.Stop()
	tique := time.NewTicker(200 * time.Millisecond)
	defer tique.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.morto:
			return s.erroMorte("terminou durante a carga")
		case <-limite.C:
			return erros.Com(erros.ModeloTravado, errors.New("tempo de carga esgotado"))
		case <-tique.C:
			rq, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.base+"/health", nil)
			rq.Header.Set("Authorization", "Bearer "+s.chave)
			if r, err := s.http.Do(rq); err == nil {
				io.Copy(io.Discard, r.Body)
				r.Body.Close()
				if r.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
	}
}

// erroMorte classifica o fim inesperado: estouro do limite do Job → MOD-003.
func (s *Servidor) erroMorte(quando string) error {
	s.mu.Lock()
	err := s.errFim
	s.mu.Unlock()
	if s.prot != nil && s.prot.estourouMemoria(err) {
		return erros.Com(erros.ModeloSemMemoria, fmt.Errorf("%s: %v", quando, err))
	}
	return erros.Com(erros.ModeloTravado, fmt.Errorf("%s: %v; saída: %s", quando, err, s.saida.cauda(400)))
}

// Contexto devolve o n_ctx do servidor.
func (s *Servidor) Contexto() int { return s.cfg.Contexto }

// PicoMemoriaPrivada (Windows: Job Object; demais: 0).
func (s *Servidor) PicoMemoriaPrivada() uint64 {
	if s.prot == nil {
		return 0
	}
	return s.prot.pico()
}

// Vivo informa se o processo continua em execução.
func (s *Servidor) Vivo() bool {
	select {
	case <-s.morto:
		return false
	default:
		return true
	}
}

// Fechar encerra o processo (o Job também o encerraria com o app).
func (s *Servidor) Fechar() {
	s.fechar.Do(func() {
		s.mu.Lock()
		s.fechado = true
		s.mu.Unlock()
		if s.cmd != nil && s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
			<-s.morto
		}
		if s.prot != nil {
			s.prot.fechar()
		}
		s.saida.limpar()
	})
}

func (s *Servidor) post(ctx context.Context, caminho string, corpo any, destino any) error {
	b, err := json.Marshal(corpo)
	if err != nil {
		return erros.Com(erros.InternoInesperado, err)
	}
	ctx, cancel := context.WithTimeout(ctx, s.cfg.TimeoutPedido)
	defer cancel()
	// Morte do processo durante o pedido cancela a espera.
	go func() {
		select {
		case <-s.morto:
			cancel()
		case <-ctx.Done():
		}
	}()
	rq, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.base+caminho, bytes.NewReader(b))
	rq.Header.Set("Authorization", "Bearer "+s.chave)
	rq.Header.Set("Content-Type", "application/json")
	r, err := s.http.Do(rq)
	if err != nil {
		if !s.Vivo() {
			return s.erroMorte("terminou durante o pedido")
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return ctx.Err() // cancelamento pela tarefa
		}
		// A conexão pode cair antes de o Wait registrar a morte do processo.
		select {
		case <-s.morto:
			return s.erroMorte("terminou durante o pedido")
		case <-time.After(2 * time.Second):
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return erros.Com(erros.ModeloTravado, errors.New("tempo do pedido esgotado"))
		}
		return erros.Com(erros.ModeloTravado, err)
	}
	defer r.Body.Close()
	corpoResp, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if err != nil {
		return erros.Com(erros.ModeloTravado, err)
	}
	if r.StatusCode != http.StatusOK {
		return erros.Com(erros.ModeloTravado, fmt.Errorf("HTTP %d: %.300s", r.StatusCode, corpoResp))
	}
	if err := json.Unmarshal(corpoResp, destino); err != nil {
		return erros.Com(erros.ModeloTravado, err)
	}
	return nil
}

// Gerar executa uma chamada com saída restrita ao JSON Schema.
func (s *Servidor) Gerar(ctx context.Context, p Pedido) (Resposta, error) {
	msgs := []Mensagem{{"system", p.Sistema}, {"user", p.Usuario}}
	msgs = append(msgs, p.Historico...)
	corpo := map[string]any{
		"messages":    msgs,
		"temperature": p.Temperatura,
		"max_tokens":  p.MaxTokens,
		"stream":      false,
	}
	if len(p.Schema) > 0 {
		nome := p.NomeSchema
		if nome == "" {
			nome = "saida"
		}
		corpo["response_format"] = map[string]any{"type": "json_schema",
			"json_schema": map[string]any{"name": nome, "strict": true, "schema": p.Schema}}
	}
	var out struct {
		Choices []struct {
			Message      struct{ Content string } `json:"message"`
			FinishReason string                   `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			Prompt     int `json:"prompt_tokens"`
			Completion int `json:"completion_tokens"`
		} `json:"usage"`
		Timings struct {
			PredictedPerSecond float64 `json:"predicted_per_second"`
		} `json:"timings"`
	}
	t0 := time.Now()
	if err := s.post(ctx, "/v1/chat/completions", corpo, &out); err != nil {
		return Resposta{}, err
	}
	if len(out.Choices) == 0 {
		return Resposta{}, erros.Com(erros.ModeloTravado, errors.New("resposta sem escolhas"))
	}
	return Resposta{
		Conteudo: []byte(out.Choices[0].Message.Content), Truncada: out.Choices[0].FinishReason == "length",
		TokensEntrada: out.Usage.Prompt, TokensSaida: out.Usage.Completion,
		TokensPorSeg: out.Timings.PredictedPerSecond, Duracao: time.Since(t0),
	}, nil
}

// ContarTokens usa o tokenizador do próprio modelo.
func (s *Servidor) ContarTokens(ctx context.Context, texto string) (int, error) {
	var out struct {
		Tokens []json.RawMessage `json:"tokens"`
	}
	if err := s.post(ctx, "/tokenize", map[string]any{"content": texto}, &out); err != nil {
		return 0, err
	}
	return len(out.Tokens), nil
}

// Embeddings devolve um vetor por texto (instância com cfg.Embeddings).
func (s *Servidor) Embeddings(ctx context.Context, textos []string) ([][]float32, error) {
	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	if err := s.post(ctx, "/v1/embeddings", map[string]any{"input": textos}, &out); err != nil {
		return nil, err
	}
	vs := make([][]float32, len(textos))
	for _, d := range out.Data {
		if d.Index >= 0 && d.Index < len(vs) {
			vs[d.Index] = d.Embedding
		}
	}
	return vs, nil
}

// anel guarda só o final da saída do processo, em memória (sem log em disco).
type anel struct {
	mu  sync.Mutex
	buf []byte
	max int
}

func (a *anel) Write(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.buf = append(a.buf, p...)
	if len(a.buf) > a.max {
		a.buf = append(a.buf[:0], a.buf[len(a.buf)-a.max:]...)
	}
	return len(p), nil
}

func (a *anel) cauda(n int) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.buf) > n {
		return string(a.buf[len(a.buf)-n:])
	}
	return string(a.buf)
}

func (a *anel) limpar() {
	a.mu.Lock()
	clear(a.buf)
	a.buf = nil
	a.mu.Unlock()
}

// PID do processo (benchmark: pico de memória pelo sistema).
func (s *Servidor) PID() int {
	if s.cmd == nil || s.cmd.Process == nil {
		return 0
	}
	return s.cmd.Process.Pid
}

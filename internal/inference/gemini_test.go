package inference

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

const chaveTeste = "AIzaSy-CHAVE-SENTINELA-0123456789abcdef"

// geminiFalso imita a REST da Gemini API (v1beta) e guarda as requisições.
type geminiFalso struct {
	t      *testing.T
	srv    *httptest.Server
	mu     sync.Mutex
	pedido []map[string]any // corpos de generateContent
	chaves []string         // x-goog-api-key recebidas
	n      int              // generateContent recebidos
	// respostas sucessivas de generateContent: (status, corpo)
	roteiro []resp
	modelo  string // JSON de GET models/{m}
	lista   string // JSON de GET models
	atraso  time.Duration
}

type resp struct {
	status int
	corpo  string
}

func novoFalso(t *testing.T) *geminiFalso {
	f := &geminiFalso{t: t,
		modelo: `{"name":"models/gemini-teste","displayName":"Gemini Teste","inputTokenLimit":1048576,"outputTokenLimit":65536,"supportedGenerationMethods":["generateContent"],"thinking":true}`,
	}
	f.srv = httptest.NewServer(http.HandlerFunc(f.servir))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *geminiFalso) servir(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.chaves = append(f.chaves, r.Header.Get("x-goog-api-key"))
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case strings.HasSuffix(r.URL.Path, ":generateContent"):
		var corpo map[string]any
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &corpo)
		f.mu.Lock()
		f.pedido = append(f.pedido, corpo)
		i := f.n
		f.n++
		atraso := f.atraso
		if i >= len(f.roteiro) {
			i = len(f.roteiro) - 1
		}
		rr := f.roteiro[i]
		f.mu.Unlock()
		if atraso > 0 {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(atraso):
			}
		}
		w.WriteHeader(rr.status)
		io.WriteString(w, rr.corpo)
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/models"):
		io.WriteString(w, f.lista)
	case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/models/"):
		io.WriteString(w, f.modelo)
	default:
		http.NotFound(w, r)
	}
}

func (f *geminiFalso) qtd() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.n
}

func (f *geminiFalso) corpo(i int) map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.pedido[i]
}

func (f *geminiFalso) definirAtraso(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.atraso = d
}

func (f *geminiFalso) cfg() ConfigGemini {
	return ConfigGemini{Chave: []byte(chaveTeste), Modelo: "gemini-teste", baseURL: f.srv.URL + "/",
		espera: func(ctx context.Context, _ time.Duration) error { return ctx.Err() }}
}

func ok(texto string) resp {
	b, _ := json.Marshal(texto)
	return resp{200, `{"candidates":[{"content":{"role":"model","parts":[{"text":"pensando…","thought":true},{"text":` + string(b) +
		`}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":120,"candidatesTokenCount":15,"thoughtsTokenCount":40,"totalTokenCount":175}}`}
}

func erroAPI(code int, status, msg, detalhes string) resp {
	if detalhes == "" {
		detalhes = "[]"
	}
	return resp{code, `{"error":{"code":` + itoa(code) + `,"status":"` + status + `","message":"` + msg + `","details":` + detalhes + `}}`}
}

func itoa(i int) string { b, _ := json.Marshal(i); return string(b) }

func codigo(err error) erros.Codigo {
	var e *erros.Erro
	if errors.As(err, &e) {
		return e.Codigo
	}
	return ""
}

var pedidoTeste = Pedido{
	Sistema: "REGRAS", Usuario: "AUTOS", NomeSchema: "x", MaxTokens: 500, Temperatura: 0.1,
	Schema:    json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"}},"required":["a"],"additionalProperties":false}`),
	Historico: []Mensagem{{Papel: "assistant", Conteudo: "{}"}, {Papel: "user", Conteudo: "corrija"}},
}

// Regra: TRV-IA-011
// Regra: TRV-SEG-012
func TestGerarMapeiaPedidoESemRetencao(t *testing.T) {
	f := novoFalso(t)
	f.roteiro = []resp{ok(`{"a":"b"}`)}
	m, err := NovoMotorGemini(context.Background(), f.cfg())
	if err != nil {
		t.Fatal(err)
	}
	if m.Contexto() != ContextoPadrao {
		t.Fatalf("contexto %d", m.Contexto())
	}
	r, err := m.Gerar(context.Background(), pedidoTeste)
	if err != nil {
		t.Fatal(err)
	}
	if string(r.Conteudo) != `{"a":"b"}` || r.Truncada {
		t.Fatalf("resposta %q (partes de raciocínio não podem entrar)", r.Conteudo)
	}
	if r.TokensEntrada != 120 || r.TokensSaida != 15 || r.TokensPensamento != 40 {
		t.Fatalf("uso %+v", r)
	}
	c := f.corpo(0)
	if c["store"] != false {
		t.Fatalf("store=false obrigatório (V4 §47.3): %v", c["store"])
	}
	if _, tem := c["tools"]; tem {
		t.Fatal("o modelo não pode receber ferramentas")
	}
	gc, _ := c["generationConfig"].(map[string]any)
	if gc["responseMimeType"] != "application/json" || gc["responseJsonSchema"] == nil {
		t.Fatalf("schema não enviado: %v", gc)
	}
	if _, tem := gc["temperature"]; tem {
		t.Fatal("temperatura não deve ser enviada (padrão do modelo Gemini 3)")
	}
	if gc["maxOutputTokens"].(float64) != float64(500+folgaPensamento) {
		t.Fatalf("maxOutputTokens %v", gc["maxOutputTokens"])
	}
	conts, _ := c["contents"].([]any)
	if len(conts) != 3 || conts[1].(map[string]any)["role"] != "model" || conts[2].(map[string]any)["role"] != "user" {
		t.Fatalf("histórico mal mapeado: %v", conts)
	}
	si, _ := json.Marshal(c["systemInstruction"])
	if !strings.Contains(string(si), "REGRAS") {
		t.Fatalf("instrução de sistema: %s", si)
	}
	ss, _ := c["safetySettings"].([]any)
	if len(ss) != 4 {
		t.Fatalf("safetySettings %v", ss)
	}
	f.mu.Lock()
	chaves := append([]string(nil), f.chaves...)
	f.mu.Unlock()
	for _, k := range chaves {
		if k != chaveTeste {
			t.Fatalf("chave deve ir no cabeçalho x-goog-api-key")
		}
	}
	if u := m.Uso(); u.Chamadas != 1 || u.TokensEntrada != 120 || u.TokensPensamento != 40 {
		t.Fatalf("uso acumulado %+v", u)
	}
}

// Regra: TRV-IA-012
func TestIntegridadeDaResposta(t *testing.T) {
	casos := []struct {
		nome     string
		r        resp
		truncada bool
		cod      erros.Codigo
	}{
		{"limite de tokens", resp{200, `{"candidates":[{"content":{"parts":[{"text":"{\"a\":"}]},"finishReason":"MAX_TOKENS"}]}`}, true, ""},
		{"segurança", resp{200, `{"candidates":[{"content":{"parts":[{"text":"x"}]},"finishReason":"SAFETY"}]}`}, false, erros.IARespostaBloqueada},
		{"recitação", resp{200, `{"candidates":[{"finishReason":"RECITATION"}]}`}, false, erros.IARespostaBloqueada},
		{"pedido bloqueado", resp{200, `{"promptFeedback":{"blockReason":"PROHIBITED_CONTENT"}}`}, false, erros.IARespostaBloqueada},
		{"sem candidato", resp{200, `{"candidates":[]}`}, false, erros.IARespostaIncompleta},
		{"vazio", resp{200, `{"candidates":[{"content":{"parts":[{"text":"  "}]},"finishReason":"STOP"}]}`}, false, erros.IARespostaIncompleta},
		{"motivo ausente", resp{200, `{"candidates":[{"content":{"parts":[{"text":"{}"}]}}]}`}, false, erros.IARespostaIncompleta},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			f := novoFalso(t)
			f.roteiro = []resp{c.r}
			m, err := NovoMotorGemini(context.Background(), f.cfg())
			if err != nil {
				t.Fatal(err)
			}
			r, err := m.Gerar(context.Background(), pedidoTeste)
			if c.cod != "" {
				if codigo(err) != c.cod {
					t.Fatalf("esperado %s, veio %v", c.cod, err)
				}
				if len(r.Conteudo) != 0 {
					t.Fatal("resposta recusada não pode trazer conteúdo aproveitável")
				}
				return
			}
			if err != nil || r.Truncada != c.truncada {
				t.Fatalf("err=%v truncada=%v", err, r.Truncada)
			}
		})
	}
}

func TestErrosDaAPI(t *testing.T) {
	infoChave := `[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"API_KEY_INVALID"}]`
	diaria := `[{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[{"quotaId":"GenerateRequestsPerDayPerProjectPerModel-FreeTier"}]},{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"3s"}]`
	casos := []struct {
		nome      string
		roteiro   []resp
		cod       erros.Codigo
		chamadas  int
		semCodigo bool
	}{
		{"chave inválida", []resp{erroAPI(400, "INVALID_ARGUMENT", "API key not valid. Please pass a valid API key.", infoChave)}, erros.IAChaveInvalida, 1, false},
		{"permissão", []resp{erroAPI(403, "PERMISSION_DENIED", "denied", "")}, erros.IAPermissaoNegada, 1, false},
		{"região", []resp{erroAPI(400, "FAILED_PRECONDITION", "User location is not supported", "")}, erros.IAPermissaoNegada, 1, false},
		{"modelo", []resp{erroAPI(404, "NOT_FOUND", "not found", "")}, erros.IAModeloIndisponivel, 1, false},
		{"cota diária não repete", []resp{erroAPI(429, "RESOURCE_EXHAUSTED", "quota", diaria)}, erros.IACota, 1, false},
		{"cota por minuto repete e passa", []resp{erroAPI(429, "RESOURCE_EXHAUSTED", "quota",
			`[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"2s"}]`), ok(`{"a":"b"}`)}, "", 2, true},
		{"503 três vezes", []resp{erroAPI(503, "UNAVAILABLE", "overloaded", "")}, erros.IAIndisponivel, tentativasTransporte, false},
		{"créditos pré-pagos esgotados", []resp{erroAPI(402, "RESOURCE_EXHAUSTED", "Your prepayment credits are depleted. Please go to AI Studio at https://ai.studio/projects to manage your project and billing.", "")}, erros.IASemCreditos, 1, false},
		{"status sem código conhecido", []resp{erroAPI(499, "UNAVAILABLE", "try later", "")}, erros.IAIndisponivel, tentativasTransporte, false},
		{"modelo fora da cota do nível", []resp{erroAPI(429, "RESOURCE_EXHAUSTED", "Quota exceeded for metric: generativelanguage.googleapis.com/generate_content_free_tier_requests, limit: 0, model: gemini-3.8-flash", "")}, erros.IAModeloIndisponivel, 1, false},
		{"400 do pedido é defeito interno", []resp{erroAPI(400, "INVALID_ARGUMENT", "schema too complex", "")}, erros.InternoInesperado, 1, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			f := novoFalso(t)
			f.roteiro = c.roteiro
			m, err := NovoMotorGemini(context.Background(), f.cfg())
			if err != nil {
				t.Fatal(err)
			}
			_, err = m.Gerar(context.Background(), pedidoTeste)
			if c.semCodigo {
				if err != nil {
					t.Fatalf("esperado sucesso após reenvio: %v", err)
				}
			} else if codigo(err) != c.cod {
				t.Fatalf("esperado %s, veio %v", c.cod, err)
			}
			if f.qtd() != c.chamadas {
				t.Fatalf("chamadas %d, esperado %d", f.qtd(), c.chamadas)
			}
			if err != nil && strings.Contains(err.Error(), chaveTeste) {
				t.Fatal("a chave apareceu no erro")
			}
		})
	}
}

func TestChaveAusenteEModeloInvalido(t *testing.T) {
	f := novoFalso(t)
	c := f.cfg()
	c.Chave = nil
	if _, err := NovoMotorGemini(context.Background(), c); codigo(err) != erros.IAChaveAusente {
		t.Fatalf("esperado IA-001: %v", err)
	}
	c = f.cfg()
	c.Modelo = "../outro"
	if _, err := NovoMotorGemini(context.Background(), c); codigo(err) != erros.IAModeloIndisponivel {
		t.Fatalf("esperado IA-004: %v", err)
	}
	c = f.cfg()
	f.modelo = `{"error":{"code":404,"status":"NOT_FOUND","message":"x"}}`
	f.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		io.WriteString(w, f.modelo)
	})
	if _, err := NovoMotorGemini(context.Background(), c); codigo(err) != erros.IAModeloIndisponivel {
		t.Fatalf("modelo inexistente: %v", err)
	}
}

func TestCancelamentoInterrompeRequisicao(t *testing.T) {
	f := novoFalso(t)
	f.roteiro = []resp{ok(`{"a":"b"}`)}
	f.atraso = 10 * time.Second
	m, err := NovoMotorGemini(context.Background(), f.cfg())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancelar)
	ini := time.Now()
	_, err = m.Gerar(ctx, pedidoTeste)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("esperado cancelamento, veio %v", err)
	}
	if time.Since(ini) > 3*time.Second {
		t.Fatal("o cancelamento não interrompeu a requisição")
	}
}

func TestTempoEsgotado(t *testing.T) {
	f := novoFalso(t)
	f.roteiro = []resp{ok(`{"a":"b"}`)}
	c := f.cfg()
	c.Timeout = 200 * time.Millisecond
	m, err := NovoMotorGemini(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	f.definirAtraso(5 * time.Second)
	_, err = m.Gerar(context.Background(), pedidoTeste)
	if codigo(err) != erros.IATempoEsgotado {
		t.Fatalf("esperado IA-008, veio %v", err)
	}
	if f.qtd() != 1 {
		t.Fatalf("tempo esgotado não pode ser repetido automaticamente (custo): %d", f.qtd())
	}
}

func TestSemConexao(t *testing.T) {
	f := novoFalso(t)
	c := f.cfg()
	f.srv.Close()
	_, err := ListarModelos(context.Background(), c)
	if codigo(err) != erros.IASemConexao {
		t.Fatalf("esperado IA-007, veio %v", err)
	}
}

// Regra: TRV-SEG-012
func TestTransporteSoPermiteAOrigem(t *testing.T) {
	tr, err := NovoTransporteRestrito(http.DefaultTransport, baseURLGemini)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{"https://example.com/x", "http://generativelanguage.googleapis.com/v1beta/models",
		"https://generativelanguage.googleapis.com.evil.com/", "https://storage.googleapis.com/upload"} {
		r, _ := http.NewRequest(http.MethodPost, u, strings.NewReader("autos"))
		if _, err := tr.RoundTrip(r); !errors.As(err, new(ErrHostBloqueado)) {
			t.Fatalf("%s deveria ser bloqueado: %v", u, err)
		}
	}
	if baseURLGemini != "https://generativelanguage.googleapis.com/" {
		t.Fatal("base da Gemini API alterada")
	}
}

func TestBaseNaoVemDoAmbiente(t *testing.T) {
	t.Setenv("GOOGLE_GEMINI_BASE_URL", "https://example.com/")
	t.Setenv("GEMINI_API_KEY", "OUTRA-CHAVE")
	if (ConfigGemini{}).base() != baseURLGemini {
		t.Fatal("a base deve ser fixa")
	}
	if _, _, err := novoCliente(context.Background(), ConfigGemini{}); codigo(err) != erros.IAChaveAusente {
		t.Fatalf("sem chave cadastrada não pode usar a do ambiente: %v", err)
	}
}

// Regra: TRV-IA-011
func TestListarModelosFiltraEClassifica(t *testing.T) {
	f := novoFalso(t)
	f.lista = `{"models":[
	 {"name":"models/gemini-3.6-flash","displayName":"Gemini 3.6 Flash","inputTokenLimit":1048576,"outputTokenLimit":65536,"supportedGenerationMethods":["generateContent","countTokens"],"thinking":true},
	 {"name":"models/gemini-3.8-flash-preview","displayName":"Prévia","supportedGenerationMethods":["generateContent"]},
	 {"name":"models/gemini-flash-latest","supportedGenerationMethods":["generateContent"]},
	 {"name":"models/gemini-2.0-flash-exp","supportedGenerationMethods":["generateContent"]},
	 {"name":"models/gemini-3.1-flash-image","supportedGenerationMethods":["generateContent"]},
	 {"name":"models/gemini-3.8-live","supportedGenerationMethods":["bidiGenerateContent"]},
	 {"name":"models/gemini-embedding-001","supportedGenerationMethods":["embedContent"]},
	 {"name":"models/gemini-3-tts","supportedGenerationMethods":["generateContent"]},
	 {"name":"models/gemma-3-27b-it","supportedGenerationMethods":["generateContent"]},
	 {"name":"models/gemini-nano-banana-2.1","displayName":"Nano Banana 2.1","supportedGenerationMethods":["generateContent"],"thinking":true},
	 {"name":"models/gemini-3.5-transcribe","displayName":"Gemini 3.5 Transcribe","supportedGenerationMethods":["generateContent"]},
	 {"name":"models/gemini-3.8-flash","displayName":"Gemini 3.8 Flash","description":"Multimodal model: text, image and audio understanding, transcription","inputTokenLimit":1048576,"outputTokenLimit":65536,"supportedGenerationMethods":["generateContent"],"thinking":true},
	 {"name":"models/imagen-4","supportedGenerationMethods":["predict"]}
	]}`
	ms, err := ListarModelos(context.Background(), f.cfg())
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	est := map[string]string{}
	for _, m := range ms {
		ids = append(ids, m.ID)
		est[m.ID] = m.Estagio
	}
	if strings.Join(ids, ",") != "gemini-3.8-flash,gemini-3.6-flash,gemini-flash-latest,gemini-3.8-flash-preview,gemini-2.0-flash-exp" {
		t.Fatalf("lista: %v", ids)
	}
	if est["gemini-3.6-flash"] != EstagioEstavel || est["gemini-flash-latest"] != EstagioApelido ||
		est["gemini-3.8-flash-preview"] != EstagioPrevia || est["gemini-2.0-flash-exp"] != EstagioExperimental {
		t.Fatalf("estágios: %v", est)
	}
	if !ms[0].Pensamento || ms[0].EntradaMax != 1048576 {
		t.Fatalf("metadados: %+v", ms[0])
	}
}

func TestListarModelosChaveInvalida(t *testing.T) {
	f := novoFalso(t)
	f.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		io.WriteString(w, `{"error":{"code":400,"status":"INVALID_ARGUMENT","message":"API key not valid. Please pass a valid API key.","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"API_KEY_INVALID"}]}}`)
	})
	_, err := ListarModelos(context.Background(), f.cfg())
	if codigo(err) != erros.IAChaveInvalida {
		t.Fatalf("esperado IA-002: %v", err)
	}
}

func TestContextoLimitadoPeloModelo(t *testing.T) {
	f := novoFalso(t)
	f.modelo = `{"name":"models/gemini-teste","inputTokenLimit":8000,"outputTokenLimit":1000,"supportedGenerationMethods":["generateContent"]}`
	f.roteiro = []resp{ok(`{}`)}
	m, err := NovoMotorGemini(context.Background(), f.cfg())
	if err != nil {
		t.Fatal(err)
	}
	if m.Contexto() != 8000 {
		t.Fatalf("contexto %d", m.Contexto())
	}
	if _, err := m.Gerar(context.Background(), Pedido{Usuario: "x", MaxTokens: 5000}); err != nil {
		t.Fatal(err)
	}
	gc := f.corpo(0)["generationConfig"].(map[string]any)
	if gc["maxOutputTokens"].(float64) != 1000 {
		t.Fatalf("saída limitada pelo modelo: %v", gc["maxOutputTokens"])
	}
	if _, tem := gc["thinkingConfig"]; tem {
		t.Fatal("modelo sem raciocínio não recebe thinkingConfig")
	}
}

func TestEstimarTokens(t *testing.T) {
	if EstimarTokens("") != 0 || EstimarTokens("abc") != 1 || EstimarTokens("ação") != 2 {
		t.Fatal("estimativa")
	}
}

// O prazo que acaba durante a espera devolve o erro real (cota), nunca
// "cancelado"; espera maior que EsperaMax não é feita.
func TestEsperaRespeitaPrazoELimite(t *testing.T) {
	cota := erroAPI(429, "RESOURCE_EXHAUSTED", "quota", `[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"30s"}]`)
	f := novoFalso(t)
	f.roteiro = []resp{cota}
	c := f.cfg()
	c.espera = esperar // espera real
	c.EsperaMax = 5 * time.Second
	m, err := NovoMotorGemini(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	ini := time.Now()
	if _, err := m.Gerar(context.Background(), pedidoTeste); codigo(err) != erros.IACota || f.qtd() != 1 || time.Since(ini) > 3*time.Second {
		t.Fatalf("EsperaMax: %v, %d chamadas, %s", err, f.qtd(), time.Since(ini))
	}
	f2 := novoFalso(t)
	f2.roteiro = []resp{erroAPI(429, "RESOURCE_EXHAUSTED", "quota", `[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"2s"}]`)}
	c2 := f2.cfg()
	c2.espera = esperar
	m2, err := NovoMotorGemini(context.Background(), c2)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	_, err = m2.Gerar(ctx, pedidoTeste)
	if errors.Is(err, context.Canceled) || codigo(err) != erros.IACota {
		t.Fatalf("prazo curto deve devolver a cota, não cancelamento: %v", err)
	}
}

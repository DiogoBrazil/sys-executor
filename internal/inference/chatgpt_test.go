package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

const tokenTeste = "TOKEN-SENTINELA-123"

type tokensFalsos struct {
	mu       sync.Mutex
	atual    string
	renovou  int
	falhaRen error
}

func (t *tokensFalsos) Token(context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.atual, nil
}
func (t *tokensFalsos) Renovar(context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.renovou++
	if t.falhaRen != nil {
		return "", t.falhaRen
	}
	t.atual = tokenTeste + "-novo"
	return t.atual, nil
}

type chatgptFalso struct {
	srv     *httptest.Server
	mu      sync.Mutex
	corpos  []map[string]any
	auth    []string
	roteiro []func(w http.ResponseWriter)
	n       int
}

func novoChatGPTFalso(t *testing.T, roteiro ...func(w http.ResponseWriter)) *chatgptFalso {
	f := &chatgptFalso{roteiro: roteiro}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var c map[string]any
		_ = json.Unmarshal(b, &c)
		f.mu.Lock()
		f.corpos = append(f.corpos, c)
		f.auth = append(f.auth, r.Header.Get("Authorization"))
		i := f.n
		f.n++
		if i >= len(f.roteiro) {
			i = len(f.roteiro) - 1
		}
		h := f.roteiro[i]
		f.mu.Unlock()
		w.Header().Set("x-oai-request-id", fmt.Sprintf("req_%d", i))
		h(w)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *chatgptFalso) qtd() int { f.mu.Lock(); defer f.mu.Unlock(); return f.n }

func (f *chatgptFalso) cfg(tok *tokensFalsos) ConfigChatGPT {
	return ConfigChatGPT{Modelo: "gpt-teste", Tokens: tok, origem: f.srv.URL, Inatividade: 2 * time.Second,
		espera: func(ctx context.Context, _ time.Duration) error { return ctx.Err() }}
}

// sse escreve os eventos sem Content-Type (como observado na PoC).
func sse(eventos ...string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		for _, e := range eventos {
			fmt.Fprintf(w, "event: x\ndata: %s\n\n", e)
			if fl, ok := w.(http.Flusher); ok {
				fl.Flush()
			}
		}
	}
}

func http400(corpo string, status int) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.WriteHeader(status)
		io.WriteString(w, corpo)
	}
}

const evCompleto = `{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":120,"output_tokens":15,"output_tokens_details":{"reasoning_tokens":40}},"output":[{"type":"message","content":[{"type":"output_text","text":"{\"a\":\"b\"}"}]}]}}`

var pedidoChatGPT = Pedido{
	Sistema: "REGRAS", Usuario: "AUTOS", NomeSchema: "analise imputação", MaxTokens: 500, Temperatura: 0.1,
	Schema:    json.RawMessage(`{"type":"object","properties":{"a":{"type":"string","maxLength":5}},"required":["a"],"additionalProperties":false}`),
	Historico: []Mensagem{{Papel: "assistant", Conteudo: ""}, {Papel: "user", Conteudo: "corrija"}},
}

// Corpo só com os campos permitidos no modo Sign in with ChatGPT; sucesso só
// com response.completed; texto dos deltas; uso registrado.
// Regra: TRV-IA-015
// Regra: TRV-SEG-012
func TestChatGPTCorpoESucesso(t *testing.T) {
	f := novoChatGPTFalso(t, sse(`{"type":"response.created"}`, `{"type":"response.output_text.delta","delta":"{\"a\":"}`,
		`{"type":"response.output_text.delta","delta":"\"b\"}"}`, evCompleto))
	tok := &tokensFalsos{atual: tokenTeste}
	m, err := NovoMotorChatGPT(f.cfg(tok))
	if err != nil {
		t.Fatal(err)
	}
	r, err := m.Gerar(context.Background(), pedidoChatGPT)
	if err != nil || string(r.Conteudo) != `{"a":"b"}` || r.Truncada || r.TokensEntrada != 120 || r.TokensPensamento != 40 {
		t.Fatalf("%q %+v %v", r.Conteudo, r, err)
	}
	c := f.corpos[0]
	if c["store"] != false || c["stream"] != true || c["instructions"] != "REGRAS" || c["model"] != "gpt-teste" {
		t.Fatalf("corpo: %v", c)
	}
	for _, proibido := range []string{"max_output_tokens", "temperature", "top_p", "metadata", "truncation", "user", "previous_response_id", "tools", "background", "conversation", "prompt"} {
		if _, ok := c[proibido]; ok {
			t.Fatalf("campo proibido enviado: %s", proibido)
		}
	}
	fmtSchema := c["text"].(map[string]any)["format"].(map[string]any)
	if fmtSchema["type"] != "json_schema" || fmtSchema["strict"] != true || fmtSchema["name"] != "analise_imputa__o" {
		t.Fatalf("formato: %v", fmtSchema)
	}
	if s, _ := json.Marshal(fmtSchema["schema"]); !strings.Contains(string(s), `"maxLength":5`) {
		t.Fatalf("schema deve ir inteiro (strict aceita maxLength): %s", s)
	}
	in := c["input"].([]any)
	if len(in) != 3 || in[0].(map[string]any)["role"] != "user" || in[1].(map[string]any)["role"] != "assistant" {
		t.Fatalf("input: %v", in)
	}
	// Assistente vazio não vira parte vazia; nenhum item com papel de sistema.
	if s, _ := json.Marshal(in[1]); !strings.Contains(string(s), "output_text") || strings.Contains(string(s), `"text":""`) {
		t.Fatalf("histórico do assistente: %s", s)
	}
	if b, _ := json.Marshal(c); strings.Contains(string(b), `"system"`) {
		t.Fatal("papel system é recusado nesse modo")
	}
	if f.auth[0] != "Bearer "+tokenTeste {
		t.Fatal("bearer ausente")
	}
	if u := m.Uso(); u.Chamadas != 1 || u.TokensSaida != 15 {
		t.Fatalf("uso %+v", u)
	}
}

// Fim sem response.completed nunca é sucesso; incompleto por tamanho vira
// Truncada; por filtro vira IA-009; falhas no fluxo têm o código certo.
// Regra: TRV-IA-015
func TestChatGPTFluxo(t *testing.T) {
	casos := []struct {
		nome    string
		roteiro func(w http.ResponseWriter)
		codigo  erros.Codigo
		trunc   bool
		envios  int
	}{
		{"sem completed (repete e desiste)", sse(`{"type":"response.output_text.delta","delta":"{\"a\""}`), erros.IARespostaIncompleta, false, tentativasChatGPT},
		{"incompleto por tamanho", sse(`{"type":"response.output_text.delta","delta":"{\"a\""}`, `{"type":"response.incomplete","response":{"incomplete_details":{"reason":"max_output_tokens"}}}`), "", true, 1},
		{"incompleto por filtro", sse(`{"type":"response.incomplete","response":{"incomplete_details":{"reason":"content_filter"}}}`), erros.IARespostaBloqueada, false, 1},
		{"limite do plano no fluxo", sse(`{"type":"response.created"}`, `{"type":"response.failed","response":{"error":{"code":"subscription_sharing_usage_limit_exceeded","message":"x"}}}`), erros.IALimitePlano, false, 1},
		{"uso indisponível no fluxo (repete)", sse(`{"type":"response.failed","response":{"error":{"code":"subscription_sharing_usage_unavailable","message":"x"}}}`), erros.IAIndisponivel, false, tentativasChatGPT},
		{"evento error", sse(`{"type":"error","code":"server_error","message":"y"}`), erros.IAIndisponivel, false, tentativasChatGPT},
		{"vazio", sse(`{"type":"response.completed","response":{"status":"completed"}}`), erros.IARespostaIncompleta, false, 1},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			f := novoChatGPTFalso(t, c.roteiro)
			m, _ := NovoMotorChatGPT(f.cfg(&tokensFalsos{atual: tokenTeste}))
			r, err := m.Gerar(context.Background(), pedidoChatGPT)
			if c.trunc {
				if err != nil || !r.Truncada {
					t.Fatalf("esperava Truncada: %+v %v", r, err)
				}
			} else if codigo(err) != c.codigo {
				t.Fatalf("esperava %s: %v", c.codigo, err)
			}
			if f.qtd() != c.envios {
				t.Fatalf("envios %d, esperado %d", f.qtd(), c.envios)
			}
			if err != nil && strings.Contains(err.Error(), tokenTeste) {
				t.Fatal("token na mensagem de erro")
			}
		})
	}
}

// Erros antes do fluxo, nos dois formatos ({"error":{…}} e {"detail":…}).
func TestChatGPTErrosHTTP(t *testing.T) {
	casos := []struct {
		nome   string
		resp   func(w http.ResponseWriter)
		codigo erros.Codigo
		envios int
	}{
		{"limite do plano não repete", http400(`{"error":{"code":"subscription_sharing_usage_limit_exceeded","message":"x"}}`, 429), erros.IALimitePlano, 1},
		{"não elegível", http400(`{"error":{"code":"subscription_sharing_user_not_eligible","message":"x"}}`, 403), erros.IAContaNaoElegivel, 1},
		{"capacidade não suportada = defeito do app", http400(`{"error":{"code":"subscription_sharing_unsupported_capability","param":"text.format","message":"x"}}`, 400), erros.InternoInesperado, 1},
		{"detail sem código (400)", http400(`{"detail":"Unsupported parameter: temperature"}`, 400), erros.InternoInesperado, 1},
		{"escopo do plano", http400(`{"error":{"code":"chatpass_v2_scope_not_authorized","message":"x"}}`, 403), erros.IAPlanoNaoAutorizado, 1},
		{"modelo inexistente", http400(`{"error":{"code":"model_not_found","message":"x"}}`, 404), erros.IAModeloIndisponivel, 1},
		{"503 repete", http400(`{"detail":"busy"}`, 503), erros.IAIndisponivel, tentativasChatGPT},
		{"429 genérico repete", http400(`{"error":{"code":"rate_limit_exceeded","message":"x"}}`, 429), erros.IACota, tentativasChatGPT},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			f := novoChatGPTFalso(t, c.resp)
			m, _ := NovoMotorChatGPT(f.cfg(&tokensFalsos{atual: tokenTeste}))
			_, err := m.Gerar(context.Background(), pedidoChatGPT)
			if codigo(err) != c.codigo || f.qtd() != c.envios {
				t.Fatalf("%v (envios %d)", err, f.qtd())
			}
		})
	}
}

// 401: renova uma vez e repete com o token novo; renovação que falha volta
// como erro da conta; 401 de novo depois de renovar → IA-018.
func TestChatGPTRenovaNo401(t *testing.T) {
	f := novoChatGPTFalso(t, http400(`{"error":{"code":"token_revoked","message":"x"}}`, 401), sse(evCompleto))
	tok := &tokensFalsos{atual: tokenTeste}
	m, _ := NovoMotorChatGPT(f.cfg(tok))
	if _, err := m.Gerar(context.Background(), pedidoChatGPT); err != nil {
		t.Fatal(err)
	}
	if tok.renovou != 1 || f.auth[1] != "Bearer "+tokenTeste+"-novo" {
		t.Fatalf("renovação: %d %v", tok.renovou, f.auth)
	}
	f2 := novoChatGPTFalso(t, http400(`{"detail":"unauthorized"}`, 401))
	tok2 := &tokensFalsos{atual: tokenTeste}
	m2, _ := NovoMotorChatGPT(f2.cfg(tok2))
	if _, err := m2.Gerar(context.Background(), pedidoChatGPT); codigo(err) != erros.IASessaoExpirada || tok2.renovou != 1 || f2.qtd() != 2 {
		t.Fatalf("401 persistente: %v renovou=%d envios=%d", err, tok2.renovou, f2.qtd())
	}
	tok3 := &tokensFalsos{atual: tokenTeste, falhaRen: erros.Novo(erros.IASessaoExpirada)}
	f3 := novoChatGPTFalso(t, http400(`{"detail":"unauthorized"}`, 401))
	m3, _ := NovoMotorChatGPT(f3.cfg(tok3))
	if _, err := m3.Gerar(context.Background(), pedidoChatGPT); codigo(err) != erros.IASessaoExpirada || f3.qtd() != 1 {
		t.Fatalf("renovação recusada: %v", err)
	}
}

// Fluxo sem atividade → IA-008 sem nova tentativa; cancelamento interrompe.
func TestChatGPTInatividadeECancelamento(t *testing.T) {
	trava := make(chan struct{})
	defer close(trava)
	f := novoChatGPTFalso(t, func(w http.ResponseWriter) {
		fmt.Fprint(w, "data: {\"type\":\"response.created\"}\n\n")
		w.(http.Flusher).Flush()
		<-trava
	})
	cfg := f.cfg(&tokensFalsos{atual: tokenTeste})
	cfg.Inatividade = 200 * time.Millisecond
	m, _ := NovoMotorChatGPT(cfg)
	if _, err := m.Gerar(context.Background(), pedidoChatGPT); codigo(err) != erros.IATempoEsgotado || f.qtd() != 1 {
		t.Fatalf("inatividade: %v envios=%d", err, f.qtd())
	}
	cfg.Inatividade = 5 * time.Second
	m, _ = NovoMotorChatGPT(cfg)
	ctx, cancelar := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancelar()
	if _, err := m.Gerar(ctx, pedidoChatGPT); !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && codigo(err) != erros.IATempoEsgotado {
		t.Fatalf("cancelamento: %v", err)
	}
}

// Barreira (camada 2) e destino único também no ChatGPT.
// Regra: TRV-SEG-013
func TestChatGPTInspecaoEDestino(t *testing.T) {
	f := novoChatGPTFalso(t, sse(evCompleto))
	cfg := f.cfg(&tokensFalsos{atual: tokenTeste})
	cfg.Inspecionar = func(b []byte) error {
		if strings.Contains(string(b), "SEGREDO") {
			return errors.New("resíduo")
		}
		return nil
	}
	m, _ := NovoMotorChatGPT(cfg)
	p := pedidoChatGPT
	p.Usuario = "contém SEGREDO"
	if _, err := m.Gerar(context.Background(), p); codigo(err) != erros.IABloqueioPrivacidade || f.qtd() != 0 {
		t.Fatalf("barreira: %v envios=%d", err, f.qtd())
	}
	cli, _ := NovoClienteRestrito(OrigemAPIOpenAI, nil)
	if _, err := cli.Get("https://chatgpt.com/backend-api/codex"); !errors.As(err, new(ErrHostBloqueado)) {
		t.Fatalf("endpoint privado do ChatGPT deve ser bloqueado: %v", err)
	}
}

// Lista: só visibility "list" e uso pela API, na ordem do servidor.
func TestListarModelosChatGPT(t *testing.T) {
	f := novoChatGPTFalso(t, func(w http.ResponseWriter) {
		io.WriteString(w, `{"models":[
			{"slug":"gpt-a","display_name":"GPT A","visibility":"list","supported_in_api":true,"context_window":500000,"supported_reasoning_levels":[{"effort":"low"},{"effort":"high"}],"default_reasoning_level":"low"},
			{"slug":"oculto","display_name":"X","visibility":"hide","supported_in_api":true},
			{"slug":"so-app","display_name":"Y","visibility":"list","supported_in_api":false},
			{"slug":"gpt-b","display_name":"GPT B","visibility":"list"}]}`)
	})
	ms, err := ListarModelosChatGPT(context.Background(), f.cfg(&tokensFalsos{atual: tokenTeste}))
	if err != nil || len(ms) != 2 || ms[0].ID != "gpt-a" || ms[1].ID != "gpt-b" || ms[0].Contexto != 500000 || len(ms[0].Raciocinio) != 2 {
		t.Fatalf("%+v %v", ms, err)
	}
}

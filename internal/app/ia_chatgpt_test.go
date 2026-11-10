package app

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/chatgpt"
	"github.com/DiogoBrazil/sys-executor/internal/credencial"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/prefs"
)

const tokenSentinela = "ACESSO-SENTINELA-XYZ"

type contaFalsa struct {
	mu          sync.Mutex
	st          chatgpt.Estado
	erroEntrar  error
	confirmada  bool
	entrou      int
	desconectou int
}

func (c *contaFalsa) Token(context.Context) (string, error)   { return tokenSentinela, nil }
func (c *contaFalsa) Renovar(context.Context) (string, error) { return tokenSentinela, nil }
func (c *contaFalsa) Estado() (chatgpt.Estado, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.st, nil
}
func (c *contaFalsa) Entrar(ctx context.Context) error {
	c.mu.Lock()
	c.entrou++
	err := c.erroEntrar
	if err == nil {
		c.st = chatgpt.Estado{Conectada: true, PlanoAutorizado: true, Email: "fulano@exemplo.invalid"}
	}
	c.mu.Unlock()
	return err
}
func (c *contaFalsa) Desconectar(context.Context) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.desconectou++
	c.st = chatgpt.Estado{}
	return c.confirmada, nil
}

func cenarioChatGPT(t *testing.T) (*IAService, *contaFalsa, *prefs.Loja, chan *VisaoIA) {
	t.Helper()
	loja, _ := prefs.Abrir(filepath.Join(t.TempDir(), "p.json"))
	conta := &contaFalsa{confirmada: true}
	fc := &FonteChatGPT{Conta: conta, Loja: loja,
		Listar: func(context.Context, inference.ConfigChatGPT) ([]inference.ModeloChatGPT, error) {
			return []inference.ModeloChatGPT{{ID: "gpt-a", Nome: "GPT A"}, {ID: "gpt-b", Nome: "GPT B"}}, nil
		},
		Testar: func(_ context.Context, c inference.ConfigChatGPT) error {
			if c.Modelo == "gpt-b" {
				return erros.Novo(erros.IALimitePlano)
			}
			return nil
		},
		NovoMotor: func(c inference.ConfigChatGPT) (MotorComUso, error) {
			return &motorUsoFalso{}, nil
		},
	}
	emitidos := make(chan *VisaoIA, 32)
	s := &IAService{Fonte: &FonteGemini{Credenciais: &credencial.LojaMemoria{}, Loja: loja}, ChatGPT: fc,
		Emitir: func(v *VisaoIA) { emitidos <- v }}
	return s, conta, loja, emitidos
}

// Nenhum token na visão da tela (TRV-SEG-014).
func semToken(t *testing.T, vs ...*VisaoIA) {
	t.Helper()
	for _, v := range vs {
		b, _ := json.Marshal(v)
		if strings.Contains(string(b), tokenSentinela) {
			t.Fatalf("token na visão: %s", b)
		}
	}
}

// Alvo do ChatGPT: conta, permissão do plano e modelo conferidos sem rede.
// Regra: TRV-IA-014
func TestFonteChatGPTAlvo(t *testing.T) {
	_, conta, loja, _ := cenarioChatGPT(t)
	f := &FonteChatGPT{Conta: conta, Loja: loja}
	casos := []struct {
		st     chatgpt.Estado
		modelo string
		cod    erros.Codigo
	}{
		{chatgpt.Estado{}, "gpt-a", erros.IAContaDesconectada},
		{chatgpt.Estado{Reconectar: true}, "gpt-a", erros.IASessaoExpirada},
		{chatgpt.Estado{Conectada: true}, "gpt-a", erros.IAPlanoNaoAutorizado},
		{chatgpt.Estado{Conectada: true, PlanoAutorizado: true}, "", erros.IAModeloIndisponivel},
	}
	for _, c := range casos {
		conta.st = c.st
		_, _ = loja.Atualizar(func(p *prefs.Preferencias) { p.ModeloChatGPT = c.modelo })
		if _, err := f.Alvo(); !errors.Is(err, erros.Novo(c.cod)) {
			t.Errorf("%+v: esperado %s, veio %v", c.st, c.cod, err)
		}
	}
	conta.st = chatgpt.Estado{Conectada: true, PlanoAutorizado: true}
	_, _ = loja.Atualizar(func(p *prefs.Preferencias) { p.ModeloChatGPT = "gpt-a" })
	a, err := f.Alvo()
	if err != nil || a.Provedor != inference.ProvedorChatGPT || a.Modelo != "gpt-a" {
		t.Fatalf("%+v %v", a, err)
	}
	var cfg inference.ConfigChatGPT
	f.NovoMotor = func(c inference.ConfigChatGPT) (MotorComUso, error) { cfg = c; return &motorUsoFalso{}, nil }
	insp := func([]byte) error { return nil }
	if _, _, err := f.Abrir(context.Background(), a, func(string) {}, insp); err != nil || cfg.Modelo != "gpt-a" || cfg.Inspecionar == nil || cfg.Tokens == nil {
		t.Fatalf("abrir: %v %+v", err, cfg)
	}
	if _, _, err := f.Abrir(context.Background(), inference.Alvo{Provedor: inference.ProvedorGemini, Modelo: "x"}, func(string) {}, nil); err == nil {
		t.Fatal("alvo de outro provedor deveria ser recusado")
	}
}

func esperarVisao(t *testing.T, ch chan *VisaoIA, cond func(*VisaoIA) bool) *VisaoIA {
	t.Helper()
	limite := time.After(3 * time.Second)
	for {
		select {
		case v := <-ch:
			if cond(v) {
				return v
			}
		case <-limite:
			t.Fatal("visão esperada não chegou")
			return nil
		}
	}
}

// Tela: entrar (assíncrono), listar, escolher modelo (= escolher o
// provedor), testar, desconectar; nenhum token na visão.
// Regra: TRV-SEG-014
func TestIAServiceChatGPT(t *testing.T) {
	s, conta, loja, ch := cenarioChatGPT(t)
	v := s.EntrarChatGPT()
	if !v.ChatGPT.Disponivel || !v.ChatGPT.Entrando {
		t.Fatalf("entrada deveria estar em andamento: %+v", v.ChatGPT)
	}
	fim := esperarVisao(t, ch, func(v *VisaoIA) bool { return v.ChatGPT.Conectada && !v.ChatGPT.Entrando })
	if len(fim.ChatGPT.Modelos) != 2 || fim.ChatGPT.Email != "fulano@exemplo.invalid" || fim.Erro != nil {
		t.Fatalf("depois de entrar: %+v %+v", fim.ChatGPT, fim.Erro)
	}
	if v := s.SelecionarModeloChatGPT("inexistente"); v.Erro == nil {
		t.Fatal("modelo fora da lista aceito")
	}
	v = s.SelecionarModeloChatGPT("gpt-a")
	if p := loja.Obter(); p.ModeloChatGPT != "gpt-a" || p.ProvedorIA != prefs.ProvedorChatGPT || v.Provedor != prefs.ProvedorChatGPT {
		t.Fatalf("escolher o modelo do ChatGPT escolhe o provedor: %+v", p)
	}
	v = s.TestarModeloChatGPT("gpt-b")
	if v.ChatGPT.Modelos[1].Teste != "falhou" || v.ChatGPT.Modelos[1].TesteErro.Codigo != "IA-019" {
		t.Fatalf("teste: %+v", v.ChatGPT.Modelos[1])
	}
	if v := s.SelecionarProvedor(prefs.ProvedorGemini); v.Provedor != prefs.ProvedorGemini {
		t.Fatalf("provedor: %q", v.Provedor)
	}
	if v := s.SelecionarProvedor("OUTRO"); v.Erro == nil {
		t.Fatal("provedor inválido aceito")
	}
	conta.confirmada = false
	v = s.DesconectarChatGPT()
	if v.ChatGPT.Conectada || len(v.ChatGPT.Modelos) != 0 || !strings.Contains(v.Aviso, "Aplicativos conectados") || conta.desconectou != 1 {
		t.Fatalf("desconectar: %+v aviso=%q", v.ChatGPT, v.Aviso)
	}
	semToken(t, &v, fim)
	for len(ch) > 0 {
		semToken(t, <-ch)
	}
}

// Entrada que falha (ex.: conta sem plano elegível) mostra o erro; não lista.
func TestIAServiceChatGPTEntradaFalha(t *testing.T) {
	s, conta, _, ch := cenarioChatGPT(t)
	conta.erroEntrar = erros.Novo(erros.IALoginNaoConcluido)
	s.EntrarChatGPT()
	v := esperarVisao(t, ch, func(v *VisaoIA) bool { return !v.ChatGPT.Entrando && v.Erro != nil })
	if v.Erro.Codigo != "IA-016" || len(v.ChatGPT.Modelos) != 0 {
		t.Fatalf("%+v", v.Erro)
	}
}

// Sem ChatGPT na montagem: operações respondem IA-015, sem pânico.
func TestIAServiceSemChatGPT(t *testing.T) {
	loja, _ := prefs.Abrir(filepath.Join(t.TempDir(), "p.json"))
	s := &IAService{Fonte: &FonteGemini{Credenciais: &credencial.LojaMemoria{}, Loja: loja}}
	for _, v := range []VisaoIA{s.EntrarChatGPT(), s.SelecionarProvedor(prefs.ProvedorChatGPT), s.DesconectarChatGPT()} {
		if v.Erro == nil || v.Erro.Codigo != "IA-015" || v.ChatGPT.Disponivel {
			t.Fatalf("%+v", v)
		}
	}
}

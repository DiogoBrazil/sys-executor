package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/ocr"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/sei"
	"github.com/DiogoBrazil/sys-executor/internal/session"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

const sentinela = "SENHA-SENTINELA-987"

type registro struct {
	mu   sync.Mutex
	json []string
}

func (r *registro) notificar(v *workflow.Visao) {
	b, _ := json.Marshal(v)
	r.mu.Lock()
	r.json = append(r.json, string(b))
	r.mu.Unlock()
}

func montarSEI(t *testing.T, obter ObterSEI) (*workflow.Gerenciador, *registro, string) {
	t.Helper()
	extOnce.Do(func() { ext = pdf.NovoExtrator() })
	p := &Pipeline{Extrator: ext, LocalizarOCR: func() (ocr.Motor, error) { return ocrFalso{conf: 90}, nil }, DPI: 72, SEI: obter}
	raiz := t.TempDir()
	r := &registro{}
	g := workflow.Novo(workflow.Opcoes{Sessoes: session.NovoGerenciador(raiz), Etapas: p.Etapas(), Notificar: r.notificar, SEIDisponivel: true})
	t.Cleanup(g.Encerrar)
	return g, r, raiz
}

// responderPedido espera um pedido do tipo dado e responde.
func responderPedido(t *testing.T, g *workflow.Gerenciador, tipo workflow.TipoPedido, campos map[string]string, segredo string) {
	t.Helper()
	v := esperar(t, g, func(v *workflow.Visao) bool { return v.Pedido != nil && v.Pedido.Tipo == tipo })
	if _, err := g.Fornecer(v.Pedido.ID, workflow.RespostaPedido{Campos: campos, Segredo: session.NovoSegredo([]byte(segredo))}); err != nil {
		t.Fatal(err)
	}
}

// SEG13-01/02: OTP e reautenticação pedidos ao usuário; credenciais nunca na visão.
func TestSEIComOTPEReautenticacao(t *testing.T) {
	pdfBytes := fixtureSEI(t)
	var senhaVista string
	obter := func(ctx context.Context, pasta string, in sei.Interlocutor) ([]byte, error) {
		in.Fase(sei.FaseConectando)
		in.Fase(sei.FaseLogin)
		l, err := in.Credenciais(ctx, nil)
		if err != nil {
			return nil, err
		}
		_ = l.Senha.Usar(func(b []byte) error { senhaVista = string(b); return nil })
		l.Senha.Zerar()
		if l.Processo != "0000.000001/2026-01" || l.Orgao != "PM" {
			t.Errorf("login: %+v", l)
		}
		in.Fase(sei.FaseOTP)
		c, err := in.CodigoOTP(ctx, nil)
		if err != nil {
			return nil, err
		}
		c.Zerar()
		in.Fase(sei.FaseLocalizando)
		in.Fase(sei.FaseReautenticacao)
		s, err := in.SenhaReautenticacao(ctx, nil)
		if err != nil {
			return nil, err
		}
		s.Zerar()
		in.Fase(sei.FaseGerando)
		in.Fase(sei.FaseBaixando)
		if _, err := os.Stat(pasta); err != nil && !os.IsNotExist(err) {
			t.Errorf("pasta: %v", err)
		}
		return pdfBytes, nil
	}
	g, r, _ := montarSEI(t, obter)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(workflow.TipoPADS)
	if _, err := g.EscolherEntrada(workflow.EntradaSEI); err != nil {
		t.Fatal(err)
	}
	responderPedido(t, g, workflow.PedidoCredenciais, map[string]string{"usuario": "u", "orgao": "PM", "processo": "0000.000001/2026-01"}, sentinela)
	responderPedido(t, g, workflow.PedidoOTP, nil, "123456")
	responderPedido(t, g, workflow.PedidoReautenticacao, nil, sentinela)
	v := esperar(t, g, func(v *workflow.Visao) bool { return v.Parada || v.Terminal })
	if v.Estado != workflow.Delimitando || v.Autos == nil || v.Autos.Compatibilidade != "COMPATIVEL" {
		t.Fatalf("%+v", v)
	}
	if senhaVista != sentinela {
		t.Fatal("a senha não chegou ao cliente SEI")
	}
	h := g.Historico()
	for _, e := range []workflow.Estado{workflow.SEIConectividade, workflow.SEILogin, workflow.SEIOTP, workflow.SEILocalizando, workflow.SEIReautenticacao, workflow.SEIGerandoPDF, workflow.SEIBaixando, workflow.Verificando} {
		achou := false
		for _, x := range h {
			achou = achou || x == e
		}
		if !achou {
			t.Errorf("estado %s não percorrido: %v", e, h)
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, j := range r.json {
		if strings.Contains(j, sentinela) || strings.Contains(j, "123456") {
			t.Fatal("segredo apareceu numa visão enviada à interface")
		}
	}
}

// T03: sem internet → aviso e volta à escolha da entrada, mantendo o tipo;
// o envio do PDF segue normalmente.
func TestT03SemInternetOfereceUpload(t *testing.T) {
	obter := func(ctx context.Context, pasta string, in sei.Interlocutor) ([]byte, error) {
		in.Fase(sei.FaseConectando)
		return nil, erros.Novo(erros.RedeSemConexao)
	}
	g, _, _ := montarSEI(t, obter)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(workflow.TipoSR)
	_, _ = g.EscolherEntrada(workflow.EntradaSEI)
	v := esperar(t, g, func(v *workflow.Visao) bool { return v.Estado == workflow.TipoEscolhido && v.Erro != nil })
	if v.Erro.Codigo != "REDE-001" || v.Tipo != workflow.TipoSR || v.Terminal {
		t.Fatalf("%+v", v)
	}
	if _, err := g.EscolherEntrada(workflow.EntradaUpload); err != nil {
		t.Fatal(err)
	}
	if _, err := g.ReceberUpload(fixtureSEI(t)); err != nil {
		t.Fatal(err)
	}
	esperar(t, g, func(v *workflow.Visao) bool { return v.Estado == workflow.AguardandoEsclarecimento }) // SR com autos de PADS
}

// Cancelar enquanto o SEI espera o usuário: a etapa pára e a sessão é apagada.
func TestCancelarDuranteLoginSEI(t *testing.T) {
	parou := make(chan struct{})
	obter := func(ctx context.Context, pasta string, in sei.Interlocutor) ([]byte, error) {
		defer close(parou)
		in.Fase(sei.FaseLogin)
		_, err := in.Credenciais(ctx, nil)
		return nil, err
	}
	g, _, raiz := montarSEI(t, obter)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(workflow.TipoPADS)
	_, _ = g.EscolherEntrada(workflow.EntradaSEI)
	esperar(t, g, func(v *workflow.Visao) bool { return v.Pedido != nil })
	if _, err := g.Cancelar(); err != nil {
		t.Fatal(err)
	}
	<-parou
	g.Encerrar()
	if es, _ := os.ReadDir(raiz); len(es) != 0 {
		t.Fatalf("sessão não apagada: %v", es)
	}
	// Pedido antigo não aceita mais resposta (e o segredo é zerado).
	s := session.NovoSegredo([]byte("x"))
	if _, err := g.Fornecer("qualquer", workflow.RespostaPedido{Segredo: s}); err == nil || !s.Vazio() {
		t.Fatal("resposta fora de hora deve ser rejeitada e zerada")
	}
}

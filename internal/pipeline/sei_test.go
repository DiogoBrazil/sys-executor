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
// Depois do download, os documentos separados vão ao cadastro e só os
// escolhidos entram nos autos (V4 §48.4, revisão de 10/11/2026).
// Cobre: CAD-SEI-01
// Regra: TRV-ING-008
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
	// V4 §48.4 (revisão de 10/11/2026): o PDF do SEI é separado pelo carimbo
	// e vai ao cadastro; o usuário escolhe os documentos de cada peça.
	v := esperar(t, g, func(v *workflow.Visao) bool {
		return v.Estado == workflow.RecebendoUpload || v.Parada || v.Terminal
	})
	if v.Estado != workflow.RecebendoUpload || v.Cadastro == nil || len(v.DocumentosSEI) != 7 {
		t.Fatalf("documentos do SEI: %+v", v)
	}
	ds := v.DocumentosSEI
	var err error
	if ds[0].Sugestao != workflow.PecaMemorando || ds[2].Sugestao != workflow.PecaDefesa || ds[6].Sugestao != workflow.PecaAlegacoes || ds[0].SEI == "" {
		t.Fatalf("sugestões: %+v", ds)
	}
	c := *v.Cadastro
	c.Pessoas = []workflow.PessoaCadastro{{Nome: "Carlos Inventado", PostoGrad: "CB PM"}}
	if v, err = g.SalvarCadastro(c); err != nil {
		t.Fatal(err)
	}
	c = *v.Cadastro
	c.AcusadoID = c.Pessoas[0].ID
	if v, err = g.SalvarCadastro(c); err != nil {
		t.Fatal(err)
	}
	memo, defesa, alegacoes := v.Cadastro.Pecas[0].ID, v.Cadastro.Pecas[1].ID, v.Cadastro.Pecas[2].ID
	for peca, doc := range map[string]string{memo: ds[0].ID, defesa: ds[2].ID, alegacoes: ds[6].ID} {
		if v, err = g.AnexarDocumentosSEI(peca, []string{doc}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := g.AnexarDocumentosSEI(defesa, []string{ds[0].ID}); err == nil {
		t.Fatal("o mesmo documento do SEI não pode entrar em duas peças (ARQ-004)")
	}
	usados := 0
	for _, d := range v.DocumentosSEI {
		if d.Usado != "" {
			usados++
		}
	}
	if usados != 3 || v.Cadastro.Pecas[0].SEI != ds[0].SEI || v.Cadastro.Pecas[0].Arquivos[0].Origem != "SEI" {
		t.Fatalf("anexados: %+v / %+v", v.DocumentosSEI, v.Cadastro.Pecas[0])
	}
	if _, err := g.ConcluirCadastro(); err != nil {
		t.Fatal(err)
	}
	v = esperar(t, g, func(v *workflow.Visao) bool { return v.Parada || v.Terminal })
	paginas := ds[0].Paginas + ds[2].Paginas + ds[6].Paginas
	if v.Estado != workflow.Delimitando || v.Autos == nil || len(v.Autos.Pecas) != 3 || v.Autos.TotalPaginas != paginas {
		t.Fatalf("só os documentos escolhidos entram nos autos: %+v", v.Autos)
	}
	if senhaVista != sentinela {
		t.Fatal("a senha não chegou ao cliente SEI")
	}
	h := g.Historico()
	for _, e := range []workflow.Estado{workflow.SEIConectividade, workflow.SEILogin, workflow.SEIOTP, workflow.SEILocalizando, workflow.SEIReautenticacao, workflow.SEIGerandoPDF, workflow.SEIBaixando, workflow.RecebendoUpload, workflow.Verificando} {
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

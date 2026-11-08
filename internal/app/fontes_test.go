package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/ocr"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/pipeline"
	"github.com/DiogoBrazil/sys-executor/internal/session"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// UI-SRC-01 / SEG-CACHE-01: PDF servido só com token da tarefa ativa, sem
// cache; realce devolve caixas da página.
func TestFontesPDFEDestaque(t *testing.T) {
	ext := pdf.NovoExtrator()
	defer ext.Fechar()
	p := &pipeline.Pipeline{Extrator: ext, LocalizarOCR: func() (ocr.Motor, error) { return nil, os.ErrNotExist }}
	g := workflow.Novo(workflow.Opcoes{Sessoes: session.NovoGerenciador(t.TempDir()), Etapas: p.Etapas()})
	defer g.Encerrar()
	f := NovaFontesService(g)
	if r := f.EnderecoPDF(); r.Erro == nil {
		t.Fatal("sem tarefa não há endereço")
	}
	b, _ := os.ReadFile("../../testdata/autos/coordenadas.pdf")
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(workflow.TipoPADS)
	_, _ = g.EscolherEntrada(workflow.EntradaUpload)
	_, _ = g.ReceberUpload(b)
	// Com -race no CI, o pdfium em WebAssembly fica bem mais lento.
	fim := time.Now().Add(4 * time.Minute)
	pronto := false
	for !pronto && time.Now().Before(fim) {
		v := g.Visao()
		pronto = v.Autos != nil && v.Autos.TotalPaginas > 0 && (v.Parada || v.Estado == workflow.AguardandoEsclarecimento)
		time.Sleep(20 * time.Millisecond)
	}
	if !pronto {
		t.Fatalf("autos não ficaram prontos: %+v", g.Visao())
	}
	end := f.EnderecoPDF()
	if end.Erro != nil || len(end.Endereco) < 30 {
		t.Fatalf("%+v", end)
	}
	srv := f.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(299) }))
	pedir := func(caminho string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, httptest.NewRequestWithContext(context.Background(), http.MethodGet, caminho, nil))
		return rr
	}
	rr := pedir(end.Endereco)
	corpo, _ := io.ReadAll(rr.Body)
	if rr.Code != 200 || string(corpo) != string(b) || rr.Header().Get("Cache-Control") != "no-store, max-age=0" || rr.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("PDF: %d %v", rr.Code, rr.Header())
	}
	if pedir("/autos/token-errado.pdf").Code != 404 {
		t.Fatal("token inválido deve dar 404")
	}
	if pedir("/index.html").Code != 299 {
		t.Fatal("outros caminhos seguem para os assets")
	}
	d := f.Destaque(1, 0, 4) // "TOPO"
	if d.Erro != nil || len(d.Caixas) != 1 || d.Caixas[0].Y0 > 80 || d.Largura < 590 {
		t.Fatalf("destaque: %+v", d)
	}
	if r := f.Buscar("base direita"); r.Erro != nil || len(r.Ocorrencias) != 1 || r.Ocorrencias[0].Pagina != 1 {
		t.Fatalf("busca: %+v", r)
	}
	// Tarefa encerrada: o endereço antigo deixa de funcionar.
	_, _ = g.Cancelar()
	if pedir(end.Endereco).Code != 404 {
		t.Fatal("após cancelar, o PDF não pode mais ser servido")
	}
}

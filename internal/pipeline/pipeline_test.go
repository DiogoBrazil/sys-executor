package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/ocr"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/session"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/pdfsint"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

var (
	extOnce sync.Once
	ext     pdf.Extrator
)

func TestMain(m *testing.M) {
	c := m.Run()
	if ext != nil {
		_ = ext.Fechar()
	}
	os.Exit(c)
}

// ocrFalso: termo de inquirição legível (confiança alta) ou rabisco (baixa).
type ocrFalso struct{ conf float64 }

func (o ocrFalso) Reconhecer(ctx context.Context, img image.Image) (ocr.Resultado, error) {
	ws := []ocr.Palavra{{Texto: "TERMO", Linha: 0}, {Texto: "DE", Linha: 0}, {Texto: "INQUIRIÇÃO", Linha: 0}, {Texto: "gravação", Linha: 1}}
	for i := range ws {
		ws[i].Confianca = o.conf
		ws[i].X1, ws[i].Y1 = 10, 10
	}
	return ocr.Resultado{Palavras: ws, Confianca: o.conf}, nil
}

func montar(t *testing.T, conf float64) (*workflow.Gerenciador, string) {
	t.Helper()
	extOnce.Do(func() { ext = pdf.NovoExtrator() })
	p := &Pipeline{Extrator: ext, LocalizarOCR: func() (ocr.Motor, error) { return ocrFalso{conf: conf}, nil }, DPI: 72}
	raiz := t.TempDir()
	g := workflow.Novo(workflow.Opcoes{Sessoes: session.NovoGerenciador(raiz), Etapas: p.Etapas()})
	t.Cleanup(g.Encerrar)
	return g, raiz
}

// prazoEspera: -race no CI deixa o pdfium (wasm) lento.
var prazoEspera = 4 * time.Minute

func esperar(t *testing.T, g *workflow.Gerenciador, cond func(*workflow.Visao) bool) *workflow.Visao {
	t.Helper()
	prazo := prazoEspera
	if d, err := time.ParseDuration(os.Getenv("SYSEXEC_TESTE_PRAZO")); err == nil {
		prazo = d // diagnóstico local de um teste parado
	}
	fim := time.Now().Add(prazo) // -race no CI deixa o pdfium (wasm) lento
	for time.Now().Before(fim) {
		if v := g.Visao(); v != nil && cond(v) {
			return v
		}
		time.Sleep(10 * time.Millisecond)
	}
	v := g.Visao()
	if v != nil && v.Pergunta != nil {
		b, _ := json.Marshal(v.Pergunta.Campos)
		t.Fatalf("tempo esgotado na pergunta %q (%s) %.600s: %+v", v.Pergunta.ID, v.Pergunta.Titulo, b, v)
	}
	t.Fatalf("tempo esgotado: %+v", v)
	return nil
}

func enviar(t *testing.T, g *workflow.Gerenciador, tipo workflow.Tipo, pdfBytes []byte) {
	t.Helper()
	if _, err := g.Iniciar(); err != nil {
		t.Fatal(err)
	}
	_, _ = g.EscolherTipo(tipo)
	_, _ = g.EscolherEntrada(workflow.EntradaUpload)
	if _, err := g.ReceberUpload(pdfBytes); err != nil {
		t.Fatal(err)
	}
}

func fixtureSEI(t *testing.T) []byte {
	b, err := os.ReadFile("../../testdata/autos/sei-sintetico.pdf")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// PADS compatível: percorre verificação → extração → classificação →
// compatibilidade e chega à delimitação, com o resumo publicado. Sem IA
// configurada, pára com IA-001 (recuperável: "Tentar novamente").
func TestPADSFixtureAteDelimitacao(t *testing.T) {
	g, _ := montar(t, 90)
	enviar(t, g, workflow.TipoPADS, fixtureSEI(t))
	v := esperar(t, g, func(v *workflow.Visao) bool { return v.Parada || v.Terminal })
	if v.Estado != workflow.Delimitando || v.Erro == nil || v.Erro.Codigo != "IA-001" || !v.Retomavel {
		t.Fatalf("%+v", v)
	}
	a := v.Autos
	if a == nil || a.TotalPaginas != 9 || len(a.Pecas) != 7 || a.Compatibilidade != "COMPATIVEL" || a.PecaCentral != "PEC-01" {
		t.Fatalf("resumo: %+v", a)
	}
	if a.Pecas[2].Funcao != "DEFESA_PREVIA" || !a.Pecas[2].Provavel || a.Pecas[6].Funcao != "ALEGACOES_FINAIS" {
		t.Fatalf("A01/A02: %+v / %+v", a.Pecas[2], a.Pecas[6])
	}
	if len(a.Midias) < 4 {
		t.Fatalf("A08: mídias %+v", a.Midias)
	}
	d := g.Dados().(*Dados)
	// O termo de inquirição da fixture tem 2 páginas com o mesmo corpo (7 e 8).
	if oc, err := d.Indice.Buscar("bebedouro", 5); err != nil || len(oc) != 2 || oc[0].Pagina+oc[1].Pagina != 15 {
		t.Fatalf("busca: %+v %v", oc, err)
	}
	// Nada do texto dos autos vai no resumo enviado à interface.
	if strings.Contains(fmt.Sprintf("%+v", a), "bebedouro") {
		t.Fatal("resumo não pode conter texto dos autos")
	}
}

// T04: SR escolhido, autos de PADS → pergunta, sem trocar o tipo.
func TestT04PerguntaSemTrocarTipo(t *testing.T) {
	g, _ := montar(t, 90)
	enviar(t, g, workflow.TipoSR, fixtureSEI(t))
	v := esperar(t, g, func(v *workflow.Visao) bool { return v.Estado == workflow.AguardandoEsclarecimento })
	if v.Tipo != workflow.TipoSR || v.Pergunta == nil || !strings.Contains(v.Pergunta.Titulo, "Julgamento de PADS") {
		t.Fatalf("%+v %+v", v, v.Pergunta)
	}
	_, _ = g.Responder(v.Pergunta.ID, "continuar", nil)
	v = esperar(t, g, func(v *workflow.Visao) bool { return v.Parada })
	if v.Estado != workflow.Delimitando || v.Tipo != workflow.TipoSR {
		t.Fatalf("%+v", v)
	}
}

// PDF protegido: volta ao envio com PDF-004, mantendo o tipo.
func TestPDFProtegidoVoltaAoEnvio(t *testing.T) {
	g, _ := montar(t, 90)
	protegido := []byte("%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]>>endobj\n4 0 obj<</Filter/Standard/V 1/R 2/O<" +
		strings.Repeat("A1", 32) + ">/U<" + strings.Repeat("A1", 32) + ">/P -4>>endobj\ntrailer<</Size 5/Root 1 0 R/Encrypt 4 0 R/ID[<00112233445566778899AABBCCDDEEFF><00112233445566778899AABBCCDDEEFF>]>>\n%%EOF\n")
	enviar(t, g, workflow.TipoPADS, protegido)
	v := esperar(t, g, func(v *workflow.Visao) bool { return v.Estado == workflow.RecebendoUpload && v.Erro != nil })
	if v.Erro.Codigo != "PDF-004" && v.Erro.Codigo != "PDF-003" {
		t.Fatalf("%+v", v.Erro)
	}
	if v.Tipo != workflow.TipoPADS || v.Terminal {
		t.Fatalf("%+v", v)
	}
}

// ING-COB-01 / T17: 800 páginas, 3 vazias e 5 com OCR ruim → mapa exato,
// nenhuma página fora de uma peça, progresso real.
func TestINGCOB01OitocentasPaginas(t *testing.T) {
	if testing.Short() {
		t.Skip("teste longo")
	}
	vazias := map[int]bool{100: true, 401: true, 799: true}
	ruins := map[int]bool{50: true, 51: true, 300: true, 650: true, 777: true}
	var ps []pdfsint.Pagina
	doc := 0
	for n := 1; n <= 800; n++ {
		switch {
		case vazias[n]:
			ps = append(ps, pdfsint.Pagina{})
		case ruins[n]:
			ps = append(ps, pdfsint.Pagina{Desenho: true})
		default:
			if n == 1 || n%10 == 1 {
				doc++
			}
			titulo := "continuação do documento"
			if n == 1 {
				titulo = "MEMORANDO DISCIPLINAR Nº 1/2026"
			} else if n%10 == 1 {
				titulo = "TERMO DE INQUIRIÇÃO DE TESTEMUNHA"
			}
			ps = append(ps, pdfsint.Pagina{
				Linhas: []string{titulo, "Texto fictício da página com conteúdo suficiente para ser lido diretamente,",
					"sem necessidade de reconhecimento óptico, e com as palavras de que o leitor precisa."},
				Rodape: fmt.Sprintf("Referência: Processo nº 0000.000001/2026-01 SEI nº %07d", 9000000+doc),
			})
		}
	}
	g, _ := montar(t, 30) // páginas de desenho: OCR com confiança 30 → ilegível
	inicio := time.Now()
	enviar(t, g, workflow.TipoPADS, pdfsint.Gerar(ps))
	viuProgresso := false
	v := esperar(t, g, func(v *workflow.Visao) bool {
		if v.Progresso != nil && v.Progresso.Unidade == "páginas" && v.Progresso.Total == 800 {
			viuProgresso = true
		}
		return v.Parada || v.Terminal || v.Estado == workflow.AguardandoEsclarecimento
	})
	t.Logf("800 páginas em %v", time.Since(inicio))
	a := v.Autos
	if a == nil || a.TotalPaginas != 800 {
		t.Fatalf("%+v", v)
	}
	cont := map[string]int{}
	for _, p := range a.Paginas {
		cont[p.Situacao]++
		if p.Peca == "" {
			t.Errorf("pág. %d fora de qualquer peça", p.Numero)
		}
	}
	if cont["VAZIA"] != 3 || cont["ILEGIVEL"] != 5 || cont["LIDA"] != 792 {
		t.Fatalf("cobertura: %+v", cont)
	}
	for n := range ruins {
		if a.Paginas[n-1].Situacao != "ILEGIVEL" {
			t.Errorf("pág. %d deveria estar ilegível", n)
		}
	}
	if !viuProgresso {
		t.Log("progresso não amostrado (extração rápida); coberto em internal/ingest/autos")
	}
}

// Sem Tesseract, um *Tesseract nulo dentro de ocr.Motor não pode chegar à
// extração (o app caía com ponteiro nulo ao encontrar página digitalizada):
// a tarefa para com OCR-001, sem pânico.
func TestOCRAusenteNaoDerrubaOApp(t *testing.T) {
	extOnce.Do(func() { ext = pdf.NovoExtrator() })
	p := &Pipeline{Extrator: ext, DPI: 72, LocalizarOCR: func() (ocr.Motor, error) {
		var tess *ocr.Tesseract
		return tess, fmt.Errorf("tesseract ausente")
	}}
	g := workflow.Novo(workflow.Opcoes{Sessoes: session.NovoGerenciador(t.TempDir()), Etapas: p.Etapas()})
	t.Cleanup(g.Encerrar)
	enviar(t, g, workflow.TipoPADS, fixtureSEI(t))
	v := esperar(t, g, func(v *workflow.Visao) bool {
		return v.Parada || v.Terminal || v.Pergunta != nil || v.Estado == workflow.Delimitando
	})
	t.Logf("estado=%s erro=%+v aviso=%+v", v.Estado, v.Erro, v.Aviso)
	if v.Erro != nil && v.Erro.Codigo != "OCR-001" && v.Erro.Codigo != "IA-001" {
		t.Fatalf("esperado OCR-001 (ou seguir sem OCR), veio %+v", v.Erro)
	}
}

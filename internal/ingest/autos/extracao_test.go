package autos

import (
	"context"
	"image"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/ocr"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/pdfsint"
)

var (
	extOnce sync.Once
	ext     pdf.Extrator
)

func extrator() pdf.Extrator {
	extOnce.Do(func() { ext = pdf.NovoExtrator() })
	return ext
}

func TestMain(m *testing.M) {
	c := m.Run()
	if ext != nil {
		_ = ext.Fechar()
	}
	os.Exit(c)
}

// ocrFalso devolve um termo de inquirição com a confiança configurada.
type ocrFalso struct {
	conf    float64
	chamado atomic.Int32
}

func (o *ocrFalso) Reconhecer(ctx context.Context, img image.Image) (ocr.Resultado, error) {
	o.chamado.Add(1)
	ws := []ocr.Palavra{
		{Texto: "TERMO", Linha: 0, X0: 100, Y0: 100, X1: 300, Y1: 140},
		{Texto: "DE", Linha: 0, X0: 310, Y0: 100, X1: 360, Y1: 140},
		{Texto: "INQUIRIÇÃO", Linha: 0, X0: 370, Y0: 100, X1: 700, Y1: 140},
		{Texto: "gravação", Linha: 1, X0: 100, Y0: 200, X1: 300, Y1: 240},
	}
	for i := range ws {
		ws[i].Confianca = o.conf
	}
	return ocr.Resultado{Palavras: ws, Confianca: o.conf}, nil
}

func lerFixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../../../testdata/autos/sei-sintetico.pdf")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestExtrairFixtureSEI(t *testing.T) {
	o := &ocrFalso{conf: 88}
	var ultimo, total atomic.Int32
	a, err := Extracao{PDF: extrator(), OCR: o, DPI: 150, Workers: 2}.Extrair(context.Background(), lerFixture(t),
		func(at, tot int) { ultimo.Store(int32(at)); total.Store(int32(tot)) })
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Paginas) != 9 {
		t.Fatalf("páginas: %d", len(a.Paginas))
	}
	if ultimo.Load() != 9 || total.Load() != 9 {
		t.Fatalf("progresso deve chegar a 9/9: %d/%d", ultimo.Load(), total.Load())
	}
	esperado := map[int]Situacao{1: Lida, 2: Lida, 3: Lida, 4: OCROk, 5: Lida, 6: Vazia, 7: Lida, 8: Lida, 9: Lida}
	for n, s := range esperado {
		p := a.Paginas[n-1]
		if p.Numero != n || p.Situacao != s {
			t.Errorf("pág. %d: situação %s, esperado %s", n, p.Situacao, s)
		}
	}
	if o.chamado.Load() != 1 {
		t.Errorf("OCR deveria rodar só na página escaneada; rodou %d vezes", o.chamado.Load())
	}
	p1 := a.Paginas[0]
	if !strings.Contains(p1.Texto, "MEMORANDO DISCIPLINAR") || !strings.Contains(p1.Texto, "(0090000001) SEI 0000.000001/2026-01 / pg. 1") {
		t.Fatalf("texto da pág. 1: %q", p1.Texto)
	}
	// O realce do título cai no alto da página.
	ini := len([]rune(p1.Texto[:strings.Index(p1.Texto, "MEMORANDO")]))
	cx := p1.Caixas(ini, ini+9)
	if len(cx) == 0 || cx[0].Y0 > 120 {
		t.Fatalf("caixa do título: %+v", cx)
	}
	p4 := a.Paginas[3]
	if len(p4.Carimbos) != 1 || p4.Carimbos[0].SEI != "0090000007" {
		t.Fatalf("o carimbo da página escaneada deve ser lido: %+v", p4.Carimbos)
	}
	if c := a.Paginas[8].Carimbos; len(c) != 2 {
		t.Fatalf("pág. 9 tem carimbo antigo e atual: %+v", c)
	}
	if p4.Origem != OrigemOCR || !strings.HasPrefix(p4.Texto, "TERMO DE INQUIRIÇÃO\ngravação") {
		t.Fatalf("pág. 4 (OCR): %+v", p4)
	}
	// 150 dpi: 100 px → 48 pt.
	if c := p4.Caixas(0, 5); len(c) != 1 || c[0].X0 != 48 {
		t.Fatalf("conversão px→pt: %+v", c)
	}
	if c := a.Contagem(); c[Lida] != 7 || c[OCROk] != 1 || c[Vazia] != 1 {
		t.Fatalf("contagem: %+v", c)
	}
}

func TestSituacaoPorConfiancaESemOCR(t *testing.T) {
	for conf, s := range map[float64]Situacao{60: OCRBaixo, 30: Ilegivel} {
		a, err := Extracao{PDF: extrator(), OCR: &ocrFalso{conf: conf}, DPI: 100}.Extrair(context.Background(), lerFixture(t), nil)
		if err != nil {
			t.Fatal(err)
		}
		if a.Paginas[3].Situacao != s {
			t.Errorf("conf %.0f: esperado %s, veio %s", conf, s, a.Paginas[3].Situacao)
		}
	}
	if _, err := (Extracao{PDF: extrator(), DPI: 100}).Extrair(context.Background(), lerFixture(t), nil); err == nil {
		t.Fatal("sem OCR, a página escaneada deve falhar (OCR-001), nunca ser ignorada")
	}
}

func TestTextoUtil(t *testing.T) {
	casos := map[string]bool{
		"":                           false,
		"   12  ":                    false,
		"MEMORANDO DISCIPLINAR Nº 1": true,
		"Ao SD PM FULANO, fica notificado de que responde a processo pela prática da transgressão prevista no art. 16 e que tem o prazo de cinco dias para a defesa, com a juntada de documentos e a indicação de testemunhas por escrito.": true,
		strings.Repeat("ÿþ€‡†", 20):                   false,
		strings.Repeat("xq zrt bvn klp wqs mnb ", 10): false,
	}
	for txt, esperado := range casos {
		if TextoUtil(txt) != esperado {
			t.Errorf("TextoUtil(%q) != %v", txt[:min(len(txt), 30)], esperado)
		}
	}
}

// Integração com o Tesseract real (Windows): SYSEXEC_TESSERACT aponta para o executável.
func TestExtrairComTesseractReal(t *testing.T) {
	if os.Getenv("SYSEXEC_TESSERACT") == "" {
		t.Skip("SYSEXEC_TESSERACT não definido")
	}
	m, err := ocr.Localizar("")
	if err != nil {
		t.Fatal(err)
	}
	a, err := Extracao{PDF: extrator(), OCR: m}.Extrair(context.Background(), lerFixture(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	p := a.Paginas[3]
	t.Logf("pág. 4: situação %s, confiança %.1f", p.Situacao, p.Confianca)
	if p.Origem != OrigemOCR || (p.Situacao != OCROk && p.Situacao != OCRBaixo) {
		t.Fatalf("%+v", p.Situacao)
	}
	for _, w := range []string{"TERMO", "INQUIRI", "testemunha", "câmeras"} {
		if !strings.Contains(p.Texto, w) {
			t.Errorf("OCR não leu %q: %q", w, p.Texto)
		}
	}
}

// PDFs com uma caixa por letra não podem virar "G O V E R N O" (espaço só
// onde há distância real entre as letras).
func TestTextoLetraALetra(t *testing.T) {
	b := pdfsint.Gerar([]pdfsint.Pagina{{LetraALetra: true, Linhas: []string{
		"GOVERNO DO ESTADO DE RONDONIA",
		"Memorando Disciplinar com texto suficiente para leitura direta da camada de texto",
	}}})
	a, err := Extracao{PDF: extrator(), OCR: &ocrFalso{conf: 90}}.Extrair(context.Background(), b, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := a.Paginas[0]
	if !strings.Contains(p.Texto, "GOVERNO DO ESTADO DE RONDONIA") || !strings.Contains(p.Texto, "Memorando Disciplinar com texto") {
		t.Fatalf("texto montado: %q", p.Texto)
	}
	if p.Situacao != Lida {
		t.Fatalf("situação: %s", p.Situacao)
	}
}

// LerTextoNativo não chama OCR: página que só tem o carimbo (escaneada) volta com
// o carimbo do rodapé e sem falhar (separação dos documentos do SEI).
func TestLerTextoNativoSemOCR(t *testing.T) {
	b := pdfsint.Gerar([]pdfsint.Pagina{
		{Linhas: []string{"MEMORANDO DISCIPLINAR Nº 1/2026"}, Rodape: "Memorando 1 (0010000) SEI 0000.000001/2026-00 / pg. 1"},
		{Rodape: "Termo 2 (0010001) SEI 0000.000001/2026-00 / pg. 2"}, // só o carimbo (escaneada)
	})
	a, err := LerTextoNativo(context.Background(), extrator(), b)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Paginas) != 2 || len(a.Paginas[0].Carimbos) != 1 || len(a.Paginas[1].Carimbos) != 1 {
		t.Fatalf("%+v", a.Paginas)
	}
	if a.Paginas[1].Carimbos[0].SEI != "0010001" || !strings.Contains(a.Paginas[0].Texto, "MEMORANDO") {
		t.Fatalf("carimbos %+v", a.Paginas[1].Carimbos)
	}
}

package ocr

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"os"
	"testing"
)

const tsvExemplo = "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
	"1\t1\t0\t0\t0\t0\t0\t0\t1654\t2339\t-1\t\n" +
	"5\t1\t1\t1\t1\t1\t100\t200\t180\t40\t96.5\tMEMORANDO\n" +
	"5\t1\t1\t1\t1\t2\t300\t200\t230\t40\t91.0\tDISCIPLINAR\n" +
	"5\t1\t1\t1\t2\t1\t100\t260\t60\t30\t40.0\tem\n" +
	"5\t1\t1\t1\t2\t2\t170\t260\t200\t30\t88.0\t10/08/2026\n" +
	"5\t1\t1\t1\t2\t3\t380\t260\t10\t30\t-1\t\n"

func TestLerTSV(t *testing.T) {
	r, err := LerTSV([]byte(tsvExemplo))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Palavras) != 4 {
		t.Fatalf("palavras: %+v", r.Palavras)
	}
	if got := r.Texto(); got != "MEMORANDO DISCIPLINAR\nem 10/08/2026" {
		t.Fatalf("texto: %q", got)
	}
	p := r.Palavras[1]
	if p.X0 != 300 || p.Y0 != 200 || p.X1 != 530 || p.Y1 != 240 {
		t.Fatalf("caixa: %+v", p)
	}
	// média ponderada: (9*96.5 + 11*91 + 2*40 + 10*88) / 32
	if r.Confianca < 88.40 || r.Confianca > 88.45 { // 2829,5 / 32 = 88,42
		t.Fatalf("confiança: %.2f", r.Confianca)
	}
}

func TestLerTSVVazioEInvalido(t *testing.T) {
	r, err := LerTSV([]byte("level\tpage_num\n"))
	if err != nil || len(r.Palavras) != 0 || r.Confianca != 0 {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := LerTSV([]byte("isto não é tsv\nnem isto\n")); err == nil {
		t.Fatal("saída inesperada deveria falhar")
	}
}

// Integração com o Tesseract real (Windows): defina SYSEXEC_TESSERACT.
func TestTesseractReal(t *testing.T) {
	if os.Getenv("SYSEXEC_TESSERACT") == "" {
		t.Skip("SYSEXEC_TESSERACT não definido")
	}
	m, err := Localizar("")
	if err != nil {
		t.Fatal(err)
	}
	branca := image.NewRGBA(image.Rect(0, 0, 800, 300))
	draw.Draw(branca, branca.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	r, err := m.Reconhecer(context.Background(), branca)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Palavras) != 0 {
		t.Fatalf("imagem em branco não deveria ter palavras: %+v", r.Palavras)
	}
}

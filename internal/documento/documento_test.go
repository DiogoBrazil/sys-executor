package documento

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/minuta"
)

// Conteúdo fictício, com os caracteres que costumam quebrar XML e Typst.
func exemplo() *minuta.Documento {
	return &minuta.Documento{Procedimento: "PADS", Blocos: []minuta.Bloco{
		{Tipo: minuta.Titulo, Texto: "JULGAMENTO DE PADS", Origem: minuta.Fixo},
		{Tipo: minuta.Cabecalho, Texto: "Autoridade disciplinar: " + minuta.APreencher, Origem: minuta.Fixo, Pendente: true},
		{Tipo: minuta.Cabecalho, Texto: "Acusado(a): SD PM FULANO FICTÍCIO, matrícula 000000000", Origem: minuta.Fixo},
		{Tipo: minuta.Secao, Texto: "1. RELATÓRIO", Origem: minuta.Fixo},
		{Tipo: minuta.Paragrafo, Texto: `Trata-se de texto fictício: ação, à, ônus, “aspas”, travessão —, A & B, 2 < 3 > 1, barra \ e "aspas retas", #cerquilha, $cifrão, *asterisco*, _sublinhado_, @arroba e [colchetes].`, Origem: minuta.IA},
		{Tipo: minuta.Item, Texto: "I – primeiro item;", Origem: minuta.Fixo},
		{Tipo: minuta.Citacao, Texto: "Texto de norma transcrito.", Origem: minuta.Fixo},
		{Tipo: minuta.Secao, Texto: "3. DECISÃO", Origem: minuta.Fixo},
		{Tipo: minuta.Subtitulo, Texto: "Quanto à transgressão do art. 16, inciso XVI, do RDPM", Origem: minuta.Fixo},
		{Tipo: minuta.Paragrafo, Texto: strings.Repeat("Parágrafo longo para forçar a quebra de página e a numeração. ", 90), Origem: minuta.Fixo},
		{Tipo: minuta.Fecho, Texto: "CUMPRA-SE.", Origem: minuta.Fixo},
		{Tipo: minuta.Fecho, Texto: minuta.APreencher + " (local), data conforme a assinatura eletrônica.", Origem: minuta.Fixo, Pendente: true},
		{Tipo: minuta.Assinatura, Texto: minuta.APreencher + " (nome, posto e função da autoridade disciplinar)", Origem: minuta.Fixo, Pendente: true},
	}}
}

func logoDeTeste(t *testing.T) []byte {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 60, 40))
	for i := range im.Pix {
		im.Pix[i] = 200
	}
	im.Set(1, 1, color.Black)
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func cabecalhoDeTeste(t *testing.T) Cabecalho {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "cabecalho"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cabecalho", "logo-pmro.png"), logoDeTeste(t), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SYSEXEC_LOGO", "")
	c := CarregarCabecalho(dir)
	if len(c.Logo) == 0 || c.Ext != "png" || c.LogoW != 60 || c.LogoH != 40 || len(c.Linhas) != 2 {
		t.Fatalf("cabeçalho: %+v", c)
	}
	return c
}

func TestCabecalhoSemLogo(t *testing.T) {
	t.Setenv("SYSEXEC_LOGO", "")
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "cabecalho"), 0o700)
	os.WriteFile(filepath.Join(dir, "cabecalho", "logo-pmro.png"), []byte("não é imagem"), 0o600)
	if c := CarregarCabecalho(dir); len(c.Logo) != 0 || len(c.Linhas) != 2 {
		t.Fatalf("arquivo inválido não pode virar logotipo: %+v", c)
	}
}

// Regra: TRV-EXP-001, TRV-EXP-002, TRV-EXP-003
func TestDOCX(t *testing.T) {
	for _, comLogo := range []bool{true, false} {
		cab := Cabecalho{Linhas: LinhasPadrao}
		if comLogo {
			cab = cabecalhoDeTeste(t)
		}
		b, err := DOCX(exemplo(), cab)
		if err != nil {
			t.Fatal(err)
		}
		z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			t.Fatal(err)
		}
		partes := map[string]string{}
		for _, f := range z.File {
			r, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			c, _ := io.ReadAll(r)
			r.Close()
			partes[f.Name] = string(c)
			if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
				dec := xml.NewDecoder(bytes.NewReader(c))
				for {
					if _, err := dec.Token(); err == io.EOF {
						break
					} else if err != nil {
						t.Fatalf("%s não é XML bem formado: %v", f.Name, err)
					}
				}
			}
			if !f.Modified.IsZero() && f.Modified.Year() > 1980 {
				t.Fatalf("%s registra a data de geração", f.Name)
			}
		}
		doc := partes["word/document.xml"]
		for _, quero := range []string{"JULGAMENTO DE PADS", "A &amp; B, 2 &lt; 3 &gt; 1", `<w:highlight w:val="yellow"/></w:rPr><w:t xml:space="preserve">[A PREENCHER]`,
			`w:pStyle w:val="Secao"`, `w:pStyle w:val="Assinatura"`, `w:top="1134" w:right="850" w:bottom="1134" w:left="1701"`} {
			if !strings.Contains(doc, quero) {
				t.Fatalf("document.xml sem %q", quero)
			}
		}
		if !strings.Contains(partes["word/styles.xml"], `w:ascii="Times New Roman"`) || !strings.Contains(partes["word/styles.xml"], `<w:sz w:val="24"/>`) ||
			!strings.Contains(partes["word/header1.xml"], "POLÍCIA MILITAR DO ESTADO DE RONDÔNIA") || !strings.Contains(partes["word/footer1.xml"], " PAGE ") {
			t.Fatal("estilos, cabeçalho ou rodapé fora do padrão (V4 §32)")
		}
		if _, tem := partes["word/media/logotipo.png"]; tem != comLogo || strings.Contains(partes["word/header1.xml"], "w:drawing") != comLogo {
			t.Fatalf("logotipo: comLogo=%v", comLogo)
		}
		// Sem autor, sem data e sem assinatura simulada.
		if c := partes["docProps/core.xml"]; strings.Contains(c, "creator") || strings.Contains(c, "lastModifiedBy") || strings.Contains(c, "created") {
			t.Fatalf("metadados demais: %s", c)
		}
	}
	if _, err := DOCX(&minuta.Documento{}, Cabecalho{}); err == nil {
		t.Fatal("minuta vazia devia falhar")
	}
}

func TestFonteTypstNaoInterpretaOTexto(t *testing.T) {
	src := FonteTypst(exemplo(), Cabecalho{Linhas: LinhasPadrao})
	// Todo o conteúdo vai dentro de literais de string: nenhuma marcação do
	// texto pode ficar solta no código.
	for _, quero := range []string{`#"JULGAMENTO DE PADS"`, `barra \\ e \"aspas retas\", #cerquilha`, `#highlight(fill: rgb("#fff176"))[#"[A PREENCHER]"]`,
		`font: ("Times New Roman", "Liberation Serif"), size: 12pt`, `left: 3cm, right: 1.5cm`, `first-line-indent: (amount: 2.5cm, all: true)`} {
		if !strings.Contains(src, quero) {
			t.Fatalf("fonte Typst sem %q:\n%s", quero, src)
		}
	}
	if strings.Contains(src, "#image(") {
		t.Fatal("sem logotipo não há imagem")
	}
}

// Regra: TRV-EXP-004
func TestNomeArquivo(t *testing.T) {
	q := time.Date(2026, 11, 11, 14, 5, 0, 0, time.UTC)
	if n := NomeArquivo("PADS", 3, q, FormatoDOCX); n != "Minuta_Julgamento_PADS_v3_2026-11-11_1405.docx" {
		t.Fatal(n)
	}
	if n := NomeArquivo("SR", 0, q, FormatoPDF); n != "Minuta_Relatorio_Sindicancia_Regular_v1_2026-11-11_1405.pdf" {
		t.Fatal(n)
	}
}

var reEspacos = regexp.MustCompile(`[\s\x{00AD}]+`)

// O PDF é gerado de verdade quando há um Typst na máquina (SYSEXEC_TYPST);
// o texto é lido de volta com o pdfium e comparado com a minuta.
// Regra: TRV-EXP-005
func TestPDFComTypst(t *testing.T) {
	ty, err := LocalizarTypst(t.TempDir())
	if err != nil {
		t.Skip("Typst não disponível (defina SYSEXEC_TYPST com o executável)")
	}
	pasta := t.TempDir()
	doc := exemplo()
	b, err := ty.PDF(context.Background(), doc, cabecalhoDeTeste(t), pasta)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatal("saída não é PDF")
	}
	if es, _ := os.ReadDir(pasta); len(es) != 0 {
		t.Fatalf("intermediários do PDF não foram apagados: %v", es)
	}
	ex := pdf.NovoExtrator()
	defer ex.Fechar()
	d, err := ex.Abrir(context.Background(), b)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Fechar()
	if d.Paginas() < 2 {
		t.Fatalf("esperava mais de uma página, veio %d", d.Paginas())
	}
	var sb strings.Builder
	for i := 0; i < d.Paginas(); i++ {
		p, err := d.Pagina(context.Background(), i)
		if err != nil {
			t.Fatal(err)
		}
		if p.Largura < 594 || p.Largura > 597 || p.Altura < 840 || p.Altura > 843 {
			t.Fatalf("página %d não é A4: %.0f × %.0f", i+1, p.Largura, p.Altura)
		}
		for _, c := range p.Caracteres {
			sb.WriteRune(c.Runa)
		}
		sb.WriteByte(' ')
	}
	limpo := reEspacos.ReplaceAllString(strings.ReplaceAll(sb.String(), "-", ""), "")
	for _, quero := range []string{"POLÍCIA MILITAR DO ESTADO DE RONDÔNIA", "JULGAMENTO DE PADS", "Autoridade disciplinar: [A PREENCHER]",
		`A & B, 2 < 3 > 1, barra \ e "aspas retas", #cerquilha, $cifrão, *asterisco*, _sublinhado_, @arroba e [colchetes].`, "CUMPRA-SE."} {
		if !strings.Contains(limpo, reEspacos.ReplaceAllString(strings.ReplaceAll(quero, "-", ""), "")) {
			t.Fatalf("PDF sem %q:\n%.1500s", quero, sb.String())
		}
	}
}

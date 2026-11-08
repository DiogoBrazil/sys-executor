// PoC F0-06 — Um DocModel → DOCX (writer OOXML próprio: archive/zip + XML) e PDF (Typst).
// Conteúdo 100% fictício. Cabeçalho conforme V4 §32.1; parâmetros §32.2 (proposta).
//
// Uso: go run . -out <dir> [-titulo "JULGAMENTO DE PADS"] [-logo logo.png]
// Sem -logo, gera um placeholder "brasão pendente" (somente desenvolvimento).
package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// ---- DocModel ----

type Kind int

const (
	Secao Kind = iota
	Paragrafo
	Alinea     // rótulo + texto (texto vazio = só rótulo, como nas alíneas da SR)
	Assinatura // espaço de assinatura, sem assinatura
)

type Bloco struct {
	Kind   Kind
	Rotulo string
	Texto  string
}

type Doc struct {
	Titulo string // "JULGAMENTO DE PADS" | "RELATÓRIO DE SINDICÂNCIA REGULAR"
	Blocos []Bloco
	Logo   []byte // PNG
	LogoW  int    // px
	LogoH  int
}

var cabecalho = []string{"SECRETARIA DE SEGURANÇA, DEFESA E CIDADANIA", "POLÍCIA MILITAR DO ESTADO DE RONDÔNIA"}

func exemplo(titulo string) Doc {
	return Doc{Titulo: titulo, Blocos: []Bloco{
		{Kind: Secao, Texto: "I – SÍNTESE DOS FATOS (FICTÍCIO)"},
		{Kind: Paragrafo, Texto: "Trata-se de minuta fictícia, gerada exclusivamente para teste de formatação, acentuação e paginação: ação, atenção, conclusão, ônus, pôr, à, às, último, Rondônia, Itaú, coração, pão, “aspas” e travessão — ç Ç ã Ã õ Õ é É ê Ê í Í ó Ó ú Ú ü."},
		{Kind: Paragrafo, Texto: "O SD PM FULANO ALFA FICTÍCIO apresentou defesa prévia (SEI nº 0090000002), negando a conduta descrita no item 1 do Memorando Disciplinar nº 01/2026."},
		{Kind: Secao, Texto: "II – QUESITOS"},
		{Kind: Alinea, Rotulo: "a)", Texto: "houve a transgressão disciplinar?"},
		{Kind: Alinea, Rotulo: "b)", Texto: ""},
		{Kind: Alinea, Rotulo: "c)", Texto: "há circunstâncias atenuantes ou agravantes a considerar, nos termos do RDPM/RO?"},
		{Kind: Secao, Texto: "III – CONCLUSÃO"},
		{Kind: Paragrafo, Texto: strings.Repeat("Parágrafo longo para forçar quebra de página e verificar a numeração automática no rodapé, a justificação e o recuo de primeira linha. ", 30)},
		{Kind: Paragrafo, Texto: "Porto Velho, data pendente de definição pela autoridade."},
		{Kind: Assinatura, Texto: "Autoridade responsável (identificação pendente)"},
	}}
}

// ---- Placeholder do brasão (somente dev) ----

func placeholder() ([]byte, int, int) {
	w, h := 240, 240
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{235, 235, 235, 255}
			if x < 4 || y < 4 || x >= w-4 || y >= h-4 || x == y || x == w-1-y {
				c = color.RGBA{150, 150, 150, 255}
			}
			im.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	png.Encode(&b, im)
	return b.Bytes(), w, h
}

// ---- DOCX ----

const emuPorCm = 360000
const twipPorCm = 567

func esc(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func run(texto string, bold bool) string {
	rpr := ""
	if bold {
		rpr = "<w:rPr><w:b/></w:rPr>"
	}
	return `<w:r>` + rpr + `<w:t xml:space="preserve">` + esc(texto) + `</w:t></w:r>`
}

func par(estilo, conteudo string) string {
	return `<w:p><w:pPr><w:pStyle w:val="` + estilo + `"/></w:pPr>` + conteudo + `</w:p>`
}

func docx(d Doc) []byte {
	var body strings.Builder
	body.WriteString(par("Titulo", run(d.Titulo, true)))
	for _, b := range d.Blocos {
		switch b.Kind {
		case Secao:
			body.WriteString(par("Secao", run(b.Texto, true)))
		case Paragrafo:
			body.WriteString(par("Corpo", run(b.Texto, false)))
		case Alinea:
			t := b.Rotulo
			if b.Texto != "" {
				t += " " + b.Texto
			}
			body.WriteString(par("Alinea", run(t, false)))
		case Assinatura:
			body.WriteString(par("Assinatura", run("_______________________________________", false)))
			body.WriteString(par("Assinatura", run(b.Texto, false)))
		}
	}
	// Margens §32.2: sup/inf 2 cm, esq 3 cm, dir 1,5 cm. Cabeçalho/rodapé a 1 cm.
	sect := fmt.Sprintf(`<w:sectPr><w:headerReference w:type="default" r:id="rIdH"/><w:footerReference w:type="default" r:id="rIdF"/>`+
		`<w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="%d" w:right="%d" w:bottom="%d" w:left="%d" w:header="%d" w:footer="%d" w:gutter="0"/></w:sectPr>`,
		2*twipPorCm, 3*twipPorCm/2, 2*twipPorCm, 3*twipPorCm, twipPorCm, twipPorCm)
	document := xmlHead + `<w:document ` + ns + `><w:body>` + body.String() + sect + `</w:body></w:document>`

	// Brasão: 2,5 cm de altura, proporcional.
	hEmu := int(2.5 * emuPorCm)
	wEmu := hEmu * d.LogoW / d.LogoH
	img := fmt.Sprintf(`<w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:docPr id="1" name="Brasao"/>`+
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic><pic:nvPicPr><pic:cNvPr id="1" name="brasao.png"/><pic:cNvPicPr/></pic:nvPicPr>`+
		`<pic:blipFill><a:blip r:embed="rIdImg"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`,
		wEmu, hEmu, wEmu, hEmu)
	header := xmlHead + `<w:hdr ` + ns + `>` + par("Cabecalho", img) + par("Cabecalho", run(cabecalho[0], true)) + par("Cabecalho", run(cabecalho[1], true)) + `</w:hdr>`
	footer := xmlHead + `<w:ftr ` + ns + `><w:p><w:pPr><w:jc w:val="right"/></w:pPr>` +
		`<w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText xml:space="preserve"> PAGE </w:instrText></w:r><w:r><w:fldChar w:fldCharType="separate"/></w:r><w:r><w:t>1</w:t></w:r><w:r><w:fldChar w:fldCharType="end"/></w:r></w:p></w:ftr>`

	files := []struct{ name, data string }{
		{"[Content_Types].xml", xmlHead + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Default Extension="png" ContentType="image/png"/>` +
			`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
			`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>` +
			`<Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>` +
			`<Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>` +
			`<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/></Types>`},
		{"_rels/.rels", xmlHead + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
			`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/></Relationships>`},
		// Metadados mínimos: sem autor, sem nomes de pessoas.
		{"docProps/core.xml", xmlHead + `<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Minuta (rascunho)</dc:title></cp:coreProperties>`},
		{"word/_rels/document.xml.rels", xmlHead + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rIdS" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
			`<Relationship Id="rIdH" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>` +
			`<Relationship Id="rIdF" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/></Relationships>`},
		{"word/_rels/header1.xml.rels", xmlHead + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rIdImg" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/brasao.png"/></Relationships>`},
		{"word/document.xml", document},
		{"word/styles.xml", styles},
		{"word/header1.xml", header},
		{"word/footer1.xml", footer},
		{"word/media/brasao.png", string(d.Logo)},
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for _, f := range files {
		w, _ := z.Create(f.name)
		w.Write([]byte(f.data))
	}
	z.Close()
	return buf.Bytes()
}

const xmlHead = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"
const ns = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
	`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"`

// TNR 12 (24 meios-pontos), entrelinha simples (240), recuo 1ª linha 2,5 cm (1418 twips), justificado.
var styles = xmlHead + `<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
	`<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Times New Roman" w:hAnsi="Times New Roman" w:eastAsia="Times New Roman" w:cs="Times New Roman"/><w:color w:val="000000"/><w:sz w:val="24"/><w:szCs w:val="24"/><w:lang w:val="pt-BR"/></w:rPr></w:rPrDefault>` +
	`<w:pPrDefault><w:pPr><w:spacing w:before="0" w:after="120" w:line="240" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>` +
	`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Corpo"><w:name w:val="Corpo"/><w:basedOn w:val="Normal"/><w:pPr><w:ind w:firstLine="1418"/><w:jc w:val="both"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Titulo"><w:name w:val="Titulo"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="240" w:after="360"/><w:jc w:val="center"/></w:pPr><w:rPr><w:b/><w:caps/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Secao"><w:name w:val="Secao"/><w:basedOn w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="240" w:after="120"/></w:pPr><w:rPr><w:b/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Alinea"><w:name w:val="Alinea"/><w:basedOn w:val="Normal"/><w:pPr><w:jc w:val="both"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Assinatura"><w:name w:val="Assinatura"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="480" w:after="0"/><w:jc w:val="center"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Cabecalho"><w:name w:val="Cabecalho"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:after="0"/><w:jc w:val="center"/></w:pPr><w:rPr><w:b/><w:caps/></w:rPr></w:style>` +
	`</w:styles>`

// ---- Typst ----

func typStr(s string) string { // string literal Typst
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func typ(d Doc) string {
	var b strings.Builder
	b.WriteString(`#set page(paper: "a4", margin: (top: 2cm + 4.2cm, bottom: 2cm, left: 3cm, right: 1.5cm), header-ascent: 0.4cm,
  header: align(center)[#image("brasao.png", height: 2.5cm) #v(-0.3em) #text(weight: "bold")[#` + typStr(cabecalho[0]) + ` \ #` + typStr(cabecalho[1]) + `]],
  footer: align(right)[#context counter(page).display()])
#set text(font: ("Times New Roman", "Liberation Serif"), size: 12pt, lang: "pt", region: "br", fill: black)
#set par(justify: true, leading: 0.65em, spacing: 0.6em)
`)
	b.WriteString("#align(center)[#text(weight: \"bold\")[#upper(" + typStr(d.Titulo) + ")]]\n#v(1em)\n")
	for _, bl := range d.Blocos {
		switch bl.Kind {
		case Secao:
			b.WriteString("#block(above: 1.2em, below: 0.6em, sticky: true)[#text(weight: \"bold\")[#" + typStr(bl.Texto) + "]]\n")
		case Paragrafo:
			b.WriteString("#par(first-line-indent: (amount: 2.5cm, all: true))[#" + typStr(bl.Texto) + "]\n")
		case Alinea:
			t := bl.Rotulo
			if bl.Texto != "" {
				t += " " + bl.Texto
			}
			b.WriteString("#par[#" + typStr(t) + "]\n")
		case Assinatura:
			b.WriteString("#v(2em)\n#align(center)[#" + typStr("_______________________________________") + " \\ #" + typStr(bl.Texto) + "]\n")
		}
	}
	return b.String()
}

func main() {
	out := flag.String("out", "out", "diretório de saída")
	titulo := flag.String("titulo", "JULGAMENTO DE PADS", "título")
	logo := flag.String("logo", "", "logo-pmro.png (opcional)")
	flag.Parse()
	d := exemplo(*titulo)
	if *logo != "" {
		b, err := os.ReadFile(*logo)
		must(err)
		cfg, err := png.DecodeConfig(bytes.NewReader(b))
		must(err)
		d.Logo, d.LogoW, d.LogoH = b, cfg.Width, cfg.Height
	} else {
		d.Logo, d.LogoW, d.LogoH = placeholder()
	}
	must(os.MkdirAll(*out, 0o700))
	must(os.WriteFile(filepath.Join(*out, "minuta.docx"), docx(d), 0o600))
	must(os.WriteFile(filepath.Join(*out, "minuta.typ"), []byte(typ(d)), 0o600))
	must(os.WriteFile(filepath.Join(*out, "brasao.png"), d.Logo, 0o600))
	// Texto canônico para conferência (DOCX × PDF).
	var t strings.Builder
	t.WriteString(strings.Join(cabecalho, "\n") + "\n" + d.Titulo + "\n")
	for _, bl := range d.Blocos {
		t.WriteString(strings.TrimSpace(bl.Rotulo+" "+bl.Texto) + "\n")
	}
	must(os.WriteFile(filepath.Join(*out, "texto.txt"), []byte(t.String()), 0o600))
	fmt.Println("ok:", *out)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

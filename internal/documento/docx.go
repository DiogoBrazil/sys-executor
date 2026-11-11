package documento

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/minuta"
)

const (
	emuPorCm   = 360000
	twipPorCm  = 567
	xmlInicio  = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"
	espacosXML = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
		`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"`
)

// escXML escapa o texto e descarta caracteres que o XML 1.0 não aceita.
func escXML(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case r == '"':
			b.WriteString("&quot;")
		case r == '\t' || r == '\n' || r == '\r':
			b.WriteByte(' ')
		case r < 0x20 || r == 0xFFFE || r == 0xFFFF:
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func runDOCX(p pedaco) string {
	rpr := ""
	if p.negrito {
		rpr += "<w:b/>"
	}
	if p.pendente {
		rpr += `<w:highlight w:val="yellow"/>`
	}
	if rpr != "" {
		rpr = "<w:rPr>" + rpr + "</w:rPr>"
	}
	return `<w:r>` + rpr + `<w:t xml:space="preserve">` + escXML(p.texto) + `</w:t></w:r>`
}

func paragrafoDOCX(estilo, conteudo string) string {
	return `<w:p><w:pPr><w:pStyle w:val="` + estilo + `"/></w:pPr>` + conteudo + `</w:p>`
}

// Times New Roman 12, entrelinha simples, justificado, recuo de primeira
// linha de 2,5 cm (V4 §32.2, parâmetros propostos).
var estilosDOCX = xmlInicio + `<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
	`<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Times New Roman" w:hAnsi="Times New Roman" w:eastAsia="Times New Roman" w:cs="Times New Roman"/><w:color w:val="000000"/><w:sz w:val="24"/><w:szCs w:val="24"/><w:lang w:val="pt-BR"/></w:rPr></w:rPrDefault>` +
	`<w:pPrDefault><w:pPr><w:spacing w:before="0" w:after="120" w:line="240" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>` +
	`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Corpo"><w:name w:val="Corpo"/><w:basedOn w:val="Normal"/><w:pPr><w:ind w:firstLine="1418"/><w:jc w:val="both"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Titulo"><w:name w:val="Titulo"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="240" w:after="360"/><w:jc w:val="center"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Identificacao"><w:name w:val="Identificacao"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:after="60"/><w:jc w:val="left"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Secao"><w:name w:val="Secao"/><w:basedOn w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="360" w:after="160"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Subtitulo"><w:name w:val="Subtitulo"/><w:basedOn w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="200" w:after="120"/><w:jc w:val="both"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Citacao"><w:name w:val="Citacao"/><w:basedOn w:val="Normal"/><w:pPr><w:ind w:left="2268"/><w:jc w:val="both"/></w:pPr><w:rPr><w:sz w:val="20"/><w:szCs w:val="20"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Data"><w:name w:val="Data"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="240"/><w:jc w:val="right"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Assinatura"><w:name w:val="Assinatura"/><w:basedOn w:val="Normal"/><w:pPr><w:keepLines/><w:spacing w:before="960" w:after="0"/><w:jc w:val="center"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Cabecalho"><w:name w:val="Cabecalho"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:after="0"/><w:jc w:val="center"/></w:pPr></w:style>` +
	`</w:styles>`

// DOCX gera o arquivo Word da minuta.
func DOCX(doc *minuta.Documento, cab Cabecalho) ([]byte, error) {
	if doc == nil || len(doc.Blocos) == 0 {
		return nil, fmt.Errorf("minuta vazia")
	}
	var corpo strings.Builder
	for _, b := range doc.Blocos {
		var runs strings.Builder
		for _, p := range pedacos(b) {
			runs.WriteString(runDOCX(p))
		}
		corpo.WriteString(paragrafoDOCX(estilo(b), runs.String()))
	}
	// Margens (V4 §32.2): superior e inferior 2 cm, esquerda 3 cm, direita
	// 1,5 cm. O cabeçalho ocupa o alto da página; o Word desce o texto.
	secao := fmt.Sprintf(`<w:sectPr><w:headerReference w:type="default" r:id="rIdH"/><w:footerReference w:type="default" r:id="rIdF"/>`+
		`<w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="%d" w:right="%d" w:bottom="%d" w:left="%d" w:header="%d" w:footer="%d" w:gutter="0"/></w:sectPr>`,
		2*twipPorCm, 3*twipPorCm/2, 2*twipPorCm, 3*twipPorCm, twipPorCm, twipPorCm)
	documento := xmlInicio + `<w:document ` + espacosXML + `><w:body>` + corpo.String() + secao + `</w:body></w:document>`

	var cabecalho strings.Builder
	cabecalho.WriteString(xmlInicio + `<w:hdr ` + espacosXML + `>`)
	tiposImagem, relsCabecalho := "", ""
	if len(cab.Logo) > 0 {
		h := int(alturaLogoCm * emuPorCm)
		w := h * cab.LogoW / cab.LogoH
		cabecalho.WriteString(paragrafoDOCX("Cabecalho", fmt.Sprintf(`<w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:docPr id="1" name="Logotipo"/>`+
			`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic><pic:nvPicPr><pic:cNvPr id="1" name="logotipo"/><pic:cNvPicPr/></pic:nvPicPr>`+
			`<pic:blipFill><a:blip r:embed="rIdImg"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`,
			w, h, w, h)))
		tiposImagem = `<Default Extension="` + cab.Ext + `" ContentType="image/` + cab.Ext + `"/>`
		relsCabecalho = `<Relationship Id="rIdImg" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/logotipo.` + cab.Ext + `"/>`
	}
	for _, l := range cab.Linhas {
		cabecalho.WriteString(paragrafoDOCX("Cabecalho", runDOCX(pedaco{texto: l, negrito: true})))
	}
	cabecalho.WriteString(paragrafoDOCX("Cabecalho", "") + `</w:hdr>`)
	rodape := xmlInicio + `<w:ftr ` + espacosXML + `><w:p><w:pPr><w:jc w:val="right"/></w:pPr>` +
		`<w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText xml:space="preserve"> PAGE </w:instrText></w:r><w:r><w:fldChar w:fldCharType="separate"/></w:r><w:r><w:t>1</w:t></w:r><w:r><w:fldChar w:fldCharType="end"/></w:r></w:p></w:ftr>`

	arquivos := []struct{ nome, dados string }{
		{"[Content_Types].xml", xmlInicio + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/>` + tiposImagem +
			`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
			`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>` +
			`<Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>` +
			`<Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>` +
			`<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/></Types>`},
		{"_rels/.rels", xmlInicio + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
			`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/></Relationships>`},
		// Metadados mínimos: sem autor e sem nomes de pessoas.
		{"docProps/core.xml", xmlInicio + `<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Minuta</dc:title></cp:coreProperties>`},
		{"word/_rels/document.xml.rels", xmlInicio + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rIdS" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
			`<Relationship Id="rIdH" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>` +
			`<Relationship Id="rIdF" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/></Relationships>`},
		{"word/_rels/header1.xml.rels", xmlInicio + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + relsCabecalho + `</Relationships>`},
		{"word/document.xml", documento},
		{"word/styles.xml", estilosDOCX},
		{"word/header1.xml", cabecalho.String()},
		{"word/footer1.xml", rodape},
	}
	if len(cab.Logo) > 0 {
		arquivos = append(arquivos, struct{ nome, dados string }{"word/media/logotipo." + cab.Ext, string(cab.Logo)})
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for _, f := range arquivos {
		// Sem data de modificação: o arquivo não registra quando foi gerado.
		w, err := z.CreateHeader(&zip.FileHeader{Name: f.nome, Method: zip.Deflate})
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(f.dados)); err != nil {
			return nil, err
		}
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

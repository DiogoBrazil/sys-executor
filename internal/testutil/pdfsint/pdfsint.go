// Package pdfsint gera PDFs sintéticos (fictícios) para testes: páginas com
// linhas de texto (Helvetica, WinAnsi), páginas em branco e páginas só com
// desenho (sem camada de texto, simulando digitalização).
package pdfsint

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

// Pagina sintética.
type Pagina struct {
	Linhas  []string // texto do topo para baixo
	Rodape  string   // linha no pé da página (ex.: "SEI nº 0090000001")
	Desenho bool     // só traços, sem texto (exige OCR)
}

// Gerar monta o PDF (A4).
func Gerar(paginas []Pagina) []byte {
	var b bytes.Buffer
	var offs []int
	obj := func(conteudo string) int {
		offs = append(offs, b.Len())
		n := len(offs)
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", n, conteudo)
		return n
	}
	b.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	catalogo := obj("<< /Type /Catalog /Pages 2 0 R >>")
	_ = catalogo
	// Reserva o objeto 2 (Pages) e 3 (fonte); preenchidos abaixo.
	posPages := len(offs)
	offs = append(offs, 0)
	fonte := obj("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	var kids []string
	for _, p := range paginas {
		conteudo := fluxo(p)
		c := obj(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(conteudo), conteudo))
		pg := obj(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>", fonte, c))
		kids = append(kids, fmt.Sprintf("%d 0 R", pg))
	}
	// Objeto 2 escrito por último, com o offset correto.
	offs[posPages] = b.Len()
	fmt.Fprintf(&b, "2 0 obj\n<< /Type /Pages /Kids [%s] /Count %d >>\nendobj\n", strings.Join(kids, " "), len(kids))
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offs)+1)
	for _, o := range offs {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offs)+1, xref)
	return b.Bytes()
}

func fluxo(p Pagina) string {
	var s strings.Builder
	if p.Desenho {
		// Traços que parecem linhas de texto manuscrito/escaneado.
		for y := 760; y > 120; y -= 22 {
			fmt.Fprintf(&s, "0.15 0.15 0.15 RG 1.2 w 70 %d m 525 %d l S\n", y, y-3)
		}
		return s.String()
	}
	y := 780
	for _, l := range p.Linhas {
		fmt.Fprintf(&s, "BT /F1 12 Tf 70 %d Td (%s) Tj ET\n", y, literal(l))
		y -= 18
	}
	if p.Rodape != "" {
		fmt.Fprintf(&s, "BT /F1 8 Tf 70 40 Td (%s) Tj ET\n", literal(p.Rodape))
	}
	return s.String()
}

// literal codifica em WinAnsi e escapa para string PDF.
func literal(s string) string {
	enc, err := charmap.Windows1252.NewEncoder().String(s)
	if err != nil {
		enc = s
	}
	r := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	return r.Replace(enc)
}

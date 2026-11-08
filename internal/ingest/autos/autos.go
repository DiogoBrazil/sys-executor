// Package autos guarda o texto dos autos por página, com origem, qualidade e
// posições na página (plano §7: evidências e mapa de cobertura).
//
// Tudo vive em memória durante a tarefa; nada é gravado em disco.
package autos

import (
	"strings"
	"unicode"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
)

// Origem do texto de uma página.
type Origem string

const (
	OrigemNativo Origem = "NATIVO" // camada de texto do PDF
	OrigemOCR    Origem = "OCR"    // reconhecimento óptico (dígitos "não confirmados", D-09.3)
)

// Situacao de leitura de uma página (mapa de cobertura, T17/ING-COB-01).
type Situacao string

const (
	Lida     Situacao = "LIDA"      // texto nativo
	OCROk    Situacao = "OCR_OK"    // OCR com confiança adequada
	OCRBaixo Situacao = "OCR_BAIXO" // OCR com confiança baixa: conteúdo pode estar errado
	Ilegivel Situacao = "ILEGIVEL"  // conteúdo visível, mas não reconhecido
	Vazia    Situacao = "VAZIA"     // página em branco
)

// Limiares de confiança do OCR (0–100). [A VALIDAR com autos reais; a F0
// mediu 84–92 em resoluções escaneadas legíveis.]
const (
	LimiarOCRBom      = 75.0
	LimiarOCRIlegivel = 40.0
)

// Fragmento liga um intervalo do texto da página (em runas) à sua caixa.
type Fragmento struct {
	Inicio, Fim int
	Caixa       pdf.Caixa
}

// Pagina é uma página dos autos.
type Pagina struct {
	Numero     int // base 1, como o usuário vê
	Texto      string
	Origem     Origem
	Confianca  float64 // só para OCR (0–100)
	Situacao   Situacao
	Largura    float64 // pontos
	Altura     float64
	Fragmentos []Fragmento
	Carimbos   []Carimbo // marcas do SEI na camada de texto (mesmo em página escaneada)
}

// Caixas devolve as caixas que cobrem o intervalo [inicio, fim) do texto.
func (p Pagina) Caixas(inicio, fim int) []pdf.Caixa {
	var out []pdf.Caixa
	for _, f := range p.Fragmentos {
		if f.Fim > inicio && f.Inicio < fim {
			out = append(out, f.Caixa)
		}
	}
	return out
}

// Autos é o conjunto das páginas extraídas.
type Autos struct {
	Paginas []Pagina
}

// Contagem resume o mapa de cobertura por situação.
func (a *Autos) Contagem() map[Situacao]int {
	c := map[Situacao]int{}
	for _, p := range a.Paginas {
		c[p.Situacao]++
	}
	return c
}

// ---- montagem do texto nativo ----

// montarTextoNativo monta o texto na ordem de leitura do pdfium e agrupa as
// caixas das letras por palavra (fragmentos para o realce).
func montarTextoNativo(p pdf.Pagina) (string, []Fragmento) {
	var sb strings.Builder
	var frs []Fragmento
	pos := 0
	var atual *Fragmento
	fechar := func() {
		if atual != nil {
			frs = append(frs, *atual)
			atual = nil
		}
	}
	for _, c := range p.Caracteres {
		r := c.Runa
		switch {
		case r == '\r':
			continue
		case r == '\n':
			fechar()
			if !strings.HasSuffix(sb.String(), "\n") && sb.Len() > 0 {
				sb.WriteByte('\n')
				pos++
			}
			continue
		case unicode.IsSpace(r) || r == 0xA0:
			fechar()
			if s := sb.String(); s != "" && !strings.HasSuffix(s, " ") && !strings.HasSuffix(s, "\n") {
				sb.WriteByte(' ')
				pos++
			}
			continue
		case r == 0xAD: // hífen suave da hifenização: invisível
			continue
		}
		sb.WriteRune(r)
		if !c.SemCaixa {
			if atual == nil {
				atual = &Fragmento{Inicio: pos, Fim: pos + 1, Caixa: c.Caixa}
			} else {
				atual.Fim = pos + 1
				atual.Caixa = uniao(atual.Caixa, c.Caixa)
			}
		}
		pos++
	}
	fechar()
	return strings.TrimRight(sb.String(), " \n"), frs
}

func uniao(a, b pdf.Caixa) pdf.Caixa {
	return pdf.Caixa{X0: min(a.X0, b.X0), Y0: min(a.Y0, b.Y0), X1: max(a.X1, b.X1), Y1: max(a.Y1, b.Y1)}
}

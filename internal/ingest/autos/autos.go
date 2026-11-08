// Package autos guarda o texto dos autos por página, com origem, qualidade e
// posições na página (plano §7: evidências e mapa de cobertura).
//
// Tudo vive em memória durante a tarefa; nada é gravado em disco.
package autos

import (
	"sort"
	"strings"

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

// montarTextoNativo ordena os trechos do pdfium em linhas (por posição
// vertical) e devolve o texto com os fragmentos posicionados.
func montarTextoNativo(p pdf.Pagina) (string, []Fragmento) {
	tr := append([]pdf.Trecho(nil), p.Trechos...)
	if len(tr) == 0 {
		return "", nil
	}
	sort.SliceStable(tr, func(i, j int) bool {
		a, b := tr[i].Caixa, tr[j].Caixa
		if mesmaLinha(a, b) {
			return a.X0 < b.X0
		}
		return a.Y0 < b.Y0
	})
	var sb strings.Builder
	var frs []Fragmento
	pos := 0
	var anterior *pdf.Caixa
	for i := range tr {
		t := strings.TrimRight(tr[i].Texto, "\r\n")
		if strings.TrimSpace(t) == "" {
			continue
		}
		c := tr[i].Caixa
		if anterior != nil {
			sep := " "
			if !mesmaLinha(*anterior, c) {
				sep = "\n"
			}
			sb.WriteString(sep)
			pos++
		}
		n := len([]rune(t))
		sb.WriteString(t)
		frs = append(frs, Fragmento{Inicio: pos, Fim: pos + n, Caixa: c})
		pos += n
		anterior = &tr[i].Caixa
	}
	return sb.String(), frs
}

// mesmaLinha: as caixas se sobrepõem em mais da metade da menor altura.
func mesmaLinha(a, b pdf.Caixa) bool {
	sob := min(a.Y1, b.Y1) - max(a.Y0, b.Y0)
	menor := min(a.Y1-a.Y0, b.Y1-b.Y0)
	return menor > 0 && sob > menor/2
}

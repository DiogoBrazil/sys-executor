package segment

import (
	"regexp"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
)

// ResultadoCompat é o resultado da checagem procedimento × autos (V4 §30.2).
type ResultadoCompat string

const (
	Compativel   ResultadoCompat = "COMPATIVEL"
	Incompativel ResultadoCompat = "INCOMPATIVEL"  // T04: alerta, nunca troca o tipo
	PecaIlegivel ResultadoCompat = "PECA_ILEGIVEL" // T06: pede versão legível
	PecaAusente  ResultadoCompat = "PECA_AUSENTE"  // peça central não encontrada
	// Divergente: alguma peça declarada parece outra peça (V4 §48.4).
	Divergente ResultadoCompat = "DECLARACAO_DIVERGENTE"
)

// Compatibilidade descreve a checagem.
type Compatibilidade struct {
	Resultado    ResultadoCompat `json:"resultado"`
	PecaCentral  string          `json:"pecaCentral"`  // ID da peça central encontrada
	TipoAparente string          `json:"tipoAparente"` // "PADS" ou "SR" quando incompatível
	// Divergencias entre a função declarada e o conteúdo (cadastro de peças).
	Divergencias []DeclaracaoDivergente `json:"divergencias,omitempty"`
}

// VerificarCompatibilidade localiza a peça central pelo conteúdo e função,
// nunca pelo nome do arquivo (V4 §30.2):
//   - PADS → Memorando Disciplinar; SR → Portaria de instauração.
//   - Peça central presente → compatível, mesmo que a peça do outro tipo apareça
//     anexada depois (T05).
//   - Ausente, mas a peça do outro tipo presente → incompatível (T04).
//   - Ausente e as primeiras páginas ilegíveis → pedir versão legível (T06).
func VerificarCompatibilidade(tipo string, pecas []Peca, a *autos.Autos) Compatibilidade {
	central, outra := MemorandoDisciplinar, PortariaSR
	outroTipo := "SR"
	if tipo == "SR" {
		central, outra, outroTipo = PortariaSR, MemorandoDisciplinar, "PADS"
	}
	for _, p := range pecas {
		if p.Funcao == central {
			return Compatibilidade{Resultado: Compativel, PecaCentral: p.ID}
		}
	}
	if pecaInicialIlegivel(pecas, a) {
		return Compatibilidade{Resultado: PecaIlegivel}
	}
	for _, p := range pecas {
		if p.Funcao == outra {
			return Compatibilidade{Resultado: Incompativel, TipoAparente: outroTipo}
		}
	}
	return Compatibilidade{Resultado: PecaAusente}
}

// pecaInicialIlegivel: alguma das 3 primeiras peças tem páginas ilegíveis ou de
// OCR fraco (onde normalmente está a peça instauradora).
func pecaInicialIlegivel(pecas []Peca, a *autos.Autos) bool {
	for i, p := range pecas {
		if i >= 3 {
			break
		}
		for n := p.PaginaInicial; n <= p.PaginaFinal && n <= len(a.Paginas); n++ {
			if s := a.Paginas[n-1].Situacao; s == autos.Ilegivel || s == autos.OCRBaixo {
				return true
			}
		}
	}
	return false
}

// MidiaReferida é uma menção a mídia/anexo audiovisual (A08–A10; V4 §17).
type MidiaReferida struct {
	Pagina int    `json:"pagina"`
	Peca   string `json:"peca"`
	Termo  string `json:"termo"`
	Inicio int    `json:"inicio"` // posição (runas) no texto da página
	Fim    int    `json:"fim"`
}

// Fronteiras Unicode explícitas: o \b do Go só entende ASCII ("áudio" falharia).
var reMidia = regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}])(v[íi]deos?|grava[çc](?:ão|ões|ao|oes)|gravad[oa]s?|filmage(?:m|ns)|c[âa]meras?|[áa]udios?|m[íi]dias?|videoconfer[êe]ncias?|dvd|pen ?drive|link externo|arquivo audiovisual)(?:$|[^\p{L}\p{N}])`)

// DetectarMidias lista as menções a mídias nas páginas, por peça.
func DetectarMidias(a *autos.Autos, pecas []Peca) []MidiaReferida {
	pecaDe := func(n int) string {
		for _, p := range pecas {
			if n >= p.PaginaInicial && n <= p.PaginaFinal {
				return p.ID
			}
		}
		return ""
	}
	var out []MidiaReferida
	for _, p := range a.Paginas {
		for _, m := range reMidia.FindAllStringSubmatchIndex(p.Texto, -1) {
			a, b := m[2], m[3] // grupo 1: o termo, sem as fronteiras
			ini := len([]rune(p.Texto[:a]))
			fim := ini + len([]rune(p.Texto[a:b]))
			out = append(out, MidiaReferida{Pagina: p.Numero, Peca: pecaDe(p.Numero), Termo: p.Texto[a:b], Inicio: ini, Fim: fim})
		}
	}
	return out
}

package analise

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

// MinTrecho é o tamanho mínimo (normalizado) de um trecho literal: trechos
// curtos casariam em qualquer lugar.
const MinTrecho = 12

var reURL = regexp.MustCompile(`https?://[^\s"'<>)]+`)

// normalizado é o texto comparável e, para cada runa dele, a runa de origem.
type normalizado struct {
	runas  []rune
	origem []int
}

// normalizar: minúsculas sem acento (runa a runa), aspas e travessões
// unificados, espaços em sequência viram um, e hifenização de fim de linha
// ("exem-\nplo") é desfeita. Assim o trecho "literal" devolvido pela IA casa
// mesmo com quebras de linha diferentes, mas nunca com palavras trocadas.
func normalizar(s string) normalizado {
	orig := []rune(s)
	alin := texto.Alinhado(s)
	var n normalizado
	espaco := false
	for i := 0; i < len(orig); i++ {
		r := alin[i]
		switch r {
		case '“', '”', '„', '«', '»', '″':
			r = '"'
		case '‘', '’', '‚', '′', '`':
			r = '\''
		case '–', '—', '−', '‐', '‑':
			r = '-'
		}
		if r == '-' && i+1 < len(orig) && quebraAdiante(orig, i+1) {
			// hífen de quebra de linha: pula o hífen e os espaços seguintes
			for i+1 < len(orig) && unicode.IsSpace(orig[i+1]) {
				i++
			}
			continue
		}
		if unicode.IsSpace(r) || r == ' ' {
			if !espaco && len(n.runas) > 0 {
				n.runas = append(n.runas, ' ')
				n.origem = append(n.origem, i)
			}
			espaco = true
			continue
		}
		espaco = false
		n.runas = append(n.runas, r)
		n.origem = append(n.origem, i)
	}
	if len(n.runas) > 0 && n.runas[len(n.runas)-1] == ' ' {
		n.runas, n.origem = n.runas[:len(n.runas)-1], n.origem[:len(n.origem)-1]
	}
	return n
}

// quebraAdiante: os próximos caracteres são espaços contendo uma quebra de linha.
func quebraAdiante(r []rune, i int) bool {
	tem := false
	for ; i < len(r) && unicode.IsSpace(r[i]); i++ {
		if r[i] == '\n' || r[i] == '\r' {
			tem = true
		}
	}
	return tem && i < len(r) && unicode.IsLetter(r[i])
}

// Localizar acha o trecho na fonte e devolve o intervalo [ini, fim) em runas
// da fonte original. Reticências ("…" ou "...") no meio do trecho separam
// partes que precisam aparecer em ordem.
func Localizar(fonte, trecho string) (ini, fim int, ok bool) {
	partes := strings.FieldsFunc(strings.ReplaceAll(trecho, "...", "…"), func(r rune) bool { return r == '…' })
	f := normalizar(fonte)
	alvoTotal := 0
	pos := 0
	ini = -1
	for _, p := range partes {
		a := normalizar(strings.TrimSpace(p)).runas
		if len(a) == 0 {
			continue
		}
		alvoTotal += len(a)
		k := indice(f.runas[pos:], a)
		if k < 0 {
			return 0, 0, false
		}
		k += pos
		if ini < 0 {
			ini = f.origem[k]
		}
		fim = f.origem[k+len(a)-1] + 1
		pos = k + len(a)
	}
	if ini < 0 || alvoTotal < MinTrecho {
		return 0, 0, false
	}
	return ini, fim, true
}

func indice(h, a []rune) int {
	for k := 0; k+len(a) <= len(h); k++ {
		igual := true
		for j := range a {
			if h[k+j] != a[j] {
				igual = false
				break
			}
		}
		if igual {
			return k
		}
	}
	return -1
}

// Padrões de instrução dentro dos autos (plano §5.6): marcados para
// visibilidade; o conteúdo continua sendo prova (V4 §13).
var padroesInjecao = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(ignore|desconsidere|esque[cç]a)\b.{0,40}\b(regras?|instru[cç](ões|oes|ão|ao)|orienta[cç](ões|oes))`),
	regexp.MustCompile(`(?i)\b(voc[eê]|a ia|o modelo|o assistente)\s+(deve|precisa|tem que)\b`),
	regexp.MustCompile(`(?i)\b(condene|absolva|puna|arquive)\b`),
	regexp.MustCompile(`(?i)\binclua\b.{0,30}\bimputa[cç]`),
	regexp.MustCompile(`(?i)\b(acesse|abra|visite|baixe)\b.{0,30}(https?://|www\.)`),
	regexp.MustCompile(`(?i)\b(revele|mostre|repita|imprima)\b.{0,30}\b(prompt|instru[cç](ões|oes)|regras)`),
	regexp.MustCompile(`(?i)\b(mude|troque|altere)\b.{0,30}\b(para|o tipo).{0,20}\b(sindic[aâ]ncia|pads|sr)\b`),
	regexp.MustCompile(`(?i)\b(system prompt|prompt do sistema|modo desenvolvedor|jailbreak)\b`),
}

// MarcaInjecao é um trecho com padrão de instrução.
type MarcaInjecao struct {
	Inicio, Fim int // runas
	Trecho      string
}

// DetectarInjecao marca padrões de instrução no texto (heurística).
func DetectarInjecao(s string) []MarcaInjecao {
	var out []MarcaInjecao
	for _, re := range padroesInjecao {
		for _, m := range re.FindAllStringIndex(s, -1) {
			ini := len([]rune(s[:m[0]]))
			fim := ini + len([]rune(s[m[0]:m[1]]))
			out = append(out, MarcaInjecao{Inicio: ini, Fim: fim, Trecho: s[m[0]:m[1]]})
		}
	}
	return out
}

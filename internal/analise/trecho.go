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
		if r == '-' && i > 0 && i+1 < len(orig) && unicode.IsLetter(orig[i-1]) && unicode.IsLetter(orig[i+1]) {
			continue // hífen entre letras é ignorado dos dois lados ("pre-viamente", "corregedor-geral")
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

// JanelaOmissao: distância máxima (runas normalizadas) entre partes de um
// trecho com omissão de frase.
const JanelaOmissao = 400

// Localizar acha o trecho na fonte e devolve o intervalo [ini, fim) em runas
// da fonte original. Reticências ("…" ou "...") no meio do trecho separam
// partes que precisam aparecer em ordem.
func Localizar(fonte, trecho string) (ini, fim int, ok bool) {
	ini, fim, _, ok = localizarPartes(fonte, trecho)
	return
}

// LocalizarComOmissao também aceita frases inteiras omitidas entre frases
// citadas (o modelo costuma pular uma linha intermediária): cada frase do
// trecho precisa existir intacta, na ordem, a no máximo JanelaOmissao da
// anterior. Palavras dentro de uma frase nunca podem faltar. Devolve o texto
// do documento com "[…]" onde houve omissão.
func LocalizarComOmissao(fonte, trecho string) (ini, fim int, citado string, ok bool) {
	if i, f, partes, ok := localizarPartes(fonte, trecho); ok {
		return i, f, juntarPartes(fonte, partes), true
	}
	frases := reFrase.Split(strings.TrimSpace(trecho), -1)
	if len(frases) < 2 {
		return 0, 0, "", false
	}
	return localizarFrases(fonte, frases)
}

var reFrase = regexp.MustCompile(`(?:[.;:!?]["'”’)]?\s+|\s*\n\s*)`)

func juntarPartes(fonte string, partes [][2]int) string {
	r := []rune(fonte)
	var b strings.Builder
	for k, p := range partes {
		if k > 0 {
			if partes[k-1][1] < p[0] && strings.TrimSpace(string(r[partes[k-1][1]:p[0]])) != "" {
				b.WriteString(" […] ")
			} else {
				b.WriteString(string(r[partes[k-1][1]:p[0]]))
			}
		}
		b.WriteString(string(r[p[0]:p[1]]))
	}
	return b.String()
}

func localizarFrases(fonte string, frases []string) (int, int, string, bool) {
	f := normalizar(fonte)
	orig := []rune(fonte)
	pos, total := 0, 0
	var partes [][2]int
	for _, fr := range frases {
		a := normalizar(strings.TrimSpace(fr)).runas
		if len(a) == 0 {
			continue
		}
		if len(a) < MinTrecho {
			return 0, 0, "", false
		}
		lim := len(f.runas)
		if len(partes) > 0 {
			lim = min(len(f.runas), pos+JanelaOmissao+len(a))
		}
		k := indice(f.runas[pos:lim], a)
		if k < 0 {
			return 0, 0, "", false
		}
		k += pos
		ini, fim := f.origem[k], f.origem[k+len(a)-1]+1
		if !fronteiraAntes(orig, ini) || !fronteiraDepois(orig, fim) {
			return 0, 0, "", false // frase truncada: omissão dentro da frase
		}
		partes = append(partes, [2]int{ini, fim})
		pos = k + len(a)
		total += len(a)
	}
	if len(partes) == 0 || total < MinTrecho {
		return 0, 0, "", false
	}
	return partes[0][0], partes[len(partes)-1][1], juntarPartes(fonte, partes), true
}

func ehFimDeFrase(r rune) bool { return strings.ContainsRune(".;:!?\n\r", r) }

// fronteiraAntes: a parte começa no início do texto ou depois de fim de frase.
func fronteiraAntes(r []rune, i int) bool {
	for i--; i >= 0; i-- {
		if r[i] == '\n' || r[i] == '\r' {
			return true
		}
		if !unicode.IsSpace(r[i]) && !strings.ContainsRune(`"'“”‘’(`, r[i]) {
			return ehFimDeFrase(r[i])
		}
	}
	return true
}

// fronteiraDepois: a parte termina no fim do texto ou antes de fim de frase.
func fronteiraDepois(r []rune, i int) bool {
	if i > 0 && ehFimDeFrase(r[i-1]) {
		return true
	}
	for ; i < len(r); i++ {
		if r[i] == '\n' || r[i] == '\r' {
			return true
		}
		if !unicode.IsSpace(r[i]) && !strings.ContainsRune(`"'“”‘’)`, r[i]) {
			return ehFimDeFrase(r[i])
		}
	}
	return true
}

func localizarPartes(fonte, trecho string) (ini, fim int, partesOrig [][2]int, ok bool) {
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
			return 0, 0, nil, false
		}
		k += pos
		if ini < 0 {
			ini = f.origem[k]
		}
		fim = f.origem[k+len(a)-1] + 1
		partesOrig = append(partesOrig, [2]int{f.origem[k], fim})
		pos = k + len(a)
	}
	if ini < 0 || alvoTotal < MinTrecho {
		return 0, 0, nil, false
	}
	return ini, fim, partesOrig, true
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

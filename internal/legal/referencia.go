package legal

import (
	"regexp"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

// Referencia aponta um dispositivo de um documento da biblioteca.
type Referencia struct {
	Documento string // id do documento
	Artigo    string // "39", "83-A"
	Paragrafo string // "1", "unico"
	Inciso    string // "VII"
	Alinea    string // "a"
}

var (
	reRefArtigo  = regexp.MustCompile(`\bart(?:igo)?s?\s*\.?\s*(\d{1,3})\s*[º°ªo]?(?:\s*-\s*([a-z])\b)?`)
	reRefParUn   = regexp.MustCompile(`\bparagrafo\s+unico\b`)
	reRefPar     = regexp.MustCompile(`(?:§|\bparagrafo)\s*(\d{1,2})`)
	reRefInciso  = regexp.MustCompile(`\binciso\s+([ivxlcdm]{1,8})\b`)
	reRefAlinea  = regexp.MustCompile(`\balinea\s+["“”']?([a-z])\b`)
	reRefRomano  = regexp.MustCompile(`^[ivxlcdm]{1,8}$`)
	reRefLetra   = regexp.MustCompile(`^["“”']?([a-z])["“”']?\)?$`)
	reRefLimpeza = regexp.MustCompile(`\s+`)
)

// InterpretarReferencia lê uma citação como "art. 39, parágrafo único, do
// RDPM", "art. 17, inciso V, do RDPM" ou "art. 39, VII, a, RDPM". Devolve
// falso quando o texto não traz artigo. O documento fica vazio quando
// nenhuma sigla conhecida aparece.
func (b *Biblioteca) InterpretarReferencia(s string) (Referencia, bool) {
	n := reRefLimpeza.ReplaceAllString(texto.SemAcentos(s), " ")
	m := reRefArtigo.FindStringSubmatchIndex(n)
	if m == nil {
		return Referencia{}, false
	}
	r := Referencia{Artigo: n[m[2]:m[3]]}
	if m[4] >= 0 {
		r.Artigo += "-" + strings.ToUpper(n[m[4]:m[5]])
	}
	r.Documento = b.documentoCitado(n)
	if reRefParUn.MatchString(n) {
		r.Paragrafo = "unico"
	} else if p := reRefPar.FindStringSubmatch(n); p != nil {
		r.Paragrafo = strings.TrimLeft(p[1], "0")
	}
	if i := reRefInciso.FindStringSubmatch(n); i != nil {
		r.Inciso = strings.ToUpper(i[1])
	}
	if a := reRefAlinea.FindStringSubmatch(n); a != nil {
		r.Alinea = a[1]
	}
	// Forma curta, depois do artigo: "art. 39, VII, a".
	for _, seg := range strings.Split(n[m[1]:], ",") {
		seg = strings.TrimSpace(seg)
		switch {
		case r.Inciso == "" && reRefRomano.MatchString(seg) && romanoValido(strings.ToUpper(seg)):
			r.Inciso = strings.ToUpper(seg)
		case r.Alinea == "" && r.Inciso != "" && reRefLetra.MatchString(seg):
			r.Alinea = reRefLetra.FindStringSubmatch(seg)[1]
		}
	}
	return r, true
}

// documentoCitado acha a sigla mais longa presente no texto normalizado.
func (b *Biblioteca) documentoCitado(n string) string {
	melhor, tam := "", 0
	for sigla, id := range b.siglas {
		if len(sigla) > tam && contemPalavra(n, sigla) {
			melhor, tam = id, len(sigla)
		}
	}
	return melhor
}

// contemPalavra: a sigla aparece inteira (não dentro de outra palavra).
func contemPalavra(n, sigla string) bool {
	for i := 0; ; {
		k := strings.Index(n[i:], sigla)
		if k < 0 {
			return false
		}
		ini, fim := i+k, i+k+len(sigla)
		antes := ini == 0 || !letraOuDigito(n[ini-1])
		depois := fim == len(n) || !letraOuDigito(n[fim])
		if antes && depois {
			return true
		}
		i = ini + 1
	}
}

func letraOuDigito(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}

// Texto escreve a referência por extenso: "art. 39, inciso VII, alínea a".
func (r Referencia) Texto() string {
	s := "art. " + r.Artigo
	if n := numeroArtigo(r.Artigo); n > 0 && n < 10 && !strings.Contains(r.Artigo, "-") {
		s += "º"
	}
	switch {
	case r.Paragrafo == "unico":
		s += ", parágrafo único"
	case r.Paragrafo != "":
		s += ", § " + r.Paragrafo + "º"
	}
	if r.Inciso != "" {
		s += ", inciso " + r.Inciso
	}
	if r.Alinea != "" {
		s += ", alínea " + r.Alinea
	}
	return s
}

func numeroArtigo(a string) int {
	n := 0
	for _, c := range a {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

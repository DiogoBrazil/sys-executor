// Package texto reúne normalizações de texto usadas na ingestão e na busca.
package texto

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// SemAcentos devolve s em minúsculas e sem diacríticos ("Inquirição" → "inquiricao").
func SemAcentos(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	r, _, _ := transform.String(t, s)
	return strings.ToLower(r)
}

// Alinhado normaliza runa a runa, preservando a contagem: a posição i do
// resultado corresponde à runa i do original (para localizar trechos).
func Alinhado(s string) []rune {
	orig := []rune(s)
	out := make([]rune, len(orig))
	for i, r := range orig {
		out[i] = unicode.ToLower(r)
		if r > unicode.MaxASCII {
			if n := []rune(SemAcentos(string(r))); len(n) > 0 {
				out[i] = n[0]
			}
		}
	}
	return out
}

// Termos quebra s em palavras normalizadas (letras e dígitos).
func Termos(s string) []string {
	return strings.FieldsFunc(SemAcentos(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

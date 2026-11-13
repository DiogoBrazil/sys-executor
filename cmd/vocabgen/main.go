// vocabgen gera o vocabulário de palavras comuns usado pela barreira de
// privacidade (internal/pseudonimo/dados/vocabulario.txt). As palavras vêm do
// acervo da biblioteca jurídica: só as que aparecem escritas em minúsculas
// (nunca nome próprio) ao menos 3 vezes. Sobrenomes que também são palavras
// (Costa, Rocha, Pinto…) ficam de fora, para nunca deixar um nome passar.
//
// Uso (na raiz do repositório):
//
//	go run ./cmd/vocabgen            # grava o vocabulário
//	go run ./cmd/vocabgen -conferir  # falha se o arquivo estiver desatualizado
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/DiogoBrazil/sys-executor/internal/legal"
	"github.com/DiogoBrazil/sys-executor/internal/pseudonimo"
)

var (
	saida    = flag.String("saida", "internal/pseudonimo/dados/vocabulario.txt", "arquivo do vocabulário")
	conferir = flag.Bool("conferir", false, "não grava; falha se o arquivo difere do gerado")
)

const minimo = 3 // ocorrências em minúsculas

func main() {
	flag.Parse()
	a, err := legal.AcervoInicial()
	if err != nil {
		fmt.Fprintln(os.Stderr, "vocabgen:", err)
		os.Exit(1)
	}
	cont := map[string]int{}
	for _, d := range a.Documentos {
		for _, p := range d.Partes {
			for _, u := range p.Unidades {
				for _, w := range strings.FieldsFunc(u.Texto, func(r rune) bool { return !unicode.IsLetter(r) }) {
					if len([]rune(w)) < 3 || strings.ToLower(w) != w {
						continue // nome próprio, sigla ou começo de frase
					}
					cont[pseudonimo.ChaveVocabulario(w)]++
				}
			}
		}
	}
	var ws []string
	for w, n := range cont {
		if n >= minimo && !pseudonimo.SobrenomeComum(w) {
			ws = append(ws, w)
		}
	}
	sort.Strings(ws)
	var b bytes.Buffer
	b.WriteString("# Vocabulário de palavras comuns (gerado por go run ./cmd/vocabgen; não editar à mão).\n")
	for _, w := range ws {
		b.WriteString(w + "\n")
	}
	if *conferir {
		atual, _ := os.ReadFile(*saida)
		if !bytes.Equal(atual, b.Bytes()) {
			fmt.Fprintln(os.Stderr, "vocabgen:", *saida, "está desatualizado: rode go run ./cmd/vocabgen")
			os.Exit(1)
		}
		fmt.Println("vocabulário conferido: em dia")
		return
	}
	if err := os.WriteFile(*saida, b.Bytes(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "vocabgen:", err)
		os.Exit(1)
	}
	fmt.Printf("gravado %s (%d palavras)\n", *saida, len(ws))
}

package autos

import (
	"regexp"
	"strconv"
	"strings"
)

// Carimbo é a marca que o SEI/RO acrescenta em cada página do PDF consolidado
// (observado em PDF real, 08/11/2026):
//
//	<nome do documento na árvore> (<nº SEI do documento>) SEI <nº do processo> / pg. <página>
//
// Os parênteses às vezes faltam ("PADS CAPA 0000000000 SEI …"). Documentos
// trazidos de outro processo mantêm os carimbos antigos, por isso uma página
// pode ter vários.
type Carimbo struct {
	Nome     string `json:"nome"`
	SEI      string `json:"sei"`
	Processo string `json:"processo"`
	PaginaPG int    `json:"paginaPG"`
}

var reCarimbo = regexp.MustCompile(`\(?\s*\b([0-9]{7,10})\s*\)?\s+SEI\s+([0-9]{4}\.[0-9]{6}/[0-9]{4}-[0-9]{2})\s*/\s*pg\.?\s*([0-9]{1,5})`)

// LerCarimbos extrai os carimbos do texto de uma página, com o nome do
// documento (o trecho da mesma linha antes do número SEI).
func LerCarimbos(texto string) []Carimbo {
	var out []Carimbo
	for _, linha := range strings.Split(texto, "\n") {
		ultimo := 0
		for _, m := range reCarimbo.FindAllStringSubmatchIndex(linha, -1) {
			nome := strings.TrimSpace(linha[ultimo:m[0]])
			nome = strings.TrimRight(nome, " -–(")
			pg, _ := strconv.Atoi(linha[m[6]:m[7]])
			out = append(out, Carimbo{Nome: nome, SEI: linha[m[2]:m[3]], Processo: linha[m[4]:m[5]], PaginaPG: pg})
			ultimo = m[1]
		}
	}
	return out
}

// SemCarimbos remove os carimbos do texto (para decidir se a página tem
// conteúdo próprio ou se é uma imagem escaneada que só tem o carimbo).
func SemCarimbos(texto string) string {
	var ls []string
	for _, l := range strings.Split(texto, "\n") {
		if reCarimbo.MatchString(l) {
			continue
		}
		ls = append(ls, l)
	}
	return strings.Join(ls, "\n")
}

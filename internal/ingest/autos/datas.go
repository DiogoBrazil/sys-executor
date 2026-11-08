package autos

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

// DataMencionada é uma data encontrada no texto dos autos (A03; D-09.3/D-09.4).
// Todas as menções são preservadas, mesmo divergentes entre peças: decidir
// qual vale é tarefa da análise, com pergunta ao usuário quando material.
type DataMencionada struct {
	Pagina     int    `json:"pagina"`
	Inicio     int    `json:"inicio"` // runas no texto da página
	Fim        int    `json:"fim"`
	Dia        int    `json:"dia"`
	Mes        int    `json:"mes"`
	Ano        int    `json:"ano"`
	Origem     Origem `json:"origem"`
	Confirmada bool   `json:"confirmada"` // false para OCR: exige verificação (D-09.3)
}

var meses = map[string]int{
	"janeiro": 1, "fevereiro": 2, "marco": 3, "abril": 4, "maio": 5, "junho": 6, "julho": 7,
	"agosto": 8, "setembro": 9, "outubro": 10, "novembro": 11, "dezembro": 12,
}

var numerais = map[string]int{
	"um": 1, "uma": 1, "primeiro": 1, "dois": 2, "duas": 2, "tres": 3, "quatro": 4, "cinco": 5, "seis": 6,
	"sete": 7, "oito": 8, "nove": 9, "dez": 10, "onze": 11, "doze": 12, "treze": 13, "quatorze": 14,
	"catorze": 14, "quinze": 15, "dezesseis": 16, "dezessete": 17, "dezoito": 18, "dezenove": 19,
	"vinte": 20, "trinta": 30, "quarenta": 40, "cinquenta": 50, "sessenta": 60, "setenta": 70,
	"oitenta": 80, "noventa": 90,
}

const palavraNum = `(?:um|uma|primeiro|dois|duas|tres|quatro|cinco|seis|sete|oito|nove|dez|onze|doze|treze|quatorze|catorze|quinze|dezesseis|dezessete|dezoito|dezenove|vinte|trinta|quarenta|cinquenta|sessenta|setenta|oitenta|noventa)`

var (
	// O texto é normalizado (minúsculas, sem acentos) com as posições preservadas.
	reNumerica = regexp.MustCompile(`(?:^|[^0-9])([0-3]?[0-9])[/.-]([01]?[0-9])[/.-]((?:19|20)[0-9]{2})(?:$|[^0-9])`)
	reMesNome  = regexp.MustCompile(`(?:^|[^0-9a-z])([0-3]?[0-9])(?:º|o)?\s+de\s+(janeiro|fevereiro|marco|abril|maio|junho|julho|agosto|setembro|outubro|novembro|dezembro)\s+de\s+((?:19|20)[0-9]{2})(?:$|[^0-9])`)
	reExtenso  = regexp.MustCompile(`(` + palavraNum + `(?:\s+e\s+` + palavraNum + `)?)\s+(?:dias?\s+)?(?:do\s+mes\s+)?de\s+(janeiro|fevereiro|marco|abril|maio|junho|julho|agosto|setembro|outubro|novembro|dezembro)\s+(?:do\s+ano\s+)?de\s+dois\s+mil(?:\s+e\s+(` + palavraNum + `(?:\s+e\s+` + palavraNum + `)?))?`)
)

// valorExtenso soma "vinte e seis" → 26.
func valorExtenso(s string) int {
	total := 0
	for _, p := range strings.Fields(s) {
		if p == "e" {
			continue
		}
		v, ok := numerais[p]
		if !ok {
			return -1
		}
		total += v
	}
	return total
}

func dataValida(d, m, a int) bool {
	if m < 1 || m > 12 || d < 1 {
		return false
	}
	t := time.Date(a, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	return t.Day() == d && int(t.Month()) == m
}

// Datas extrai as menções de data de todas as páginas.
func (a *Autos) Datas() []DataMencionada {
	var out []DataMencionada
	for _, p := range a.Paginas {
		out = append(out, datasDaPagina(p)...)
	}
	return out
}

func datasDaPagina(p Pagina) []DataMencionada {
	norm := string(texto.Alinhado(p.Texto))
	// Alinhado preserva a contagem de runas; os índices de byte do regexp são
	// convertidos para runas sobre o texto normalizado.
	runa := func(b int) int { return len([]rune(norm[:b])) }
	var out []DataMencionada
	add := func(ini, fim, d, m, ano int) {
		if !dataValida(d, m, ano) {
			return
		}
		out = append(out, DataMencionada{
			Pagina: p.Numero, Inicio: runa(ini), Fim: runa(fim), Dia: d, Mes: m, Ano: ano,
			Origem: p.Origem, Confirmada: p.Origem != OrigemOCR,
		})
	}
	for _, m := range reNumerica.FindAllStringSubmatchIndex(norm, -1) {
		d, _ := strconv.Atoi(norm[m[2]:m[3]])
		mes, _ := strconv.Atoi(norm[m[4]:m[5]])
		ano, _ := strconv.Atoi(norm[m[6]:m[7]])
		add(m[2], m[7], d, mes, ano)
	}
	for _, m := range reMesNome.FindAllStringSubmatchIndex(norm, -1) {
		d, _ := strconv.Atoi(norm[m[2]:m[3]])
		ano, _ := strconv.Atoi(norm[m[6]:m[7]])
		add(m[2], m[7], d, meses[norm[m[4]:m[5]]], ano)
	}
	for _, m := range reExtenso.FindAllStringSubmatchIndex(norm, -1) {
		d := valorExtenso(norm[m[2]:m[3]])
		ano := 2000
		if m[6] >= 0 {
			v := valorExtenso(norm[m[6]:m[7]])
			if v < 0 {
				continue
			}
			ano += v
		}
		add(m[0], m[1], d, meses[norm[m[4]:m[5]]], ano)
	}
	return out
}

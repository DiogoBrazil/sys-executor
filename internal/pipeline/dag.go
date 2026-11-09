package pipeline

import (
	"sort"
	"strings"
)

// DAG de invalidação (F4-05): cada nó é uma decisão do usuário (pergunta) ou
// um resultado calculado (extração, análise). Rever uma decisão invalida
// todos os descendentes, e só eles são refeitos; nada é perguntado de novo
// sem necessidade.
type DAG struct {
	deps map[string][]string // nó → nós de que depende
}

// NovoDAG cria um DAG vazio.
func NovoDAG() *DAG { return &DAG{deps: map[string][]string{}} }

// Depende registra que nó depende de cada um de base. Prefixos terminados em
// ":*" valem para todos os nós com aquele prefixo (ex.: "resultado:*").
func (g *DAG) Depende(no string, base ...string) {
	for _, b := range base {
		if !contem(g.deps[no], b) {
			g.deps[no] = append(g.deps[no], b)
		}
	}
}

func contem(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

func casa(padrao, no string) bool {
	if strings.HasSuffix(padrao, ":*") {
		return strings.HasPrefix(no, strings.TrimSuffix(padrao, "*"))
	}
	return padrao == no
}

// Descendentes devolve, em ordem estável, os nós que dependem (direta ou
// indiretamente) de no.
func (g *DAG) Descendentes(no string) []string {
	vistos := map[string]bool{}
	fila := []string{no}
	for len(fila) > 0 {
		atual := fila[0]
		fila = fila[1:]
		for n, ds := range g.deps {
			if vistos[n] || n == no {
				continue
			}
			for _, d := range ds {
				if casa(d, atual) {
					vistos[n] = true
					fila = append(fila, n)
					break
				}
			}
		}
	}
	out := make([]string, 0, len(vistos))
	for n := range vistos {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

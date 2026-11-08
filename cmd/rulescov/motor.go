package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const caminhoPlanoTestes = "docs/planning/PLANO_TESTES_ACEITACAO.md"

// idsSecao1 lê os IDs da 1ª coluna das tabelas da seção "## 1." do plano de
// testes (motor determinístico).
func idsSecao1(plano string) []string {
	ini := strings.Index(plano, "\n## 1.")
	fim := strings.Index(plano, "\n## 2.")
	if ini < 0 || fim < ini {
		return nil
	}
	ids := map[string]bool{}
	for _, l := range strings.Split(plano[ini:fim], "\n") {
		if !strings.HasPrefix(l, "| ") || strings.HasPrefix(l, "| ID ") {
			continue
		}
		cel := strings.TrimSpace(strings.Split(strings.TrimPrefix(l, "|"), "|")[0])
		for _, id := range expandir(cel) {
			ids[id] = true
		}
	}
	return ordenarIDs(ids)
}

var (
	reIntervalo = regexp.MustCompile(`^(.*?)(\d+)…(\d+)$`)
	reSufixos   = regexp.MustCompile(`^(.*?)(\d+)((?:/\d+)+)$`)
)

// expandir: "A18 / A19" → A18, A19; "EQV-01…08" → EQV-01..EQV-08;
// "DOS-CMP-01/02/03" → DOS-CMP-01, DOS-CMP-02, DOS-CMP-03.
func expandir(cel string) []string {
	var out []string
	for _, parte := range regexp.MustCompile(`\s*/\s+|\s+/\s*|,\s*`).Split(cel, -1) {
		parte = strings.TrimSpace(parte)
		if parte == "" {
			continue
		}
		if m := reIntervalo.FindStringSubmatch(parte); m != nil {
			a, _ := strconv.Atoi(m[2])
			b, _ := strconv.Atoi(m[3])
			for i := a; i <= b; i++ {
				out = append(out, fmt.Sprintf("%s%0*d", m[1], len(m[2]), i))
			}
			continue
		}
		if m := reSufixos.FindStringSubmatch(parte); m != nil {
			out = append(out, m[1]+m[2])
			for _, s := range strings.Split(strings.TrimPrefix(m[3], "/"), "/") {
				out = append(out, m[1]+s)
			}
			continue
		}
		out = append(out, parte)
	}
	return out
}

var reCobre = regexp.MustCompile(`//\s*Cobre:\s*(.+)$`)

func ordenarIDs(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// verificarMotor: GOV-COV-01 para as regras determinísticas (saída da Fase 3).
func verificarMotor(raiz string) (total int, faltam []string, err error) {
	b, err := os.ReadFile(filepath.Join(raiz, caminhoPlanoTestes))
	if err != nil {
		return 0, nil, err
	}
	exigidos := idsSecao1(string(b))
	cob, err := anotacoes(filepath.Join(raiz, "internal"), reCobre)
	if err != nil {
		return 0, nil, err
	}
	for _, id := range exigidos {
		if !cob[id] {
			faltam = append(faltam, id)
		}
	}
	return len(exigidos), faltam, nil
}

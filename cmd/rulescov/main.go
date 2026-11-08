// Comando rulescov: governança da V4 (plano §3; GOV-LOCK-01, GOV-COV-01).
//
// Esqueleto da Fase 1:
//  1. Confere o SHA-256 da V4 contra policies/V4.lock. Divergência = falha:
//     a V4 mudou sem revisão registrada em policies/CHANGELOG.md.
//  2. Extrai os IDs de teste da V4 (A01, SR01, T01, R1, E1, EX1, PADS-E01, D1,
//     P1, V1, S1…) e confere se cada um aparece na matriz de requisitos.
//     Na Fase 3 a conferência passa a ser contra o catálogo de regras
//     (internal/rules/catalog) e contra os testes.
//
// Uso: go run ./cmd/rulescov [-raiz .] [-estrito] [-atualizar-lock]
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	caminhoV4     = "docs/rules/sys-executor_especificacao_consolidada_V4.md"
	caminhoLock   = "policies/V4.lock"
	caminhoMatriz = "docs/planning/MATRIZ_REQUISITOS_V4.md"
)

// Padrões de IDs de teste da V4 (plano §3). Os de uma letra exigem fronteira
// de palavra para não colidir com texto comum.
var padroes = []*regexp.Regexp{
	regexp.MustCompile(`\bPADS-E\d{2}\b`),
	regexp.MustCompile(`\bSR\d{2}\b`),
	regexp.MustCompile(`\bA\d{2}\b`),
	regexp.MustCompile(`\bT\d{2}\b`),
	regexp.MustCompile(`\bEX\d\b`),
	regexp.MustCompile(`\bR\d\b`),
	regexp.MustCompile(`\bE\d\b`),
	regexp.MustCompile(`\bD\d\b`),
	regexp.MustCompile(`\bP1\b`),
	regexp.MustCompile(`\bV\d\b`),
	regexp.MustCompile(`\bS\d\b`),
}

func main() {
	raiz := flag.String("raiz", ".", "raiz do repositório")
	estrito := flag.Bool("estrito", false, "falhar se algum ID da V4 não estiver mapeado")
	atualizar := flag.Bool("atualizar-lock", false, "regravar policies/V4.lock (exige entrada nova em policies/CHANGELOG.md)")
	flag.Parse()

	v4, err := os.ReadFile(filepath.Join(*raiz, caminhoV4))
	if err != nil {
		falhar("não foi possível ler a V4: %v", err)
	}
	soma := sha256.Sum256(v4)
	hash := hex.EncodeToString(soma[:])

	if *atualizar {
		if err := os.WriteFile(filepath.Join(*raiz, caminhoLock), []byte(hash+"  "+caminhoV4+"\n"), 0o644); err != nil {
			falhar("gravar lock: %v", err)
		}
		fmt.Println("V4.lock atualizado:", hash)
		fmt.Println("Registre a revisão humana em policies/CHANGELOG.md.")
		return
	}

	lock, err := os.ReadFile(filepath.Join(*raiz, caminhoLock))
	if err != nil {
		falhar("policies/V4.lock ausente: %v", err)
	}
	campos := strings.Fields(string(lock))
	if len(campos) == 0 || campos[0] != hash {
		falhar("GOV-LOCK-01: a V4 mudou (SHA-256 %s) e o lock não foi revisado. Atualize o catálogo, registre a revisão em policies/CHANGELOG.md e rode com -atualizar-lock.", hash)
	}
	fmt.Println("GOV-LOCK-01 ok: V4", hash[:16]+"…")

	matriz, err := os.ReadFile(filepath.Join(*raiz, caminhoMatriz))
	if err != nil {
		falhar("não foi possível ler a matriz: %v", err)
	}
	ids := extrairIDs(string(v4))
	presentes := extrairIDs(string(matriz))
	var faltam []string
	for id := range ids {
		if !presentes[id] {
			faltam = append(faltam, id)
		}
	}
	lista := ordenar(ids)
	fmt.Printf("GOV-COV-01 (esqueleto): %d IDs de teste na V4; %d mapeados na matriz; %d sem mapeamento.\n",
		len(lista), len(lista)-len(faltam), len(faltam))
	if len(faltam) > 0 {
		sort.Strings(faltam)
		fmt.Println("Sem mapeamento:", strings.Join(faltam, " "))
		if *estrito {
			os.Exit(1)
		}
	}
}

// naoSaoIDs são nomes de versões da especificação que casam com os padrões.
var naoSaoIDs = map[string]bool{"V3": true, "V4": true}

func extrairIDs(texto string) map[string]bool {
	ids := map[string]bool{}
	for _, re := range padroes {
		for _, m := range re.FindAllString(texto, -1) {
			if naoSaoIDs[m] {
				continue
			}
			ids[m] = true
		}
	}
	return ids
}

func ordenar(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func falhar(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "rulescov: "+f+"\n", a...)
	os.Exit(1)
}

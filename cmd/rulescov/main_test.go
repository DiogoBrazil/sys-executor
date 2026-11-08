package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtrairIDs(t *testing.T) {
	ids := extrairIDs("Casos A01, SR12, T07, R9, E2, EX1, PADS-E05, D1, P1, V2 e S3; conforme V3 e V4; R$ 10; AB12; T123.")
	for _, id := range []string{"A01", "SR12", "T07", "R9", "E2", "EX1", "PADS-E05", "D1", "P1", "V2", "S3"} {
		if !ids[id] {
			t.Errorf("faltou %s", id)
		}
	}
	for _, id := range []string{"V3", "V4", "AB12", "T123", "R"} {
		if ids[id] {
			t.Errorf("falso positivo %s", id)
		}
	}
}

func TestExpandir(t *testing.T) {
	casos := map[string]string{
		"A18 / A19 / A20":                   "A18 A19 A20",
		"EQV-01…03":                         "EQV-01 EQV-02 EQV-03",
		"DOS-CMP-01/02/03":                  "DOS-CMP-01 DOS-CMP-02 DOS-CMP-03",
		"DOS-37-01, DOS-38-01, DOS-38II-01": "DOS-37-01 DOS-38-01 DOS-38II-01",
		"CMP-AUS-01 / SEG13-05":             "CMP-AUS-01 SEG13-05",
		"R9":                                "R9",
	}
	for e, s := range casos {
		if got := strings.Join(expandir(e), " "); got != s {
			t.Errorf("%q: %q, esperado %q", e, got, s)
		}
	}
}

func escrever(t *testing.T, raiz, rel, conteudo string) {
	t.Helper()
	p := filepath.Join(raiz, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(conteudo), 0o644); err != nil {
		t.Fatal(err)
	}
}

// O repositório real passa; um repositório sintético com cada tipo de
// defeito reprova em todos eles.
func TestVerificarCatalogo(t *testing.T) {
	v4, _ := os.ReadFile("../../" + caminhoV4)
	matriz, _ := os.ReadFile("../../" + caminhoMatriz)
	rel, err := verificarCatalogo("../..", extrairIDs(string(v4)), string(matriz))
	if err != nil || rel.falhas() != 0 {
		t.Fatalf("repositório: %v %+v", err, rel)
	}

	raiz := t.TempDir()
	escrever(t, raiz, "internal/rules/catalog/a.yaml", `
- id: PADS-REI-001
  origem: "§1"
  fluxo: PADS
  rotulo: [APROVADO]
  tipo: [DETERMINISTICA]
  enunciado: "garantida por símbolo inexistente"
  modulo: "engine.X"
  enforced_by: [engine.Inexistente]
  testes: ["R1"]
  versao: 1
- id: PADS-REI-002
  origem: "§1"
  fluxo: PADS
  rotulo: [APROVADO]
  tipo: [DETERMINISTICA]
  enunciado: "do motor, sem enforced_by"
  modulo: "engine.Y"
  testes: ["R2"]
  versao: 1
`)
	escrever(t, raiz, "internal/rules/engine/e.go", "package engine\n\nfunc Existe() {}\n")
	escrever(t, raiz, "internal/rules/engine/e_test.go", "package engine\n\n// Regra: PADS-REI-777\n")
	rel, err = verificarCatalogo(raiz, map[string]bool{"R1": true, "A01": true}, "| PADS-REI-001 |\n| PADS-REI-003 |\n")
	if err != nil {
		t.Fatal(err)
	}
	checar := map[string][]string{
		"V4 sem regra":         rel.V4SemRegra,
		"só na matriz":         rel.SoNaMatriz,
		"só no catálogo":       rel.SoNoCatalogo,
		"símbolo inexistente":  rel.SimbolosFaltam,
		"sem teste":            rel.SemTeste,
		"motor sem enforced":   rel.MotorSemEnforce,
		"Regra: fora do catál": rel.RegraInexistente,
	}
	for nome, l := range checar {
		if len(l) != 1 {
			t.Errorf("%s: %v", nome, l)
		}
	}
}

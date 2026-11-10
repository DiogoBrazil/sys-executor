package enquadramento

import (
	"strconv"
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/rules/engine"
)

// Integridade do catálogo curado: todos os incisos dos arts. 15–17 do RDPM e
// 29/32 do Estatuto, sem lacuna; natureza fixa no RDPM, nunca no Estatuto.
// Regra: PADS-ENQ-001
func TestCatalogoIntegro(t *testing.T) {
	ds, err := Todos()
	if err != nil {
		t.Fatal(err)
	}
	cont := map[string]int{}
	for _, d := range ds {
		cont[d.Diploma+"-"+strconv.Itoa(d.Artigo)]++
		if d.Fonte.Arquivo == "" || d.Fonte.Pagina == 0 {
			t.Errorf("%s sem fonte", d.ID)
		}
		if strings.Contains(d.Texto, "Redação dada") || strings.Contains(d.Texto, "Seção") || strings.Contains(d.Texto, "- ") {
			t.Errorf("%s: resíduo de extração: %q", d.ID, d.Texto)
		}
		if n, ok := d.NaturezaFixa(); ok != (d.Diploma == RDPM) || (ok && n == "") {
			t.Errorf("%s: natureza %v/%v", d.ID, n, ok)
		}
	}
	quer := map[string]int{"RDPM-15": 17, "RDPM-16": 44, "RDPM-17": 39, "ESTATUTO-29": 23, "ESTATUTO-32": 7}
	for k, n := range quer {
		if cont[k] != n {
			t.Errorf("%s: %d itens, esperado %d", k, cont[k], n)
		}
	}
	if len(ds) != 130 {
		t.Fatalf("%d itens", len(ds))
	}
}

// Os três dispositivos do exemplo do usuário (09/11/2026).
func TestExemploDoUsuario(t *testing.T) {
	casos := map[string]struct {
		trecho string
		nat    engine.Natureza
	}{
		"RDPM-17-XXXIX": {"dirigir-se de maneira desrespeitosa ou desatenciosa a subordinado, par ou superior hierárquico", engine.Grave},
		"RDPM-17-IX":    {"deixar de cumprir ordem recebida, embaraçar ou retardar a sua execução", engine.Grave},
		"RDPM-16-XLIII": {"chegar atrasado à atividade para a qual esteja escalado", engine.Media},
	}
	for id, c := range casos {
		d, ok := PorID(id)
		n, _ := d.NaturezaFixa()
		if !ok || d.Texto != c.trecho || n != c.nat {
			t.Errorf("%s: %+v", id, d)
		}
	}
	d, _ := PorID("RDPM-17-XXXIX")
	if d.Referencia() != "art. 17, inciso XXXIX, do RDPM" {
		t.Fatal(d.Referencia())
	}
	e, _ := PorID("EST-29-XVIII-a")
	if !strings.HasPrefix(e.Texto, "abster-se o Militar do Estado") || !strings.Contains(e.Referencia(), "alínea a, do Estatuto dos Policiais Militares") {
		t.Fatalf("%+v %s", e, e.Referencia())
	}
}

func TestBuscarEDoCitado(t *testing.T) {
	if r := Buscar(RDPM, "atrasado"); len(r) != 1 || r[0].ID != "RDPM-16-XLIII" {
		t.Fatalf("%+v", r)
	}
	if r := Buscar(RDPM, "17 XXXIX"); len(r) != 1 || r[0].ID != "RDPM-17-XXXIX" {
		t.Fatalf("%+v", r)
	}
	if r := Buscar(Estatuto, "hierarquia"); len(r) == 0 || r[0].Diploma != Estatuto {
		t.Fatalf("%+v", r)
	}
	if d, ok := DoCitado("RDPM", 17, "v", ""); !ok || d.ID != "RDPM-17-V" {
		t.Fatalf("%+v", d)
	}
	if d, ok := DoCitado("", 16, "XLIII", ""); !ok || d.ID != "RDPM-16-XLIII" {
		t.Fatalf("sem diploma explícito: %+v", d)
	}
	if d, ok := DoCitado("Estatuto dos Policiais Militares", 32, "IV", ""); !ok || d.ID != "EST-32-IV" {
		t.Fatalf("%+v", d)
	}
	if _, ok := DoCitado("Código Penal Militar", 209, "", ""); ok {
		t.Fatal("diploma fora do catálogo")
	}
}

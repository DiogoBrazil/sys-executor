package autos

import "testing"

func TestIndiceBusca(t *testing.T) {
	a := &Autos{Paginas: []Pagina{
		{Numero: 1, Texto: "MEMORANDO DISCIPLINAR\nausentar-se do posto em 10/08/2026"},
		{Numero: 2, Texto: "TERMO DE INQUIRIÇÃO\nAs câmeras registraram a saída do posto."},
		{Numero: 3, Texto: ""},
	}}
	ix, err := Indexar(a)
	if err != nil {
		t.Fatal(err)
	}
	defer ix.Fechar()
	oc, err := ix.Buscar("inquiricao", 10)
	if err != nil || len(oc) != 1 || oc[0].Pagina != 2 {
		t.Fatalf("%+v %v", oc, err)
	}
	if r := []rune(a.Paginas[1].Texto); string(r[oc[0].Inicio:oc[0].Fim]) != "INQUIRIÇÃO" {
		t.Fatalf("posição: %+v", oc[0])
	}
	if oc, _ = ix.Buscar("Câmeras  posto", 10); len(oc) != 1 || oc[0].Pagina != 2 {
		t.Fatalf("E entre termos: %+v", oc)
	}
	if oc, _ = ix.Buscar("posto", 10); len(oc) != 2 {
		t.Fatalf("posto: %+v", oc)
	}
	// Sintaxe FTS5 maliciosa é neutralizada (vira termos literais).
	if _, err := ix.Buscar(`posto" OR texto:* NEAR(`, 10); err != nil {
		t.Fatalf("consulta deveria ser sanitizada: %v", err)
	}
	if oc, _ := ix.Buscar("  ", 10); oc != nil {
		t.Fatal("consulta vazia")
	}
}

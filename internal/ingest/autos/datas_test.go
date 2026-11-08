package autos

import "testing"

// A03: datas divergentes em peças diferentes são todas preservadas; as de OCR
// ficam "não confirmadas" (D-09.3).
func TestDatasPreservadas(t *testing.T) {
	a := &Autos{Paginas: []Pagina{
		{Numero: 1, Origem: OrigemNativo, Texto: "MEMORANDO DISCIPLINAR\nausentou-se do posto em 10/08/2026, às 14h00."},
		{Numero: 2, Origem: OrigemNativo, Texto: "Declarou que o fato ocorreu em 9 de agosto de 2026."},
		{Numero: 3, Origem: OrigemOCR, Texto: "Aos onze dias do mês de agosto de dois mil e vinte e seis, compareceu a testemunha."},
		{Numero: 4, Origem: OrigemNativo, Texto: "Prazo: 31/02/2026 (inválida) e telefone 10/08/20266, processo 0000.000001/2026-01."},
	}}
	ds := a.Datas()
	type esp struct {
		pag, d, m, ano int
		conf           bool
	}
	esperado := []esp{{1, 10, 8, 2026, true}, {2, 9, 8, 2026, true}, {3, 11, 8, 2026, false}}
	if len(ds) != len(esperado) {
		t.Fatalf("datas: %+v", ds)
	}
	for i, e := range esperado {
		d := ds[i]
		if d.Pagina != e.pag || d.Dia != e.d || d.Mes != e.m || d.Ano != e.ano || d.Confirmada != e.conf {
			t.Errorf("%d: %+v, esperado %+v", i, d, e)
		}
		trecho := string([]rune(a.Paginas[d.Pagina-1].Texto)[d.Inicio:d.Fim])
		if trecho == "" {
			t.Errorf("%d: posição vazia", i)
		}
		t.Logf("pág. %d: %q", d.Pagina, trecho)
	}
}

func TestValorExtenso(t *testing.T) {
	for s, v := range map[string]int{"dez": 10, "vinte e seis": 26, "primeiro": 1, "trinta e um": 31, "xis": -1} {
		if valorExtenso(s) != v {
			t.Errorf("%q", s)
		}
	}
}

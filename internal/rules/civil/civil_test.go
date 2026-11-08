package civil

import (
	"errors"
	"testing"
)

// Cobre: DT-01, DT-02, DT-03, DT-04, A27, A28
func TestMenosDeUmAno(t *testing.T) {
	base := D(7, 11, 2026)
	casos := []struct {
		id    string
		outra Data
		ok    bool
	}{
		{"DT-01", D(5, 11, 2027), true},
		{"DT-02/A27", D(6, 11, 2027), true},  // última data válida
		{"DT-03/A28", D(7, 11, 2027), false}, // exatamente um ano
		{"DT-04", D(8, 11, 2027), false},
	}
	for _, c := range casos {
		got, err := MenosDeUmAno(base, c.outra, SemEscolha29Fev)
		if err != nil || got != c.ok {
			t.Errorf("%s: %v (%v), esperado %v", c.id, got, err, c.ok)
		}
		// Ordem das datas não importa.
		if inv, _ := MenosDeUmAno(c.outra, base, SemEscolha29Fev); inv != c.ok {
			t.Errorf("%s invertido", c.id)
		}
	}
}

// Cobre: REI-PRZ-01, REI-PRZ-02, REI-PRZ-03, REI-D07-01, REI-D07-02, REI-D07-04
func TestDentroPrazoReincidencia(t *testing.T) {
	casos := []struct {
		id            string
		termino, fato Data
		anos          int
		ok            bool
	}{
		{"REI-PRZ-01", D(10, 1, 2025), D(9, 1, 2026), 1, true},
		{"REI-PRZ-02", D(10, 1, 2024), D(10, 1, 2026), 2, true}, // exatamente 2 anos (D-07)
		{"REI-PRZ-03", D(10, 1, 2022), D(11, 1, 2026), 4, false},
		{"REI-D07-01", D(10, 1, 2025), D(10, 1, 2026), 1, true}, // exatamente 1 ano
		{"REI-D07-02", D(10, 1, 2025), D(11, 1, 2026), 1, false},
	}
	for _, c := range casos {
		got, err := DentroPrazoReincidencia(c.termino, c.fato, c.anos, SemEscolha29Fev)
		if err != nil || got != c.ok {
			t.Errorf("%s: %v (%v), esperado %v", c.id, got, err, c.ok)
		}
	}
	// REI-D07-04: o mesmo par de datas (DT-03) diverge entre as duas regras.
	a, b := D(7, 11, 2026), D(7, 11, 2027)
	rei, _ := DentroPrazoReincidencia(a, b, 1, SemEscolha29Fev)
	menos, _ := MenosDeUmAno(a, b, SemEscolha29Fev)
	if !rei || menos {
		t.Fatalf("REI-D07-04: reincidência conta o aniversário (%v); 'inferior a um ano' não (%v)", rei, menos)
	}
}

// Cobre: REI-D07-03, DT-06
func TestVinteNoveDeFevereiro(t *testing.T) {
	// REI-D07-03: término em 29/02/2024; prazo de 1 ano cai em ano não bissexto.
	if _, err := DentroPrazoReincidencia(D(29, 2, 2024), D(28, 2, 2025), 1, SemEscolha29Fev); !errors.Is(err, ErrCasoLimite29Fev) {
		t.Fatalf("REI-D07-03: esperado ErrCasoLimite29Fev, veio %v", err)
	}
	if _, err := DentroPrazoReincidencia(D(29, 2, 2024), D(1, 3, 2025), 1, SemEscolha29Fev); !errors.Is(err, ErrCasoLimite29Fev) {
		t.Fatal("REI-D07-03 (01/03)")
	}
	// Com a escolha do usuário, o motor calcula.
	if ok, _ := DentroPrazoReincidencia(D(29, 2, 2024), D(1, 3, 2025), 1, Usar28Fev); ok {
		t.Fatal("28/02 escolhido: 01/03/2025 passa do prazo")
	}
	if ok, _ := DentroPrazoReincidencia(D(29, 2, 2024), D(1, 3, 2025), 1, Usar01Mar); !ok {
		t.Fatal("01/03 escolhido: 01/03/2025 ainda conta")
	}
	// DT-06: janela iniciada em 29/02/2028.
	if _, err := MenosDeUmAno(D(29, 2, 2028), D(28, 2, 2029), SemEscolha29Fev); !errors.Is(err, ErrCasoLimite29Fev) {
		t.Fatalf("DT-06: esperado ErrCasoLimite29Fev, veio %v", err)
	}
	// 29/02 para 29/02 (anos bissextos) não é caso-limite.
	if d, err := Aniversario(D(29, 2, 2024), 4, SemEscolha29Fev); err != nil || d != D(29, 2, 2028) {
		t.Fatalf("%v %v", d, err)
	}
}

// Cobre: R5, R6, R7, R8, A23, A24
func TestComparar(t *testing.T) {
	d := D(7, 11, 2026)
	casos := []struct {
		id   string
		a, b Instante
		r    Relacao
	}{
		{"R5/A23", Instante{d, H(8, 0)}, Instante{d, H(16, 0)}, Antes},
		{"R6", Instante{d, H(18, 0)}, Instante{d, H(8, 0)}, Depois},
		{"R7/A24", Instante{d, Hora{}}, Instante{d, Hora{}}, Indeterminado},
		{"R7-meia", Instante{d, H(8, 0)}, Instante{d, Hora{}}, Indeterminado},
		{"R8/A24", Instante{d, H(8, 0)}, Instante{d, H(8, 0)}, Simultaneo},
		{"datas", Instante{D(6, 11, 2026), Hora{}}, Instante{d, Hora{}}, Antes},
	}
	for _, c := range casos {
		if got := Comparar(c.a, c.b); got != c.r {
			t.Errorf("%s: %s, esperado %s", c.id, got, c.r)
		}
	}
}

func TestDatas(t *testing.T) {
	if _, err := NovaData(2026, 2, 29); err == nil {
		t.Fatal("29/02/2026 não existe")
	}
	if D(1, 3, 2026).String() != "01/03/2026" || !Bissexto(2000) || Bissexto(1900) || !Bissexto(2028) {
		t.Fatal("formatação/bissexto")
	}
}

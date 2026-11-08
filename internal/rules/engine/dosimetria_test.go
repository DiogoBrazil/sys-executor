package engine

import "testing"

func ag(inciso int, alinea string) Circunstancia {
	return Circunstancia{Artigo: 39, Inciso: inciso, Alinea: alinea, Fundamento: "fundamento específico"}
}
func at(inciso int) Circunstancia {
	return Circunstancia{Artigo: 38, Inciso: inciso, Fundamento: "fundamento específico"}
}

// Cobre: C13-DIA-01, C13-DIA-02, C13-DIA-03
func TestDiasTabela13(t *testing.T) {
	casos := []struct {
		id         string
		atenuantes []Circunstancia
		agravantes []Circunstancia
		dias       int
	}{
		{"C13-DIA-01", nil, nil, 2},
		{"C13-DIA-02", nil, []Circunstancia{ag(7, "a"), ag(7, "b"), ag(7, "c")}, 3}, // não 5
		{"C13-DIA-03", []Circunstancia{at(3), at(4)}, []Circunstancia{ag(7, "a")}, 1},
	}
	for _, c := range casos {
		for _, s := range []Sancao{Detencao, Prisao} {
			r := Dosimetria(EntradaDosimetria{Sancao: s, Atenuantes: c.atenuantes, Agravantes: c.agravantes, ComportamentoAnterior: Bom})
			if r.Dias != c.dias {
				t.Errorf("%s (%s): %d dias, esperado %d (%s)", c.id, s, r.Dias, c.dias, r.Regra)
			}
		}
	}
}

// Cobre: DOS-D05-01, DOS-D05-02, DOS-D05-03, DOS-D05-04
func TestContagemD05(t *testing.T) {
	r := Dosimetria(EntradaDosimetria{Sancao: Detencao, ComportamentoAnterior: Bom,
		Atenuantes: []Circunstancia{at(3), at(4)}, Agravantes: []Circunstancia{ag(7, "a"), ag(7, "d")}})
	if r.Dias != 2 || len(r.AgravantesContadas) != 2 {
		t.Fatalf("DOS-D05-01: %+v", r)
	}
	r = Dosimetria(EntradaDosimetria{Sancao: Detencao, ComportamentoAnterior: Bom,
		Atenuantes: []Circunstancia{at(3)}, Agravantes: []Circunstancia{ag(7, "a"), ag(7, "b"), ag(7, "e")}})
	if r.Dias != 3 || len(r.AgravantesContadas) != 3 {
		t.Fatalf("DOS-D05-02: %+v", r)
	}
	// Reincidência (três punições válidas) conta uma vez; listada à parte, é ignorada.
	r = Dosimetria(EntradaDosimetria{Sancao: Prisao, ComportamentoAnterior: Bom, Reincidencia: true,
		Atenuantes: []Circunstancia{at(3), at(4)}, Agravantes: []Circunstancia{ag(7, "a"), ag(3, ""), ag(3, "")}})
	if r.Dias != 2 || len(r.AgravantesContadas) != 2 {
		t.Fatalf("DOS-D05-03: %+v", r)
	}
	// Conexão com duas secundárias: inciso II uma vez.
	r = Dosimetria(EntradaDosimetria{Sancao: Prisao, ComportamentoAnterior: Bom, ConexasSecundarias: 2, Agravantes: []Circunstancia{ag(2, "")}})
	if len(r.AgravantesContadas) != 1 {
		t.Fatalf("DOS-D05-04: %+v", r)
	}
}

// Cobre: DOS-LIM-01
func TestLimites(t *testing.T) {
	if r := Dosimetria(EntradaDosimetria{Sancao: Repreensao, Agravantes: []Circunstancia{ag(7, "a")}, ComportamentoAnterior: Bom}); r.Dias != 0 {
		t.Fatal("repreensão não tem dias")
	}
	for _, s := range []Sancao{Detencao, Prisao} {
		for _, comb := range [][]Circunstancia{nil, {ag(7, "a"), ag(7, "b"), ag(7, "c"), ag(7, "d"), ag(7, "e")}} {
			r := Dosimetria(EntradaDosimetria{Sancao: s, Agravantes: comb, ComportamentoAnterior: Insuficiente})
			if err := ValidarDias(s, r.Dias); err != nil || r.Dias > LimiteDiasDetencao {
				t.Fatalf("%s: %d dias: %v", s, r.Dias, err)
			}
		}
	}
	if ValidarDias(Detencao, 11) == nil || ValidarDias(Repreensao, 1) == nil || ValidarDias(Detencao, 0) == nil {
		t.Fatal("limites não validados")
	}
}

// Cobre: DOS-CMP-01, DOS-CMP-02, DOS-CMP-03
func TestCircunstanciaComportamentoAnterior(t *testing.T) {
	casos := map[Comportamento][2]int{Otimo: {1, 0}, Excepcional: {1, 0}, Insuficiente: {0, 1}, Bom: {0, 0}}
	for c, n := range casos {
		r := Dosimetria(EntradaDosimetria{Sancao: Detencao, ComportamentoAnterior: c})
		if len(r.AtenuantesContadas) != n[0] || len(r.AgravantesContadas) != n[1] {
			t.Errorf("%s: %+v", c, r)
		}
	}
}

// Cobre: DOS-36-01, DOS-37-01, DOS-38-01, DOS-38II-01
func TestFundamentacaoObrigatoria(t *testing.T) {
	if ps := VerificarArt36(map[int]string{1: "a", 2: "b", 3: "c", 4: "d", 5: "e", 6: "f"}); len(ps) != 1 || ps[0].Ref != "VII" {
		t.Fatalf("DOS-36-01: %+v", ps)
	}
	if ps := VerificarArt36(map[int]string{1: "x", 2: "x", 3: "x", 4: "x", 5: "x", 6: "x", 7: "ausência de elementos"}); len(ps) != 0 {
		t.Fatal("DOS-36-01: 'ausência de elementos' é exame válido")
	}
	r := Dosimetria(EntradaDosimetria{Sancao: Detencao, ComportamentoAnterior: Bom,
		Justificacoes: []Circunstancia{{Artigo: 37, Inciso: 1}},
		Atenuantes:    []Circunstancia{{Artigo: 38, Inciso: 3}}})
	if len(r.AtenuantesContadas) != 0 || len(r.Pendencias) != 2 {
		t.Fatalf("DOS-37-01/DOS-38-01: sem fundamento não conta e gera pendência: %+v", r)
	}
	sim, nao := true, false
	r = Dosimetria(EntradaDosimetria{Sancao: Detencao, ComportamentoAnterior: Bom, ServicosRelevantes: &sim})
	if len(r.AtenuantesContadas) != 0 || !temPend(r.Pendencias, PendFundamentacao) {
		t.Fatalf("DOS-38II-01 sem justificativa: %+v", r)
	}
	r = Dosimetria(EntradaDosimetria{Sancao: Detencao, ComportamentoAnterior: Bom, ServicosRelevantes: &sim, JustificativaServicos: "atuação registrada em ofício"})
	if len(r.AtenuantesContadas) != 1 || r.Dias != 1 {
		t.Fatalf("DOS-38II-01 com justificativa: %+v", r)
	}
	r = Dosimetria(EntradaDosimetria{Sancao: Detencao, ComportamentoAnterior: Bom, ServicosRelevantes: &nao, Atenuantes: []Circunstancia{at(2)}})
	if len(r.AtenuantesContadas) != 0 {
		t.Fatalf("DOS-38II-01 'não': sem atenuante II: %+v", r)
	}
}

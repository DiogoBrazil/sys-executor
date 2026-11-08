package engine

import (
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/rules/civil"
)

var D, H = civil.D, civil.H

func inst(d civil.Data) civil.Instante { return civil.Instante{Data: d} }
func instH(d civil.Data, h civil.Hora) *civil.Instante {
	i := civil.Instante{Data: d, Hora: h}
	return &i
}
func term(d civil.Data) *civil.Instante { i := inst(d); return &i }

func temPend(ps []Pendencia, t TipoPendencia) bool {
	for _, p := range ps {
		if p.Tipo == t {
			return true
		}
	}
	return false
}

// Cobre: R1, R2, R3, R4, R5, R6, R7, R8, A21, A22, A23, A24, A26, C13-REI-01, C13-REI-02
func TestReincidenciaTabela(t *testing.T) {
	fato := inst(D(7, 11, 2026))
	casos := []struct {
		id       string
		nAtual   Natureza
		fato     civil.Instante
		anterior PunicaoAnterior
		ok       bool
		pend     TipoPendencia
	}{
		{"R1", Grave, fato, PunicaoAnterior{"A", Grave, term(D(10, 5, 2024))}, true, ""},
		{"R2/A22", Grave, fato, PunicaoAnterior{"A", Grave, term(D(15, 12, 2026))}, false, ""},
		{"R3/A26", Grave, fato, PunicaoAnterior{"A", Grave, nil}, false, PendDataTerminoDesconhecida},
		{"R4/A21/C13-REI-01", Grave, inst(D(10, 7, 2026)), PunicaoAnterior{"A", Leve, term(D(10, 1, 2025))}, false, ""},
		{"R5/A23", Grave, civil.Instante{Data: D(7, 11, 2026), Hora: H(16, 0)}, PunicaoAnterior{"A", Grave, instH(D(7, 11, 2026), H(8, 0))}, true, ""},
		{"R6", Grave, civil.Instante{Data: D(7, 11, 2026), Hora: H(8, 0)}, PunicaoAnterior{"A", Grave, instH(D(7, 11, 2026), H(18, 0))}, false, ""},
		{"R7/A24", Grave, fato, PunicaoAnterior{"A", Grave, term(D(7, 11, 2026))}, false, PendHorarios},
		{"R8/A24", Grave, civil.Instante{Data: D(7, 11, 2026), Hora: H(8, 0)}, PunicaoAnterior{"A", Grave, instH(D(7, 11, 2026), H(8, 0))}, false, ""},
		{"C13-REI-02", Grave, inst(D(10, 7, 2026)), PunicaoAnterior{"A", Grave, term(D(10, 1, 2025))}, true, ""},
	}
	for _, c := range casos {
		r := Reincidencia(c.nAtual, c.fato, []PunicaoAnterior{c.anterior}, civil.SemEscolha29Fev)
		if r.Reconhecida != c.ok {
			t.Errorf("%s: reconhecida=%v, esperado %v (%+v)", c.id, r.Reconhecida, c.ok, r.Descartadas)
		}
		if c.pend != "" && !temPend(r.Pendencias, c.pend) {
			t.Errorf("%s: esperada pendência %s", c.id, c.pend)
		}
		if !c.ok && len(r.Descartadas) != 1 {
			t.Errorf("%s: o descarte precisa ter motivo registrado", c.id)
		}
	}
}

// Cobre: R9, A25
func TestReincidenciaVariasPunicoesUmaAgravante(t *testing.T) {
	r := Reincidencia(Grave, inst(D(7, 11, 2026)), []PunicaoAnterior{
		{"A", Grave, term(D(10, 1, 2020))}, // fora do prazo
		{"B", Grave, term(D(10, 5, 2024))},
		{"C", Grave, term(D(15, 3, 2026))},
	}, civil.SemEscolha29Fev)
	if !r.Reconhecida || len(r.Validas) != 2 || len(r.Descartadas) != 1 {
		t.Fatalf("%+v", r)
	}
	// Uma atenuante e nenhuma outra agravante: 1 × 1 → 2 dias (não 3).
	d := Dosimetria(EntradaDosimetria{Sancao: Prisao, Reincidencia: r.Reconhecida, ComportamentoAnterior: Bom,
		Atenuantes: []Circunstancia{{Artigo: 38, Inciso: 3, Fundamento: "confissão espontânea"}}})
	if d.Dias != 2 || len(d.AgravantesContadas) != 1 {
		t.Fatalf("R9: %+v", d)
	}
}

// Cobre: REI-D07-03
func TestReincidencia29Fev(t *testing.T) {
	r := Reincidencia(Leve, inst(D(28, 2, 2025)), []PunicaoAnterior{{"A", Leve, term(D(29, 2, 2024))}}, civil.SemEscolha29Fev)
	if r.Reconhecida || !temPend(r.Pendencias, PendCaso29Fev) {
		t.Fatalf("%+v", r)
	}
	r = Reincidencia(Leve, inst(D(28, 2, 2025)), []PunicaoAnterior{{"A", Leve, term(D(29, 2, 2024))}}, civil.Usar28Fev)
	if !r.Reconhecida {
		t.Fatal("com 28/02 escolhido pelo usuário, 28/02/2025 está no prazo")
	}
}

// Cobre: REI-CNX-01
func TestReincidenciaConexasUsaPrincipal(t *testing.T) {
	conx := AplicarConexao([]ImputacaoProcedente{
		{ID: "IMP-1", Natureza: Grave, Fato: inst(D(7, 11, 2026)), Grupo: "g"},
		{ID: "IMP-2", Natureza: Media, Fato: inst(D(7, 11, 2026)), Grupo: "g"},
	}, map[string]string{"g": "IMP-1"})
	anterior := []PunicaoAnterior{{"A", Media, term(D(1, 1, 2026))}}
	for _, p := range conx.Principais {
		if r := Reincidencia(p.Natureza, p.Fato, anterior, civil.SemEscolha29Fev); r.Reconhecida {
			t.Fatal("conexas: só a natureza da principal (grave) conta; anterior média não reincide")
		}
	}
	if len(conx.Principais) != 1 {
		t.Fatal("só a principal entra")
	}
}

// Cobre: PRG-REI-01, PRG-REI-02
func TestPerguntasReincidencia(t *testing.T) {
	g := Grave
	sim, nao := true, false
	if ProximaPerguntaReincidencia(nil, nil, false) != SemPergunta {
		t.Fatal("PRG-REI-01: natureza não fixada → nenhuma pergunta")
	}
	if ProximaPerguntaReincidencia(&g, nil, false) != PerguntarSeHaAnterior {
		t.Fatal("pergunta se há anterior da mesma natureza")
	}
	if ProximaPerguntaReincidencia(&g, &nao, false) != SemPergunta {
		t.Fatal("PRG-REI-02: resposta não → não pede data")
	}
	if ProximaPerguntaReincidencia(&g, &sim, false) != PerguntarDataTermino {
		t.Fatal("sim → pede data de término")
	}
	if ProximaPerguntaReincidencia(&g, &sim, true) != SemPergunta {
		t.Fatal("data já conhecida dos autos → não pergunta de novo")
	}
}

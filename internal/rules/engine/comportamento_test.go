package engine

import (
	"fmt"
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/rules/civil"
)

func hist(id string, n Natureza, d civil.Data) Transgressao {
	return Transgressao{ID: id, Natureza: n, Data: d, Origem: Historico}
}
func atual(id string, n Natureza, d civil.Data) Transgressao {
	return Transgressao{ID: id, Natureza: n, Data: d, Origem: Atual}
}
func comp(c Comportamento) *Comportamento { return &c }
func pts(p int) *int                      { return &p }

// sancoes gera as sanções das atuais (independentes).
func sancoesDe(fs []Transgressao) []SancaoAplicada {
	var out []SancaoAplicada
	for _, f := range fs {
		if f.Origem == Atual {
			out = append(out, SancaoAplicada{f.ID, f.Natureza, f.Natureza.Sancao()})
		}
	}
	return out
}

func pontuar(c Comportamento, p int, fatos ...Transgressao) ResultadoPontuacao {
	return Pontuacao(EntradaPontuacao{Comportamento: comp(c), Pontos: pts(p), Sancoes: sancoesDe(fatos), Fatos: fatos})
}

// Cobre: CMP-FX-01…08
func TestFaixas(t *testing.T) {
	esperado := map[int]Comportamento{41: Excepcional, 40: Otimo, 30: Otimo, 29: Bom, 10: Bom, 9: Insuficiente, 0: Insuficiente, 45: Excepcional}
	for p, c := range esperado {
		if ClassificarSaldo(p) != c {
			t.Errorf("%d pontos: %s, esperado %s", p, ClassificarSaldo(p), c)
		}
	}
}

// Cobre: EQV-01…08, C13-EQV-01, E3, EQV-DUP-01, EQV-PEND-01
func TestEquivalencias(t *testing.T) {
	g, m, l := Grave, Media, Leve
	n := 0
	f := func(nat ...Natureza) []Transgressao {
		var out []Transgressao
		for _, x := range nat {
			n++
			out = append(out, Transgressao{ID: fmt.Sprint(n), Natureza: x, Origem: Historico})
		}
		return out
	}
	casos := []struct {
		id    string
		fatos []Transgressao
		texto string
	}{
		{"EQV-01", f(g, m, m), "2 graves"},
		{"EQV-02", f(g, l, l, l, l), "2 graves"},
		{"EQV-03", f(l, l, l, l, l, l, l, l), "2 graves"},
		{"EQV-04", f(m, m, m, m), "2 graves"},
		{"EQV-05/E3", f(m, m, l, l), "1 grave e 1 média"},
		{"EQV-06", f(l, l, l), "1 média e 1 leve"},
		{"EQV-07", f(m, m, m, l, l), "2 graves"},
		{"EQV-08", f(l), "1 leve"},
		{"C13-EQV-01", f(l, l, l, l, m, m), "2 graves"},
	}
	for _, c := range casos {
		if got := Equivaler(c.fatos).String(); got != c.texto {
			t.Errorf("%s: %q, esperado %q", c.id, got, c.texto)
		}
	}
	// EQV-DUP-01: a mesma infração (mesmo ID) só conta uma vez.
	dup := []Transgressao{{ID: "X", Natureza: Grave, Origem: Historico}, {ID: "X", Natureza: Grave, Origem: Historico}}
	if e := Equivaler(dup); e.Unidades != 4 || len(e.Usadas) != 1 {
		t.Fatalf("EQV-DUP-01: %+v", e)
	}
	// EQV-PEND-01: acusação pendente não entra.
	pendente := []Transgressao{{ID: "P", Natureza: Grave, Origem: Pendente}, {ID: "Q", Natureza: Leve, Origem: Historico}}
	if e := Equivaler(pendente); e.Unidades != 1 {
		t.Fatalf("EQV-PEND-01: %+v", e)
	}
}

// Cobre: E1, C13-4M-01, E2, E3
func TestGatilhosE1E2E3(t *testing.T) {
	e1 := []Transgressao{
		hist("M1", Media, D(20, 12, 2025)), hist("M2", Media, D(12, 2, 2026)), hist("M3", Media, D(23, 7, 2026)),
		atual("M4", Media, D(4, 11, 2026)),
	}
	if r := pontuar(Otimo, 35, e1...); r.Regressao != RegressaoBomOtimo || r.PontosFinais != 20 || r.ComportamentoFinal != Bom {
		t.Fatalf("E1 ótimo: %+v", r)
	}
	if r := pontuar(Bom, 25, e1...); r.PontosFinais != 9 || r.ComportamentoFinal != Insuficiente || len(r.Descontos) != 0 {
		t.Fatalf("E1 bom: %+v", r)
	}
	e2 := []Transgressao{
		hist("M1", Media, D(10, 1, 2026)), hist("M2", Media, D(15, 3, 2026)),
		hist("L1", Leve, D(20, 5, 2026)), hist("L2", Leve, D(10, 7, 2026)), hist("L3", Leve, D(15, 9, 2026)),
		atual("L4", Leve, D(7, 11, 2026)),
	}
	if r := pontuar(Otimo, 33, e2...); r.PontosFinais != 20 || r.ComportamentoFinal != Bom {
		t.Fatalf("E2: %+v", r)
	}
	e3 := []Transgressao{hist("M1", Media, D(10, 1, 2026)), hist("L1", Leve, D(20, 5, 2026)), hist("L2", Leve, D(1, 6, 2026)), atual("M2", Media, D(7, 11, 2026))}
	r := pontuar(Otimo, 35, e3...)
	if r.Regressao != SemRegressao || r.PontosFinais != 33 || len(r.Descontos) != 1 {
		t.Fatalf("E3: sem gatilho, desconto ordinário: %+v", r)
	}
}

// Cobre: DT-05, CMP-D06-01, CMP-D06-02, CMP-D06-03
func TestJanelaD06(t *testing.T) {
	// DT-05: três fatos que não cabem numa única janela; mostra o subconjunto.
	r := GatilhoBomOtimo([]Transgressao{
		hist("G1", Grave, D(1, 1, 2025)), hist("G2", Grave, D(30, 12, 2025)), atual("G3", Grave, D(2, 1, 2026)),
	}, civil.SemEscolha29Fev)
	if !r.Acionado || len(r.Janela.Fatos) != 2 || r.Janela.Fatos[0].ID != "G2" || r.Janela.Inicio != D(30, 12, 2025) {
		t.Fatalf("DT-05: %+v", r.Janela)
	}
	// CMP-D06-01: o histórico sozinho fecha o gatilho; a atual está fora da janela.
	r = GatilhoBomOtimo([]Transgressao{
		hist("G1", Grave, D(1, 2, 2025)), hist("G2", Grave, D(1, 6, 2025)), atual("L1", Leve, D(1, 10, 2026)),
	}, civil.SemEscolha29Fev)
	if r.Acionado || r.HistoricoSozinho == nil || !temPend(r.Pendencias, PendHistoricoSozinho) {
		t.Fatalf("CMP-D06-01: %+v", r)
	}
	if p := pontuar(Otimo, 35, hist("G1", Grave, D(1, 2, 2025)), hist("G2", Grave, D(1, 6, 2025)), atual("L1", Leve, D(1, 10, 2026))); p.Concluido {
		t.Fatal("CMP-D06-01: nada é recalculado silenciosamente; aguarda resposta")
	}
	// CMP-D06-02: 1 grave do histórico + 1 grave atual dentro de um ano.
	if r := GatilhoBomOtimo([]Transgressao{hist("G1", Grave, D(10, 3, 2026)), atual("G2", Grave, D(9, 3, 2027))}, civil.SemEscolha29Fev); !r.Acionado {
		t.Fatalf("CMP-D06-02: %+v", r)
	}
	// CMP-D06-03: janelas móveis; a escolhida contém fato atual.
	// Uma janela antiga só de histórico (M1–M4) já soma 2 graves; outra, mais
	// recente (M5–M8), também soma 2 graves e contém o fato atual M6.
	r = GatilhoBomOtimo([]Transgressao{
		hist("M1", Media, D(1, 1, 2025)), hist("M2", Media, D(1, 3, 2025)), hist("M3", Media, D(1, 5, 2025)), hist("M4", Media, D(1, 7, 2025)),
		hist("M5", Media, D(1, 2, 2026)), atual("M6", Media, D(15, 4, 2026)), hist("M7", Media, D(1, 6, 2026)), hist("M8", Media, D(1, 9, 2026)),
	}, civil.SemEscolha29Fev)
	if !r.Acionado || !r.Janela.ContemAtual || r.Janela.Equivalente.Graves < 2 {
		t.Fatalf("CMP-D06-03: %+v", r.Janela)
	}
	for _, f := range r.Janela.Fatos {
		if ok, _ := civil.MenosDeUmAno(r.Janela.Inicio, f.Data, civil.SemEscolha29Fev); !ok {
			t.Fatalf("CMP-D06-03: fato %s fora da janela demonstrada", f.ID)
		}
	}
	ids := fmt.Sprint(idsDe(r.Janela.Fatos))
	if !strings.Contains(ids, "M6") {
		t.Fatalf("CMP-D06-03: janela sem o fato atual: %s", ids)
	}
}

func idsDe(fs []Transgressao) []string { return ids(fs) }

// Cobre: DT-06
func TestJanela29FevSoPerguntaQuandoMaterial(t *testing.T) {
	// Não material: as duas escolhas dão o mesmo resultado → sem pergunta.
	r := GatilhoBomOtimo([]Transgressao{hist("G1", Grave, D(29, 2, 2028)), atual("G2", Grave, D(1, 6, 2028))}, civil.SemEscolha29Fev)
	if !r.Acionado || temPend(r.Pendencias, PendCaso29Fev) {
		t.Fatalf("não material: %+v", r)
	}
	// Material: o fato atual cai exatamente em 28/02/2029.
	r = GatilhoBomOtimo([]Transgressao{hist("G1", Grave, D(29, 2, 2028)), atual("G2", Grave, D(28, 2, 2029))}, civil.SemEscolha29Fev)
	if r.Acionado || !temPend(r.Pendencias, PendCaso29Fev) {
		t.Fatalf("material: %+v", r)
	}
}

// Cobre: MP-01, MP-02, MP-03, MP-04, MP-05, MP-06, A18, A19, A20, C13-REG-01, C13-PTS-01, C13-INS-01
func TestPontuacao154(t *testing.T) {
	g := func(id string, d int) Transgressao { return atual(id, Grave, D(d, 3, 2026)) }
	l := atual("L1", Leve, D(20, 3, 2026))
	casos := []struct {
		id     string
		c      Comportamento
		p      int
		fatos  []Transgressao
		pontos int
		final  Comportamento
	}{
		{"MP-01", Otimo, 35, []Transgressao{g("G1", 1), g("G2", 10)}, 20, Bom},
		{"MP-02/A19", Otimo, 35, []Transgressao{g("G1", 1), g("G2", 10), g("G3", 15)}, 20, Bom},
		{"MP-03/A18", Otimo, 35, []Transgressao{g("G1", 1), g("G2", 10), l}, 20, Bom},
		{"MP-04/A20/C13-REG-01", Bom, 29, []Transgressao{g("G1", 1), g("G2", 10)}, 9, Insuficiente},
		{"MP-05", Otimo, 35, []Transgressao{g("G1", 1)}, 31, Otimo},
		{"MP-06", Otimo, 35, []Transgressao{g("G1", 1), l}, 30, Otimo},
		{"C13-PTS-01", Otimo, 31, []Transgressao{atual("M1", Media, D(1, 3, 2026))}, 29, Bom},
		{"C13-INS-01", Insuficiente, 7, []Transgressao{g("G1", 1)}, 3, Insuficiente},
	}
	for _, c := range casos {
		r := pontuar(c.c, c.p, c.fatos...)
		if !r.Concluido || r.PontosFinais != c.pontos || r.ComportamentoFinal != c.final {
			t.Errorf("%s: %d %s (concluído=%v), esperado %d %s", c.id, r.PontosFinais, r.ComportamentoFinal, r.Concluido, c.pontos, c.final)
		}
		if r.Regressao != SemRegressao && len(r.Descontos) != 0 {
			t.Errorf("%s: regressão especial não acumula descontos ordinários", c.id)
		}
	}
}

// Cobre: EX1, A30, C13-EXC-01, EX2, A29, C13-EXC-02
func TestExcepcional(t *testing.T) {
	ex := func(id string, ano int) Transgressao {
		f := hist(id, Grave, D(1, 6, ano))
		f.PeriodoExcepcionalAtual = true
		return f
	}
	// EX1: duas graves do período atual (anos distantes: sem janela) + grave atual.
	r := pontuar(Excepcional, 45, ex("G1", 2019), ex("G2", 2024), atual("G3", Grave, D(7, 11, 2026)))
	if r.Regressao != RegressaoExcepcional || r.PontosFinais != 30 || r.ComportamentoFinal != Otimo || len(r.Descontos) != 0 {
		t.Fatalf("EX1: %+v", r)
	}
	m := MontarMemoria(EntradaPontuacao{Comportamento: comp(Excepcional), Pontos: pts(45)}, r, ResultadoConexao{})
	if !strings.Contains(m.Itens[5].Linhas[0], "depende do ato administrativo") {
		t.Fatal("A30: a minuta propõe; não afirma alteração já feita")
	}
	// EX2: as três graves consumidas em 2022 não contam de novo.
	consumida := func(id string) Transgressao {
		f := hist(id, Grave, D(1, 3, 2022))
		f.ConsumidaEmRegressao, f.PeriodoExcepcionalAtual = true, true
		return f
	}
	r = pontuar(Excepcional, 45, consumida("C1"), consumida("C2"), consumida("C3"), atual("G4", Grave, D(1, 5, 2026)))
	if r.Regressao != SemRegressao || r.PontosFinais != 41 {
		t.Fatalf("EX2: sem gatilho, desconto ordinário: %+v", r)
	}
	if len(r.Excepcional.Desconsideradas) != 3 {
		t.Fatalf("EX2: consumidas devem aparecer como desconsideradas: %+v", r.Excepcional)
	}
	// I-02: o histórico do período já com 3 graves → inconsistência.
	r = pontuar(Excepcional, 45, ex("G1", 2019), ex("G2", 2020), ex("G3", 2021), atual("L", Leve, D(1, 1, 2026)))
	if r.Concluido || !temPend(r.Pendencias, PendExcepcionalInconsistente) {
		t.Fatalf("I-02: %+v", r)
	}
}

// Cobre: CMP-DIV-01, CMP-AUS-01, SEG13-05, CMP-NEG-01, CMP-LIM-01
func TestPontuacaoCasosLimite(t *testing.T) {
	if r := pontuar(Otimo, 25, atual("G", Grave, D(1, 1, 2026))); r.Concluido || !temPend(r.Pendencias, PendDivergenciaComportamento) {
		t.Fatalf("CMP-DIV-01: %+v", r)
	}
	r := Pontuacao(EntradaPontuacao{Sancoes: []SancaoAplicada{{"G", Grave, Prisao}}})
	if r.Concluido || !temPend(r.Pendencias, PendDadosComportamento) {
		t.Fatalf("CMP-AUS-01: %+v", r)
	}
	m := MontarMemoria(EntradaPontuacao{}, r, ResultadoConexao{})
	if !strings.Contains(m.Itens[0].Linhas[0], "[A PREENCHER]") || !strings.Contains(m.Itens[5].Linhas[0], "[A PREENCHER]") {
		t.Fatal("SEG13-05: sem pontos, nada é inventado")
	}
	r = pontuar(Insuficiente, 2, atual("G", Grave, D(1, 1, 2026)))
	if !r.Concluido || r.PontosFinais != -2 || !r.SaldoNegativo || !temPend(r.Pendencias, PendSaldoNegativo) {
		t.Fatalf("CMP-NEG-01: %+v", r)
	}
	// CMP-LIM-01: punição de outro processo, julgado antes, é sempre histórico:
	// nunca recebe desconto deste julgamento nem entra na regra do valor fixo.
	outro := hist("OUTRO-PADS", Grave, D(1, 2, 2026))
	r = Pontuacao(EntradaPontuacao{Comportamento: comp(Otimo), Pontos: pts(35),
		Sancoes: []SancaoAplicada{{"G", Grave, Prisao}}, Fatos: []Transgressao{outro, atual("G", Grave, D(1, 3, 2027))}})
	for _, d := range r.Descontos {
		if d.ImputacaoID == "OUTRO-PADS" {
			t.Fatal("CMP-LIM-01: processo distinto não é unificado")
		}
	}
	if r.PontosFinais != 31 {
		t.Fatalf("CMP-LIM-01: %+v", r)
	}
}

// Cobre: C13-CNX-01, C13-IND-01
func TestConexao(t *testing.T) {
	imps := []ImputacaoProcedente{
		{ID: "IMP-1", Natureza: Grave, Fato: inst(D(1, 3, 2026)), Grupo: "g"},
		{ID: "IMP-2", Natureza: Media, Fato: inst(D(1, 3, 2026)), Grupo: "g"},
	}
	sem := AplicarConexao(imps, nil)
	if len(sem.Sancoes) != 0 || !temPend(sem.Pendencias, PendPrincipalConexao) || !strings.Contains(sem.Pendencias[0].Mensagem, "IMP-1") {
		t.Fatalf("I-04: sem confirmação, só sugestão: %+v", sem)
	}
	c := AplicarConexao(imps, map[string]string{"g": "IMP-1"})
	if len(c.Sancoes) != 1 || c.Sancoes[0].Sancao != Prisao || len(c.Secundarias["IMP-1"]) != 1 {
		t.Fatalf("C13-CNX-01: %+v", c)
	}
	if len(c.FatosAtuais()) != 1 {
		t.Fatal("C13-CNX-01: sem dupla contagem nas equivalências")
	}
	d := Dosimetria(EntradaDosimetria{Sancao: Prisao, ConexasSecundarias: len(c.Secundarias["IMP-1"]), ComportamentoAnterior: Bom})
	if len(d.AgravantesContadas) != 1 || d.AgravantesContadas[0] != "art. 39, II" {
		t.Fatalf("C13-CNX-01: média conexa como agravante: %+v", d)
	}
	// C13-IND-01: leve e grave independentes → repreensão + prisão; −1 −4.
	ind := AplicarConexao([]ImputacaoProcedente{
		{ID: "L", Natureza: Leve, Fato: inst(D(1, 3, 2026))}, {ID: "G", Natureza: Grave, Fato: inst(D(2, 3, 2026))},
	}, nil)
	if len(ind.Sancoes) != 2 || ind.Sancoes[0].Sancao != Repreensao || ind.Sancoes[1].Sancao != Prisao {
		t.Fatalf("C13-IND-01: %+v", ind)
	}
	r := Pontuacao(EntradaPontuacao{Comportamento: comp(Otimo), Pontos: pts(38), Sancoes: ind.Sancoes, Fatos: ind.FatosAtuais()})
	if r.PontosFinais != 33 || len(r.Descontos) != 2 {
		t.Fatalf("C13-IND-01: %+v", r)
	}
}

// Memória de cálculo com os sete itens (V4 §8.8); propõe, não declara a
// alteração administrativa.
// Regra: PADS-CMP-015, PADS-CMP-014
func TestMemoriaSeteItens(t *testing.T) {
	fatos := []Transgressao{hist("M1", Media, D(20, 12, 2025)), hist("M2", Media, D(12, 2, 2026)), hist("M3", Media, D(23, 7, 2026)), atual("M4", Media, D(4, 11, 2026))}
	e := EntradaPontuacao{Comportamento: comp(Otimo), Pontos: pts(35), Sancoes: sancoesDe(fatos), Fatos: fatos}
	conx := AplicarConexao([]ImputacaoProcedente{{ID: "M4", Natureza: Media, Fato: inst(D(4, 11, 2026))}}, nil)
	m := MontarMemoria(e, Pontuacao(e), conx)
	for i, it := range m.Itens {
		if it.Titulo == "" || len(it.Linhas) == 0 {
			t.Errorf("item %d vazio", i+1)
		}
	}
	tudo := fmt.Sprint(m)
	for _, s := range []string{"35 pontos", "M4", "20/12/2025", "2 graves", "fixa de 20 pontos", "comportamento bom", "arts. 70 e 71",
		"Propõe-se", "depende do ato administrativo competente"} {
		if !strings.Contains(tudo, s) {
			t.Errorf("memória sem %q", s)
		}
	}
	for _, s := range []string{"foi alterado", "passa a ser", "fica alterado", "ficha"} {
		if strings.Contains(tudo, s) {
			t.Errorf("memória declara alteração efetivada: %q", s)
		}
	}
}

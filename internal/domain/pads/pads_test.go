package pads

import (
	"errors"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/rules/civil"
	"github.com/DiogoBrazil/sys-executor/internal/rules/engine"
)

func imp(id string, n engine.Natureza) Imputacao {
	return Imputacao{ID: id, Texto: "conduta " + id, Dispositivo: "art. 15", Natureza: n, Fato: civil.Instante{Data: civil.D(7, 11, 2026)}}
}

func conjunto(t *testing.T, ids ...string) *ImputacaoSet {
	t.Helper()
	s := NovoImputacaoSet()
	for _, id := range ids {
		if err := s.Adicionar(imp(id, engine.Media)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Congelar(); err != nil {
		t.Fatal(err)
	}
	return s
}

func julgar(t *testing.T, s *ImputacaoSet, res map[string]Resultado) *Julgamento {
	t.Helper()
	j, err := NovoJulgamento([]string{"acusado"}, s)
	if err != nil {
		t.Fatal(err)
	}
	for id, r := range res {
		if err := j.Registrar(ResultadoImputacao{id, r, true}); err != nil {
			t.Fatal(err)
		}
	}
	return j
}

// Regra: PADS-ESC-001
func TestConjuntoCongeladoAuditavel(t *testing.T) {
	s := NovoImputacaoSet()
	if err := s.Congelar(); !errors.Is(err, ErrVazio) {
		t.Fatalf("congelar vazio: %v", err)
	}
	_ = s.Adicionar(imp("IMP-1", engine.Grave))
	if err := s.Adicionar(imp("IMP-1", engine.Leve)); !errors.Is(err, ErrIDRepetido) {
		t.Fatal("ID repetido aceito")
	}
	if err := s.Adicionar(Imputacao{}); !errors.Is(err, ErrIDInvalido) {
		t.Fatal("imputação sem ID aceita")
	}
	_ = s.Congelar()
	if err := s.Adicionar(imp("IMP-2", engine.Leve)); !errors.Is(err, ErrCongelado) {
		t.Fatal("adição com conjunto congelado")
	}
	if err := s.Remover("IMP-1"); !errors.Is(err, ErrCongelado) {
		t.Fatal("remoção com conjunto congelado")
	}
	if _, err := NovoJulgamento([]string{"a"}, NovoImputacaoSet()); !errors.Is(err, ErrNaoCongelado) {
		t.Fatal("julgamento com conjunto aberto")
	}
	j, _ := NovoJulgamento([]string{"a"}, s)
	if err := s.Reabrir(""); !errors.Is(err, ErrMotivo) {
		t.Fatal("reabertura sem motivo")
	}
	_ = s.Reabrir("correção da extração")
	_ = s.Adicionar(imp("IMP-2", engine.Leve))
	_ = s.Congelar()
	if err := j.Registrar(ResultadoImputacao{"IMP-1", Procedente, true}); !errors.Is(err, ErrConjuntoAlterado) {
		t.Fatal("julgamento da versão 1 aceitou resultado depois da versão 2")
	}
	if s.Versao() != 2 || len(s.Auditoria()) != 6 {
		t.Fatalf("auditoria: versão %d, %+v", s.Versao(), s.Auditoria())
	}
	// Cópias: alterar a devolvida não muda o conjunto.
	ims := s.Imputacoes()
	ims[0].Texto = "adulterado"
	if x, _ := s.Imputacao("IMP-1"); x.Texto == "adulterado" {
		t.Fatal("conjunto exposto por referência")
	}
}

// Regra: PADS-ESC-002, PADS-ESC-004, PADS-ESC-007
func TestResultadoSoParaImputacaoDoMemorando(t *testing.T) {
	j := julgar(t, conjunto(t, "IMP-1"), nil)
	// Conduta diversa achada na instrução (PADS-E01/E03/E05): não há como julgá-la.
	if err := j.Registrar(ResultadoImputacao{"IMP-OUTRA", Procedente, true}); !errors.Is(err, ErrForaDoEscopo) {
		t.Fatalf("conduta fora do memorando aceita: %v", err)
	}
	if err := j.Registrar(ResultadoImputacao{"IMP-1", "ABSOLVIDO", true}); err == nil {
		t.Fatal("resultado fora do domínio aceito")
	}
}

// Regra: PADS-ESC-003
func TestResultadoGlobal(t *testing.T) {
	casos := []struct {
		nome string
		ids  []string
		res  map[string]Resultado
		g    Global
	}{
		{"PADS-E01 improcedência", []string{"A"}, map[string]Resultado{"A": Improcedente}, Improcedencia},
		{"PADS-E02 parcial", []string{"A", "B"}, map[string]Resultado{"A": Procedente, "B": Improcedente}, ProcedenciaParcial},
		{"PADS-E03 improcedência total", []string{"A", "B"}, map[string]Resultado{"A": Improcedente, "B": Improcedente}, Improcedencia},
		{"PADS-E04 ambas", []string{"A", "B"}, map[string]Resultado{"A": Procedente, "B": Procedente}, Procedencia},
	}
	for _, c := range casos {
		g, err := julgar(t, conjunto(t, c.ids...), c.res).ResultadoGlobal()
		if err != nil || g != c.g {
			t.Errorf("%s: %s, %v", c.nome, g, err)
		}
	}
	j := julgar(t, conjunto(t, "A", "B"), map[string]Resultado{"A": Procedente})
	if _, err := j.ResultadoGlobal(); !errors.Is(err, ErrPendente) {
		t.Fatal("global com imputação sem resultado")
	}
	_ = j.Registrar(ResultadoImputacao{"B", Improcedente, false})
	if _, err := j.ResultadoGlobal(); !errors.Is(err, ErrPendente) {
		t.Fatal("resultado não confirmado pelo usuário conta")
	}
	if p := j.Procedentes(nil); len(p) != 1 || p[0].ID != "A" {
		t.Fatalf("procedentes: %+v", p)
	}
}

// Cobre: PADS-1AC-01
// Regra: PADS-ESC-006
func TestUmAcusado(t *testing.T) {
	s := conjunto(t, "A")
	for _, acs := range [][]string{nil, {"x", "y"}, {" "}} {
		var e ErrAcusadoUnico
		if _, err := NovoJulgamento(acs, s); !errors.As(err, &e) {
			t.Fatalf("%v: %v", acs, err)
		}
	}
}

// Regra: PADS-CNX-002, PADS-CMP-004, PADS-CMP-006
func TestInvariantesDoCalculo(t *testing.T) {
	j := julgar(t, conjunto(t, "A", "B"), map[string]Resultado{"A": Procedente, "B": Improcedente})
	ok := []engine.SancaoAplicada{{ImputacaoID: "A", Natureza: engine.Media, Sancao: engine.Detencao}}
	if vs := j.VerificarCalculo(ok, engine.ResultadoPontuacao{Descontos: []engine.DescontoAplicado{{ImputacaoID: "A", Sancao: engine.Detencao, Pontos: 2}}}); len(vs) != 0 {
		t.Fatalf("cálculo válido reprovado: %+v", vs)
	}
	ruim := []engine.SancaoAplicada{
		{ImputacaoID: "B", Sancao: engine.Detencao}, // improcedente
		{ImputacaoID: "X", Sancao: engine.Detencao}, // fora do memorando
		{ImputacaoID: "A", Sancao: engine.Detencao},
		{ImputacaoID: "A", Sancao: engine.Repreensao}, // dupla
	}
	vs := j.VerificarCalculo(ruim, engine.ResultadoPontuacao{
		Regressao: engine.RegressaoBomOtimo,
		Descontos: []engine.DescontoAplicado{{ImputacaoID: "C", Pontos: 1}},
	})
	esperadas := map[string]bool{"PADS-ESC-002": false, "PADS-ESC-003": false, "PADS-CNX-002": false, "PADS-CMP-006": false, "PADS-CMP-004": false}
	for _, v := range vs {
		esperadas[v.Regra] = true
	}
	for r, achou := range esperadas {
		if !achou {
			t.Errorf("violação %s não detectada: %+v", r, vs)
		}
	}
}

// O cálculo real do motor, alimentado pelo julgamento, respeita as invariantes.
// Regra: PADS-CMP-006
func TestMotorRespeitaInvariantes(t *testing.T) {
	s := NovoImputacaoSet()
	_ = s.Adicionar(imp("A", engine.Grave))
	b := imp("B", engine.Grave)
	b.Fato = civil.Instante{Data: civil.D(1, 3, 2026)}
	_ = s.Adicionar(b)
	_ = s.Congelar()
	j := julgar(t, s, map[string]Resultado{"A": Procedente, "B": Procedente})
	conx := engine.AplicarConexao(j.Procedentes(nil), nil)
	comp, pts := engine.Otimo, 35
	fatos := []engine.Transgressao{}
	for _, p := range conx.Principais {
		fatos = append(fatos, engine.Transgressao{ID: p.ID, Natureza: p.Natureza, Data: p.Fato.Data, Origem: engine.Atual})
	}
	r := engine.Pontuacao(engine.EntradaPontuacao{Comportamento: &comp, Pontos: &pts, Sancoes: conx.Sancoes, Fatos: fatos})
	if r.Regressao == engine.SemRegressao {
		t.Fatalf("duas graves em menos de um ano deveriam regredir: %+v", r)
	}
	if vs := j.VerificarCalculo(conx.Sancoes, r); len(vs) != 0 {
		t.Fatalf("%+v", vs)
	}
}

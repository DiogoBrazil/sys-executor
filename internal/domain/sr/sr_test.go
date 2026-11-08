package sr

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func objeto(t *testing.T) *ObjetoSR {
	t.Helper()
	o, err := NovoObjeto("Portaria nº X",
		[]Fato{{ID: "F1"}, {ID: "F2"}},
		[]Pessoa{{"M1", Militar}, {"M2", Militar}, {"M3", Militar}, {"M9", Militar}, {"C1", Civil}},
		[]string{"M1", "M2", "M3"})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

var enq = []Enquadramento{{Tipo: Transgressao, Diploma: "RDPM", Artigo: "art. 15", LegalPassageID: "LP-0001"}}

func temViolacao(vs []Violacao, regra string) bool {
	for _, v := range vs {
		if v.Regra == regra {
			return true
		}
	}
	return false
}

// Cobre: SR01
// Regra: SR-EST-005
func TestSindicadosFormaisEAutorizados(t *testing.T) {
	o := objeto(t)
	if got := o.Sindicados(); len(got) != 3 {
		t.Fatalf("três sindicados da portaria: %v", got)
	}
	a, _ := o.ProporMilitarNovo("M9", "participação indicada no depoimento", []string{"EV-0003"})
	_ = o.Decidir(a.ID, true)
	if got := o.Sindicados(); len(got) != 4 || got[3] != "M9" {
		t.Fatalf("formais + autorizados: %v", got)
	}
	if _, err := NovoObjeto("p", nil, []Pessoa{{"C1", Civil}}, []string{"C1"}); !errors.Is(err, ErrCivilNaoSujeito) {
		t.Fatal("civil aceito como sindicado da portaria")
	}
	if _, err := NovoObjeto("p", nil, []Pessoa{{"M1", Militar}}, []string{"M1", "M1"}); !errors.Is(err, ErrIDRepetido) {
		t.Fatal("sindicado repetido")
	}
}

// Cobre: SR19, SR35
// Regra: SR-ESC-001
func TestFatoNovoSoComAutorizacao(t *testing.T) {
	o := objeto(t)
	a, err := o.ProporFatoNovo(Fato{ID: "F3", Descricao: "outro fato"}, "resumo com fontes", []string{"EV-0010"})
	if err != nil || a.Estado != Pendente {
		t.Fatalf("%+v %v", a, err)
	}
	c := []ConclusaoSR{{FatoID: "F3", PessoaID: "M1", Resultado: Ausencia}}
	if !temViolacao(o.ValidarConclusoes(c), "SR-ESC-001") {
		t.Fatal("SR19: pendente não entra na IV")
	}
	_ = o.Decidir(a.ID, false)
	if !temViolacao(o.ValidarConclusoes(c), "SR-ESC-001") || len(o.FatosAdmitidos()) != 2 {
		t.Fatal("SR35: recusado fica ausente")
	}
	if err := o.Decidir(a.ID, true); !errors.Is(err, ErrJaDecidida) {
		t.Fatal("decisão refeita")
	}
	b, _ := o.ProporFatoNovo(Fato{ID: "F4"}, "r", nil)
	_ = o.Decidir(b.ID, true)
	if vs := o.ValidarConclusoes([]ConclusaoSR{{FatoID: "F4", PessoaID: "M1", Resultado: Ausencia}}); len(vs) != 0 {
		t.Fatalf("autorizado entra: %+v", vs)
	}
	if len(o.FatosOriginais()) != 2 {
		t.Fatal("a portaria não muda")
	}
}

// Cobre: SR20
// Regra: SR-ESC-002
func TestMilitarNovoPerguntaAntesPortariaIntocada(t *testing.T) {
	o := objeto(t)
	c := []ConclusaoSR{{FatoID: "F1", PessoaID: "M9", Resultado: Inconclusivo}}
	if !temViolacao(o.ValidarConclusoes(c), "SR-ESC-002") {
		t.Fatal("militar não sindicado como sujeito")
	}
	a, err := o.ProporMilitarNovo("M9", "militar de outra OPM", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !temViolacao(o.ValidarConclusoes(c), "SR-ESC-002") {
		t.Fatal("pendente ainda não é sujeito")
	}
	_ = o.Decidir(a.ID, true)
	if vs := o.ValidarConclusoes(c); len(vs) != 0 {
		t.Fatalf("%+v", vs)
	}
	if len(o.SindicadosOriginais()) != 3 {
		t.Fatal("portaria alterada")
	}
	if _, err := o.ProporMilitarNovo("M1", "", nil); !errors.Is(err, ErrJaNaPortaria) {
		t.Fatal("militar da portaria proposto como novo")
	}
	if err := o.Decidir("AUT-99", true); !errors.Is(err, ErrSemAutorizacao) {
		t.Fatal("autorização inexistente")
	}
}

// Cobre: SR21
// Regra: SR-ESC-003
func TestCivilNuncaSujeito(t *testing.T) {
	o := objeto(t)
	if _, err := o.ProporMilitarNovo("C1", "", nil); !errors.Is(err, ErrCivilNaoSujeito) {
		t.Fatal("civil proposto como sindicado")
	}
	vs := o.ValidarConclusoes([]ConclusaoSR{{FatoID: "F1", PessoaID: "C1", Resultado: Indicios, Enquadramentos: enq}})
	if !temViolacao(vs, "SR-ESC-003") {
		t.Fatalf("validador deveria bloquear civil na IV: %+v", vs)
	}
}

// Cobre: SR12
// Regra: SR-SOL-003, SR-SOL-004, SR-SOL-005
func TestConclusaoPorFatoESindicado(t *testing.T) {
	o := objeto(t)
	ok := []ConclusaoSR{
		{FatoID: "F1", PessoaID: "M1", Resultado: Indicios, Enquadramentos: enq},
		{FatoID: "F2", PessoaID: "M1", Resultado: Ausencia},
	}
	if vs := o.ValidarConclusoes(ok); len(vs) != 0 {
		t.Fatalf("SR12: %+v", vs)
	}
	ruins := map[string][]ConclusaoSR{
		"SR-SOL-003": {ok[0], ok[0]},
		"SR-SOL-004": {{FatoID: "F1", PessoaID: "M2", Resultado: "CULPADO"}, {FatoID: "F2", PessoaID: "M2", Resultado: Ausencia, Enquadramentos: enq}},
		"SR-SOL-005": {{FatoID: "F1", PessoaID: "M3", Resultado: Indicios}, {FatoID: "F2", PessoaID: "M3", Resultado: Indicios, Enquadramentos: []Enquadramento{{Tipo: "OUTRO"}}}},
	}
	for regra, cs := range ruins {
		if !temViolacao(o.ValidarConclusoes(cs), regra) {
			t.Errorf("%s não detectada", regra)
		}
	}
}

// SR ⇒ nenhuma sanção, pontuação ou reincidência: a estrutura da conclusão
// não tem onde guardá-las, nem depende do motor do PADS.
// Regra: TRV-ESC-004
func TestSRSemSancaoPontuacaoReincidencia(t *testing.T) {
	proibidos := []string{"sanc", "punic", "pont", "reincid", "comportamento", "dias", "natureza"}
	for _, tipo := range []reflect.Type{reflect.TypeOf(ConclusaoSR{}), reflect.TypeOf(Enquadramento{}), reflect.TypeOf(ObjetoSR{})} {
		for i := 0; i < tipo.NumField(); i++ {
			f := tipo.Field(i)
			nome := strings.ToLower(f.Name)
			for _, p := range proibidos {
				if strings.Contains(nome, p) {
					t.Errorf("%s.%s: SR não tem %s", tipo.Name(), f.Name, p)
				}
			}
			if strings.Contains(f.Type.PkgPath(), "rules/engine") {
				t.Errorf("%s.%s usa tipo do motor do PADS", tipo.Name(), f.Name)
			}
		}
	}
}

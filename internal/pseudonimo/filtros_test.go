package pseudonimo

import (
	"strings"
	"testing"
)

// Todos os nomes abaixo são fictícios.

func contem(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// Pedaço de nome conhecido (sobrenomes, prenome + último sobrenome, palavra
// solta com maiúscula) é ocultado como variante da mesma pessoa. Antes, o
// pedaço não era trocado e também não ia à revisão.
// Regra: TRV-SEG-013
func TestPedacoDeNomeConhecidoEOcultado(t *testing.T) {
	m := NovoMapa()
	m.CadastrarPessoa("Fulano Beltrano da Costa", nil)
	txt := "Disse que Beltrano da Costa chegou; Fulano Costa saiu; o Costa voltou; a costa do rio."
	oc := m.Ocultar(txt)
	for _, s := range []string{"Beltrano", "Fulano", " Costa"} {
		if strings.Contains(oc, s) {
			t.Fatalf("%q escapou: %q", s, oc)
		}
	}
	if !strings.Contains(oc, "a costa do rio") {
		t.Fatalf("palavra comum em minúsculas não é nome: %q", oc)
	}
	if strings.Contains(oc, "[PESSOA-02") {
		t.Fatalf("os pedaços são da mesma pessoa: %q", oc)
	}
	if s := m.Suspeitos(oc); len(s) != 0 {
		t.Fatalf("nada a revisar: %q", s)
	}
	if de, err := m.Restaurar(oc); err != nil || de != txt {
		t.Fatalf("ida e volta: %v %q", err, de)
	}
	if r := m.Resumir(); r.Pessoas != 1 {
		t.Fatalf("uma pessoa: %+v", r)
	}
}

// Pedaço comum a duas pessoas é ocultado sem ser atribuído a nenhuma, e
// não conta como pessoa a mais no resumo.
// Regra: TRV-SEG-013
func TestPedacoComumADuasPessoas(t *testing.T) {
	m := NovoMapa()
	m.CadastrarPessoa("Ana Clara Inventado Moraes", nil)
	m.CadastrarPessoa("Bruno Ficticio Moraes", nil)
	oc := m.Ocultar("O Moraes estava lá.")
	if strings.Contains(oc, "Moraes") || strings.Contains(oc, "[PESSOA-01") || strings.Contains(oc, "[PESSOA-02") {
		t.Fatalf("%q", oc)
	}
	if r := m.Resumir(); r.Pessoas != 2 {
		t.Fatalf("duas pessoas: %+v", r)
	}
}

// Palavra solta derivada não bloqueia as instruções nem o schema (podem ter
// a mesma palavra comum), mas o nome inteiro continua bloqueando.
// Regra: TRV-SEG-013
func TestPalavraSoltaNaoBloqueiaInstrucoes(t *testing.T) {
	m := NovoMapa()
	m.CadastrarPessoa("Fulano Prova Inventado", nil)
	m.Ocultar("texto")
	if err := m.conferir("Analise a Prova dos autos.", "nada", nil, []byte(`{"Prova":"x"}`)); err != nil {
		t.Fatalf("palavra comum nas instruções: %v", err)
	}
	if err := m.conferir("Fulano Inventado", "nada", nil, nil); err == nil {
		t.Fatal("pedaço de duas palavras nas instruções tem de bloquear")
	}
}

// Parentes da qualificação viram âncora, com a pontuação que o OCR deixar;
// "pai" no meio de um relato, sem pontuação, não.
// Regra: TRV-SEG-011
func TestParentesDaQualificacao(t *testing.T) {
	txt := "Pai: Josué Inventado Lopes\nMãe. Rita Ficticia Lopes\nCÔNJUGE; Carla Exemplo Duarte\n" +
		"filiação - Pedro Teste Rocha e Marta Teste Rocha\nCasado com Lúcia Modelo Prado.\n" +
		"Mãe: não declarada. Disse que o pai Renato Simulado Alves viajou."
	var nomes []string
	for _, a := range detectar(txt) {
		if a.tipo == Pessoa {
			nomes = append(nomes, a.forma)
		}
	}
	for _, s := range []string{"Josué Inventado Lopes", "Rita Ficticia Lopes", "Carla Exemplo Duarte",
		"Pedro Teste Rocha", "Marta Teste Rocha", "Lúcia Modelo Prado"} {
		if !contem(nomes, s) {
			t.Errorf("faltou %q em %q", s, nomes)
		}
	}
	for _, s := range nomes {
		if strings.Contains(s, "Renato") || strings.Contains(strings.ToLower(s), "declarada") || strings.Contains(s, " e ") {
			t.Errorf("não devia ser âncora: %q", s)
		}
	}
}

// Os falsos positivos do teste real de 10/11/2026 (só a forma, com termos
// que não são dados pessoais) saem da revisão; nomes sem âncora continuam.
// Regra: TRV-SEG-013
func TestRevisaoSemFalsosPositivosConhecidos(t *testing.T) {
	txt := "Cometeu Transgressão da Disciplina conforme Autos de Infração lavrados.\n" +
		"Lotado na UNISP de Cujubim, natural de Campo Grande.\n" +
		"Escolaridade: Ensino Médio, estado civil: União Estável, religião: Igreja Cristã.\n" +
		"Grau de instrução: Ensino Fundamental Incompleto.\n" +
		"Conduzia uma Honda CG vermelha; viu SO CNE AOC e AAA MULLLSL no papel.\n" +
		"na praça, encontrou Gilberto Ficticio Ramos e Sandra Inventada Neves.\n" +
		"depois chamou Renata Maria e Evelly para depor; também ANA DIAS e Otto Schmidt.\n" +
		"Seguiu para Rolim de Moura com Paulo Rolim de Moura."
	s := NovoMapa().Suspeitos(txt)
	for _, fp := range []string{"Transgressão da Disciplina", "Autos de Infração", "UNISP de Cujubim", "Cujubim",
		"Campo Grande", "Ensino Médio", "União Estável", "Igreja Cristã", "Ensino Fundamental Incompleto",
		"Honda CG", "AAA MULLLSL", "Rolim de Moura"} {
		if contem(s, fp) {
			t.Errorf("falso positivo %q em %q", fp, s)
		}
	}
	for _, n := range []string{"Gilberto Ficticio Ramos", "Sandra Inventada Neves", "Renata Maria e Evelly",
		"ANA DIAS", "Otto Schmidt", "Paulo Rolim de Moura"} {
		if !contem(s, n) {
			t.Errorf("faltou o nome %q em %q", n, s)
		}
	}
}

// "Não é dado pessoal" vale só para a frase exata.
// Regra: TRV-SEG-013
func TestLiberarSoAFraseExata(t *testing.T) {
	m := NovoMapa()
	m.Liberar("Avenida Gilberto Ficticio Ramos")
	s := m.Suspeitos("Mora na Avenida Gilberto Ficticio Ramos.\nFalou com Gilberto Ficticio Ramos ontem.")
	if !contem(s, "Gilberto Ficticio Ramos") {
		t.Fatalf("o nome sozinho continua a revisar: %q", s)
	}
}

func TestRuido(t *testing.T) {
	for w, esp := range map[string]bool{"MULLLSL": true, "AAA": true, "BPM": true, "CNE": false, "AOC": false,
		"ANA": false, "Schmidt": false, "Hirschfeld": false, "Sá": false} {
		if ruido(w) != esp {
			t.Errorf("ruido(%q) = %v", w, !esp)
		}
	}
}

func TestDividirE(t *testing.T) {
	for in, esp := range map[string]string{
		"Fulano Silva e Beltrano Costa":  "Fulano Silva|Beltrano Costa",
		"Antônio Carlos Ribeiro e Silva": "Antônio Carlos Ribeiro e Silva",
		"Renata Maria e Evelly":          "Renata Maria e Evelly",
		"Maria e José Silva":             "Maria e José Silva",
	} {
		if got := strings.Join(dividirE(in), "|"); got != esp {
			t.Errorf("dividirE(%q) = %q", in, got)
		}
	}
}

// Título em maiúsculas com conector minúsculo não é nome.
// Regra: TRV-SEG-013
func TestTituloComConectorMinusculo(t *testing.T) {
	s := NovoMapa().Suspeitos("NECESSIDADE e PROPORCIONALIDADE DA REAÇÃO\nviu FULANO INVENTADO no local")
	if contem(s, "NECESSIDADE e PROPORCIONALIDADE") || len(s) != 1 || s[0] != "FULANO INVENTADO" {
		t.Fatalf("%q", s)
	}
}

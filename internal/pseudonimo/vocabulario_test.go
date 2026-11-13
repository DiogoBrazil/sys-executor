package pseudonimo

import (
	"strings"
	"testing"
)

// Os falsos "possíveis nomes" do teste da SR (12/11/2026) deixam de ser
// perguntados; nomes de pessoa continuam, mesmo com palavras comuns no meio.
// Nomes próprios que não são de pessoa e não estão em nenhuma lista
// ("Eldorado") continuam a ser perguntados: é o lado seguro.
// Regra: TRV-SEG-009
func TestSuspeitosIgnoramPalavrasComuns(t *testing.T) {
	texto := "Relatório do núcleo de inteligência e da embalagem apreendida na câmera.\n" +
		"Consta o Manual de Inquérito e o Auto de Prisão em Flagrante de Delito, com o Adendo GMAIL e a reunião pelo Google Meet.\n" +
		"O veículo Ford Ka estava no Jardim Eldorado, local de trabalho; Local de Trabalho, Orientação Sexual, Doença Grave, Tipo Natureza.\n" +
		"LIBERDADE PROVISÓRIA foi pedida pela Promotoria do Controle Externo e pela Varas de Garantias.\n" +
		"Ouvidos Rudinei Joao Bessegatto Pogere, Felipe Moura e José Pinto Rocha; também Delclécio Linhares.\n" +
		"Depuseram ainda Maria da Conceição e Ana Trindade, que viram o Fiat Uno e o Celular Samsung na Investigação Preliminar.\n" +
		"Contato: rudinei.pogere@exemplo.invalido e felipe_moura."
	ss := NovoMapa().Suspeitos(texto)
	j := strings.Join(ss, " | ")
	for _, comum := range []string{"Manual de Inquérito", "Auto de Prisão", "Flagrante de Delito", "Adendo GMAIL", "Google Meet",
		"Ford Ka", "Local de Trabalho", "Orientação Sexual", "Doença Grave", "Tipo Natureza",
		"LIBERDADE PROVISÓRIA", "Promotoria do Controle Externo", "Varas de Garantias", "Fiat Uno", "Celular Samsung", "Investigação Preliminar"} {
		if strings.Contains(j, comum) {
			t.Errorf("palavras comuns tomadas por nome: %q em %v", comum, ss)
		}
	}
	for _, nome := range []string{"Rudinei Joao Bessegatto Pogere", "Felipe Moura", "José Pinto Rocha", "Delclécio Linhares",
		"Maria da Conceição", "Ana Trindade"} {
		if !strings.Contains(j, nome) {
			t.Errorf("nome de pessoa deixou de ser perguntado: %q (suspeitos: %v)", nome, ss)
		}
	}
}

// Sobrenome que também é palavra nunca entra no vocabulário.
func TestVocabularioSemSobrenomes(t *testing.T) {
	v := vocabulario()
	if len(v) < 5000 {
		t.Fatalf("vocabulário pequeno demais: %d", len(v))
	}
	for _, w := range []string{"costa", "rocha", "pinto", "campos", "silva", "clara", "rosa", "maria", "jose", "joao"} {
		if v[w] {
			t.Errorf("%q no vocabulário", w)
		}
	}
}

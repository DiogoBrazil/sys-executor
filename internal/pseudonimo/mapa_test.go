package pseudonimo

import (
	"strings"
	"testing"
)

// Envolvidos do cadastro de peças: nome e outras grafias são a mesma pessoa
// e são ocultados sem âncora (posto, papel) no texto.
// Regra: TRV-SEG-011
func TestCadastrarPessoa(t *testing.T) {
	m := NovoMapa()
	m.CadastrarPessoa("  Diego   Suposto ", []string{"Dieguinho", "x"})
	m.CadastrarPessoa("Ana Fictícia", nil)
	o := m.Ocultar("Ouvido, DIEGO SUPOSTO (o Dieguinho) disse que ana ficticia estava lá.")
	for _, nome := range []string{"Diego", "DIEGO", "Dieguinho", "ana ficticia"} {
		if strings.Contains(o, nome) {
			t.Fatalf("%q ficou em claro: %s", nome, o)
		}
	}
	if !strings.Contains(o, "[PESSOA-01") || !strings.Contains(o, "[PESSOA-02") || strings.Contains(o, "[PESSOA-03") {
		t.Fatalf("grafias da mesma pessoa viraram pessoas diferentes: %s", o)
	}
	r, err := m.Restaurar(o)
	if err != nil || !strings.Contains(r, "Dieguinho") || !strings.Contains(r, "DIEGO SUPOSTO") {
		t.Fatalf("restauração: %q %v", r, err)
	}
}

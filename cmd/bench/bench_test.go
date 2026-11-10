package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestFormasProcuradas(t *testing.T) {
	got := formasProcuradas([]string{"ANTÔNIO FICTÍCIO DA SILVA"}, []string{"100000001"})
	want := []string{"100000001", "antonio ficticio", "antonio ficticio da silva", "ficticio silva"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("formas = %v, quer %v", got, want)
	}
}

func TestContemComLimiteDePalavra(t *testing.T) {
	casos := []struct {
		texto, forma string
		quer         bool
	}{
		{normal("o SD PM Carlos   Inventado declarou"), "carlos inventado", true},
		{normal("carlos inventadora"), "carlos inventado", false},
		{normal("matrícula 1000000012"), "100000001", false},
		{normal("[PESSOA-01] declarou"), "carlos inventado", false},
	}
	for _, c := range casos {
		if got := contem(c.texto, c.forma); got != c.quer {
			t.Errorf("contem(%q, %q) = %v, quer %v", c.texto, c.forma, got, c.quer)
		}
	}
}

// O corpo é conferido depois de decodificar o JSON: formas escapadas
// (quebra de linha, unicode) também contam.
func TestConferirCorpoJSONEscapado(t *testing.T) {
	v := novoVazamento()
	v.iniciarCaso("c", []string{"CARLOS INVENTADO"}, nil)
	v.conferir([]byte(`{"contents":[{"parts":[{"text":"o CB PM CARLOS\nINVENTAD\u004f declarou"}]}]}`))
	v.conferir([]byte(`{"input":"o [PESSOA-01] declarou"}`))
	if v.pedidos != 2 || v.comNome != 1 {
		t.Fatalf("pedidos %d, com nome %d", v.pedidos, v.comNome)
	}
}

// Regra: métrica de vazamento do plano D-11 §4.9 (nenhum nome dos casos
// sintéticos em nenhum pedido), nas três entradas, com a pseudonimização e a
// barreira do aplicativo. Sem rede (provedor falso, que reenvia com o
// histórico restaurado).
func TestVazamentoZeroNasTresEntradas(t *testing.T) {
	cfg := config{provedor: provFalso, modelo: "eco", entradas: []string{entMemoria, entMonolitica, entPecas}, pseudonimizar: true}
	rod, err := rodada(context.Background(), cfg, "", func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if rod.vaz.pedidos == 0 {
		t.Fatal("nenhum pedido conferido")
	}
	if rod.vaz.comNome != 0 {
		t.Fatalf("%d pedido(s) com nome dos casos: %v", rod.vaz.comNome, rod.vaz.listaAchados())
	}
	for _, r := range rod.rs {
		if r.prep.pecasOK != r.prep.pecasEsp {
			t.Errorf("%s (%s): peças %d/%d: %v", r.caso, r.entrada, r.prep.pecasOK, r.prep.pecasEsp, r.prep.divergem)
		}
		if len(r.erros) > 0 {
			t.Errorf("%s (%s): erros %v", r.caso, r.entrada, r.erros)
		}
	}
}

// A métrica precisa enxergar o vazamento quando ele existe: sem a
// pseudonimização, os nomes saem em claro.
func TestVazamentoDetectadoSemPseudonimizacao(t *testing.T) {
	cfg := config{provedor: provFalso, modelo: "eco", entradas: []string{entMemoria}, pseudonimizar: false}
	rod, err := rodada(context.Background(), cfg, "pads-02", func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if rod.vaz.comNome == 0 {
		t.Fatal("sem pseudonimização a métrica deveria apontar os nomes")
	}
}

func TestTetoDeChamadas(t *testing.T) {
	cfg := config{provedor: provFalso, modelo: "eco", entradas: []string{entMemoria}, pseudonimizar: true, teto: 3}
	rod, err := rodada(context.Background(), cfg, "pads-01-falta-servico", func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if rod.med.chamadas != 3 || rod.med.puladas == 0 {
		t.Fatalf("chamadas %d, puladas %d", rod.med.chamadas, rod.med.puladas)
	}
	if !rod.rs[0].teto || len(rod.rs[0].erros) != 0 {
		t.Fatalf("caso deveria parar no teto sem erro: %+v", rod.rs[0].erros)
	}
}

func TestLojaDevPermissoes(t *testing.T) {
	l := lojaDev{caminho: filepath.Join(t.TempDir(), "sub", "chatgpt-dev.json")}
	if l.Existe() {
		t.Fatal("não deveria existir")
	}
	if _, err := l.Ler(); err == nil {
		t.Fatal("ausente deveria dar erro")
	}
	if err := l.Guardar([]byte(`{"versao":1}`)); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" { // no Windows, as permissões POSIX não se aplicam
		st, err := os.Stat(l.caminho)
		if err != nil || st.Mode().Perm() != 0o600 {
			t.Fatalf("permissão %v (%v)", st.Mode().Perm(), err)
		}
		if d, _ := os.Stat(filepath.Dir(l.caminho)); d.Mode().Perm() != 0o700 {
			t.Fatalf("pasta %v", d.Mode().Perm())
		}
	}
	if b, err := l.Ler(); err != nil || string(b) != `{"versao":1}` {
		t.Fatalf("ler: %q %v", b, err)
	}
	if err := l.Remover(); err != nil || l.Existe() {
		t.Fatal("remover")
	}
}

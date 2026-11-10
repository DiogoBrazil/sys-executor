package analise

import (
	"context"
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/inference"
)

// O schema da chamada é conferido no Go, inclusive o que o provedor não
// impõe (a Gemini API recebe o schema sem minItems/maxItems).
func TestConferirSchema(t *testing.T) {
	sch := objeto(map[string]any{
		"id":    enum("IMP-1", "IMP-2"),
		"itens": lista(objeto(map[string]any{"t": str(5), "n": inteiro(1, 3)}), 1, 2),
		"nota":  str(0),
	})
	bom := `{"id":"IMP-1","itens":[{"t":"abc","n":2}],"nota":"livre"}`
	if err := ConferirSchema(sch, []byte(bom)); err != nil {
		t.Fatal(err)
	}
	casos := map[string]string{
		`{"id":"IMP-9","itens":[{"t":"abc","n":2}],"nota":""}`:                               "fora das opções",
		`{"id":"IMP-1","itens":[],"nota":""}`:                                                "mínimo",
		`{"id":"IMP-1","itens":[{"t":"a","n":1},{"t":"b","n":1},{"t":"c","n":1}],"nota":""}`: "máximo 2",
		`{"id":"IMP-1","itens":[{"t":"abcdef","n":1}],"nota":""}`:                            "caracteres, máximo 5",
		`{"id":"IMP-1","itens":[{"t":"a","n":4}],"nota":""}`:                                 "máximo 3",
		`{"id":"IMP-1","itens":[{"t":"a","n":1.5}],"nota":""}`:                               "inteiro",
		`{"id":"IMP-1","itens":[{"t":"a","n":1}]}`:                                           "falta o campo",
		`{"id":"IMP-1","itens":[{"t":"a","n":1}],"nota":"","x":1}`:                           "não existe",
		`[1]`: "objeto",
	}
	for resp, quer := range casos {
		if err := ConferirSchema(sch, []byte(resp)); err == nil || !strings.Contains(err.Error(), quer) {
			t.Errorf("%s: esperava %q, veio %v", resp, quer, err)
		}
	}
}

// Resposta que estoura o limite de itens volta ao modelo com o motivo.
func TestChamarReenviaQuandoSchemaNaoConfere(t *testing.T) {
	m := &motorFalso{ctx: 16384, filas: map[string][]string{"teste": {
		`{"itens":["a","b","c"]}`, `{"itens":["a"]}`,
	}}}
	x := executor(t, m, "PADS")
	ch := Chamada{Etapa: "classificacao", Tarefa: "t", NomeSchema: "teste", Reserva: 100,
		Schema: objeto(map[string]any{"itens": lista(str(10), 0, 2)})}
	if err := x.Chamar(context.Background(), ch, func([]byte) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if len(m.pedidos) != 2 || !strings.Contains(ultimo(m.pedidos[1].Historico).Conteudo, "máximo 2") {
		t.Fatalf("deveria reenviar com o motivo: %d pedidos", len(m.pedidos))
	}
}

func ultimo(hs []inference.Mensagem) inference.Mensagem { return hs[len(hs)-1] }

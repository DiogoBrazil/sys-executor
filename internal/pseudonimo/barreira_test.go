package pseudonimo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
)

// Todos os nomes e números abaixo são fictícios.

// Grafias que só diferem em caixa, acento ou espaços/quebras de linha são
// ocultadas, e cada grafia real volta exata na restauração.
// Regra: TRV-SEG-011
func TestOcultarSemDiferencaDeCaixaNemAcento(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{"Hélio Fictício"})
	txt := "HELIO FICTICIO assinou; hélio fictício depôs; Hélio\nFictício voltou; o sr. Hélio   Fictício saiu."
	oc := m.Ocultar(txt)
	if strings.Contains(strings.ToLower(oc), "fict") || strings.Contains(strings.ToLower(oc), "helio") || strings.Contains(strings.ToLower(oc), "hélio") {
		t.Fatalf("grafia escapou: %q", oc)
	}
	if !strings.Contains(oc, "[PESSOA-01") || strings.Contains(oc, "[PESSOA-02") {
		t.Fatalf("todas as grafias são a mesma pessoa: %q", oc)
	}
	de, err := m.Restaurar(oc)
	if err != nil || de != txt {
		t.Fatalf("ida e volta: %v\n%q\n%q", err, de, txt)
	}
}

// Nome de uma só palavra em minúsculas não é trocado ("machado", o objeto),
// mas com inicial maiúscula é; nome composto é trocado em qualquer caixa.
// Regra: TRV-SEG-011
func TestPalavraUnicaSoComMaiuscula(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{"Machado", "Rosa Branca"})
	oc := m.Ocultar("Machado agrediu com um machado; rosa branca viu.")
	if !strings.HasPrefix(oc, "[PESSOA-01] agrediu com um machado;") || strings.Contains(oc, "rosa branca") {
		t.Fatalf("%q", oc)
	}
}

// Forma escapada em JSON (quebra de linha dentro do nome, nos dados
// estruturados) é ocultada e restaurada como JSON válido.
// Regra: TRV-SEG-011
func TestFormaEscapadaEmJSON(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{"Carlos Inventado"})
	dados, _ := json.Marshal(map[string]string{"acusado": "CARLOS\nINVENTADO"})
	oc := m.Ocultar(string(dados))
	if strings.Contains(oc, "CARLOS") || strings.Contains(oc, "INVENTADO") {
		t.Fatalf("forma escapada escapou: %s", oc)
	}
	var v map[string]string
	if err := json.Unmarshal([]byte(oc), &v); err != nil {
		t.Fatalf("JSON quebrado: %s", oc)
	}
	b, err := m.RestaurarJSON([]byte(`{"quem":"` + v["acusado"] + `"}`))
	var r map[string]string
	if err != nil || json.Unmarshal(b, &r) != nil || r["quem"] != "CARLOS\nINVENTADO" {
		t.Fatalf("restauração: %s %v", b, err)
	}
}

// A forma mais longa recusada pelo limite de palavra não impede a mais curta
// na mesma posição ("Ana Reis" em "Ana Reisinho" não; "Ana" sim).
func TestFormaCurtaQuandoALongaFalhaNoLimite(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{"Ana Reis", "Ana"})
	oc := m.Ocultar("Ana Reisinho chegou.")
	if !strings.HasPrefix(oc, "[PESSOA-0") || !strings.Contains(oc, "Reisinho") {
		t.Fatalf("%q", oc)
	}
}

// Pessoas diferentes com nomes parecidos não se fundem.
func TestHomonimosParciaisSaoPessoasDiferentes(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{"João Silva", "João Silveira"})
	oc := m.Ocultar("João Silva e João Silveira")
	if oc != "[PESSOA-01] e [PESSOA-02]" {
		t.Fatalf("%q", oc)
	}
}

// Marcadores deformados de todos os tipos, inclusive em minúsculas, são
// recusados na volta.
func TestMarcadorDeformadoTodosOsTipos(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{"Maria Souza"})
	for _, ruim := range []string{`"[pessoa-01]"`, `"CPF-01 informado"`, `"[EMAIL_01]"`, `"RG-02"`} {
		if _, err := m.RestaurarJSON([]byte(`{"q":` + ruim + `}`)); err == nil {
			t.Errorf("%s deveria ser recusado", ruim)
		}
	}
	for _, ok := range []string{`"CPF 529.982.247-25 nos autos"`, `"matrícula 100012345"`, `"a pessoa 2 disse"`} {
		if _, err := m.RestaurarJSON([]byte(`{"q":` + ok + `}`)); err != nil {
			t.Errorf("%s não é marcador: %v", ok, err)
		}
	}
}

// Barreira (camada 1): CPF/e-mail digitados no pedido são aprendidos e
// ocultados; dado conhecido nas instruções ou no schema bloqueia (IA-014);
// possível nome sem âncora pára para revisão; liberado, segue.
// Regra: TRV-SEG-013
func TestBarreiraCamada1(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{"Hélio Fictício"})
	eco := &motorEco{respostas: []string{`{"ok":true}`}}
	p := Envolver(eco, m)

	// digitado pelo usuário: CPF válido e e-mail (fictícios)
	if _, err := p.Gerar(context.Background(), inference.Pedido{Usuario: `{"descricao":"contato: helio@exemplo.com, CPF 529.982.247-25"}`}); err != nil {
		t.Fatal(err)
	}
	if u := eco.pedidos[0].Usuario; strings.Contains(u, "@exemplo.com") || strings.Contains(u, "529.982.247-25") {
		t.Fatalf("dado digitado foi enviado: %s", u)
	}

	// dado conhecido nas instruções ou no schema: bloqueio, nada enviado
	n := len(eco.pedidos)
	for _, ped := range []inference.Pedido{
		{Sistema: "Regras para HÉLIO FICTÍCIO", Usuario: "x"},
		{Usuario: "x", Schema: json.RawMessage(`{"enum":["Hélio Fictício"]}`)},
	} {
		_, err := p.Gerar(context.Background(), ped)
		var e *erros.Erro
		if !errors.As(err, &e) || e.Codigo != erros.IABloqueioPrivacidade || strings.Contains(err.Error(), "Fictício") {
			t.Fatalf("esperado IA-014 sem o dado na mensagem: %v", err)
		}
	}
	if len(eco.pedidos) != n {
		t.Fatal("nada pode ser enviado depois do bloqueio")
	}

	// possível nome sem âncora: revisão, nada enviado; liberado, segue
	_, err := p.Gerar(context.Background(), inference.Pedido{Usuario: "Na Avenida Jorge Teixeira, viu-se Lucas Imaginário correndo."})
	var rev ErrRevisao
	if !errors.As(err, &rev) || len(rev.Itens) != 2 || strings.Contains(err.Error(), "Lucas") || len(eco.pedidos) != n {
		t.Fatalf("revisão: %v %+v", err, rev)
	}
	m.Liberar("Jorge Teixeira")
	m.AcrescentarNomes([]string{"Lucas Imaginário"})
	if _, err := p.Gerar(context.Background(), inference.Pedido{Usuario: "Na Avenida Jorge Teixeira, viu-se Lucas Imaginário correndo."}); err != nil {
		t.Fatal(err)
	}
	if u := eco.pedidos[len(eco.pedidos)-1].Usuario; strings.Contains(u, "Lucas") || !strings.Contains(u, "Jorge Teixeira") {
		t.Fatalf("após a revisão: %q", u)
	}
}

// Títulos e cabeçalhos em maiúsculas não são suspeitos; nomes com âncora já
// são aprendidos e não pedem revisão.
func TestSuspeitosIgnoramCabecalhos(t *testing.T) {
	m := NovoMapa()
	txt := "POLÍCIA MILITAR DO ESTADO DE RONDÔNIA\nTERMO DE INQUIRIÇÃO DE TESTEMUNHA\nDEFESA PRÉVIA\nO Sd PM Pedro Ficto compareceu."
	m.Aprender(txt)
	if s := m.Suspeitos(m.Ocultar(txt)); len(s) != 0 {
		t.Fatalf("suspeitos indevidos: %q", s)
	}
}

// Barreira (camada 2): os bytes da requisição são conferidos; resíduo
// bloqueia, e a mensagem não contém o dado.
// Regra: TRV-SEG-013
func TestBarreiraCamada2Inspecionar(t *testing.T) {
	m := NovoMapa()
	m.AcrescentarNomes([]string{"Hélio Fictício"})
	limpo := []byte(`{"contents":[{"parts":[{"text":"o [PESSOA-01] faltou"}]}],"systemInstruction":{"parts":[{"text":"` + InstrucaoMarcadores + `"}]}}`)
	if err := m.Inspecionar(limpo); err != nil {
		t.Fatalf("corpo limpo bloqueado: %v", err)
	}
	for _, sujo := range []string{
		`{"contents":[{"parts":[{"text":"o HELIO FICTICIO faltou"}]}]}`,
		`{"input":[{"role":"user","content":"tel (69) 99999-1234"}]}`,
		`{"x":"529.982.247-25"}`,
		`não é json`,
	} {
		err := m.Inspecionar([]byte(sujo))
		var vz ErrVazamento
		if !errors.As(err, &vz) || strings.Contains(err.Error(), "HELIO") || strings.Contains(err.Error(), "99999") {
			t.Errorf("%s: esperado bloqueio sem o dado: %v", sujo, err)
		}
	}
}

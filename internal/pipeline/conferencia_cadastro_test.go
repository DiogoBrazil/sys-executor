package pipeline

import (
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// Nomes fictícios.

func cadastroArt17XI(fatos ...workflow.FatoCadastro) *workflow.Cadastro {
	return &workflow.Cadastro{Tipo: workflow.TipoPADS, AcusadoID: "P1",
		Pessoas: []workflow.PessoaCadastro{{ID: "P1", Nome: "CARLOS INVENTADO DA SILVA", PostoGrad: "CB QPPM"}},
		Fatos:   fatos}
}

// Memorando cita só "art. 17": pareia com o art. 17, XI do cadastro (um só
// candidato), sem o bloco "sem correspondência" e sem repetir o inciso; o
// acusado vem com o posto do cadastro mesmo com a leitura "DASILVA".
// Regra: PADS-ESC-001
func TestCadastroPareiaPeloArtigoQuandoMemorandoNaoCitaInciso(t *testing.T) {
	cad := cadastroArt17XI(workflow.FatoCadastro{ID: "FATO-06", Enquadramentos: []workflow.EnquadramentoCadastro{{DispositivoID: "RDPM-17-XI"}}})
	r := analise.PropostaDelimitacaoPADS{
		Acusados:   []analise.PessoaProposta{{Nome: "Mat. 100000100 CARLOS INVENTADO DASILVA"}},
		Imputacoes: []analise.ImputacaoProposta{{ID: "IMP-1", Diploma: "RDPM", Artigo: 17, Pendencias: []string{analise.PendenciaInciso, analise.PendenciaNatureza}}},
	}
	c := conferirComCadastro(r, cad)
	if len(c.extras) != 0 {
		t.Fatalf("sem bloco extra: %+v", c.extras)
	}
	if c.acusado != "CB QPPM CARLOS INVENTADO DA SILVA" || len(c.alertasAcusado) != 0 {
		t.Fatalf("acusado: %q %q", c.acusado, c.alertasAcusado)
	}
	if !contemTexto(c.suprimir["IMP-1"], analise.PendenciaInciso) {
		t.Fatalf("o inciso do cadastro resolve a pendência: %+v", c.suprimir)
	}
	if len(c.alertas["IMP-1"]) != 1 || !strings.Contains(c.alertas["IMP-1"][0], "inciso XI") {
		t.Fatalf("uma nota só: %q", c.alertas["IMP-1"])
	}
	if !strings.Contains(c.padroes[campoDiploma+"IMP-1"], "RDPM") {
		t.Fatalf("enquadramento preenchido: %+v", c.padroes)
	}
}

// Dois incisos do art. 17 no cadastro: ambíguo, o usuário escolhe (como antes).
// Regra: PADS-ESC-001
func TestCadastroNaoPareiaPeloArtigoComAmbiguidade(t *testing.T) {
	cad := cadastroArt17XI(workflow.FatoCadastro{ID: "FATO-01", Enquadramentos: []workflow.EnquadramentoCadastro{
		{DispositivoID: "RDPM-17-XI"}, {DispositivoID: "RDPM-17-XII"}}})
	r := analise.PropostaDelimitacaoPADS{Imputacoes: []analise.ImputacaoProposta{{ID: "IMP-1", Diploma: "RDPM", Artigo: 17}}}
	c := conferirComCadastro(r, cad)
	if len(c.extras) != 2 || len(c.suprimir["IMP-1"]) != 0 {
		t.Fatalf("ambíguo não pareia: extras=%d suprimir=%v", len(c.extras), c.suprimir)
	}
	if got := rotuloFato(c.extras[0]); got != "fato 1" {
		t.Fatalf("rótulo pela posição: %q", got)
	}
}

func TestMesmaPessoaComPalavrasGrudadas(t *testing.T) {
	if !mesmaPessoa("Carlos Inventado da Silva", "CARLOS INVENTADO DASILVA") {
		t.Fatal("DASILVA = DA SILVA")
	}
	if mesmaPessoa("Carlos Inventado da Silva", "Bruno Ficticio Souza") {
		t.Fatal("pessoas diferentes")
	}
}

// Sem acusado lido no Memorando, o campo vem do cadastro com o posto.
// Regra: PADS-ESC-001
func TestCadastroAcusadoComPostoSemLeitura(t *testing.T) {
	c := conferirComCadastro(analise.PropostaDelimitacaoPADS{}, cadastroArt17XI())
	if c.acusado != "CB QPPM CARLOS INVENTADO DA SILVA" {
		t.Fatalf("%q", c.acusado)
	}
}

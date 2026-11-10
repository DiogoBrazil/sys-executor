package segment

import (
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
)

func autosDeclarados() *autos.Autos {
	return &autos.Autos{Paginas: []autos.Pagina{
		pag(1, "MEMORANDO DISCIPLINAR Nº 1/2026\nAo CB PM Fulano…", "0090000001"),
		pag(2, "continuação do memorando", "0090000001"),
		pag(3, "TERMO DE DECLARAÇÕES\nO declarante disse…", ""),
		pag(4, "Excelentíssimo Senhor, a defesa requer…", ""),
		pag(5, "MANIFESTAÇÃO\nA defesa reitera…", ""),
	}}
}

// Peças declaradas no cadastro: páginas contíguas na ordem declarada,
// função do usuário (Base USUARIO) e conferência do conteúdo, sem troca.
// Cobre: CAD-PEC-02
// Regra: TRV-ING-001
func TestDeDeclaracaoEVerificar(t *testing.T) {
	a := autosDeclarados()
	ds := []Declarada{
		{Cadastro: "PEC-01", Funcao: MemorandoDisciplinar, Titulo: "Memorando Disciplinar", Paginas: 2},
		{Cadastro: "PEC-05", Funcao: TermoInquiricao, Titulo: "Depoimento de testemunha", Paginas: 1},
		{Cadastro: "PEC-02", Funcao: DefesaPrevia, Titulo: "Defesa prévia", SEI: "0090000077", Paginas: 1},
		{Cadastro: "PEC-03", Funcao: AlegacoesFinais, Titulo: "Alegações finais", Paginas: 1},
	}
	ps, err := DeDeclaracao(a, ds)
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 4 || ps[0].PaginaFinal != 2 || ps[1].PaginaInicial != 3 || ps[3].PaginaFinal != 5 {
		t.Fatalf("páginas: %+v", ps)
	}
	for i, p := range ps {
		if p.Base != BaseUsuario || p.Declarada != ds[i].Cadastro || p.Funcao != ds[i].Funcao {
			t.Fatalf("peça %d: %+v", i, p)
		}
	}
	if ps[2].SEI != "0090000077" || !ps[1].Externo {
		t.Fatalf("SEI: %+v", ps)
	}
	c := VerificarDeclaradas("PADS", ps, a)
	if c.Resultado != Divergente || c.PecaCentral != "PEC-01" || len(c.Divergencias) != 1 {
		t.Fatalf("conferência: %+v", c)
	}
	if d := c.Divergencias[0]; d.Peca != "PEC-02" || d.Declarada != TermoInquiricao || d.Aparente != TermoDeclaracoes || d.Pagina != 3 {
		t.Fatalf("divergência: %+v", d)
	}
	// A declaração prevalece: a função não foi trocada.
	if ps[1].Funcao != TermoInquiricao {
		t.Fatal("função declarada trocada")
	}
	// Manifestação (ambígua) declarada como alegações finais não diverge.
	ds[1].Funcao = TermoDeclaracoes
	ps, _ = DeDeclaracao(a, ds)
	if c := VerificarDeclaradas("PADS", ps, a); c.Resultado != Compativel {
		t.Fatalf("%+v", c)
	}
	// Soma de páginas diferente dos autos: erro.
	ds[0].Paginas = 1
	if _, err := DeDeclaracao(a, ds); err == nil {
		t.Fatal("soma de páginas divergente aceita")
	}
}

// T04/T06 com peças declaradas: central de outro procedimento e central
// ilegível.
func TestVerificarDeclaradasCentral(t *testing.T) {
	a := autosDeclarados()
	ds := []Declarada{{Cadastro: "PEC-01", Funcao: PortariaSR, Paginas: 2}, {Cadastro: "PEC-02", Funcao: Anexo, Paginas: 3}}
	ps, _ := DeDeclaracao(a, ds)
	if c := VerificarDeclaradas("SR", ps, a); c.Resultado != Incompativel || c.TipoAparente != "PADS" {
		t.Fatalf("T04: %+v", c)
	}
	a.Paginas[1].Situacao = autos.Ilegivel
	ds[0].Funcao = MemorandoDisciplinar
	ps, _ = DeDeclaracao(a, ds)
	if c := VerificarDeclaradas("PADS", ps, a); c.Resultado != PecaIlegivel {
		t.Fatalf("T06: %+v", c)
	}
	ds[0].Funcao = Anexo
	ps, _ = DeDeclaracao(a, ds)
	if c := VerificarDeclaradas("PADS", ps, a); c.Resultado != PecaAusente {
		t.Fatalf("sem central: %+v", c)
	}
}

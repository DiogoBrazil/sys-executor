package workflow

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

func pdfFalso(s string) []byte { return []byte("%PDF-1.7\n% " + s + "\n%%EOF\n") }

// Cadastro peça por peça (V4 §48.4): peças fixas da modalidade, pessoas,
// depoimentos com depoente e papel, vários arquivos por peça, arquivo
// repetido recusado, pendências e conclusão; tudo na sessão.
// Cobre: CAD-PEC-01
// Regra: TRV-ING-007
func TestCadastroPADS(t *testing.T) {
	visto := make(chan *Cadastro, 1)
	etapas := map[Estado]Etapa{Verificando: func(ctx context.Context, c *Contexto) (Estado, error) {
		visto <- c.Cadastro()
		return Extraindo, nil
	}}
	g, _, _ := novo(t, etapas)
	g.contarPaginas = func(b []byte) (int, error) { return 2, nil }
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoPADS)
	v, err := g.EscolherEntrada(EntradaUpload)
	if err != nil || v.Estado != RecebendoUpload || v.Cadastro == nil || len(v.Cadastro.Pecas) != 3 {
		t.Fatalf("cadastro inicial: %+v %v", v, err)
	}
	memo, defesa, aleg := v.Cadastro.Pecas[0], v.Cadastro.Pecas[1], v.Cadastro.Pecas[2]
	if memo.Tipo != PecaMemorando || defesa.Tipo != PecaDefesa || aleg.Tipo != PecaAlegacoes || len(v.PendenciasCadastro) == 0 {
		t.Fatalf("peças fixas: %+v", v.Cadastro.Pecas)
	}
	if _, err := g.ConcluirCadastro(); err == nil {
		t.Fatal("concluir com pendências")
	}
	// Pessoas e depoimentos (a mesma pessoa com duas oitivas).
	c := *v.Cadastro
	c.Pessoas = []PessoaCadastro{{Nome: "  Carlos   Inventado ", PostoGrad: "CB PM"}, {Nome: "Diego Suposto", Grafias: []string{"DIEGO"}}}
	v, err = g.SalvarCadastro(c)
	if err != nil {
		t.Fatal(err)
	}
	acusado, test := v.Cadastro.Pessoas[0], v.Cadastro.Pessoas[1]
	if acusado.Nome != "Carlos Inventado" || !strings.HasPrefix(acusado.ID, "PES-") {
		t.Fatalf("pessoa: %+v", acusado)
	}
	c = *v.Cadastro
	c.AcusadoID = acusado.ID
	c.Fatos = []FatoCadastro{{Resumo: "usou a viatura", Data: "05/04/2026", Enquadramentos: []EnquadramentoCadastro{{DispositivoID: "RDPM-17-V", Natureza: "LEVE"}}}}
	c.Pecas = append(c.Pecas,
		PecaCadastro{Tipo: PecaDepoimento, Papel: PapelTestemunha, PessoaIDs: []string{test.ID}, Data: "06/04/2026"},
		PecaCadastro{Tipo: PecaDepoimento, Papel: PapelTestemunha, PessoaIDs: []string{test.ID}, Data: "07/04/2026"})
	c.Pecas[1].NaoApresentada = true // defesa prévia não apresentada
	v, err = g.SalvarCadastro(c)
	if err != nil {
		t.Fatal(err)
	}
	if f := v.Cadastro.Fatos[0]; f.Enquadramentos[0].Natureza != "" {
		t.Fatal("no RDPM a natureza é a do artigo, não a informada")
	}
	dep1, dep2 := v.Cadastro.Pecas[3], v.Cadastro.Pecas[4]
	if dep1.ID == dep2.ID || dep1.PessoaIDs[0] != dep2.PessoaIDs[0] {
		t.Fatalf("oitivas distintas da mesma pessoa: %+v %+v", dep1, dep2)
	}
	// Depoimento de "sindicado" não existe no PADS; pessoa inexistente recusada.
	ruim := *v.Cadastro
	ruim.Pecas = append(ruim.Pecas, PecaCadastro{Tipo: PecaDepoimento, Papel: PapelSindicado, PessoaIDs: []string{"PES-99"}})
	if _, err := g.SalvarCadastro(ruim); err == nil {
		t.Fatal("depoimento inválido aceito")
	}
	// Arquivos: vários por peça, repetido recusado, ordem alterável.
	if _, err := g.AnexarArquivos(memo.ID, []ArquivoEntrada{{Nome: "memo.pdf", Dados: pdfFalso("m1")}, {Nome: "anexo.pdf", Dados: pdfFalso("m2")}}); err != nil {
		t.Fatal(err)
	}
	if _, err := g.AnexarArquivos(dep1.ID, []ArquivoEntrada{{Nome: "copia.pdf", Dados: pdfFalso("m1")}}); codigo(err) != erros.ArquivoRepetido {
		t.Fatalf("arquivo repetido: %v", err)
	}
	if _, err := g.AnexarArquivos(dep1.ID, []ArquivoEntrada{{Nome: "x.pdf", Dados: []byte("não é pdf")}}); err == nil {
		t.Fatal("arquivo que não é PDF aceito")
	}
	if _, err := g.AnexarArquivos(defesa.ID, []ArquivoEntrada{{Nome: "d.pdf", Dados: pdfFalso("d")}}); err == nil {
		t.Fatal("peça não apresentada não recebe arquivo")
	}
	v = g.Visao()
	arqs := v.Cadastro.Pecas[0].Arquivos
	if len(arqs) != 2 || arqs[0].Nome != "memo.pdf" || arqs[0].Paginas != 2 || len(arqs[0].Hash) != 12 {
		t.Fatalf("arquivos: %+v", arqs)
	}
	v, _ = g.MoverArquivo(memo.ID, arqs[1].ID, -1)
	if v.Cadastro.Pecas[0].Arquivos[0].Nome != "anexo.pdf" {
		t.Fatal("reordenar")
	}
	v, _ = g.RemoverArquivo(memo.ID, arqs[1].ID)
	if len(v.Cadastro.Pecas[0].Arquivos) != 1 {
		t.Fatal("remover")
	}
	if _, err := g.Sessao().Ler(arqs[1].NomeLogico(memo.ID)); err == nil {
		t.Fatal("arquivo removido continua na sessão")
	}
	for _, id := range []string{dep1.ID, dep2.ID} {
		if _, err := g.AnexarArquivos(id, []ArquivoEntrada{{Nome: "termo.pdf", Dados: pdfFalso(id)}}); err != nil {
			t.Fatal(err)
		}
	}
	if v := g.Visao(); len(v.PendenciasCadastro) != 1 || !strings.Contains(v.PendenciasCadastro[0], "Alegações finais") {
		t.Fatalf("pendências: %v", v.PendenciasCadastro)
	}
	c = *g.Visao().Cadastro
	c.Pecas[2].NaoApresentada = true
	if _, err := g.SalvarCadastro(c); err != nil {
		t.Fatal(err)
	}
	// Peça removida: os arquivos dela saem da sessão.
	c = *g.Visao().Cadastro
	arqDep2 := c.Pecas[4].Arquivos[0].NomeLogico(dep2.ID)
	if _, err := g.Sessao().Ler(arqDep2); err != nil {
		t.Fatal(err)
	}
	c.Pecas = c.Pecas[:4]
	if _, err := g.SalvarCadastro(c); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Sessao().Ler(arqDep2); err == nil {
		t.Fatal("arquivo de peça removida continua na sessão")
	}
	if v, err := g.ConcluirCadastro(); err != nil || v.Estado != Verificando {
		t.Fatalf("concluir: %+v %v", v, err)
	}
	var cadastroVisto *Cadastro
	select {
	case cadastroVisto = <-visto:
	case <-time.After(5 * time.Second):
		t.Fatal("o pipeline não recebeu o cadastro")
	}
	if !cadastroVisto.Concluido || len(cadastroVisto.Pecas) != 4 || cadastroVisto.AcusadoID != acusado.ID {
		t.Fatalf("pipeline recebe o cadastro concluído: %+v", cadastroVisto)
	}
}

// SR: sindicados e ofendidos (ou ausência declarada); depoimento de sindicado.
func TestCadastroSR(t *testing.T) {
	g, _, _ := novo(t, nil)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoSR)
	v, _ := g.EscolherEntrada(EntradaUpload)
	if len(v.Cadastro.Pecas) != 1 || v.Cadastro.Pecas[0].Tipo != PecaPortaria {
		t.Fatalf("%+v", v.Cadastro.Pecas)
	}
	c := *v.Cadastro
	c.Pessoas = []PessoaCadastro{{Nome: "Sind Um"}, {Nome: "Sind Dois"}}
	v, _ = g.SalvarCadastro(c)
	c = *v.Cadastro
	c.Sindicados = []string{c.Pessoas[0].ID, c.Pessoas[1].ID}
	c.Pecas = append(c.Pecas, PecaCadastro{Tipo: PecaDepoimento, Papel: PapelSindicado, PessoaIDs: []string{c.Pessoas[0].ID}})
	v, err := g.SalvarCadastro(c)
	if err != nil {
		t.Fatal(err)
	}
	if ps := strings.Join(v.PendenciasCadastro, " "); !strings.Contains(ps, "ofendido") || !strings.Contains(ps, "Portaria") {
		t.Fatalf("pendências SR: %v", v.PendenciasCadastro)
	}
	c = *v.Cadastro
	c.SemOfendido = true
	c.Pecas = append(c.Pecas, PecaCadastro{Tipo: PecaDefesa})
	if _, err := g.SalvarCadastro(c); err == nil {
		t.Fatal("defesa prévia não existe na SR")
	}
}

// IDs provisórios da interface: a pessoa nova ganha ID definitivo e as
// referências a ela (acusado, depoente) são reescritas; ID inexistente
// nunca é aceito como definitivo.
func TestCadastroIDsProvisorios(t *testing.T) {
	g, _, _ := novo(t, nil)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoPADS)
	v, _ := g.EscolherEntrada(EntradaUpload)
	c := *v.Cadastro
	c.Pessoas = []PessoaCadastro{{ID: "NOVO-1", Nome: "Carlos Inventado"}, {ID: "PES-77", Nome: "Diego Suposto"}}
	c.AcusadoID = "NOVO-1"
	c.Pecas = append(c.Pecas, PecaCadastro{ID: "NOVO-2", Tipo: PecaDepoimento, Papel: PapelTestemunha, PessoaIDs: []string{"PES-77"}})
	v, err := g.SalvarCadastro(c)
	if err != nil {
		t.Fatal(err)
	}
	ac, dep := v.Cadastro.Pessoas[0], v.Cadastro.Pessoas[1]
	if ac.ID == "NOVO-1" || dep.ID == "PES-77" || v.Cadastro.AcusadoID != ac.ID || v.Cadastro.Pecas[3].PessoaIDs[0] != dep.ID {
		t.Fatalf("remapeamento: %+v", v.Cadastro)
	}
	if v.Cadastro.Pecas[3].ID == "NOVO-2" {
		t.Fatal("peça nova com ID provisório")
	}
}

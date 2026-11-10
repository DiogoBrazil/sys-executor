package pipeline

import (
	"context"
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/ocr"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/pseudonimo"
	"github.com/DiogoBrazil/sys-executor/internal/rules/civil"
	"github.com/DiogoBrazil/sys-executor/internal/session"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/pdfsint"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

func montarComCadastro(t *testing.T) *workflow.Gerenciador {
	t.Helper()
	extOnce.Do(func() { ext = pdf.NovoExtrator() })
	p := &Pipeline{Extrator: ext, LocalizarOCR: func() (ocr.Motor, error) { return ocrFalso{conf: 90}, nil }, DPI: 72}
	g := workflow.Novo(workflow.Opcoes{Sessoes: session.NovoGerenciador(t.TempDir()), Etapas: p.Etapas(),
		ContarPaginas: func(b []byte) (int, error) { return pdf.ContarPaginas(context.Background(), ext, b) }})
	t.Cleanup(g.Encerrar)
	return g
}

func pdfDe(linhas ...[]string) []byte {
	var ps []pdfsint.Pagina
	for _, l := range linhas {
		ps = append(ps, pdfsint.Pagina{Linhas: l})
	}
	return pdfsint.Gerar(ps)
}

// Cadastro PADS completo (sintético): os arquivos são juntados na ordem
// processual (Memorando, defesa, oitivas, alegações), cada peça declarada
// vira uma peça dos autos e a tarefa chega à delimitação. Conteúdo que
// contradiz a declaração vira pergunta; "Voltar ao cadastro" o preserva.
// Cobre: CAD-PEC-03
// Regra: TRV-ING-007
func TestCadastroPADSAteDelimitacao(t *testing.T) {
	g := montarComCadastro(t)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(workflow.TipoPADS)
	v, _ := g.EscolherEntrada(workflow.EntradaUpload)
	c := *v.Cadastro
	c.Pessoas = []workflow.PessoaCadastro{{Nome: "Carlos Inventado", PostoGrad: "CB PM"}, {Nome: "Diego Suposto"}}
	v, err := g.SalvarCadastro(c)
	if err != nil {
		t.Fatal(err)
	}
	c = *v.Cadastro
	c.AcusadoID = c.Pessoas[0].ID
	c.Pecas = append(c.Pecas, workflow.PecaCadastro{Tipo: workflow.PecaDepoimento, Papel: workflow.PapelDeclarante, PessoaIDs: []string{c.Pessoas[1].ID}})
	c.Pecas[2].NaoApresentada = true // alegações finais
	v, err = g.SalvarCadastro(c)
	if err != nil {
		t.Fatal(err)
	}
	memo, defesa, dep := v.Cadastro.Pecas[0].ID, v.Cadastro.Pecas[1].ID, v.Cadastro.Pecas[3].ID
	anexar := func(peca string, b []byte) {
		t.Helper()
		if _, err := g.AnexarArquivos(peca, []workflow.ArquivoEntrada{{Nome: peca + ".pdf", Dados: b}}); err != nil {
			t.Fatal(err)
		}
	}
	// O depoimento é anexado antes da defesa: a junção segue a ordem processual.
	anexar(dep, pdfDe([]string{"TERMO DE INQUIRIÇÃO DE TESTEMUNHA", "Diego Suposto disse que viu a viatura."}))
	anexar(memo, pdfDe([]string{"MEMORANDO DISCIPLINAR Nº 1/2026", "Ao CB PM Carlos Inventado."}, []string{"continuação do memorando"}))
	anexar(defesa, pdfDe([]string{"DEFESA PRÉVIA", "A defesa nega os fatos."}))
	if _, err := g.ConcluirCadastro(); err != nil {
		t.Fatal(err)
	}
	v = esperar(t, g, func(v *workflow.Visao) bool {
		return v.Estado == workflow.AguardandoEsclarecimento || v.Parada || v.Terminal
	})
	a := v.Autos
	if a == nil || a.TotalPaginas != 4 || len(a.Pecas) != 3 {
		t.Fatalf("autos juntados: %+v", a)
	}
	if a.Pecas[0].Funcao != "MEMORANDO_DISCIPLINAR" || a.Pecas[0].PaginaFinal != 2 || a.Pecas[1].Funcao != "DEFESA_PREVIA" ||
		a.Pecas[2].Funcao != "TERMO_DECLARACOES" || !strings.Contains(a.Pecas[2].Titulo, "Depoimento de declarante — Diego Suposto") {
		t.Fatalf("peças declaradas: %+v", a.Pecas)
	}
	// Declarado como declarante, o termo é de inquirição: pergunta, sem troca.
	if v.Pergunta == nil || v.Pergunta.ID != perguntaCompat || !strings.Contains(v.Pergunta.Contexto, "termo de inquirição") {
		t.Fatalf("pergunta: %+v", v.Pergunta)
	}
	v, err = g.Responder(perguntaCompat, "outro", nil)
	if err != nil || v.Estado != workflow.RecebendoUpload || v.Cadastro == nil || v.Cadastro.Concluido ||
		len(v.Cadastro.Pecas[3].Arquivos) != 1 {
		t.Fatalf("voltar ao cadastro: %+v %v", v, err)
	}
	c = *v.Cadastro
	c.Pecas[3].Papel = workflow.PapelTestemunha
	if _, err := g.SalvarCadastro(c); err != nil {
		t.Fatal(err)
	}
	if _, err := g.ConcluirCadastro(); err != nil {
		t.Fatal(err)
	}
	v = esperar(t, g, func(v *workflow.Visao) bool { return v.Parada || v.Terminal })
	if v.Estado != workflow.Delimitando || v.Erro == nil || v.Erro.Codigo != "IA-001" || v.Autos.Compatibilidade != "COMPATIVEL" {
		t.Fatalf("%+v", v)
	}
	d := g.Dados().(*Dados)
	if len(d.Declaradas) != 3 || d.Declaradas[0].Cadastro != memo || d.Pecas[2].Declarada != dep {
		t.Fatalf("declaradas: %+v", d.Declaradas)
	}
}

// Pessoas do cadastro semeiam o mapa (inclusive sem âncora no texto) e a
// marcação de criança bloqueia o ChatGPT.
// Regra: TRV-SEG-011
func TestSemearMapaDoCadastro(t *testing.T) {
	cad := &workflow.Cadastro{Pessoas: []workflow.PessoaCadastro{{ID: "PES-01", Nome: "Joana Inventada", Grafias: []string{"Jojô"}}, {ID: "PES-02", Nome: "Pedrinho Fictício", Crianca: true}}}
	m := pseudonimo.NovoMapa()
	if !semearMapa(m, cad) {
		t.Fatal("criança não sinalizada")
	}
	o := m.Ocultar("joana inventada (Jojô) levou Pedrinho Fictício.")
	if strings.Contains(o, "Jojô") || strings.Contains(o, "joana") || strings.Contains(o, "Pedrinho") {
		t.Fatal(o)
	}
}

// E9: o cadastro confere a leitura do Memorando. O dispositivo do cadastro
// preenche a imputação de mesmo dispositivo citado; o que não corresponde
// vira imputação extra, com o alerta e a confirmação da V4 §22.1; acusado
// divergente gera alerta.
// Regra: PADS-ENQ-001
func TestConferirComCadastro(t *testing.T) {
	r := analise.PropostaDelimitacaoPADS{
		Acusados: []analise.PessoaProposta{{Nome: "Carlos Outro", PostoGraduacao: "CB PM"}},
		Imputacoes: []analise.ImputacaoProposta{
			{ID: "IMP-1", Conduta: "usou a viatura", Diploma: "RDPM", Artigo: 17, Inciso: "V", Data: civil.D(5, 4, 2026)},
			{ID: "IMP-2", Conduta: "faltou", Diploma: "RDPM", Artigo: 16, Inciso: "XLIII"},
		},
	}
	cad := &workflow.Cadastro{Tipo: workflow.TipoPADS, AcusadoID: "PES-01",
		Pessoas: []workflow.PessoaCadastro{{ID: "PES-01", Nome: "Carlos Inventado"}},
		Fatos: []workflow.FatoCadastro{{ID: "FATO-02", Resumo: "viatura", Data: "06/04/2026", Enquadramentos: []workflow.EnquadramentoCadastro{
			{DispositivoID: "RDPM-17-V"}, {DispositivoID: "EST-29-I", Natureza: "MEDIA"}}}}}
	pg := perguntaImputacoes(r, workflow.Resposta{}, nil, cad)
	valor := map[string]string{}
	alertas := map[string]string{}
	for _, c := range pg.Campos {
		valor[c.ID] = c.Valor
	}
	for _, gr := range pg.Grupos {
		alertas[gr.ID] = strings.Join(gr.Alertas, " ")
	}
	if valor["diploma_IMP-1"] != "RDPM" || valor["disp_rdpm_IMP-1"] != "RDPM-17-V" || !strings.Contains(alertas["IMP-1"], "06/04/2026") {
		t.Fatalf("IMP-1: %v / %s", valor, alertas["IMP-1"])
	}
	if valor["diploma_IMP-2"] != "" || !strings.Contains(alertas["IMP-2"], "Nenhum dispositivo informado no cadastro") {
		t.Fatalf("IMP-2 sem correspondência não pode ser preenchida: %q %s", valor["diploma_IMP-2"], alertas["IMP-2"])
	}
	if valor["incluir_IMP-X1"] != "sim" || valor["disp_est_IMP-X1"] != "EST-29-I" || valor["natureza_IMP-X1"] != "MEDIA" || valor["conduta_IMP-X1"] != "viatura" {
		t.Fatalf("extra do cadastro: %v", valor)
	}
	if !strings.Contains(alertas["acusado"], "Carlos Inventado") {
		t.Fatalf("acusado divergente: %s", alertas["acusado"])
	}
	// A extra do cadastro, incluída, exige a confirmação expressa (V4 §22.1).
	resp := workflow.Resposta{Opcao: "confirmar", Valores: valor}
	if div, pend := divergencias(r, resp); pend == 0 || div["IMP-X1"] == "" {
		t.Fatalf("divergência da extra: %v %d", div, pend)
	}
}

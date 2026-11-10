package workflow

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/session"
)

var pdfMinimo = []byte("%PDF-1.7\n1 0 obj<<>>endobj\ntrailer<<>>\n%%EOF\n")

type gravador struct {
	mu     sync.Mutex
	visoes []*Visao
	sinal  chan struct{}
}

func novoGravador() *gravador { return &gravador{sinal: make(chan struct{}, 100)} }

func (r *gravador) notificar(v *Visao) {
	r.mu.Lock()
	r.visoes = append(r.visoes, v)
	r.mu.Unlock()
	select {
	case r.sinal <- struct{}{}:
	default:
	}
}

// esperar aguarda até a condição sobre a visão atual ou falha após 5 s.
func esperar(t *testing.T, g *Gerenciador, cond func(*Visao) bool) *Visao {
	t.Helper()
	limite := time.After(5 * time.Second)
	for {
		if v := g.Visao(); cond(v) {
			return v
		}
		select {
		case <-limite:
			t.Fatalf("tempo esgotado; visão atual: %+v", g.Visao())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func novo(t *testing.T, etapas map[Estado]Etapa) (*Gerenciador, *gravador, string) {
	t.Helper()
	raiz := t.TempDir()
	r := novoGravador()
	g := Novo(Opcoes{Sessoes: session.NovoGerenciador(raiz), Notificar: r.notificar, Etapas: etapas})
	t.Cleanup(g.Encerrar)
	return g, r, raiz
}

func codigo(err error) erros.Codigo {
	var e *erros.Erro
	if errors.As(err, &e) {
		return e.Codigo
	}
	return ""
}

func pastasDeSessao(t *testing.T, raiz string) int {
	t.Helper()
	es, _ := os.ReadDir(raiz)
	return len(es)
}

// ---------- tabela de transições ----------

func TestTabelaDeTransicoes(t *testing.T) {
	todos := Todos()
	if len(todos) != 28 {
		t.Fatalf("lista de estados mudou (%d); revise a tabela e os testes", len(todos))
	}
	for _, e := range todos {
		if _, ok := descricoes[e]; !ok {
			t.Errorf("%s sem descrição para a interface", e)
		}
		if e.Terminal() {
			for _, p := range todos {
				if Permitida(e, p) {
					t.Errorf("terminal %s não pode sair para %s", e, p)
				}
			}
			continue
		}
		if !Permitida(e, Cancelada) || !Permitida(e, Falha) {
			t.Errorf("%s deve permitir CANCELADA e FALHA", e)
		}
	}
	// Todo estado é alcançável a partir de NOVA.
	alc := map[Estado]bool{Nova: true}
	for mudou := true; mudou; {
		mudou = false
		for de := range alc {
			for _, p := range todos {
				if !alc[p] && Permitida(de, p) {
					alc[p] = true
					mudou = true
				}
			}
		}
	}
	for _, e := range todos {
		if !alc[e] {
			t.Errorf("%s inalcançável a partir de NOVA", e)
		}
	}
	// Caminho feliz até FINALIZADA (PADS) é válido.
	feliz := []Estado{Nova, TipoEscolhido, EntradaEscolhida, RecebendoUpload, Verificando, Extraindo,
		Classificando, Compatibilidade, Delimitando, Analisando, ConsultandoFontes, Calculando,
		Redigindo, Validando, AguardandoRevisao, Ajustando, Validando, AguardandoRevisao, Finalizando, Finalizada}
	for i := 1; i < len(feliz); i++ {
		if !Permitida(feliz[i-1], feliz[i]) {
			t.Errorf("caminho feliz quebrado em %s → %s", feliz[i-1], feliz[i])
		}
	}
	// Proibições relevantes.
	for _, par := range [][2]Estado{
		{Nova, EntradaEscolhida},        // T01: entrada antes do tipo
		{AguardandoRevisao, Finalizada}, // A31: finalizar só via FINALIZANDO (comando do usuário)
		{RecebendoUpload, SEILogin},     // T02: upload nunca passa pelo SEI
		{Verificando, AguardandoEsclarecimento},
		{Extraindo, Redigindo},
	} {
		if Permitida(par[0], par[1]) {
			t.Errorf("transição proibida aceita: %s → %s", par[0], par[1])
		}
	}
}

// ---------- T01: ordem das escolhas ----------

func TestT01OrdemDasEscolhas(t *testing.T) {
	g, _, _ := novo(t, nil)
	if _, err := g.EscolherTipo(TipoPADS); codigo(err) != erros.TarefaSemTarefa {
		t.Fatalf("sem tarefa: %v", err)
	}
	v, err := g.Iniciar()
	if err != nil || v.Estado != Nova {
		t.Fatalf("iniciar: %+v %v", v, err)
	}
	if _, err := g.EscolherEntrada(EntradaUpload); codigo(err) != erros.TarefaOrdemInvalida {
		t.Fatalf("T01: entrada antes do tipo deveria falhar, veio %v", err)
	}
	if _, err := g.EscolherTipo("XYZ"); codigo(err) != erros.TarefaOrdemInvalida {
		t.Fatalf("tipo inválido: %v", err)
	}
	if v, _ = g.EscolherTipo(TipoSR); v.Tipo != TipoSR || v.Estado != TipoEscolhido {
		t.Fatalf("%+v", v)
	}
	if v, _ = g.EscolherTipo(TipoPADS); v.Tipo != TipoPADS { // pode trocar antes da entrada
		t.Fatalf("%+v", v)
	}
	if v, err = g.EscolherEntrada(EntradaUpload); err != nil || v.Estado != RecebendoUpload || v.Tipo != TipoPADS {
		t.Fatalf("%+v %v", v, err)
	}
	// Voltar preserva o tipo.
	if v, err = g.VoltarParaEntrada(); err != nil || v.Estado != TipoEscolhido || v.Tipo != TipoPADS || v.Entrada != "" {
		t.Fatalf("voltar: %+v %v", v, err)
	}
}

// SEI ainda indisponível (Fase 2): aviso amigável, a tarefa continua na escolha.
func TestSEIIndisponivelPreservaEscolha(t *testing.T) {
	g, _, _ := novo(t, nil)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoSR)
	if _, err := g.EscolherEntrada(EntradaSEI); codigo(err) != erros.TarefaEtapaIndisponiv {
		t.Fatalf("esperado TAR-004, veio %v", err)
	}
	if v := g.Visao(); v.Estado != TipoEscolhido || v.Tipo != TipoSR {
		t.Fatalf("escolha perdida: %+v", v)
	}
}

// ---------- SES-CONC-01 ----------

func TestUmaTarefaPorVez(t *testing.T) {
	g, _, raiz := novo(t, nil)
	if _, err := g.Iniciar(); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Iniciar(); codigo(err) != erros.TarefaJaAtiva {
		t.Fatalf("segunda tarefa: %v", err)
	}
	if err := g.Descartar(); codigo(err) != erros.TarefaJaAtiva {
		t.Fatal("não pode descartar tarefa ativa")
	}
	if _, err := g.Cancelar(); err != nil {
		t.Fatal(err)
	}
	if n := pastasDeSessao(t, raiz); n != 0 {
		t.Fatalf("cancelar deveria apagar a sessão; restam %d", n)
	}
	if _, err := g.Iniciar(); err != nil {
		t.Fatalf("após cancelar, nova tarefa deve ser possível: %v", err)
	}
}

// ---------- T02: upload sem SEI nem credencial ----------

func TestT02UploadNaoPassaPeloSEI(t *testing.T) {
	g, r, raiz := novo(t, nil)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoPADS)
	_, _ = g.EscolherEntrada(EntradaUpload)

	for dados, cod := range map[string]erros.Codigo{"": erros.PDFVazio, "<html>": erros.PDFInvalido} {
		if _, err := g.ReceberUpload([]byte(dados)); codigo(err) != cod {
			t.Fatalf("%q: esperado %s, veio %v", dados, cod, err)
		}
		if g.Visao().Estado != RecebendoUpload {
			t.Fatal("arquivo inválido deve manter a tarefa aguardando outro arquivo")
		}
	}
	if _, err := g.ReceberUpload(pdfMinimo); err != nil {
		t.Fatal(err)
	}
	v := esperar(t, g, func(v *Visao) bool { return v != nil && v.Parada })
	for _, e := range g.Historico() {
		if strings.HasPrefix(string(e), "SEI_") {
			t.Fatalf("T02: upload passou por %s", e)
		}
	}
	// Sem etapa registrada: pára com alerta explícito, sem simular sucesso nem destruir a tarefa.
	if v.Estado != Verificando || v.Terminal || v.Erro == nil || v.Erro.Codigo != string(erros.TarefaEtapaIndisponiv) {
		t.Fatalf("esperado parada com TAR-004, veio %+v", v)
	}
	if n := pastasDeSessao(t, raiz); n != 1 {
		t.Fatalf("a sessão deve continuar disponível para consulta; há %d", n)
	}
	if _, err := g.Cancelar(); err != nil || pastasDeSessao(t, raiz) != 0 {
		t.Fatal("cancelar deve apagar a sessão")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.visoes) == 0 {
		t.Fatal("nenhuma notificação de estado")
	}
}

// ---------- processamento, progresso e cancelamento ----------

func TestEtapasProgressoECancelamento(t *testing.T) {
	bloqueio := make(chan struct{})
	var lido []byte
	etapas := map[Estado]Etapa{
		Verificando: func(ctx context.Context, c *Contexto) (Estado, error) {
			b, err := c.Sessao.Ler(NomeAutos)
			lido = b
			return Extraindo, err
		},
		Extraindo: func(ctx context.Context, c *Contexto) (Estado, error) {
			c.Progresso(3, 10, "páginas")
			close(bloqueio)
			<-ctx.Done()
			return "", ctx.Err()
		},
	}
	g, _, raiz := novo(t, etapas)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoSR)
	_, _ = g.EscolherEntrada(EntradaUpload)
	if _, err := g.ReceberUpload(pdfMinimo); err != nil {
		t.Fatal(err)
	}
	<-bloqueio
	v := esperar(t, g, func(v *Visao) bool { return v.Progresso != nil })
	if v.Estado != Extraindo || v.Progresso.Atual != 3 || v.Progresso.Total != 10 {
		t.Fatalf("%+v", v)
	}
	if string(lido) != string(pdfMinimo) {
		t.Fatal("etapa não leu o PDF da sessão")
	}
	// Linha do tempo: obter concluído, extrair ativo, SR sem etapa de cálculo.
	sit := map[string]Situacao{}
	for _, m := range v.Etapas {
		sit[m.Chave] = m.Situacao
	}
	if sit["obter"] != Concluida || sit["extrair"] != Ativa || sit["analisar"] != Pendente {
		t.Fatalf("linha do tempo: %+v", v.Etapas)
	}
	if _, ok := sit["calcular"]; ok {
		t.Fatal("SR não tem etapa de cálculo (V4 §21)")
	}
	v, err := g.Cancelar()
	if err != nil || v.Estado != Cancelada || v.Progresso != nil {
		t.Fatalf("%+v %v", v, err)
	}
	g.Encerrar() // espera a goroutine
	if n := pastasDeSessao(t, raiz); n != 0 {
		t.Fatalf("restam %d sessões", n)
	}
	if err := g.Descartar(); err != nil || g.Visao() != nil {
		t.Fatal("descartar deve limpar a tarefa terminal")
	}
}

// Progresso só aparece com métrica real (V4 §31.1; T07).
func TestSemProgressoSemMetrica(t *testing.T) {
	pausa := make(chan struct{})
	etapas := map[Estado]Etapa{
		Verificando: func(ctx context.Context, c *Contexto) (Estado, error) {
			close(pausa)
			<-ctx.Done()
			return "", ctx.Err()
		},
	}
	g, _, _ := novo(t, etapas)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoPADS)
	_, _ = g.EscolherEntrada(EntradaUpload)
	_, _ = g.ReceberUpload(pdfMinimo)
	<-pausa
	if v := g.Visao(); v.Progresso != nil {
		t.Fatalf("progresso inventado: %+v", v.Progresso)
	}
}

func TestPanicoNaEtapaViraFalhaAmigavel(t *testing.T) {
	etapas := map[Estado]Etapa{
		Verificando: func(ctx context.Context, c *Contexto) (Estado, error) { panic("defeito") },
	}
	g, _, _ := novo(t, etapas)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoPADS)
	_, _ = g.EscolherEntrada(EntradaUpload)
	_, _ = g.ReceberUpload(pdfMinimo)
	v := esperar(t, g, func(v *Visao) bool { return v.Terminal })
	if v.Estado != Falha || v.Erro.Codigo != string(erros.InternoInesperado) || strings.Contains(v.Erro.Mensagem, "defeito") {
		t.Fatalf("%+v", v.Erro)
	}
}

// Etapa que devolve transição inválida não corrompe o estado.
func TestEtapaComTransicaoInvalida(t *testing.T) {
	etapas := map[Estado]Etapa{
		Verificando: func(ctx context.Context, c *Contexto) (Estado, error) { return Finalizada, nil },
	}
	g, _, _ := novo(t, etapas)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoPADS)
	_, _ = g.EscolherEntrada(EntradaUpload)
	_, _ = g.ReceberUpload(pdfMinimo)
	v := esperar(t, g, func(v *Visao) bool { return v.Terminal })
	if v.Estado != Falha {
		t.Fatalf("%+v", v)
	}
}

// Encerrar o app com tarefa ativa cancela e apaga a sessão.
func TestEncerrarAppComTarefaAtiva(t *testing.T) {
	g, _, raiz := novo(t, nil)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoPADS)
	g.Encerrar()
	if v := g.Visao(); v.Estado != Cancelada {
		t.Fatalf("%+v", v)
	}
	if n := pastasDeSessao(t, raiz); n != 0 {
		t.Fatalf("restam %d sessões", n)
	}
}

// ---------- perguntas e erros recuperáveis ----------

type dadosTeste struct{ liberado *bool }

func (d dadosTeste) Liberar() { *d.liberado = true }

func TestPerguntaRespostaEDestinos(t *testing.T) {
	liberado := false
	var execucoes int
	etapas := map[Estado]Etapa{
		Verificando: func(ctx context.Context, c *Contexto) (Estado, error) {
			c.Dados = dadosTeste{liberado: &liberado}
			return Extraindo, nil
		},
		Extraindo:     func(ctx context.Context, c *Contexto) (Estado, error) { return Classificando, nil },
		Classificando: func(ctx context.Context, c *Contexto) (Estado, error) { return Compatibilidade, nil },
		Compatibilidade: func(ctx context.Context, c *Contexto) (Estado, error) {
			execucoes++
			if r, ok := c.Resposta("compat"); ok && r == "continuar" {
				return Delimitando, nil
			}
			c.Perguntar(Pergunta{ID: "compat", Titulo: "Os autos parecem ser de SR", Opcoes: []Opcao{
				{ID: "continuar", Rotulo: "Continuar como PADS"},
				{ID: "outro", Rotulo: "Enviar outro PDF", Destino: RecebendoUpload},
				{ID: "cancelar", Rotulo: "Cancelar", Destino: Cancelada},
			}})
			return AguardandoEsclarecimento, nil
		},
	}
	g, _, raiz := novo(t, etapas)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoPADS)
	_, _ = g.EscolherEntrada(EntradaUpload)
	_, _ = g.ReceberUpload(pdfMinimo)
	v := esperar(t, g, func(v *Visao) bool { return v.Estado == AguardandoEsclarecimento })
	if v.Pergunta == nil || v.Pergunta.ID != "compat" || len(v.Pergunta.Opcoes) != 3 || v.Tipo != TipoPADS {
		t.Fatalf("pergunta: %+v", v.Pergunta)
	}
	if _, err := g.Responder("compat", "inexistente", nil); codigo(err) != erros.TarefaOrdemInvalida {
		t.Fatal("opção inválida deve ser rejeitada")
	}
	if _, err := g.Responder("outra", "continuar", nil); codigo(err) != erros.TarefaOrdemInvalida {
		t.Fatal("pergunta errada deve ser rejeitada")
	}
	// "Enviar outro PDF": volta ao upload, mantendo o tipo; dados liberados ao receber o novo PDF.
	if v, _ = g.Responder("compat", "outro", nil); v.Estado != RecebendoUpload || v.Tipo != TipoPADS || v.Pergunta != nil {
		t.Fatalf("%+v", v)
	}
	_, _ = g.ReceberUpload(pdfMinimo)
	if !liberado {
		t.Fatal("dados da análise anterior deveriam ser liberados")
	}
	esperar(t, g, func(v *Visao) bool { return v.Estado == AguardandoEsclarecimento })
	// "Continuar": a etapa que perguntou roda de novo e lê a resposta.
	_, _ = g.Responder("compat", "continuar", nil)
	v = esperar(t, g, func(v *Visao) bool { return v.Parada })
	if v.Estado != Delimitando || execucoes != 3 {
		t.Fatalf("%+v execucoes=%d", v, execucoes)
	}
	if pastasDeSessao(t, raiz) != 1 {
		t.Fatal("sessão deve existir enquanto a tarefa está parada")
	}
}

func TestPerguntaCancelar(t *testing.T) {
	etapas := map[Estado]Etapa{
		Verificando: func(ctx context.Context, c *Contexto) (Estado, error) { return Extraindo, nil },
		Extraindo:   func(ctx context.Context, c *Contexto) (Estado, error) { return Classificando, nil },
		Classificando: func(ctx context.Context, c *Contexto) (Estado, error) {
			c.Perguntar(Pergunta{ID: "p", Opcoes: []Opcao{{ID: "c", Destino: Cancelada}}})
			return AguardandoEsclarecimento, nil
		},
	}
	g, _, raiz := novo(t, etapas)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoSR)
	_, _ = g.EscolherEntrada(EntradaUpload)
	_, _ = g.ReceberUpload(pdfMinimo)
	esperar(t, g, func(v *Visao) bool { return v.Estado == AguardandoEsclarecimento })
	if v, _ := g.Responder("p", "c", nil); v.Estado != Cancelada || pastasDeSessao(t, raiz) != 0 {
		t.Fatalf("%+v", v)
	}
}

// PDF protegido/corrompido: volta ao upload com alerta, sem perder o tipo.
func TestErroRecuperavelVoltaAoUpload(t *testing.T) {
	etapas := map[Estado]Etapa{
		Verificando: func(ctx context.Context, c *Contexto) (Estado, error) {
			return RecebendoUpload, erros.Novo(erros.PDFProtegido)
		},
	}
	g, _, _ := novo(t, etapas)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoSR)
	_, _ = g.EscolherEntrada(EntradaUpload)
	_, _ = g.ReceberUpload(pdfMinimo)
	v := esperar(t, g, func(v *Visao) bool { return v.Estado == RecebendoUpload })
	if v.Terminal || v.Tipo != TipoSR || v.Erro == nil || v.Erro.Codigo != "PDF-004" {
		t.Fatalf("%+v", v)
	}
}

func TestValidarCampos(t *testing.T) {
	cs := []Campo{
		{ID: "pts", Rotulo: "Pontos", Tipo: CampoNumero, Obrigatorio: true, Min: 0, Max: 50},
		{ID: "data", Rotulo: "Término", Tipo: CampoData, MostrarSe: &Condicao{"tem", "sim"}, Obrigatorio: true},
		{ID: "tem", Rotulo: "Possui", Tipo: CampoSimNao, Obrigatorio: true},
		{ID: "nat", Rotulo: "Natureza", Tipo: CampoEscolha, Opcoes: []OpcaoVisao{{ID: "LEVE"}, {ID: "MEDIA"}}},
	}
	v, err := ValidarCampos(cs, map[string]string{"pts": " 35 ", "tem": "nao", "data": "lixo"})
	if err != nil || v["pts"] != "35" || v["data"] != "" {
		t.Fatalf("campo oculto não é validado nem guardado: %v %v", v, err)
	}
	for nome, vals := range map[string]map[string]string{
		"obrigatório":   {"tem": "nao"},
		"fora da faixa": {"pts": "51", "tem": "nao"},
		"data inválida": {"pts": "1", "tem": "sim", "data": "31/02/2026"},
		"escolha":       {"pts": "1", "tem": "nao", "nat": "GRAVE"},
		"sim/não":       {"pts": "1", "tem": "talvez"},
	} {
		if _, err := ValidarCampos(cs, vals); codigo(err) != erros.TarefaRespostaInvalid {
			t.Errorf("%s: %v", nome, err)
		}
	}
}

// dadosRevisaveis simula o DAG do pipeline.
type dadosRevisaveis struct{ revisoes []string }

func (d *dadosRevisaveis) Revisar(id string) (Estado, []string, bool) {
	d.revisoes = append(d.revisoes, id)
	if id == "delim" {
		return Delimitando, []string{"result"}, true
	}
	return Analisando, nil, true
}

// T08 + F4-05: pergunta com campos; a resposta pode ser revista e só o que
// depende dela é refeito; erro recuperável pára e "Tentar novamente" repete.
// Cobre: T08
func TestPerguntasComCamposRevisaoERetomada(t *testing.T) {
	var mu sync.Mutex
	dados := &dadosRevisaveis{}
	falharUmaVez := true
	etapas := map[Estado]Etapa{
		Verificando:     func(ctx context.Context, c *Contexto) (Estado, error) { return Extraindo, nil },
		Extraindo:       func(ctx context.Context, c *Contexto) (Estado, error) { return Classificando, nil },
		Classificando:   func(ctx context.Context, c *Contexto) (Estado, error) { c.Dados = dados; return Compatibilidade, nil },
		Compatibilidade: func(ctx context.Context, c *Contexto) (Estado, error) { return Delimitando, nil },
		Delimitando: func(ctx context.Context, c *Contexto) (Estado, error) {
			mu.Lock()
			f := falharUmaVez
			falharUmaVez = false
			mu.Unlock()
			if f {
				return Delimitando, erros.Novo(erros.IAChaveAusente) // recuperável
			}
			if _, ok := c.RespostaCompleta("delim"); ok {
				return Analisando, nil
			}
			valor := "IMP-1"
			if ant, ok := c.Anterior("delim"); ok {
				valor = ant.Valor("imp") // reabre com o valor dado antes
			}
			c.Perguntar(Pergunta{ID: "delim", Titulo: "Confirme as imputações", Revisavel: true,
				Campos:  []Campo{{ID: "imp", Rotulo: "Imputação", Tipo: CampoTexto, Obrigatorio: true, Valor: valor}},
				Opcoes:  []Opcao{{ID: "ok", Rotulo: "Confirmar", Validar: true}},
				Resumir: func(r Resposta) string { return r.Valor("imp") }})
			return AguardandoEsclarecimento, nil
		},
		Analisando: func(ctx context.Context, c *Contexto) (Estado, error) {
			if _, ok := c.RespostaCompleta("result"); ok {
				return ConsultandoFontes, nil
			}
			c.Perguntar(Pergunta{ID: "result", Titulo: "Resultado", Revisavel: true,
				Opcoes: []Opcao{{ID: "proc", Rotulo: "Procedente"}}})
			return AguardandoEsclarecimento, nil
		},
	}
	g, _, _ := novo(t, etapas)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoPADS)
	_, _ = g.EscolherEntrada(EntradaUpload)
	_, _ = g.ReceberUpload(pdfMinimo)

	// IA-001 na etapa: parada recuperável, tarefa viva, "Tentar novamente".
	v := esperar(t, g, func(v *Visao) bool { return v.Parada })
	if v.Estado != Delimitando || !v.Retomavel || v.Erro == nil || v.Erro.Codigo != "IA-001" || v.Terminal {
		t.Fatalf("%+v", v)
	}
	_, _ = g.Retomar()
	v = esperar(t, g, func(v *Visao) bool { return v.Estado == AguardandoEsclarecimento })
	if len(v.Pergunta.Campos) != 1 || v.Pergunta.Campos[0].Valor != "IMP-1" {
		t.Fatalf("%+v", v.Pergunta)
	}
	if _, err := g.Responder("delim", "ok", map[string]string{"imp": " "}); codigo(err) != erros.TarefaRespostaInvalid {
		t.Fatalf("campo obrigatório vazio: %v", err)
	}
	_, _ = g.Responder("delim", "ok", map[string]string{"imp": "IMP-1 e IMP-2"})
	esperar(t, g, func(v *Visao) bool { return v.Pergunta != nil && v.Pergunta.ID == "result" })
	_, _ = g.Responder("result", "proc", nil)
	v = esperar(t, g, func(v *Visao) bool { return v.Parada && v.Estado == ConsultandoFontes })
	if len(v.Respostas) != 2 || v.Respostas[0].Resumo != "IMP-1 e IMP-2" || !v.Respostas[0].Revisavel {
		t.Fatalf("respostas: %+v", v.Respostas)
	}
	// Revisar a delimitação: descarta ela e o resultado; reabre com o valor anterior.
	if _, err := g.Revisar("delim"); err != nil {
		t.Fatal(err)
	}
	v = esperar(t, g, func(v *Visao) bool { return v.Pergunta != nil && v.Pergunta.ID == "delim" })
	if v.Pergunta.Campos[0].Valor != "IMP-1 e IMP-2" || len(v.Respostas) != 0 || dados.revisoes[0] != "delim" {
		t.Fatalf("revisão: %+v %+v", v.Pergunta, v.Respostas)
	}
	_, _ = g.Responder("delim", "ok", map[string]string{"imp": "IMP-1"})
	v = esperar(t, g, func(v *Visao) bool { return v.Pergunta != nil && v.Pergunta.ID == "result" })
	if _, err := g.Revisar("inexistente"); codigo(err) != erros.TarefaOrdemInvalida {
		t.Fatal("revisar resposta inexistente")
	}
}

// A interface só recebe eventos: a última visão emitida depois de uma parada
// por erro recuperável precisa ser retomável e trazer o erro (antes, a
// última emissão saía com rodando=true e a tela escondia o erro).
func TestUltimaVisaoEmitidaDaParadaEhRetomavel(t *testing.T) {
	etapas := map[Estado]Etapa{
		Verificando:     func(ctx context.Context, c *Contexto) (Estado, error) { return Extraindo, nil },
		Extraindo:       func(ctx context.Context, c *Contexto) (Estado, error) { return Classificando, nil },
		Classificando:   func(ctx context.Context, c *Contexto) (Estado, error) { return Compatibilidade, nil },
		Compatibilidade: func(ctx context.Context, c *Contexto) (Estado, error) { return Delimitando, nil },
		Delimitando: func(ctx context.Context, c *Contexto) (Estado, error) {
			return Delimitando, erros.Novo(erros.IACota)
		},
	}
	g, r, _ := novo(t, etapas)
	_, _ = g.Iniciar()
	_, _ = g.EscolherTipo(TipoPADS)
	_, _ = g.EscolherEntrada(EntradaUpload)
	_, _ = g.ReceberUpload(pdfMinimo)
	esperar(t, g, func(v *Visao) bool { return v.Retomavel })
	limite := time.Now().Add(2 * time.Second)
	for {
		r.mu.Lock()
		ult := r.visoes[len(r.visoes)-1]
		r.mu.Unlock()
		if ult.Parada && ult.Retomavel && ult.Erro != nil && ult.Erro.Codigo == "IA-005" {
			return
		}
		if time.Now().After(limite) {
			t.Fatalf("última visão emitida: parada=%v retomavel=%v erro=%+v", ult.Parada, ult.Retomavel, ult.Erro)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// A sequência das visões só cresce (a interface descarta a mais antiga quando
// a resposta de um comando chega depois dos eventos da etapa).
func TestSequenciaDaVisaoCresce(t *testing.T) {
	g, _, _ := novo(t, nil)
	v1, _ := g.Iniciar()
	v2, _ := g.EscolherTipo(TipoPADS)
	v3 := g.Visao()
	if v1 == nil || v2 == nil || v3 == nil || !(v1.Sequencia < v2.Sequencia && v2.Sequencia < v3.Sequencia) {
		t.Fatalf("sequência não crescente: %v %v %v", v1, v2, v3)
	}
}

// Visibilidade encadeada: o campo só vale se a condição dele e a do campo
// de que depende valerem; BUSCA aceita só uma das opções.
func TestCamposEncadeadosEBusca(t *testing.T) {
	cs := []Campo{
		{ID: "incluir", Tipo: CampoSimNao, Obrigatorio: true},
		{ID: "diploma", Tipo: CampoEscolha, Obrigatorio: true, Opcoes: []OpcaoVisao{{ID: "RDPM"}, {ID: "EST"}}, MostrarSe: &Condicao{Campo: "incluir", Valor: "sim"}},
		{ID: "disp", Rotulo: "Dispositivo", Tipo: CampoBusca, Obrigatorio: true, Opcoes: []OpcaoVisao{{ID: "RDPM-17-V"}}, MostrarSe: &Condicao{Campo: "diploma", Valor: "RDPM"}},
	}
	// "incluir" = não: diploma e dispositivo ocultos, mesmo com valor antigo.
	if vs, err := ValidarCampos(cs, map[string]string{"incluir": "nao", "diploma": "RDPM"}); err != nil || vs["diploma"] != "" {
		t.Fatalf("encadeado: %v %v", vs, err)
	}
	if _, err := ValidarCampos(cs, map[string]string{"incluir": "sim", "diploma": "RDPM"}); err == nil {
		t.Fatal("dispositivo obrigatório visível e vazio")
	}
	if _, err := ValidarCampos(cs, map[string]string{"incluir": "sim", "diploma": "RDPM", "disp": "RDPM-99-X"}); err == nil {
		t.Fatal("BUSCA fora das opções aceita")
	}
	if vs, err := ValidarCampos(cs, map[string]string{"incluir": "sim", "diploma": "RDPM", "disp": "RDPM-17-V"}); err != nil || vs["disp"] != "RDPM-17-V" {
		t.Fatalf("%v %v", vs, err)
	}
}

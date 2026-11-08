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
	if _, err := g.Responder("compat", "inexistente"); codigo(err) != erros.TarefaOrdemInvalida {
		t.Fatal("opção inválida deve ser rejeitada")
	}
	if _, err := g.Responder("outra", "continuar"); codigo(err) != erros.TarefaOrdemInvalida {
		t.Fatal("pergunta errada deve ser rejeitada")
	}
	// "Enviar outro PDF": volta ao upload, mantendo o tipo; dados liberados ao receber o novo PDF.
	if v, _ = g.Responder("compat", "outro"); v.Estado != RecebendoUpload || v.Tipo != TipoPADS || v.Pergunta != nil {
		t.Fatalf("%+v", v)
	}
	_, _ = g.ReceberUpload(pdfMinimo)
	if !liberado {
		t.Fatal("dados da análise anterior deveriam ser liberados")
	}
	esperar(t, g, func(v *Visao) bool { return v.Estado == AguardandoEsclarecimento })
	// "Continuar": a etapa que perguntou roda de novo e lê a resposta.
	_, _ = g.Responder("compat", "continuar")
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
	if v, _ := g.Responder("p", "c"); v.Estado != Cancelada || pastasDeSessao(t, raiz) != 0 {
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

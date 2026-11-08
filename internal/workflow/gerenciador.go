package workflow

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/session"
)

// TamanhoMaximoPDF limita o upload (autos longos de 800 páginas ficam bem abaixo).
const TamanhoMaximoPDF = 1 << 30

// NomeAutos é o nome lógico do PDF dos autos dentro da sessão.
const NomeAutos = "autos.pdf"

// Contexto é o que uma etapa do processamento recebe.
type Contexto struct {
	Sessao *session.Sessao
	Tipo   Tipo
	// Progresso informa uma métrica real (ex.: páginas extraídas). Etapas sem
	// métrica não o chamam, e a interface mostra só a indicação de atividade.
	Progresso func(atual, total int, unidade string)
}

// Etapa executa o trabalho de um estado e devolve o próximo estado.
type Etapa func(ctx context.Context, c *Contexto) (Estado, error)

// Notificador recebe cada nova visão da tarefa (nil = sem tarefa).
type Notificador func(*Visao)

type tarefa struct {
	id        string
	tipo      Tipo
	entrada   Entrada
	estado    Estado
	ref       Estado // último estado de trabalho (linha do tempo em pausa/falha)
	retorno   Estado // origem de AGUARDANDO_ESCLARECIMENTO
	historico []Estado
	progresso *Progresso
	erro      *erros.Erro
	aviso     *erros.Erro
	sessao    *session.Sessao
	cancelar  context.CancelFunc
}

// Gerenciador mantém a única tarefa da instância.
type Gerenciador struct {
	mu        sync.Mutex
	sessoes   *session.Gerenciador
	notificar Notificador
	etapas    map[Estado]Etapa
	seiPronto bool
	t         *tarefa
	trabalho  sync.WaitGroup
}

// Opcoes configura o gerenciador.
type Opcoes struct {
	Sessoes   *session.Gerenciador
	Notificar Notificador
	// Etapas registradas por estado. Estado de processamento sem etapa
	// registrada encerra a tarefa com TAR-004 (nunca simula sucesso).
	Etapas map[Estado]Etapa
	// SEIDisponivel habilita a obtenção pelo SEI (Fase 2).
	SEIDisponivel bool
}

// Novo cria o gerenciador.
func Novo(o Opcoes) *Gerenciador {
	n := o.Notificar
	if n == nil {
		n = func(*Visao) {}
	}
	return &Gerenciador{sessoes: o.Sessoes, notificar: n, etapas: o.Etapas, seiPronto: o.SEIDisponivel}
}

func novoID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---- consultas ----

// Visao devolve o retrato atual (nil se não houver tarefa).
func (g *Gerenciador) Visao() *Visao {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.visaoLocked()
}

func (g *Gerenciador) visaoLocked() *Visao {
	t := g.t
	if t == nil {
		return nil
	}
	v := &Visao{
		ID: t.id, Tipo: t.tipo, Entrada: t.entrada, Estado: t.estado,
		Descricao: descricoes[t.estado], Terminal: t.estado.Terminal(),
		Etapas: marcos(t.tipo, t.ref, t.estado), Erro: visaoErro(t.erro), Aviso: visaoErro(t.aviso),
	}
	if t.progresso != nil {
		p := *t.progresso
		v.Progresso = &p
	}
	return v
}

// Historico devolve os estados percorridos (testes e diagnóstico em memória).
func (g *Gerenciador) Historico() []Estado {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.t == nil {
		return nil
	}
	return append([]Estado(nil), g.t.historico...)
}

// ---- comandos do usuário ----

// Iniciar cria uma nova tarefa. Só uma por vez (SES-CONC-01). Uma tarefa
// terminal anterior é descartada.
func (g *Gerenciador) Iniciar() (*Visao, error) {
	g.mu.Lock()
	if g.t != nil && !g.t.estado.Terminal() {
		g.mu.Unlock()
		return nil, erros.Novo(erros.TarefaJaAtiva)
	}
	s, err := g.sessoes.Criar()
	if err != nil {
		g.mu.Unlock()
		return nil, erros.De(err)
	}
	g.t = &tarefa{id: novoID(), estado: Nova, ref: Nova, historico: []Estado{Nova}, sessao: s}
	return g.emitirUnlock()
}

// EscolherTipo é a primeira escolha (PADS ou SR) — V4 §30.1; T01.
func (g *Gerenciador) EscolherTipo(tipo Tipo) (*Visao, error) {
	if tipo != TipoPADS && tipo != TipoSR {
		return nil, erros.Novo(erros.TarefaOrdemInvalida)
	}
	g.mu.Lock()
	if err := g.mover(TipoEscolhido); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	g.t.tipo = tipo
	g.t.entrada = ""
	return g.emitirUnlock()
}

// EscolherEntrada é a segunda escolha (SEI ou upload) — só depois do tipo.
func (g *Gerenciador) EscolherEntrada(e Entrada) (*Visao, error) {
	g.mu.Lock()
	if g.t == nil {
		g.mu.Unlock()
		return nil, erros.Novo(erros.TarefaSemTarefa)
	}
	if g.t.estado != TipoEscolhido {
		g.mu.Unlock()
		return nil, erros.Novo(erros.TarefaOrdemInvalida)
	}
	var prox Estado
	switch e {
	case EntradaUpload:
		prox = RecebendoUpload
	case EntradaSEI:
		if !g.seiPronto {
			g.mu.Unlock()
			return nil, erros.Novo(erros.TarefaEtapaIndisponiv)
		}
		prox = SEIConectividade
	default:
		g.mu.Unlock()
		return nil, erros.Novo(erros.TarefaOrdemInvalida)
	}
	if err := g.mover(EntradaEscolhida); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	g.t.entrada = e
	if err := g.mover(prox); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	return g.emitirUnlock()
}

// VoltarParaEntrada retorna à escolha de entrada preservando o tipo (T03).
func (g *Gerenciador) VoltarParaEntrada() (*Visao, error) {
	g.mu.Lock()
	if err := g.mover(TipoEscolhido); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	g.t.entrada = ""
	return g.emitirUnlock()
}

// ReceberUpload recebe o PDF consolidado (T02: nenhuma credencial, nenhum
// acesso ao SEI). Arquivo inválido mantém a tarefa aguardando outro arquivo.
func (g *Gerenciador) ReceberUpload(dados []byte) (*Visao, error) {
	g.mu.Lock()
	if g.t == nil {
		g.mu.Unlock()
		return nil, erros.Novo(erros.TarefaSemTarefa)
	}
	if g.t.estado != RecebendoUpload {
		g.mu.Unlock()
		return nil, erros.Novo(erros.TarefaOrdemInvalida)
	}
	if err := conferirPDF(dados); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	if err := g.t.sessao.Gravar(NomeAutos, dados); err != nil {
		g.falharLocked(erros.De(err))
		return g.emitirUnlock()
	}
	if err := g.mover(Verificando); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	g.iniciarProcessamentoLocked()
	return g.emitirUnlock()
}

// conferirPDF faz a verificação mínima de formato (detalhes na Fase 2).
func conferirPDF(d []byte) error {
	switch {
	case len(d) == 0:
		return erros.Novo(erros.PDFVazio)
	case len(d) > TamanhoMaximoPDF:
		return erros.Novo(erros.PDFGrande)
	case !bytes.HasPrefix(bytes.TrimLeft(d[:min(len(d), 1024)], "\x00\t\r\n \xef\xbb\xbf"), []byte("%PDF-")):
		return erros.Novo(erros.PDFInvalido)
	}
	return nil
}

// Cancelar interrompe a tarefa e descarta a sessão.
func (g *Gerenciador) Cancelar() (*Visao, error) {
	g.mu.Lock()
	if g.t == nil || g.t.estado.Terminal() {
		g.mu.Unlock()
		return nil, erros.Novo(erros.TarefaSemTarefa)
	}
	if g.t.cancelar != nil {
		g.t.cancelar()
	}
	g.encerrarLocked(Cancelada, erros.Novo(erros.TarefaCancelada))
	return g.emitirUnlock()
}

// Descartar remove a tarefa terminal da memória (volta à tela inicial).
func (g *Gerenciador) Descartar() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.t != nil && !g.t.estado.Terminal() {
		return erros.Novo(erros.TarefaJaAtiva)
	}
	g.t = nil
	g.notificar(nil)
	return nil
}

// Encerrar é chamado no fechamento do app: cancela a tarefa ativa, espera
// as etapas pararem e apaga a sessão.
func (g *Gerenciador) Encerrar() {
	g.mu.Lock()
	if g.t != nil && !g.t.estado.Terminal() {
		if g.t.cancelar != nil {
			g.t.cancelar()
		}
		g.encerrarLocked(Cancelada, erros.Novo(erros.TarefaCancelada))
	}
	g.mu.Unlock()
	g.trabalho.Wait()
}

// ---- processamento ----

func (g *Gerenciador) iniciarProcessamentoLocked() {
	ctx, cancel := context.WithCancel(context.Background())
	t := g.t
	t.cancelar = cancel
	c := &Contexto{Sessao: t.sessao, Tipo: t.tipo}
	c.Progresso = func(atual, total int, unidade string) {
		g.mu.Lock()
		if g.t == t && !t.estado.Terminal() && total > 0 {
			t.progresso = &Progresso{Atual: atual, Total: total, Unidade: unidade}
			g.emitirLocked()
		}
		g.mu.Unlock()
	}
	g.trabalho.Add(1)
	go func() {
		defer g.trabalho.Done()
		g.executar(ctx, t, c)
	}()
}

// executar roda as etapas registradas em sequência até um estado que dependa
// do usuário, terminal, ou sem etapa disponível.
func (g *Gerenciador) executar(ctx context.Context, t *tarefa, c *Contexto) {
	for {
		g.mu.Lock()
		if g.t != t || t.estado.Terminal() || ctx.Err() != nil {
			g.mu.Unlock()
			return
		}
		atual := t.estado
		if atual == AguardandoRevisao || atual == AguardandoEsclarecimento {
			g.mu.Unlock()
			return
		}
		etapa, ok := g.etapas[atual]
		if !ok {
			g.falharLocked(erros.Com(erros.TarefaEtapaIndisponiv, fmt.Errorf("sem etapa para %s", atual)))
			g.emitirLocked()
			g.mu.Unlock()
			return
		}
		g.mu.Unlock()

		prox, err := executarSeguro(ctx, etapa, c)

		g.mu.Lock()
		if g.t != t || t.estado.Terminal() {
			g.mu.Unlock()
			return
		}
		if ctx.Err() != nil {
			g.mu.Unlock()
			return
		}
		if err != nil {
			g.falharLocked(erros.De(err))
			g.emitirLocked()
			g.mu.Unlock()
			return
		}
		t.progresso = nil
		if err := g.mover(prox); err != nil {
			g.falharLocked(erros.Com(erros.InternoInesperado, err))
		}
		g.emitirLocked()
		g.mu.Unlock()
	}
}

func executarSeguro(ctx context.Context, e Etapa, c *Contexto) (prox Estado, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = erros.Com(erros.InternoInesperado, fmt.Errorf("pânico na etapa: %v", r))
		}
	}()
	return e(ctx, c)
}

// ---- auxiliares (exigem g.mu) ----

var errTransicao = errors.New("transição inválida")

func (g *Gerenciador) mover(para Estado) error {
	if g.t == nil {
		return erros.Novo(erros.TarefaSemTarefa)
	}
	de := g.t.estado
	if de == AguardandoEsclarecimento {
		if para != g.t.retorno && para != Cancelada && para != Falha {
			return erros.Com(erros.TarefaOrdemInvalida, errTransicao)
		}
	} else if !Permitida(de, para) {
		return erros.Com(erros.TarefaOrdemInvalida, fmt.Errorf("%w: %s → %s", errTransicao, de, para))
	}
	if para == AguardandoEsclarecimento {
		g.t.retorno = de
	}
	g.t.estado = para
	if para != AguardandoEsclarecimento && !para.Terminal() {
		g.t.ref = para
	}
	g.t.historico = append(g.t.historico, para)
	return nil
}

func (g *Gerenciador) falharLocked(e *erros.Erro) {
	if g.t.cancelar != nil {
		g.t.cancelar()
	}
	g.encerrarLocked(Falha, e)
}

// encerrarLocked leva a tarefa a um estado terminal e apaga a sessão.
func (g *Gerenciador) encerrarLocked(estado Estado, e *erros.Erro) {
	t := g.t
	t.estado = estado
	t.historico = append(t.historico, estado)
	t.erro = e
	t.progresso = nil
	if t.sessao != nil {
		if err := t.sessao.Encerrar(); err != nil {
			t.aviso = erros.De(err)
		}
	}
}

func (g *Gerenciador) emitirLocked() { g.notificar(g.visaoLocked()) }

func (g *Gerenciador) emitirUnlock() (*Visao, error) {
	v := g.visaoLocked()
	g.notificar(v)
	g.mu.Unlock()
	return v, nil
}

package pipeline

import (
	"context"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/pads"
	"github.com/DiogoBrazil/sys-executor/internal/domain/sr"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// FonteMotor prepara o motor de IA da tarefa (escolhe o modelo e sobe o
// sidecar). Erros do catálogo MOD-00x são recuperáveis: a tarefa pára com o
// aviso e o usuário tenta de novo depois de instalar ou trocar o modelo.
type FonteMotor interface {
	Abrir(ctx context.Context, atividade func(string)) (inference.Motor, func(), error)
}

// EstadoIA guarda, em memória, o que as etapas de IA produziram.
type EstadoIA struct {
	motor       inference.Motor
	fecharMotor func()
	bundle      *policy.Bundle
	reg         analise.Registro
	dag         *DAG
	classificou bool

	DelimPADS  *analise.PropostaDelimitacaoPADS
	Conjunto   *pads.ImputacaoSet
	Julgamento *pads.Julgamento
	Acusado    string

	DelimSR *analise.PropostaObjetoSR
	Objeto  *sr.ObjetoSR
	nomes   map[string]string // pessoa ID → nome (SR)

	Extracoes  map[string]analise.Extracao // por peça
	Pontos     []analise.Ponto             // todos os pontos conferidos + descrições de mídia do usuário
	Analises   map[string]analise.AnaliseImputacao
	AnalisesSR map[string]analise.AnaliseFatoPessoa
	// Condutas diversas do PADS: só registro interno (I-05), nunca na minuta.
	CondutasDiversas   []analise.Achado
	autorizacoesFeitas map[string]string // chave do achado → ID da autorização
	Pendencias         []string
}

func novoEstadoIA() *EstadoIA {
	g := NovoDAG()
	// Nós e dependências (perguntas e resultados calculados).
	g.Depende("acusado", "delimitacao")
	g.Depende("extracao", "delimitacao")
	g.Depende("autorizacao:*", "extracao")
	g.Depende("midias", "extracao")
	g.Depende("analise", "extracao", "midias", "autorizacao:*")
	g.Depende("resultado:*", "analise")
	g.Depende("conclusao:*", "analise")
	g.Depende("conexao", "resultado:*")
	g.Depende("reincidencia:*", "resultado:*", "conexao")
	g.Depende("servicos", "resultado:*")
	g.Depende("historico", "comportamento")
	return &EstadoIA{dag: g, Extracoes: map[string]analise.Extracao{}, Analises: map[string]analise.AnaliseImputacao{},
		AnalisesSR: map[string]analise.AnaliseFatoPessoa{}, autorizacoesFeitas: map[string]string{}, nomes: map[string]string{}}
}

func (e *EstadoIA) liberar() {
	if e != nil && e.fecharMotor != nil {
		e.fecharMotor()
		e.fecharMotor, e.motor = nil, nil
	}
}

// no converte o ID da pergunta no nó do DAG.
func no(perguntaID string) string {
	switch {
	case perguntaID == perguntaDelimitacao || perguntaID == perguntaObjeto:
		return "delimitacao"
	}
	return perguntaID
}

// Revisar implementa workflow.Revisavel: invalida a decisão e os
// descendentes, limpa os resultados calculados afetados e indica onde
// recomeçar.
func (d *Dados) Revisar(perguntaID string) (workflow.Estado, []string, bool) {
	e := d.IA
	if e == nil {
		return "", nil, false
	}
	n := no(perguntaID)
	afetados := append([]string{n}, e.dag.Descendentes(n)...)
	var descartar []string
	recomecar := workflow.Analisando
	for _, a := range afetados {
		switch {
		case a == "delimitacao":
			recomecar = workflow.Delimitando
			e.Conjunto, e.Julgamento, e.Objeto, e.Acusado = nil, nil, nil, ""
			descartar = append(descartar, perguntaDelimitacao, perguntaObjeto, "acusado")
		case a == "extracao":
			e.Extracoes, e.Pontos, e.CondutasDiversas = map[string]analise.Extracao{}, nil, nil
			e.autorizacoesFeitas = map[string]string{}
		case a == "analise":
			e.Analises, e.AnalisesSR = map[string]analise.AnaliseImputacao{}, map[string]analise.AnaliseFatoPessoa{}
		case a == "midias":
			e.Pontos = semDescricoesDoUsuario(e.Pontos)
			descartar = append(descartar, a)
		default:
			descartar = append(descartar, a)
		}
	}
	// Perguntas com prefixo (resultado:IMP-1…) entram por padrão.
	descartar = append(descartar, perguntasComPrefixo(d, afetados)...)
	return recomecar, descartar, true
}

func semDescricoesDoUsuario(ps []analise.Ponto) []analise.Ponto {
	var out []analise.Ponto
	for _, p := range ps {
		if p.Quem != quemUsuario {
			out = append(out, p)
		}
	}
	return out
}

// perguntasComPrefixo expande "resultado:*" para as perguntas concretas.
func perguntasComPrefixo(d *Dados, afetados []string) []string {
	var out []string
	for _, a := range afetados {
		if !strings.HasSuffix(a, ":*") {
			continue
		}
		pref := strings.TrimSuffix(a, "*")
		for _, id := range d.perguntasConhecidas() {
			if strings.HasPrefix(id, pref) {
				out = append(out, id)
			}
		}
	}
	return out
}

// perguntasConhecidas lista os IDs de pergunta que as etapas podem ter feito.
func (d *Dados) perguntasConhecidas() []string {
	e := d.IA
	var ids []string
	if e.Conjunto != nil {
		for _, id := range e.Conjunto.IDs() {
			ids = append(ids, "resultado:"+id)
		}
		for _, n := range []string{"LEVE", "MEDIA", "GRAVE"} {
			ids = append(ids, "reincidencia:"+n)
		}
	}
	if e.Objeto != nil {
		for _, f := range e.Objeto.FatosAdmitidos() {
			for _, s := range e.Objeto.Sindicados() {
				ids = append(ids, "conclusao:"+f+":"+s)
			}
		}
		for _, a := range e.Objeto.Autorizacoes() {
			ids = append(ids, "autorizacao:"+a.ID)
		}
	}
	return ids
}

// garantirMotor abre o motor uma vez por tarefa.
func (p *Pipeline) garantirMotor(ctx context.Context, c *workflow.Contexto, d *Dados) error {
	if d.IA == nil {
		d.IA = novoEstadoIA()
	}
	if d.IA.motor != nil {
		return nil
	}
	if p.IA == nil {
		return errSemIA
	}
	if d.IA.bundle == nil {
		b, err := policy.Carregar()
		if err != nil {
			return err
		}
		d.IA.bundle = b
	}
	m, fechar, err := p.IA.Abrir(ctx, c.Atividade)
	if err != nil {
		return err
	}
	d.IA.motor, d.IA.fecharMotor = m, fechar
	return nil
}

func (e *EstadoIA) executor(fluxo string) *analise.Executor {
	return &analise.Executor{Motor: e.motor, Bundle: e.bundle, Fluxo: fluxo}
}

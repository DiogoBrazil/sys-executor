package pipeline

import (
	"context"
	"errors"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/pads"
	"github.com/DiogoBrazil/sys-executor/internal/domain/sr"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/pseudonimo"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// FonteMotor prepara o motor de IA da tarefa: o provedor e o modelo
// escolhidos pelo usuário (V4 §§47, 48). Erros IA-0xx são recuperáveis: a
// tarefa pára com o aviso e o usuário tenta de novo depois de corrigir a
// configuração ou a conexão.
type FonteMotor interface {
	// Alvo confere a configuração sem rede (provedor escolhido, credencial
	// presente, modelo escolhido) e devolve o alvo, para a confirmação de
	// envio e para conferir que ele não mudou depois da autorização.
	Alvo() (inference.Alvo, error)
	// Abrir prepara o motor exatamente para o alvo autorizado. inspecionar
	// recebe o corpo exato de cada requisição antes de sair (barreira de
	// privacidade, camada 2; TRV-SEG-013): o motor não pode enviar nada que
	// ele recuse.
	Abrir(ctx context.Context, alvo inference.Alvo, atividade func(string), inspecionar func([]byte) error) (inference.Motor, func(), error)
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
	midiasAplicadas    bool              // respostas sobre mídias já convertidas em evidência/ressalva
	Pendencias         []string

	// Envio à IA (V4 §§47.3, 48.1): alvo (provedor e modelo) para o qual o
	// envio foi autorizado e mapa da pseudonimização (só memória; descartado
	// com a tarefa).
	autorizado *inference.Alvo
	mapa       *pseudonimo.Mapa
	// Barreira de privacidade (V4 §48.6): possíveis nomes da revisão feita
	// na autorização (nil = ainda não calculados) e revisão pendente surgida
	// depois dela.
	suspeitosEnvio []suspeito
	revisao        *revisaoPendente
	revisoes       int
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
	// Expande "resultado:*" etc. ANTES de limpar o estado (as perguntas
	// concretas dependem do conjunto/objeto atual).
	descartar := perguntasComPrefixo(d, afetados)
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
			e.Pendencias = semPendenciasDeMidia(e.Pendencias)
			e.midiasAplicadas = false
		case a == "analise":
			e.Analises, e.AnalisesSR = map[string]analise.AnaliseImputacao{}, map[string]analise.AnaliseFatoPessoa{}
		case a == "midias":
			e.Pontos = semDescricoesDoUsuario(e.Pontos)
			e.Pendencias = semPendenciasDeMidia(e.Pendencias)
			e.midiasAplicadas = false
			descartar = append(descartar, a)
		default:
			descartar = append(descartar, a)
		}
	}
	return recomecar, descartar, true
}

const prefixoPendMidia = "Mídia mencionada em "

func semPendenciasDeMidia(ps []string) []string {
	var out []string
	for _, p := range ps {
		if !strings.HasPrefix(p, prefixoPendMidia) {
			out = append(out, p)
		}
	}
	return out
}

// motorFalhou descarta o cliente depois de falha da Gemini API (chave,
// permissão, modelo, rede, cota…): "Tentar novamente" relê a chave e o
// modelo configurados e abre outro.
func (e *EstadoIA) motorFalhou(err error) {
	var x *erros.Erro
	if errors.As(err, &x) && strings.HasPrefix(string(x.Codigo), "IA-") {
		e.liberar()
	}
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
	if d.IA.autorizado == nil || d.IA.mapa == nil {
		// Defesa em profundidade: nunca abrir o motor sem a autorização da tarefa.
		return erros.Novo(erros.IAEnvioNaoAutorizado)
	}
	m, fechar, err := p.IA.Abrir(ctx, *d.IA.autorizado, c.Atividade, d.IA.mapa.Inspecionar)
	if err != nil {
		return err
	}
	d.IA.motor, d.IA.fecharMotor = pseudonimo.Envolver(m, d.IA.mapa), fechar
	return nil
}

func (e *EstadoIA) executor(fluxo string) *analise.Executor {
	return &analise.Executor{Motor: e.motor, Bundle: e.bundle, Fluxo: fluxo}
}

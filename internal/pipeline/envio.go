package pipeline

import (
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/pseudonimo"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// PerguntaEnvioIA é a confirmação de envio à Gemini API, feita em cada
// tarefa antes da primeira etapa de IA (V4 §47.3, TRV-SEG-010). Sem a
// autorização, nada é enviado.
const PerguntaEnvioIA = "envio_ia"

const (
	opcaoAutorizar = "autorizar"
	opcaoNaoEnviar = "nao_enviar"
	campoOcultar   = "ocultar"
	// fatorEstimativa cobre instruções das etapas, dados estruturados e
	// eventuais reenvios sobre o texto dos autos (estimativa aproximada).
	fatorEstimativa = 1.5
)

// autorizarEnvio garante a autorização da tarefa. Devolve aguardar=true
// quando a pergunta foi feita; erro recuperável quando a IA não está
// configurada ou o usuário recusou.
func (p *Pipeline) autorizarEnvio(c *workflow.Contexto, d *Dados) (aguardar bool, err error) {
	if d.IA == nil {
		d.IA = novoEstadoIA()
	}
	e := d.IA
	if e.envioAutorizado {
		return false, nil
	}
	if p.IA == nil {
		return false, errSemIA
	}
	modelo, err := p.IA.Modelo()
	if err != nil {
		return false, err // IA-001 / IA-004 / IA-012: configurar e tentar de novo
	}
	if e.mapa == nil {
		e.mapa = pseudonimo.NovoMapa()
		if d.Autos != nil {
			for _, pg := range d.Autos.Paginas {
				e.mapa.Aprender(pg.Texto)
			}
		}
	}
	resp, ok := c.RespostaCompleta(PerguntaEnvioIA)
	if !ok {
		c.Perguntar(perguntaEnvio(modelo, d, e.mapa.Resumir()))
		return true, nil
	}
	if resp.Opcao != opcaoAutorizar {
		// Recusa: nada é enviado; "Tentar novamente" pergunta de novo.
		c.Esquecer(PerguntaEnvioIA)
		return false, erros.Novo(erros.IAEnvioNaoAutorizado)
	}
	var extras []string
	for _, l := range strings.Split(resp.Valor(campoOcultar), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			extras = append(extras, l)
		}
	}
	e.mapa.AcrescentarNomes(extras)
	e.envioAutorizado = true
	return false, nil
}

func perguntaEnvio(modelo string, d *Dados, r pseudonimo.Resumo) workflow.Pergunta {
	paginas, tokens := 0, 0
	if d.Autos != nil {
		paginas = len(d.Autos.Paginas)
		for _, pg := range d.Autos.Paginas {
			tokens += inference.EstimarTokens(pg.Texto)
		}
	}
	est := int(float64(tokens)*fatorEstimativa/1000 + 0.5)
	if est < 1 {
		est = 1
	}
	ctx := fmt.Sprintf("Para analisar os autos, o sys-executor enviará à Gemini API (Google), modelo %s, trechos do texto extraído "+
		"das %d peça(s) identificadas (%d página(s)). Estimativa aproximada do envio: %d mil tokens de entrada no total, "+
		"somadas as instruções de cada etapa. O arquivo PDF não é enviado.\n\n"+
		"Antes do envio, os dados pessoais reconhecidos (%s) serão trocados por marcadores, e a resposta volta com os dados originais. "+
		"É um melhor esforço: nomes sem posto, papel processual ou rótulo podem não ser reconhecidos. "+
		"Se quiser ocultar outros nomes ou números, informe-os abaixo, um por linha.\n\n"+
		"Sem a sua autorização, nada é enviado. Esta confirmação não substitui eventual autorização institucional "+
		"para tratar os dados em serviço externo.",
		modelo, len(d.Pecas), paginas, est, r.Texto())
	return workflow.Pergunta{
		ID: PerguntaEnvioIA, Titulo: "Autorizar o envio à Gemini API", Contexto: ctx,
		Campos: []workflow.Campo{{ID: campoOcultar, Rotulo: "Outros nomes ou dados a ocultar (opcional)",
			Tipo: workflow.CampoTextoLongo, Ajuda: "Um por linha. Ex.: o nome de uma testemunha citada sem posto ou papel."}},
		Opcoes: []workflow.Opcao{
			{ID: opcaoAutorizar, Rotulo: "Autorizar o envio"},
			{ID: opcaoNaoEnviar, Rotulo: "Não enviar"},
		},
		Resumir: func(rs workflow.Resposta) string {
			if rs.Opcao == opcaoAutorizar {
				return "envio à Gemini API autorizado"
			}
			return "envio não autorizado"
		},
	}
}

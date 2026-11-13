package pipeline

import (
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/pseudonimo"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// PerguntaEnvioIA é a confirmação de envio à IA, feita em cada tarefa
// antes da primeira etapa de IA (V4 §47.3, TRV-SEG-010). Vale para um
// provedor e um modelo (V4 §48.1): se o usuário trocar qualquer um deles
// depois, a pergunta é feita de novo. Sem a autorização, nada é enviado.
const PerguntaEnvioIA = "envio_ia"

const (
	opcaoAutorizar = "autorizar"
	opcaoNaoEnviar = "nao_enviar"
	campoOcultar   = "ocultar"
	// campoCrianca: só no ChatGPT (V4 §48.6; Termos do Sign in with ChatGPT).
	campoCrianca = "crianca"
)

// autorizarEnvio garante a autorização da tarefa. Devolve aguardar=true
// quando a pergunta foi feita; erro recuperável quando a IA não está
// configurada ou o usuário recusou.
func (p *Pipeline) autorizarEnvio(c *workflow.Contexto, d *Dados) (aguardar bool, err error) {
	if d.IA == nil {
		d.IA = novoEstadoIA()
	}
	e := d.IA
	if p.IA == nil {
		return false, errSemIA
	}
	alvo, err := p.IA.Alvo()
	if err != nil {
		return false, err // IA-001 / IA-004 / IA-012 / IA-015: configurar e tentar de novo
	}
	if alvo.Provedor == inference.ProvedorChatGPT && e.criancaSemChatGPT {
		// Declarado na tarefa: dados de criança não vão ao ChatGPT.
		return false, erros.Novo(erros.IACriancaChatGPT)
	}
	if e.autorizado != nil {
		if *e.autorizado == alvo {
			return false, nil
		}
		// Provedor ou modelo mudou depois da autorização: o consentimento
		// dado a um não vale para o outro (V4 §48.1). Nada é enviado ao novo
		// alvo sem nova confirmação.
		e.liberar()
		e.autorizado = nil
		c.Esquecer(PerguntaEnvioIA)
	}
	if e.mapa == nil {
		e.mapa = pseudonimo.NovoMapa()
		// Envolvidos do cadastro primeiro (com as outras grafias); a
		// detecção nas páginas completa o mapa.
		if semearMapa(e.mapa, c.Cadastro()) {
			e.criancaSemChatGPT = true // marcado no cadastro (V4 §48.6)
		}
		if d.Autos != nil {
			for _, pg := range d.Autos.Paginas {
				e.mapa.Aprender(pg.Texto)
			}
		}
	}
	if alvo.Provedor == inference.ProvedorChatGPT && e.criancaSemChatGPT {
		return false, erros.Novo(erros.IACriancaChatGPT)
	}
	if e.suspeitosEnvio == nil {
		// Revisão item a item dos possíveis nomes sem âncora (TRV-SEG-013).
		e.suspeitosEnvio = suspeitosDasPaginas(d, e.mapa)
	}
	resp, ok := c.RespostaCompleta(PerguntaEnvioIA)
	if !ok {
		c.Perguntar(perguntaEnvio(alvo, d, e.mapa.Resumir(), e.suspeitosEnvio))
		return true, nil
	}
	if resp.Opcao != opcaoAutorizar {
		// Recusa: nada é enviado; "Tentar novamente" pergunta de novo.
		c.Esquecer(PerguntaEnvioIA)
		return false, erros.Novo(erros.IAEnvioNaoAutorizado)
	}
	if alvo.Provedor == inference.ProvedorChatGPT && resp.Valor(campoCrianca) != "nao" {
		// Sem a resposta "não", o ChatGPT fica bloqueado nesta tarefa; a
		// análise só segue com outro serviço escolhido pelo usuário.
		e.criancaSemChatGPT = true
		c.Esquecer(PerguntaEnvioIA)
		return false, erros.Novo(erros.IACriancaChatGPT)
	}
	aplicarDecisoes(e.mapa, e.suspeitosEnvio, resp)
	var extras []string
	for _, l := range strings.Split(resp.Valor(campoOcultar), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			extras = append(extras, l)
		}
	}
	e.mapa.AcrescentarNomes(extras)
	a := alvo
	e.autorizado = &a
	return false, nil
}

func perguntaEnvio(alvo inference.Alvo, d *Dados, r pseudonimo.Resumo, ss []suspeito) workflow.Pergunta {
	paginas := 0
	if d.Autos != nil {
		paginas = len(d.Autos.Paginas)
	}
	// O serviço, o modelo e a estimativa de tokens não aparecem aqui (V4 §50,
	// Emenda 4): o serviço fica em Configurações do assistente. O vínculo da
	// autorização a ele continua (alvo).
	ctx := fmt.Sprintf("Para analisar os autos, o JURISPM enviará ao assistente trechos do texto extraído "+
		"das %d peça(s) identificadas (%d página(s)). O arquivo PDF não é enviado.\n\n"+
		"Antes do envio, os dados pessoais reconhecidos (%s) serão trocados por marcadores, e a resposta volta com os dados originais. "+
		"É um melhor esforço: nomes sem posto, papel processual ou rótulo podem não ser reconhecidos. "+
		"Se quiser ocultar outros nomes ou números, informe-os abaixo, um por linha.\n\n"+
		"Sem a sua autorização, nada é enviado. Esta confirmação não substitui eventual autorização institucional "+
		"para tratar os dados em serviço externo.",
		len(d.Pecas), paginas, r.Texto())
	gs, cs := camposRevisao(ss)
	if alvo.Provedor == inference.ProvedorChatGPT {
		cs = append(cs, workflow.Campo{ID: campoCrianca, Tipo: workflow.CampoSimNao, Obrigatorio: true,
			Rotulo: "Os autos contêm dados pessoais de criança menor de 13 anos?",
			Ajuda:  "Os termos do serviço configurado para o assistente proíbem enviar esses dados, mesmo com os nomes ocultados. Se houver, esse serviço fica bloqueado nesta tarefa: escolha outro em Configurações do assistente."})
	}
	if len(ss) > 0 {
		ctx += fmt.Sprintf("\n\nHá %d possível(is) nome(s) não reconhecido(s) abaixo: decida cada um antes de autorizar.", len(ss))
	}
	return workflow.Pergunta{
		ID: PerguntaEnvioIA, Titulo: "Autorizar o envio ao assistente", Contexto: ctx,
		Grupos: gs,
		Campos: append(cs, workflow.Campo{ID: campoOcultar, Rotulo: "Outros nomes ou dados a ocultar (opcional)",
			Tipo: workflow.CampoTextoLongo, Ajuda: "Um por linha. Ex.: o nome de uma testemunha citada sem posto ou papel."}),
		Opcoes: []workflow.Opcao{
			{ID: opcaoAutorizar, Rotulo: "Autorizar o envio", Validar: true},
			{ID: opcaoNaoEnviar, Rotulo: "Não enviar"},
		},
		Resumir: func(rs workflow.Resposta) string {
			if rs.Opcao == opcaoAutorizar {
				return "envio ao assistente autorizado"
			}
			return "envio não autorizado"
		},
	}
}

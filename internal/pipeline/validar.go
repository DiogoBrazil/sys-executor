package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/minuta"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// maxCiclosCorrecao: tentativas de correção automática por trecho antes de
// a falha virar pendência para o usuário (plano §5.4, etapa 15).
const maxCiclosCorrecao = 2

// Ajuste é um pedido de ajuste do usuário e o que foi feito com ele.
type Ajuste struct {
	Pedido    string
	Resultado string
	Aplicado  bool
}

func assinatura(ps []analise.Paragrafo) string {
	h := sha256.New()
	for _, p := range ps {
		h.Write([]byte(p.Texto))
		h.Write([]byte{0})
		h.Write([]byte(strings.Join(p.Evid, ",")))
		h.Write([]byte{1})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func inicioDe(texto string) string {
	r := []rune(strings.TrimSpace(texto))
	if len(r) > 60 {
		return string(r[:60]) + "…"
	}
	return string(r)
}

// validar é a etapa 14/15 (V4 §42): a IA confere se as provas citadas
// sustentam cada trecho redigido; o que não se sustenta é reescrito (até
// maxCiclosCorrecao vezes) e, se persistir, vira pendência visível. Depois o
// aplicativo confere a minuta inteira contra as decisões e o cálculo.
func (p *Pipeline) validar(ctx context.Context, c *workflow.Contexto) (workflow.Estado, error) {
	d := dados(c)
	if !prontoParaRedigir(c, d) || d.Redacao == nil {
		return workflow.Validando, erros.Novo(erros.TarefaEtapaIndisponiv)
	}
	if aguardar, err := p.prepararIA(ctx, c, d); err != nil {
		return workflow.Validando, err
	} else if aguardar {
		return workflow.AguardandoEsclarecimento, nil
	}
	rd, r := d.Redacao, p.novoRedator(d)
	for i, slot := range r.ordem {
		desc := r.descricao(slot)
		c.Atividade("Conferindo: " + desc + ".")
		c.Progresso(i, len(r.ordem), unidadeDeProgresso)
		for {
			ps := r.paragrafos(slot)
			ass := assinatura(ps)
			if len(ps) == 0 || rd.Verificado[slot] == ass {
				break
			}
			vs, err := analise.VerificarSuporte(ctx, r.x, desc, ps, r.pontos(slot), r.resultado(slot))
			if analise.IsSemSeguranca(err) {
				rd.PendValidacao[slot] = []string{"A conferência automática não pôde ser concluída no trecho " + desc + ": confira-o com atenção redobrada."}
				rd.Verificado[slot] = ass
				break
			}
			if err != nil {
				d.IA.motorFalhou(err)
				return workflow.Validando, err
			}
			var problemas, pend []string
			for _, v := range vs {
				if v.Suporte == analise.Suporta && !v.Contradiz {
					continue
				}
				motivo := v.Motivo
				if v.Contradiz {
					motivo = "o texto contraria o resultado que você confirmou. " + motivo
				}
				pend = append(pend, fmt.Sprintf("Confira no trecho %s o parágrafo que começa com “%s”: %s", desc, inicioDe(ps[v.Indice].Texto), strings.TrimSpace(motivo)))
				if v.Problema() {
					problemas = append(problemas, fmt.Sprintf("parágrafo %d (“%s”): %s", v.Indice+1, inicioDe(ps[v.Indice].Texto), strings.TrimSpace(motivo)))
				}
			}
			if len(problemas) == 0 || rd.Ciclos[slot] >= maxCiclosCorrecao {
				rd.PendValidacao[slot], rd.Verificado[slot] = pend, ass
				break
			}
			rd.Ciclos[slot]++
			c.Atividade("Corrigindo: " + desc + ".")
			mantido, err := r.redigir(ctx, slot, &analise.Reescrita{Anterior: r.textoAtual(slot), Problemas: problemas})
			if err != nil {
				d.IA.motorFalhou(err)
				return workflow.Validando, err
			}
			if mantido {
				rd.PendValidacao[slot], rd.Verificado[slot] = pend, ass
				break
			}
		}
	}
	c.Progresso(len(r.ordem), len(r.ordem), unidadeDeProgresso)

	evid := map[string]bool{}
	for _, pt := range d.IA.Pontos {
		evid[pt.ID] = true
	}
	doc := r.compor()
	if r.sr {
		d.Achados = minuta.ValidarSR(doc, *d.entradaMinutaSR, d.IA.Objeto, evid)
	} else {
		d.Achados = minuta.ValidarPADS(doc, *d.entradaMinuta, evid)
	}
	for _, a := range d.Achados {
		doc.Pendencia("Validação automática: " + a.Mensagem + ".")
	}
	if t := doc.Texto(); t != d.textoVersao {
		d.textoVersao = t
		d.versaoMinuta++
	}
	c.PublicarMinuta(visaoMinuta(d, len(d.Achados) == 0))
	if len(d.Achados) > 0 {
		// A minuta vai à revisão com o alerta e as pendências à vista (V4 §42.2).
		return workflow.AguardandoRevisao, erros.Novo(erros.ValidacaoBloqueou)
	}
	return workflow.AguardandoRevisao, nil
}

// ajustar aplica um pedido de ajuste do usuário (V4 §31.3): a IA diz o que
// o pedido atinge e reescreve só esses trechos; o resto fica como está.
// Decisões confirmadas e textos fixos não são alterados por aqui.
func (p *Pipeline) ajustar(ctx context.Context, c *workflow.Contexto) (workflow.Estado, error) {
	d := dados(c)
	pedido := strings.TrimSpace(c.Ajuste())
	if pedido == "" || d.Redacao == nil || !prontoParaRedigir(c, d) {
		c.ConcluirAjuste()
		return workflow.Validando, nil
	}
	if aguardar, err := p.prepararIA(ctx, c, d); err != nil {
		return workflow.Ajustando, err
	} else if aguardar {
		return workflow.AguardandoEsclarecimento, nil
	}
	r := p.novoRedator(d)
	var ts []analise.TrechoAjustavel
	for _, slot := range r.ordem {
		ts = append(ts, analise.TrechoAjustavel{ID: slot, Descricao: r.descricao(slot)})
	}
	c.Atividade("Lendo o pedido de ajuste.")
	cl, err := analise.ClassificarAjuste(ctx, r.x, pedido, ts)
	if err != nil {
		d.IA.motorFalhou(err)
		return workflow.Ajustando, err
	}
	reg := Ajuste{Pedido: pedido}
	switch cl.Tipo {
	case analise.AjusteRedacao:
		if d.ajusteAplicado == nil {
			d.ajusteAplicado = map[string]bool{}
		}
		var feitos, mantidos []string
		for i, slot := range cl.Trechos {
			desc := r.descricao(slot)
			if d.ajusteAplicado[slot] { // refeito antes de uma falha: não repetir
				feitos = append(feitos, desc)
				continue
			}
			c.Atividade("Ajustando: " + desc + ".")
			c.Progresso(i, len(cl.Trechos), unidadeDeProgresso)
			mantido, err := r.redigir(ctx, slot, &analise.Reescrita{Anterior: r.textoAtual(slot), Pedido: pedido})
			if err != nil {
				d.IA.motorFalhou(err)
				return workflow.Ajustando, err
			}
			if mantido {
				mantidos = append(mantidos, desc)
				continue
			}
			d.ajusteAplicado[slot] = true
			d.Redacao.Ciclos[slot] = 0
			feitos = append(feitos, desc)
		}
		if len(feitos) > 0 {
			reg.Aplicado = true
			reg.Resultado = "Ajuste aplicado em: " + strings.Join(feitos, "; ") + ". Os demais trechos não foram alterados."
		}
		if len(mantidos) > 0 {
			reg.Resultado = strings.TrimSpace(reg.Resultado + " Não foi possível reescrever com apoio nas provas: " + strings.Join(mantidos, "; ") + ". O texto anterior foi mantido.")
		}
	case analise.AjusteDecisao:
		reg.Resultado = "O pedido altera uma decisão já confirmada (resultado, punição, dias, pontos ou circunstâncias). O assistente não muda decisões: use “Alterar” em “Suas respostas”, e a minuta é refeita em seguida."
		if r.sr {
			reg.Resultado = "O pedido altera uma decisão já confirmada (conclusão, enquadramento, fatos ou sindicados). O assistente não muda decisões: use “Alterar” em “Suas respostas”, e a minuta é refeita em seguida."
		}
	case analise.AjusteTextoFixo:
		reg.Resultado = "O pedido trata de texto padrão do aplicativo (cabeçalho, dispositivo da decisão, providências, fecho ou formatação), que o assistente não reescreve. Faça essa alteração no arquivo DOCX."
		if r.sr {
			reg.Resultado = "O pedido trata de texto composto pelo aplicativo (cabeçalho, dados da sindicância, solução, fecho ou formatação), que o assistente não reescreve. Para mudar a solução, use “Alterar” na conclusão ou no enquadramento; o restante, faça no arquivo DOCX."
		}
	default:
		reg.Resultado = "Não foi possível identificar o que ajustar. Diga em qual parte (relatório, fundamentação de qual imputação, exame do art. 36) e o que deve mudar."
		if r.sr {
			reg.Resultado = "Não foi possível identificar o que ajustar. Diga em qual parte (síntese dos fatos ou análise das provas de qual fato) e o que deve mudar."
		}
	}
	d.Ajustes = append(d.Ajustes, reg)
	d.ajusteAplicado = nil
	c.ConcluirAjuste()
	return workflow.Validando, nil
}

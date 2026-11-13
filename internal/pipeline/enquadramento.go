package pipeline

import (
	"fmt"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/rules/engine"
	"github.com/DiogoBrazil/sys-executor/internal/rules/enquadramento"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// Enquadramento manual no formulário PADS-ESC-001 (V4 §48.5, PADS-ENQ-001):
// a IA propõe conduta e data a partir do Memorando; o dispositivo e a
// natureza vêm do usuário, pelo catálogo curado. O dispositivo citado no
// Memorando aparece só para conferência — nunca é pré-preenchido nem
// sugerido. Escolha diferente da citada (ou imputação sem citação
// correspondente) exige confirmação expressa, com a remissão à V4 §22.1.

const (
	campoDiploma     = "diploma_"
	campoDispRDPM    = "disp_rdpm_"
	campoDispEst     = "disp_est_"
	campoNatureza    = "natureza_"
	campoConfirmaDiv = "confirma_div_"
)

var opcoesDiploma = []workflow.OpcaoVisao{
	{ID: enquadramento.RDPM, Rotulo: "RDPM (arts. 15 a 17)"},
	{ID: enquadramento.Estatuto, Rotulo: "Estatuto (arts. 29 e 32)"},
}

func opcoesDispositivos(diploma string) []workflow.OpcaoVisao {
	var out []workflow.OpcaoVisao
	for _, d := range enquadramento.DoDiploma(diploma) {
		r := d.Rotulo()
		if n, ok := d.NaturezaFixa(); ok {
			r += " (" + rotuloNatureza[string(n)] + ")"
		}
		out = append(out, workflow.OpcaoVisao{ID: d.ID, Rotulo: r})
	}
	return out
}

// camposEnquadramento: diploma → dispositivo (com filtro) → natureza (só no
// Estatuto). Valores anteriores (revisão) são mantidos; nada vem da IA.
func camposEnquadramento(id string, val func(campo, padrao string) string, sim *workflow.Condicao) []workflow.Campo {
	ehRDPM := &workflow.Condicao{Campo: campoDiploma + id, Valor: enquadramento.RDPM}
	ehEst := &workflow.Condicao{Campo: campoDiploma + id, Valor: enquadramento.Estatuto}
	return []workflow.Campo{
		{ID: campoDiploma + id, Rotulo: "Enquadramento: diploma", Tipo: workflow.CampoEscolha, Opcoes: opcoesDiploma,
			Obrigatorio: true, Grupo: id, Valor: val(campoDiploma+id, ""), MostrarSe: sim,
			Ajuda: "Escolha você o enquadramento; o aplicativo não sugere dispositivo."},
		{ID: campoDispRDPM + id, Rotulo: "Dispositivo do RDPM", Tipo: workflow.CampoBusca, Opcoes: opcoesDispositivos(enquadramento.RDPM),
			Obrigatorio: true, Grupo: id, Valor: val(campoDispRDPM+id, ""), MostrarSe: ehRDPM,
			Ajuda: "Digite o artigo, o inciso ou parte do texto. A natureza é a do artigo: 15 leve, 16 média, 17 grave."},
		{ID: campoDispEst + id, Rotulo: "Dispositivo do Estatuto", Tipo: workflow.CampoBusca, Opcoes: opcoesDispositivos(enquadramento.Estatuto),
			Obrigatorio: true, Grupo: id, Valor: val(campoDispEst+id, ""), MostrarSe: ehEst,
			Ajuda: "Digite o artigo, o inciso ou parte do texto."},
		{ID: campoNatureza + id, Rotulo: "Natureza atribuída", Tipo: workflow.CampoEscolha, Opcoes: opcoesNatureza,
			Obrigatorio: true, Grupo: id, Valor: val(campoNatureza+id, ""), MostrarSe: ehEst,
			Ajuda: "O Estatuto não define a natureza: atribua-a pela semelhança de intensidade com as transgressões do RDPM."},
	}
}

// escolhido devolve o dispositivo e a natureza informados para a imputação.
func escolhido(resp workflow.Resposta, id string) (enquadramento.Dispositivo, engine.Natureza, bool) {
	var dispID string
	switch resp.Valor(campoDiploma + id) {
	case enquadramento.RDPM:
		dispID = resp.Valor(campoDispRDPM + id)
	case enquadramento.Estatuto:
		dispID = resp.Valor(campoDispEst + id)
	}
	d, ok := enquadramento.PorID(dispID)
	if !ok || d.Diploma != resp.Valor(campoDiploma+id) {
		return enquadramento.Dispositivo{}, "", false
	}
	if n, fixa := d.NaturezaFixa(); fixa {
		return d, n, true
	}
	n := engine.Natureza(resp.Valor(campoNatureza + id))
	if _, ok := naturezaValida[n]; !ok {
		return enquadramento.Dispositivo{}, "", false
	}
	return d, n, true
}

var naturezaValida = map[engine.Natureza]bool{engine.Leve: true, engine.Media: true, engine.Grave: true}

// citadoNoMemorando: o dispositivo que a IA leu no Memorando para a
// imputação proposta (só conferência).
func citadoNoMemorando(im analise.ImputacaoProposta) (enquadramento.Dispositivo, bool) {
	return enquadramento.DoCitado(im.Diploma, im.Artigo, im.Inciso, im.Alinea)
}

// descricaoCitado: linha informativa do grupo (o citado, nunca proposto).
func descricaoCitado(im analise.ImputacaoProposta) string {
	if ref := im.Dispositivo(); ref != "" {
		return "Dispositivo citado no Memorando (para conferência): " + ref + "."
	}
	return "O Memorando não cita dispositivo para esta conduta (segundo a leitura do assistente)."
}

// divergencias confere a escolha de cada imputação incluída contra o
// dispositivo citado no Memorando. Devolve, por grupo, o texto do alerta de
// toda escolha divergente (o campo de confirmação acompanha todas, para a
// confirmação já dada não se perder) e quantas ainda não foram confirmadas.
func divergencias(r analise.PropostaDelimitacaoPADS, resp workflow.Resposta) (map[string]string, int) {
	out := map[string]string{}
	pendentes := 0
	citados := map[string]analise.ImputacaoProposta{}
	for _, im := range r.Imputacoes {
		citados[im.ID] = im
	}
	for _, id := range idsDelimitacao(r, extrasDe(resp.Valores)) {
		if resp.Valor("incluir_"+id) != "sim" {
			continue
		}
		d, nat, ok := escolhido(resp, id)
		if !ok {
			continue // a validação dos campos já recusa
		}
		im, proposta := citados[id]
		c, temCitado := citadoNoMemorando(im)
		var msg string
		switch {
		case !proposta:
			msg = fmt.Sprintf("Esta imputação não foi identificada pelo assistente no Memorando. Confira no Memorando se ela consta da acusação: "+
				"o PADS só julga as imputações do Memorando Disciplinar (V4 §22.1). Escolhido: %s.", d.Referencia())
		case !temCitado && im.Dispositivo() != "":
			msg = fmt.Sprintf("O Memorando cita %s, que não está no catálogo (RDPM arts. 15 a 17; Estatuto arts. 29 e 32). "+
				"Escolhido: %s. Confirme que o enquadramento corresponde à acusação do Memorando (V4 §22.1).", im.Dispositivo(), d.Referencia())
		case !temCitado:
			continue // o Memorando não cita dispositivo: vale a escolha do usuário
		case c.ID != d.ID:
			msg = fmt.Sprintf("O dispositivo escolhido (%s) é diferente do citado no Memorando (%s). "+
				"O julgamento fica vinculado às acusações do Memorando: não se troca o dispositivo para transformar uma acusação em outra (V4 §22.1). "+
				"Confirme só se a escolha corrige o enquadramento da mesma conduta acusada.", d.Referencia(), c.Referencia())
		default:
			continue
		}
		if string(im.Natureza) != "" && im.Natureza != nat {
			msg += fmt.Sprintf(" Observação: o Memorando declara natureza %s; a do enquadramento é %s.",
				rotuloNatureza[string(im.Natureza)], rotuloNatureza[string(nat)])
		}
		out[id] = msg
		if resp.Valor(campoConfirmaDiv+id) != "sim" {
			pendentes++
		}
	}
	return out, pendentes
}

// idsDelimitacao: as imputações propostas pela IA e as extras (as do
// cadastro sem correspondência no Memorando, ou a vaga em branco).
func idsDelimitacao(r analise.PropostaDelimitacaoPADS, extras int) []string {
	var ids []string
	for _, im := range r.Imputacoes {
		ids = append(ids, im.ID)
	}
	for k := 1; k <= extras; k++ {
		ids = append(ids, fmt.Sprintf("IMP-X%d", k))
	}
	return ids
}

func idExtra(k int) string { return fmt.Sprintf("IMP-X%d", k) }

// extrasDe conta as imputações extras presentes na resposta (no mínimo
// maxExtras).
func extrasDe(vs map[string]string) int {
	n := maxExtras
	for k := maxExtras + 1; ; k++ {
		if _, ok := vs["incluir_"+idExtra(k)]; !ok {
			return n
		}
		n = k
	}
}

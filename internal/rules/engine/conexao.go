package engine

import "github.com/DiogoBrazil/sys-executor/internal/rules/civil"

// ImputacaoProcedente é uma imputação do memorando julgada procedente.
type ImputacaoProcedente struct {
	ID       string
	Natureza Natureza
	Fato     civil.Instante
	// Grupo de conexão (V4 §5.1): imputações com o mesmo grupo não vazio são
	// conexas entre si; vazio = independente.
	Grupo string
}

// ResultadoConexao separa principais (que recebem sanção) e secundárias
// (absorvidas como agravante, sem sanção autônoma nem dupla contagem).
type ResultadoConexao struct {
	Sancoes     []SancaoAplicada
	Principais  []ImputacaoProcedente
	Secundarias map[string][]string // ID da principal → secundárias
	Pendencias  []Pendencia
}

// SugerirPrincipal indica a de maior gravidade (I-04: só sugestão; o usuário
// confirma). Empate de gravidade → sem sugestão.
func SugerirPrincipal(grupo []ImputacaoProcedente) (id string, empate bool) {
	melhor := -1
	for _, im := range grupo {
		u := im.Natureza.Unidades()
		switch {
		case u > melhor:
			melhor, id, empate = u, im.ID, false
		case u == melhor:
			empate = true
		}
	}
	if empate {
		return "", true
	}
	return id, false
}

// AplicarConexao monta as sanções do PADS: cada independente recebe a sua
// (V4 §5.2); em cada grupo conexo, só a principal CONFIRMADA pelo usuário
// (V4 §5.1; I-04). Sem confirmação → pendência e nenhuma sanção do grupo.
func AplicarConexao(imps []ImputacaoProcedente, principalConfirmada map[string]string) ResultadoConexao {
	r := ResultadoConexao{Secundarias: map[string][]string{}}
	grupos := map[string][]ImputacaoProcedente{}
	var ordem []string
	for _, im := range imps {
		if im.Grupo == "" {
			r.Principais = append(r.Principais, im)
			r.Sancoes = append(r.Sancoes, SancaoAplicada{im.ID, im.Natureza, im.Natureza.Sancao()})
			continue
		}
		if _, ok := grupos[im.Grupo]; !ok {
			ordem = append(ordem, im.Grupo)
		}
		grupos[im.Grupo] = append(grupos[im.Grupo], im)
	}
	for _, g := range ordem {
		membros := grupos[g]
		if len(membros) == 1 {
			im := membros[0]
			r.Principais = append(r.Principais, im)
			r.Sancoes = append(r.Sancoes, SancaoAplicada{im.ID, im.Natureza, im.Natureza.Sancao()})
			continue
		}
		escolhida := principalConfirmada[g]
		var principal *ImputacaoProcedente
		for i := range membros {
			if membros[i].ID == escolhida {
				principal = &membros[i]
			}
		}
		if principal == nil {
			sug, empate := SugerirPrincipal(membros)
			msg := "Confirme qual transgressão conexa é a principal"
			if !empate {
				msg += " (sugestão: " + sug + ", a de maior gravidade)"
			}
			r.Pendencias = append(r.Pendencias, pend(PendPrincipalConexao, g, "%s.", msg))
			continue
		}
		r.Principais = append(r.Principais, *principal)
		r.Sancoes = append(r.Sancoes, SancaoAplicada{principal.ID, principal.Natureza, principal.Natureza.Sancao()})
		for _, m := range membros {
			if m.ID != principal.ID {
				r.Secundarias[principal.ID] = append(r.Secundarias[principal.ID], m.ID)
			}
		}
	}
	return r
}

// FatosAtuais converte as principais em transgressões atuais para os
// gatilhos de comportamento (secundárias conexas não entram: sem dupla
// contagem, V4 §8.7).
func (r ResultadoConexao) FatosAtuais() []Transgressao {
	var out []Transgressao
	for _, p := range r.Principais {
		out = append(out, Transgressao{ID: p.ID, Natureza: p.Natureza, Data: p.Fato.Data, Origem: Atual})
	}
	return out
}

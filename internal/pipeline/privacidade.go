package pipeline

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/pseudonimo"
	"github.com/DiogoBrazil/sys-executor/internal/texto"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// Revisão item a item dos possíveis dados pessoais não reconhecidos
// (barreira de privacidade, V4 §48.6, TRV-SEG-013). A lista mostra só o
// possível nome e onde ele aparece, nunca o texto dos autos; não há opção
// de ignorar todos de uma vez.

const (
	prefixoRevisao   = "revisao_privacidade"
	campoSuspeito    = "suspeito_"
	decOcultar       = "ocultar"
	decManter        = "manter"
	opcaoContinuar   = "continuar"
	grupoRevisao     = "revisao"
	maxFontesPorItem = 1
)

var opcoesDecisao = []workflow.OpcaoVisao{
	{ID: decOcultar, Rotulo: "É nome de pessoa ou dado pessoal: ocultar"},
	{ID: decManter, Rotulo: "Não é dado pessoal: enviar como está"},
}

// suspeito é um possível nome não reconhecido, com onde aparece nos autos.
type suspeito struct {
	Forma  string
	Fontes []workflow.Fonte
}

// suspeitosDasPaginas varre as páginas (já com os nomes conhecidos
// ocultados) e junta os possíveis nomes, sem repetir, com a 1ª ocorrência.
func suspeitosDasPaginas(d *Dados, m *pseudonimo.Mapa) []suspeito {
	out := []suspeito{}
	if d.Autos == nil {
		return out
	}
	vistos := map[string]bool{}
	for _, pg := range d.Autos.Paginas {
		for _, s := range m.Suspeitos(m.Ocultar(pg.Texto)) {
			k := texto.SemAcentos(s)
			if vistos[k] {
				continue
			}
			vistos[k] = true
			out = append(out, suspeito{Forma: s, Fontes: fontesDe(d, s)})
		}
	}
	return out
}

// fontesDe localiza o trecho nas páginas (para o usuário conferir no
// visualizador); vazio se ele não estiver nos autos (texto digitado).
func fontesDe(d *Dados, s string) []workflow.Fonte {
	if d.Autos == nil {
		return nil
	}
	// Entre as palavras, qualquer espaçamento: o item junta as palavras com
	// um espaço, e a página pode ter espaços duplos ou quebra de linha.
	ws := strings.Fields(s)
	if len(ws) == 0 {
		return nil
	}
	for i, w := range ws {
		ws[i] = regexp.QuoteMeta(w)
	}
	re := regexp.MustCompile(strings.Join(ws, `\s+`))
	var fs []workflow.Fonte
	for _, pg := range d.Autos.Paginas {
		loc := re.FindStringIndex(pg.Texto)
		if loc == nil {
			continue
		}
		ini := utf8.RuneCountInString(pg.Texto[:loc[0]])
		trecho := pg.Texto[loc[0]:loc[1]]
		fs = append(fs, workflow.Fonte{Rotulo: fmt.Sprintf("pág. %d", pg.Numero), Pagina: pg.Numero,
			Inicio: ini, Fim: ini + utf8.RuneCountInString(trecho), Trecho: trecho})
		if len(fs) == maxFontesPorItem {
			break
		}
	}
	return fs
}

// camposRevisao: um campo de decisão obrigatório por item.
func camposRevisao(ss []suspeito) ([]workflow.GrupoCampos, []workflow.Campo) {
	if len(ss) == 0 {
		return nil, nil
	}
	g := workflow.GrupoCampos{ID: grupoRevisao, Titulo: "Possíveis nomes não reconhecidos",
		Descricao: []string{"Estes trechos parecem nomes de pessoas, mas não têm posto, papel processual ou rótulo que permitam reconhecê-los com segurança. " +
			"Decida cada um: o que for ocultado vai à IA como marcador e volta com o texto original."}}
	cs := make([]workflow.Campo, len(ss))
	for i, s := range ss {
		cs[i] = workflow.Campo{ID: fmt.Sprintf("%s%d", campoSuspeito, i), Rotulo: s.Forma, Tipo: workflow.CampoEscolha,
			Opcoes: opcoesDecisao, Obrigatorio: true, Grupo: grupoRevisao, Fontes: s.Fontes}
	}
	return []workflow.GrupoCampos{g}, cs
}

// aplicarDecisoes registra no mapa o que o usuário decidiu.
func aplicarDecisoes(m *pseudonimo.Mapa, ss []suspeito, r workflow.Resposta) {
	for i, s := range ss {
		switch r.Valor(fmt.Sprintf("%s%d", campoSuspeito, i)) {
		case decOcultar:
			m.AcrescentarNomes([]string{s.Forma})
		case decManter:
			m.Liberar(s.Forma)
		}
	}
}

// revisaoPendente: possíveis nomes surgidos depois da autorização (texto
// digitado pelo usuário, histórico de reenvio), à espera de decisão.
type revisaoPendente struct {
	id    string
	itens []suspeito
}

func perguntaRevisao(r *revisaoPendente) workflow.Pergunta {
	gs, cs := camposRevisao(r.itens)
	return workflow.Pergunta{
		ID: r.id, Titulo: "Revisar possíveis dados pessoais antes do envio",
		Contexto: "O próximo envio à IA contém trechos que parecem nomes de pessoas e ainda não foram decididos. " +
			"Nada foi enviado. Decida cada item para continuar.",
		Grupos: gs, Campos: cs,
		Opcoes: []workflow.Opcao{
			{ID: opcaoContinuar, Rotulo: "Aplicar e continuar", Validar: true},
			{ID: opcaoNaoEnviar, Rotulo: "Não enviar"},
		},
		Resumir: func(rs workflow.Resposta) string {
			if rs.Opcao != opcaoContinuar {
				return "envio não autorizado"
			}
			return fmt.Sprintf("%d item(ns) revisado(s)", len(r.itens))
		},
	}
}

// comRevisao envolve as etapas que chamam a IA: quando a barreira pára o
// envio por possível nome não reconhecido (pseudonimo.ErrRevisao), pergunta
// ao usuário, aplica as decisões e refaz a etapa (o que já foi feito fica
// guardado em Dados.IA).
func (p *Pipeline) comRevisao(estado workflow.Estado, etapa workflow.Etapa) workflow.Etapa {
	return func(ctx context.Context, c *workflow.Contexto) (workflow.Estado, error) {
		d := dados(c)
		if d.IA != nil && d.IA.revisao != nil {
			r := d.IA.revisao
			resp, ok := c.RespostaCompleta(r.id)
			if !ok {
				c.Perguntar(perguntaRevisao(r))
				return workflow.AguardandoEsclarecimento, nil
			}
			d.IA.revisao = nil
			if resp.Opcao != opcaoContinuar {
				c.Esquecer(r.id)
				return estado, erros.Novo(erros.IAEnvioNaoAutorizado)
			}
			aplicarDecisoes(d.IA.mapa, r.itens, resp)
		}
		st, err := etapa(ctx, c)
		var rev pseudonimo.ErrRevisao
		if err != nil && errors.As(err, &rev) && d.IA != nil && d.IA.mapa != nil {
			d.IA.revisoes++
			r := &revisaoPendente{id: fmt.Sprintf("%s_%d", prefixoRevisao, d.IA.revisoes)}
			for _, s := range rev.Itens {
				r.itens = append(r.itens, suspeito{Forma: s, Fontes: fontesDe(d, s)})
			}
			d.IA.revisao = r
			c.Perguntar(perguntaRevisao(r))
			return workflow.AguardandoEsclarecimento, nil
		}
		return st, err
	}
}

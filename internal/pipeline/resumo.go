package pipeline

import (
	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/segment"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

var rotulos = map[segment.Funcao]string{
	segment.MemorandoDisciplinar: "Memorando Disciplinar",
	segment.PortariaSR:           "Portaria de instauração de SR",
	segment.Portaria:             "Portaria",
	segment.Notificacao:          "Notificação",
	segment.DefesaPrevia:         "Defesa prévia",
	segment.AlegacoesFinais:      "Alegações finais",
	segment.TermoInquiricao:      "Termo de inquirição",
	segment.TermoDeclaracoes:     "Termo de declarações",
	segment.Interrogatorio:       "Interrogatório",
	segment.AtaVideoconferencia:  "Ata de videoconferência",
	segment.Ata:                  "Ata",
	segment.Despacho:             "Despacho",
	segment.Oficio:               "Ofício",
	segment.Certidao:             "Certidão",
	segment.Relatorio:            "Relatório",
	segment.TermoRemessa:         "Termo de remessa",
	segment.Anexo:                "Anexo",
	segment.Capa:                 "Capa do processo",
	segment.TermoJuntada:         "Termo de juntada",
	segment.Representacao:        "Representação",
	segment.Procuracao:           "Procuração",
	segment.Informacao:           "Informação",
	segment.Ciencia:              "Ciência",
	segment.Indeterminada:        "Função não identificada",
}

// gravidade ordena as situações de leitura (a pior representa a peça).
var gravidade = map[autos.Situacao]int{
	autos.Vazia: 0, autos.Lida: 1, autos.OCROk: 2, autos.OCRBaixo: 3, autos.Ilegivel: 4,
}

// Resumir monta o resumo exibido (sem texto dos autos).
func Resumir(d *Dados) *workflow.ResumoAutos {
	r := &workflow.ResumoAutos{}
	if d.Autos == nil {
		return r
	}
	r.TotalPaginas = len(d.Autos.Paginas)
	pecaDe := make([]string, len(d.Autos.Paginas)+1)
	for _, p := range d.Pecas {
		pior := autos.Vazia
		for n := p.PaginaInicial; n <= p.PaginaFinal && n <= len(d.Autos.Paginas); n++ {
			pecaDe[n] = p.ID
			if s := d.Autos.Paginas[n-1].Situacao; gravidade[s] > gravidade[pior] {
				pior = s
			}
		}
		r.Pecas = append(r.Pecas, workflow.PecaResumo{
			ID: p.ID, Titulo: p.Titulo, Funcao: string(p.Funcao), Rotulo: rotulos[p.Funcao],
			PaginaInicial: p.PaginaInicial, PaginaFinal: p.PaginaFinal, SEI: p.SEI,
			Provavel: p.Provavel, Externo: p.Externo, Situacao: string(pior),
		})
	}
	for _, p := range d.Autos.Paginas {
		r.Paginas = append(r.Paginas, workflow.PaginaResumo{
			Numero: p.Numero, Situacao: string(p.Situacao), Origem: string(p.Origem),
			Confianca: p.Confianca, Peca: pecaDe[p.Numero],
		})
	}
	for _, m := range d.Midias {
		r.Midias = append(r.Midias, workflow.MidiaResumo{Pagina: m.Pagina, Peca: m.Peca, Termo: m.Termo, Inicio: m.Inicio, Fim: m.Fim})
	}
	r.Compatibilidade = string(d.Compat.Resultado)
	r.PecaCentral = d.Compat.PecaCentral
	return r
}

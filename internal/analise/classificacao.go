package analise

import (
	"context"
	"fmt"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/segment"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/rules/catalog"
)

// Funções que a IA pode atribuir na classificação residual. Manifestação
// (defesa prévia × alegações finais) fica com a cronologia do Go (D-09.5), e
// as peças centrais (memorando, portaria) só por título.
var funcoesIA = []segment.Funcao{
	segment.Notificacao, segment.TermoInquiricao, segment.TermoDeclaracoes, segment.Interrogatorio,
	segment.AtaVideoconferencia, segment.Ata, segment.Despacho, segment.Oficio, segment.Certidao,
	segment.Relatorio, segment.TermoRemessa, segment.TermoJuntada, segment.Representacao, segment.Procuracao,
	segment.Informacao, segment.Email, segment.Ciencia, segment.Anexo, segment.Indeterminada,
}

type saidaClassif struct {
	Funcao string `json:"funcao"`
	Trecho string `json:"trecho_literal"`
}

// ClassificarResiduais pede à IA a função das peças sem título reconhecido
// (etapa 2; TRV-ING-001). Só o início da peça é enviado. Resultado sempre
// "provável" (Base IA) e conferido por trecho literal.
func ClassificarResiduais(ctx context.Context, x *Executor, a *autos.Autos, pecas []segment.Peca, reg *Registro) ([]segment.Peca, []string, error) {
	out := append([]segment.Peca(nil), pecas...)
	var pend []string
	opcoes := make([]string, len(funcoesIA))
	for i, f := range funcoesIA {
		opcoes[i] = string(f)
	}
	schema := objeto(map[string]any{"funcao": enum(opcoes...), "trecho_literal": str(300)})
	for i, p := range out {
		if p.Funcao != segment.Indeterminada || p.Base != segment.BaseNenhuma {
			continue
		}
		bs := Blocos(a, p, 2400, reg)
		if len(bs) == 0 {
			continue
		}
		b := bs[0]
		var f segment.Funcao
		err := x.Chamar(ctx, Chamada{
			Etapa: catalog.EtapaClassificacao, Alvo: p.Titulo,
			Evidencias: []policy.Evidencia{b.Evidencia()}, Tarefa: "classifique a função desta peça nos autos.",
			NomeSchema: "classificacao", Schema: schema, Reserva: 300,
		}, func(js []byte) error {
			var s saidaClassif
			if err := decodificarEstrito(js, &s); err != nil {
				return err
			}
			f = segment.Funcao(s.Funcao)
			if !permitida(f) {
				f = ""
				return fmt.Errorf("função %q fora da lista permitida", s.Funcao)
			}
			if f == segment.Indeterminada {
				return nil
			}
			if _, _, ok := Localizar(b.Texto, s.Trecho); !ok {
				return fmt.Errorf("o trecho_literal não aparece literalmente na peça")
			}
			return nil
		})
		if err != nil && !IsSemSeguranca(err) {
			return out, pend, err
		}
		if err != nil || f == "" || f == segment.Indeterminada {
			pend = append(pend, fmt.Sprintf("A função da peça %q não foi identificada.", p.Titulo))
			continue
		}
		out[i].Funcao, out[i].Base, out[i].Provavel = f, segment.BaseIA, true
	}
	return out, pend, nil
}

func permitida(f segment.Funcao) bool {
	for _, g := range funcoesIA {
		if g == f {
			return true
		}
	}
	return false
}

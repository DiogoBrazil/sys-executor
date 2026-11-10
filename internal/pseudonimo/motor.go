package pseudonimo

import (
	"context"
	"errors"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
)

// InstrucaoMarcadores é acrescentada às instruções de sistema quando há
// pseudonimização (TRV-SEG-011). Não altera nenhuma regra jurídica: só diz
// à IA como tratar os marcadores.
const InstrucaoMarcadores = `MARCADORES DE DADOS PESSOAIS: nomes de pessoas e dados pessoais dos autos foram substituídos por marcadores entre colchetes, como [PESSOA-01], [PESSOA-01.2] (outra grafia da mesma pessoa), [CPF-01], [RG-01], [MATRICULA-01], [TELEFONE-01], [ENDERECO-01]. Marcadores com o mesmo número antes do ponto são a mesma pessoa. Ao copiar trechos ou mencionar essas pessoas e dados, reproduza os marcadores exatamente como aparecem, com os colchetes. Não tente deduzir, completar nem inventar os dados reais.`

// reenviosMarcador: correções pedidas à IA quando ela altera marcadores.
const reenviosMarcador = 2

// Motor envolve outro motor: oculta os dados pessoais na ida e restaura a
// grafia exata na volta. É transparente para a análise (as conferências
// literais e as regras continuam iguais).
type Motor struct {
	interno inference.Motor
	mapa    *Mapa
}

// Envolver devolve o motor com pseudonimização (TRV-SEG-011).
func Envolver(m inference.Motor, mapa *Mapa) *Motor {
	return &Motor{interno: m, mapa: mapa}
}

// Interno devolve o motor envolvido (ex.: para ler o uso acumulado).
func (p *Motor) Interno() inference.Motor { return p.interno }

// Contexto delega ao motor envolvido.
func (p *Motor) Contexto() int { return p.interno.Contexto() }

// ContarTokens delega (os marcadores têm tamanho parecido com o dos nomes).
func (p *Motor) ContarTokens(ctx context.Context, t string) (int, error) {
	return p.interno.ContarTokens(ctx, t)
}

// Gerar oculta, envia, restaura e, se a IA alterar marcadores, pede a
// correção até reenviosMarcador vezes; depois disso, IA-010 (nunca entrega
// resposta com marcador sem restaurar nem com dado trocado).
func (p *Motor) Gerar(ctx context.Context, ped inference.Pedido) (inference.Resposta, error) {
	q := ped
	if q.Sistema != "" {
		q.Sistema += "\n\n" + InstrucaoMarcadores
	} else {
		q.Sistema = InstrucaoMarcadores
	}
	// Dados com padrão reconhecível que estejam no próprio pedido (texto
	// digitado pelo usuário, histórico de reenvio) são aprendidos antes.
	p.mapa.Aprender(ped.Usuario)
	for _, h := range ped.Historico {
		p.mapa.Aprender(h.Conteudo)
	}
	q.Usuario = p.mapa.Ocultar(ped.Usuario)
	q.Historico = make([]inference.Mensagem, len(ped.Historico))
	for i, h := range ped.Historico {
		q.Historico[i] = inference.Mensagem{Papel: h.Papel, Conteudo: p.mapa.Ocultar(h.Conteudo)}
	}
	var ultimo error
	for tentativa := 0; tentativa <= reenviosMarcador; tentativa++ {
		// Barreira, camada 1 (TRV-SEG-013): o pedido exato que vai sair.
		if err := p.conferir(q); err != nil {
			return inference.Resposta{}, err
		}
		r, err := p.interno.Gerar(ctx, q)
		if err != nil {
			return r, err
		}
		if r.Truncada {
			// O validador pede resposta mais concisa; o conteúdo parcial não é usado.
			if c, e := p.mapa.RestaurarJSON(r.Conteudo); e == nil {
				r.Conteudo = c
			} else {
				r.Conteudo = nil
			}
			return r, nil
		}
		c, err := p.mapa.RestaurarJSON(r.Conteudo)
		if err == nil {
			r.Conteudo = c
			return r, nil
		}
		ultimo = err
		var em ErrMarcador
		if !errors.As(err, &em) {
			break
		}
		q.Historico = append(q.Historico,
			inference.Mensagem{Papel: "assistant", Conteudo: string(r.Conteudo)},
			inference.Mensagem{Papel: "user", Conteudo: em.Error() +
				". Use somente os marcadores que aparecem nos documentos, copiados exatamente, com os colchetes, e responda de novo somente com o JSON do schema."})
	}
	return inference.Resposta{}, erros.Com(erros.IARespostaIncompleta, ultimo)
}

// conferir aplica a camada 1 da barreira. Vazamento vira IA-014 (bloqueio,
// sem opção de ignorar); possível nome não reconhecido volta como
// ErrRevisao, para a etapa perguntar ao usuário e tentar de novo.
func (p *Motor) conferir(q inference.Pedido) error {
	hs := make([]string, len(q.Historico))
	for i, h := range q.Historico {
		hs[i] = h.Conteudo
	}
	err := p.mapa.conferir(q.Sistema, q.Usuario, hs, q.Schema)
	var vz ErrVazamento
	if errors.As(err, &vz) {
		return erros.Com(erros.IABloqueioPrivacidade, vz)
	}
	return err
}

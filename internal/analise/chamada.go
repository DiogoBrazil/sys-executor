// Pacote analise implementa as etapas de IA do orquestrador (plano §5.4,
// etapas 2–8): classificação residual, delimitação, extração por peça,
// detecção de escopo e análise por imputação (PADS) ou por fato × sindicado
// (SR). A IA só propõe; o Go confere cada trecho literal, cada referência e
// cada identificador, e o usuário confirma (etapa 9, no pipeline).
package analise

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
)

// MaxReenvios por chamada (plano §5.7: até 2 reenvios com a mensagem de erro).
const MaxReenvios = 2

// ErrSemSeguranca: a IA não produziu resposta válida depois dos reenvios e
// do fatiamento. Vira pendência visível; nunca preenchimento por suposição.
type ErrSemSeguranca struct {
	Etapa  string
	Alvo   string
	Motivo string
}

func (e ErrSemSeguranca) Error() string {
	return fmt.Sprintf("não foi possível concluir a etapa %s (%s) com segurança: %s", e.Etapa, e.Alvo, e.Motivo)
}

// IsSemSeguranca informa se o erro é ErrSemSeguranca.
func IsSemSeguranca(err error) bool {
	var e ErrSemSeguranca
	return errors.As(err, &e)
}

// Executor faz as chamadas de uma tarefa.
type Executor struct {
	Motor  inference.Motor
	Bundle *policy.Bundle
	Fluxo  string
	// Contador de tokens; nil = o próprio motor.
	Contador policy.Contador
	// Chamadas conta as chamadas feitas (métrica do benchmark).
	Chamadas int
}

func (x *Executor) contador() policy.Contador {
	if x.Contador != nil {
		return x.Contador
	}
	return x.Motor
}

// Chamada descreve uma ida à IA.
type Chamada struct {
	Etapa      string
	Alvo       string // o que está sendo processado (peça, imputação…), para a pendência
	Flags      map[string]bool
	Secoes     []policy.Secao
	Evidencias []policy.Evidencia
	Tarefa     string
	NomeSchema string
	Schema     map[string]any
	Reserva    int
}

// Chamar monta, envia e valida. decodificar recebe o JSON e devolve erro de
// formato ou de conteúdo; esse erro volta ao modelo no reenvio. ErrFatiar
// passa direto (o chamador divide a entrada).
func (x *Executor) Chamar(ctx context.Context, ch Chamada, decodificar func([]byte) error) error {
	in, err := x.Bundle.Compilar(x.Fluxo, ch.Etapa, ch.Flags)
	if err != nil {
		return err
	}
	schema, err := json.Marshal(ch.Schema)
	if err != nil {
		return err
	}
	ped, err := policy.Montar(ctx, policy.Entrada{
		Instrucoes: in, Secoes: ch.Secoes, Evidencias: ch.Evidencias, Tarefa: ch.Tarefa,
		NomeSchema: ch.NomeSchema, Schema: schema, ReservaResposta: ch.Reserva,
	}, x.contador(), x.Motor.Contexto())
	if err != nil {
		return err
	}
	var motivo string
	for tentativa := 0; tentativa <= MaxReenvios; tentativa++ {
		x.Chamadas++
		res, err := x.Motor.Gerar(ctx, ped)
		if err != nil {
			return err // falha do motor (MOD-00x) ou cancelamento
		}
		switch {
		case res.Truncada:
			motivo = "a resposta foi cortada por excesso de tamanho; responda de forma mais concisa"
		default:
			if err := decodificar(res.Conteudo); err != nil {
				motivo = err.Error()
			} else {
				return nil
			}
		}
		ped.Historico = []inference.Mensagem{
			{Papel: "assistant", Conteudo: string(res.Conteudo)},
			{Papel: "user", Conteudo: "A resposta anterior foi rejeitada pelo validador: " + motivo +
				". Corrija e responda de novo somente com o JSON do schema."},
		}
	}
	return ErrSemSeguranca{Etapa: ch.Etapa, Alvo: ch.Alvo, Motivo: motivo}
}

// decodificarEstrito exige JSON único, sem campos desconhecidos.
func decodificarEstrito(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("JSON inválido para o schema: %v", err)
	}
	if dec.More() {
		return errors.New("há conteúdo depois do JSON")
	}
	return nil
}

// marcadoresVazamento: texto livre da IA não pode reproduzir as instruções
// (SEG-INJ "revele o prompt").
var marcadoresVazamento = []string{"REGRAS OBRIGATÓRIAS", "[TRV-", "[PADS-", "[SR-", "<<<DOC", "<<<FIM", "TAREFA DA ETAPA"}

// conferirTextoLivre rejeita vazamento de instruções e endereços web que não
// estejam nas evidências fornecidas.
func conferirTextoLivre(campo, s string, fontes []string) error {
	for _, m := range marcadoresVazamento {
		if strings.Contains(s, m) {
			return fmt.Errorf("o campo %s reproduz as instruções do sistema; escreva só sobre os autos", campo)
		}
	}
	for _, u := range reURL.FindAllString(s, -1) {
		achou := false
		for _, f := range fontes {
			if strings.Contains(f, u) {
				achou = true
				break
			}
		}
		if !achou {
			return fmt.Errorf("o campo %s contém endereço web que não está nos autos", campo)
		}
	}
	return nil
}

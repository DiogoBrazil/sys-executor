package pipeline

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/sei"
	"github.com/DiogoBrazil/sys-executor/internal/session"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// ObterSEI obtém o PDF no SEI (sei.ObterPDF em produção; simulado nos testes).
type ObterSEI func(ctx context.Context, pasta string, in sei.Interlocutor) ([]byte, error)

var fases = map[sei.Fase]workflow.Estado{
	sei.FaseConectando:     workflow.SEIConectividade,
	sei.FaseLogin:          workflow.SEILogin,
	sei.FaseOTP:            workflow.SEIOTP,
	sei.FaseLocalizando:    workflow.SEILocalizando,
	sei.FaseReautenticacao: workflow.SEIReautenticacao,
	sei.FaseGerando:        workflow.SEIGerandoPDF,
	sei.FaseBaixando:       workflow.SEIBaixando,
}

// obterSEI conduz a obtenção pelo SEI. Qualquer falha volta à escolha da
// entrada com o alerta, preservando o tipo (T03: oferecer o envio do PDF).
func (p *Pipeline) obterSEI(ctx context.Context, c *workflow.Contexto) (workflow.Estado, error) {
	if p.SEI == nil {
		return workflow.TipoEscolhido, erros.Novo(erros.TarefaEtapaIndisponiv)
	}
	b, err := p.SEI(ctx, filepath.Join(c.Sessao.Dir(), "sei"), &interlocutor{c: c})
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return workflow.TipoEscolhido, err
	}
	_ = c.Avancar(workflow.SEIBaixando)
	if err := c.Sessao.Gravar(workflow.NomeAutos, b); err != nil {
		return "", err
	}
	return workflow.Verificando, nil
}

// interlocutor traduz os pedidos do cliente SEI em pedidos da tarefa.
type interlocutor struct{ c *workflow.Contexto }

func (i *interlocutor) Fase(f sei.Fase) {
	if e, ok := fases[f]; ok {
		_ = i.c.Avancar(e)
	}
}

func (i *interlocutor) Credenciais(ctx context.Context, alerta *erros.Erro) (sei.Login, error) {
	r, err := i.c.Solicitar(ctx, workflow.Pedido{Tipo: workflow.PedidoCredenciais, Alerta: alerta})
	if err != nil {
		return sei.Login{}, err
	}
	if r.Segredo == nil || strings.TrimSpace(r.Campos["processo"]) == "" {
		if r.Segredo != nil {
			r.Segredo.Zerar()
		}
		return sei.Login{}, erros.Com(erros.SEICredencial, errors.New("dados incompletos"))
	}
	return sei.Login{Usuario: r.Campos["usuario"], Senha: r.Segredo, Orgao: r.Campos["orgao"], Processo: r.Campos["processo"]}, nil
}

func (i *interlocutor) segredo(ctx context.Context, tipo workflow.TipoPedido, alerta *erros.Erro) (*session.Segredo, error) {
	r, err := i.c.Solicitar(ctx, workflow.Pedido{Tipo: tipo, Alerta: alerta})
	if err != nil {
		return nil, err
	}
	if r.Segredo == nil {
		return session.NovoSegredo(nil), nil
	}
	return r.Segredo, nil
}

func (i *interlocutor) CodigoOTP(ctx context.Context, alerta *erros.Erro) (*session.Segredo, error) {
	return i.segredo(ctx, workflow.PedidoOTP, alerta)
}

func (i *interlocutor) SenhaReautenticacao(ctx context.Context, alerta *erros.Erro) (*session.Segredo, error) {
	return i.segredo(ctx, workflow.PedidoReautenticacao, alerta)
}

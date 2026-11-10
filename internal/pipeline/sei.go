package pipeline

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/segment"
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
	// V4 §48.4 (revisão de 10/11/2026): o PDF de todos os documentos é
	// separado pelo carimbo, um documento por número SEI, e vai ao cadastro,
	// onde o usuário escolhe os de cada peça. O PDF completo não é gravado.
	docs, err := p.separarSEI(ctx, c.Tipo, b)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return workflow.TipoEscolhido, err
	}
	if err := c.ReceberDocumentosSEI(docs); err != nil {
		return "", err
	}
	return workflow.RecebendoUpload, nil
}

// separarSEI lê só o texto nativo (o carimbo do SEI é texto mesmo em página
// escaneada; o OCR fica para os documentos escolhidos), separa pelo carimbo
// e sugere o tipo de peça pelo título do documento na árvore. A sugestão é
// conferida pelo usuário no cadastro; a classificação pelo conteúdo continua
// sendo a conferência depois (TRV-ING-001).
func (p *Pipeline) separarSEI(ctx context.Context, tipo workflow.Tipo, b []byte) ([]workflow.DocumentoSEIEntrada, error) {
	sep, ok := p.Extrator.(pdf.Separador)
	if p.Separador != nil {
		sep, ok = p.Separador, true
	}
	if !ok {
		return nil, erros.Com(erros.InternoInesperado, errors.New("separação de PDF indisponível"))
	}
	a, err := autos.LerTextoNativo(ctx, p.Extrator, b)
	if err != nil {
		return nil, err
	}
	pecas := segment.Segmentar(a)
	faixas := make([]pdf.Faixa, len(pecas))
	for i, pc := range pecas {
		faixas[i] = pdf.Faixa{Inicio: pc.PaginaInicial, Fim: pc.PaginaFinal}
	}
	partes, err := sep.Separar(ctx, b, faixas)
	if err != nil {
		return nil, err
	}
	out := make([]workflow.DocumentoSEIEntrada, len(pecas))
	for i, pc := range pecas {
		sug, papel := sugestaoPeca(tipo, pc.Funcao)
		titulo := pc.Titulo
		if titulo == "" {
			titulo = "Documento sem título"
		}
		out[i] = workflow.DocumentoSEIEntrada{SEI: pc.SEI, Titulo: titulo, Sugestao: sug, Papel: papel,
			Paginas: pc.PaginaFinal - pc.PaginaInicial + 1, Dados: partes[i]}
	}
	return out, nil
}

// sugestaoPeca traduz a função lida do título para o tipo do cadastro. No
// PADS, o interrogatório do acusado vai em "Outros documentos" (plano D-11
// §8.1, decisão pendente); na SR, é o depoimento do sindicado.
func sugestaoPeca(tipo workflow.Tipo, f segment.Funcao) (string, string) {
	switch f {
	case segment.MemorandoDisciplinar:
		return workflow.PecaMemorando, ""
	case segment.PortariaSR:
		return workflow.PecaPortaria, ""
	case segment.DefesaPrevia:
		return workflow.PecaDefesa, ""
	case segment.AlegacoesFinais:
		return workflow.PecaAlegacoes, ""
	case segment.TermoInquiricao:
		return workflow.PecaDepoimento, workflow.PapelTestemunha
	case segment.TermoDeclaracoes:
		return workflow.PecaDepoimento, workflow.PapelDeclarante
	case segment.Interrogatorio:
		if tipo == workflow.TipoSR {
			return workflow.PecaDepoimento, workflow.PapelSindicado
		}
	}
	return workflow.PecaOutro, ""
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

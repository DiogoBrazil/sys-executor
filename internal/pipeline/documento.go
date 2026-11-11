package pipeline

import (
	"context"
	"errors"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/documento"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// GerarDocumento produz o DOCX ou o PDF da minuta em revisão (V4 §31.3) e
// sugere o nome do arquivo. dados é o estado da tarefa (workflow.Gerenciador
// .EmRevisao); pasta é a sessão da tarefa, usada só para os intermediários
// do PDF. Nada é gravado fora dela aqui: quem grava é o usuário, ao escolher
// o destino.
func (p *Pipeline) GerarDocumento(ctx context.Context, dados any, pasta, formato string) ([]byte, string, error) {
	d, ok := dados.(*Dados)
	if !ok || d == nil || d.Minuta == nil {
		return nil, "", erros.Novo(erros.TarefaOrdemInvalida)
	}
	agora := time.Now
	if p.Agora != nil {
		agora = p.Agora
	}
	nome := documento.NomeArquivo(d.Minuta.Procedimento, d.versaoMinuta, agora(), formato)
	cab := documento.CarregarCabecalho(p.Componentes)
	switch formato {
	case documento.FormatoDOCX:
		b, err := documento.DOCX(d.Minuta, cab)
		if err != nil {
			return nil, "", erros.Com(erros.ExportacaoFalhou, err)
		}
		return b, nome, nil
	case documento.FormatoPDF:
		t, err := documento.LocalizarTypst(p.Componentes)
		if err != nil {
			return nil, "", erros.Com(erros.ExportacaoSemPDF, err)
		}
		b, err := t.PDF(ctx, d.Minuta, cab, pasta)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, "", err
			}
			return nil, "", erros.Com(erros.ExportacaoFalhou, err)
		}
		return b, nome, nil
	}
	return nil, "", erros.Novo(erros.TarefaOrdemInvalida)
}

// Package pipeline registra as etapas de processamento dos autos na tarefa
// (plano §7). Fase 2: verificação, extração/OCR, classificação das peças e
// compatibilidade procedimento × autos. As etapas analíticas seguintes
// (delimitação em diante) chegam nas Fases 3–6. Fase 4: delimitação e
// análise com a IA (etapas 2–9 do plano §5.4). Fase 5: consulta à
// biblioteca jurídica (etapa 11).
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/ocr"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/segment"
	"github.com/DiogoBrazil/sys-executor/internal/legal"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// Dados é o estado da análise de uma tarefa (somente memória).
type Dados struct {
	Autos  *autos.Autos
	Pecas  []segment.Peca
	Midias []segment.MidiaReferida
	Datas  []autos.DataMencionada // todas as menções, inclusive divergentes (A03)
	Indice *autos.Indice
	Compat segment.Compatibilidade
	// Declaradas: peças do cadastro, na ordem dos autos juntados (nil quando
	// os autos vieram de um PDF único do SEI).
	Declaradas []segment.Declarada
	// IA: estado das etapas de IA (Fase 4), nil antes da delimitação.
	IA *EstadoIA
	// Fontes: dispositivos recuperados da biblioteca jurídica (Fase 5), nil
	// antes da consulta.
	Fontes *FontesJuridicas
}

// Liberar fecha o índice em memória (chamado pela tarefa ao terminar).
func (d *Dados) Liberar() {
	d.IA.liberar()
	if d.Indice != nil {
		_ = d.Indice.Fechar()
		d.Indice = nil
	}
}

// Pipeline reúne as dependências das etapas.
type Pipeline struct {
	Extrator pdf.Extrator
	// LocalizarOCR devolve o motor de OCR (procurado só quando necessário).
	LocalizarOCR func() (ocr.Motor, error)
	DPI          int
	// SEI obtém o PDF pelo SEI/RO (nil: modalidade indisponível).
	SEI ObterSEI
	// Separador divide o PDF do SEI por documento (nil = o do Extrator).
	Separador pdf.Separador
	// IA prepara o motor de IA local (nil: etapas de IA indisponíveis).
	IA FonteMotor
	// Juntador monta os autos a partir do cadastro de peças (nil: usa o
	// Extrator, se ele souber juntar).
	Juntador pdf.Juntador
	// Biblioteca abre a biblioteca jurídica (nil = o acervo inicial embutido).
	Biblioteca func() (*legal.Biblioteca, error)

	once   sync.Once
	motor  ocr.Motor
	errOCR error
}

func (p *Pipeline) ocr() ocr.Motor {
	p.once.Do(func() {
		if p.LocalizarOCR != nil {
			p.motor, p.errOCR = p.LocalizarOCR()
			if p.errOCR != nil {
				p.motor = nil // nunca usar um motor que falhou ao ser localizado
			}
		}
	})
	return p.motor
}

// Etapas devolve as etapas desta fase, por estado.
func (p *Pipeline) Etapas() map[workflow.Estado]workflow.Etapa {
	return map[workflow.Estado]workflow.Etapa{
		workflow.SEIConectividade:  p.obterSEI,
		workflow.Verificando:       p.verificar,
		workflow.Extraindo:         p.extrair,
		workflow.Classificando:     p.classificar,
		workflow.Compatibilidade:   p.compatibilidade,
		workflow.Delimitando:       p.comRevisao(workflow.Delimitando, p.delimitar),
		workflow.Analisando:        p.comRevisao(workflow.Analisando, p.analisar),
		workflow.ConsultandoFontes: p.consultarFontes,
	}
}

func dados(c *workflow.Contexto) *Dados {
	d, ok := c.Dados.(*Dados)
	if !ok {
		d = &Dados{}
		c.Dados = d
	}
	return d
}

// verificar abre o PDF para conferir que é legível. PDF protegido, corrompido
// ou sem páginas volta ao envio, com o alerta, sem perder o tipo escolhido.
func (p *Pipeline) verificar(ctx context.Context, c *workflow.Contexto) (workflow.Estado, error) {
	d := dados(c)
	d.Declaradas = nil
	if cad := c.Cadastro(); cad != nil {
		// Cadastro de peças: junta os arquivos (a junção já confere cada um).
		ds, err := p.juntarCadastro(ctx, c.Sessao, cad)
		if err != nil {
			var e *erros.Erro
			if errors.As(err, &e) && e.Categoria == erros.CatPDF {
				return workflow.RecebendoUpload, err
			}
			return "", err
		}
		d.Declaradas = ds
		return workflow.Extraindo, nil
	}
	b, err := c.Sessao.Ler(workflow.NomeAutos)
	if err != nil {
		return "", err
	}
	doc, err := p.Extrator.Abrir(ctx, b)
	if err != nil {
		var e *erros.Erro
		if errors.As(err, &e) && e.Categoria == erros.CatPDF {
			return workflow.RecebendoUpload, err
		}
		return "", err
	}
	doc.Fechar()
	return workflow.Extraindo, nil
}

func (p *Pipeline) extrair(ctx context.Context, c *workflow.Contexto) (workflow.Estado, error) {
	b, err := c.Sessao.Ler(workflow.NomeAutos)
	if err != nil {
		return "", err
	}
	ex := autos.Extracao{PDF: p.Extrator, OCR: p.ocr(), DPI: p.DPI}
	a, err := ex.Extrair(ctx, b, func(at, tot int) { c.Progresso(at, tot, "páginas") })
	if err != nil {
		if errors.Is(err, erros.Novo(erros.OCRFalhou)) && p.errOCR != nil {
			return "", erros.Com(erros.OCRFalhou, fmt.Errorf("OCR indisponível: %w", p.errOCR))
		}
		return "", err
	}
	dados(c).Autos = a
	return workflow.Classificando, nil
}

func (p *Pipeline) classificar(ctx context.Context, c *workflow.Contexto) (workflow.Estado, error) {
	d := dados(c)
	if d.Autos == nil {
		return "", erros.Com(erros.InternoInesperado, errors.New("autos não extraídos"))
	}
	if d.Declaradas != nil {
		ps, err := segment.DeDeclaracao(d.Autos, d.Declaradas)
		if err != nil {
			return "", erros.Com(erros.InternoInesperado, err)
		}
		d.Pecas = ps
	} else {
		d.Pecas = segment.Segmentar(d.Autos)
	}
	d.Midias = segment.DetectarMidias(d.Autos, d.Pecas)
	d.Datas = d.Autos.Datas()
	if d.Indice != nil {
		_ = d.Indice.Fechar()
	}
	ix, err := autos.Indexar(d.Autos)
	if err != nil {
		return "", erros.Com(erros.InternoInesperado, err)
	}
	d.Indice = ix
	c.Publicar(Resumir(d))
	return workflow.Compatibilidade, nil
}

const perguntaCompat = "compatibilidade"

func (p *Pipeline) compatibilidade(ctx context.Context, c *workflow.Contexto) (workflow.Estado, error) {
	d := dados(c)
	if d.Declaradas != nil {
		d.Compat = segment.VerificarDeclaradas(string(c.Tipo), d.Pecas, d.Autos)
	} else {
		d.Compat = segment.VerificarCompatibilidade(string(c.Tipo), d.Pecas, d.Autos)
	}
	c.Publicar(Resumir(d))
	if d.Compat.Resultado == segment.Compativel {
		return workflow.Delimitando, nil
	}
	if r, ok := c.Resposta(perguntaCompat); ok && r == "continuar" {
		return workflow.Delimitando, nil
	}
	if d.Declaradas != nil {
		c.Perguntar(perguntaDeclaradas(c.Tipo, d))
		return workflow.AguardandoEsclarecimento, nil
	}
	c.Perguntar(perguntaCompatibilidade(c.Tipo, d.Compat))
	return workflow.AguardandoEsclarecimento, nil
}

// perguntaCompatibilidade monta a pergunta (V4 §30.2): nunca troca o tipo.
func perguntaCompatibilidade(tipo workflow.Tipo, comp segment.Compatibilidade) workflow.Pergunta {
	nome, peca := "Julgamento de PADS", "o Memorando Disciplinar"
	outroNome := "Relatório de Sindicância Regular"
	if tipo == workflow.TipoSR {
		nome, peca, outroNome = "Relatório de Sindicância Regular", "a Portaria de instauração", "Julgamento de PADS"
	}
	opcoes := []workflow.Opcao{
		{ID: "outro", Rotulo: "Enviar outro PDF", Destino: workflow.RecebendoUpload},
		{ID: "continuar", Rotulo: "Continuar como " + nome},
		{ID: "cancelar", Rotulo: "Cancelar a tarefa", Destino: workflow.Cancelada},
	}
	p := workflow.Pergunta{ID: perguntaCompat, Opcoes: opcoes}
	switch comp.Resultado {
	case segment.Incompativel:
		p.Titulo = "Os autos parecem ser de " + outroNome
		p.Contexto = fmt.Sprintf("Você escolheu %s, mas não foi encontrado %s, e os autos têm a peça central do outro procedimento. "+
			"O tipo de minuta não é trocado automaticamente: confira o PDF ou cancele e escolha o outro tipo.", nome, peca)
	case segment.PecaIlegivel:
		p.Titulo = "A peça instauradora está ilegível"
		p.Contexto = fmt.Sprintf("As primeiras peças dos autos têm leitura de baixa qualidade e %s não pôde ser identificado com segurança. "+
			"Envie uma versão mais legível do PDF, se tiver.", peca)
	default:
		p.Titulo = "Peça central não encontrada"
		p.Contexto = fmt.Sprintf("Não foi encontrado %s nos autos. Confira se o PDF foi gerado com \"Todos os documentos disponíveis\".", peca)
	}
	return p
}

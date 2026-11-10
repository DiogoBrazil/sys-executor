package main

// Entradas comparadas pelo bench (plano D-11 §4.9: "comparação monolítico ×
// peças, com os mesmos casos sintéticos, antes de afirmar qualquer ganho"):
//
//   - memoria: os autos e as peças do gabarito, sem PDF (como o bench era
//     antes da etapa 8; isola as etapas de IA da ingestão);
//   - monolitica: um único PDF com o carimbo do SEI em cada página, extraído
//     pelo pdfium e separado por segment.Segmentar. É o único caminho
//     monolítico que resta no aplicativo: o PDF que o próprio SEI gera (E6);
//   - pecas: um PDF por peça, juntados pelo pdf.Juntador e montados por
//     segment.DeDeclaracao com a função declarada, como no cadastro (V4
//     §48.4); os envolvidos semeiam o mapa de pseudônimos (CadastrarPessoa).
//
// Os PDFs são gerados em memória (pdfsint, só dados fictícios) e nunca vão
// ao disco.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/segment"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/casos"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/pdfsint"
)

const (
	entMemoria    = "memoria"
	entMonolitica = "monolitica"
	entPecas      = "pecas"
	// processoSintetico: nº de processo fictício do carimbo.
	processoSintetico = "0000.000001/2026-00"
)

// preparada é a entrada pronta para as etapas de IA.
type preparada struct {
	autos     *autos.Autos
	pecas     []segment.Peca
	cadastro  bool          // pessoas declaradas semeiam o mapa
	tempo     time.Duration // geração do PDF + junção + extração + separação
	paginas   int
	pecasOK   int // peças com a mesma função e as mesmas páginas do gabarito
	pecasEsp  int
	pecasObt  int
	divergem  []string
	semIngest bool // entrada em memória (sem PDF)
}

// leitor reúne o extrator e o juntador do pdfium (uma instância por rodada).
type leitor struct {
	ext pdf.Extrator
	jun pdf.Juntador
}

func novoLeitor() (*leitor, error) {
	e := pdf.NovoExtrator()
	j, ok := e.(pdf.Juntador)
	if !ok {
		return nil, fmt.Errorf("o extrator do pdfium não junta PDFs")
	}
	// Aquece o pdfium (o wasm demora a iniciar na 1ª abertura), para o tempo
	// de preparação medir só a entrada.
	doc, err := e.Abrir(context.Background(), pdfsint.Gerar([]pdfsint.Pagina{{Linhas: []string{"aquecimento"}}}))
	if err != nil {
		_ = e.Fechar()
		return nil, err
	}
	doc.Fechar()
	return &leitor{ext: e, jun: j}, nil
}

func (l *leitor) fechar() {
	if l != nil {
		_ = l.ext.Fechar()
	}
}

// quebrarLinhas cabe o texto da página na largura do PDF sintético.
func quebrarLinhas(s string, max int) []string {
	var out []string
	for _, par := range strings.Split(s, "\n") {
		linha := ""
		for _, w := range strings.Fields(par) {
			if linha != "" && len([]rune(linha))+1+len([]rune(w)) > max {
				out = append(out, linha)
				linha = w
				continue
			}
			if linha != "" {
				linha += " "
			}
			linha += w
		}
		out = append(out, linha)
	}
	return out
}

func preparar(ctx context.Context, l *leitor, c casos.Caso, modo string) (preparada, error) {
	if modo == entMemoria {
		return preparada{autos: c.Autos, pecas: c.Pecas, semIngest: true, paginas: len(c.Autos.Paginas),
			pecasOK: len(c.Pecas), pecasEsp: len(c.Pecas), pecasObt: len(c.Pecas)}, nil
	}
	t0 := time.Now()
	var (
		a   *autos.Autos
		ps  []segment.Peca
		err error
	)
	ex := autos.Extracao{PDF: l.ext} // sem OCR: os PDFs sintéticos têm camada de texto
	switch modo {
	case entMonolitica:
		var pgs []pdfsint.Pagina
		for _, p := range c.Pecas {
			for n := p.PaginaInicial; n <= p.PaginaFinal; n++ {
				// Carimbo do SEI/RO (autos.LerCarimbos):
				// "<documento> (<nº SEI>) SEI <processo> / pg. <página>".
				pgs = append(pgs, pdfsint.Pagina{Linhas: quebrarLinhas(c.Autos.Paginas[n-1].Texto, 72),
					Rodape: fmt.Sprintf("%s (%s) SEI %s / pg. %d", p.Titulo, p.SEI, processoSintetico, n)})
			}
		}
		if a, err = ex.Extrair(ctx, pdfsint.Gerar(pgs), nil); err != nil {
			return preparada{}, err
		}
		ps = segment.Segmentar(a)
	case entPecas:
		var arqs [][]byte
		ds := make([]segment.Declarada, len(c.Pecas))
		for i, p := range c.Pecas {
			var pgs []pdfsint.Pagina
			for n := p.PaginaInicial; n <= p.PaginaFinal; n++ {
				pgs = append(pgs, pdfsint.Pagina{Linhas: quebrarLinhas(c.Autos.Paginas[n-1].Texto, 72)})
			}
			arqs = append(arqs, pdfsint.Gerar(pgs))
			ds[i] = segment.Declarada{Cadastro: fmt.Sprintf("PEC-%02d", i+1), Funcao: p.Funcao, Titulo: p.Titulo}
		}
		junto, paginas, err := l.jun.Juntar(ctx, arqs)
		if err != nil {
			return preparada{}, err
		}
		for i, n := range paginas {
			ds[i].Paginas = n
		}
		if a, err = ex.Extrair(ctx, junto, nil); err != nil {
			return preparada{}, err
		}
		if ps, err = segment.DeDeclaracao(a, ds); err != nil {
			return preparada{}, err
		}
	default:
		return preparada{}, fmt.Errorf("entrada desconhecida %q (memoria, monolitica ou pecas)", modo)
	}
	r := preparada{autos: a, pecas: ps, cadastro: modo == entPecas, tempo: time.Since(t0), paginas: len(a.Paginas),
		pecasEsp: len(c.Pecas), pecasObt: len(ps)}
	r.pecasOK, r.divergem = conferirPecas(c.Pecas, ps)
	return r, nil
}

// conferirPecas compara a separação obtida com o gabarito: função e páginas.
func conferirPecas(esp, obt []segment.Peca) (int, []string) {
	ok := 0
	var div []string
	usadas := map[int]bool{}
	for _, e := range esp {
		achou := false
		for j, o := range obt {
			if !usadas[j] && o.Funcao == e.Funcao && o.PaginaInicial == e.PaginaInicial && o.PaginaFinal == e.PaginaFinal {
				usadas[j], achou = true, true
				break
			}
		}
		if achou {
			ok++
			continue
		}
		div = append(div, fmt.Sprintf("esperada %s pp. %d–%d", e.Funcao, e.PaginaInicial, e.PaginaFinal))
	}
	for j, o := range obt {
		if !usadas[j] {
			div = append(div, fmt.Sprintf("obtida %s pp. %d–%d", o.Funcao, o.PaginaInicial, o.PaginaFinal))
		}
	}
	return ok, div
}

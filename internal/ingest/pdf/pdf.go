// Package pdf extrai texto com posições e renderiza páginas dos autos
// (plano §7; ADR-004: go-pdfium em WebAssembly/wazero, sem CGO).
//
// O PDF é lido só da memória: nenhum arquivo temporário é criado.
package pdf

import (
	"context"
	"errors"
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/klippa-app/go-pdfium"
	pdfiumerr "github.com/klippa-app/go-pdfium/errors"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// Caixa é um retângulo em pontos (1/72 pol.) com origem no canto superior
// esquerdo da página (Y cresce para baixo), como o PDF.js usa na tela.
type Caixa struct {
	X0, Y0, X1, Y1 float64
}

// Caractere da camada de texto, na ordem de leitura calculada pelo pdfium
// (que já insere os espaços e quebras de linha entre palavras e linhas).
type Caractere struct {
	Runa     rune
	Caixa    Caixa
	SemCaixa bool // caractere gerado pelo pdfium (espaço/quebra), sem posição
}

// Pagina é o conteúdo textual nativo de uma página.
type Pagina struct {
	Indice     int // base 0
	Largura    float64
	Altura     float64
	Caracteres []Caractere
}

// Extrator abre documentos PDF (porta PDFExtractor do plano §4).
type Extrator interface {
	Abrir(ctx context.Context, dados []byte) (Documento, error)
	Fechar() error
}

// Documento é um PDF aberto.
type Documento interface {
	Paginas() int
	Pagina(ctx context.Context, i int) (Pagina, error)
	// Renderizar devolve a página como imagem na resolução pedida (para OCR).
	Renderizar(ctx context.Context, i int, dpi int) (image.Image, error)
	Fechar()
}

// ---- implementação pdfium (WebAssembly) ----

type extratorPdfium struct {
	once sync.Once
	pool pdfium.Pool
	err  error
}

// NovoExtrator cria o extrator. O runtime WebAssembly (~2 s) é iniciado na
// primeira abertura e reaproveitado até Fechar.
func NovoExtrator() Extrator { return &extratorPdfium{} }

func (e *extratorPdfium) iniciar() error {
	e.once.Do(func() {
		e.pool, e.err = webassembly.Init(webassembly.Config{MinIdle: 1, MaxIdle: 1, MaxTotal: 1})
	})
	return e.err
}

func (e *extratorPdfium) Fechar() error {
	if e.pool != nil {
		return e.pool.Close()
	}
	return nil
}

func (e *extratorPdfium) Abrir(ctx context.Context, dados []byte) (Documento, error) {
	if err := e.iniciar(); err != nil {
		return nil, erros.Com(erros.InternoInesperado, err)
	}
	inst, err := e.pool.GetInstance(30 * time.Second)
	if err != nil {
		return nil, erros.Com(erros.InternoInesperado, err)
	}
	d, err := inst.OpenDocument(&requests.OpenDocument{File: &dados})
	if err != nil {
		inst.Close()
		return nil, classificarErro(err)
	}
	pc, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: d.Document})
	if err != nil {
		inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: d.Document})
		inst.Close()
		return nil, erros.Com(erros.PDFCorrompido, err)
	}
	if pc.PageCount <= 0 {
		inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: d.Document})
		inst.Close()
		return nil, erros.Novo(erros.PDFVazio)
	}
	return &documentoPdfium{inst: inst, doc: d.Document, n: pc.PageCount}, nil
}

func classificarErro(err error) error {
	switch {
	case errors.Is(err, pdfiumerr.ErrPassword), errors.Is(err, pdfiumerr.ErrSecurity):
		return erros.Com(erros.PDFProtegido, err)
	default:
		return erros.Com(erros.PDFCorrompido, err)
	}
}

type documentoPdfium struct {
	mu   sync.Mutex
	inst pdfium.Pdfium
	doc  references.FPDF_DOCUMENT
	n    int
}

func (d *documentoPdfium) Paginas() int { return d.n }

func (d *documentoPdfium) ref(i int) requests.Page {
	return requests.Page{ByIndex: &requests.PageByIndex{Document: d.doc, Index: i}}
}

func (d *documentoPdfium) Pagina(ctx context.Context, i int) (Pagina, error) {
	if err := ctx.Err(); err != nil {
		return Pagina{}, err
	}
	if i < 0 || i >= d.n {
		return Pagina{}, fmt.Errorf("página %d fora do intervalo", i)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	tam, err := d.inst.GetPageSize(&requests.GetPageSize{Page: d.ref(i)})
	if err != nil {
		return Pagina{}, erros.Com(erros.PDFCorrompido, err)
	}
	// Modo por caractere: o modo por retângulos devolve regiões e duplica
	// letras quando as caixas se sobrepõem (visto em PDF real do SEI).
	st, err := d.inst.GetPageTextStructured(&requests.GetPageTextStructured{
		Page: d.ref(i),
		Mode: requests.GetPageTextStructuredModeChars,
	})
	if err != nil {
		return Pagina{}, erros.Com(erros.PDFCorrompido, err)
	}
	p := Pagina{Indice: i, Largura: tam.Width, Altura: tam.Height}
	for _, c := range st.Chars {
		rs := []rune(c.Text)
		if len(rs) == 0 || rs[0] == 0 {
			continue
		}
		pp := c.PointPosition
		car := Caractere{Runa: rs[0]}
		if pp.Right-pp.Left <= 0 && pp.Top-pp.Bottom == 0 {
			car.SemCaixa = true
		} else {
			car.Caixa = normalizar(pp.Left, pp.Top, pp.Right, pp.Bottom, tam.Height)
		}
		p.Caracteres = append(p.Caracteres, car)
	}
	return p, nil
}

// normalizar converte a posição do pdfium (origem embaixo à esquerda, Y para
// cima) para origem em cima à esquerda, garantindo X0<X1 e Y0<Y1.
func normalizar(esq, topo, dir, base, altura float64) Caixa {
	y0, y1 := altura-topo, altura-base
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	if esq > dir {
		esq, dir = dir, esq
	}
	return Caixa{X0: esq, Y0: y0, X1: dir, Y1: y1}
}

func (d *documentoPdfium) Renderizar(ctx context.Context, i int, dpi int) (image.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	r, err := d.inst.RenderPageInDPI(&requests.RenderPageInDPI{Page: d.ref(i), DPI: dpi})
	if err != nil {
		return nil, erros.Com(erros.PDFCorrompido, err)
	}
	defer r.Cleanup()
	// A imagem do pdfium é liberada no Cleanup: copia para memória Go.
	src := r.Result.Image
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst, nil
}

func (d *documentoPdfium) Fechar() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.inst != nil {
		d.inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: d.doc})
		d.inst.Close()
		d.inst = nil
	}
}

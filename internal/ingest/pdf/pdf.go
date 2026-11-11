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
	"io"
	"sync"
	"time"

	"github.com/klippa-app/go-pdfium"
	pdfiumerr "github.com/klippa-app/go-pdfium/errors"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/tetratelabs/wazero"

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
		e.pool, e.err = webassembly.Init(webassembly.Config{
			MinIdle: 1, MaxIdle: 1, MaxTotal: 1,
			// O app é compilado sem console (-H windowsgui): stdout/stderr são
			// identificadores inválidos e o wazero falharia ao instanciar o
			// módulo ("GetFileType /dev/stdout"). Saídas descartadas.
			Stdout: io.Discard,
			Stderr: io.Discard,
			// Nenhuma pasta do disco montada no WebAssembly (o padrão monta o
			// volume inteiro): o PDF entra só pela memória.
			FSConfig: wazero.NewFSConfig(),
		})
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
		car := Caractere{Runa: runaDeTexto(rs[0])}
		if pp.Right-pp.Left <= 0 && pp.Top-pp.Bottom == 0 {
			car.SemCaixa = true
		} else {
			car.Caixa = normalizar(pp.Left, pp.Top, pp.Right, pp.Bottom, tam.Height)
		}
		p.Caracteres = append(p.Caracteres, car)
	}
	return p, nil
}

// runaDeTexto troca o marcador U+0002 pelo hífen. O pdfium usa esse
// marcador no lugar do hífen que cai no fim de uma linha e junta as duas
// partes; sem a troca, "Verifica-se" seria lido como "Verificase".
func runaDeTexto(r rune) rune {
	if r == 0x02 {
		return '-'
	}
	return r
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

// Juntador monta, em memória, um único PDF com as páginas de vários
// arquivos, na ordem dada (cadastro de peças, V4 §48.4). Devolve também
// quantas páginas cada arquivo tem, para mapear peça → páginas dos autos.
type Juntador interface {
	Juntar(ctx context.Context, arquivos [][]byte) (pdf []byte, paginas []int, err error)
}

// Juntar usa uma única instância do pdfium (o pool tem uma) e não grava
// nada em disco. PDF protegido, corrompido ou vazio é recusado.
func (e *extratorPdfium) Juntar(ctx context.Context, arquivos [][]byte) ([]byte, []int, error) {
	if len(arquivos) == 0 {
		return nil, nil, erros.Novo(erros.PDFVazio)
	}
	if err := e.iniciar(); err != nil {
		return nil, nil, erros.Com(erros.InternoInesperado, err)
	}
	inst, err := e.pool.GetInstance(30 * time.Second)
	if err != nil {
		return nil, nil, erros.Com(erros.InternoInesperado, err)
	}
	defer inst.Close()
	novo, err := inst.FPDF_CreateNewDocument(&requests.FPDF_CreateNewDocument{})
	if err != nil {
		return nil, nil, erros.Com(erros.InternoInesperado, err)
	}
	defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: novo.Document})
	paginas := make([]int, len(arquivos))
	total := 0
	for i := range arquivos {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		d, err := inst.OpenDocument(&requests.OpenDocument{File: &arquivos[i]})
		if err != nil {
			return nil, nil, classificarErro(err)
		}
		pc, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: d.Document})
		if err == nil && pc.PageCount <= 0 {
			err = erros.Novo(erros.PDFVazio)
		}
		if err == nil {
			_, err = inst.FPDF_ImportPagesByIndex(&requests.FPDF_ImportPagesByIndex{Source: d.Document, Destination: novo.Document, Index: total})
		}
		inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: d.Document})
		if err != nil {
			var e *erros.Erro
			if errors.As(err, &e) {
				return nil, nil, err
			}
			return nil, nil, erros.Com(erros.PDFCorrompido, err)
		}
		paginas[i] = pc.PageCount
		total += pc.PageCount
	}
	salvo, err := inst.FPDF_SaveAsCopy(&requests.FPDF_SaveAsCopy{Document: novo.Document})
	if err != nil || salvo.FileBytes == nil {
		return nil, nil, erros.Com(erros.InternoInesperado, fmt.Errorf("salvar o PDF juntado: %v", err))
	}
	return *salvo.FileBytes, paginas, nil
}

// Faixa de páginas (base 1, inclusiva).
type Faixa struct{ Inicio, Fim int }

// Separador divide um PDF em vários, um por faixa de páginas, em memória
// (documentos do PDF gerado pelo SEI, V4 §48.4, revisão de 10/11/2026).
type Separador interface {
	Separar(ctx context.Context, dados []byte, faixas []Faixa) ([][]byte, error)
}

// Separar usa uma única instância do pdfium e não grava nada em disco.
// Faixa fora do documento é recusada.
func (e *extratorPdfium) Separar(ctx context.Context, dados []byte, faixas []Faixa) ([][]byte, error) {
	if len(faixas) == 0 {
		return nil, erros.Novo(erros.PDFVazio)
	}
	if err := e.iniciar(); err != nil {
		return nil, erros.Com(erros.InternoInesperado, err)
	}
	inst, err := e.pool.GetInstance(30 * time.Second)
	if err != nil {
		return nil, erros.Com(erros.InternoInesperado, err)
	}
	defer inst.Close()
	d, err := inst.OpenDocument(&requests.OpenDocument{File: &dados})
	if err != nil {
		return nil, classificarErro(err)
	}
	defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: d.Document})
	pc, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: d.Document})
	if err != nil {
		return nil, erros.Com(erros.PDFCorrompido, err)
	}
	out := make([][]byte, 0, len(faixas))
	for _, f := range faixas {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if f.Inicio < 1 || f.Fim < f.Inicio || f.Fim > pc.PageCount {
			return nil, erros.Com(erros.InternoInesperado, fmt.Errorf("faixa %d–%d fora de %d páginas", f.Inicio, f.Fim, pc.PageCount))
		}
		idx := make([]int, 0, f.Fim-f.Inicio+1)
		for i := f.Inicio; i <= f.Fim; i++ {
			idx = append(idx, i-1)
		}
		novo, err := inst.FPDF_CreateNewDocument(&requests.FPDF_CreateNewDocument{})
		if err != nil {
			return nil, erros.Com(erros.InternoInesperado, err)
		}
		_, err = inst.FPDF_ImportPagesByIndex(&requests.FPDF_ImportPagesByIndex{Source: d.Document, Destination: novo.Document, PageIndices: idx, Index: 0})
		var salvo *responses.FPDF_SaveAsCopy
		if err == nil {
			salvo, err = inst.FPDF_SaveAsCopy(&requests.FPDF_SaveAsCopy{Document: novo.Document})
		}
		inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: novo.Document})
		if err != nil || salvo == nil || salvo.FileBytes == nil {
			return nil, erros.Com(erros.InternoInesperado, fmt.Errorf("separar páginas %d–%d: %v", f.Inicio, f.Fim, err))
		}
		out = append(out, *salvo.FileBytes)
	}
	return out, nil
}

// ContarPaginas abre o PDF só para conferir que é legível e contar as páginas.
func ContarPaginas(ctx context.Context, e Extrator, dados []byte) (int, error) {
	d, err := e.Abrir(ctx, dados)
	if err != nil {
		return 0, err
	}
	defer d.Fechar()
	return d.Paginas(), nil
}

package autos

import (
	"context"
	"image"
	"image/color"
	"runtime"
	"strings"
	"sync"
	"unicode"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/ocr"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
)

// Extracao configura a leitura dos autos.
type Extracao struct {
	PDF     pdf.Extrator
	OCR     ocr.Motor // nil: páginas que exigem OCR fazem a extração falhar (OCR-001)
	DPI     int       // resolução do OCR (padrão 300)
	Workers int       // OCR em paralelo (padrão: metade dos núcleos)
}

// Extrair lê todas as páginas. progresso(atual, total) é chamado a cada
// página concluída (métrica real para a interface).
func (e Extracao) Extrair(ctx context.Context, dados []byte, progresso func(atual, total int)) (*Autos, error) {
	if progresso == nil {
		progresso = func(int, int) {}
	}
	dpi := e.DPI
	if dpi <= 0 {
		dpi = 300
	}
	workers := e.Workers
	if workers <= 0 {
		workers = max(1, runtime.NumCPU()/2)
	}
	doc, err := e.PDF.Abrir(ctx, dados)
	if err != nil {
		return nil, err
	}
	defer doc.Fechar()

	n := doc.Paginas()
	a := &Autos{Paginas: make([]Pagina, n)}
	var (
		mu     sync.Mutex
		feitas int
	)
	concluir := func() {
		mu.Lock()
		feitas++
		f := feitas
		mu.Unlock()
		progresso(f, n)
	}

	// 1ª passada: texto nativo (rápido) e seleção das páginas para OCR.
	var paraOCR []int
	for i := 0; i < n; i++ {
		p, err := doc.Pagina(ctx, i)
		if err != nil {
			return nil, err
		}
		texto, frs := montarTextoNativo(p)
		pg := Pagina{Numero: i + 1, Largura: p.Largura, Altura: p.Altura, Carimbos: LerCarimbos(texto)}
		// O carimbo do SEI é texto mesmo numa página escaneada: não conta como conteúdo.
		if TextoUtil(SemCarimbos(texto)) {
			pg.Texto, pg.Fragmentos, pg.Origem, pg.Situacao = texto, frs, OrigemNativo, Lida
			a.Paginas[i] = pg
			concluir()
			continue
		}
		a.Paginas[i] = pg
		paraOCR = append(paraOCR, i)
	}
	if len(paraOCR) == 0 {
		return a, nil
	}

	// 2ª passada: renderização (sequencial, uma instância do pdfium) e OCR
	// em paralelo (processos separados).
	type trabalho struct {
		i   int
		img image.Image
	}
	fila := make(chan trabalho, workers)
	ctx, cancelar := context.WithCancel(ctx)
	defer cancelar()
	var (
		wg       sync.WaitGroup
		errOnce  sync.Once
		primeiro error
	)
	falhar := func(err error) {
		errOnce.Do(func() { primeiro = err; cancelar() })
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range fila {
				pg := a.Paginas[t.i]
				r, err := e.OCR.Reconhecer(ctx, t.img)
				if err != nil {
					falhar(err)
					continue
				}
				aplicarOCR(&pg, r, dpi)
				a.Paginas[t.i] = pg
				concluir()
			}
		}()
	}
	for _, i := range paraOCR {
		if ctx.Err() != nil {
			break
		}
		// Página em branco: detectada numa renderização leve, sem OCR.
		baixa, err := doc.Renderizar(ctx, i, 36)
		if err != nil {
			falhar(err)
			break
		}
		if emBranco(baixa) {
			a.Paginas[i].Situacao = Vazia
			concluir()
			continue
		}
		if e.OCR == nil {
			falhar(erros.Novo(erros.OCRFalhou))
			break
		}
		img, err := doc.Renderizar(ctx, i, dpi)
		if err != nil {
			falhar(err)
			break
		}
		select {
		case fila <- trabalho{i: i, img: cinza(img)}:
		case <-ctx.Done():
		}
	}
	close(fila)
	wg.Wait()
	if primeiro != nil {
		return nil, primeiro
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return a, nil
}

// aplicarOCR converte o resultado do OCR (pixels) para a página (pontos).
func aplicarOCR(pg *Pagina, r ocr.Resultado, dpi int) {
	pg.Origem = OrigemOCR
	pg.Confianca = r.Confianca
	escala := 72.0 / float64(dpi)
	var sb strings.Builder
	pos, linha := 0, -1
	for _, w := range r.Palavras {
		if linha >= 0 {
			if w.Linha != linha {
				sb.WriteByte('\n')
			} else {
				sb.WriteByte(' ')
			}
			pos++
		}
		linha = w.Linha
		n := len([]rune(w.Texto))
		sb.WriteString(w.Texto)
		pg.Fragmentos = append(pg.Fragmentos, Fragmento{Inicio: pos, Fim: pos + n, Caixa: pdf.Caixa{
			X0: float64(w.X0) * escala, Y0: float64(w.Y0) * escala,
			X1: float64(w.X1) * escala, Y1: float64(w.Y1) * escala,
		}})
		pos += n
	}
	pg.Texto = sb.String()
	switch {
	case len(r.Palavras) < 3 || r.Confianca < LimiarOCRIlegivel:
		pg.Situacao = Ilegivel
	case r.Confianca < LimiarOCRBom:
		pg.Situacao = OCRBaixo
	default:
		pg.Situacao = OCROk
	}
}

// palavrasComuns do português: texto real as contém; lixo de codificação não.
var palavrasComuns = map[string]bool{
	"de": true, "da": true, "do": true, "das": true, "dos": true, "que": true, "e": true,
	"o": true, "a": true, "os": true, "as": true, "em": true, "no": true, "na": true,
	"por": true, "para": true, "com": true, "se": true, "não": true, "ao": true, "um": true, "uma": true,
}

// TextoUtil decide se a camada de texto nativa é aproveitável ou se a página
// precisa de OCR (página escaneada, texto vazio ou com codificação quebrada).
func TextoUtil(texto string) bool {
	var letras, total int
	for _, r := range texto {
		if unicode.IsSpace(r) {
			continue
		}
		total++
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letras++
		}
	}
	if letras < 20 {
		return false // praticamente sem texto: provavelmente imagem
	}
	if float64(letras)/float64(total) < 0.6 {
		return false // excesso de símbolos: codificação quebrada
	}
	palavras := strings.FieldsFunc(strings.ToLower(texto), func(r rune) bool { return !unicode.IsLetter(r) })
	if len(palavras) < 30 {
		return true // pouco texto para avaliar o idioma (títulos, carimbos)
	}
	comuns := 0
	for _, p := range palavras {
		if palavrasComuns[p] {
			comuns++
		}
	}
	return float64(comuns)/float64(len(palavras)) >= 0.04
}

// emBranco: menos de 0,2% de pixels escuros, ignorando a faixa inferior de
// 12% da página, onde o SEI imprime o carimbo (que existe até em página vazia).
func emBranco(img image.Image) bool {
	b := img.Bounds()
	limite := b.Min.Y + b.Dy()*88/100
	escuros, total := 0, b.Dx()*(limite-b.Min.Y)
	if total <= 0 {
		return true
	}
	for y := b.Min.Y; y < limite; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if color.GrayModel.Convert(img.At(x, y)).(color.Gray).Y < 160 {
				escuros++
			}
		}
	}
	return float64(escuros)/float64(total) < 0.002
}

// cinza reduz a imagem a tons de cinza (PNG 4× menor para o OCR).
func cinza(img image.Image) image.Image {
	b := img.Bounds()
	g := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			g.Set(x, y, img.At(x, y))
		}
	}
	return g
}

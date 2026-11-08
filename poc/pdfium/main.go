// PoC F0-04 — go-pdfium (WebAssembly/wazero, sem CGO): texto por página com posições e renderização.
// Uso: go run . <arquivo.pdf> [workers]
package main

import (
	"fmt"
	"image/png"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
)

func main() {
	path := os.Args[1]
	workers := 1
	if len(os.Args) > 2 {
		workers, _ = strconv.Atoi(os.Args[2])
	}
	data, err := os.ReadFile(path)
	must(err)

	t0 := time.Now()
	pool, err := webassembly.Init(webassembly.Config{MinIdle: 1, MaxIdle: workers, MaxTotal: workers})
	must(err)
	defer pool.Close()
	fmt.Printf("init: %v\n", time.Since(t0))

	inst, err := pool.GetInstance(30 * time.Second)
	must(err)
	doc, err := inst.OpenDocument(&requests.OpenDocument{File: &data})
	must(err)
	pc, err := inst.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	must(err)
	n := pc.PageCount
	fmt.Printf("páginas: %d\n", n)
	inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})
	inst.Close()

	// Extração paralela: cada worker abre o documento na sua instância.
	t1 := time.Now()
	var mu sync.Mutex
	totalChars, totalRects := 0, 0
	var wg sync.WaitGroup
	pages := make(chan int, n)
	for i := 0; i < n; i++ {
		pages <- i
	}
	close(pages)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			in, err := pool.GetInstance(30 * time.Second)
			must(err)
			defer in.Close()
			d, err := in.OpenDocument(&requests.OpenDocument{File: &data})
			must(err)
			defer in.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: d.Document})
			for p := range pages {
				st, err := in.GetPageTextStructured(&requests.GetPageTextStructured{
					Page: requests.Page{ByIndex: &requests.PageByIndex{Document: d.Document, Index: p}},
					Mode: requests.GetPageTextStructuredModeRects,
				})
				must(err)
				mu.Lock()
				totalRects += len(st.Rects)
				for _, r := range st.Rects {
					totalChars += len([]rune(r.Text))
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	el := time.Since(t1)
	fmt.Printf("texto estruturado (rects): %v total, %.1f ms/página, %d rects, %d chars, workers=%d\n",
		el, float64(el.Milliseconds())/float64(n), totalRects, totalChars, workers)

	// Amostra de texto + posição da 1ª página.
	in, _ := pool.GetInstance(30 * time.Second)
	d, _ := in.OpenDocument(&requests.OpenDocument{File: &data})
	st, _ := in.GetPageTextStructured(&requests.GetPageTextStructured{
		Page: requests.Page{ByIndex: &requests.PageByIndex{Document: d.Document, Index: 0}},
		Mode: requests.GetPageTextStructuredModeRects,
	})
	for i, r := range st.Rects {
		if i >= 4 {
			break
		}
		fmt.Printf("  rect %q @ (%.0f,%.0f)-(%.0f,%.0f)\n", r.Text, r.PointPosition.Left, r.PointPosition.Top, r.PointPosition.Right, r.PointPosition.Bottom)
	}

	// Renderização para OCR (200 dpi, 1ª página).
	t2 := time.Now()
	img, err := in.RenderPageInDPI(&requests.RenderPageInDPI{
		Page: requests.Page{ByIndex: &requests.PageByIndex{Document: d.Document, Index: 0}}, DPI: 200,
	})
	must(err)
	fmt.Printf("render 200dpi pág.1: %v (%dx%d)\n", time.Since(t2), img.Result.Image.Bounds().Dx(), img.Result.Image.Bounds().Dy())
	if out := os.Getenv("RENDER_OUT"); out != "" {
		f, _ := os.Create(out)
		png.Encode(f, img.Result.Image)
		f.Close()
	}
	img.Cleanup()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	fmt.Printf("heap Go: %d MB (sys %d MB)\n", ms.HeapAlloc>>20, ms.Sys>>20)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

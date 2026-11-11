// bibgen gera o acervo inicial da biblioteca jurídica (V4 §40.2; plano §10):
// lê os PDFs de docs/law_library identificados no manifesto, extrai o texto,
// separa os dispositivos e grava internal/legal/dados/acervo.json.gz, que é
// embutido no executável.
//
// Uso (na raiz do repositório):
//
//	go run ./cmd/bibgen                       # gera o acervo
//	go run ./cmd/bibgen -conferir             # falha se o acervo estiver desatualizado
//	go run ./cmd/bibgen -texto <pasta>        # grava também o texto extraído, para conferência
//	go run ./cmd/bibgen -despejar <arq.pdf>   # só escreve o texto de um PDF na saída padrão
//
// Originais escaneados precisam de OCR: -tesseract <executável> e -tessdata
// <pasta> (no WSL, o tesseract.exe do Windows serve, com a pasta em formato
// Windows). Sem OCR, eles entram no acervo marcados como "sem texto".
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/ocr"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/legal"
)

var (
	lib       = flag.String("lib", "docs/law_library", "pasta dos originais")
	saida     = flag.String("saida", "internal/legal/dados/acervo.json.gz", "arquivo do acervo")
	conferir  = flag.Bool("conferir", false, "não grava; falha se o acervo gravado difere do gerado")
	texto     = flag.String("texto", "", "pasta para gravar o texto extraído de cada original")
	despejar  = flag.String("despejar", "", "escreve o texto de um PDF e termina")
	tesseract = flag.String("tesseract", "", "executável do Tesseract (OCR dos originais escaneados)")
	tessdata  = flag.String("tessdata", "", "pasta tessdata do Tesseract")
	dpi       = flag.Int("dpi", 300, "resolução do OCR")
)

func main() {
	flag.Parse()
	if err := executar(); err != nil {
		fmt.Fprintln(os.Stderr, "bibgen:", err)
		os.Exit(1)
	}
}

func executar() error {
	ctx := context.Background()
	ex := pdf.NovoExtrator()
	defer ex.Fechar()
	var motor ocr.Motor
	if *tesseract != "" {
		motor = &ocr.Tesseract{Executavel: *tesseract, Tessdata: *tessdata, Idioma: "por"}
	}
	if *despejar != "" {
		b, err := os.ReadFile(*despejar)
		if err != nil {
			return err
		}
		pags, _, err := ler(ctx, ex, motor, b)
		if err != nil {
			return err
		}
		fmt.Print(emTexto(pags))
		return nil
	}
	m, err := legal.LerManifesto()
	if err != nil {
		return err
	}
	// Todo PDF da pasta tem de estar no manifesto: nada entra sem identificação.
	noManifesto := map[string]bool{}
	for _, d := range m.Documentos {
		noManifesto[d.Arquivo] = true
	}
	arqs, _ := filepath.Glob(filepath.Join(*lib, "*.pdf"))
	for _, a := range arqs {
		if !noManifesto[filepath.Base(a)] {
			return fmt.Errorf("%q está em %s e não está no manifesto (internal/legal/dados/manifesto.yaml)", filepath.Base(a), *lib)
		}
	}
	anterior, _ := legal.AcervoInicial()
	ac := legal.Acervo{Versao: m.Versao}
	fmt.Printf("%-22s %5s %6s %8s %10s  %s\n", "documento", "págs", "partes", "artigos", "unidades", "observação")
	for _, d := range m.Documentos {
		b, err := os.ReadFile(filepath.Join(*lib, d.Arquivo))
		if err != nil {
			return fmt.Errorf("%s: %w", d.ID, err)
		}
		h := sha256.Sum256(b)
		d.SHA256 = hex.EncodeToString(h[:])
		// Original inalterado e já lido por OCR: aproveita o texto do acervo
		// anterior (o OCR não precisa estar disponível em toda máquina).
		if ant := doAcervo(anterior, d.ID); ant != nil && ant.SHA256 == d.SHA256 && ant.PaginasOCR > 0 && motor == nil {
			d.Paginas, d.PaginasOCR, d.OCR, d.Partes = ant.Paginas, ant.PaginasOCR, ant.OCR, ant.Partes
		} else {
			pags, nOCR, err := ler(ctx, ex, motor, b)
			if err != nil {
				return fmt.Errorf("%s: %w", d.ID, err)
			}
			d.Paginas, d.PaginasOCR, d.OCR = len(pags), nOCR, nOCR*2 > len(pags)
			d.Partes = legal.Analisar(pags)
			if *texto != "" {
				if err := os.MkdirAll(*texto, 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(*texto, d.ID+".txt"), []byte(emTexto(pags)), 0o644); err != nil {
					return err
				}
			}
		}
		d.SemTexto = len(d.Partes) == 0
		arts, unid := 0, 0
		for _, p := range d.Partes {
			arts += p.Artigos()
			unid += len(p.Unidades)
		}
		obs := ""
		switch {
		case d.SemTexto:
			obs = "SEM TEXTO (escaneado; rode com -tesseract)"
		case d.OCR:
			obs = "texto por OCR"
		case d.PaginasOCR > 0:
			obs = fmt.Sprintf("%d página(s) por OCR", d.PaginasOCR)
		}
		fmt.Printf("%-22s %5d %6d %8d %10d  %s\n", d.ID, d.Paginas, len(d.Partes), arts, unid, obs)
		ac.Documentos = append(ac.Documentos, d)
	}
	gz, err := ac.Gravar()
	if err != nil {
		return err
	}
	if *conferir {
		atual, _ := os.ReadFile(*saida)
		if !bytes.Equal(atual, gz) {
			return fmt.Errorf("%s está desatualizado: rode go run ./cmd/bibgen", *saida)
		}
		fmt.Println("acervo conferido: em dia")
		return nil
	}
	if err := os.WriteFile(*saida, gz, 0o644); err != nil {
		return err
	}
	fmt.Printf("gravado %s (%d bytes)\n", *saida, len(gz))
	return nil
}

func doAcervo(a *legal.Acervo, id string) *legal.Documento {
	if a == nil {
		return nil
	}
	for i := range a.Documentos {
		if a.Documentos[i].ID == id {
			return &a.Documentos[i]
		}
	}
	return nil
}

// ler extrai o texto de cada página; página sem texto passa pelo OCR, se
// houver motor. Devolve também quantas páginas vieram do OCR.
func ler(ctx context.Context, ex pdf.Extrator, motor ocr.Motor, b []byte) ([]legal.PaginaTexto, int, error) {
	doc, err := ex.Abrir(ctx, b)
	if err != nil {
		return nil, 0, err
	}
	defer doc.Fechar()
	var out []legal.PaginaTexto
	nOCR := 0
	for i := 0; i < doc.Paginas(); i++ {
		p, err := doc.Pagina(ctx, i)
		if err != nil {
			return nil, 0, err
		}
		rs := make([]rune, len(p.Caracteres))
		for k, c := range p.Caracteres {
			rs[k] = c.Runa
		}
		t := strings.ReplaceAll(string(rs), "\r", "")
		if len(strings.TrimSpace(t)) < 40 && motor != nil {
			img, err := doc.Renderizar(ctx, i, *dpi)
			if err != nil {
				return nil, 0, err
			}
			r, err := motor.Reconhecer(ctx, img)
			if err != nil {
				return nil, 0, fmt.Errorf("OCR da página %d: %w", i+1, err)
			}
			t = r.Texto()
			nOCR++
		}
		out = append(out, legal.PaginaTexto{Numero: i + 1, Texto: t})
	}
	return out, nOCR, nil
}

func emTexto(pags []legal.PaginaTexto) string {
	var sb strings.Builder
	for _, p := range pags {
		fmt.Fprintf(&sb, "\n===== página %d =====\n%s", p.Numero, p.Texto)
	}
	return sb.String()
}

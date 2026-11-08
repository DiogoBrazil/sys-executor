// Comando autos-mapa (pendência P-09): mostra a ESTRUTURA de um PDF
// consolidado do SEI para calibrar a segmentação de peças, sem revelar o
// conteúdo. Executado pelo próprio usuário, no computador dele.
//
// Imprime só indicadores: se a página tem texto, onde aparecem os marcadores
// do SEI (rodapé "SEI nº", "código verificador", "assinado eletronicamente",
// "Referência:") e a função reconhecida de cada peça. Nenhum texto dos autos é
// impresso, e todos os dígitos saem mascarados. Nada é gravado em disco.
//
// Uso (PowerShell): .\autos-mapa.exe "C:\caminho\do\processo.pdf"
package main

import (
	"context"
	"fmt"
	"image"
	"os"
	"regexp"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/ocr"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/segment"
	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

var marcadores = []struct {
	nome string
	re   *regexp.Regexp
}{
	{"SEI_N", regexp.MustCompile(`sei\s*n[º°o.]*\s*:?\s*[0-9]{6,10}`)},
	{"VERIFICADOR", regexp.MustCompile(`codigo verificador`)},
	{"ASSINADO", regexp.MustCompile(`assinado eletronicamente`)},
	{"REFERENCIA", regexp.MustCompile(`referencia:?\s*processo`)},
	{"CRC", regexp.MustCompile(`codigo crc`)},
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println(`Uso: autos-mapa.exe "C:\caminho\do\processo.pdf"`)
		os.Exit(2)
	}
	dados, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("Não foi possível ler o arquivo.")
		os.Exit(1)
	}
	ext := pdf.NovoExtrator()
	defer ext.Fechar()
	ctx := context.Background()

	// Só a camada de texto (sem OCR): páginas escaneadas aparecem como SEM_TEXTO.
	a, err := autos.Extracao{PDF: ext, OCR: nada{}}.Extrair(ctx, dados, nil)
	if err != nil {
		fmt.Println("Não foi possível ler o PDF:", mascarar(err.Error()))
		os.Exit(1)
	}
	fmt.Println("== sys-executor — estrutura do PDF (sem conteúdo; dígitos mascarados) ==")
	fmt.Printf("páginas: %d\n\n", len(a.Paginas))
	fmt.Println("pág | texto      | marcadores (posição: T=topo, M=meio, R=rodapé)")
	seis := map[string]bool{}
	for _, p := range a.Paginas {
		sit := "SEM_TEXTO"
		if p.Situacao == autos.Vazia {
			sit = "em branco"
		} else if p.Origem == autos.OrigemNativo {
			sit = fmt.Sprintf("%-10s", faixa(len([]rune(p.Texto))))
		}
		norm := string(texto.Alinhado(p.Texto))
		var ms []string
		for _, m := range marcadores {
			for _, loc := range m.re.FindAllStringIndex(norm, -1) {
				ini := len([]rune(norm[:loc[0]]))
				fim := len([]rune(norm[:loc[1]]))
				ms = append(ms, m.nome+"@"+posicao(p, ini, fim))
				if m.nome == "SEI_N" {
					seis[norm[loc[0]:loc[1]]] = true
				}
			}
		}
		fmt.Printf("%3d | %-10s | %s\n", p.Numero, sit, strings.Join(ms, " "))
	}
	fmt.Printf("\nnúmeros SEI distintos encontrados: %d\n\n", len(seis))
	fmt.Println("Peças (segmentação atual):")
	for _, pc := range segment.Segmentar(a) {
		sei := "sem SEI"
		if pc.SEI != "" {
			sei = "SEI " + mascarar(pc.SEI)
		}
		fmt.Printf("  %s pp. %d-%d  %-26s %s externo=%v provável=%v\n", pc.ID, pc.PaginaInicial, pc.PaginaFinal, pc.Funcao, sei, pc.Externo, pc.Provavel)
	}
	fmt.Println("\n== fim. Revise e cole este resultado na conversa. ==")
}

// nada é um OCR que não reconhece nada (a ferramenta não usa OCR).
type nada struct{}

func (nada) Reconhecer(context.Context, image.Image) (ocr.Resultado, error) {
	return ocr.Resultado{}, nil
}

func faixa(n int) string {
	switch {
	case n == 0:
		return "vazia"
	case n < 300:
		return "pouco"
	case n < 1500:
		return "médio"
	default:
		return "muito"
	}
}

// posicao classifica a altura do trecho na página (terço superior, meio, rodapé).
func posicao(p autos.Pagina, ini, fim int) string {
	cx := p.Caixas(ini, fim)
	if len(cx) == 0 || p.Altura == 0 {
		return "?"
	}
	y := cx[0].Y0 / p.Altura
	switch {
	case y < 0.15:
		return "T"
	case y > 0.85:
		return "R"
	default:
		return "M"
	}
}

var digitos = regexp.MustCompile(`[0-9]`)

func mascarar(s string) string { return digitos.ReplaceAllString(s, "#") }

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

// Marcadores tolerantes a quebras de linha e espaços extras entre as palavras.
var marcadores = []struct {
	nome string
	re   *regexp.Regexp
}{
	{"SEI_N", regexp.MustCompile(`sei\s*n[º°o.]*\s*:?\s*[0-9]{6,10}`)},
	{"SEI_PAR", regexp.MustCompile(`\(\s*[0-9]{7,10}\s*\)`)}, // número SEI entre parênteses, como na árvore
	{"VERIFICADOR", regexp.MustCompile(`codigo\s+verificador`)},
	{"ASSINADO", regexp.MustCompile(`assinad[oa]\s+eletronicamente`)},
	{"REFERENCIA", regexp.MustCompile(`referencia\s*:?\s*processo`)},
	{"CRC", regexp.MustCompile(`codigo\s+crc`)},
	{"AUTENTICIDADE", regexp.MustCompile(`autenticidade\s+(deste|do)\s+documento`)},
	{"PROCESSO_N", regexp.MustCompile(`processo\s+n[º°o.]*\s*:?\s*[0-9]{4}\.[0-9]{6}/[0-9]{4}`)},
}

// vocabulario: palavras genéricas que podem aparecer no esqueleto das linhas.
// Qualquer outra palavra (nomes, lugares, conteúdo) é trocada por "▒".
var vocabulario = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`sei n nº no numero processo documento documentos assinado assinada eletronicamente por em as às
		conforme horario oficial de da do das dos e a o os as na no com para que se brasilia fundamento art artigo decreto lei
		autenticidade deste pode ser conferida site informando codigo verificador crc referencia pagina pag pg folha fls fl
		governo estado rondonia policia militar secretaria seguranca defesa cidadania comando geral corregedoria
		memorando portaria termo notificacao ata despacho oficio certidao defesa previa alegacoes finais manifestacao
		anexo inquiricao declaracoes declaracao videoconferencia presenca remessa relatorio sindicancia disciplinar
		administrativo pads sr julgamento parecer requerimento peticao citacao intimacao mandado oitiva testemunha
		interrogatorio qualificacao ciencia sobre recibo protocolo externo interno capa volume pdf gerado
		emissao data hora usuario unidade orgao assinatura eletronica nome cargo funcao matricula cpf rg`) {
		vocabulario[w] = true
	}
}

var reURL = regexp.MustCompile(`https?://\S+|www\.\S+`)

// esqueleto mostra a forma de uma linha sem o conteúdo.
func esqueleto(linha string) string {
	linha = reURL.ReplaceAllString(linha, " [url] ")
	var out []string
	for _, tok := range strings.Fields(linha) {
		if tok == "[url]" {
			out = append(out, tok)
			continue
		}
		limpo := strings.Trim(texto.SemAcentos(tok), ".,;:()[]\"'“”«»-–—/|")
		switch {
		case limpo == "":
			out = append(out, mascarar(tok))
		case vocabulario[limpo]:
			out = append(out, mascarar(tok))
		case regexp.MustCompile(`^[0-9./\-]+$`).MatchString(limpo):
			out = append(out, mascarar(tok))
		default:
			out = append(out, "▒")
		}
	}
	return strings.Join(out, " ")
}

func linhasNaoVazias(t string) []string {
	var ls []string
	for _, l := range strings.Split(t, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			ls = append(ls, l)
		}
	}
	return ls
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
	fmt.Printf("\nnúmeros SEI distintos encontrados: %d\n", len(seis))

	// Esqueleto do início e do fim de cada página (cabeçalhos, títulos, rodapés).
	repet := map[string]int{}
	type borda struct{ ini, fim []string }
	bordas := make([]borda, len(a.Paginas))
	for i, p := range a.Paginas {
		ls := linhasNaoVazias(p.Texto)
		var b borda
		for k := 0; k < len(ls) && k < 2; k++ {
			b.ini = append(b.ini, esqueleto(ls[k]))
		}
		for k := max(len(b.ini), len(ls)-3); k < len(ls); k++ {
			b.fim = append(b.fim, esqueleto(ls[k]))
		}
		for _, l := range append(append([]string{}, b.ini...), b.fim...) {
			repet[l]++
		}
		bordas[i] = b
	}
	marca := func(l string) string {
		if repet[l] >= 3 {
			return "[repete x" + fmt.Sprint(repet[l]) + "] " + l
		}
		return l
	}
	fmt.Println("\nEsqueleto das bordas (▒ = palavra omitida; # = dígito):")
	for i, b := range bordas {
		fmt.Printf("--- pág. %d\n", i+1)
		for _, l := range b.ini {
			fmt.Println("  ^ " + marca(l))
		}
		for _, l := range b.fim {
			fmt.Println("  v " + marca(l))
		}
	}
	fmt.Println()
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

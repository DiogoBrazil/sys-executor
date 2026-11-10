package pdfsint_test

// Gera os PDFs sintéticos usados no teste manual no Windows (só dados
// fictícios de internal/testutil/casos). Não roda na suíte normal:
//
//	SAIDA_PDF=/mnt/c/Users/<usuário>/Documents/testes-sys-executor go test -run TestGerarPDFsSinteticos ./internal/testutil/pdfsint
//
// Cria, para os casos pads-02-duas-imputacoes e sr-01-extravio, o PDF
// consolidado (fluxo SEI legado) e uma pasta com um PDF por peça (cadastro
// de peças, V4 §48.4).

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/segment"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/casos"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/pdfsint"
)

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

func TestGerarPDFsSinteticos(t *testing.T) {
	dir := os.Getenv("SAIDA_PDF")
	if dir == "" {
		t.Skip("defina SAIDA_PDF para gerar os PDFs sintéticos")
	}
	alvos := map[string]string{"pads-02-duas-imputacoes": "pads-sintetico", "sr-01-extravio": "sr-sintetico"}
	nomes := map[segment.Funcao]string{segment.MemorandoDisciplinar: "memorando", segment.PortariaSR: "portaria",
		segment.TermoInquiricao: "termo-inquiricao", segment.TermoDeclaracoes: "termo-declaracoes",
		segment.AlegacoesFinais: "alegacoes-finais", segment.DefesaPrevia: "defesa-previa"}
	for _, c := range casos.Todos() {
		base, ok := alvos[c.Nome]
		if !ok {
			continue
		}
		pagina := func(n int, sei string) pdfsint.Pagina {
			return pdfsint.Pagina{Linhas: quebrarLinhas(c.Autos.Paginas[n-1].Texto, 72), Rodape: "SEI nº " + sei}
		}
		var todas []pdfsint.Pagina
		pasta := filepath.Join(dir, "pecas-"+base)
		if err := os.MkdirAll(pasta, 0o755); err != nil {
			t.Fatal(err)
		}
		for i, p := range c.Pecas {
			var pgs []pdfsint.Pagina
			for n := p.PaginaInicial; n <= p.PaginaFinal; n++ {
				pgs = append(pgs, pagina(n, p.SEI))
			}
			todas = append(todas, pgs...)
			nome := fmt.Sprintf("%02d-%s.pdf", i+1, nomes[p.Funcao])
			if err := os.WriteFile(filepath.Join(pasta, nome), pdfsint.Gerar(pgs), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, base+"-texto.pdf"), pdfsint.Gerar(todas), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: %d páginas, %d peças", base, len(todas), len(c.Pecas))
	}
}

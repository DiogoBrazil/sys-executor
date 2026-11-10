package pdf

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/testutil/pdfsint"
)

var (
	extOnce sync.Once
	ext     Extrator
)

// extrator compartilhado: o runtime WebAssembly leva ~2 s para iniciar.
func extrator(t *testing.T) Extrator {
	t.Helper()
	extOnce.Do(func() { ext = NovoExtrator() })
	return ext
}

func TestMain(m *testing.M) {
	c := m.Run()
	if ext != nil {
		_ = ext.Fechar()
	}
	os.Exit(c)
}

func abrirFixture(t *testing.T, nome string) Documento {
	t.Helper()
	b, err := os.ReadFile("../../../testdata/autos/" + nome)
	if err != nil {
		t.Fatal(err)
	}
	d, err := extrator(t).Abrir(context.Background(), b)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(d.Fechar)
	return d
}

func TestCoordenadasNoTopoEsquerdo(t *testing.T) {
	d := abrirFixture(t, "coordenadas.pdf")
	if d.Paginas() != 2 {
		t.Fatalf("páginas: %d", d.Paginas())
	}
	p, err := d.Pagina(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if p.Largura < 590 || p.Largura > 600 || p.Altura < 838 || p.Altura > 845 {
		t.Fatalf("A4 esperado, veio %.1fx%.1f", p.Largura, p.Altura)
	}
	var texto strings.Builder
	var caixas []Caixa
	for _, c := range p.Caracteres {
		texto.WriteRune(c.Runa)
		caixas = append(caixas, c.Caixa)
		if !c.SemCaixa && (c.Caixa.X0 > c.Caixa.X1 || c.Caixa.Y0 > c.Caixa.Y1) {
			t.Errorf("caixa degenerada: %+v", c)
		}
	}
	rs := []rune(texto.String())
	achar := func(s string) *Caixa {
		k := strings.Index(string(rs), s)
		if k < 0 {
			return nil
		}
		return &caixas[len([]rune(string(rs)[:k]))]
	}
	topo, base := achar("TOPO"), achar("BASE")
	if topo == nil || base == nil {
		t.Fatalf("texto: %q", string(rs))
	}
	// Margem de 2 cm ≈ 56,7 pt.
	if topo.Y0 > 80 || topo.X0 > 70 {
		t.Errorf("TOPO deveria estar no alto à esquerda: %+v", *topo)
	}
	if base.Y1 < p.Altura-80 || base.X1 < p.Largura-200 {
		t.Errorf("BASE deveria estar embaixo à direita: %+v", *base)
	}
	vazia, err := d.Pagina(context.Background(), 1)
	if err != nil || len(vazia.Caracteres) != 0 {
		t.Fatalf("página 2 deveria ser vazia: %+v %v", vazia.Caracteres, err)
	}
	img, err := d.Renderizar(context.Background(), 0, 72)
	if err != nil || img.Bounds().Dx() < 590 || img.Bounds().Dx() > 600 {
		t.Fatalf("render: %v %v", img.Bounds(), err)
	}
}

func TestErrosDeAbertura(t *testing.T) {
	_, err := extrator(t).Abrir(context.Background(), []byte("%PDF-1.7\nlixo sem estrutura"))
	if !errors.Is(err, erros.Novo(erros.PDFCorrompido)) {
		t.Fatalf("esperado PDF-003, veio %v", err)
	}
	if _, err := extrator(t).Abrir(context.Background(), pdfComSenha()); !errors.Is(err, erros.Novo(erros.PDFProtegido)) {
		t.Fatalf("esperado PDF-004, veio %v", err)
	}
}

// pdfComSenha monta um PDF mínimo com /Encrypt (RC4 40 bits) cujo valor /U não
// confere com a senha vazia: o leitor precisa pedir senha.
func pdfComSenha() []byte {
	var b strings.Builder
	offs := []int{}
	obj := func(s string) { offs = append(offs, b.Len()); b.WriteString(s) }
	b.WriteString("%PDF-1.4\n")
	obj("1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n")
	obj("2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n")
	obj("3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]>>endobj\n")
	h := strings.Repeat("A1", 32)
	obj("4 0 obj<</Filter/Standard/V 1/R 2/O<" + h + ">/U<" + h + ">/P -4>>endobj\n")
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offs)+1)
	for _, o := range offs {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer<</Size %d/Root 1 0 R/Encrypt 4 0 R/ID[<00112233445566778899AABBCCDDEEFF><00112233445566778899AABBCCDDEEFF>]>>\nstartxref\n%d\n%%%%EOF\n", len(offs)+1, xref)
	return []byte(b.String())
}

func TestContextoCancelado(t *testing.T) {
	d := abrirFixture(t, "coordenadas.pdf")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := d.Pagina(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("esperado cancelamento, veio %v", err)
	}
}

// Cadastro de peças (V4 §48.4): vários arquivos viram um PDF só, na ordem
// dada, com a contagem de páginas de cada um; o texto de cada página é
// preservado.
func TestJuntarPreservaOrdemETexto(t *testing.T) {
	e := NovoExtrator()
	defer e.Fechar()
	j := e.(Juntador)
	a := pdfsint.Gerar([]pdfsint.Pagina{{Linhas: []string{"PRIMEIRO ARQUIVO pagina um"}}, {Linhas: []string{"PRIMEIRO ARQUIVO pagina dois"}}})
	b := pdfsint.Gerar([]pdfsint.Pagina{{Linhas: []string{"SEGUNDO ARQUIVO pagina unica"}}})
	junto, pags, err := j.Juntar(context.Background(), [][]byte{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if len(pags) != 2 || pags[0] != 2 || pags[1] != 1 {
		t.Fatalf("páginas %v", pags)
	}
	d, err := e.Abrir(context.Background(), junto)
	if err != nil {
		t.Fatal(err)
	}
	if d.Paginas() != 3 {
		t.Fatalf("%d páginas", d.Paginas())
	}
	for i, quer := range []string{"PRIMEIRO ARQUIVO pagina um", "PRIMEIRO ARQUIVO pagina dois", "SEGUNDO ARQUIVO pagina unica"} {
		p, err := d.Pagina(context.Background(), i)
		var txt strings.Builder
		for _, c := range p.Caracteres {
			txt.WriteRune(c.Runa)
		}
		if err != nil || !strings.Contains(txt.String(), quer) {
			t.Fatalf("página %d: %q %v", i+1, txt.String(), err)
		}
	}
	d.Fechar() // libera a instância (o pool tem uma)
	_, _, err = j.Juntar(context.Background(), [][]byte{a, []byte("não é pdf")})
	var ee *erros.Erro
	if !errors.As(err, &ee) || ee.Codigo != erros.PDFCorrompido {
		t.Fatalf("arquivo inválido deveria ser PDF-003: %v", err)
	}
	if n, err := ContarPaginas(context.Background(), e, a); err != nil || n != 2 {
		t.Fatalf("contar: %d %v", n, err)
	}
}

// Separar divide o PDF por faixas, na ordem e com o texto de cada página;
// faixa fora do documento é recusada (documentos do SEI, V4 §48.4).
func TestSepararPorFaixas(t *testing.T) {
	e := NovoExtrator()
	defer e.Fechar()
	s := e.(Separador)
	src := pdfsint.Gerar([]pdfsint.Pagina{{Linhas: []string{"DOC A pagina um"}}, {Linhas: []string{"DOC A pagina dois"}}, {Linhas: []string{"DOC B unica"}}})
	partes, err := s.Separar(context.Background(), src, []Faixa{{1, 2}, {3, 3}})
	if err != nil {
		t.Fatal(err)
	}
	quer := [][]string{{"DOC A pagina um", "DOC A pagina dois"}, {"DOC B unica"}}
	for k, parte := range partes {
		d, err := e.Abrir(context.Background(), parte)
		if err != nil {
			t.Fatal(err)
		}
		if d.Paginas() != len(quer[k]) {
			t.Fatalf("parte %d: %d páginas", k, d.Paginas())
		}
		for i, q := range quer[k] {
			p, err := d.Pagina(context.Background(), i)
			var txt strings.Builder
			for _, c := range p.Caracteres {
				txt.WriteRune(c.Runa)
			}
			if err != nil || !strings.Contains(txt.String(), q) {
				t.Fatalf("parte %d página %d: %q %v", k, i+1, txt.String(), err)
			}
		}
		d.Fechar()
	}
	if _, err := s.Separar(context.Background(), src, []Faixa{{2, 4}}); err == nil {
		t.Fatal("faixa fora do documento deveria falhar")
	}
}

package erros

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// ERR-xx: toda entrada do catálogo é completa, amigável e sem detalhes técnicos.
func TestCatalogoCompletoEAmigavel(t *testing.T) {
	formato := regexp.MustCompile(`^[A-Z]{2,4}-\d{3}$`)
	tecnico := regexp.MustCompile(`(?i)%[vsdq]|panic|stack|goroutine|nil pointer|exception|0x[0-9a-f]+|\berr(or)?\b|\.go\b`)
	vistos := map[Codigo]bool{}
	for _, d := range Definicoes() {
		if !formato.MatchString(string(d.Codigo)) {
			t.Errorf("%s: código fora do padrão", d.Codigo)
		}
		if vistos[d.Codigo] {
			t.Errorf("%s: duplicado", d.Codigo)
		}
		vistos[d.Codigo] = true
		for nome, txt := range map[string]string{"titulo": d.Titulo, "mensagem": d.Mensagem, "acao": d.Acao} {
			if strings.TrimSpace(txt) == "" {
				t.Errorf("%s: %s vazio", d.Codigo, nome)
			}
			if tecnico.MatchString(txt) {
				t.Errorf("%s: %s contém termo técnico: %q", d.Codigo, nome, txt)
			}
		}
		if d.Categoria == "" {
			t.Errorf("%s: sem categoria", d.Codigo)
		}
	}
	// Áreas exigidas pela V4 §33.
	exig := []Categoria{CatSEI, CatRede, CatPDF, CatOCR, CatArquivo, CatModelo, CatBiblioteca, CatValidacao, CatExportacao, CatPesquisa, CatSessao}
	for _, c := range exig {
		achou := false
		for _, d := range Definicoes() {
			if d.Categoria == c {
				achou = true
				break
			}
		}
		if !achou {
			t.Errorf("categoria %s sem nenhum código", c)
		}
	}
}

// Nenhuma mensagem sugere sucesso (V4 §33: nunca simular sucesso) nem apagamento forense (DOC-01).
func TestSemSucessoSimuladoNemPromessaForense(t *testing.T) {
	proibido := regexp.MustCompile(`(?i)sucesso|conclu[ií]d[oa] com|apagad[oa]s? (definitiva|permanente)|irrecuper|forense|garantid`)
	for _, d := range Definicoes() {
		if proibido.MatchString(d.Mensagem + " " + d.Acao) {
			t.Errorf("%s: texto proibido: %q / %q", d.Codigo, d.Mensagem, d.Acao)
		}
	}
}

func TestCausaNaoVazaNaDefinicao(t *testing.T) {
	segredo := errors.New("senha=SENTINELA-123")
	e := Com(PDFInvalido, segredo)
	if strings.Contains(e.Titulo+e.Mensagem+e.Acao, "SENTINELA") {
		t.Fatal("causa vazou para os textos de interface")
	}
	if !errors.Is(e, segredo) {
		t.Fatal("causa deve ser acessível por errors.Is para diagnóstico em memória")
	}
}

func TestDeEIs(t *testing.T) {
	if De(nil) != nil {
		t.Fatal("De(nil) deve ser nil")
	}
	e := De(fmt.Errorf("camada: %w", Novo(PDFVazio)))
	if e.Codigo != PDFVazio {
		t.Fatalf("esperado PDF-001, veio %s", e.Codigo)
	}
	if !errors.Is(e, Novo(PDFVazio)) || errors.Is(e, Novo(PDFInvalido)) {
		t.Fatal("errors.Is deve comparar pelo código")
	}
	if g := De(errors.New("x")); g.Codigo != InternoInesperado {
		t.Fatalf("erro comum deve virar INT-001, veio %s", g.Codigo)
	}
	if Novo("NAO-999").Codigo != InternoInesperado {
		t.Fatal("código desconhecido deve virar INT-001")
	}
}

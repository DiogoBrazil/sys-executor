package documento

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/minuta"
)

// Typst é o compilador empacotado que gera o PDF (ADR-007).
type Typst struct {
	Executavel string
	// Fontes: pasta com fontes adicionais (Liberation Serif, reserva métrica
	// da Times New Roman); "" quando não existe.
	Fontes string
}

// ErrSemTypst: o componente não está instalado.
var ErrSemTypst = fmt.Errorf("componente Typst não encontrado")

// LocalizarTypst procura o Typst em <componentes>\typst (instalação) ou em
// SYSEXEC_TYPST (desenvolvimento, aponta para o executável).
func LocalizarTypst(componentes string) (*Typst, error) {
	exe := os.Getenv("SYSEXEC_TYPST")
	if exe == "" {
		exe = filepath.Join(componentes, "typst", nomeTypst)
	}
	if st, err := os.Stat(exe); err != nil || st.IsDir() {
		return nil, ErrSemTypst
	}
	t := &Typst{Executavel: exe}
	if f := filepath.Join(filepath.Dir(exe), "fonts"); ehPasta(f) {
		t.Fontes = f
	}
	return t, nil
}

func ehPasta(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// textoTypst escreve o texto como literal de string do Typst: nada do
// conteúdo é interpretado como marcação ou código.
func textoTypst(s string) string {
	s = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(s)
	return `#"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func pedacoTypst(p pedaco) string {
	t := textoTypst(p.texto)
	if p.negrito {
		t = "#strong[" + t + "]"
	}
	if p.pendente {
		t = "#highlight(fill: rgb(\"#fff176\"))[" + t + "]"
	}
	return t
}

const nomeLogoTypst = "logotipo"

// FonteTypst gera o código-fonte Typst da minuta (mesma formatação do DOCX).
func FonteTypst(doc *minuta.Documento, cab Cabecalho) string {
	var b strings.Builder
	var linhas []string
	for _, l := range cab.Linhas {
		linhas = append(linhas, "#strong["+textoTypst(l)+"]")
	}
	cabecalho := strings.Join(linhas, " \\ ")
	alturaCab := 0.9 + 0.5*float64(len(cab.Linhas))
	if len(cab.Logo) > 0 {
		cabecalho = fmt.Sprintf("#image(%q, height: %.1fcm) #v(-0.5em) ", nomeLogoTypst+"."+cab.Ext, alturaLogoCm) + cabecalho
		alturaCab += alturaLogoCm + 0.2
	}
	fmt.Fprintf(&b, `#set document(title: "Minuta", author: ())
#set page(paper: "a4", margin: (top: 2cm + %.1fcm, bottom: 2cm, left: 3cm, right: 1.5cm), header-ascent: 0.4cm,
  header: align(center)[%s],
  footer: align(right)[#context counter(page).display()])
#set text(font: ("Times New Roman", "Liberation Serif"), size: 12pt, lang: "pt", region: "br", fill: black)
#set par(justify: true, leading: 0.65em, spacing: 1.1em)
`, alturaCab, cabecalho)
	for _, bl := range doc.Blocos {
		var t strings.Builder
		for _, p := range pedacos(bl) {
			t.WriteString(pedacoTypst(p))
		}
		c := t.String()
		switch estilo(bl) {
		case "Titulo":
			b.WriteString("#v(0.6em)\n#align(center)[" + c + "]\n#v(1em)\n")
		case "Identificacao":
			b.WriteString("#block(above: 0.4em, below: 0.4em)[#set par(justify: false)\n" + c + "]\n")
		case "Secao":
			b.WriteString("#block(above: 1.8em, below: 1.1em, sticky: true)[" + c + "]\n")
		case "Subtitulo":
			b.WriteString("#block(above: 1.3em, below: 1.1em, sticky: true)[" + c + "]\n")
		case "Citacao":
			b.WriteString("#pad(left: 4cm)[#set text(size: 10pt)\n" + c + "]\n")
		case "Data":
			b.WriteString("#v(1em)\n#align(right)[" + c + "]\n")
		case "Assinatura":
			b.WriteString("#block(above: 4em, width: 100%, breakable: false)[#align(center)[" + c + "]]\n")
		default:
			b.WriteString("#par(first-line-indent: (amount: 2.5cm, all: true))[" + c + "]\n")
		}
	}
	return b.String()
}

// PDF compila a minuta. pasta é um diretório de trabalho protegido (a sessão
// da tarefa): os arquivos intermediários são apagados ao final.
func (t *Typst) PDF(ctx context.Context, doc *minuta.Documento, cab Cabecalho, pasta string) ([]byte, error) {
	if doc == nil || len(doc.Blocos) == 0 {
		return nil, fmt.Errorf("minuta vazia")
	}
	dir, err := os.MkdirTemp(pasta, "pdf-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "minuta.typ"), []byte(FonteTypst(doc, cab)), 0o600); err != nil {
		return nil, err
	}
	if len(cab.Logo) > 0 {
		if err := os.WriteFile(filepath.Join(dir, nomeLogoTypst+"."+cab.Ext), cab.Logo, 0o600); err != nil {
			return nil, err
		}
	}
	args := []string{"compile", "--root", dir}
	if t.Fontes != "" {
		args = append(args, "--font-path", t.Fontes)
	}
	args = append(args, filepath.Join(dir, "minuta.typ"), filepath.Join(dir, "minuta.pdf"))
	cmd := exec.CommandContext(ctx, t.Executavel, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr // só para o diagnóstico em memória; nada é gravado
	ocultarJanela(cmd)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		msg := stderr.String()
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return nil, fmt.Errorf("typst: %w: %s", err, msg)
	}
	return os.ReadFile(filepath.Join(dir, "minuta.pdf"))
}

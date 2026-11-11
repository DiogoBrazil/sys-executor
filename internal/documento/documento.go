// Package documento gera os arquivos da minuta (V4 §32; ADR-007): um DOCX
// editável, por um escritor OOXML próprio, e um PDF, pelo Typst empacotado.
// Os dois saem do mesmo minuta.Documento. Nenhum arquivo leva assinatura,
// autor ou indicação de ato oficial.
package documento

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // logotipo do cabeçalho
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/minuta"
)

// Formatos de saída.
const (
	FormatoDOCX = "docx"
	FormatoPDF  = "pdf"
)

// Cabecalho institucional (V4 §32.1): logotipo centralizado e as linhas em
// maiúsculas e negrito.
type Cabecalho struct {
	Linhas []string
	// Logo: imagem PNG ou JPEG (nil = cabeçalho só com o texto).
	Logo         []byte
	Ext          string // "png" ou "jpeg"
	LogoW, LogoH int    // pixels
}

// LinhasPadrao são as linhas aprovadas do cabeçalho (V4 §32.1).
var LinhasPadrao = []string{"SECRETARIA DE SEGURANÇA, DEFESA E CIDADANIA", "POLÍCIA MILITAR DO ESTADO DE RONDÔNIA"}

// nomesLogo: arquivos procurados em <componentes>\cabecalho, nesta ordem.
var nomesLogo = []string{"logo-pmro.png", "logo-cabecalho-docs.jpeg", "logo-cabecalho-docs.jpg", "logo-cabecalho-docs.png", "logo.png", "logo.jpeg", "logo.jpg"}

// CarregarCabecalho monta o cabeçalho padrão com o logotipo que estiver em
// <componentes>\cabecalho (ou em SYSEXEC_LOGO, no desenvolvimento). Sem
// imagem válida, o cabeçalho sai só com o texto.
func CarregarCabecalho(componentes string) Cabecalho {
	c := Cabecalho{Linhas: LinhasPadrao}
	candidatos := []string{}
	if p := os.Getenv("SYSEXEC_LOGO"); p != "" {
		candidatos = append(candidatos, p)
	}
	for _, n := range nomesLogo {
		candidatos = append(candidatos, filepath.Join(componentes, "cabecalho", n))
	}
	for _, p := range candidatos {
		b, err := os.ReadFile(p)
		if err != nil || len(b) > 4<<20 {
			continue
		}
		cfg, formato, err := image.DecodeConfig(bytes.NewReader(b))
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || (formato != "png" && formato != "jpeg") {
			continue
		}
		c.Logo, c.Ext, c.LogoW, c.LogoH = b, formato, cfg.Width, cfg.Height
		break
	}
	return c
}

// alturaLogoCm é a altura do logotipo no cabeçalho.
const alturaLogoCm = 2.2

// NomeArquivo sugere o nome do arquivo (V4 §32.2): tipo de minuta, versão e
// data, sem nome de pessoa nem número de processo (o aplicativo não o tem).
func NomeArquivo(procedimento string, versao int, quando time.Time, formato string) string {
	tipo := "Minuta_Julgamento_PADS"
	if procedimento == "SR" {
		tipo = "Minuta_Relatorio_Sindicancia_Regular"
	}
	if versao < 1 {
		versao = 1
	}
	return fmt.Sprintf("%s_v%d_%s.%s", tipo, versao, quando.Format("2006-01-02_1504"), formato)
}

// pedaco é um trecho de texto com a mesma formatação.
type pedaco struct {
	texto    string
	negrito  bool
	pendente bool // campo a preencher: realçado
}

// pedacos divide o texto de um bloco em trechos formatados.
func pedacos(b minuta.Bloco) []pedaco {
	negrito := b.Tipo == minuta.Titulo || b.Tipo == minuta.Secao || b.Tipo == minuta.Subtitulo ||
		(b.Tipo == minuta.Fecho && b.Texto == strings.ToUpper(b.Texto))
	texto := b.Texto
	var out []pedaco
	if b.Tipo == minuta.Cabecalho {
		// "Rótulo: valor" — o rótulo em negrito.
		if i := strings.Index(texto, ": "); i > 0 {
			out = append(out, pedaco{texto: texto[:i+2], negrito: true})
			texto = texto[i+2:]
		}
	}
	for texto != "" {
		i := strings.Index(texto, minuta.APreencher)
		if i < 0 {
			out = append(out, pedaco{texto: texto, negrito: negrito})
			break
		}
		if i > 0 {
			out = append(out, pedaco{texto: texto[:i], negrito: negrito})
		}
		out = append(out, pedaco{texto: minuta.APreencher, negrito: negrito, pendente: true})
		texto = texto[i+len(minuta.APreencher):]
	}
	return out
}

// estilo de parágrafo de cada tipo de bloco (o mesmo nome no DOCX e no Typst).
func estilo(b minuta.Bloco) string {
	switch b.Tipo {
	case minuta.Titulo:
		return "Titulo"
	case minuta.Cabecalho:
		return "Identificacao"
	case minuta.Secao:
		return "Secao"
	case minuta.Subtitulo:
		return "Subtitulo"
	case minuta.Citacao:
		return "Citacao"
	case minuta.Assinatura:
		return "Assinatura"
	case minuta.Fecho:
		if b.Texto != strings.ToUpper(b.Texto) {
			return "Data"
		}
	}
	return "Corpo" // parágrafo, item e "CUMPRA-SE."
}

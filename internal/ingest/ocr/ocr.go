// Package ocr reconhece texto de páginas rasterizadas (plano §7; ADR-005).
//
// O Tesseract roda como processo separado: a imagem PNG entra por stdin e o
// resultado TSV sai por stdout, sem arquivos temporários (validado na F0-05).
// Datas, números e artigos vindos de OCR são "não confirmados" (D-09.3): a
// confiança do OCR não protege dígitos (a F0 leu "2017" como "2047" com 92%).
package ocr

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// Palavra reconhecida, com caixa em pixels da imagem enviada.
type Palavra struct {
	Texto          string
	Confianca      float64 // 0–100
	X0, Y0, X1, Y1 int
	Linha          int // índice sequencial da linha na página
}

// Resultado do OCR de uma página.
type Resultado struct {
	Palavras  []Palavra
	Confianca float64 // média ponderada pelo tamanho das palavras (0 se não houver)
}

// Texto monta o texto da página, uma linha por linha reconhecida.
func (r Resultado) Texto() string {
	var b strings.Builder
	linha := -1
	for _, p := range r.Palavras {
		if p.Linha != linha {
			if linha >= 0 {
				b.WriteByte('\n')
			}
			linha = p.Linha
		} else {
			b.WriteByte(' ')
		}
		b.WriteString(p.Texto)
	}
	return b.String()
}

// Motor reconhece uma imagem (porta OCREngine do plano §4).
type Motor interface {
	Reconhecer(ctx context.Context, img image.Image) (Resultado, error)
}

// Tesseract executa o binário empacotado.
type Tesseract struct {
	Executavel string // caminho do tesseract.exe
	Tessdata   string // pasta com por.traineddata e osd.traineddata
	Idioma     string // padrão "por"
}

// Localizar procura o Tesseract em <componentes>\tesseract (instalação) ou na
// variável SYSEXEC_TESSERACT (desenvolvimento, aponta para o executável).
func Localizar(componentes string) (*Tesseract, error) {
	exe := os.Getenv("SYSEXEC_TESSERACT")
	if exe == "" {
		exe = filepath.Join(componentes, "tesseract", nomeExecutavel)
	}
	tessdata := os.Getenv("SYSEXEC_TESSDATA")
	if tessdata == "" {
		tessdata = filepath.Join(filepath.Dir(exe), "tessdata")
	}
	if _, err := os.Stat(exe); err != nil {
		return nil, erros.Com(erros.OCRFalhou, err)
	}
	if _, err := os.Stat(filepath.Join(tessdata, "por.traineddata")); err != nil {
		return nil, erros.Com(erros.OCRFalhou, err)
	}
	return &Tesseract{Executavel: exe, Tessdata: tessdata, Idioma: "por"}, nil
}

// Reconhecer envia a imagem por stdin e lê o TSV de stdout.
func (t *Tesseract) Reconhecer(ctx context.Context, img image.Image) (Resultado, error) {
	var entrada bytes.Buffer
	if err := png.Encode(&entrada, img); err != nil {
		return Resultado{}, erros.Com(erros.OCRFalhou, err)
	}
	idioma := t.Idioma
	if idioma == "" {
		idioma = "por"
	}
	cmd := exec.CommandContext(ctx, t.Executavel, "stdin", "stdout",
		"-l", idioma, "--tessdata-dir", t.Tessdata, "--psm", "3",
		"-c", "tessedit_create_tsv=1", // o alias "tsv" exige tessdata/configs, que não empacotamos
	)
	cmd.Stdin = &entrada
	var saida, stderr bytes.Buffer
	cmd.Stdout = &saida
	cmd.Stderr = &stderr // descartado (sem logs); só serve ao diagnóstico em memória
	cmd.Env = append(os.Environ(), "OMP_THREAD_LIMIT=1")
	ocultarJanela(cmd)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return Resultado{}, ctx.Err()
		}
		return Resultado{}, erros.Com(erros.OCRFalhou, fmt.Errorf("%w: %s", err, limitar(stderr.String(), 200)))
	}
	return LerTSV(saida.Bytes())
}

func limitar(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// LerTSV interpreta a saída TSV do Tesseract (nível 5 = palavra).
func LerTSV(b []byte) (Resultado, error) {
	type chave struct{ pag, bloco, par, linha int }
	var r Resultado
	linhas := map[chave]int{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	primeira := true
	var somaPeso, somaConf float64
	for sc.Scan() {
		campos := strings.Split(sc.Text(), "\t")
		if primeira {
			primeira = false
			if len(campos) > 0 && campos[0] == "level" {
				continue
			}
		}
		if len(campos) < 12 || campos[0] != "5" {
			continue
		}
		n := make([]int, 10)
		ok := true
		for i := 0; i < 10; i++ {
			v, err := strconv.Atoi(campos[i])
			if err != nil {
				ok = false
				break
			}
			n[i] = v
		}
		conf, err := strconv.ParseFloat(campos[10], 64)
		texto := strings.TrimSpace(campos[11])
		if !ok || err != nil || conf < 0 || texto == "" {
			continue
		}
		k := chave{n[1], n[2], n[3], n[4]}
		id, existe := linhas[k]
		if !existe {
			id = len(linhas)
			linhas[k] = id
		}
		r.Palavras = append(r.Palavras, Palavra{
			Texto: texto, Confianca: conf, Linha: id,
			X0: n[6], Y0: n[7], X1: n[6] + n[8], Y1: n[7] + n[9],
		})
		peso := float64(len([]rune(texto)))
		somaPeso += peso
		somaConf += conf * peso
	}
	if err := sc.Err(); err != nil {
		return Resultado{}, erros.Com(erros.OCRFalhou, err)
	}
	if len(r.Palavras) == 0 && !primeira && len(bytes.TrimSpace(b)) > 0 && !bytes.HasPrefix(b, []byte("level")) {
		return Resultado{}, erros.Com(erros.OCRFalhou, errors.New("saída TSV inesperada"))
	}
	sort.SliceStable(r.Palavras, func(i, j int) bool { return r.Palavras[i].Linha < r.Palavras[j].Linha })
	if somaPeso > 0 {
		r.Confianca = somaConf / somaPeso
	}
	return r, nil
}

package analise

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/segment"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
)

// Registro emite os identificadores de evidência da tarefa (EV-nnnn).
type Registro struct{ n int }

// Novo devolve o próximo identificador.
func (r *Registro) Novo() string {
	r.n++
	return fmt.Sprintf("EV-%04d", r.n)
}

// Bloco é um trecho contíguo de uma peça (uma ou mais páginas, ou parte de
// uma página longa), enviado como um documento delimitado.
type Bloco struct {
	ID      string
	Peca    segment.Peca
	Texto   string
	paginas []int // página de cada segmento
	inicios []int // início (runas) de cada segmento em Texto
	desloc  []int // deslocamento do segmento dentro da página original
}

// Posicao converte uma runa do bloco em (página, runa na página).
func (b Bloco) Posicao(r int) (pagina, offset int) {
	k := 0
	for i := range b.inicios {
		if b.inicios[i] <= r {
			k = i
		}
	}
	return b.paginas[k], r - b.inicios[k] + b.desloc[k]
}

// PaginaInicial do bloco.
func (b Bloco) PaginaInicial() int { return b.paginas[0] }

// Evidencia converte o bloco no documento do prompt.
func (b Bloco) Evidencia() policy.Evidencia {
	return policy.Evidencia{ID: b.ID, Peca: b.Peca.ID, Titulo: b.Peca.Titulo, Pagina: b.PaginaInicial(), Texto: b.Texto}
}

// separadorPaginas entre páginas de um bloco (não casa com trechos: a
// normalização o reduz a um espaço).
const separadorPaginas = "\n\n"

// Blocos divide a peça em blocos de até maxRunas, sem cortar páginas quando
// possível; página maior que o limite é dividida em fronteira de espaço.
func Blocos(a *autos.Autos, p segment.Peca, maxRunas int, reg *Registro) []Bloco {
	if maxRunas < 500 {
		maxRunas = 500
	}
	var out []Bloco
	var atual *Bloco
	tam := 0
	fechar := func() {
		if atual != nil && strings.TrimSpace(atual.Texto) != "" {
			atual.ID = reg.Novo()
			out = append(out, *atual)
		}
		atual, tam = nil, 0
	}
	adicionar := func(pag, desloc int, s string) {
		n := len([]rune(s))
		if atual != nil && tam+n+len(separadorPaginas) > maxRunas {
			fechar()
		}
		if atual == nil {
			atual = &Bloco{Peca: p}
		} else {
			atual.Texto += separadorPaginas
			tam += len(separadorPaginas)
		}
		atual.paginas = append(atual.paginas, pag)
		atual.inicios = append(atual.inicios, tam)
		atual.desloc = append(atual.desloc, desloc)
		atual.Texto += s
		tam += n
	}
	for n := p.PaginaInicial; n <= p.PaginaFinal && n <= len(a.Paginas); n++ {
		pg := a.Paginas[n-1]
		r := []rune(pg.Texto)
		if strings.TrimSpace(pg.Texto) == "" {
			continue
		}
		for ini := 0; ini < len(r); {
			fim := min(len(r), ini+maxRunas)
			if fim < len(r) {
				for k := fim; k > ini+maxRunas/2; k-- {
					if unicode.IsSpace(r[k]) {
						fim = k
						break
					}
				}
			}
			adicionar(n, ini, string(r[ini:fim]))
			ini = fim
		}
	}
	fechar()
	return out
}

// Ponto é um trecho literal dos autos conferido pelo Go (EvidenceSpan).
type Ponto struct {
	ID      string `json:"id"`
	Peca    string `json:"peca"`
	Titulo  string `json:"titulo"`
	SEI     string `json:"sei"`
	Pagina  int    `json:"pagina"`
	Inicio  int    `json:"inicio"` // runas na página
	Fim     int    `json:"fim"`
	Trecho  string `json:"trecho"`
	Alvo    string `json:"alvo"` // IMP-n (PADS) ou F-n (SR)
	Pessoa  string `json:"pessoa"`
	Quem    string `json:"quem"`
	Sentido string `json:"sentido"`
	Resumo  string `json:"resumo"`
	OCR     bool   `json:"ocr"` // página lida por OCR: conteúdo "não confirmado" (D-09.3)
}

// localizarNoBloco confere o trecho e devolve o ponto com página e posição.
// O trecho guardado é o do documento (não o da IA).
func localizarNoBloco(a *autos.Autos, b Bloco, trecho string) (Ponto, bool) {
	ini, fim, citado, ok := LocalizarComOmissao(b.Texto, trecho)
	if !ok {
		return Ponto{}, false
	}
	pag, off := b.Posicao(ini)
	pagFim, offFim := b.Posicao(fim - 1)
	if pagFim != pag {
		offFim = len([]rune(a.Paginas[pag-1].Texto)) - 1
	}
	p := Ponto{
		Peca: b.Peca.ID, Titulo: b.Peca.Titulo, SEI: b.Peca.SEI, Pagina: pag, Inicio: off, Fim: offFim + 1,
		Trecho: strings.ReplaceAll(citado, separadorPaginas, " "), OCR: a.Paginas[pag-1].Origem == autos.OrigemOCR,
	}
	return p, true
}

// EvidenciaDoPonto: o ponto como documento curto para a análise.
func (p Ponto) Evidencia() policy.Evidencia {
	t := p.Trecho
	if p.Quem != "" {
		t = "Quem afirma: " + p.Quem + "\n" + t
	}
	if p.OCR {
		t += "\n(trecho lido por OCR: números e datas não confirmados)"
	}
	return policy.Evidencia{ID: p.ID, Peca: p.Peca, Titulo: p.Titulo, Pagina: p.Pagina, Texto: t}
}

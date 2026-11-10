package segment

import (
	"fmt"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
)

// BaseUsuario: função declarada pelo usuário no cadastro de peças (V4
// §48.4). Prevalece sobre a leitura do conteúdo, que só serve de conferência:
// a divergência é perguntada, nunca trocada em silêncio (V4 §30.2).
const BaseUsuario Base = "USUARIO"

// Declarada é uma peça do cadastro, na ordem em que entra nos autos.
type Declarada struct {
	Cadastro string // ID da peça no cadastro (PEC-nn)
	Funcao   Funcao
	Titulo   string
	SEI      string // informado pelo usuário (pode ficar vazio)
	Paginas  int    // soma das páginas dos arquivos da peça
}

// DeDeclaracao monta as peças dos autos juntados a partir do cadastro: as
// páginas são contíguas, na ordem declarada. O número SEI, quando o usuário
// não informou, vem do carimbo dominante das páginas da peça.
func DeDeclaracao(a *autos.Autos, ds []Declarada) ([]Peca, error) {
	total := 0
	for _, d := range ds {
		if d.Paginas <= 0 {
			return nil, fmt.Errorf("peça %s sem páginas", d.Cadastro)
		}
		total += d.Paginas
	}
	if total != len(a.Paginas) {
		return nil, fmt.Errorf("peças declaradas somam %d página(s); os autos têm %d", total, len(a.Paginas))
	}
	proc := processoDominante(a)
	pecas := make([]Peca, 0, len(ds))
	ini := 1
	for i, d := range ds {
		p := Peca{
			ID: fmt.Sprintf("PEC-%02d", i+1), PaginaInicial: ini, PaginaFinal: ini + d.Paginas - 1,
			SEI: d.SEI, Titulo: limitar(d.Titulo, 120), Funcao: d.Funcao, Base: BaseUsuario, Declarada: d.Cadastro,
		}
		if p.SEI == "" && proc != "" {
			p.SEI = seiDominante(a, p, proc)
		}
		p.Externo = p.SEI == ""
		pecas = append(pecas, p)
		ini += d.Paginas
	}
	return pecas, nil
}

func seiDominante(a *autos.Autos, p Peca, proc string) string {
	cont := map[string]int{}
	melhor, n := "", 0
	for k := p.PaginaInicial; k <= p.PaginaFinal; k++ {
		if c, ok := carimboDaPagina(a.Paginas[k-1], proc); ok && c.SEI != "" {
			cont[c.SEI]++
			if v := cont[c.SEI]; v > n || (v == n && c.SEI < melhor) {
				melhor, n = c.SEI, v
			}
		}
	}
	return melhor
}

// DeclaracaoDivergente: a primeira página da peça parece outra peça
// processual (título reconhecido ou nome no carimbo do SEI).
type DeclaracaoDivergente struct {
	Peca      string `json:"peca"`
	Declarada Funcao `json:"declarada"`
	Aparente  Funcao `json:"aparente"`
	Pagina    int    `json:"pagina"`
}

// fortes: funções cujo título é inequívoco; só elas acusam divergência
// (um ofício ou uma procuração dentro da defesa não contradiz a declaração).
var fortes = map[Funcao]bool{
	MemorandoDisciplinar: true, PortariaSR: true, DefesaPrevia: true, AlegacoesFinais: true,
	TermoInquiricao: true, TermoDeclaracoes: true, Interrogatorio: true,
}

// aparente lê a função da primeira página da peça: o nome no carimbo do
// SEI e, sem ele, o título no alto da página.
func aparente(a *autos.Autos, p Peca) (Funcao, bool) {
	pg := a.Paginas[p.PaginaInicial-1]
	if proc := processoDominante(a); proc != "" {
		if c, ok := carimboDaPagina(pg, proc); ok {
			if f, ok := classificar(c.Nome); ok {
				return f, true
			}
		}
	}
	if pp := pistas(pg); pp.funcao != "" {
		return pp.funcao, true
	}
	return "", false
}

// VerificarDeclaradas confere as peças declaradas (V4 §§30.2, 48.4):
//   - a peça central declarada (Memorando no PADS; Portaria na SR) com
//     página ilegível ou de OCR fraco → pedir versão legível;
//   - a peça central cujo conteúdo é a peça central do outro procedimento →
//     incompatível (o tipo nunca é trocado);
//   - qualquer peça cuja primeira página mostra outra peça processual →
//     divergência, perguntada ao usuário.
func VerificarDeclaradas(tipo string, pecas []Peca, a *autos.Autos) Compatibilidade {
	central, outra, outroTipo := MemorandoDisciplinar, PortariaSR, "SR"
	if tipo == "SR" {
		central, outra, outroTipo = PortariaSR, MemorandoDisciplinar, "PADS"
	}
	var comp Compatibilidade
	for _, p := range pecas {
		if p.Funcao == central && comp.PecaCentral == "" {
			comp.PecaCentral = p.ID
			for n := p.PaginaInicial; n <= p.PaginaFinal; n++ {
				if s := a.Paginas[n-1].Situacao; s == autos.Ilegivel || s == autos.OCRBaixo {
					return Compatibilidade{Resultado: PecaIlegivel, PecaCentral: p.ID}
				}
			}
		}
		f, ok := aparente(a, p)
		if !ok || f == p.Funcao || !fortes[f] {
			continue
		}
		if p.Funcao == central && f == outra {
			return Compatibilidade{Resultado: Incompativel, PecaCentral: p.ID, TipoAparente: outroTipo}
		}
		if p.Funcao == PortariaSR && f == Portaria {
			continue
		}
		comp.Divergencias = append(comp.Divergencias, DeclaracaoDivergente{Peca: p.ID, Declarada: p.Funcao, Aparente: f, Pagina: p.PaginaInicial})
	}
	switch {
	case comp.PecaCentral == "":
		comp.Resultado = PecaAusente
	case len(comp.Divergencias) > 0:
		comp.Resultado = Divergente
	default:
		comp.Resultado = Compativel
	}
	return comp
}

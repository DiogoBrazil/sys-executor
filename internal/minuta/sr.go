package minuta

import (
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/sr"
)

// Relatório de Sindicância Regular (V4 §§23 a 25; §49). O aplicativo compõe
// a estrutura, a seção I (dados seguros do cadastro e da leitura) e a seção
// IV (conclusões e enquadramentos confirmados pelo usuário). A IA redige só
// a síntese (II) e a análise das provas de cada fato (III).

// PessoaSR é um envolvido citado na seção I, com o número SEI do termo em
// que foi ouvido, se houver.
type PessoaSR struct {
	Nome string
	SEI  string
}

// EnquadramentoSR é um possível crime ou transgressão indiciado (V4 §25.3),
// confirmado pelo usuário.
type EnquadramentoSR struct {
	Tipo sr.TipoEnquadramento
	// Referencia por extenso: "art. 17, inciso XXXVIII, do RDPM".
	Referencia string
	// Texto: redação do dispositivo na biblioteca ("" quando digitado).
	Texto string
	// Informado: digitado pelo usuário, fora da biblioteca (vai para conferência).
	Informado bool
	// Rubrica do tipo penal na norma ("Peculato"), quando houver.
	Rubrica string
}

// ConclusaoSR é o resultado confirmado de um sindicado num fato.
type ConclusaoSR struct {
	SindicadoID    string
	Sindicado      string // posto ou graduação e nome
	Indicios       bool
	Enquadramentos []EnquadramentoSR
}

// FatoSR é um fato investigado, com a análise redigida e as conclusões.
type FatoSR struct {
	ID         string
	Descricao  string
	Analise    analise.Trecho
	Conclusoes []ConclusaoSR
}

// EntradaSR reúne o que o relatório usa.
type EntradaSR struct {
	// Portaria: identificação confirmada (número, autoridade e data).
	Portaria   string
	Sindicados []PessoaSR
	// Ofendidos: pessoas e instituições do cadastro. SemOfendido: o usuário
	// declarou que não há ofendido individualizado.
	Ofendidos   []string
	SemOfendido bool
	Testemunhas []PessoaSR
	// Pericias: laudos encontrados na leitura das peças ("Laudo nº …").
	Pericias []string
	// Outras: termos de declarações e outros itens relevantes (seletiva).
	Outras []PessoaSR
	// SemCadastro: os autos vieram sem o cadastro das peças; quem foi ouvido
	// não é conhecido com segurança (as alíneas c, d e f ficam a conferir).
	SemCadastro bool
	Sintese     analise.Trecho
	Fatos       []FatoSR
}

// Rótulos fixos das seções e alíneas (V4 §23.1).
const (
	SecaoDadosSR    = "I – DADOS DA SINDICÂNCIA"
	SecaoSinteseSR  = "II – DA SÍNTESE DOS FATOS EM APURAÇÃO"
	SecaoAnaliseSR  = "III – ANÁLISE DAS PROVAS"
	SecaoSolucaoSR  = "IV – SOLUÇÃO"
	TituloSR        = "RELATÓRIO DE SINDICÂNCIA REGULAR"
	semRegistros    = "Não há registros."
	slotSinteseSR   = "sintese"
	prefixoAnaliseS = "analise:"
)

// AlineasSR são os rótulos da seção I, na ordem.
var AlineasSR = []string{"a. Portaria:", "b. Sindicados:", "c. Ofendido:", "d. Testemunhas ouvidas:", "e. Perícias realizadas:", "f. Outras informações:"}

// SlotAnaliseSR é o trecho da análise do fato.
func SlotAnaliseSR(fato string) string { return prefixoAnaliseS + fato }

// SlotSinteseSR é o trecho da síntese.
const SlotSinteseSR = slotSinteseSR

func comSEI(p PessoaSR) string {
	n := strings.TrimSpace(p.Nome)
	if s := strings.TrimSpace(p.SEI); s != "" {
		n += " (SEI nº " + s + ")"
	}
	return n
}

func listaSR(xs []string) string {
	var ok []string
	for _, x := range xs {
		if t := strings.TrimSpace(x); t != "" {
			ok = append(ok, semPontoFinal(t))
		}
	}
	if len(ok) == 0 {
		return ""
	}
	return strings.Join(ok, "; ") + "."
}

func pessoasSR(ps []PessoaSR) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, comSEI(p))
	}
	return out
}

// alinea escreve "x. Rótulo: conteúdo". Conteúdo vazio deixa só o rótulo
// (V4 §23.2), e a pendência, se houver, vai ao usuário.
func (d *Documento) alinea(rotulo, conteudo, pendencia string) {
	t := rotulo
	if conteudo != "" {
		t += " " + conteudo
	}
	d.add(Item, t)
	if conteudo == "" && pendencia != "" {
		d.pendencia(pendencia)
	}
}

// ComporSR monta a minuta do Relatório de Sindicância Regular.
func ComporSR(e EntradaSR) *Documento {
	d := &Documento{Procedimento: "SR"}
	d.add(Titulo, TituloSR)

	// I – DADOS DA SINDICÂNCIA (V4 §23.3)
	d.add(Secao, SecaoDadosSR)
	portaria := semPontoFinal(e.Portaria)
	if portaria != "" {
		portaria += "."
	}
	d.alinea(AlineasSR[0], portaria, "Informe a identificação da Portaria de instauração (alínea a).")
	d.alinea(AlineasSR[1], listaSR(pessoasSR(e.Sindicados)), "Informe os sindicados (alínea b).")
	switch {
	case e.SemCadastro && len(e.Ofendidos) == 0:
		d.alinea(AlineasSR[2], "", "Informe o ofendido (alínea c): os autos vieram sem o cadastro das peças.")
	case len(e.Ofendidos) > 0:
		d.alinea(AlineasSR[2], listaSR(e.Ofendidos), "")
	case e.SemOfendido:
		d.alinea(AlineasSR[2], semRegistros, "")
	default:
		d.alinea(AlineasSR[2], "", "Confira o ofendido: não foi identificado com segurança (alínea c).")
	}
	if len(e.Testemunhas) > 0 {
		d.alinea(AlineasSR[3], listaSR(pessoasSR(e.Testemunhas)), "")
	} else if e.SemCadastro {
		d.alinea(AlineasSR[3], "", "Informe as testemunhas ouvidas (alínea d): os autos vieram sem o cadastro das peças.")
	} else {
		d.alinea(AlineasSR[3], semRegistros, "")
	}
	d.alinea(AlineasSR[4], listaSR(e.Pericias), "Nenhum laudo pericial foi identificado na leitura das peças: confira se há perícia a relacionar (alínea e).")
	pendOutras := ""
	if e.SemCadastro {
		pendOutras = "Confira as outras informações (alínea f), como os termos de declarações: os autos vieram sem o cadastro das peças."
	}
	d.alinea(AlineasSR[5], listaSR(pessoasSR(e.Outras)), pendOutras)

	// II – SÍNTESE (V4 §23.5)
	d.add(Secao, SecaoSinteseSR)
	d.trecho(slotSinteseSR, e.Sintese, "síntese dos fatos em apuração")

	// III – ANÁLISE DAS PROVAS (V4 §24): por fato; os sindicados dentro dele.
	d.add(Secao, SecaoAnaliseSR)
	for i, f := range e.Fatos {
		if len(e.Fatos) > 1 {
			d.add(Subtitulo, fmt.Sprintf("Fato %d – %s", i+1, semPontoFinal(f.Descricao)))
		}
		d.trecho(SlotAnaliseSR(f.ID), f.Analise, fmt.Sprintf("análise das provas do fato %d", i+1))
	}

	// IV – SOLUÇÃO (V4 §25; §49): composta das conclusões confirmadas.
	d.add(Secao, SecaoSolucaoSR)
	solucaoSR(d, e)

	d.add(Fecho, APreencher+" (local), data conforme a assinatura eletrônica.")
	d.add(Assinatura, APreencher+" (nome e posto ou graduação), Encarregado(a) da Sindicância Regular")
	return d
}

var nomeTipoSR = map[sr.TipoEnquadramento]string{
	sr.Transgressao: "transgressão disciplinar", sr.CrimeMilitar: "crime militar", sr.CrimeComum: "crime comum",
}

// ordemTiposSR: como os modelos do usuário, crime comum, crime militar e
// transgressão.
var ordemTiposSR = []sr.TipoEnquadramento{sr.CrimeComum, sr.CrimeMilitar, sr.Transgressao}

func juntarE(xs []string) string {
	switch len(xs) {
	case 0:
		return ""
	case 1:
		return xs[0]
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " e " + xs[len(xs)-1]
}

func juntarOu(xs []string) string {
	switch len(xs) {
	case 0:
		return ""
	case 1:
		return xs[0]
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " ou " + xs[len(xs)-1]
}

// frasesConclusao escreve o resultado de um sindicado num fato: os
// indícios, por tipo, com os dispositivos, e o que não foi indiciado; ou a
// ausência de indícios (V4 §25.5, S1 a S3; §49).
func frasesConclusao(d *Documento, c ConclusaoSR) []string {
	if !c.Indicios {
		return []string{"não foram identificados indícios suficientes de crime comum, crime militar ou transgressão disciplinar"}
	}
	var out []string
	presentes := map[sr.TipoEnquadramento]bool{}
	for _, t := range ordemTiposSR {
		var dispositivos []string
		for _, q := range c.Enquadramentos {
			if q.Tipo != t {
				continue
			}
			ref := strings.TrimSpace(q.Referencia)
			if ref == "" {
				ref = APreencher
			}
			x := "no " + ref
			if q.Rubrica != "" {
				x += " (" + strings.ToLower(q.Rubrica) + ")"
			}
			if q.Texto != "" {
				x += " – “" + semPontoFinal(q.Texto) + "”"
			}
			if q.Informado {
				d.pendencia("Confira o dispositivo informado para " + c.Sindicado + " (" + ref + "): ele não está na biblioteca e foi citado como você digitou.")
			}
			dispositivos = append(dispositivos, x)
		}
		if len(dispositivos) == 0 {
			continue
		}
		presentes[t] = true
		nome, previsto := nomeTipoSR[t], "previsto"
		if t == sr.Transgressao {
			previsto = "prevista"
		}
		if len(dispositivos) > 1 {
			nome, previsto = pluralTipoSR[t], previsto+"s"
		}
		out = append(out, fmt.Sprintf("há indícios da prática de %s %s %s", nome, previsto, juntarE(dispositivos)))
	}
	if len(out) == 0 {
		d.pendencia("Há conclusão de indícios sem enquadramento confirmado para " + c.Sindicado + ": informe o dispositivo.")
		return []string{"há indícios da prática de " + APreencher}
	}
	var ausentes []string
	for _, t := range ordemTiposSR {
		if !presentes[t] {
			ausentes = append(ausentes, nomeTipoSR[t])
		}
	}
	if len(ausentes) > 0 {
		out = append(out, "não foram identificados indícios de "+juntarOu(ausentes))
	}
	return out
}

var pluralTipoSR = map[sr.TipoEnquadramento]string{
	sr.Transgressao: "transgressões disciplinares", sr.CrimeMilitar: "crimes militares", sr.CrimeComum: "crimes comuns",
}

func maiuscula(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return strings.ToUpper(string(r[:1])) + string(r[1:])
}

// assinaturaConclusao identifica resultados iguais (para a conclusão conjunta).
func assinaturaConclusao(c ConclusaoSR) string {
	if !c.Indicios {
		return "AUSENCIA"
	}
	var b strings.Builder
	b.WriteString("INDICIOS")
	for _, q := range c.Enquadramentos {
		b.WriteString("|" + string(q.Tipo) + ":" + q.Referencia)
	}
	return b.String()
}

func solucaoSR(d *Documento, e EntradaSR) {
	if len(e.Fatos) == 0 {
		d.add(Paragrafo, APreencher+" — conclusões.")
		d.pendencia("Nenhum fato confirmado: a solução fica a preencher.")
		return
	}
	d.add(Paragrafo, "Diante da análise dos elementos constantes dos autos, conclui-se o seguinte:")
	// Um fato e o mesmo resultado para todos: conclusão conjunta (V4 §25.1;
	// SR10), com a ressalva de que as condutas foram examinadas uma a uma.
	if len(e.Fatos) == 1 {
		f := e.Fatos[0]
		cs := f.Conclusoes
		iguais := len(cs) > 1
		for _, c := range cs[min(1, len(cs)):] {
			iguais = iguais && assinaturaConclusao(c) == assinaturaConclusao(cs[0])
		}
		if iguais {
			nomes := make([]string, len(cs))
			for i, c := range cs {
				nomes[i] = "ao " + c.Sindicado
			}
			fs := frasesConclusao(d, cs[0])
			b := d.add(Paragrafo, fmt.Sprintf("Quanto ao fato investigado, em relação %s, %s. As condutas de cada sindicado foram examinadas individualmente na análise das provas.",
				juntarE(nomes), juntarFrases(fs)))
			b.Trecho = "solucao"
			return
		}
	}
	// Individual por sindicado; com vários fatos, por fato dentro dele (S2, S3).
	ordem, porSind := sindicadosDaSolucao(e)
	for i, id := range ordem {
		itens := porSind[id]
		d.add(Item, fmt.Sprintf("%c) %s.", 'a'+rune(i), itens[0].c.Sindicado))
		for _, it := range itens {
			fs := frasesConclusao(d, it.c)
			t := maiuscula(juntarFrases(fs))
			if len(e.Fatos) > 1 {
				t = fmt.Sprintf("Quanto ao fato %d, %s", it.fato, juntarFrases(fs))
			}
			b := d.add(Paragrafo, t+".")
			b.Trecho = "solucao"
		}
	}
}

// juntarFrases: "há indícios de A; não foram identificados indícios de B".
func juntarFrases(fs []string) string {
	if len(fs) == 1 {
		return fs[0]
	}
	return strings.Join(fs[:len(fs)-1], "; ") + "; " + fs[len(fs)-1]
}

type itemSolucao struct {
	fato int
	c    ConclusaoSR
}

func sindicadosDaSolucao(e EntradaSR) ([]string, map[string][]itemSolucao) {
	var ordem []string
	por := map[string][]itemSolucao{}
	for i, f := range e.Fatos {
		for _, c := range f.Conclusoes {
			if _, ok := por[c.SindicadoID]; !ok {
				ordem = append(ordem, c.SindicadoID)
			}
			por[c.SindicadoID] = append(por[c.SindicadoID], itemSolucao{fato: i + 1, c: c})
		}
	}
	return ordem, por
}

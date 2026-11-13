package pipeline

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/sr"
	"github.com/DiogoBrazil/sys-executor/internal/legal"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/rules/enquadramento"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// Enquadramento da SR (V4 §25.3; regra do usuário de 12/11/2026): nas
// conclusões com indícios suficientes, a IA propõe, o aplicativo confere na
// biblioteca e o usuário confirma, tira ou acrescenta.
//   - transgressão: inciso dos arts. 15, 16 ou 17 do RDPM ou dos arts. 29 ou
//     32 do Estatuto (catálogo de enquadramento);
//   - crime militar: CPM ou outra lei penal, esta só com o art. 9º, II, do CPM;
//   - crime comum: CP ou outra lei penal, nunca o CPM.

// EnquadSR é um enquadramento conferido na biblioteca.
type EnquadSR struct {
	Tipo     sr.TipoEnquadramento
	Passagem legal.Passagem
	// Art9Alinea: alínea do art. 9º, II, do CPM (crime militar fora do CPM).
	Art9Alinea string
	Fundamento string
	Evid       []string
	DaIA       bool
}

// propostaSR é o que a IA propôs para uma conclusão (fato:sindicado).
type propostaSR struct {
	Itens     []EnquadSR
	Pendencia string
}

const prefixoPerguntaEnq = "enquadramento:"

// leis penais da biblioteca que a SR pode indicar (V4 §25.3): códigos e leis
// com tipos penais. Os processuais e a Lei dos Juizados ficam de fora.
var diplomasPenaisSR = []struct{ doc, sigla, nome, citacao string }{
	{"cp", "CP", "Código Penal (Decreto-Lei nº 2.848/1940)", "do Código Penal"},
	{"cpm", "CPM", "Código Penal Militar (Decreto-Lei nº 1.001/1969)", "do Código Penal Militar"},
	{"tortura", "Lei 9.455", "Lei nº 9.455/1997 (crimes de tortura)", "da Lei nº 9.455/1997"},
	{"abuso-de-autoridade", "Lei 13.869", "Lei nº 13.869/2019 (crimes de abuso de autoridade)", "da Lei nº 13.869/2019"},
	{"estatuto-do-desarmamento", "Lei 10.826", "Lei nº 10.826/2003 (armas de fogo e munição)", "da Lei nº 10.826/2003"},
	{"maria-da-penha", "Lei 11.340", "Lei nº 11.340/2006 (violência doméstica e familiar contra a mulher)", "da Lei nº 11.340/2006"},
	{"ctb", "CTB", "Código de Trânsito Brasileiro (Lei nº 9.503/1997)", "do Código de Trânsito Brasileiro"},
	{"eca", "ECA", "Estatuto da Criança e do Adolescente (Lei nº 8.069/1990)", "do Estatuto da Criança e do Adolescente"},
	{"eaoab", "Lei 8.906", "Estatuto da Advocacia (Lei nº 8.906/1994)", "da Lei nº 8.906/1994"},
}

var citacaoDoc = map[string]string{"rdpm": "do RDPM", "estatuto": "do " + enquadramento.NomeEstatuto}

func init() {
	for _, d := range diplomasPenaisSR {
		citacaoDoc[d.doc] = d.citacao
	}
}

func docPenal(doc string) bool {
	for _, d := range diplomasPenaisSR {
		if d.doc == doc {
			return true
		}
	}
	return false
}

func docDaSigla(sigla string) string {
	for _, d := range diplomasPenaisSR {
		if d.sigla == sigla {
			return d.doc
		}
	}
	return ""
}

var rotuloTipoSR = map[sr.TipoEnquadramento]string{sr.Transgressao: "Transgressão disciplinar", sr.CrimeMilitar: "Crime militar", sr.CrimeComum: "Crime comum"}

// ReferenciaMinuta: como o enquadramento é citado na minuta:
// "art. 303 do Código Penal Militar", "art. 1º, inciso I, da Lei nº
// 9.455/1997, c/c o art. 9º, inciso II, alínea c, do Código Penal Militar".
func (q EnquadSR) ReferenciaMinuta() string {
	p := q.Passagem
	r := legal.Referencia{Artigo: p.Artigo, Paragrafo: p.Paragrafo, Inciso: p.Inciso, Alinea: p.Alinea}
	s := r.Texto() + ", " + citacaoDoc[p.DocumentoID]
	if p.Paragrafo == "" && p.Inciso == "" && p.Alinea == "" {
		s = r.Texto() + " " + citacaoDoc[p.DocumentoID]
	}
	if q.Art9Alinea != "" {
		s += ", c/c o art. 9º, inciso II, alínea " + q.Art9Alinea + ", do Código Penal Militar"
	}
	return s
}

// Rotulo para a tela: "Crime militar — art. 303 do Código Penal Militar (Peculato)".
func (q EnquadSR) Rotulo() string {
	s := rotuloTipoSR[q.Tipo] + " — " + q.ReferenciaMinuta()
	if q.Passagem.Rubrica != "" {
		s += " (" + q.Passagem.Rubrica + ")"
	}
	return s
}

// chave para comparar enquadramentos (dedup).
func (q EnquadSR) chave() string {
	p := q.Passagem
	return string(q.Tipo) + "|" + p.DocumentoID + "|" + p.Artigo + "|" + p.Paragrafo + "|" + p.Inciso + "|" + p.Alinea + "|" + q.Art9Alinea
}

func (p *Pipeline) biblioteca() (*legal.Biblioteca, error) {
	abrir := p.Biblioteca
	if abrir == nil {
		abrir = legal.Padrao
	}
	return abrir()
}

// transgressaoDoCatalogo confere o item do catálogo e traz o texto da biblioteca.
func transgressaoDoCatalogo(b *legal.Biblioteca, d enquadramento.Dispositivo) (EnquadSR, error) {
	doc := map[string]string{enquadramento.RDPM: "rdpm", enquadramento.Estatuto: "estatuto"}[d.Diploma]
	ps, ok := b.Dispositivo(legal.Referencia{Documento: doc, Artigo: strconv.Itoa(d.Artigo), Inciso: d.Inciso, Alinea: d.Alinea})
	if !ok {
		return EnquadSR{}, fmt.Errorf("o %s não foi localizado na biblioteca", d.Referencia())
	}
	return EnquadSR{Tipo: sr.Transgressao, Passagem: ps}, nil
}

// crimeDaBiblioteca confere o tipo penal: o dispositivo existe e está em
// vigor na biblioteca, no diploma permitido para o tipo.
func crimeDaBiblioteca(b *legal.Biblioteca, tipo sr.TipoEnquadramento, r legal.Referencia, art9 string) (EnquadSR, error) {
	if !docPenal(r.Documento) {
		return EnquadSR{}, fmt.Errorf("crime só pode ser indicado no CP, no CPM ou numa lei penal da biblioteca")
	}
	switch {
	case tipo == sr.CrimeComum && r.Documento == "cpm":
		return EnquadSR{}, fmt.Errorf("crime comum não pode ser enquadrado no CPM; no CPM é crime militar")
	case tipo == sr.CrimeComum && art9 != "":
		return EnquadSR{}, fmt.Errorf("crime comum não leva o art. 9º do CPM; com o art. 9º, II, é crime militar")
	case tipo == sr.CrimeMilitar && r.Documento != "cpm" && art9 == "":
		return EnquadSR{}, fmt.Errorf("crime militar fora do CPM exige a alínea do art. 9º, inciso II, do CPM")
	case tipo == sr.CrimeMilitar && r.Documento == "cpm":
		art9 = ""
	}
	if art9 != "" && !strings.Contains("abcde", art9) {
		return EnquadSR{}, fmt.Errorf("a alínea do art. 9º, II, do CPM vai de a até e")
	}
	ps, ok := b.Dispositivo(r)
	if !ok {
		return EnquadSR{}, fmt.Errorf("o %s %s não foi localizado na biblioteca (ou não está em vigor)", r.Texto(), citacaoDoc[r.Documento])
	}
	return EnquadSR{Tipo: tipo, Passagem: ps, Art9Alinea: art9}, nil
}

// resolverItem confere um item proposto pela IA.
func resolverItem(b *legal.Biblioteca, it analise.ItemEnquadramento) (EnquadSR, error) {
	var q EnquadSR
	var err error
	switch it.Tipo {
	case analise.EnqTransgressao:
		d, ok := enquadramento.PorID(it.TransgressaoID)
		if !ok {
			return EnquadSR{}, errors.New("transgressão exige transgressao_id de TRANSGRESSOES_POSSIVEIS")
		}
		q, err = transgressaoDoCatalogo(b, d)
	case analise.EnqCrimeMilitar, analise.EnqCrimeComum:
		doc := docDaSigla(it.Diploma)
		if doc == "" || strings.TrimSpace(it.Artigo) == "" {
			return EnquadSR{}, errors.New("crime exige diploma de DIPLOMAS_PENAIS e artigo")
		}
		r := legal.Referencia{Documento: doc, Artigo: normalizarArtigo(it.Artigo), Paragrafo: normalizarParagrafo(it.Paragrafo),
			Inciso: strings.ToUpper(strings.TrimSpace(it.Inciso)), Alinea: strings.ToLower(strings.TrimSpace(it.Alinea))}
		q, err = crimeDaBiblioteca(b, sr.TipoEnquadramento(it.Tipo), r, it.Art9Alinea)
	default:
		return EnquadSR{}, fmt.Errorf("tipo inválido %q", it.Tipo)
	}
	if err != nil {
		return EnquadSR{}, err
	}
	q.Fundamento, q.Evid, q.DaIA = it.Fundamento, it.Evid, true
	return q, nil
}

var reArtNum = regexp.MustCompile(`^(\d{1,3})\s*[º°o]?\s*(?:-\s*([A-Za-z]))?$`)

func normalizarArtigo(s string) string {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "art."))
	m := reArtNum.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return strings.TrimSpace(s)
	}
	if m[2] != "" {
		return m[1] + "-" + strings.ToUpper(m[2])
	}
	return m[1]
}

func normalizarParagrafo(s string) string {
	s = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "§")))
	switch {
	case s == "":
		return ""
	case strings.Contains(s, "nico"):
		return "unico"
	}
	return normalizarArtigo(s)
}

var (
	reNumeroLei = regexp.MustCompile(`(?i)\bn\s*[º°o.]\s*`)
	reCC        = regexp.MustCompile(`(?i)\s*(?:,\s*)?\b(?:c/c|combinado com|cc)\b\s*`)
	reArt9      = regexp.MustCompile(`(?i)\bart(?:igo)?\.?\s*9\s*[º°o]?\b`)
	reArt9Alin  = regexp.MustCompile(`(?i)\b(?:ii|inciso\s+ii)\s*,?\s*(?:al[íi]nea\s*)?["“']?([a-e])\b`)
)

// interpretarLinhaSR lê um enquadramento digitado pelo usuário: "art. 303
// do CPM", "art. 17, XXXVIII, do RDPM", "art. 1º, I, da Lei 9.455 c/c art.
// 9º, II, c, do CPM". O tipo vem do diploma: RDPM e Estatuto são
// transgressão; CPM, crime militar; outra lei penal, crime comum, ou crime
// militar quando combinada com o art. 9º, II, do CPM.
func interpretarLinhaSR(b *legal.Biblioteca, linha string) (EnquadSR, error) {
	t := strings.TrimSpace(reNumeroLei.ReplaceAllString(linha, ""))
	principal, complemento := t, ""
	if loc := reCC.FindStringIndex(t); loc != nil {
		principal, complemento = t[:loc[0]], t[loc[1]:]
	}
	r, ok := b.InterpretarReferencia(principal)
	if !ok || r.Documento == "" {
		return EnquadSR{}, fmt.Errorf("“%s”: não reconheci o artigo e o diploma (ex.: art. 303 do CPM)", linha)
	}
	switch {
	case r.Documento == "rdpm" || r.Documento == "estatuto":
		dip := map[string]string{"rdpm": enquadramento.RDPM, "estatuto": enquadramento.Estatuto}[r.Documento]
		n, _ := strconv.Atoi(r.Artigo)
		var d enquadramento.Dispositivo
		var achou bool
		if r.Inciso != "" {
			id := map[string]string{enquadramento.RDPM: "RDPM", enquadramento.Estatuto: "EST"}[dip] + "-" + strconv.Itoa(n) + "-" + r.Inciso
			if r.Alinea != "" {
				id += "-" + r.Alinea
			}
			d, achou = enquadramento.PorID(id)
		}
		if !achou {
			return EnquadSR{}, fmt.Errorf("“%s”: transgressão disciplinar é um inciso dos arts. 15, 16 ou 17 do RDPM ou dos arts. 29 ou 32 do Estatuto", linha)
		}
		q, err := transgressaoDoCatalogo(b, d)
		if err != nil {
			return EnquadSR{}, fmt.Errorf("“%s”: %v", linha, err)
		}
		return q, nil
	case !docPenal(r.Documento):
		return EnquadSR{}, fmt.Errorf("“%s”: este diploma não define crimes; use o CP, o CPM ou uma lei penal da biblioteca", linha)
	}
	tipo, art9 := sr.CrimeComum, ""
	if r.Documento == "cpm" {
		tipo = sr.CrimeMilitar
	}
	if complemento != "" {
		if !reArt9.MatchString(complemento) {
			return EnquadSR{}, fmt.Errorf("“%s”: o complemento depois de c/c só pode ser o art. 9º, inciso II, do CPM", linha)
		}
		m := reArt9Alin.FindStringSubmatch(complemento)
		if m == nil {
			return EnquadSR{}, fmt.Errorf("“%s”: informe a alínea do art. 9º, inciso II, do CPM (a a e)", linha)
		}
		tipo, art9 = sr.CrimeMilitar, strings.ToLower(m[1])
	}
	q, err := crimeDaBiblioteca(b, tipo, r, art9)
	if err != nil {
		return EnquadSR{}, fmt.Errorf("“%s”: %v", linha, err)
	}
	return q, nil
}

// conclusaoSR: a conclusão confirmada do par (INDICIOS ou AUSENCIA).
func conclusaoSR(c *workflow.Contexto, fato, sind string) string {
	if r, ok := c.RespostaCompleta("conclusao:" + fato + ":" + sind); ok {
		return r.Valor("conclusao")
	}
	return ""
}

// fontesSR é a etapa "Consultando fontes jurídicas" da SR: o enquadramento
// de cada conclusão com indícios, proposto, conferido e confirmado.
func (p *Pipeline) fontesSR(ctx context.Context, c *workflow.Contexto, d *Dados, b *legal.Biblioteca) (workflow.Estado, error) {
	e := d.IA
	f := &FontesJuridicas{Enquadramento: map[string]legal.Passagem{}, EnquadramentoSR: map[string][]EnquadSR{}}
	if e == nil || e.Objeto == nil {
		d.Fontes = f
		return workflow.Redigindo, nil
	}
	if e.PropostasSR == nil {
		e.PropostasSR = map[string]propostaSR{}
	}
	d.cadastro, d.conclusoesSR = c.Cadastro(), map[string]string{}
	for _, fato := range e.Objeto.FatosAdmitidos() {
		for _, s := range e.Objeto.Sindicados() {
			d.conclusoesSR[fato+":"+s] = conclusaoSR(c, fato, s)
		}
	}
	for _, fato := range e.Objeto.FatosAdmitidos() {
		for _, s := range e.Objeto.Sindicados() {
			if conclusaoSR(c, fato, s) != string(sr.Indicios) {
				continue
			}
			chave := fato + ":" + s
			if _, ok := e.PropostasSR[chave]; !ok {
				if aguardar, err := p.prepararIA(ctx, c, d); err != nil {
					return workflow.ConsultandoFontes, err
				} else if aguardar {
					return workflow.AguardandoEsclarecimento, nil
				}
				c.Atividade(fmt.Sprintf("Propondo o enquadramento do fato %s quanto a %s.", fato, e.nomes[s]))
				pr, err := p.proporEnquadramento(ctx, d, b, fato, s)
				if err != nil {
					e.motorFalhou(err)
					return workflow.ConsultandoFontes, err
				}
				e.PropostasSR[chave] = pr
			}
			id := prefixoPerguntaEnq + chave
			resp, ok := c.RespostaCompleta(id)
			if !ok {
				ant, _ := c.Anterior(id)
				c.Perguntar(perguntaEnquadramentoSR(d, b, fato, s, e.PropostasSR[chave], ant))
				publicarPainel(c, d)
				return workflow.AguardandoEsclarecimento, nil
			}
			qs, err := enquadramentosDaResposta(b, e.PropostasSR[chave], resp.Valores)
			if err != nil { // a biblioteca mudou entre a resposta e agora: pergunta de novo
				c.Esquecer(id)
				return workflow.ConsultandoFontes, nil
			}
			f.EnquadramentoSR[chave] = qs
		}
	}
	d.Fontes = f
	publicarPainel(c, d)
	return workflow.Redigindo, nil
}

func (p *Pipeline) proporEnquadramento(ctx context.Context, d *Dados, b *legal.Biblioteca, fato, s string) (propostaSR, error) {
	e := d.IA
	ds, _ := enquadramento.Todos()
	var cands []analise.TransgressaoCandidata
	for _, x := range ds {
		cands = append(cands, analise.TransgressaoCandidata{ID: x.ID, Referencia: x.Referencia(), Texto: x.Texto})
	}
	var dips []analise.DiplomaPenal
	for _, x := range diplomasPenaisSR {
		dips = append(dips, analise.DiplomaPenal{Sigla: x.sigla, Nome: x.nome})
	}
	var art9 []string
	if ps, ok := b.Dispositivo(legal.Referencia{Documento: "cpm", Artigo: "9", Inciso: "II"}); ok {
		art9 = append(art9, "II – "+ps.Texto)
		for _, a := range []string{"a", "b", "c", "d", "e"} {
			if pa, ok := b.Dispositivo(legal.Referencia{Documento: "cpm", Artigo: "9", Inciso: "II", Alinea: a}); ok {
				art9 = append(art9, a+") "+pa.Texto)
			}
		}
	}
	fo, _ := e.Objeto.Fato(fato)
	an := e.AnalisesSR[fato+":"+s]
	dados := analise.DadosEnquadramentoSR{FatoID: fato, Fato: fo.Descricao, SindicadoID: s, Sindicado: e.nomes[s], Conclusao: string(sr.Indicios),
		Ocorrencia: an.Ocorrencia, Autoria: an.Autoria, Fundamentos: an.Fundamentos}
	var achados []EnquadSR
	itens, pend, err := analise.ProporEnquadramentoSR(ctx, e.executor(policy.FluxoSR), dados, cands, dips, art9, pontosDe(e, fato),
		func(it analise.ItemEnquadramento) error {
			_, err := resolverItem(b, it)
			return err
		})
	if err != nil {
		return propostaSR{}, err
	}
	vistos := map[string]bool{}
	for _, it := range itens {
		q, err := resolverItem(b, it)
		if err != nil || vistos[q.chave()] {
			continue
		}
		vistos[q.chave()] = true
		achados = append(achados, q)
	}
	return propostaSR{Itens: achados, Pendencia: pend}, nil
}

const campoAcrescentar = "acrescentar"

func campoManter(i int) string { return fmt.Sprintf("manter_%d", i+1) }

func perguntaEnquadramentoSR(d *Dados, b *legal.Biblioteca, fato, s string, pr propostaSR, ant workflow.Resposta) workflow.Pergunta {
	e := d.IA
	fo, _ := e.Objeto.Fato(fato)
	id := prefixoPerguntaEnq + fato + ":" + s
	val := func(campo, padrao string) string {
		if v, ok := ant.Valores[campo]; ok {
			return v
		}
		return padrao
	}
	g := workflow.GrupoCampos{ID: fato + s, Titulo: fmt.Sprintf("Fato %s — %s", fato, e.nomes[s]), Descricao: []string{fo.Descricao}}
	if pr.Pendencia != "" {
		g.Alertas = append(g.Alertas, pr.Pendencia)
	}
	if len(pr.Itens) == 0 && pr.Pendencia == "" {
		g.Alertas = append(g.Alertas, "O assistente não encontrou enquadramento com suporte nas provas: informe-o abaixo ou altere a conclusão.")
	}
	grupos := []workflow.GrupoCampos{g}
	var campos []workflow.Campo
	for i, q := range pr.Itens {
		gi := workflow.GrupoCampos{ID: fmt.Sprintf("%s%s-%d", fato, s, i+1), Titulo: q.Rotulo(),
			Fontes: fontesDosFundamentos(e, []analise.Fundamento{{Texto: q.Fundamento, Evid: q.Evid}})}
		if q.Passagem.Caput != "" {
			gi.Descricao = append(gi.Descricao, "Caput: “"+q.Passagem.Caput+"”")
		}
		gi.Descricao = append(gi.Descricao, "Texto na biblioteca: “"+q.Passagem.Texto+"”", "Fundamento da proposta: "+q.Fundamento)
		if q.Tipo != sr.Transgressao {
			gi.Alertas = append(gi.Alertas, "Confira a redação vigente na data do fato: a biblioteca traz o texto atual.")
		}
		grupos = append(grupos, gi)
		campos = append(campos, workflow.Campo{ID: campoManter(i), Rotulo: "Manter este enquadramento", Tipo: workflow.CampoSimNao,
			Obrigatorio: true, Grupo: gi.ID, Valor: val(campoManter(i), "sim")})
	}
	campos = append(campos, workflow.Campo{ID: campoAcrescentar, Rotulo: "Acrescentar ou trocar (um dispositivo por linha)", Tipo: workflow.CampoTextoLongo,
		Grupo: g.ID, Valor: val(campoAcrescentar, ""),
		Ajuda: "Ex.: art. 303 do CPM · art. 312 do CP · art. 17, XXXVIII, do RDPM · art. 29, IV, do Estatuto · " +
			"art. 1º, I, da Lei 9.455 c/c art. 9º, II, c, do CPM. RDPM e Estatuto: transgressão; CPM: crime militar; " +
			"outra lei penal: crime comum, ou crime militar com o art. 9º, II, do CPM. Cada linha é conferida na biblioteca."})
	titulo := "Enquadramento do fato " + fato
	if nome := e.nomes[s]; nome != "" {
		titulo += ": " + nome
	}
	return workflow.Pergunta{ID: id, Titulo: titulo, Revisavel: true,
		Contexto: "Há indícios suficientes: confira o enquadramento proposto. Crime comum, crime militar e transgressão podem ser indicados juntos. " +
			"Desmarque o que não se aplica e acrescente o que faltar. O enquadramento é seu: o assistente apenas sugere.",
		Grupos: grupos, Campos: campos,
		Opcoes: []workflow.Opcao{{ID: "confirmar", Rotulo: "Confirmar", Validar: true}},
		Conferir: func(vs map[string]string) error {
			_, err := enquadramentosDaResposta(b, pr, vs)
			return err
		},
		Resumir: func(r workflow.Resposta) string {
			qs, err := enquadramentosDaResposta(b, pr, r.Valores)
			if err != nil {
				return "a conferir"
			}
			var xs []string
			for _, q := range qs {
				xs = append(xs, q.ReferenciaMinuta())
			}
			return strings.Join(xs, "; ")
		}}
}

// enquadramentosDaResposta: os itens mantidos e as linhas acrescentadas,
// conferidos na biblioteca. Indícios exigem ao menos um (SR-SOL-005).
func enquadramentosDaResposta(b *legal.Biblioteca, pr propostaSR, vs map[string]string) ([]EnquadSR, error) {
	var out []EnquadSR
	vistos := map[string]bool{}
	for i, q := range pr.Itens {
		if vs[campoManter(i)] != "sim" || vistos[q.chave()] {
			continue
		}
		vistos[q.chave()] = true
		out = append(out, q)
	}
	var problemas []string
	for _, l := range strings.Split(vs[campoAcrescentar], "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}
		q, err := interpretarLinhaSR(b, l)
		if err != nil {
			problemas = append(problemas, err.Error())
			continue
		}
		if !vistos[q.chave()] {
			vistos[q.chave()] = true
			out = append(out, q)
		}
	}
	if len(problemas) > 0 {
		return nil, errors.New(strings.Join(problemas, " "))
	}
	if len(out) == 0 {
		return nil, errors.New("com indícios suficientes, mantenha ou acrescente ao menos um enquadramento (ou altere a conclusão)")
	}
	return out, nil
}

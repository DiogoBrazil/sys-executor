package minuta

import (
	"fmt"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/pads"
	"github.com/DiogoBrazil/sys-executor/internal/rules/engine"
)

// ImputacaoPADS é uma imputação do memorando, com o resultado confirmado
// pelo usuário e a fundamentação redigida.
type ImputacaoPADS struct {
	ID          string
	Conduta     string
	Dispositivo string // "art. 17, inciso V, do RDPM"
	// TextoDispositivo: redação do dispositivo recuperada da biblioteca
	// (vazio quando não localizado: o texto não é transcrito).
	TextoDispositivo string
	Natureza         engine.Natureza
	Procedente       bool
	Fundamentacao    analise.Trecho
}

// SancaoPADS é a punição de uma imputação principal, pelo cálculo oficial.
type SancaoPADS struct {
	ImputacaoID string
	Dispositivo string
	Natureza    engine.Natureza
	Sancao      engine.Sancao
	Dias        int
	// Chaves contadas pelo motor ("art. 38, I"; "art. 39, VII, a").
	AtenuantesContadas, AgravantesContadas []string
	// Circunstâncias reconhecidas pelo usuário, com o fundamento.
	Atenuantes, Agravantes, Justificacoes []engine.Circunstancia
	Secundarias                           []string // dispositivos das conexas absorvidas
	// Fatores do art. 36 (sete), redigidos pela IA.
	Fatores          []analise.FatorArt36
	PendenciaFatores string
}

// EntradaPADS reúne tudo o que a minuta do julgamento usa.
type EntradaPADS struct {
	Acusado   string // posto ou graduação e nome
	Matricula string
	// Memorando: como citar a peça inicial ("Memorando Disciplinar (SEI nº …)").
	Memorando  string
	Imputacoes []ImputacaoPADS
	Global     pads.Global
	Relatorio  analise.Trecho
	Sancoes    []SancaoPADS
	// Pontuacao: comportamento proposto (só com sanção).
	Pontuacao engine.ResultadoPontuacao
	// Servicos: justificativa dos relevantes serviços (art. 38, II).
	Servicos string
	// RotulosArt36: enunciado dos incisos I a VII, da biblioteca.
	RotulosArt36 [7]string
	// PendenciasCalculo: ressalvas do motor.
	PendenciasCalculo []string
}

var romanos = []string{"", "I", "II", "III", "IV", "V", "VI", "VII"}

var nomeSancao = map[engine.Sancao]string{engine.Repreensao: "REPREENSÃO", engine.Detencao: "DETENÇÃO", engine.Prisao: "PRISÃO"}
var incisoSancao = map[engine.Sancao]string{engine.Repreensao: "I", engine.Detencao: "II", engine.Prisao: "III"}
var alineaNatureza = map[engine.Natureza]string{engine.Leve: "a", engine.Media: "b", engine.Grave: "c"}
var nomeNatureza = map[engine.Natureza]string{engine.Leve: "LEVE", engine.Media: "MÉDIA", engine.Grave: "GRAVE"}
var nomeComportamento = map[engine.Comportamento]string{engine.Excepcional: "EXCEPCIONAL", engine.Otimo: "ÓTIMO", engine.Bom: "BOM", engine.Insuficiente: "INSUFICIENTE"}

var nomeGlobal = map[pads.Global]string{pads.Procedencia: "PROCEDENTE", pads.ProcedenciaParcial: "PARCIALMENTE PROCEDENTE", pads.Improcedencia: "IMPROCEDENTE"}

func ou(s, padrao string) string {
	if strings.TrimSpace(s) == "" {
		return padrao
	}
	return strings.TrimSpace(s)
}

// semPontoFinal tira a pontuação final de um texto de norma, para encaixe
// dentro de uma frase.
func semPontoFinal(s string) string {
	s = strings.TrimSpace(s)
	for strings.HasSuffix(s, ";") || strings.HasSuffix(s, ".") || strings.HasSuffix(s, "; e") || strings.HasSuffix(s, ":") {
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(s, "; e"), ";"), "."), ":"))
	}
	return s
}

func (d *Documento) trecho(slot string, t analise.Trecho, oQue string) {
	if t.Pendencia != "" || len(t.Paragrafos) == 0 {
		b := d.add(Paragrafo, APreencher+" — "+oQue+".")
		b.Trecho = slot
		d.pendencia(ou(t.Pendencia, "Falta redigir: "+oQue+"."))
		return
	}
	for _, p := range t.Paragrafos {
		d.Blocos = append(d.Blocos, Bloco{Tipo: Paragrafo, Texto: p.Texto, Origem: IA, Trecho: slot, Evid: p.Evid})
	}
	for _, l := range t.Lacunas {
		d.pendencia("Lacuna apontada na redação (" + oQue + "): " + l)
	}
}

// ComporPADS monta a minuta do julgamento de PADS (V4 §3): RELATÓRIO,
// FUNDAMENTAÇÃO e DECISÃO, com as providências finais.
func ComporPADS(e EntradaPADS) *Documento {
	d := &Documento{Procedimento: "PADS"}
	acusado := ou(e.Acusado, APreencher)
	identificacao := acusado
	if m := strings.TrimSpace(e.Matricula); m != "" {
		identificacao += ", matrícula " + m
	}
	memorando := ou(e.Memorando, "Memorando Disciplinar "+APreencher)

	d.add(Titulo, "JULGAMENTO DE PADS")
	d.add(Cabecalho, "Autoridade disciplinar: "+APreencher)
	d.add(Cabecalho, "Acusado(a): "+identificacao)
	d.add(Cabecalho, "Referência: PADS nº "+APreencher)

	// 1. RELATÓRIO
	d.add(Secao, "1. RELATÓRIO")
	d.add(Paragrafo, fmt.Sprintf("Trata-se do Processo Administrativo Disciplinar Sumário (PADS) nº %s, instaurado por meio do %s, em relação a %s.", APreencher, memorando, identificacao))
	d.add(Paragrafo, fmt.Sprintf("Conforme o Memorando Disciplinar, %s ao(à) acusado(a):",
		plural(len(e.Imputacoes), "foi imputada a seguinte transgressão disciplinar", "foram imputadas as seguintes transgressões disciplinares")))
	for i, im := range e.Imputacoes {
		t := fmt.Sprintf("%s – %s (%s", romanoDe(i+1), semPontoFinal(im.Conduta), im.Dispositivo)
		if im.TextoDispositivo != "" {
			t += ": “" + semPontoFinal(im.TextoDispositivo) + "”"
		}
		d.add(Item, t+")"+fimDeItem(i, len(e.Imputacoes)))
	}
	d.add(Paragrafo, "Ao(à) acusado(a) foi assegurado o exercício da ampla defesa e do contraditório, nos termos do art. 5º, inciso LV, da Constituição Federal e do art. 4º do RDPM.")
	d.trecho("relatorio", e.Relatorio, "relato dos fatos, da defesa e das provas colhidas")

	// 2. FUNDAMENTAÇÃO
	d.add(Secao, "2. FUNDAMENTAÇÃO")
	for _, im := range e.Imputacoes {
		d.add(Subtitulo, "Quanto à imputação do "+im.Dispositivo)
		d.trecho("fundamentacao:"+im.ID, im.Fundamentacao, "fundamentação da imputação do "+im.Dispositivo)
	}

	// 3. DECISÃO
	d.add(Secao, "3. DECISÃO")
	d.add(Paragrafo, "Dito isso, RESOLVO:")
	d.add(Paragrafo, fmt.Sprintf("Julgar %s a acusação descrita no %s, nos seguintes termos:", ou(nomeGlobal[e.Global], APreencher), memorando))
	for i, im := range e.Imputacoes {
		r := "IMPROCEDENTE"
		if im.Procedente {
			r = "PROCEDENTE"
		}
		d.add(Item, fmt.Sprintf("%s – imputação do %s: %s%s", romanoDe(i+1), im.Dispositivo, r, fimDeItem(i, len(e.Imputacoes))))
	}
	if len(e.Sancoes) == 0 {
		d.add(Paragrafo, fmt.Sprintf("ABSOLVER %s da acusação descrita no %s, pelos fundamentos expostos.", entreVirgulas(identificacao), memorando))
		providencias(d, false)
		fecho(d)
		return d
	}
	for _, s := range e.Sancoes {
		decisaoSancao(d, e, s, identificacao)
	}
	comportamento(d, e)
	privativa := false
	for _, s := range e.Sancoes {
		privativa = privativa || s.Sancao == engine.Detencao || s.Sancao == engine.Prisao
	}
	if privativa {
		// Redação aprovada (V4 §10), só com detenção ou prisão.
		id := identificacao + ","
		if strings.TrimSpace(e.Matricula) == "" {
			id = acusado + ", matrícula " + APreencher + ","
		}
		b := d.add(Paragrafo, fmt.Sprintf("Com base no artigo 45 do RDPM e considerando as especificidades da OPM, em conformidade com a Lei nº 2.484, de 06 de junho de 2011, "+
			"que inclui o § 4º ao artigo 2º da Lei nº 1.063/2002 — a qual impõe ao Estado o dever de fornecer alimentação ao militar em situação de prisão ou detenção administrativa —, "+
			"o(a) %s deverá iniciar o cumprimento da punição disciplinar em sua residência, às 08h do dia seguinte ao da sua ciência da nota de punição, "+
			"a ser elaborada após o esgotamento dos recursos cabíveis, sem prejuízo de suas atribuições.", id))
		b.Trecho = "clausula_residencial"
	}
	for _, p := range e.PendenciasCalculo {
		d.pendencia("Cálculo: " + p)
	}
	providencias(d, true)
	fecho(d)
	return d
}

// entreVirgulas fecha com vírgula a identificação que traz a matrícula como
// aposto ("SD PM FULANO, matrícula 000,"), para encaixe no meio da frase.
func entreVirgulas(identificacao string) string {
	if strings.Contains(identificacao, ", matrícula ") {
		return identificacao + ","
	}
	return identificacao
}

func romanoDe(n int) string {
	vals := []int{40, 10, 9, 5, 4, 1}
	sims := []string{"XL", "X", "IX", "V", "IV", "I"}
	var sb strings.Builder
	for i, v := range vals {
		for n >= v {
			sb.WriteString(sims[i])
			n -= v
		}
	}
	return sb.String()
}

func fimDeItem(i, n int) string {
	switch {
	case i == n-1:
		return "."
	case i == n-2:
		return "; e"
	}
	return ";"
}

// descricaoCircunstancia escreve uma circunstância contada pelo motor.
func descricaoCircunstancia(chave string, e EntradaPADS, s SancaoPADS, reconhecidas []engine.Circunstancia) string {
	switch chave {
	case "art. 38, I":
		return fmt.Sprintf("art. 38, inciso I — o(a) acusado(a) encontra-se no comportamento %s, com %d pontos", nomeComportamento[e.Pontuacao.ComportamentoAnterior], e.Pontuacao.PontosAnteriores)
	case "art. 38, II":
		return "art. 38, inciso II — relevantes serviços prestados à Corporação: " + semPontoFinal(ou(e.Servicos, APreencher))
	case "art. 39, I":
		return "art. 39, inciso I — o(a) acusado(a) encontra-se no comportamento INSUFICIENTE"
	case "art. 39, II":
		t := "art. 39, inciso II — prática, em conexão, de duas ou mais transgressões"
		if len(s.Secundarias) > 0 {
			t += " (" + strings.Join(s.Secundarias, "; ") + ")"
		}
		return t
	case "art. 39, III":
		return "art. 39, inciso III — reincidência, por haver punição anterior de mesma natureza cujo cumprimento terminou dentro do prazo do parágrafo único do art. 39"
	}
	for _, c := range reconhecidas {
		if c.Chave() == chave {
			ref := fmt.Sprintf("art. %d, inciso %s", c.Artigo, romanoDe(c.Inciso))
			if c.Alinea != "" {
				ref += ", alínea " + c.Alinea
			}
			return ref + " — " + semPontoFinal(ou(c.Fundamento, APreencher))
		}
	}
	return chave
}

func decisaoSancao(d *Documento, e EntradaPADS, s SancaoPADS, identificacao string) {
	identificacao = entreVirgulas(identificacao)
	slot := "art36:" + s.ImputacaoID
	d.add(Subtitulo, "Quanto à transgressão do "+s.Dispositivo)
	d.add(Paragrafo, fmt.Sprintf("Considerar a transgressão de natureza %s, à qual corresponde a punição de %s, nos termos do art. 41, inciso %s, e do art. 58, inciso II, alínea %s, do RDPM.",
		nomeNatureza[s.Natureza], nomeSancao[s.Sancao], incisoSancao[s.Sancao], alineaNatureza[s.Natureza]))
	if len(s.Secundarias) > 0 {
		d.add(Paragrafo, fmt.Sprintf("Por serem conexas a esta, %s não %s punição própria e %s como circunstância agravante, nos termos do art. 58, inciso VI, do RDPM: %s.",
			plural(len(s.Secundarias), "a seguinte transgressão procedente", "as seguintes transgressões procedentes"),
			plural(len(s.Secundarias), "recebe", "recebem"), plural(len(s.Secundarias), "é considerada", "são consideradas"), strings.Join(s.Secundarias, "; ")))
	}
	d.add(Paragrafo, "Em conformidade com o art. 36, incisos I a VII, do RDPM, verifico o seguinte:")
	for i := 1; i <= 7; i++ {
		rot := semPontoFinal(e.RotulosArt36[i-1])
		if rot != "" {
			rot = strings.ToUpper(rot[:1]) + rot[1:]
		} else {
			rot = "Fator do inciso " + romanos[i]
		}
		texto, evid := analise.AusenciaDeElementos, []string(nil)
		for _, f := range s.Fatores {
			if f.Inciso == i && strings.TrimSpace(f.Texto) != "" {
				texto, evid = f.Texto, f.Evid
			}
		}
		b := Bloco{Tipo: Item, Texto: fmt.Sprintf("%s – %s: %s", romanos[i], rot, texto), Origem: IA, Trecho: slot, Evid: evid}
		if texto == analise.AusenciaDeElementos {
			b.Origem = Fixo
		}
		d.Blocos = append(d.Blocos, b)
	}
	if s.PendenciaFatores != "" {
		d.pendencia(s.PendenciaFatores)
	}
	// Art. 37.
	if len(s.Justificacoes) == 0 {
		d.add(Paragrafo, "Quanto ao art. 37 do RDPM, não foram identificadas causas de justificação que excluam a transgressão.")
	} else {
		var js []string
		for _, c := range s.Justificacoes {
			js = append(js, descricaoCircunstancia(c.Chave(), e, s, s.Justificacoes))
		}
		d.add(Paragrafo, "Quanto ao art. 37 do RDPM, foi apontada causa de justificação: "+strings.Join(js, "; ")+". "+APreencher+" — a causa de justificação exclui a transgressão: confira o resultado desta imputação.")
		d.pendencia("Há causa de justificação reconhecida em imputação procedente (" + s.Dispositivo + ").")
	}
	lista := func(chaves []string, reconhecidas []engine.Circunstancia) string {
		var xs []string
		for _, k := range chaves {
			xs = append(xs, descricaoCircunstancia(k, e, s, reconhecidas))
		}
		return strings.Join(xs, "; ")
	}
	if len(s.AtenuantesContadas) == 0 {
		d.add(Paragrafo, "Quanto ao art. 38 do RDPM, não há circunstâncias atenuantes em favor do(a) acusado(a).")
	} else {
		d.add(Paragrafo, fmt.Sprintf("Quanto ao art. 38 do RDPM, %s: %s.",
			plural(len(s.AtenuantesContadas), "reconheço a seguinte circunstância atenuante", "reconheço as seguintes circunstâncias atenuantes"), lista(s.AtenuantesContadas, s.Atenuantes)))
	}
	if len(s.AgravantesContadas) == 0 {
		d.add(Paragrafo, "Quanto ao art. 39 do RDPM, não há circunstâncias agravantes em desfavor do(a) acusado(a).")
	} else {
		d.add(Paragrafo, fmt.Sprintf("Quanto ao art. 39 do RDPM, %s: %s.",
			plural(len(s.AgravantesContadas), "reconheço a seguinte circunstância agravante", "reconheço as seguintes circunstâncias agravantes"), lista(s.AgravantesContadas, s.Agravantes)))
	}
	if s.Sancao == engine.Repreensao {
		d.add(Paragrafo, fmt.Sprintf("Assim, com base no art. 41, inciso I, e no art. 58, inciso II, alínea a, do RDPM, imponho a %s a punição disciplinar de REPREENSÃO, por ter praticado a transgressão disciplinar do %s.",
			identificacao, s.Dispositivo))
		return
	}
	dias := APreencher
	if s.Dias > 0 {
		dias = numeroExtenso(s.Dias)
	}
	d.add(Paragrafo, fmt.Sprintf("Assim, com base no art. 41, inciso %s, e no art. 58, inciso II, alínea %s, do RDPM, e consideradas as circunstâncias acima, imponho a %s a punição disciplinar de %s %s de %s, por ter praticado a transgressão disciplinar do %s.",
		incisoSancao[s.Sancao], alineaNatureza[s.Natureza], identificacao, dias, plural(s.Dias, "dia", "dias"), nomeSancao[s.Sancao], s.Dispositivo))
}

// comportamento escreve a proposta de pontuação (V4 §§8, 15.4), sem afirmar
// alteração administrativa já efetivada (PADS-RED-006).
func comportamento(d *Documento, e EntradaPADS) {
	p := e.Pontuacao
	if !p.Concluido {
		d.add(Paragrafo, "Pontuação e comportamento disciplinar após a punição: "+APreencher+".")
		d.pendencia("O cálculo do comportamento não foi concluído: informe a pontuação e o comportamento na minuta.")
		return
	}
	final := nomeComportamento[p.ComportamentoFinal]
	if p.Regressao != engine.SemRegressao {
		// Justificativa de referência aprovada (V4 §15.4), adaptada ao caso.
		b := d.add(Paragrafo, fmt.Sprintf("Considerando que as transgressões disciplinares reconhecidas no presente procedimento, analisadas conjuntamente, satisfazem os requisitos para a regressão especial de comportamento prevista no art. 70 do RDPM, "+
			"propõe-se a reclassificação do(a) acusado(a) para o comportamento %s, com atribuição de %s pontos. A pontuação decorre da aplicação da regra especial de regressão, que estabelece diretamente a pontuação correspondente ao novo comportamento. "+
			"Assim, para fins do cálculo proposto, não se acumulam descontos ordinários referentes às punições integrantes do mesmo julgamento, evitando-se a incidência simultânea de critérios distintos sobre o conjunto de transgressões considerado.",
			final, numeroExtenso(p.PontosFinais)))
		b.Trecho = "comportamento"
		return
	}
	total := 0
	for _, x := range p.Descontos {
		total += x.Pontos
	}
	situacao := "permanecendo no comportamento " + final
	if p.ComportamentoFinal != p.ComportamentoAnterior {
		situacao = "passando ao comportamento " + final
	}
	b := d.add(Paragrafo, fmt.Sprintf("Conforme o art. 71 do RDPM, propõe-se a subtração de %s %s do comportamento disciplinar do(a) acusado(a), que passa de %d para %d pontos, %s. A alteração depende do ato administrativo competente.",
		numeroExtenso(total), plural(total, "ponto", "pontos"), p.PontosAnteriores, p.PontosFinais, situacao))
	b.Trecho = "comportamento"
	if p.SaldoNegativo {
		d.pendencia("O saldo de pontos calculado ficou negativo: confira como deve constar.")
	}
}

// providencias: V4 §3.4. A nota de punição só depois de a decisão se tornar
// irrecorrível (RDPM, art. 32, parágrafo único).
func providencias(d *Documento, comPunicao bool) {
	d.add(Paragrafo, "Como próximas providências:")
	itens := []string{"Publicar este julgamento em Boletim Interno", "Notificar o(a) acusado(a) desta decisão"}
	if comPunicao {
		itens = append(itens, "Aguardar o prazo regulamentar para a interposição de recurso",
			"Elaborar e publicar a nota de punição somente depois de a decisão se tornar irrecorrível (RDPM, art. 32, parágrafo único)")
	}
	itens = append(itens, "Manter os documentos do presente PADS em arquivo")
	for i, t := range itens {
		d.add(Item, romanoDe(i+1)+" – "+t+fimDeItem(i, len(itens)))
	}
}

func fecho(d *Documento) {
	d.add(Fecho, "CUMPRA-SE.")
	d.add(Fecho, APreencher+" (local), data conforme a assinatura eletrônica.")
	d.add(Assinatura, APreencher+" (nome, posto e função da autoridade disciplinar)")
}

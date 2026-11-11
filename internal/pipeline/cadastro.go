package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/segment"
	"github.com/DiogoBrazil/sys-executor/internal/pseudonimo"
	"github.com/DiogoBrazil/sys-executor/internal/rules/enquadramento"
	"github.com/DiogoBrazil/sys-executor/internal/session"
	"github.com/DiogoBrazil/sys-executor/internal/texto"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// Cadastro de peças (V4 §48.4): o usuário declara cada peça e seus
// arquivos; o Go junta os arquivos, na ordem processual, num único PDF da
// sessão (a numeração de páginas e as evidências continuam globais) e cada
// peça declarada vira uma peça dos autos com a função informada.

// funcaoDeclarada: tipo e papel do cadastro → função documental.
func funcaoDeclarada(p workflow.PecaCadastro) segment.Funcao {
	switch p.Tipo {
	case workflow.PecaMemorando:
		return segment.MemorandoDisciplinar
	case workflow.PecaPortaria:
		return segment.PortariaSR
	case workflow.PecaDefesa:
		return segment.DefesaPrevia
	case workflow.PecaAlegacoes:
		return segment.AlegacoesFinais
	case workflow.PecaDepoimento:
		switch p.Papel {
		case workflow.PapelDeclarante:
			return segment.TermoDeclaracoes
		case workflow.PapelSindicado:
			return segment.Interrogatorio
		}
		return segment.TermoInquiricao
	}
	return segment.Anexo
}

// ordemProcessual: peça central, defesa prévia, oitivas, outros documentos
// e alegações finais (a ordem do cadastro vale dentro de cada grupo).
var ordemProcessual = map[string]int{
	workflow.PecaMemorando: 0, workflow.PecaPortaria: 0, workflow.PecaDefesa: 1,
	workflow.PecaDepoimento: 2, workflow.PecaOutro: 3, workflow.PecaAlegacoes: 4,
}

func pecasNaOrdem(cad *workflow.Cadastro) []workflow.PecaCadastro {
	var ps []workflow.PecaCadastro
	for _, p := range cad.Pecas {
		if !p.NaoApresentada && len(p.Arquivos) > 0 {
			ps = append(ps, p)
		}
	}
	sort.SliceStable(ps, func(i, j int) bool { return ordemProcessual[ps[i].Tipo] < ordemProcessual[ps[j].Tipo] })
	return ps
}

var papeis = map[string]string{
	workflow.PapelTestemunha: "testemunha", workflow.PapelDeclarante: "declarante", workflow.PapelSindicado: "sindicado",
}

// tituloDeclarado: rótulo da peça nos autos (nomes são pseudonimizados
// antes de qualquer envio, como todo o texto).
func tituloDeclarado(cad *workflow.Cadastro, p workflow.PecaCadastro) string {
	t := rotulos[funcaoDeclarada(p)]
	if p.Tipo == workflow.PecaDepoimento {
		t = "Depoimento de " + papeis[p.Papel]
		var nomes []string
		for _, id := range p.PessoaIDs {
			if ps, ok := cad.Pessoa(id); ok {
				nomes = append(nomes, ps.Nome)
			}
		}
		if len(nomes) > 0 {
			t += " — " + strings.Join(nomes, ", ")
		}
	}
	if p.Titulo != "" {
		t += " — " + p.Titulo
	}
	if p.Data != "" {
		t += " (" + p.Data + ")"
	}
	return t
}

func (p *Pipeline) juntador() (pdf.Juntador, error) {
	if p.Juntador != nil {
		return p.Juntador, nil
	}
	if j, ok := p.Extrator.(pdf.Juntador); ok {
		return j, nil
	}
	return nil, erros.Com(erros.InternoInesperado, errors.New("junção de PDF indisponível"))
}

// juntarCadastro lê os arquivos da sessão na ordem processual e grava o PDF
// juntado como os autos da tarefa.
func (p *Pipeline) juntarCadastro(ctx context.Context, s *session.Sessao, cad *workflow.Cadastro) ([]segment.Declarada, error) {
	j, err := p.juntador()
	if err != nil {
		return nil, err
	}
	pecas := pecasNaOrdem(cad)
	var arquivos [][]byte
	var dono []int // índice da peça de cada arquivo
	for i, pc := range pecas {
		for _, a := range pc.Arquivos {
			b, err := s.Ler(a.NomeLogico(pc.ID))
			if err != nil {
				return nil, err
			}
			arquivos = append(arquivos, b)
			dono = append(dono, i)
		}
	}
	juntado, paginas, err := j.Juntar(ctx, arquivos)
	if err != nil {
		return nil, err
	}
	ds := make([]segment.Declarada, len(pecas))
	for i, pc := range pecas {
		ds[i] = segment.Declarada{Cadastro: pc.ID, Funcao: funcaoDeclarada(pc), Titulo: tituloDeclarado(cad, pc), SEI: pc.SEI}
	}
	for k, n := range paginas {
		ds[dono[k]].Paginas += n
	}
	if err := s.Gravar(workflow.NomeAutos, juntado); err != nil {
		return nil, err
	}
	return ds, nil
}

// semearMapa registra no mapa de pseudônimos os envolvidos cadastrados, com
// as outras grafias informadas (antes da detecção nas páginas). Devolve se
// algum envolvido foi marcado como criança menor de 13 anos.
func semearMapa(m *pseudonimo.Mapa, cad *workflow.Cadastro) (crianca bool) {
	if cad == nil {
		return false
	}
	for _, ps := range cad.Pessoas {
		m.CadastrarPessoa(ps.Nome, ps.Grafias)
		crianca = crianca || ps.Crianca
	}
	return crianca
}

// ---- conferência da delimitação com o cadastro (E5/E9) ----

// mesmaPessoa: os termos do nome mais curto estão todos no mais longo
// (o Memorando costuma trazer posto, graduação e matrícula junto do nome).
func mesmaPessoa(a, b string) bool {
	ta, tb := termosNome(a), termosNome(b)
	if len(ta) == 0 || len(tb) == 0 {
		return false
	}
	// Palavras grudadas pela leitura ("DACRUZ" = "DA CRUZ"): as letras do
	// nome mais curto aparecem, na ordem, dentro do mais longo.
	ja, jb := juntarNome(a), juntarNome(b)
	if len(ja) > len(jb) {
		ja, jb = jb, ja
	}
	if len(ja) >= 6 && strings.Contains(jb, ja) {
		return true
	}
	if len(ta) > len(tb) {
		ta, tb = tb, ta
	}
	conj := map[string]bool{}
	for _, t := range tb {
		conj[t] = true
	}
	for _, t := range ta {
		if !conj[t] {
			return false
		}
	}
	return true
}

// juntarNome: as palavras do nome, sem espaços, sem números, sem "mat."/
// "matrícula" e sem marcas de mascaramento.
func juntarNome(s string) string {
	var b strings.Builder
	for _, t := range texto.Termos(s) {
		if t == "mat" || t == "matricula" || strings.IndexFunc(t, unicode.IsDigit) >= 0 {
			continue
		}
		b.WriteString(t)
	}
	return b.String()
}

func termosNome(s string) []string {
	var out []string
	for _, t := range texto.Termos(s) {
		if t == "mat" || t == "matricula" || strings.IndexFunc(t, unicode.IsDigit) >= 0 {
			continue // "Mat. 100…" lido junto com o nome
		}
		if len([]rune(t)) > 2 { // sem "de", "da", "pm"…
			out = append(out, t)
		}
	}
	return out
}

func nomeComPosto(ps workflow.PessoaCadastro) string {
	return strings.TrimSpace(ps.PostoGrad + " " + ps.Nome)
}

// enquadramentoCadastrado é um dispositivo informado no assistente, com o
// fato de origem.
type enquadramentoCadastrado struct {
	workflow.EnquadramentoCadastro
	Fato    workflow.FatoCadastro
	NumFato int // posição do fato no cadastro (1, 2…), para o rótulo
}

func enquadramentosDoCadastro(cad *workflow.Cadastro) []enquadramentoCadastrado {
	if cad == nil {
		return nil
	}
	var out []enquadramentoCadastrado
	for i, f := range cad.Fatos {
		for _, e := range f.Enquadramentos {
			out = append(out, enquadramentoCadastrado{EnquadramentoCadastro: e, Fato: f, NumFato: i + 1})
		}
	}
	return out
}

// rotuloFato: "fato 1" (a posição; o ID interno cresce a cada fato criado).
func rotuloFato(x enquadramentoCadastrado) string {
	f := x.Fato
	s := fmt.Sprintf("fato %d", x.NumFato)
	if f.Resumo != "" {
		s += " (" + f.Resumo + ")"
	}
	return s
}

func plural(n int, um, varios string) string {
	if n == 1 {
		return um
	}
	return fmt.Sprintf(varios, n)
}

// opcaoVoltarCadastro: volta ao assistente com o cadastro preservado.
var opcaoVoltarCadastro = workflow.Opcao{ID: "outro", Rotulo: "Voltar ao cadastro", Destino: workflow.RecebendoUpload}

// perguntaDeclaradas: conferência do conteúdo com a função declarada
// (V4 §§30.2, 48.4). A declaração do usuário não é trocada pelo app.
func perguntaDeclaradas(tipo workflow.Tipo, d *Dados) workflow.Pergunta {
	nome, peca := "Julgamento de PADS", "o Memorando Disciplinar"
	if tipo == workflow.TipoSR {
		nome, peca = "Relatório de Sindicância Regular", "a Portaria de instauração"
	}
	p := workflow.Pergunta{ID: perguntaCompat, Opcoes: []workflow.Opcao{
		opcaoVoltarCadastro,
		{ID: "continuar", Rotulo: "Continuar como cadastrado"},
		{ID: "cancelar", Rotulo: "Cancelar a tarefa", Destino: workflow.Cancelada},
	}}
	comp := d.Compat
	switch comp.Resultado {
	case segment.PecaIlegivel:
		p.Titulo = "A peça central está ilegível"
		p.Contexto = fmt.Sprintf("O arquivo cadastrado como %s tem página de leitura de baixa qualidade. "+
			"Volte ao cadastro e anexe uma versão mais legível, se tiver.", peca)
	case segment.Incompativel:
		p.Titulo = "O documento central parece de outro procedimento"
		p.Contexto = fmt.Sprintf("Você escolheu %s, mas o arquivo cadastrado como %s parece a peça central do outro procedimento. "+
			"O tipo de minuta não é trocado automaticamente: confira o arquivo ou cancele e escolha o outro tipo.", nome, peca)
		p.Opcoes = []workflow.Opcao{opcaoVoltarCadastro, {ID: "cancelar", Rotulo: "Cancelar a tarefa", Destino: workflow.Cancelada}}
	case segment.PecaAusente:
		p.Titulo = "Peça central sem arquivo"
		p.Contexto = fmt.Sprintf("Anexe %s no cadastro.", peca)
		p.Opcoes = []workflow.Opcao{opcaoVoltarCadastro, {ID: "cancelar", Rotulo: "Cancelar a tarefa", Destino: workflow.Cancelada}}
	default:
		p.Titulo = "Confira as peças cadastradas"
		linhas := []string{"O conteúdo de " + plural(len(comp.Divergencias), "uma peça", "%d peças") +
			" parece diferente do tipo cadastrado. Vale o que você cadastrou; confira antes de continuar:"}
		for _, dv := range comp.Divergencias {
			titulo := dv.Peca
			for _, pc := range d.Pecas {
				if pc.ID == dv.Peca {
					titulo = pc.Titulo
				}
			}
			linhas = append(linhas, fmt.Sprintf("• %s: cadastrada como %s; a página %d parece %s.",
				titulo, strings.ToLower(rotulos[dv.Declarada]), dv.Pagina, strings.ToLower(rotulos[dv.Aparente])))
		}
		p.Contexto = strings.Join(linhas, "\n")
	}
	return p
}

// conferenciaCadastro: o que o cadastro acrescenta ao formulário PADS-ESC-001.
type conferenciaCadastro struct {
	padroes        map[string]string   // valores iniciais por campo (o usuário confere)
	alertas        map[string][]string // por imputação
	suprimir       map[string][]string // pendências da leitura já resolvidas pelo cadastro, por imputação
	alertasAcusado []string
	acusado        string // valor inicial do acusado ("" = o lido pela IA)
	extras         []enquadramentoCadastrado
}

// Pendências da leitura do Memorando que o formulário ou o cadastro já
// resolvem (V4: não repetir pergunta já respondida).
const (
	pendInciso   = analise.PendenciaInciso
	pendNatureza = analise.PendenciaNatureza
)

func conferirComCadastro(r analise.PropostaDelimitacaoPADS, cad *workflow.Cadastro) conferenciaCadastro {
	out := conferenciaCadastro{padroes: map[string]string{}, alertas: map[string][]string{}, suprimir: map[string][]string{}}
	if cad == nil {
		return out
	}
	// Acusado: a leitura do Memorando prevalece; o cadastro confere.
	if ps, ok := cad.Pessoa(cad.AcusadoID); ok {
		switch len(r.Acusados) {
		case 0:
			out.acusado = nomeComPosto(ps)
		case 1:
			if mesmaPessoa(ps.Nome, r.Acusados[0].Nome) {
				// A mesma pessoa: vale a grafia do cadastro, com o posto.
				out.acusado = nomeComPosto(ps)
			} else {
				out.alertasAcusado = append(out.alertasAcusado, fmt.Sprintf(
					"O acusado informado no cadastro (%s) não coincide com o lido no Memorando (%s). Confira no Memorando.",
					nomeComPosto(ps), strings.TrimSpace(r.Acusados[0].PostoGraduacao+" "+r.Acusados[0].Nome)))
			}
		default:
			for _, a := range r.Acusados {
				if mesmaPessoa(ps.Nome, a.Nome) {
					// O campo é uma escolha entre os lidos: o valor é o nome lido.
					out.acusado = a.Nome
					break
				}
			}
			if out.acusado == "" {
				out.alertasAcusado = append(out.alertasAcusado, fmt.Sprintf(
					"O acusado informado no cadastro (%s) não está entre os lidos no Memorando.", nomeComPosto(ps)))
			}
		}
	}
	// Dispositivos: pareados pelo dispositivo citado na imputação.
	restantes := enquadramentosDoCadastro(cad)
	if len(restantes) == 0 {
		return out
	}
	usado := make([]bool, len(restantes))
	// Imputações que citam só o artigo, por diploma e artigo (pareamento
	// pelo artigo só quando não há ambiguidade).
	soArtigo := map[string]int{}
	for _, im := range r.Imputacoes {
		if _, ok := citadoNoMemorando(im); !ok && im.Inciso == "" {
			if dip := enquadramento.DiplomaDoCitado(im.Diploma, im.Artigo); dip != "" {
				soArtigo[dip+"-"+strconv.Itoa(im.Artigo)]++
			}
		}
	}
	for _, im := range r.Imputacoes {
		citado, ok := citadoNoMemorando(im)
		achou := -1
		for i, x := range restantes {
			if ok && !usado[i] && x.DispositivoID == citado.ID {
				achou = i
				break
			}
		}
		if achou < 0 && !ok && im.Inciso == "" {
			if dip := enquadramento.DiplomaDoCitado(im.Diploma, im.Artigo); dip != "" && soArtigo[dip+"-"+strconv.Itoa(im.Artigo)] == 1 {
				cand := -1
				for i, x := range restantes {
					if d, ok := enquadramento.PorID(x.DispositivoID); ok && !usado[i] && d.Diploma == dip && d.Artigo == im.Artigo {
						if cand >= 0 {
							cand = -2 // dois do mesmo artigo no cadastro: o usuário escolhe
							break
						}
						cand = i
					}
				}
				if cand >= 0 {
					achou = cand
					d, _ := enquadramento.PorID(restantes[cand].DispositivoID)
					out.suprimir[im.ID] = append(out.suprimir[im.ID], pendInciso)
					out.alertas[im.ID] = append(out.alertas[im.ID], fmt.Sprintf(
						"O Memorando cita só o art. %d; o inciso %s é o informado no seu cadastro. Confira.", im.Artigo, d.Inciso))
				}
			}
		}
		if achou < 0 {
			out.alertas[im.ID] = append(out.alertas[im.ID],
				"Nenhum dispositivo informado no cadastro corresponde ao citado nesta imputação. Escolha o enquadramento abaixo.")
			continue
		}
		usado[achou] = true
		x := restantes[achou]
		preencherEnquadramento(out.padroes, im.ID, x.EnquadramentoCadastro)
		if x.Fato.Data != "" && x.Fato.Data != dataTexto(im.Data) {
			out.alertas[im.ID] = append(out.alertas[im.ID], fmt.Sprintf(
				"A data informada no cadastro (%s) é diferente da lida no Memorando (%s). Confira.", x.Fato.Data, orTraco(dataTexto(im.Data))))
		}
	}
	for i, x := range restantes {
		if !usado[i] {
			k := len(out.extras) + 1
			out.extras = append(out.extras, x)
			preencherEnquadramento(out.padroes, idExtra(k), x.EnquadramentoCadastro)
		}
	}
	return out
}

func preencherEnquadramento(pad map[string]string, id string, e workflow.EnquadramentoCadastro) {
	d, ok := enquadramento.PorID(e.DispositivoID)
	if !ok {
		return
	}
	pad[campoDiploma+id] = d.Diploma
	if d.Diploma == enquadramento.RDPM {
		pad[campoDispRDPM+id] = d.ID
		return
	}
	pad[campoDispEst+id] = d.ID
	if e.Natureza != "" {
		pad[campoNatureza+id] = e.Natureza
	}
}

func orTraco(s string) string {
	if s == "" {
		return "sem data"
	}
	return s
}

// conferirObjetoSR: sindicados do cadastro × os lidos na Portaria.
func conferirObjetoSR(r analise.PropostaObjetoSR, cad *workflow.Cadastro) (porSindicado map[string][]string, gerais []string) {
	porSindicado = map[string][]string{}
	if cad == nil || len(cad.Sindicados) == 0 {
		return porSindicado, nil
	}
	achados := map[string]bool{}
	for _, s := range r.Sindicados {
		ok := false
		for _, id := range cad.Sindicados {
			if ps, existe := cad.Pessoa(id); existe && mesmaPessoa(ps.Nome, s.Nome) {
				achados[id], ok = true, true
			}
		}
		if !ok {
			porSindicado[s.ID] = append(porSindicado[s.ID], "Este sindicado, lido na Portaria, não foi informado no cadastro. Confira.")
		}
	}
	for _, id := range cad.Sindicados {
		if ps, existe := cad.Pessoa(id); existe && !achados[id] {
			gerais = append(gerais, fmt.Sprintf("O sindicado informado no cadastro (%s) não foi encontrado na leitura da Portaria. Confira na Portaria.", nomeComPosto(ps)))
		}
	}
	return porSindicado, gerais
}

// opcaoOutroPDF: com cadastro, volta a ele (preservado); sem, novo PDF.
func opcaoOutroPDF(c *workflow.Contexto) workflow.Opcao {
	if c.Cadastro() != nil {
		return opcaoVoltarCadastro
	}
	return workflow.Opcao{ID: "outro", Rotulo: "Enviar outro PDF", Destino: workflow.RecebendoUpload}
}

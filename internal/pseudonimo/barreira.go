package pseudonimo

// Barreira de privacidade antes do envio à IA (V4 §48.6, TRV-SEG-013).
//
//   - Camada 1 (Motor.Gerar, depois de ocultar): nenhum dado pessoal já
//     conhecido pode restar no pedido — instruções, dados, histórico de
//     reenvio e schema. Dado com padrão inequívoco (CPF válido, e-mail,
//     telefone, RG/matrícula com rótulo) e nome com âncora que apareçam no
//     texto do pedido são aprendidos e ocultados; possível nome sem âncora
//     ainda não decidido pelo usuário pára o envio para revisão item a item.
//   - Camada 2 (Inspecionar, no transporte HTTP): os bytes exatos da
//     requisição são decodificados e conferidos; qualquer resíduo bloqueia.
//
// Nenhuma mensagem de erro daqui contém o dado encontrado.

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ErrVazamento: restou dado pessoal conhecido no conteúdo a enviar. O envio
// é bloqueado sem opção de ignorar (é falha do aplicativo).
type ErrVazamento struct {
	Onde  string // "pedido.usuario", "pedido.sistema", "requisicao" …
	Tipos []Tipo
}

func (e ErrVazamento) Error() string {
	ts := make([]string, len(e.Tipos))
	for i, t := range e.Tipos {
		ts[i] = string(t)
	}
	return fmt.Sprintf("barreira de privacidade: dado pessoal reconhecido ainda presente em %s (%s); nada foi enviado", e.Onde, strings.Join(ts, ", "))
}

// ErrRevisao: há possíveis nomes de pessoas não reconhecidos e ainda não
// decididos pelo usuário. Itens só vão para a pergunta de revisão (tela
// local), nunca para a mensagem de erro.
type ErrRevisao struct{ Itens []string }

func (e ErrRevisao) Error() string {
	return fmt.Sprintf("barreira de privacidade: %d possível(is) nome(s) não reconhecido(s) aguardando revisão; nada foi enviado", len(e.Itens))
}

// marcadoresDaInstrucao: exemplos citados em InstrucaoMarcadores (não são
// dados; podem aparecer no pedido sem existir no mapa).
var marcadoresDaInstrucao = func() map[string]bool {
	m := map[string]bool{}
	for _, x := range reMarcador.FindAllString(InstrucaoMarcadores, -1) {
		m[x] = true
	}
	return m
}()

// residuos aponta os tipos de dado pessoal ainda presentes no texto:
// grafias conhecidas (mesmas regras do Ocultar) e dados com padrão
// inequívoco. Com marcadores=true, também marcadores que não são deste mapa
// (integridade do que o aplicativo montou; não vale para o histórico, que
// devolve à IA a resposta errada dela no pedido de correção).
func (m *Mapa) residuos(t string, marcadores bool) []Tipo {
	vistos := map[Tipo]bool{}
	m.varrer(t, func(_, _ int, k string) {
		// Palavra solta derivada de um nome ("Costa", de "Fulano da Costa")
		// é ocultada, mas não serve de prova de vazamento: pode ser palavra
		// comum nas instruções ou no schema, e bloquear ali não teria saída.
		if tp, ok := m.chave[k]; ok && !m.fraca[k] {
			vistos[tp] = true
		}
	})
	for _, mk := range reMarcador.FindAllString(t, -1) {
		if _, ok := m.forma[mk]; marcadores && !ok && !marcadoresDaInstrucao[mk] {
			vistos["MARCADOR"] = true
		}
	}
	for _, x := range reCPF.FindAllString(t, -1) {
		if cpfValido(x) {
			vistos[CPF] = true
		}
	}
	if reEmail.MatchString(t) {
		vistos[Email] = true
	}
	if reTelefone.MatchString(t) {
		vistos[Telefone] = true
	}
	out := make([]Tipo, 0, len(vistos))
	for tp := range vistos {
		out = append(out, tp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Possível nome sem âncora: duas a dez palavras com inicial maiúscula na
// mesma linha (conectores permitidos), depois divididas pelas paradas e pelo
// "e" entre dois nomes. Sequências inteiras em maiúsculas só contam quando a
// linha também tem minúsculas (títulos e cabeçalhos não).
var reCandidato = regexp.MustCompile(palavra + `(?:[ \t]+(?:` + conector + `[ \t]+)?` + palavra + `){1,9}`)

// Suspeitos devolve, sem repetição e na ordem em que aparecem, os possíveis
// nomes do texto que não estão no mapa nem foram liberados pelo usuário.
// O texto deve estar já ocultado (os nomes conhecidos viraram marcadores).
func (m *Mapa) Suspeitos(t string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.suspeitos(t)
}

func (m *Mapa) suspeitos(t string) []string {
	var out []string
	vistos := map[string]bool{}
	comuns := minusculasDoTexto(t)
	for _, linha := range strings.Split(t, "\n") {
		temMinuscula := temMinusculaForaConector(linha)
		linha = mascararQualificacao(linha)
		for _, ix := range reCandidato.FindAllStringIndex(linha, -1) {
			cand := linha[ix[0]:ix[1]]
			if fraseNaoNome(cand) {
				continue
			}
			for _, seg := range segmentosDeNome(cand) {
				if !temMinuscula && !temMinusculaForaConector(seg) {
					continue
				}
				if fraseNaoNome(seg) || soPalavrasComuns(seg, comuns) {
					continue
				}
				k := chaveNormal(seg)
				if vistos[k] || m.conhecida(k) {
					continue
				}
				vistos[k] = true
				out = append(out, seg)
			}
		}
	}
	return out
}

// conhecida: a chave já é dado registrado, foi liberada, ou está contida
// numa chave registrada (pedaço de um dado já ocultado; os pedaços de nome
// de pessoa são ocultados pela derivação, ver derivar). A liberação vale só
// para a frase exata: liberar "Escola Estadual Fulano Silva" não libera
// "Fulano Silva" sozinho.
func (m *Mapa) conhecida(k string) bool {
	if _, ok := m.chave[k]; ok || m.liberados[k] {
		return true
	}
	for c := range m.chave {
		if strings.Contains(" "+c+" ", " "+k+" ") {
			return true
		}
	}
	return false
}

// segmentosDeNome divide a sequência nas palavras que não são nome (lista
// de paradas: instituição, cargo, documento, lugar, mês…; números romanos;
// ruído de OCR) e devolve os pedaços com ao menos duas palavras de nome,
// separando "Fulano Silva e Beltrano Costa" em dois.
func segmentosDeNome(s string) []string {
	var out []string
	var atual []string
	nomes := 0
	fechar := func() {
		for len(atual) > 0 && eConector(atual[len(atual)-1]) {
			atual = atual[:len(atual)-1]
		}
		if nomes >= 2 {
			out = append(out, dividirE(strings.Join(atual, " "))...)
		}
		atual, nomes = nil, 0
	}
	for _, w := range strings.Fields(s) {
		lw := strings.ToLower(w)
		if paradas[lw] || paradas[semAcento(lw)] || reRomano.MatchString(w) || ruido(w) {
			fechar()
			continue
		}
		if eConector(w) {
			if len(atual) > 0 {
				atual = append(atual, w)
			}
			continue
		}
		atual = append(atual, w)
		nomes++
	}
	fechar()
	return out
}

// conferir é a camada 1, sobre o pedido já ocultado.
func (m *Mapa) conferir(sistema, usuario string, historico []string, schema []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	type campo struct {
		onde, texto           string
		suspeitos, marcadores bool
	}
	campos := []campo{{"pedido.sistema", sistema, false, true}, {"pedido.schema", string(schema), false, true}, {"pedido.usuario", usuario, true, true}}
	for _, h := range historico {
		campos = append(campos, campo{"pedido.historico", h, true, false})
	}
	var rev []string
	vistos := map[string]bool{}
	for _, c := range campos {
		if ts := m.residuos(c.texto, c.marcadores); len(ts) > 0 {
			return ErrVazamento{Onde: c.onde, Tipos: ts}
		}
		if c.suspeitos {
			for _, s := range m.suspeitos(c.texto) {
				if k := chaveNormal(s); !vistos[k] {
					vistos[k] = true
					rev = append(rev, s)
				}
			}
		}
	}
	if len(rev) > 0 {
		return ErrRevisao{Itens: rev}
	}
	return nil
}

// Inspecionar é a camada 2: recebe o corpo exato da requisição HTTP (JSON),
// percorre todos os textos e bloqueia se restar dado pessoal conhecido ou
// com padrão inequívoco. Não aprende nem oculta: só bloqueia.
func (m *Mapa) Inspecionar(corpo []byte) error {
	if len(corpo) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(corpo, &v); err != nil {
		// Corpo que não é JSON não é conferível: não sai.
		return ErrVazamento{Onde: "requisicao (corpo ilegível)", Tipos: []Tipo{"DESCONHECIDO"}}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	vistos := map[Tipo]bool{}
	var percorrer func(any)
	percorrer = func(x any) {
		switch y := x.(type) {
		case string:
			for _, tp := range m.residuos(y, false) {
				vistos[tp] = true
			}
		case []any:
			for _, z := range y {
				percorrer(z)
			}
		case map[string]any:
			for _, z := range y {
				percorrer(z)
			}
		}
	}
	percorrer(v)
	if len(vistos) == 0 {
		return nil
	}
	ts := make([]Tipo, 0, len(vistos))
	for tp := range vistos {
		ts = append(ts, tp)
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
	return ErrVazamento{Onde: "requisicao", Tipos: ts}
}

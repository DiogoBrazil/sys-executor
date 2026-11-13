package workflow

// Cadastro das peças (V4 §48.4): no lugar do PDF monolítico, o usuário
// cadastra peça por peça — Memorando ou Portaria, defesa prévia, depoimentos
// (com o depoente e o papel), alegações finais e outros documentos —, cada
// uma com um ou vários arquivos. Tudo vive só na sessão efêmera (V4 §29:
// sem rascunho persistente); os arquivos ficam cifrados na sessão e a
// interface só vê os metadados (nome, páginas, resumo do hash).

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/rules/enquadramento"
)

// Tipos de peça declarados pelo usuário.
const (
	PecaMemorando  = "MEMORANDO"
	PecaPortaria   = "PORTARIA"
	PecaDefesa     = "DEFESA_PREVIA"
	PecaAlegacoes  = "ALEGACOES_FINAIS"
	PecaDepoimento = "DEPOIMENTO"
	PecaOutro      = "OUTRO"
)

// Papéis do depoente (a escolha no formulário não presume compromisso
// legal: a condição processual é a documentada no termo).
const (
	PapelTestemunha = "TESTEMUNHA"
	PapelDeclarante = "DECLARANTE"
	PapelSindicado  = "SINDICADO"
)

// Limites do cadastro (proteção contra uso indevido, não de conteúdo).
const (
	MaxArquivosCadastro = 300
	MaxBytesCadastro    = 1 << 30 // 1 GiB no total (memória do pdfium)
)

// PessoaCadastro é um envolvido. A mesma pessoa pode ter vários papéis e
// várias oitivas; grafias e apelidos ajudam a pseudonimização.
type PessoaCadastro struct {
	ID        string   `json:"id"`
	Nome      string   `json:"nome"`
	PostoGrad string   `json:"postoGrad"`
	Grafias   []string `json:"grafias"`
	Crianca   bool     `json:"crianca"` // menor de 13 anos (V4 §48.6)
	// Instituicao: ente ofendido (ex.: "Administração Pública"), V4 §23.3,
	// c. Só pode ser ofendido; não tem posto e não é pseudonimizado (não é
	// dado pessoal).
	Instituicao bool `json:"instituicao"`
	// Civil: pessoa civil (ofendido, testemunha ou declarante), sem posto ou
	// graduação. Falso = militar. Sindicado e acusado são sempre militares.
	Civil bool `json:"civil"`
}

// PostosGraduacoes: lista de posto/graduação da PMRO oferecida no cadastro
// (pedido do usuário, 12/11/2026).
var PostosGraduacoes = []string{"CEL PM", "TC PM", "MAJ PM", "CAP PM", "1º TEN PM", "2º TEN PM", "CAD PM",
	"ST PM", "1º SGT PM", "2º SGT PM", "3º SGT PM", "CB PM", "SD PM"}

// EnquadramentoCadastro: dispositivo do catálogo e, no Estatuto, a natureza.
type EnquadramentoCadastro struct {
	DispositivoID string `json:"dispositivoId"`
	Natureza      string `json:"natureza"`
}

// FatoCadastro (PADS): o fato cometido com um ou mais dispositivos; cada
// dispositivo vira uma imputação, conferida no formulário contra o Memorando.
type FatoCadastro struct {
	ID             string                  `json:"id"`
	Resumo         string                  `json:"resumo"`
	Data           string                  `json:"data"` // DD/MM/AAAA, opcional
	Enquadramentos []EnquadramentoCadastro `json:"enquadramentos"`
}

// ArquivoCadastro é um arquivo de uma peça (o conteúdo fica na sessão).
type ArquivoCadastro struct {
	ID      string `json:"id"`
	Nome    string `json:"nome"`
	Paginas int    `json:"paginas"`
	Hash    string `json:"hash"`   // 12 primeiros dígitos do SHA-256 (conferência visual)
	Origem  string `json:"origem"` // UPLOAD ou SEI
	sha     string
	bytes   int
}

// NomeLogico na sessão cifrada.
func (a ArquivoCadastro) NomeLogico(peca string) string { return "peca-" + peca + "-" + a.ID + ".pdf" }

// PecaCadastro é uma peça processual.
type PecaCadastro struct {
	ID             string            `json:"id"`
	Tipo           string            `json:"tipo"`
	Titulo         string            `json:"titulo"`
	Descricao      string            `json:"descricao"`
	Papel          string            `json:"papel"`     // depoimento: TESTEMUNHA, DECLARANTE, SINDICADO
	PessoaIDs      []string          `json:"pessoaIds"` // depoente (1) ou pessoas ligadas ao documento
	Data           string            `json:"data"`
	SEI            string            `json:"sei"`
	NaoApresentada bool              `json:"naoApresentada"` // defesa prévia ou alegações finais
	Arquivos       []ArquivoCadastro `json:"arquivos"`
}

// Cadastro é o conjunto da tarefa.
type Cadastro struct {
	Tipo        Tipo             `json:"tipo"`
	Pessoas     []PessoaCadastro `json:"pessoas"`
	AcusadoID   string           `json:"acusadoId"`  // PADS
	Fatos       []FatoCadastro   `json:"fatos"`      // PADS
	Portaria    string           `json:"portaria"`   // SR: número/identificação
	Sindicados  []string         `json:"sindicados"` // SR
	Ofendidos   []string         `json:"ofendidos"`  // SR
	SemOfendido bool             `json:"semOfendido"`
	Pecas       []PecaCadastro   `json:"pecas"`
	Concluido   bool             `json:"concluido"`
	seq         int
}

func (c *Cadastro) novoID(prefixo string) string {
	c.seq++
	return fmt.Sprintf("%s-%02d", prefixo, c.seq)
}

func novoCadastro(tipo Tipo) *Cadastro {
	c := &Cadastro{Tipo: tipo}
	central := PecaMemorando
	if tipo == TipoSR {
		central = PecaPortaria
	}
	c.Pecas = append(c.Pecas, PecaCadastro{ID: c.novoID("PEC"), Tipo: central})
	if tipo == TipoPADS {
		c.Pecas = append(c.Pecas, PecaCadastro{ID: c.novoID("PEC"), Tipo: PecaDefesa},
			PecaCadastro{ID: c.novoID("PEC"), Tipo: PecaAlegacoes})
	}
	return c
}

func (c *Cadastro) copia() *Cadastro {
	if c == nil {
		return nil
	}
	n := *c
	n.Pessoas = append([]PessoaCadastro(nil), c.Pessoas...)
	for i := range n.Pessoas {
		n.Pessoas[i].Grafias = append([]string(nil), c.Pessoas[i].Grafias...)
	}
	n.Fatos = append([]FatoCadastro(nil), c.Fatos...)
	for i := range n.Fatos {
		n.Fatos[i].Enquadramentos = append([]EnquadramentoCadastro(nil), c.Fatos[i].Enquadramentos...)
	}
	n.Sindicados = append([]string(nil), c.Sindicados...)
	n.Ofendidos = append([]string(nil), c.Ofendidos...)
	n.Pecas = append([]PecaCadastro(nil), c.Pecas...)
	for i := range n.Pecas {
		n.Pecas[i].PessoaIDs = append([]string(nil), c.Pecas[i].PessoaIDs...)
		n.Pecas[i].Arquivos = append([]ArquivoCadastro(nil), c.Pecas[i].Arquivos...)
	}
	return &n
}

func (c *Cadastro) peca(id string) *PecaCadastro {
	for i := range c.Pecas {
		if c.Pecas[i].ID == id {
			return &c.Pecas[i]
		}
	}
	return nil
}

func (c *Cadastro) temFato(id string) bool {
	for _, f := range c.Fatos {
		if f.ID == id {
			return true
		}
	}
	return false
}

func (c *Cadastro) pessoa(id string) *PessoaCadastro {
	for i := range c.Pessoas {
		if c.Pessoas[i].ID == id {
			return &c.Pessoas[i]
		}
	}
	return nil
}

// Pessoa devolve o envolvido pelo ID (para o pipeline).
func (c *Cadastro) Pessoa(id string) (PessoaCadastro, bool) {
	if p := c.pessoa(id); p != nil {
		return *p, true
	}
	return PessoaCadastro{}, false
}

var (
	reDataCad = regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`)
)

func limpar(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		s = string(r[:max])
	}
	return s
}

// aplicar recebe os metadados editados na interface (sem os arquivos, que
// só mudam pelos comandos próprios) e valida referências e formatos.
func (c *Cadastro) aplicar(n Cadastro) error {
	var problemas []string
	// Pessoas: IDs novos ("") ganham ID; nomes obrigatórios.
	// Pessoa nova vem com ID provisório da interface (ex.: "NOVO-1"), já
	// usado nas referências (acusado, depoente…): ganha ID definitivo e as
	// referências são reescritas. ID que não existe no cadastro nunca é
	// aceito como definitivo.
	pessoas := make([]PessoaCadastro, 0, len(n.Pessoas))
	ids := map[string]bool{}
	novoDe := map[string]string{}
	for _, p := range n.Pessoas {
		p.Nome, p.PostoGrad = limpar(p.Nome, 160), limpar(p.PostoGrad, 60)
		if p.Nome == "" {
			problemas = append(problemas, "pessoa sem nome")
			continue
		}
		if p.Instituicao {
			p.PostoGrad, p.Crianca, p.Grafias, p.Civil = "", false, nil, false
		}
		if p.Civil {
			p.PostoGrad = ""
		}
		if c.pessoa(p.ID) == nil || ids[p.ID] {
			provisorio := p.ID
			p.ID = c.novoID("PES")
			if provisorio != "" {
				novoDe[provisorio] = p.ID
			}
		}
		var gs []string
		for _, g := range p.Grafias {
			if g = limpar(g, 160); g != "" {
				gs = append(gs, g)
			}
		}
		p.Grafias = gs
		ids[p.ID] = true
		pessoas = append(pessoas, p)
	}
	resolver := func(id string) string {
		if n, ok := novoDe[id]; ok {
			return n
		}
		return id
	}
	instituicao, civil := map[string]bool{}, map[string]bool{}
	for _, p := range pessoas {
		instituicao[p.ID], civil[p.ID] = p.Instituicao, p.Civil
	}
	conferirPessoa := func(id, onde string) bool {
		if id != "" && !ids[id] {
			problemas = append(problemas, onde+": pessoa inexistente")
			return false
		}
		// Instituição só pode ser ofendida (V4 §23.3, c).
		if id != "" && instituicao[id] && onde != "ofendido" {
			problemas = append(problemas, onde+": uma instituição só pode ser ofendida")
			return false
		}
		// Sindicado e acusado são sempre militares.
		if id != "" && civil[id] && (onde == "sindicado" || onde == "acusado") {
			problemas = append(problemas, onde+": deve ser militar (com posto ou graduação)")
			return false
		}
		return id != ""
	}
	n.AcusadoID = resolver(n.AcusadoID)
	if n.AcusadoID != "" {
		conferirPessoa(n.AcusadoID, "acusado")
	}
	var sind, ofend []string
	for _, s := range n.Sindicados {
		if s = resolver(s); conferirPessoa(s, "sindicado") {
			sind = append(sind, s)
		}
	}
	for _, o := range n.Ofendidos {
		if o = resolver(o); conferirPessoa(o, "ofendido") {
			ofend = append(ofend, o)
		}
	}
	// Fatos e enquadramentos (PADS).
	var fatos []FatoCadastro
	for _, f := range n.Fatos {
		f.Resumo, f.Data = limpar(f.Resumo, 2000), strings.TrimSpace(f.Data)
		if f.Data != "" && !reDataCad.MatchString(f.Data) {
			problemas = append(problemas, "fato: data no formato DD/MM/AAAA")
		}
		if !c.temFato(f.ID) {
			f.ID = c.novoID("FATO")
		}
		var enqs []EnquadramentoCadastro
		for _, e := range f.Enquadramentos {
			d, ok := enquadramento.PorID(e.DispositivoID)
			if !ok {
				problemas = append(problemas, "fato: dispositivo fora do catálogo")
				continue
			}
			if d.Diploma == enquadramento.RDPM {
				e.Natureza = "" // a do artigo
			} else if e.Natureza != "" && e.Natureza != "LEVE" && e.Natureza != "MEDIA" && e.Natureza != "GRAVE" {
				problemas = append(problemas, "fato: natureza inválida")
				continue
			}
			enqs = append(enqs, e)
		}
		f.Enquadramentos = enqs
		fatos = append(fatos, f)
	}
	// Peças: a ordem é a da interface; arquivos preservados do servidor.
	var pecas []PecaCadastro
	vistas := map[string]bool{}
	for _, p := range n.Pecas {
		atual := c.peca(p.ID)
		switch p.Tipo {
		case PecaMemorando, PecaPortaria, PecaDefesa, PecaAlegacoes, PecaDepoimento, PecaOutro:
		default:
			problemas = append(problemas, "peça de tipo desconhecido")
			continue
		}
		if (p.Tipo == PecaMemorando || p.Tipo == PecaDefesa || p.Tipo == PecaAlegacoes) && c.Tipo != TipoPADS ||
			p.Tipo == PecaPortaria && c.Tipo != TipoSR {
			problemas = append(problemas, "peça incompatível com a modalidade")
			continue
		}
		if atual == nil || vistas[p.ID] {
			p.ID, p.Arquivos = c.novoID("PEC"), nil
		} else {
			p.Arquivos = atual.Arquivos
		}
		p.Titulo, p.Descricao, p.SEI, p.Data = limpar(p.Titulo, 200), limpar(p.Descricao, 1000), limpar(p.SEI, 40), strings.TrimSpace(p.Data)
		if p.Data != "" && !reDataCad.MatchString(p.Data) {
			problemas = append(problemas, "peça: data no formato DD/MM/AAAA")
		}
		var pids []string
		for _, id := range p.PessoaIDs {
			if id = resolver(id); conferirPessoa(id, "peça") {
				pids = append(pids, id)
			}
		}
		p.PessoaIDs = pids
		if p.Tipo == PecaDepoimento {
			switch p.Papel {
			case PapelTestemunha, PapelDeclarante:
			case PapelSindicado:
				if c.Tipo != TipoSR {
					problemas = append(problemas, "depoimento de sindicado só na Sindicância")
				}
			default:
				problemas = append(problemas, "depoimento sem papel do depoente")
			}
		} else {
			p.Papel = ""
		}
		if p.Tipo != PecaDefesa && p.Tipo != PecaAlegacoes {
			p.NaoApresentada = false
		}
		if p.NaoApresentada {
			p.Arquivos = nil
		}
		vistas[p.ID] = true
		pecas = append(pecas, p)
	}
	if len(problemas) > 0 {
		return erros.Com(erros.TarefaRespostaInvalid, errors.New(strings.Join(unicos(problemas), "; ")))
	}
	c.Pessoas, c.AcusadoID, c.Fatos, c.Sindicados, c.Ofendidos = pessoas, n.AcusadoID, fatos, sind, ofend
	c.Portaria, c.SemOfendido = limpar(n.Portaria, 120), n.SemOfendido && len(ofend) == 0
	c.Pecas = pecas
	return nil
}

func unicos(xs []string) []string {
	var out []string
	visto := map[string]bool{}
	for _, x := range xs {
		if !visto[x] {
			visto[x] = true
			out = append(out, x)
		}
	}
	return out
}

// Pendencias: o que falta para concluir (vazio = pode concluir).
func (c *Cadastro) Pendencias() []string {
	var out []string
	conta := map[string]int{}
	for _, p := range c.Pecas {
		conta[p.Tipo]++
		semArquivo := len(p.Arquivos) == 0 && !p.NaoApresentada
		switch p.Tipo {
		case PecaMemorando:
			if semArquivo {
				out = append(out, "Anexe o Memorando Disciplinar.")
			}
		case PecaPortaria:
			if semArquivo {
				out = append(out, "Anexe a Portaria de instauração.")
			}
		case PecaDefesa:
			if semArquivo {
				out = append(out, "Defesa prévia: anexe o documento ou marque que não foi apresentada.")
			}
		case PecaAlegacoes:
			if semArquivo {
				out = append(out, "Alegações finais: anexe o documento ou marque que não foram apresentadas.")
			}
		case PecaDepoimento:
			if len(p.PessoaIDs) == 0 {
				out = append(out, "Informe o depoente de cada depoimento.")
			}
			if semArquivo {
				out = append(out, "Anexe o termo de cada depoimento.")
			}
		case PecaOutro:
			if p.Titulo == "" {
				out = append(out, "Dê um nome a cada documento adicional.")
			}
			if semArquivo {
				out = append(out, "Anexe o arquivo de cada documento adicional.")
			}
		}
	}
	switch c.Tipo {
	case TipoPADS:
		if c.AcusadoID == "" {
			out = append(out, "Informe o acusado.")
		}
		if conta[PecaMemorando] != 1 || conta[PecaDefesa] != 1 || conta[PecaAlegacoes] != 1 {
			out = append(out, "O PADS tem um Memorando, uma defesa prévia e umas alegações finais.")
		}
	case TipoSR:
		if len(c.Sindicados) == 0 {
			out = append(out, "Informe ao menos um sindicado.")
		}
		if len(c.Ofendidos) == 0 && !c.SemOfendido {
			out = append(out, "Informe os ofendidos ou marque que não há ofendido individualizado.")
		}
		if conta[PecaPortaria] != 1 {
			out = append(out, "A Sindicância tem uma Portaria de instauração.")
		}
	}
	return unicos(out)
}

// ArquivoEntrada é um arquivo lido pelo Go (diálogo nativo ou SEI).
type ArquivoEntrada struct {
	Nome   string
	Dados  []byte
	Origem string
}

func (c *Cadastro) totais() (arquivos, bytes int) {
	for _, p := range c.Pecas {
		for _, a := range p.Arquivos {
			arquivos++
			bytes += a.bytes
		}
	}
	return
}

// anexar valida e registra os arquivos de uma peça; grava o conteúdo pela
// função dada (sessão cifrada). Arquivo repetido (mesmo SHA-256) é recusado.
func (c *Cadastro) anexar(pecaID string, as []ArquivoEntrada, contar func([]byte) (int, error), gravar func(nome string, dados []byte) error) error {
	p := c.peca(pecaID)
	if p == nil {
		return erros.Novo(erros.TarefaOrdemInvalida)
	}
	if p.NaoApresentada {
		return erros.Com(erros.TarefaRespostaInvalid, errors.New("peça marcada como não apresentada"))
	}
	nArq, nBytes := c.totais()
	for _, a := range as {
		if nArq+1 > MaxArquivosCadastro || nBytes+len(a.Dados) > MaxBytesCadastro {
			return erros.Novo(erros.CadastroLimite)
		}
		if err := conferirPDF(a.Dados); err != nil {
			return err
		}
		h := sha256.Sum256(a.Dados)
		sha := hex.EncodeToString(h[:])
		for _, q := range c.Pecas {
			for _, x := range q.Arquivos {
				if x.sha == sha {
					return erros.Com(erros.ArquivoRepetido, fmt.Errorf("já anexado em: %s", rotuloPeca(q)))
				}
			}
		}
		n, err := contar(a.Dados)
		if err != nil {
			return err
		}
		arq := ArquivoCadastro{ID: c.novoID("ARQ"), Nome: limpar(a.Nome, 160), Paginas: n, Hash: sha[:12], Origem: a.Origem, sha: sha, bytes: len(a.Dados)}
		if arq.Origem == "" {
			arq.Origem = "UPLOAD"
		}
		if err := gravar(arq.NomeLogico(p.ID), a.Dados); err != nil {
			return err
		}
		p.Arquivos = append(p.Arquivos, arq)
		nArq, nBytes = nArq+1, nBytes+len(a.Dados)
	}
	return nil
}

func rotuloPeca(p PecaCadastro) string {
	nomes := map[string]string{PecaMemorando: "Memorando Disciplinar", PecaPortaria: "Portaria", PecaDefesa: "defesa prévia",
		PecaAlegacoes: "alegações finais", PecaDepoimento: "depoimento", PecaOutro: "documento"}
	if p.Titulo != "" {
		return nomes[p.Tipo] + ": " + p.Titulo
	}
	return nomes[p.Tipo]
}

func (c *Cadastro) remover(pecaID, arqID string, apagar func(nome string)) error {
	p := c.peca(pecaID)
	if p == nil {
		return erros.Novo(erros.TarefaOrdemInvalida)
	}
	for i, a := range p.Arquivos {
		if a.ID == arqID {
			apagar(a.NomeLogico(p.ID))
			p.Arquivos = append(p.Arquivos[:i], p.Arquivos[i+1:]...)
			return nil
		}
	}
	return erros.Novo(erros.TarefaOrdemInvalida)
}

func (c *Cadastro) mover(pecaID, arqID string, delta int) error {
	p := c.peca(pecaID)
	if p == nil {
		return erros.Novo(erros.TarefaOrdemInvalida)
	}
	for i, a := range p.Arquivos {
		if a.ID == arqID {
			j := i + delta
			if j < 0 || j >= len(p.Arquivos) {
				return nil
			}
			p.Arquivos[i], p.Arquivos[j] = p.Arquivos[j], p.Arquivos[i]
			return nil
		}
	}
	return erros.Novo(erros.TarefaOrdemInvalida)
}

// ---- comandos do gerenciador ----

func (g *Gerenciador) emCadastro() error {
	if g.t == nil {
		return erros.Novo(erros.TarefaSemTarefa)
	}
	if g.t.estado != RecebendoUpload || g.t.cadastro == nil {
		return erros.Novo(erros.TarefaOrdemInvalida)
	}
	return nil
}

// SalvarCadastro aplica os metadados editados (pessoas, fatos, peças e a
// ordem das peças). Arquivos de peças removidas saem da sessão.
func (g *Gerenciador) SalvarCadastro(n Cadastro) (*Visao, error) {
	g.mu.Lock()
	if err := g.emCadastro(); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	c := g.t.cadastro
	antes := c.copia()
	if err := c.aplicar(n); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	for _, p := range antes.Pecas {
		atual := c.peca(p.ID)
		for _, a := range p.Arquivos {
			if atual == nil || !temArquivo(atual, a.ID) {
				_ = g.t.sessao.ApagarLogico(a.NomeLogico(p.ID))
			}
		}
	}
	return g.emitirUnlock()
}

func temArquivo(p *PecaCadastro, id string) bool {
	for _, a := range p.Arquivos {
		if a.ID == id {
			return true
		}
	}
	return false
}

// AnexarArquivos acrescenta arquivos a uma peça (lidos pelo Go; a interface
// nunca vê caminhos nem conteúdo).
func (g *Gerenciador) AnexarArquivos(pecaID string, as []ArquivoEntrada) (*Visao, error) {
	g.mu.Lock()
	if err := g.emCadastro(); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	contar := g.contarPaginas
	if contar == nil {
		contar = func([]byte) (int, error) { return 0, nil }
	}
	if err := g.t.cadastro.anexar(pecaID, as, contar, g.t.sessao.Gravar); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	return g.emitirUnlock()
}

// RemoverArquivo tira um arquivo da peça e da sessão.
func (g *Gerenciador) RemoverArquivo(pecaID, arqID string) (*Visao, error) {
	g.mu.Lock()
	if err := g.emCadastro(); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	if err := g.t.cadastro.remover(pecaID, arqID, func(n string) { _ = g.t.sessao.ApagarLogico(n) }); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	return g.emitirUnlock()
}

// MoverArquivo muda a ordem do arquivo dentro da peça (delta -1 ou +1).
func (g *Gerenciador) MoverArquivo(pecaID, arqID string, delta int) (*Visao, error) {
	g.mu.Lock()
	if err := g.emCadastro(); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	if delta != -1 && delta != 1 {
		g.mu.Unlock()
		return nil, erros.Novo(erros.TarefaOrdemInvalida)
	}
	if err := g.t.cadastro.mover(pecaID, arqID, delta); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	return g.emitirUnlock()
}

// ConcluirCadastro confere a completude e inicia o processamento. A análise
// anterior (se o usuário voltou ao cadastro) é descartada, como com autos
// novos.
func (g *Gerenciador) ConcluirCadastro() (*Visao, error) {
	g.mu.Lock()
	if err := g.emCadastro(); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	t := g.t
	if ps := t.cadastro.Pendencias(); len(ps) > 0 {
		g.mu.Unlock()
		return nil, erros.Com(erros.TarefaRespostaInvalid, errors.New(strings.Join(ps, " ")))
	}
	t.cadastro.Concluido = true
	t.autos, t.pergunta, t.erro, t.parada, t.painel, t.atividade = nil, nil, nil, false, nil, ""
	t.respostas, t.anteriores, t.respondidas = map[string]Resposta{}, map[string]Resposta{}, nil
	liberarDados(t)
	if err := g.mover(Verificando); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	g.iniciarProcessamentoLocked()
	return g.emitirUnlock()
}

// Cadastro devolve o cadastro concluído da tarefa (cópia), para o pipeline;
// nil quando a entrada foi o PDF único (SEI).
func (c *Contexto) Cadastro() *Cadastro {
	c.g.mu.Lock()
	defer c.g.mu.Unlock()
	if c.t.cadastro == nil || !c.t.cadastro.Concluido {
		return nil
	}
	return c.t.cadastro.copia()
}

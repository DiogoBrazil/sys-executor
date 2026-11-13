package legal

import (
	"database/sql"
	_ "embed"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"
	_ "modernc.org/sqlite" // SQLite puro Go (sem CGO), com FTS5

	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

// Passagem é um trecho recuperado da biblioteca (LegalPassage do plano §10),
// com o documento e a versão de onde saiu. Só uma Passagem pode sustentar
// citação de norma. Ela não prova vigência nem aplicabilidade (V4 §40.3).
type Passagem struct {
	DocumentoID string `json:"documento_id"`
	Documento   string `json:"documento"`
	Versao      string `json:"versao"`
	Categoria   string `json:"categoria"`
	Arquivo     string `json:"arquivo"`
	// Parte: norma interna, quando o documento reúne várias.
	Parte string `json:"parte,omitempty"`
	// Referencia por extenso: "art. 39, inciso VII, alínea a, do RDPM".
	Referencia string `json:"referencia"`
	Artigo     string `json:"artigo,omitempty"`
	Paragrafo  string `json:"paragrafo,omitempty"`
	Inciso     string `json:"inciso,omitempty"`
	Alinea     string `json:"alinea,omitempty"`
	// Texto do dispositivo pedido. Para artigo inteiro, o caput com os
	// parágrafos, incisos e alíneas em vigor.
	Texto string `json:"texto"`
	// Caput do artigo (ou do parágrafo) a que o inciso ou a alínea pertence.
	Caput    string `json:"caput,omitempty"`
	Contexto string `json:"contexto,omitempty"`
	// Pagina no original; 0 quando o original é Markdown (sem páginas).
	Pagina int `json:"pagina"`
	// Rubrica da norma acima do dispositivo ("Peculato"), se houver.
	Rubrica string `json:"rubrica,omitempty"`
	// OCR: texto reconhecido de imagem; conferir no original antes de citar.
	OCR bool `json:"ocr,omitempty"`
	// Exata: veio de referência exata (e não da busca por palavras).
	Exata bool `json:"exata"`
}

// Biblioteca é o acervo aberto para consulta: referência exata e busca por
// palavras (SQLite FTS5 em memória). É somente leitura e pode ser usada por
// várias tarefas ao mesmo tempo.
type Biblioteca struct {
	acervo *Acervo
	notas  []NotaVigencia
	siglas map[string]string // sigla normalizada → id do documento
	docs   map[string]*Documento
	// artigos indexados na busca, na ordem do rowid (a partir de 1).
	blocos   []bloco
	tesauro  map[string][]string // radical → radicais equivalentes
	mu       sync.Mutex
	db       *sql.DB
	fechada  bool
	curtaDoc map[string]string // id → nome curto para a referência
}

// bloco é a unidade da busca por palavras: um artigo (com tudo o que está
// em vigor abaixo dele) ou um trecho de página.
type bloco struct {
	doc      *Documento
	parte    int
	artigo   string
	unidades []Unidade // em vigor
}

//go:embed dados/tesauro.yaml
var tesauroYAML []byte

var (
	padraoOnce sync.Once
	padrao     *Biblioteca
	padraoErr  error
)

// Padrao devolve a biblioteca do acervo inicial embutido (aberta uma vez).
func Padrao() (*Biblioteca, error) {
	padraoOnce.Do(func() {
		a, err := AcervoInicial()
		if err != nil {
			padraoErr = err
			return
		}
		padrao, padraoErr = Abrir(a)
	})
	return padrao, padraoErr
}

// Abrir indexa o acervo.
func Abrir(a *Acervo) (*Biblioteca, error) {
	notas, err := NotasVigencia()
	if err != nil {
		return nil, err
	}
	b := &Biblioteca{acervo: a, notas: notas, siglas: map[string]string{}, docs: map[string]*Documento{}, curtaDoc: map[string]string{}}
	if err := b.lerTesauro(); err != nil {
		return nil, err
	}
	for i := range a.Documentos {
		d := &a.Documentos[i]
		b.docs[d.ID] = d
		for k, s := range d.Siglas {
			b.siglas[strings.Join(strings.Fields(texto.SemAcentos(s)), " ")] = d.ID
			if k == 0 {
				b.curtaDoc[d.ID] = s
			}
		}
		if b.curtaDoc[d.ID] == "" {
			b.curtaDoc[d.ID] = d.Titulo
		}
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // ":memory:" é por conexão
	b.db = db
	if _, err := db.Exec(`CREATE VIRTUAL TABLE blocos USING fts5(titulo, texto, doc UNINDEXED, tokenize = 'unicode61 remove_diacritics 2')`); err != nil {
		db.Close()
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		db.Close()
		return nil, err
	}
	st, err := tx.Prepare(`INSERT INTO blocos(rowid, titulo, texto, doc) VALUES (?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		db.Close()
		return nil, err
	}
	for i := range a.Documentos {
		d := &a.Documentos[i]
		for pi, p := range d.Partes {
			for _, bl := range b.blocosDaParte(d, pi, p) {
				b.blocos = append(b.blocos, bl)
				titulo := d.Titulo + " " + p.Titulo + " " + bl.unidades[0].Contexto
				for _, u := range bl.unidades { // rubricas: "Peculato", "Furto qualificado"
					if u.Rubrica != "" {
						titulo += " " + u.Rubrica
					}
				}
				if _, err := st.Exec(len(b.blocos), titulo, textoDoBloco(bl.unidades), d.ID); err != nil {
					tx.Rollback()
					db.Close()
					return nil, err
				}
			}
		}
	}
	st.Close()
	if err := tx.Commit(); err != nil {
		db.Close()
		return nil, err
	}
	return b, nil
}

// Fechar libera o índice.
func (b *Biblioteca) Fechar() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.fechada {
		return nil
	}
	b.fechada = true
	return b.db.Close()
}

// Documentos lista os originais do acervo (sem o texto).
func (b *Biblioteca) Documentos() []Documento {
	out := make([]Documento, len(b.acervo.Documentos))
	for i, d := range b.acervo.Documentos {
		d.Partes = nil
		out[i] = d
	}
	return out
}

// emVigor: a unidade pode ser oferecida à redação. Ficam de fora a redação
// histórica, o que o próprio texto diz estar revogado e o que tem nota de
// vigência curada (D-01).
func (b *Biblioteca) emVigor(doc string, u Unidade) bool {
	if u.Historica || u.Revogada {
		return false
	}
	for _, n := range b.notas {
		if n.alcanca(doc, u) {
			return false
		}
	}
	return true
}

// Nota devolve a nota de vigência que alcança o dispositivo, se houver.
func (b *Biblioteca) Nota(r Referencia) (NotaVigencia, bool) {
	u := Unidade{Artigo: r.Artigo, Paragrafo: r.Paragrafo, Inciso: r.Inciso, Alinea: r.Alinea}
	for _, n := range b.notas {
		if n.alcanca(r.Documento, u) {
			return n, true
		}
	}
	return NotaVigencia{}, false
}

func (b *Biblioteca) blocosDaParte(d *Documento, pi int, p Parte) []bloco {
	var out []bloco
	for _, u := range p.Unidades {
		if !b.emVigor(d.ID, u) {
			continue
		}
		if u.Artigo != "" && len(out) > 0 && out[len(out)-1].artigo == u.Artigo {
			out[len(out)-1].unidades = append(out[len(out)-1].unidades, u)
			continue
		}
		out = append(out, bloco{doc: d, parte: pi, artigo: u.Artigo, unidades: []Unidade{u}})
	}
	return out
}

// rotuloUnidade: "§ 1º", "I –", "a)" antes do texto, como na norma.
func rotuloUnidade(u Unidade) string {
	switch {
	case u.Alinea != "":
		return u.Alinea + ") "
	case u.Inciso != "":
		return u.Inciso + " – "
	case u.Paragrafo == "unico":
		return "Parágrafo único. "
	case u.Paragrafo != "":
		return rotuloParagrafo(u.Paragrafo) + " "
	}
	return ""
}

func textoDoBloco(us []Unidade) string {
	var sb strings.Builder
	for i, u := range us {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(rotuloUnidade(u))
		sb.WriteString(u.Texto)
	}
	return sb.String()
}

func (b *Biblioteca) passagem(d *Documento, pi int, r Referencia, us []Unidade, exata bool) Passagem {
	p := Passagem{
		DocumentoID: d.ID, Documento: d.Titulo, Versao: d.Versao(), Categoria: d.Categoria, Arquivo: d.Arquivo,
		Artigo: r.Artigo, Paragrafo: r.Paragrafo, Inciso: r.Inciso, Alinea: r.Alinea,
		Texto: textoDoBloco(us), Contexto: us[0].Contexto, Pagina: us[0].Pagina, Rubrica: us[0].Rubrica, OCR: d.OCR, Exata: exata,
	}
	if len(us) == 1 {
		p.Texto = us[0].Texto
	}
	nome := b.curtaDoc[d.ID]
	if d.Composto || pi != d.principal() {
		p.Parte = d.Partes[pi].Titulo
		if p.Parte != "" {
			nome = p.Parte + " (em " + nome + ")"
		}
	}
	if r.Artigo == "" {
		p.Referencia = fmt.Sprintf("%s, pág. %d", nome, p.Pagina)
	} else { // original em Markdown: sem página, a localização é o dispositivo
		p.Referencia = r.Texto() + ", " + nome
	}
	return p
}

// Dispositivo recupera o dispositivo citado, na redação em vigor. Para
// artigo inteiro (sem parágrafo, inciso ou alínea), devolve o artigo com
// tudo o que está em vigor abaixo dele. Devolve falso quando o documento
// não existe, o dispositivo não foi encontrado ou não está em vigor.
func (b *Biblioteca) Dispositivo(r Referencia) (Passagem, bool) {
	d := b.docs[r.Documento]
	if d == nil || len(d.Partes) == 0 || d.Composto || r.Artigo == "" {
		return Passagem{}, false
	}
	pi := d.principal()
	alvo := Unidade{Artigo: r.Artigo, Paragrafo: r.Paragrafo, Inciso: r.Inciso, Alinea: r.Alinea}
	inteiro := r.Paragrafo == "" && r.Inciso == "" && r.Alinea == ""
	var us []Unidade
	var caput string
	for _, u := range d.Partes[pi].Unidades {
		if u.Artigo != r.Artigo || !b.emVigor(d.ID, u) {
			continue
		}
		switch {
		case inteiro, dentroDe(u, alvo):
			us = append(us, u)
		case caput == "" && u.Inciso == "" && u.Alinea == "" && u.Paragrafo == r.Paragrafo:
			caput = u.Texto // caput do artigo ou do parágrafo que contém o inciso
		}
	}
	if len(us) == 0 || (!inteiro && us[0].chave() != alvo.chave()) {
		return Passagem{}, false
	}
	p := b.passagem(d, pi, r, us, true)
	if r.Inciso != "" {
		p.Caput = caput
	}
	return p, true
}

// ---------- busca por palavras ----------

func (b *Biblioteca) lerTesauro() error {
	var arq struct {
		Grupos [][]string `yaml:"grupos"`
	}
	if err := yaml.Unmarshal(tesauroYAML, &arq); err != nil {
		return fmt.Errorf("tesauro da biblioteca: %w", err)
	}
	b.tesauro = map[string][]string{}
	for _, g := range arq.Grupos {
		var rs []string
		for _, t := range g {
			for _, w := range texto.Termos(t) {
				rs = append(rs, radical(w))
			}
		}
		for _, r := range rs {
			b.tesauro[r] = rs
		}
	}
	return nil
}

var palavrasVazias = map[string]bool{
	"a": true, "o": true, "as": true, "os": true, "de": true, "da": true, "do": true, "das": true, "dos": true,
	"e": true, "ou": true, "em": true, "no": true, "na": true, "nos": true, "nas": true, "um": true, "uma": true,
	"que": true, "se": true, "por": true, "para": true, "com": true, "sem": true, "ao": true, "aos": true,
	"qual": true, "quais": true, "como": true, "quando": true, "e'": true, "ser": true, "sao": true, "sobre": true,
	"art": true, "artigo": true, "inciso": true, "alinea": true, "paragrafo": true,
}

// radical corta plural e terminação, para "transgressão" achar
// "transgressões" (o FTS5 não reduz palavras do português). A busca usa o
// radical como prefixo.
func radical(t string) string {
	if len(t) < 6 {
		return t
	}
	for _, suf := range []string{"coes", "cao", "oes", "aes", "ais", "eis", "ao", "es", "as", "os", "s", "a", "o", "e"} {
		if strings.HasSuffix(t, suf) && len(t)-len(suf) >= 4 {
			r := t[:len(t)-len(suf)]
			if suf == "coes" || suf == "cao" {
				r += "c"
			}
			return r
		}
	}
	return t
}

// gruposDaConsulta: para cada palavra útil, os radicais equivalentes.
func (b *Biblioteca) gruposDaConsulta(consulta string) [][]string {
	var gs [][]string
	visto := map[string]bool{}
	for _, t := range texto.Termos(consulta) {
		if palavrasVazias[t] || (len(t) < 3 && !soDigitos(t)) {
			continue
		}
		r := radical(t)
		if visto[r] {
			continue
		}
		visto[r] = true
		if eq := b.tesauro[r]; len(eq) > 0 {
			gs = append(gs, eq)
		} else {
			gs = append(gs, []string{r})
		}
	}
	return gs
}

func soDigitos(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

func consultaFTS(gs [][]string, juncao string) string {
	ps := make([]string, len(gs))
	for i, g := range gs {
		alt := make([]string, len(g))
		for k, r := range g {
			alt[k] = `"` + strings.ReplaceAll(r, `"`, ``) + `"*`
		}
		ps[i] = "(" + strings.Join(alt, " OR ") + ")"
	}
	return strings.Join(ps, juncao)
}

// Buscar faz a recuperação da biblioteca (plano §10, sem a parte semântica:
// V4 §47.7, busca somente lexical): primeiro a referência exata, se a
// consulta citar artigo e documento; depois a busca por palavras (BM25),
// com o tesauro. Os resultados trazem documento e versão.
func (b *Biblioteca) Buscar(consulta string, limite int) ([]Passagem, error) {
	return b.BuscarEm(consulta, limite)
}

// BuscarEm é a Buscar restrita a alguns documentos (ids do manifesto): por
// exemplo, só a legislação penal, ou só o RDPM e o Estatuto. Sem ids, busca
// em todo o acervo.
func (b *Biblioteca) BuscarEm(consulta string, limite int, docs ...string) ([]Passagem, error) {
	if limite <= 0 {
		limite = 8
	}
	permitido := func(id string) bool { return len(docs) == 0 || slices.Contains(docs, id) }
	var out []Passagem
	exato := ""
	if r, ok := b.InterpretarReferencia(consulta); ok && r.Documento != "" && permitido(r.Documento) {
		if p, ok := b.Dispositivo(r); ok {
			out = append(out, p)
			exato = r.Documento + "|" + r.Artigo
		}
	}
	gs := b.gruposDaConsulta(consulta)
	if len(gs) == 0 {
		return out, nil
	}
	ids, err := b.consultar(consultaFTS(gs, " AND "), limite, docs)
	if err != nil {
		return nil, err
	}
	if len(ids) < limite && len(gs) > 1 { // completa com resultados parciais
		mais, err := b.consultar(consultaFTS(gs, " OR "), limite*2, docs)
		if err != nil {
			return nil, err
		}
		ids = append(ids, mais...)
	}
	visto := map[int]bool{}
	// O mesmo artigo com o mesmo texto em dois documentos (a norma avulsa e
	// a cópia dentro de uma coletânea) aparece uma vez só: a de melhor posição.
	repetido := map[string]int{} // assinatura → posição em out
	for i, p := range out {
		repetido[assinatura(p.Artigo, p.Texto)] = i
	}
	for _, id := range ids {
		if visto[id] || id < 1 || id > len(b.blocos) {
			continue
		}
		visto[id] = true
		bl := b.blocos[id-1]
		if exato != "" && bl.doc.ID+"|"+bl.artigo == exato && bl.parte == bl.doc.principal() {
			continue // já veio pela referência exata
		}
		p := b.passagem(bl.doc, bl.parte, Referencia{Documento: bl.doc.ID, Artigo: bl.artigo}, bl.unidades, false)
		a := assinatura(p.Artigo, p.Texto)
		if k, ok := repetido[a]; ok && bl.artigo != "" {
			// Entre a norma avulsa e a cópia na coletânea, fica a avulsa.
			if b.docs[out[k].DocumentoID].Composto && !bl.doc.Composto {
				out[k] = p
			}
			continue
		}
		if len(out) >= limite {
			continue
		}
		repetido[a] = len(out)
		out = append(out, p)
	}
	return out, nil
}

func assinatura(artigo, t string) string {
	ts := texto.Termos(t)
	if len(ts) > 25 {
		ts = ts[:25]
	}
	return artigo + "|" + strings.Join(ts, " ")
}

func (b *Biblioteca) consultar(q string, limite int, docs []string) ([]int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.fechada {
		return nil, fmt.Errorf("biblioteca fechada")
	}
	// O texto pesa mais que o título (documento, parte e capítulo).
	sqlDocs, args := "", []any{q}
	if len(docs) > 0 {
		sqlDocs = " AND doc IN (?" + strings.Repeat(", ?", len(docs)-1) + ")"
		for _, d := range docs {
			args = append(args, d)
		}
	}
	args = append(args, limite)
	rows, err := b.db.Query(`SELECT rowid FROM blocos WHERE blocos MATCH ?`+sqlDocs+` ORDER BY bm25(blocos, 1.0, 3.0, 0.0), rowid LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("busca na biblioteca: %w", err)
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		ids = append(ids, n)
	}
	return ids, rows.Err()
}

// Estatistica resume o acervo aberto (para a interface e os testes).
type Estatistica struct {
	Documentos   int `json:"documentos"`
	Dispositivos int `json:"dispositivos"`
	SemTexto     int `json:"sem_texto"`
}

// Resumo conta documentos e dispositivos em vigor.
func (b *Biblioteca) Resumo() Estatistica {
	e := Estatistica{Documentos: len(b.acervo.Documentos)}
	for _, d := range b.acervo.Documentos {
		if d.SemTexto {
			e.SemTexto++
		}
	}
	for _, bl := range b.blocos {
		e.Dispositivos += len(bl.unidades)
	}
	return e
}

// SiglasConhecidas lista as siglas aceitas nas referências (ordenadas).
func (b *Biblioteca) SiglasConhecidas() []string {
	out := make([]string, 0, len(b.siglas))
	for s := range b.siglas {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

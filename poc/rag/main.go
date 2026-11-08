// PoC F0-07 — Biblioteca jurídica real em SQLite (modernc, sem CGO) com FTS5 + embeddings
// Qwen3-Embedding via llama-server, busca vetorial exata em Go e fusão RRF.
//
// Uso:
//
//	go run . index -lib <docs/law_library> -db <lib.db> -server <llama-server> -model <emb.gguf>
//	go run . eval  -db <lib.db> -server <llama-server> -model <emb.gguf> -queries consultas.json
package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	"golang.org/x/text/unicode/norm"
	_ "modernc.org/sqlite"
)

var (
	lib     = flag.String("lib", "", "pasta com os PDFs")
	dbPath  = flag.String("db", "lib.db", "SQLite")
	server  = flag.String("server", "", "llama-server")
	model   = flag.String("model", "", "GGUF de embeddings")
	queries = flag.String("queries", "consultas.json", "consultas de avaliação")
	ctxEmb  = flag.String("ctx", "2048", "contexto/lote do servidor de embeddings")
	limite  = flag.Int("limite", 0, "embeddings só dos N primeiros chunks (benchmark)")
	lote    = flag.Int("lote", 8, "textos por requisição de embeddings")
)

// Qwen3-Embedding: instrução em inglês rendeu mais (F0-07: hit@5 vetorial 10→14/20). INSTR=pt para comparar.
var instrucao = func() string {
	if os.Getenv("INSTR") == "pt" {
		return "Instruct: Dada uma pergunta sobre direito disciplinar militar, recupere o dispositivo normativo que a responde\nQuery: "
	}
	return "Instruct: Given a question about Brazilian military police disciplinary law, retrieve the legal provision that answers it\nQuery: "
}()

func main() {
	if len(os.Args) < 2 {
		fmt.Println("uso: index|eval [flags]")
		os.Exit(2)
	}
	cmd := os.Args[1]
	flag.CommandLine.Parse(os.Args[2:])
	db, err := sql.Open("sqlite", *dbPath)
	must(err)
	defer db.Close()
	switch cmd {
	case "index":
		index(db)
	case "eval":
		eval(db)
	}
}

// ---------- indexação ----------

var reArt = regexp.MustCompile(`(?m)^\s*Art\.?\s*(\d+)\s*[º°o]?[\s.\-–]`)

// Cabeçalhos estruturais (limite de chunk e contexto hierárquico).
var reTit = regexp.MustCompile(`(?m)^\s*(CAP[ÍI]TULO|T[ÍI]TULO|SE[ÇC][ÃA]O|Cap[íi]tulo|Se[çc][ãa]o)\s+[IVXLC]+\b.*$`)

var contexto = flag.Bool("contexto", false, "prefixar 'documento — capítulo — artigo' no texto embutido")

type chunk struct {
	doc, artigo, texto string
	pagina             int
	ctx                string // cabeçalho hierárquico (capítulo/seção) vigente
}

func extrair(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pool, err := webassembly.Init(webassembly.Config{MinIdle: 1, MaxIdle: 1, MaxTotal: 1})
	if err != nil {
		return nil, err
	}
	defer pool.Close()
	in, err := pool.GetInstance(30 * time.Second)
	if err != nil {
		return nil, err
	}
	d, err := in.OpenDocument(&requests.OpenDocument{File: &data})
	if err != nil {
		return nil, err
	}
	pc, _ := in.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: d.Document})
	var pages []string
	for i := 0; i < pc.PageCount; i++ {
		t, err := in.GetPageText(&requests.GetPageText{Page: requests.Page{ByIndex: &requests.PageByIndex{Document: d.Document, Index: i}}})
		if err != nil {
			return nil, err
		}
		pages = append(pages, norm.NFC.String(t.Text))
	}
	return pages, nil
}

// Chunk por artigo; artigos longos ou texto fora de artigo em janelas de ~1200 caracteres.
func chunkar(doc string, pages []string) []chunk {
	var full strings.Builder
	var offs []int // offset inicial de cada página
	for _, p := range pages {
		offs = append(offs, full.Len())
		full.WriteString(strings.ReplaceAll(p, "\r", ""))
		full.WriteString("\n")
	}
	s := full.String()
	pag := func(off int) int {
		i := sort.SearchInts(offs, off+1)
		return i
	}
	locs := reArt.FindAllStringSubmatchIndex(s, -1)
	tits := reTit.FindAllStringIndex(s, -1)
	cab := func(off int) string { // cabeçalho vigente: linha do título + linha seguinte
		c := ""
		for _, t := range tits {
			if t[0] >= off {
				break
			}
			linhas := strings.SplitN(strings.TrimSpace(s[t[0]:min(len(s), t[1]+160)]), "\n", 3)
			c = strings.Join(strings.Fields(strings.Join(linhas[:min(2, len(linhas))], " ")), " ")
		}
		return c
	}
	// Limites: inícios de artigo e de cabeçalho (o cabeçalho não vaza para o artigo anterior).
	type lim struct {
		off int
		art string
	}
	var lims []lim
	for _, l := range locs {
		lims = append(lims, lim{l[0], s[l[2]:l[3]]})
	}
	if *contexto {
		for _, t := range tits {
			lims = append(lims, lim{t[0], ""})
		}
	}
	sort.SliceStable(lims, func(a, b int) bool { return lims[a].off < lims[b].off })
	var out []chunk
	add := func(art string, a, b int) {
		txt := strings.TrimSpace(s[a:b])
		for len([]rune(txt)) > 0 {
			r := []rune(txt)
			n := min(len(r), 1200)
			if n < len(r) { // corta em fim de frase/linha
				if k := strings.LastIndexAny(string(r[:n]), ".;\n"); k > 400 {
					n = len([]rune(string(r[:n])[:k+1]))
				}
			}
			piece := strings.TrimSpace(string(r[:n]))
			if len([]rune(piece)) > 40 {
				out = append(out, chunk{doc, art, piece, pag(a), cab(a)})
			}
			txt = string(r[n:])
		}
	}
	prev := 0
	prevArt := ""
	for _, l := range lims {
		add(prevArt, prev, l.off)
		prev, prevArt = l.off, l.art
	}
	add(prevArt, prev, len(s))
	return out
}

func index(db *sql.DB) {
	for _, q := range []string{
		`DROP TABLE IF EXISTS chunk`, `DROP TABLE IF EXISTS chunk_fts`,
		`CREATE TABLE chunk(id INTEGER PRIMARY KEY, doc TEXT, artigo TEXT, pagina INT, ctx TEXT, texto TEXT, emb BLOB, emb_modelo TEXT)`,
		`CREATE VIRTUAL TABLE chunk_fts USING fts5(texto, content='chunk', content_rowid='id', tokenize='unicode61 remove_diacritics 2')`,
	} {
		_, err := db.Exec(q)
		must(err)
	}
	files, _ := filepath.Glob(filepath.Join(*lib, "*.pdf"))
	var all []chunk
	t0 := time.Now()
	for _, f := range files {
		pages, err := extrair(f)
		must(err)
		cs := chunkar(strings.TrimSuffix(filepath.Base(f), ".pdf"), pages)
		fmt.Printf("%4d chunks  %s\n", len(cs), filepath.Base(f))
		all = append(all, cs...)
	}
	fmt.Printf("extração+chunking: %v, %d chunks\n", time.Since(t0), len(all))

	tx, _ := db.Begin()
	for _, c := range all {
		r, err := tx.Exec(`INSERT INTO chunk(doc, artigo, pagina, ctx, texto) VALUES(?,?,?,?,?)`, c.doc, c.artigo, c.pagina, c.ctx, c.texto)
		must(err)
		id, _ := r.LastInsertId()
		_, err = tx.Exec(`INSERT INTO chunk_fts(rowid, texto) VALUES(?,?)`, id, c.texto)
		must(err)
	}
	must(tx.Commit())

	srv := startServer()
	defer srv.stop()
	t1 := time.Now()
	rows, _ := db.Query(`SELECT id, doc, artigo, ctx, texto FROM chunk ORDER BY id`)
	var ids []int64
	var txts []string
	for rows.Next() {
		var id int64
		var doc, art, cx, t string
		rows.Scan(&id, &doc, &art, &cx, &t)
		if *contexto {
			h := doc
			if cx != "" {
				h += " — " + cx
			}
			if art != "" {
				h += " — Art. " + art
			}
			t = h + "\n" + t
		}
		ids, txts = append(ids, id), append(txts, t)
	}
	rows.Close()
	tx, _ = db.Begin()
	if *limite > 0 {
		ids, txts = ids[:*limite], txts[:*limite]
	}
	lote := *lote
	for i := 0; i < len(txts); i += lote {
		j := min(i+lote, len(txts))
		vs := srv.embed(txts[i:j])
		for k, v := range vs {
			_, err := tx.Exec(`UPDATE chunk SET emb=?, emb_modelo=? WHERE id=?`, f32blob(v), filepath.Base(*model), ids[i+k])
			must(err)
		}
		if (i/lote)%25 == 0 {
			fmt.Printf("  embeddings %d/%d (%.0fs)\n", j, len(txts), time.Since(t1).Seconds())
		}
	}
	must(tx.Commit())
	fmt.Printf("embeddings: %v (%.1f chunks/s)\n", time.Since(t1), float64(len(txts))/time.Since(t1).Seconds())
	if st, err := os.Stat(*dbPath); err == nil {
		fmt.Printf("tamanho do banco: %.1f MB\n", float64(st.Size())/1e6)
	}
}

// ---------- avaliação ----------

type consulta struct {
	Q      string `json:"q"`
	Doc    string `json:"doc"`    // substring do nome do documento esperado
	Artigo string `json:"artigo"` // número do artigo esperado
}

type hit struct {
	id          int64
	doc, artigo string
}

func eval(db *sql.DB) {
	b, err := os.ReadFile(*queries)
	must(err)
	var qs []consulta
	must(json.Unmarshal(b, &qs))

	// Carrega vetores em memória (força bruta).
	t0 := time.Now()
	rows, err := db.Query(`SELECT id, doc, artigo, emb FROM chunk WHERE emb IS NOT NULL`)
	must(err)
	var meta []hit
	var vecs [][]float32
	for rows.Next() {
		var h hit
		var blob []byte
		rows.Scan(&h.id, &h.doc, &h.artigo, &blob)
		meta, vecs = append(meta, h), append(vecs, blobf32(blob))
	}
	rows.Close()
	byID := map[int64]hit{}
	for _, h := range meta {
		byID[h.id] = h
	}
	fmt.Printf("vetores carregados: %d em %v (dim %d)\n", len(vecs), time.Since(t0), len(vecs[0]))

	srv := startServer()
	defer srv.stop()

	type score struct{ h1, h5 int }
	var sBM, sVec, sRRF score
	var tVec time.Duration
	for _, q := range qs {
		ok := func(h hit) bool { // Doc aceita alternativas "A|B" (ex.: RDPM avulso ou dentro da Coletânea)
			for _, d := range strings.Split(q.Doc, "|") {
				if strings.Contains(strings.ToUpper(h.doc), strings.ToUpper(d)) && h.artigo == q.Artigo {
					return true
				}
			}
			return false
		}

		bm := bm25(db, q.Q, 20)
		qv := srv.embed([]string{instrucao + q.Q})[0]
		t := time.Now()
		vec := knn(qv, vecs, meta, 20)
		tVec += time.Since(t)
		rrf := fuse(bm, vec)

		rank := func(ids []int64) int {
			for i, id := range ids {
				if ok(byID[id]) {
					return i + 1
				}
			}
			return 0
		}
		rb, rv, rr := rank(bm), rank(vec), rank(rrf)
		for _, p := range []struct {
			s *score
			r int
		}{{&sBM, rb}, {&sVec, rv}, {&sRRF, rr}} {
			if p.r == 1 {
				p.s.h1++
			}
			if p.r >= 1 && p.r <= 5 {
				p.s.h5++
			}
		}
		top := byID[rrf[0]]
		fmt.Printf("BM25=%-2d VET=%-2d RRF=%-2d | esperado %s art.%s | top1 %s art.%s | %s\n", rb, rv, rr, q.Doc, q.Artigo, short(top.doc), top.artigo, q.Q)
	}
	n := len(qs)
	fmt.Printf("\nconsultas=%d  (posição 0 = fora do top-20)\n", n)
	fmt.Printf("BM25   hit@1=%d/%d hit@5=%d/%d\n", sBM.h1, n, sBM.h5, n)
	fmt.Printf("Vetor  hit@1=%d/%d hit@5=%d/%d\n", sVec.h1, n, sVec.h5, n)
	fmt.Printf("RRF    hit@1=%d/%d hit@5=%d/%d\n", sRRF.h1, n, sRRF.h5, n)
	fmt.Printf("busca vetorial exata: %.2f ms/consulta sobre %d vetores\n", float64(tVec.Microseconds())/1000/float64(n), len(vecs))
}

var stop = map[string]bool{}

// Experimento F0-07: na Fase 5 viria de um tesauro curado (catálogo de regras).
var sinonimos = map[string][]string{
	"falta": {"transgressão"}, "faltas": {"transgressões"}, "infração": {"transgressão"},
	"sanção": {"punição"}, "pena": {"punição"}, "penalidades": {"punições"}, "punir": {"punição", "punições"},
	"diminuem": {"atenuam"}, "severa": {"agravam"}, "afastam": {"excluem", "justificam"},
	"admitiu": {"confissão"}, "culpa": {"confissão"}, "decisão": {"julgamento"}, "colabora": {"concorre"},
	"suspender": {"relevação"}, "graves": {"grave"}, "recursos": {"recurso"},
}

func init() {
	for _, w := range strings.Fields("a o as os de da do das dos e em no na nos nas um uma uns umas por para com sem que se ao aos à às é ser qual quais quando como pode deve há quem sobre seu sua seus suas ou mais") {
		stop[w] = true
	}
}

func bm25(db *sql.DB, q string, k int) []int64 {
	var terms []string
	for _, w := range strings.FieldsFunc(strings.ToLower(q), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if stop[w] || len([]rune(w)) < 2 {
			continue
		}
		ws := []string{w}
		if os.Getenv("SINON") == "1" { // sinônimos do domínio (coloquial → termo normativo)
			ws = append(ws, sinonimos[w]...)
		}
		for _, x := range ws {
			r := []rune(x)
			if os.Getenv("PREFIXO") == "1" && len(r) > 5 { // stemming rudimentar
				terms = append(terms, `"`+string(r[:5])+`"*`)
			} else {
				terms = append(terms, `"`+x+`"`)
			}
		}
	}
	rows, err := db.Query(`SELECT rowid FROM chunk_fts WHERE chunk_fts MATCH ? ORDER BY bm25(chunk_fts) LIMIT ?`, strings.Join(terms, " OR "), k)
	must(err)
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

func knn(q []float32, vecs [][]float32, meta []hit, k int) []int64 {
	type sc struct {
		id int64
		s  float32
	}
	all := make([]sc, len(vecs))
	for i, v := range vecs {
		var dot float32
		for j := range v {
			dot += v[j] * q[j]
		}
		all[i] = sc{meta[i].id, dot}
	}
	sort.Slice(all, func(a, b int) bool { return all[a].s > all[b].s })
	var ids []int64
	for _, s := range all[:min(k, len(all))] {
		ids = append(ids, s.id)
	}
	return ids
}

func fuse(lists ...[]int64) []int64 {
	sc := map[int64]float64{}
	for _, l := range lists {
		for i, id := range l {
			sc[id] += 1 / (60 + float64(i+1))
		}
	}
	var ids []int64
	for id := range sc {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool { return sc[ids[a]] > sc[ids[b]] })
	return ids
}

// ---------- llama-server (embeddings) ----------

type srv struct {
	cmd  *exec.Cmd
	base string
	key  string
}

func startServer() *srv {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	kb := make([]byte, 16)
	rand.Read(kb)
	s := &srv{base: fmt.Sprintf("http://127.0.0.1:%d", port), key: hex.EncodeToString(kb)}
	s.cmd = exec.Command(*server, "-m", *model, "--embedding", "--pooling", "last", "--host", "127.0.0.1", "--port", fmt.Sprint(port),
		"--api-key", s.key, "--no-webui", "--offline", "-c", *ctxEmb, "-b", *ctxEmb, "-ub", *ctxEmb, "-np", "1", "-ngl", "0", "--log-disable")
	must(s.cmd.Start())
	t0 := time.Now()
	for time.Since(t0) < 2*time.Minute {
		if r, err := http.Get(s.base + "/health"); err == nil {
			r.Body.Close()
			if r.StatusCode == 200 {
				return s
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	s.stop()
	must(fmt.Errorf("llama-server de embeddings não subiu"))
	return nil
}

func (s *srv) stop() { s.cmd.Process.Kill(); s.cmd.Wait() }

func (s *srv) embed(in []string) [][]float32 {
	body, _ := json.Marshal(map[string]any{"input": in, "encoding_format": "float"})
	rq, _ := http.NewRequest("POST", s.base+"/v1/embeddings", bytes.NewReader(body))
	rq.Header.Set("Authorization", "Bearer "+s.key)
	rq.Header.Set("Content-Type", "application/json")
	rs, err := http.DefaultClient.Do(rq)
	must(err)
	defer rs.Body.Close()
	var out struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		}
	}
	raw := new(bytes.Buffer)
	raw.ReadFrom(rs.Body)
	if err := json.Unmarshal(raw.Bytes(), &out); err != nil || len(out.Data) != len(in) {
		must(fmt.Errorf("embeddings: HTTP %d %s", rs.StatusCode, trunc(raw.String(), 300)))
	}
	vs := make([][]float32, len(in))
	for _, d := range out.Data {
		vs[d.Index] = normalizar(d.Embedding)
	}
	return vs
}

func normalizar(v []float32) []float32 {
	var n float64
	for _, x := range v {
		n += float64(x) * float64(x)
	}
	n = math.Sqrt(n)
	for i := range v {
		v[i] = float32(float64(v[i]) / n)
	}
	return v
}

func f32blob(v []float32) []byte {
	b := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(x))
	}
	return b
}

func blobf32(b []byte) []float32 {
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return v
}

func short(s string) string { return trunc(s, 28) }
func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

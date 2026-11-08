package autos

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite" // SQLite puro Go (sem CGO), com FTS5

	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

// Indice é a busca textual dos autos: SQLite FTS5 **em memória** (nada em
// disco; um arquivo SQLite guardaria o texto dos autos em claro).
type Indice struct {
	db      *sql.DB
	paginas []Pagina
}

// Ocorrencia é um trecho encontrado, com posição (runas) no texto da página.
type Ocorrencia struct {
	Pagina int    `json:"pagina"`
	Inicio int    `json:"inicio"`
	Fim    int    `json:"fim"`
	Trecho string `json:"trecho"`
}

// Indexar cria o índice com uma linha por página.
func Indexar(a *Autos) (*Indice, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // ":memory:" é por conexão: uma só conexão mantém o banco
	if _, err := db.Exec(`CREATE VIRTUAL TABLE paginas USING fts5(texto, tokenize = 'unicode61 remove_diacritics 2')`); err != nil {
		db.Close()
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		db.Close()
		return nil, err
	}
	st, err := tx.Prepare(`INSERT INTO paginas(rowid, texto) VALUES (?, ?)`)
	if err != nil {
		tx.Rollback()
		db.Close()
		return nil, err
	}
	for _, p := range a.Paginas {
		if _, err := st.Exec(p.Numero, p.Texto); err != nil {
			tx.Rollback()
			db.Close()
			return nil, err
		}
	}
	st.Close()
	if err := tx.Commit(); err != nil {
		db.Close()
		return nil, err
	}
	return &Indice{db: db, paginas: a.Paginas}, nil
}

// Fechar libera o banco em memória.
func (i *Indice) Fechar() error { return i.db.Close() }

// Buscar procura páginas com todos os termos da consulta (ordem BM25) e
// devolve, por página, a primeira ocorrência do primeiro termo. A consulta é
// reduzida a termos entre aspas: nenhuma sintaxe FTS5 do usuário é interpretada.
func (i *Indice) Buscar(consulta string, limite int) ([]Ocorrencia, error) {
	termos := texto.Termos(consulta)
	if len(termos) == 0 {
		return nil, nil
	}
	q := make([]string, len(termos))
	for k, t := range termos {
		q[k] = `"` + strings.ReplaceAll(t, `"`, ``) + `"`
	}
	if limite <= 0 {
		limite = 20
	}
	rows, err := i.db.Query(`SELECT rowid FROM paginas WHERE paginas MATCH ? ORDER BY bm25(paginas) LIMIT ?`, strings.Join(q, " "), limite)
	if err != nil {
		return nil, fmt.Errorf("busca: %w", err)
	}
	defer rows.Close()
	var out []Ocorrencia
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		if n < 1 || n > len(i.paginas) {
			continue
		}
		p := i.paginas[n-1]
		if ini, fim, ok := localizar(p.Texto, termos[0]); ok {
			r := []rune(p.Texto)
			a, b := max(0, ini-60), min(len(r), fim+60)
			out = append(out, Ocorrencia{Pagina: n, Inicio: ini, Fim: fim, Trecho: string(r[a:b])})
		}
	}
	return out, rows.Err()
}

// localizar acha o termo normalizado no texto (posições em runas do original).
func localizar(s, termo string) (int, int, bool) {
	norm := texto.Alinhado(s)
	alvo := []rune(termo)
	for k := 0; k+len(alvo) <= len(norm); k++ {
		if string(norm[k:k+len(alvo)]) == termo {
			return k, k + len(alvo), true
		}
	}
	return 0, 0, false
}

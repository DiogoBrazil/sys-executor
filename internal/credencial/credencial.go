// Pacote credencial guarda a chave da Gemini API do usuário (V4 §47.4,
// TRV-SEG-009). No Windows, a chave é cifrada com DPAPI no escopo da conta do
// usuário e gravada numa pasta com acesso restrito a essa conta. A chave
// nunca vai para preferências, interface, eventos, erros ou logs.
//
// A senha do SEI não passa por aqui: ela continua só em memória (V4 §33).
package credencial

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// ErrAusente: nenhuma chave cadastrada.
var ErrAusente = errors.New("chave não cadastrada")

// ErrSemProtecao: o sistema não oferece proteção de dados adequada (fora do
// Windows). A chave não é gravada em texto puro.
var ErrSemProtecao = errors.New("proteção de credenciais indisponível neste sistema")

// Loja guarda, lê e remove a chave. Ler devolve uma cópia que o chamador
// deve zerar depois de usar.
type Loja interface {
	Guardar(chave []byte) error
	Ler() ([]byte, error)
	Remover() error
	Existe() bool
}

// entropia vincula o dado cifrado a este uso: outro programa da mesma conta
// que chame a DPAPI sem ela não decifra o arquivo.
var entropia = []byte("sys-executor/gemini-api-key/v1")

const nomeArquivo = "gemini.dpapi"

// LojaArquivo grava a chave cifrada (DPAPI) em <dir>/gemini.dpapi.
type LojaArquivo struct {
	mu  sync.Mutex
	dir string
}

// NovaLojaArquivo usa a pasta dir (criada e protegida na primeira gravação).
func NovaLojaArquivo(dir string) *LojaArquivo { return &LojaArquivo{dir: dir} }

func (l *LojaArquivo) caminho() string { return filepath.Join(l.dir, nomeArquivo) }

// Guardar cifra e grava atomicamente. Uma chave vazia é recusada.
func (l *LojaArquivo) Guardar(chave []byte) error {
	if len(bytes.TrimSpace(chave)) == 0 {
		return errors.New("chave vazia")
	}
	cif, err := proteger(chave, entropia)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(l.dir, 0o700); err != nil {
		return err
	}
	if err := protegerDiretorio(l.dir); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(l.dir, ".gemini-*.tmp")
	if err != nil {
		return err
	}
	nome := tmp.Name()
	_, err = tmp.Write(cif)
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(nome, l.caminho())
	}
	if err != nil {
		os.Remove(nome)
	}
	return err
}

// Ler decifra a chave. Ausente → ErrAusente.
func (l *LojaArquivo) Ler() ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	cif, err := os.ReadFile(l.caminho())
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrAusente
	}
	if err != nil {
		return nil, err
	}
	return desproteger(cif, entropia)
}

// Remover apaga o arquivo (ausente não é erro).
func (l *LojaArquivo) Remover() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.Remove(l.caminho()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Existe informa se há chave gravada (sem decifrá-la).
func (l *LojaArquivo) Existe() bool {
	_, err := os.Stat(l.caminho())
	return err == nil
}

// LojaMemoria guarda a chave só em memória (testes e desenvolvimento).
type LojaMemoria struct {
	mu    sync.Mutex
	chave []byte
}

func (l *LojaMemoria) Guardar(chave []byte) error {
	if len(bytes.TrimSpace(chave)) == 0 {
		return errors.New("chave vazia")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	zerar(l.chave)
	l.chave = append([]byte(nil), chave...)
	return nil
}

func (l *LojaMemoria) Ler() ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.chave) == 0 {
		return nil, ErrAusente
	}
	return append([]byte(nil), l.chave...), nil
}

func (l *LojaMemoria) Remover() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	zerar(l.chave)
	l.chave = nil
	return nil
}

func (l *LojaMemoria) Existe() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.chave) > 0
}

// Zerar sobrescreve b (melhor esforço; V4 §33).
func Zerar(b []byte) { zerar(b) }

func zerar(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Package session implementa a sessão efêmera de uma tarefa (V4 §29; plano §13).
//
//   - Cada tarefa ganha uma pasta com nome aleatório sob a raiz de trabalho,
//     acessível só pela conta do Windows que executa o app (DACL protegida).
//   - O conteúdo é cifrado com AES-256-GCM usando uma chave aleatória que
//     existe **só na memória** do processo. Se o app cair, os arquivos que
//     sobrarem ficam ilegíveis (a chave morreu com o processo) e são apagados
//     na próxima abertura pela varredura de resíduos, sem nenhuma restauração.
//   - Nomes de arquivo são opacos; o nome lógico fica só em memória.
//   - Um marcador com PID e instante de criação do processo identifica o dono
//     da pasta. Ele não contém dado algum do processo administrativo.
//
// Não há promessa de apagamento forense (V4 §29.2): a sobrescrita antes da
// remoção é melhor esforço e não alcança cópias feitas pelo sistema (SSD,
// backups, caches).
package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

const nomeMarcador = "sessao.marcador"

// marcador é gravado em claro na pasta da sessão: só identifica o processo dono.
type marcador struct {
	PID    int   `json:"pid"`
	Inicio int64 `json:"inicio"`
}

// Gerenciador cria sessões e varre resíduos sob uma raiz de trabalho.
type Gerenciador struct {
	raiz string
}

// NovoGerenciador usa raiz (ex.: %LOCALAPPDATA%\sys-executor\trabalho).
func NovoGerenciador(raiz string) *Gerenciador { return &Gerenciador{raiz: raiz} }

// Raiz devolve a pasta raiz de trabalho.
func (g *Gerenciador) Raiz() string { return g.raiz }

// Sessao é a área efêmera de uma tarefa.
type Sessao struct {
	mu      sync.Mutex
	dir     string
	aead    cipher.AEAD
	chave   []byte
	nomes   map[string]string // nome lógico → arquivo opaco
	fechada bool
}

func aleatorio(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand não falha em sistemas suportados
	}
	return hex.EncodeToString(b)
}

// Criar prepara uma nova sessão protegida. Se a proteção de acesso falhar, a
// pasta é removida e a sessão não é criada (nunca trabalhar desprotegido).
func (g *Gerenciador) Criar() (*Sessao, error) {
	if err := os.MkdirAll(g.raiz, 0o700); err != nil {
		return nil, erros.Com(erros.SessaoCriacao, err)
	}
	if err := protegerDiretorio(g.raiz); err != nil {
		return nil, erros.Com(erros.SessaoProtecao, err)
	}
	dir := filepath.Join(g.raiz, aleatorio(16))
	if err := os.Mkdir(dir, 0o700); err != nil {
		return nil, erros.Com(erros.SessaoCriacao, err)
	}
	if err := protegerDiretorio(dir); err != nil {
		_ = os.RemoveAll(dir)
		return nil, erros.Com(erros.SessaoProtecao, err)
	}
	ini, err := inicioProcesso(os.Getpid())
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, erros.Com(erros.SessaoCriacao, err)
	}
	m, _ := json.Marshal(marcador{PID: os.Getpid(), Inicio: ini})
	if err := os.WriteFile(filepath.Join(dir, nomeMarcador), m, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return nil, erros.Com(erros.SessaoCriacao, err)
	}
	chave := make([]byte, 32)
	if _, err := rand.Read(chave); err != nil {
		_ = os.RemoveAll(dir)
		return nil, erros.Com(erros.SessaoCriacao, err)
	}
	bloco, err := aes.NewCipher(chave)
	if err != nil {
		return nil, erros.Com(erros.SessaoCriacao, err)
	}
	aead, err := cipher.NewGCM(bloco)
	if err != nil {
		return nil, erros.Com(erros.SessaoCriacao, err)
	}
	return &Sessao{dir: dir, aead: aead, chave: chave, nomes: map[string]string{}}, nil
}

// Dir devolve a pasta da sessão (para ferramentas externas nas fases seguintes).
func (s *Sessao) Dir() string { return s.dir }

var errFechada = errors.New("sessão encerrada")

// Gravar cifra dados e grava sob um nome opaco. Regravar o mesmo nome lógico
// substitui o conteúdo.
func (s *Sessao) Gravar(nome string, dados []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fechada {
		return erros.Com(erros.SessaoCriacao, errFechada)
	}
	arq, ok := s.nomes[nome]
	if !ok {
		arq = aleatorio(12) + ".bin"
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return erros.Com(erros.ArquivoEscrita, err)
	}
	cifrado := s.aead.Seal(nonce, nonce, dados, []byte(arq))
	if err := os.WriteFile(filepath.Join(s.dir, arq), cifrado, 0o600); err != nil {
		return erros.Com(erros.ArquivoEscrita, err)
	}
	s.nomes[nome] = arq
	return nil
}

// Ler decifra o conteúdo gravado sob o nome lógico.
func (s *Sessao) Ler(nome string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fechada {
		return nil, erros.Com(erros.ArquivoLeitura, errFechada)
	}
	arq, ok := s.nomes[nome]
	if !ok {
		return nil, erros.Com(erros.ArquivoLeitura, fs.ErrNotExist)
	}
	c, err := os.ReadFile(filepath.Join(s.dir, arq))
	if err != nil {
		return nil, erros.Com(erros.ArquivoLeitura, err)
	}
	ns := s.aead.NonceSize()
	if len(c) < ns {
		return nil, erros.Com(erros.ArquivoLeitura, errors.New("conteúdo truncado"))
	}
	d, err := s.aead.Open(nil, c[:ns], c[ns:], []byte(arq))
	if err != nil {
		return nil, erros.Com(erros.ArquivoLeitura, err)
	}
	return d, nil
}

// Encerrar descarta a chave e apaga a pasta (sobrescrita + remoção, melhor
// esforço). É idempotente. Se algo não puder ser removido agora (arquivo
// preso por outro processo), devolve SES-003: a varredura da próxima
// abertura conclui a limpeza.
func (s *Sessao) Encerrar() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fechada {
		return nil
	}
	s.fechada = true
	for i := range s.chave {
		s.chave[i] = 0
	}
	s.aead = nil
	s.nomes = nil
	if err := apagarPasta(s.dir); err != nil {
		return erros.Com(erros.SessaoLimpeza, err)
	}
	return nil
}

// apagarPasta sobrescreve com zeros os arquivos regulares e remove a pasta.
func apagarPasta(dir string) error {
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			sobrescrever(p)
		}
		return nil
	})
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("pasta ainda existe")
	}
	return nil
}

func sobrescrever(p string) {
	f, err := os.OpenFile(p, os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return
	}
	zeros := make([]byte, 64<<10)
	for rest := st.Size(); rest > 0; {
		n := int64(len(zeros))
		if rest < n {
			n = rest
		}
		if _, err := f.Write(zeros[:n]); err != nil {
			return
		}
		rest -= n
	}
	_ = f.Sync()
}

// ProtegerPasta cria dir (se preciso) e aplica a mesma proteção de acesso das
// sessões (somente a conta atual). Usado na pasta de dados do WebView2.
func ProtegerPasta(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return protegerDiretorio(dir)
}

// EsvaziarPasta remove o conteúdo de dir, tentando de novo até o prazo
// (arquivos ainda presos por processos que estão encerrando). Devolve true se
// a pasta ficou vazia.
func EsvaziarPasta(dir string, prazo time.Duration) bool {
	fim := time.Now().Add(prazo)
	for {
		entradas, err := os.ReadDir(dir)
		if err != nil {
			return os.IsNotExist(err)
		}
		for _, e := range entradas {
			_ = os.RemoveAll(filepath.Join(dir, e.Name()))
		}
		if restantes, _ := os.ReadDir(dir); len(restantes) == 0 {
			return true
		}
		if time.Now().After(fim) {
			return false
		}
		time.Sleep(250 * time.Millisecond)
	}
}

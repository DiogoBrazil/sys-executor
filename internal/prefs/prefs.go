// Package prefs guarda as preferências persistentes mínimas (V4 §29.3).
//
// Só configurações entram aqui: nunca dados de processos, credenciais,
// histórico de tarefas nem caminhos de autos. O formato é estrito (campos
// tipados, chaves desconhecidas rejeitadas) para que nada além disso possa
// ser gravado. Arquivo corrompido ou inválido → valores padrão.
package prefs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

// versaoAtual 3: dois provedores de IA (V4 §48.1) — entram o provedor
// escolhido e o modelo do ChatGPT. Histórico: v2 = modelo Gemini (D-10,
// V4 §47); v1 = inferência local. Credenciais (chave da Gemini API, tokens
// do ChatGPT) NUNCA entram aqui (pacote credencial, com DPAPI).
const versaoAtual = 3

// Provedores admitidos em ProvedorIA ("" = nenhum escolhido ainda).
const (
	ProvedorGemini  = "GEMINI"
	ProvedorChatGPT = "CHATGPT"
)

// Preferencias é o conjunto completo e fechado de preferências.
type Preferencias struct {
	Versao        int    `json:"versao"`
	PesquisaWeb   bool   `json:"pesquisa_web"`   // V4 §41: desligada na primeira instalação
	PastaDownload string `json:"pasta_download"` // vazio = perguntar ao salvar
	ProvedorIA    string `json:"provedor_ia"`    // "", GEMINI ou CHATGPT (escolha explícita, V4 §48.1)
	ModeloGemini  string `json:"modelo_gemini"`  // vazio = nenhum escolhido (V4 §47.5)
	ModeloChatGPT string `json:"modelo_chatgpt"` // slug da lista da conta; vazio = nenhum
}

// preferenciasV2 é o formato anterior (só Gemini), lido para migrar.
type preferenciasV2 struct {
	Versao        int    `json:"versao"`
	PesquisaWeb   bool   `json:"pesquisa_web"`
	PastaDownload string `json:"pasta_download"`
	ModeloGemini  string `json:"modelo_gemini"`
}

// preferenciasV1 é o formato anterior (inferência local), lido só para
// migrar o que continua valendo (pesquisa web e pasta de download).
type preferenciasV1 struct {
	Versao        int    `json:"versao"`
	PesquisaWeb   bool   `json:"pesquisa_web"`
	PastaDownload string `json:"pasta_download"`
	ModeloManual  string `json:"modelo_manual"`
	GPU           struct {
		Estado           string `json:"estado"`
		SondaEmAndamento bool   `json:"sonda_em_andamento"`
	} `json:"gpu"`
}

// Padrao devolve as preferências da primeira instalação.
func Padrao() Preferencias {
	return Preferencias{Versao: versaoAtual}
}

var (
	reModelo        = regexp.MustCompile(`^(gemini-[a-z0-9.-]{1,100})?$`)
	reModeloChatGPT = regexp.MustCompile(`^([a-z0-9][a-z0-9.\-]{0,99})?$`)
)

// Validar confere cada campo.
func (p Preferencias) Validar() error {
	if p.Versao != versaoAtual {
		return fmt.Errorf("versão %d não suportada", p.Versao)
	}
	if p.PastaDownload != "" && !filepath.IsAbs(p.PastaDownload) {
		return errors.New("pasta de download deve ser um caminho absoluto")
	}
	if !reModelo.MatchString(p.ModeloGemini) || !reModeloChatGPT.MatchString(p.ModeloChatGPT) {
		return errors.New("identificador de modelo inválido")
	}
	switch p.ProvedorIA {
	case "", ProvedorGemini, ProvedorChatGPT:
	default:
		return errors.New("provedor de IA inválido")
	}
	return nil
}

// Loja carrega e grava as preferências de um arquivo.
type Loja struct {
	mu      sync.Mutex
	caminho string
	atual   Preferencias
}

// Abrir lê o arquivo. Ausente → padrão. Inválido → padrão (o arquivo ruim é
// substituído na próxima gravação). Arquivos das versões 2 e 1 são migrados.
func Abrir(caminho string) (*Loja, error) {
	l := &Loja{caminho: caminho, atual: Padrao()}
	b, err := os.ReadFile(caminho)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return l, nil
	case err != nil:
		return l, nil // ilegível: segue com o padrão
	}
	var p Preferencias
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&p) == nil && p.Validar() == nil {
		l.atual = p
		return l, nil
	}
	for _, migrar := range []func([]byte) (Preferencias, bool){migrarV2, migrarV1} {
		if m, ok := migrar(b); ok {
			l.atual = m
			if err := l.gravar(m); err != nil {
				return l, err
			}
			return l, nil
		}
	}
	return l, nil
}

// migrarV2 mantém tudo; quem já tinha escolhido um modelo Gemini continua
// com a Gemini API como provedor (a escolha foi explícita).
func migrarV2(b []byte) (Preferencias, bool) {
	var v2 preferenciasV2
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&v2) != nil || v2.Versao != 2 {
		return Preferencias{}, false
	}
	p := Padrao()
	p.PesquisaWeb, p.PastaDownload, p.ModeloGemini = v2.PesquisaWeb, v2.PastaDownload, v2.ModeloGemini
	if p.ModeloGemini != "" {
		p.ProvedorIA = ProvedorGemini
	}
	if p.Validar() != nil {
		return Preferencias{}, false
	}
	return p, true
}

// migrarV1 aproveita pesquisa web e pasta de download do formato anterior.
func migrarV1(b []byte) (Preferencias, bool) {
	var v1 preferenciasV1
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&v1) != nil || v1.Versao != 1 {
		return Preferencias{}, false
	}
	p := Padrao()
	p.PesquisaWeb = v1.PesquisaWeb
	p.PastaDownload = v1.PastaDownload
	if p.Validar() != nil {
		return Preferencias{}, false
	}
	return p, true
}

// Obter devolve uma cópia das preferências atuais.
func (l *Loja) Obter() Preferencias {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.atual
}

// Atualizar aplica f sobre uma cópia, valida e grava atomicamente.
func (l *Loja) Atualizar(f func(*Preferencias)) (Preferencias, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	nova := l.atual
	f(&nova)
	nova.Versao = versaoAtual
	if err := nova.Validar(); err != nil {
		return l.atual, err
	}
	if err := l.gravar(nova); err != nil {
		return l.atual, err
	}
	l.atual = nova
	return nova, nil
}

// gravar escreve em arquivo temporário na mesma pasta e renomeia (atômico).
func (l *Loja) gravar(p Preferencias) error {
	if err := os.MkdirAll(filepath.Dir(l.caminho), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(l.caminho), ".preferencias-*.tmp")
	if err != nil {
		return err
	}
	nome := tmp.Name()
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		os.Remove(nome)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(nome)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(nome)
		return err
	}
	if err := os.Rename(nome, l.caminho); err != nil {
		os.Remove(nome)
		return err
	}
	return nil
}

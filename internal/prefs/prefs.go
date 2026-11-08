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

const versaoAtual = 1

// EstadoGPU resume o que se sabe sobre o uso seguro da GPU (D-08, D-09.1).
type EstadoGPU string

const (
	GPUDesconhecida EstadoGPU = "desconhecida" // nunca sondada: usar CPU
	GPUEstavel      EstadoGPU = "estavel"      // sonda concluída
	GPUInstavel     EstadoGPU = "instavel"     // sonda interrompida por queda: usar CPU
	GPUDesativada   EstadoGPU = "desativada"   // usuário optou por CPU
)

// GPU é o estado da sonda segura de GPU.
type GPU struct {
	Estado           EstadoGPU `json:"estado"`
	SondaEmAndamento bool      `json:"sonda_em_andamento"`
}

// Preferencias é o conjunto completo e fechado de preferências.
type Preferencias struct {
	Versao        int    `json:"versao"`
	PesquisaWeb   bool   `json:"pesquisa_web"`   // V4 §41: desligada na primeira instalação
	PastaDownload string `json:"pasta_download"` // vazio = perguntar ao salvar
	ModeloManual  string `json:"modelo_manual"`  // vazio = recomendação automática
	GPU           GPU    `json:"gpu"`
}

// Padrao devolve as preferências da primeira instalação.
func Padrao() Preferencias {
	return Preferencias{Versao: versaoAtual, GPU: GPU{Estado: GPUDesconhecida}}
}

var reModelo = regexp.MustCompile(`^[A-Za-z0-9._-]{0,120}$`)

// Validar confere cada campo.
func (p Preferencias) Validar() error {
	if p.Versao != versaoAtual {
		return fmt.Errorf("versão %d não suportada", p.Versao)
	}
	if p.PastaDownload != "" && !filepath.IsAbs(p.PastaDownload) {
		return errors.New("pasta de download deve ser um caminho absoluto")
	}
	if !reModelo.MatchString(p.ModeloManual) {
		return errors.New("identificador de modelo inválido")
	}
	switch p.GPU.Estado {
	case GPUDesconhecida, GPUEstavel, GPUInstavel, GPUDesativada:
	default:
		return fmt.Errorf("estado de GPU inválido: %q", p.GPU.Estado)
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
// substituído na próxima gravação). Também aplica a sentinela da sonda de GPU:
// se o app abriu com a sonda "em andamento", a última tentativa derrubou o
// processo ou a máquina → a GPU é marcada instável (D-09.1).
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
	}
	if l.atual.GPU.SondaEmAndamento {
		l.atual.GPU = GPU{Estado: GPUInstavel}
		if err := l.gravar(l.atual); err != nil {
			return l, err
		}
	}
	return l, nil
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

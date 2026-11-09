package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Credencial da PoC. ATENÇÃO: na PoC fica em arquivo 0600 (sem DPAPI) e
// guarda também o access token, para não renovar a cada comando. O app
// usará DPAPI e manterá o access token só em memória (plano §4.3).
type Credencial struct {
	Versao       int       `json:"versao"`
	HostID       string    `json:"host_id"`
	ClientID     string    `json:"client_id,omitempty"`
	Sub          string    `json:"sub,omitempty"`
	Email        string    `json:"email,omitempty"`
	Escopos      []string  `json:"escopos,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expira       time.Time `json:"expira,omitempty"`
	CedoRenovar  time.Time `json:"cedo_renovar,omitempty"`
	Salvo        time.Time `json:"salvo"`
}

func pastaPoC() (string, error) {
	if d := os.Getenv("SYSEXEC_POC_DIR"); d != "" {
		return d, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "sys-executor-poc"), nil
}

func caminhoCredencial() (string, error) {
	d, err := pastaPoC()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "chatgpt.json"), nil
}

func lerCredencial() (*Credencial, error) {
	p, err := caminhoCredencial()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return &Credencial{Versao: 1}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Credencial
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("credencial ilegível: %w", err)
	}
	return &c, nil
}

// gravar: atômico (temporário + rename), permissão 0600, pasta 0700.
func (c *Credencial) gravar() error {
	p, err := caminhoCredencial()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	c.Salvo = time.Now().UTC()
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".chatgpt-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

func removerCredencial() error {
	p, err := caminhoCredencial()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// garantirHostID cria o ext_agent_host_id estável (urn:uuid v4) na 1ª vez.
func (c *Credencial) garantirHostID() error {
	if c.HostID != "" {
		return nil
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	c.HostID = fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
	return c.gravar()
}

func (c *Credencial) temEscopo(e string) bool {
	for _, s := range c.Escopos {
		if s == e {
			return true
		}
	}
	return false
}

// mascararEmail: a***@dominio (o relatório não leva o e-mail).
func mascararEmail(e string) string {
	i := strings.IndexByte(e, '@')
	if i <= 0 {
		return "(sem e-mail)"
	}
	return e[:1] + "***" + e[i:]
}

package inference

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// Arquiteturas aceitas na importação (suportadas pelo llama.cpp fixado).
var arquiteturasSuportadas = map[string]bool{
	"qwen2": true, "qwen3": true, "qwen35": true, "qwen3moe": true, "llama": true,
	"gemma2": true, "gemma3": true, "phi3": true, "mistral3": true, "granite": true,
}

const nomeIndice = "modelos.json"

// Progresso informa bytes feitos e total (total 0 = desconhecido).
type Progresso func(feito, total int64)

// GerenciadorModelos administra a pasta de modelos (V4 §37; T27; MOD-01…04).
type GerenciadorModelos struct {
	Dir      string
	Catalogo CatalogoModelos
	HTTP     *http.Client

	mu    sync.Mutex
	emUso map[string]int
}

// NovoGerenciadorModelos usa o catálogo embutido.
func NovoGerenciadorModelos(dir string) (*GerenciadorModelos, error) {
	c, err := Catalogo()
	if err != nil {
		return nil, err
	}
	return &GerenciadorModelos{Dir: dir, Catalogo: c, HTTP: &http.Client{Timeout: 0}, emUso: map[string]int{}}, nil
}

type entradaIndice struct {
	Modelo    Modelo `json:"modelo"`
	Arquivo   string `json:"arquivo"`
	Tamanho   int64  `json:"tamanho"`
	Modificao int64  `json:"modificado"`
}

func (g *GerenciadorModelos) lerIndice() ([]entradaIndice, error) {
	b, err := os.ReadFile(filepath.Join(g.Dir, nomeIndice))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var es []entradaIndice
	if err := json.Unmarshal(b, &es); err != nil {
		return nil, nil // índice danificado: recomeça (os arquivos continuam)
	}
	return es, nil
}

func (g *GerenciadorModelos) gravarIndice(es []entradaIndice) error {
	sort.Slice(es, func(i, j int) bool { return es[i].Modelo.ID < es[j].Modelo.ID })
	b, err := json.MarshalIndent(es, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(g.Dir, nomeIndice+".tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(g.Dir, nomeIndice))
}

// Listar devolve os modelos instalados cujo arquivo continua íntegro
// (mesmo tamanho e data do registro; o hash foi conferido na instalação).
func (g *GerenciadorModelos) Listar() ([]Instalado, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	es, err := g.lerIndice()
	if err != nil {
		return nil, err
	}
	var out []Instalado
	for _, e := range es {
		cam := filepath.Join(g.Dir, e.Arquivo)
		st, err := os.Stat(cam)
		if err != nil || st.Size() != e.Tamanho || st.ModTime().UnixNano() != e.Modificao {
			continue
		}
		f, err := os.Open(cam)
		if err != nil {
			continue
		}
		meta, err := LerGGUF(f)
		f.Close()
		if err != nil {
			continue
		}
		m := e.Modelo
		if c, ok := g.Catalogo.PorID(m.ID); ok { // catálogo novo prevalece (medições atualizadas)
			m = c
		}
		out = append(out, Instalado{Modelo: m, Caminho: cam, Meta: meta})
	}
	return out, nil
}

func (g *GerenciadorModelos) registrar(m Modelo, arquivo string) (Instalado, error) {
	cam := filepath.Join(g.Dir, arquivo)
	st, err := os.Stat(cam)
	if err != nil {
		return Instalado{}, err
	}
	f, err := os.Open(cam)
	if err != nil {
		return Instalado{}, err
	}
	meta, err := LerGGUF(f)
	f.Close()
	if err != nil {
		return Instalado{}, erros.Com(erros.ModeloIncompativel, err)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	es, err := g.lerIndice()
	if err != nil {
		return Instalado{}, err
	}
	var novo []entradaIndice
	for _, e := range es {
		if e.Modelo.ID != m.ID {
			novo = append(novo, e)
		}
	}
	novo = append(novo, entradaIndice{Modelo: m, Arquivo: arquivo, Tamanho: st.Size(), Modificao: st.ModTime().UnixNano()})
	if err := g.gravarIndice(novo); err != nil {
		return Instalado{}, erros.Com(erros.ArquivoEscrita, err)
	}
	return Instalado{Modelo: m, Caminho: cam, Meta: meta}, nil
}

// Baixar instala um modelo do catálogo por HTTPS, retomando um download
// parcial, e confere tamanho e SHA-256 antes de registrar (T27).
func (g *GerenciadorModelos) Baixar(ctx context.Context, id string, prog Progresso) (Instalado, error) {
	m, ok := g.Catalogo.PorID(id)
	if !ok {
		return Instalado{}, erros.Com(erros.ModeloAusente, fmt.Errorf("modelo %q fora do catálogo", id))
	}
	u, err := url.Parse(m.URL)
	if err != nil || u.Scheme != "https" {
		return Instalado{}, erros.Com(erros.ModeloDownload, errors.New("URL do catálogo não é HTTPS"))
	}
	return g.baixar(ctx, m, prog)
}

func (g *GerenciadorModelos) baixar(ctx context.Context, m Modelo, prog Progresso) (Instalado, error) {
	if err := os.MkdirAll(g.Dir, 0o700); err != nil {
		return Instalado{}, erros.Com(erros.ArquivoEscrita, err)
	}
	parcial := filepath.Join(g.Dir, m.Arquivo+".parcial")
	h := sha256.New()
	feito, err := retomar(parcial, h)
	if err != nil {
		return Instalado{}, erros.Com(erros.ArquivoLeitura, err)
	}
	if feito > m.Tamanho {
		_ = os.Remove(parcial)
		h.Reset()
		feito = 0
	}
	if feito < m.Tamanho {
		if feito, err = g.transferir(ctx, m, parcial, feito, h, prog); err != nil {
			return Instalado{}, err
		}
	}
	soma := hex.EncodeToString(h.Sum(nil))
	if feito != m.Tamanho || soma != m.SHA256 {
		_ = os.Remove(parcial)
		return Instalado{}, erros.Com(erros.ModeloIntegridade, fmt.Errorf("tamanho %d/%d, sha %s", feito, m.Tamanho, soma))
	}
	final := filepath.Join(g.Dir, m.Arquivo)
	if err := os.Rename(parcial, final); err != nil {
		return Instalado{}, erros.Com(erros.ArquivoEscrita, err)
	}
	return g.registrar(m, m.Arquivo)
}

// retomar soma ao hash o que já foi baixado.
func retomar(parcial string, h hash.Hash) (int64, error) {
	f, err := os.Open(parcial)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(h, f)
}

func (g *GerenciadorModelos) transferir(ctx context.Context, m Modelo, parcial string, feito int64, h hash.Hash, prog Progresso) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
	if err != nil {
		return feito, erros.Com(erros.ModeloDownload, err)
	}
	req.Header.Set("User-Agent", "sys-executor")
	if feito > 0 {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(feito, 10)+"-")
	}
	resp, err := g.HTTP.Do(req)
	if err != nil {
		return feito, erros.Com(erros.ModeloDownload, err)
	}
	defer resp.Body.Close()
	modo := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	switch {
	case resp.StatusCode == http.StatusPartialContent && feito > 0:
		if !strings.HasPrefix(resp.Header.Get("Content-Range"), fmt.Sprintf("bytes %d-", feito)) {
			return feito, erros.Com(erros.ModeloDownload, errors.New("Content-Range inesperado"))
		}
	case resp.StatusCode == http.StatusOK:
		modo = os.O_CREATE | os.O_WRONLY | os.O_TRUNC // servidor ignorou o Range: recomeça
		h.Reset()
		feito = 0
	default:
		return feito, erros.Com(erros.ModeloDownload, fmt.Errorf("HTTP %d", resp.StatusCode))
	}
	f, err := os.OpenFile(parcial, modo, 0o600)
	if err != nil {
		return feito, erros.Com(erros.ArquivoEscrita, err)
	}
	defer f.Close()
	buf := make([]byte, 1<<20)
	ultimo := time.Time{}
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if feito+int64(n) > m.Tamanho {
				return feito, erros.Com(erros.ModeloIntegridade, errors.New("arquivo maior que o esperado"))
			}
			if _, err := f.Write(buf[:n]); err != nil {
				return feito, erros.Com(erros.ArquivoSemEspac, err)
			}
			h.Write(buf[:n])
			feito += int64(n)
			if prog != nil && time.Since(ultimo) > 200*time.Millisecond {
				prog(feito, m.Tamanho)
				ultimo = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return feito, erros.Com(erros.ModeloDownload, rerr)
		}
	}
	if prog != nil {
		prog(feito, m.Tamanho)
	}
	return feito, nil
}

// Importar copia um GGUF existente para a pasta de modelos, validando o
// cabeçalho, a arquitetura e calculando o SHA-256 (MOD-03). Se o hash for de
// um modelo do catálogo, ele é reconhecido como tal (perfil testado); senão
// fica disponível só na seleção manual.
func (g *GerenciadorModelos) Importar(ctx context.Context, origem string, prog Progresso) (Instalado, error) {
	f, err := os.Open(origem)
	if err != nil {
		return Instalado{}, erros.Com(erros.ArquivoLeitura, err)
	}
	defer f.Close()
	meta, err := LerGGUF(f)
	if err != nil {
		return Instalado{}, erros.Com(erros.ModeloIncompativel, err)
	}
	if !arquiteturasSuportadas[meta.Arquitetura] {
		return Instalado{}, erros.Com(erros.ModeloIncompativel, fmt.Errorf("arquitetura %q", meta.Arquitetura))
	}
	papel := PapelGerador
	if meta.Pooling != 0 {
		papel = PapelEmbeddings
	} else if !meta.TemplateDeChat {
		return Instalado{}, erros.Com(erros.ModeloIncompativel, errors.New("sem template de chat"))
	}
	st, err := f.Stat()
	if err != nil {
		return Instalado{}, erros.Com(erros.ArquivoLeitura, err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return Instalado{}, erros.Com(erros.ArquivoLeitura, err)
	}
	if err := os.MkdirAll(g.Dir, 0o700); err != nil {
		return Instalado{}, erros.Com(erros.ArquivoEscrita, err)
	}
	tmp, err := os.CreateTemp(g.Dir, "importando-*.parcial")
	if err != nil {
		return Instalado{}, erros.Com(erros.ArquivoEscrita, err)
	}
	defer os.Remove(tmp.Name())
	h := sha256.New()
	_, err = copiar(ctx, io.MultiWriter(tmp, h), f, st.Size(), prog)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return Instalado{}, err
		}
		return Instalado{}, erros.Com(erros.ArquivoEscrita, err)
	}
	soma := hex.EncodeToString(h.Sum(nil))
	m, doCatalogo := g.Catalogo.PorSHA(soma)
	if !doCatalogo {
		nome := meta.Nome
		if nome == "" {
			nome = strings.TrimSuffix(filepath.Base(origem), filepath.Ext(origem))
		}
		m = Modelo{
			ID: "importado-" + soma[:12], Nome: nome + " (importado)", Papel: papel, Perfil: "importado",
			Arquivo: "importado-" + soma[:12] + ".gguf", SHA256: soma, Tamanho: st.Size(),
			Licenca: meta.Licenca, Arquitetura: meta.Arquitetura,
			// Sem medição: estimativa conservadora do pico privado (40% dos pesos + 1 GiB).
			PrivadoMB: int(st.Size()*2/5/mib) + 1024,
			Nota:      "Importado pelo usuário; sem perfil testado (somente seleção manual).",
		}
	}
	if err := os.Rename(tmp.Name(), filepath.Join(g.Dir, m.Arquivo)); err != nil {
		return Instalado{}, erros.Com(erros.ArquivoEscrita, err)
	}
	return g.registrar(m, m.Arquivo)
}

func copiar(ctx context.Context, dst io.Writer, src io.Reader, total int64, prog Progresso) (int64, error) {
	buf := make([]byte, 4<<20)
	var feito int64
	for {
		if err := ctx.Err(); err != nil {
			return feito, err
		}
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, err := dst.Write(buf[:n]); err != nil {
				return feito, err
			}
			feito += int64(n)
			if prog != nil {
				prog(feito, total)
			}
		}
		if rerr == io.EOF {
			return feito, nil
		}
		if rerr != nil {
			return feito, rerr
		}
	}
}

// Remover apaga o arquivo e o registro (MOD-04). Modelo em uso → MOD-009.
func (g *GerenciadorModelos) Remover(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.emUso[id] > 0 {
		return erros.Novo(erros.ModeloEmUso)
	}
	es, err := g.lerIndice()
	if err != nil {
		return err
	}
	var novo []entradaIndice
	achou := false
	for _, e := range es {
		if e.Modelo.ID == id {
			achou = true
			if err := os.Remove(filepath.Join(g.Dir, e.Arquivo)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return erros.Com(erros.ArquivoEscrita, err)
			}
			_ = os.Remove(filepath.Join(g.Dir, e.Arquivo+".parcial"))
			continue
		}
		novo = append(novo, e)
	}
	if !achou {
		return erros.Com(erros.ModeloAusente, fmt.Errorf("modelo %s não instalado", id))
	}
	return g.gravarIndice(novo)
}

// Reservar marca o modelo como em uso pela tarefa; devolve a liberação.
func (g *GerenciadorModelos) Reservar(id string) func() {
	g.mu.Lock()
	g.emUso[id]++
	g.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			g.mu.Lock()
			g.emUso[id]--
			g.mu.Unlock()
		})
	}
}

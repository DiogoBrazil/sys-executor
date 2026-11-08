// Pacote inference: IA local (V4 §§36–38; plano §9).
//
//   - catálogo de modelos testados (hashes, licença, memória medida);
//   - detecção de hardware, orçamento de memória e contexto adaptativo;
//   - recomendação automática e seleção manual;
//   - gerenciador de modelos (download retomável com SHA-256, importação de
//     GGUF, listagem e remoção);
//   - sidecar llama-server (loopback, porta e chave aleatórias, sem web UI,
//     sem log em disco, Job Object no Windows) e cliente com json_schema.
//
// Nenhuma regra jurídica depende do modelo (V4 §36.4): o pacote só gera texto
// estruturado sob schema; validar e decidir é de outros pacotes.
package inference

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
)

// Papel do modelo.
const (
	PapelGerador    = "gerador"
	PapelEmbeddings = "embeddings"
)

// Modelo é uma entrada do catálogo (ou um GGUF importado fora dele).
type Modelo struct {
	ID          string `json:"id"`
	Nome        string `json:"nome"`
	Papel       string `json:"papel"`
	Perfil      string `json:"perfil"`
	Arquivo     string `json:"arquivo"`
	URL         string `json:"url"`
	SHA256      string `json:"sha256"`
	Tamanho     int64  `json:"tamanho"`
	Licenca     string `json:"licenca"`
	Arquitetura string `json:"arquitetura"`
	PrivadoMB   int    `json:"privado_mb"` // pico privado medido a 8192 de contexto
	Qualidade   int    `json:"qualidade"`  // maior = melhor no benchmark
	Testado     bool   `json:"testado"`    // perfil testado: elegível à recomendação automática
	Nota        string `json:"nota"`
}

// CatalogoModelos é o conjunto embutido.
type CatalogoModelos struct {
	Versao  int      `json:"versao"`
	Obs     string   `json:"observacao"`
	Modelos []Modelo `json:"modelos"`
}

//go:embed catalogo_modelos.json
var catalogoJSON []byte

var (
	reSHA     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	reIDModel = regexp.MustCompile(`^[a-z0-9][a-z0-9.\-]*$`)
	reArquivo = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._\-]*\.gguf$`)
)

// Catalogo devolve o catálogo embutido, validado.
func Catalogo() (CatalogoModelos, error) {
	var c CatalogoModelos
	if err := json.Unmarshal(catalogoJSON, &c); err != nil {
		return c, err
	}
	return c, c.Validar()
}

// Validar confere campos e unicidade; URL só HTTPS do Hugging Face, fixada em commit.
func (c CatalogoModelos) Validar() error {
	ids, arqs := map[string]bool{}, map[string]bool{}
	reURL := regexp.MustCompile(`^https://huggingface\.co/[\w.\-]+/[\w.\-]+/resolve/[0-9a-f]{40}/[\w.\-]+\.gguf$`)
	for _, m := range c.Modelos {
		switch {
		case !reIDModel.MatchString(m.ID), ids[m.ID]:
			return fmt.Errorf("catálogo de modelos: id inválido ou repetido %q", m.ID)
		case !reArquivo.MatchString(m.Arquivo), arqs[m.Arquivo]:
			return fmt.Errorf("catálogo de modelos: arquivo inválido ou repetido %q", m.Arquivo)
		case !reSHA.MatchString(m.SHA256):
			return fmt.Errorf("catálogo de modelos: sha256 inválido em %s", m.ID)
		case !reURL.MatchString(m.URL):
			return fmt.Errorf("catálogo de modelos: URL fora do padrão em %s", m.ID)
		case m.Papel != PapelGerador && m.Papel != PapelEmbeddings:
			return fmt.Errorf("catálogo de modelos: papel inválido em %s", m.ID)
		case m.Tamanho <= 0 || m.PrivadoMB <= 0:
			return fmt.Errorf("catálogo de modelos: tamanho/memória ausentes em %s", m.ID)
		}
		ids[m.ID], arqs[m.Arquivo] = true, true
	}
	return nil
}

// PorID busca um modelo.
func (c CatalogoModelos) PorID(id string) (Modelo, bool) {
	for _, m := range c.Modelos {
		if m.ID == id {
			return m, true
		}
	}
	return Modelo{}, false
}

// PorSHA busca um modelo pelo hash (importação de um GGUF do catálogo).
func (c CatalogoModelos) PorSHA(sha string) (Modelo, bool) {
	for _, m := range c.Modelos {
		if m.SHA256 == sha {
			return m, true
		}
	}
	return Modelo{}, false
}

// Geradores devolve os geradores, do melhor para o pior no benchmark.
func (c CatalogoModelos) Geradores() []Modelo {
	var out []Modelo
	for _, m := range c.Modelos {
		if m.Papel == PapelGerador {
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Qualidade > out[j].Qualidade })
	return out
}

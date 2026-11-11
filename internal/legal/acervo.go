package legal

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
)

// Categorias da V4 §40.5.
const (
	CatLegislacao  = "LEGISLACAO"
	CatRegulamento = "REGULAMENTO_DISCIPLINAR"
	CatNorma       = "NORMA_INSTITUCIONAL"
	CatJuris       = "JURISPRUDENCIA"
	CatOutros      = "OUTROS"
)

// RotuloCategoria é o nome da categoria na interface.
var RotuloCategoria = map[string]string{
	CatLegislacao:  "Legislação",
	CatRegulamento: "Regulamento disciplinar",
	CatNorma:       "Norma institucional",
	CatJuris:       "Jurisprudência",
	CatOutros:      "Outros documentos jurídicos",
}

// Documento é um original da biblioteca, com a identificação e o texto
// estruturado. O SHA-256 do arquivo identifica a versão indexada (V4 §40.6).
type Documento struct {
	ID        string   `json:"id" yaml:"id"`
	Arquivo   string   `json:"arquivo" yaml:"arquivo"`
	Titulo    string   `json:"titulo" yaml:"titulo"`
	Siglas    []string `json:"siglas" yaml:"siglas"`
	Categoria string   `json:"categoria" yaml:"categoria"`
	// Composto: coletânea de várias normas; cada parte é uma norma.
	Composto bool `json:"composto,omitempty" yaml:"composto"`

	SHA256  string `json:"sha256" yaml:"-"`
	Paginas int    `json:"paginas" yaml:"-"`
	// PaginasOCR: páginas sem texto, lidas por reconhecimento de imagem.
	PaginasOCR int `json:"paginas_ocr,omitempty" yaml:"-"`
	// OCR: a maior parte do texto veio de reconhecimento de imagem
	// (conferir no original antes de citar).
	OCR bool `json:"ocr,omitempty" yaml:"-"`
	// SemTexto: original escaneado ainda não lido (fora da busca).
	SemTexto bool    `json:"sem_texto,omitempty" yaml:"-"`
	Partes   []Parte `json:"partes" yaml:"-"`
}

// Versao identifica a versão indexada do original (começo do SHA-256).
func (d Documento) Versao() string {
	if len(d.SHA256) >= 12 {
		return d.SHA256[:12]
	}
	return d.SHA256
}

// principal é a parte com mais artigos: a norma em si, e não o ato que a
// aprova. As siglas do documento apontam para ela.
func (d Documento) principal() int {
	melhor, n := 0, -1
	for i, p := range d.Partes {
		if a := p.Artigos(); a > n {
			melhor, n = i, a
		}
	}
	return melhor
}

// Acervo é o conjunto indexado.
type Acervo struct {
	Versao     int         `json:"versao"`
	Documentos []Documento `json:"documentos"`
}

// Manifesto é a identificação curada dos originais.
type Manifesto struct {
	Versao     int         `yaml:"versao"`
	Documentos []Documento `yaml:"documentos"`
}

//go:embed dados/manifesto.yaml
var manifestoYAML []byte

//go:embed dados/acervo.json.gz
var acervoGZ []byte

// LerManifesto devolve o manifesto embutido.
func LerManifesto() (Manifesto, error) {
	var m Manifesto
	if err := yaml.Unmarshal(manifestoYAML, &m); err != nil {
		return m, fmt.Errorf("manifesto da biblioteca: %w", err)
	}
	vistos := map[string]bool{}
	for _, d := range m.Documentos {
		if d.ID == "" || d.Arquivo == "" || d.Titulo == "" || RotuloCategoria[d.Categoria] == "" {
			return m, fmt.Errorf("manifesto da biblioteca: documento %q incompleto", d.ID)
		}
		if vistos[d.ID] {
			return m, fmt.Errorf("manifesto da biblioteca: id %q repetido", d.ID)
		}
		vistos[d.ID] = true
	}
	return m, nil
}

// AcervoInicial devolve o acervo embutido no executável (gerado por
// cmd/bibgen a partir de docs/law_library).
func AcervoInicial() (*Acervo, error) {
	return lerAcervo(acervoGZ)
}

func lerAcervo(gz []byte) (*Acervo, error) {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, fmt.Errorf("acervo da biblioteca: %w", err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("acervo da biblioteca: %w", err)
	}
	var a Acervo
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, fmt.Errorf("acervo da biblioteca: %w", err)
	}
	return &a, nil
}

// Gravar serializa o acervo de forma reprodutível (mesma entrada, mesmos bytes).
func (a *Acervo) Gravar() ([]byte, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := w.Write(b); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Pacote catalog: catálogo de regras da V4 (REGRAS_IA_ESTRATEGIA.md §2.1).
//
// É a fonte única de onde se derivam instruções à IA, perguntas, textos
// fixos, validadores e testes. Os arquivos *.yaml vão embutidos no
// executável; a carga é estrita (campo desconhecido = erro) e cada entrada é
// validada. A governança (cmd/rulescov) confere o catálogo contra a matriz,
// a V4, os símbolos Go de enforced_by e as anotações "// Cobre:" dos testes.
package catalog

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

//go:embed *.yaml
var embutido embed.FS

// Rótulos (V4: aprovado, interpretação registrada, proposta técnica, a validar).
const (
	Aprovado      = "APROVADO"
	Interpretacao = "INTERPRETACAO"
	Proposta      = "PROPOSTA"
	AValidar      = "A_VALIDAR"
)

// Tipos de regra.
const (
	Deterministica = "DETERMINISTICA"
	PoliticaIA     = "POLITICA_IA"
	Validador      = "VALIDADOR"
	Pergunta       = "PERGUNTA"
	TextoFixo      = "TEXTO_FIXO"
	UI             = "UI"
	Seguranca      = "SEGURANCA"
	Arquitetura    = "ARQUITETURA"
	Processo       = "PROCESSO"
)

// Situação da regra. IDs nunca são reutilizados: a regra descontinuada fica
// no catálogo como revogada.
const (
	Vigente  = "vigente"
	Revogada = "revogada"
)

// Etapas do orquestrador às quais uma regra de IA se aplica (plano §5.4).
// O núcleo entra em toda chamada.
const (
	EtapaNucleo          = "nucleo"
	EtapaClassificacao   = "classificacao"
	EtapaCompatibilidade = "compatibilidade"
	EtapaDelimitacao     = "delimitacao"
	EtapaExtracao        = "extracao"
	EtapaEscopo          = "escopo"
	EtapaAnalise         = "analise"
	EtapaRedacao         = "redacao"
	EtapaValidacao       = "validacao"
	EtapaAjuste          = "ajuste"
	// EtapaEnquadramento: proposta de enquadramento das conclusões da SR com
	// indícios (V4 §25.3); no PADS o enquadramento é do usuário (§48.5).
	EtapaEnquadramento = "enquadramento"
)

// Etapas lista as etapas de IA (fora o núcleo), em ordem.
var Etapas = []string{EtapaClassificacao, EtapaCompatibilidade, EtapaDelimitacao, EtapaExtracao,
	EtapaEscopo, EtapaAnalise, EtapaRedacao, EtapaValidacao, EtapaAjuste, EtapaEnquadramento}

// Condições (flags do caso) que ativam regras.
const (
	CondMidiaAusente = "midia_ausente"
	CondTemPunicao   = "tem_punicao"
)

// Condicoes lista as condições conhecidas.
var Condicoes = []string{CondMidiaAusente, CondTemPunicao}

var (
	etapasValidas  = conjunto(append([]string{EtapaNucleo}, Etapas...)...)
	condValidas    = conjunto(Condicoes...)
	rotulosValidos = conjunto(Aprovado, Interpretacao, Proposta, AValidar)
	tiposValidos   = conjunto(Deterministica, PoliticaIA, Validador, Pergunta, TextoFixo, UI, Seguranca, Arquitetura, Processo)
	fluxosValidos  = conjunto("PADS", "SR", "TRV")

	reID      = regexp.MustCompile(`^(PADS|SR|TRV)-[A-Z0-9]+-\d{3}$`)
	reTeste   = regexp.MustCompile(`^[A-Z][A-Z0-9]*(-[A-Z0-9]+)*$`)
	reInterp  = regexp.MustCompile(`^I-\d{2}$`)
	reSimbolo = regexp.MustCompile(`^[a-z][a-z0-9]*\.[A-Z][A-Za-z0-9]*$`)
)

// Regra é uma entrada do catálogo.
type Regra struct {
	ID            string   `yaml:"id"`
	Origem        string   `yaml:"origem"`
	Fluxo         string   `yaml:"fluxo"`
	Rotulo        []string `yaml:"rotulo"`
	Interpretacao string   `yaml:"interpretacao,omitempty"` // I-0n do registro de decisões
	Tipo          []string `yaml:"tipo"`
	Enunciado     string   `yaml:"enunciado"`
	Modulo        string   `yaml:"modulo"`                // módulo planejado (matriz)
	EnforcedBy    []string `yaml:"enforced_by,omitempty"` // símbolos Go que garantem a regra (pacote.Nome)
	Etapas        []string `yaml:"etapas,omitempty"`
	Condicao      []string `yaml:"condicao,omitempty"`
	InstrucaoIA   string   `yaml:"instrucao_ia,omitempty"`
	Falha         string   `yaml:"falha,omitempty"`
	Testes        []string `yaml:"testes"`
	Versao        int      `yaml:"versao"`
	Status        string   `yaml:"status,omitempty"` // vazio = vigente

	Arquivo string `yaml:"-"`
}

// Tem diz se a regra é do tipo t.
func (r Regra) Tem(t string) bool {
	for _, x := range r.Tipo {
		if x == t {
			return true
		}
	}
	return false
}

// Vigente diz se a regra não foi revogada.
func (r Regra) Vigente() bool { return r.Status == "" || r.Status == Vigente }

// Catalogo é o conjunto carregado e validado.
type Catalogo struct {
	Regras []Regra
	porID  map[string]int
}

// PorID devolve a regra pelo identificador.
func (c *Catalogo) PorID(id string) (Regra, bool) {
	i, ok := c.porID[id]
	if !ok {
		return Regra{}, false
	}
	return c.Regras[i], true
}

// Testes devolve o conjunto de IDs de teste citados pelas regras vigentes.
func (c *Catalogo) Testes() map[string][]string {
	out := map[string][]string{}
	for _, r := range c.Regras {
		if !r.Vigente() {
			continue
		}
		for _, t := range r.Testes {
			out[t] = append(out[t], r.ID)
		}
	}
	return out
}

// Selecionar devolve as regras de IA vigentes do fluxo (PADS ou SR, mais as
// transversais) para a etapa, com todas as condições satisfeitas, em ordem de
// ID. A seleção é sempre por chave, nunca por similaridade (§2.2 da
// estratégia).
func (c *Catalogo) Selecionar(fluxo, etapa string, flags map[string]bool) []Regra {
	var out []Regra
	for _, r := range c.Regras {
		if !r.Vigente() || !r.Tem(PoliticaIA) || (r.Fluxo != fluxo && r.Fluxo != "TRV") || !contem(r.Etapas, etapa) {
			continue
		}
		ok := true
		for _, cnd := range r.Condicao {
			ok = ok && flags[cnd]
		}
		if ok {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Carregar lê o catálogo embutido no executável.
func Carregar() (*Catalogo, error) { return CarregarFS(embutido) }

// CarregarFS lê todos os *.yaml da raiz de fsys, em ordem de nome.
func CarregarFS(fsys fs.FS) (*Catalogo, error) {
	nomes, err := fs.Glob(fsys, "*.yaml")
	if err != nil {
		return nil, err
	}
	if len(nomes) == 0 {
		return nil, errors.New("catálogo vazio")
	}
	sort.Strings(nomes)
	c := &Catalogo{porID: map[string]int{}}
	var problemas []string
	for _, nome := range nomes {
		b, err := fs.ReadFile(fsys, nome)
		if err != nil {
			return nil, err
		}
		regras, err := decodificar(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", nome, err)
		}
		for _, r := range regras {
			r.Arquivo = path.Base(nome)
			for _, p := range validar(r) {
				problemas = append(problemas, fmt.Sprintf("%s %s: %s", nome, r.ID, p))
			}
			if i, dup := c.porID[r.ID]; dup {
				problemas = append(problemas, fmt.Sprintf("%s %s: ID repetido (já em %s)", nome, r.ID, c.Regras[i].Arquivo))
				continue
			}
			c.porID[r.ID] = len(c.Regras)
			c.Regras = append(c.Regras, r)
		}
	}
	if len(problemas) > 0 {
		return nil, fmt.Errorf("catálogo inválido:\n  %s", strings.Join(problemas, "\n  "))
	}
	return c, nil
}

// decodificar é estrito: campo desconhecido ou documento extra é erro.
func decodificar(b []byte) ([]Regra, error) {
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var regras []Regra
	if err := dec.Decode(&regras); err != nil {
		return nil, err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("mais de um documento YAML no arquivo")
	}
	return regras, nil
}

func validar(r Regra) []string {
	var p []string
	if !reID.MatchString(r.ID) {
		p = append(p, "ID fora do padrão FLUXO-AREA-nnn")
	} else if pref := r.ID[:strings.Index(r.ID, "-")]; pref != r.Fluxo {
		p = append(p, fmt.Sprintf("fluxo %q diverge do prefixo do ID", r.Fluxo))
	}
	if !fluxosValidos[r.Fluxo] {
		p = append(p, fmt.Sprintf("fluxo inválido %q", r.Fluxo))
	}
	if strings.TrimSpace(r.Origem) == "" {
		p = append(p, "origem vazia")
	}
	if strings.TrimSpace(r.Enunciado) == "" {
		p = append(p, "enunciado vazio")
	}
	if len(r.Rotulo) == 0 {
		p = append(p, "sem rótulo")
	}
	for _, x := range r.Rotulo {
		if !rotulosValidos[x] {
			p = append(p, fmt.Sprintf("rótulo inválido %q", x))
		}
	}
	if r.Interpretacao != "" && (!reInterp.MatchString(r.Interpretacao) || !contem(r.Rotulo, Interpretacao)) {
		p = append(p, "interpretacao exige rótulo INTERPRETACAO e formato I-nn")
	}
	if len(r.Tipo) == 0 {
		p = append(p, "sem tipo")
	}
	for _, x := range r.Tipo {
		if !tiposValidos[x] {
			p = append(p, fmt.Sprintf("tipo inválido %q", x))
		}
	}
	if len(r.Testes) == 0 {
		p = append(p, "sem testes")
	}
	vistos := map[string]bool{}
	for _, t := range r.Testes {
		if !reTeste.MatchString(t) {
			p = append(p, fmt.Sprintf("ID de teste inválido %q", t))
		}
		if vistos[t] {
			p = append(p, fmt.Sprintf("teste repetido %q", t))
		}
		vistos[t] = true
	}
	for _, s := range r.EnforcedBy {
		if !reSimbolo.MatchString(s) {
			p = append(p, fmt.Sprintf("enforced_by %q não é pacote.Simbolo exportado", s))
		}
	}
	if r.InstrucaoIA != "" && !contem(r.Tipo, PoliticaIA) {
		p = append(p, "instrucao_ia só em regra POLITICA_IA")
	}
	if contem(r.Tipo, PoliticaIA) && r.Vigente() && (strings.TrimSpace(r.InstrucaoIA) == "" || len(r.Etapas) == 0) {
		p = append(p, "regra POLITICA_IA vigente sem instrucao_ia ou sem etapas")
	}
	for _, e := range r.Etapas {
		if !etapasValidas[e] {
			p = append(p, fmt.Sprintf("etapa inválida %q", e))
		}
	}
	for _, c := range r.Condicao {
		if !condValidas[c] {
			p = append(p, fmt.Sprintf("condição inválida %q", c))
		}
	}
	if r.Versao < 1 {
		p = append(p, "versao deve ser ≥ 1")
	}
	if r.Status != "" && r.Status != Vigente && r.Status != Revogada {
		p = append(p, fmt.Sprintf("status inválido %q", r.Status))
	}
	return p
}

func conjunto(xs ...string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func contem(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

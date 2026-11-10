// Pacote enquadramento é o catálogo curado de dispositivos para o
// enquadramento manual das imputações do PADS (V4 §48.5, PADS-ENQ-001).
//
//   - RDPM, arts. 15 (leve), 16 (média) e 17 (grave): a natureza é a do
//     artigo, fixa;
//   - Estatuto dos Policiais Militares (Decreto-Lei nº 09-A/1982), arts. 29
//     (preceitos da ética) e 32 (deveres): a natureza é atribuída pelo
//     usuário, por semelhança de intensidade com as transgressões do RDPM.
//
// O texto é o da redação vigente, com a fonte (arquivo e página) de cada
// item; o arquivo dispositivos.yaml exige revisão humana a cada mudança.
// O aplicativo nunca sugere dispositivo: o usuário escolhe, e a divergência
// com o citado no Memorando exige confirmação (V4 §22.1).
package enquadramento

import (
	"bytes"
	_ "embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"

	"github.com/DiogoBrazil/sys-executor/internal/rules/engine"
	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

//go:embed dispositivos.yaml
var dados []byte

// Diplomas do catálogo.
const (
	RDPM     = "RDPM"
	Estatuto = "ESTATUTO"
)

// NomeEstatuto é o nome por extenso usado no texto do dispositivo.
const NomeEstatuto = "Estatuto dos Policiais Militares (Decreto-Lei nº 09-A/1982)"

// Fonte do texto (para conferência).
type Fonte struct {
	Arquivo string `yaml:"arquivo" json:"arquivo"`
	Pagina  int    `yaml:"pagina" json:"pagina"`
}

// Dispositivo é um item do catálogo.
type Dispositivo struct {
	ID       string `yaml:"id" json:"id"`
	Diploma  string `yaml:"diploma" json:"diploma"`
	Artigo   int    `yaml:"artigo" json:"artigo"`
	Inciso   string `yaml:"inciso" json:"inciso"`
	Alinea   string `yaml:"alinea" json:"alinea"`
	Texto    string `yaml:"texto" json:"texto"`
	Natureza string `yaml:"natureza" json:"natureza"` // LEVE/MEDIA/GRAVE no RDPM; vazio no Estatuto
	Fonte    Fonte  `yaml:"fonte" json:"fonte"`
	Redacao  string `yaml:"redacao" json:"redacao"`
}

// Referencia: "art. 17, inciso V, do RDPM" (mesmo formato das imputações).
func (d Dispositivo) Referencia() string {
	s := "art. " + strconv.Itoa(d.Artigo) + ", inciso " + d.Inciso
	if d.Alinea != "" {
		s += ", alínea " + d.Alinea
	}
	if d.Diploma == RDPM {
		return s + ", do RDPM"
	}
	return s + ", do " + NomeEstatuto
}

// Rotulo para a lista de escolha: "art. 17, XXXIX — dirigir-se…".
func (d Dispositivo) Rotulo() string {
	s := "art. " + strconv.Itoa(d.Artigo) + ", " + d.Inciso
	if d.Alinea != "" {
		s += ", " + d.Alinea + ")"
	}
	return s + " — " + d.Texto
}

// NaturezaFixa: a natureza do RDPM; vazia no Estatuto (o usuário atribui).
func (d Dispositivo) NaturezaFixa() (engine.Natureza, bool) {
	switch d.Natureza {
	case "LEVE":
		return engine.Leve, true
	case "MEDIA":
		return engine.Media, true
	case "GRAVE":
		return engine.Grave, true
	}
	return "", false
}

type arquivo struct {
	Versao       int           `yaml:"versao"`
	Dispositivos []Dispositivo `yaml:"dispositivos"`
}

var (
	carregar sync.Once
	todos    []Dispositivo
	porID    map[string]Dispositivo
	errCarga error
)

func iniciar() {
	var a arquivo
	dec := yaml.NewDecoder(bytes.NewReader(dados))
	dec.KnownFields(true)
	if err := dec.Decode(&a); err != nil {
		errCarga = fmt.Errorf("catálogo de enquadramento ilegível: %w", err)
		return
	}
	porID = map[string]Dispositivo{}
	naturezaDoArtigo := map[int]string{15: "LEVE", 16: "MEDIA", 17: "GRAVE"}
	for _, d := range a.Dispositivos {
		switch {
		case d.ID == "" || d.Texto == "" || d.Artigo <= 0 || d.Inciso == "":
			errCarga = fmt.Errorf("dispositivo incompleto: %+v", d)
		case porID[d.ID].ID != "":
			errCarga = fmt.Errorf("dispositivo repetido: %s", d.ID)
		case d.Diploma == RDPM && naturezaDoArtigo[d.Artigo] != d.Natureza:
			errCarga = fmt.Errorf("%s: natureza %q não é a do art. %d do RDPM", d.ID, d.Natureza, d.Artigo)
		case d.Diploma == Estatuto && d.Natureza != "":
			errCarga = fmt.Errorf("%s: no Estatuto a natureza é atribuída pelo usuário", d.ID)
		case d.Diploma != RDPM && d.Diploma != Estatuto:
			errCarga = fmt.Errorf("%s: diploma %q desconhecido", d.ID, d.Diploma)
		}
		if errCarga != nil {
			return
		}
		porID[d.ID] = d
		todos = append(todos, d)
	}
}

// Todos devolve o catálogo na ordem do arquivo (artigo e inciso).
func Todos() ([]Dispositivo, error) {
	carregar.Do(iniciar)
	return todos, errCarga
}

// PorID devolve um dispositivo do catálogo.
func PorID(id string) (Dispositivo, bool) {
	carregar.Do(iniciar)
	d, ok := porID[id]
	return d, ok
}

// DoDiploma filtra pelo diploma.
func DoDiploma(diploma string) []Dispositivo {
	ds, _ := Todos()
	var out []Dispositivo
	for _, d := range ds {
		if d.Diploma == diploma {
			out = append(out, d)
		}
	}
	return out
}

// Buscar filtra pelo diploma e por termos (sem acento nem caixa), no
// número ("17", "XXXIX") e no texto.
func Buscar(diploma, termo string) []Dispositivo {
	ts := texto.Termos(termo)
	var out []Dispositivo
	for _, d := range DoDiploma(diploma) {
		alvo := " " + strings.Join(texto.Termos(d.Rotulo()), " ") + " "
		ok := true
		for _, t := range ts {
			if !strings.Contains(alvo, " "+t) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, d)
		}
	}
	return out
}

var (
	reRDPM     = regexp.MustCompile(`(?i)\bRDPM\b|regulamento\s+disciplinar`)
	reEstatuto = regexp.MustCompile(`(?i)estatuto|09-?a`)
)

// DoCitado encontra no catálogo o dispositivo que o Memorando cita (como a
// IA o extraiu), só para conferência: nunca é proposto ao usuário.
func DoCitado(diploma string, artigo int, inciso, alinea string) (Dispositivo, bool) {
	dip := ""
	switch {
	case reRDPM.MatchString(diploma):
		dip = RDPM
	case reEstatuto.MatchString(diploma):
		dip = Estatuto
	case diploma == "" && artigo >= 15 && artigo <= 17:
		dip = RDPM // o Memorando costuma citar só "art. 17, inciso V"
	default:
		return Dispositivo{}, false
	}
	id := map[string]string{RDPM: "RDPM", Estatuto: "EST"}[dip] + "-" + strconv.Itoa(artigo) + "-" + strings.ToUpper(strings.TrimSpace(inciso))
	if a := strings.ToLower(strings.TrimSpace(alinea)); a != "" {
		id += "-" + a
	}
	return PorID(id)
}

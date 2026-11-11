package legal

import (
	_ "embed"
	"fmt"

	"go.yaml.in/yaml/v3"
)

// StatusRevogadoPeloUsuario: o usuário considera o dispositivo revogado,
// sem que o ato revogador esteja na biblioteca (D-01).
const StatusRevogadoPeloUsuario = "CONSIDERADO_REVOGADO_PELO_USUARIO"

// NotaVigencia é o registro curado sobre a vigência de um dispositivo.
type NotaVigencia struct {
	Documento       string `yaml:"documento" json:"documento"`
	Artigo          string `yaml:"artigo" json:"artigo"`
	Paragrafo       string `yaml:"paragrafo" json:"paragrafo"`
	Inciso          string `yaml:"inciso" json:"inciso"`
	Alinea          string `yaml:"alinea" json:"alinea"`
	Status          string `yaml:"status" json:"status"`
	FonteRevogadora string `yaml:"fonte_revogadora" json:"fonte_revogadora"`
	Origem          string `yaml:"origem" json:"origem"`
}

//go:embed dados/notas_vigencia.yaml
var notasYAML []byte

// NotasVigencia devolve as notas curadas embutidas.
func NotasVigencia() ([]NotaVigencia, error) {
	var arq struct {
		Notas []NotaVigencia `yaml:"notas"`
	}
	if err := yaml.Unmarshal(notasYAML, &arq); err != nil {
		return nil, fmt.Errorf("notas de vigência: %w", err)
	}
	for _, n := range arq.Notas {
		if n.Documento == "" || n.Artigo == "" || n.Status == "" || n.Origem == "" {
			return nil, fmt.Errorf("notas de vigência: nota incompleta (%s art. %s)", n.Documento, n.Artigo)
		}
	}
	return arq.Notas, nil
}

// alcanca: a nota vale para a unidade (o próprio dispositivo ou o que está
// abaixo dele).
func (n NotaVigencia) alcanca(doc string, u Unidade) bool {
	if n.Documento != doc {
		return false
	}
	return dentroDe(u, Unidade{Artigo: n.Artigo, Paragrafo: n.Paragrafo, Inciso: n.Inciso, Alinea: n.Alinea})
}

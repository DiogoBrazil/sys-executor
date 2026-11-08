// Pacote sr: invariantes de escopo do relatório de Sindicância Regular
// (V4 §§23–25; plano §8).
//
// A portaria delimita fatos e sindicados. Fato novo ou militar novo só entra
// em III/IV depois de autorização expressa do usuário; a portaria não muda.
// Civis podem ser analisados em III, mas nunca são sujeitos da IV. A SR não
// tem sanção, pontuação nem reincidência: ConclusaoSR simplesmente não tem
// esses campos.
package sr

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrIDInvalido      = errors.New("identificador vazio")
	ErrIDRepetido      = errors.New("identificador repetido")
	ErrJaNaPortaria    = errors.New("já consta da portaria")
	ErrSemAutorizacao  = errors.New("autorização inexistente")
	ErrJaDecidida      = errors.New("autorização já decidida")
	ErrCivilNaoSujeito = errors.New("civil não pode ser sindicado")
)

// TipoPessoa distingue militar e civil.
type TipoPessoa string

const (
	Militar TipoPessoa = "MILITAR"
	Civil   TipoPessoa = "CIVIL"
)

// Pessoa envolvida nos autos.
type Pessoa struct {
	ID   string
	Tipo TipoPessoa
}

// Fato investigado.
type Fato struct {
	ID        string
	Descricao string
}

// TipoAutorizacao: inclusão de fato novo ou de militar novo (§24.3).
type TipoAutorizacao string

const (
	FatoNovo    TipoAutorizacao = "FATO_NOVO"
	MilitarNovo TipoAutorizacao = "MILITAR_NOVO"
)

// EstadoAutorizacao: PENDENTE → nada em III/IV; AUTORIZADO → incluído;
// RECUSADO → ausente.
type EstadoAutorizacao string

const (
	Pendente   EstadoAutorizacao = "PENDENTE"
	Autorizado EstadoAutorizacao = "AUTORIZADO"
	Recusado   EstadoAutorizacao = "RECUSADO"
)

// Autorizacao é o pedido mostrado ao usuário antes de incluir.
type Autorizacao struct {
	ID     string
	Tipo   TipoAutorizacao
	Ref    string // ID do fato ou da pessoa
	Resumo string
	Evid   []string
	Estado EstadoAutorizacao
}

// ObjetoSR é a delimitação da portaria mais as autorizações decididas.
type ObjetoSR struct {
	Portaria            string
	fatosOriginais      []Fato
	sindicadosOriginais []string

	fatos        map[string]Fato
	pessoas      map[string]Pessoa
	autorizacoes []Autorizacao
}

// NovoObjeto cria o objeto a partir da portaria. Sindicados precisam ser
// militares cadastrados em pessoas.
func NovoObjeto(portaria string, fatos []Fato, pessoas []Pessoa, sindicados []string) (*ObjetoSR, error) {
	o := &ObjetoSR{Portaria: portaria, fatos: map[string]Fato{}, pessoas: map[string]Pessoa{}}
	for _, p := range pessoas {
		if err := o.RegistrarPessoa(p); err != nil {
			return nil, err
		}
	}
	for _, f := range fatos {
		if err := o.registrarFato(f); err != nil {
			return nil, err
		}
		o.fatosOriginais = append(o.fatosOriginais, f)
	}
	vistos := map[string]bool{}
	for _, id := range sindicados {
		if vistos[id] {
			return nil, fmt.Errorf("%w: sindicado %s", ErrIDRepetido, id)
		}
		if err := o.exigirMilitar(id); err != nil {
			return nil, err
		}
		vistos[id] = true
		o.sindicadosOriginais = append(o.sindicadosOriginais, id)
	}
	return o, nil
}

func (o *ObjetoSR) registrarFato(f Fato) error {
	if strings.TrimSpace(f.ID) == "" {
		return ErrIDInvalido
	}
	if _, ok := o.fatos[f.ID]; ok {
		return fmt.Errorf("%w: fato %s", ErrIDRepetido, f.ID)
	}
	o.fatos[f.ID] = f
	return nil
}

// RegistrarPessoa cadastra uma pessoa dos autos (não a torna sindicada).
func (o *ObjetoSR) RegistrarPessoa(p Pessoa) error {
	if strings.TrimSpace(p.ID) == "" {
		return ErrIDInvalido
	}
	if p.Tipo != Militar && p.Tipo != Civil {
		return fmt.Errorf("tipo de pessoa inválido %q", p.Tipo)
	}
	if _, ok := o.pessoas[p.ID]; ok {
		return fmt.Errorf("%w: pessoa %s", ErrIDRepetido, p.ID)
	}
	o.pessoas[p.ID] = p
	return nil
}

func (o *ObjetoSR) exigirMilitar(id string) error {
	p, ok := o.pessoas[id]
	if !ok {
		return fmt.Errorf("pessoa %s não cadastrada", id)
	}
	if p.Tipo != Militar {
		return fmt.Errorf("%w: %s", ErrCivilNaoSujeito, id)
	}
	return nil
}

// ProporFatoNovo registra o fato fora da portaria e cria a autorização
// pendente (SR19). Nada entra em III/IV até a decisão.
func (o *ObjetoSR) ProporFatoNovo(f Fato, resumo string, evid []string) (Autorizacao, error) {
	if err := o.registrarFato(f); err != nil {
		return Autorizacao{}, err
	}
	return o.novaAutorizacao(FatoNovo, f.ID, resumo, evid), nil
}

// ProporMilitarNovo cria a autorização pendente para incluir um militar como
// sindicado (SR20), inclusive de outra OPM. Civil é recusado (§24.4).
func (o *ObjetoSR) ProporMilitarNovo(pessoaID, resumo string, evid []string) (Autorizacao, error) {
	if err := o.exigirMilitar(pessoaID); err != nil {
		return Autorizacao{}, err
	}
	for _, s := range o.sindicadosOriginais {
		if s == pessoaID {
			return Autorizacao{}, fmt.Errorf("%w: %s", ErrJaNaPortaria, pessoaID)
		}
	}
	return o.novaAutorizacao(MilitarNovo, pessoaID, resumo, evid), nil
}

func (o *ObjetoSR) novaAutorizacao(t TipoAutorizacao, ref, resumo string, evid []string) Autorizacao {
	a := Autorizacao{ID: fmt.Sprintf("AUT-%d", len(o.autorizacoes)+1), Tipo: t, Ref: ref, Resumo: resumo,
		Evid: append([]string(nil), evid...), Estado: Pendente}
	o.autorizacoes = append(o.autorizacoes, a)
	return a
}

// Decidir registra a resposta do usuário; uma decisão não é refeita aqui.
func (o *ObjetoSR) Decidir(id string, autorizado bool) error {
	for i := range o.autorizacoes {
		if o.autorizacoes[i].ID != id {
			continue
		}
		if o.autorizacoes[i].Estado != Pendente {
			return fmt.Errorf("%w: %s", ErrJaDecidida, id)
		}
		o.autorizacoes[i].Estado = Recusado
		if autorizado {
			o.autorizacoes[i].Estado = Autorizado
		}
		return nil
	}
	return fmt.Errorf("%w: %s", ErrSemAutorizacao, id)
}

// Autorizacoes devolve cópias.
func (o *ObjetoSR) Autorizacoes() []Autorizacao {
	out := append([]Autorizacao(nil), o.autorizacoes...)
	for i := range out {
		out[i].Evid = append([]string(nil), out[i].Evid...)
	}
	return out
}

func (o *ObjetoSR) autorizados(t TipoAutorizacao) []string {
	var out []string
	for _, a := range o.autorizacoes {
		if a.Tipo == t && a.Estado == Autorizado {
			out = append(out, a.Ref)
		}
	}
	return out
}

// FatosOriginais são os da portaria, intocados por autorizações.
func (o *ObjetoSR) FatosOriginais() []Fato { return append([]Fato(nil), o.fatosOriginais...) }

// SindicadosOriginais são os da portaria, intocados por autorizações.
func (o *ObjetoSR) SindicadosOriginais() []string {
	return append([]string(nil), o.sindicadosOriginais...)
}

// FatosAdmitidos: originais ∪ autorizados — os únicos em III e IV.
func (o *ObjetoSR) FatosAdmitidos() []string {
	var out []string
	for _, f := range o.fatosOriginais {
		out = append(out, f.ID)
	}
	return append(out, o.autorizados(FatoNovo)...)
}

// Sindicados: formais + autorizados (SR-EST-005); são os sujeitos da IV.
func (o *ObjetoSR) Sindicados() []string {
	return append(o.SindicadosOriginais(), o.autorizados(MilitarNovo)...)
}

// Resultado da conclusão (§25.2).
type Resultado string

const (
	Indicios     Resultado = "INDICIOS"
	Ausencia     Resultado = "AUSENCIA"
	Inconclusivo Resultado = "INCONCLUSIVO"
)

// TipoEnquadramento (§25.3).
type TipoEnquadramento string

const (
	CrimeComum   TipoEnquadramento = "CRIME_COMUM"
	CrimeMilitar TipoEnquadramento = "CRIME_MILITAR"
	Transgressao TipoEnquadramento = "TRANSGRESSAO"
)

// Enquadramento com diploma e artigo confirmados por fonte (LegalPassageID).
type Enquadramento struct {
	Tipo           TipoEnquadramento
	Diploma        string
	Artigo         string
	LegalPassageID string
}

// ConclusaoSR é a conclusão por fato × sindicado (SR-SOL-003).
type ConclusaoSR struct {
	FatoID         string
	PessoaID       string
	Resultado      Resultado
	Enquadramentos []Enquadramento
	Conjunta       string // grupo de conclusão conjunta (vazio = individual)
}

// Violacao é a quebra de uma invariante do escopo da SR.
type Violacao struct {
	Regra    string
	Ref      string
	Mensagem string
}

// ValidarConclusoes confere as invariantes de escopo da seção IV:
//   - fato ∈ originais ∪ autorizados (SR-ESC-001; SR19, SR35);
//   - sujeito ∈ sindicados originais ∪ autorizados (SR-ESC-002; SR20);
//   - civil nunca é sujeito (SR-ESC-003; SR21);
//   - uma conclusão por par fato × sindicado (SR-SOL-003);
//   - resultado e enquadramentos no domínio; indícios exigem enquadramento
//     e ausência não tem enquadramento (SR-SOL-004/005).
func (o *ObjetoSR) ValidarConclusoes(cs []ConclusaoSR) []Violacao {
	fatos := conjunto(o.FatosAdmitidos())
	sujeitos := conjunto(o.Sindicados())
	pares := map[[2]string]bool{}
	var vs []Violacao
	for _, c := range cs {
		ref := c.FatoID + "×" + c.PessoaID
		if !fatos[c.FatoID] {
			vs = append(vs, Violacao{"SR-ESC-001", ref, "fato fora da portaria sem autorização"})
		}
		if p, ok := o.pessoas[c.PessoaID]; ok && p.Tipo == Civil {
			vs = append(vs, Violacao{"SR-ESC-003", ref, "civil como sujeito da solução"})
		} else if !sujeitos[c.PessoaID] {
			vs = append(vs, Violacao{"SR-ESC-002", ref, "militar não sindicado (sem autorização) como sujeito da solução"})
		}
		par := [2]string{c.FatoID, c.PessoaID}
		if pares[par] {
			vs = append(vs, Violacao{"SR-SOL-003", ref, "mais de uma conclusão para o mesmo fato e sindicado"})
		}
		pares[par] = true
		switch c.Resultado {
		case Indicios:
			if len(c.Enquadramentos) == 0 {
				vs = append(vs, Violacao{"SR-SOL-005", ref, "indícios sem enquadramento específico"})
			}
		case Ausencia, Inconclusivo:
			if c.Resultado == Ausencia && len(c.Enquadramentos) > 0 {
				vs = append(vs, Violacao{"SR-SOL-004", ref, "ausência de indícios com enquadramento"})
			}
		default:
			vs = append(vs, Violacao{"SR-SOL-004", ref, fmt.Sprintf("resultado inválido %q", c.Resultado)})
		}
		for _, e := range c.Enquadramentos {
			if e.Tipo != CrimeComum && e.Tipo != CrimeMilitar && e.Tipo != Transgressao {
				vs = append(vs, Violacao{"SR-SOL-005", ref, fmt.Sprintf("tipo de enquadramento inválido %q", e.Tipo)})
			}
			if strings.TrimSpace(e.Diploma) == "" || strings.TrimSpace(e.Artigo) == "" || strings.TrimSpace(e.LegalPassageID) == "" {
				vs = append(vs, Violacao{"SR-SOL-005", ref, "enquadramento sem diploma, artigo ou fonte confirmada"})
			}
		}
	}
	return vs
}

func conjunto(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

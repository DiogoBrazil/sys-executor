package analise

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/segment"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/rules/catalog"
	"github.com/DiogoBrazil/sys-executor/internal/rules/civil"
	"github.com/DiogoBrazil/sys-executor/internal/rules/engine"
	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

// PessoaProposta é uma pessoa extraída (acusado, sindicado…).
type PessoaProposta struct {
	ID             string `json:"id"`
	Nome           string `json:"nome"`
	PostoGraduacao string `json:"postoGraduacao"`
	Matricula      string `json:"matricula"`
	Ponto          Ponto  `json:"ponto"`
}

// ImputacaoProposta é uma imputação extraída do memorando, a confirmar.
type ImputacaoProposta struct {
	ID         string          `json:"id"`
	Conduta    string          `json:"conduta"`
	Diploma    string          `json:"diploma"`
	Artigo     int             `json:"artigo"`
	Inciso     string          `json:"inciso"`
	Alinea     string          `json:"alinea"`
	Natureza   engine.Natureza `json:"natureza"` // "" = não informada no memorando
	Data       civil.Data      `json:"data"`     // zero = desconhecida
	DataOrigem string          `json:"dataOrigem"`
	Ponto      Ponto           `json:"ponto"`
	Pendencias []string        `json:"pendencias"`
}

// Dispositivo em texto ("art. 16, inciso XVI, alínea a, do RDPM").
func (i ImputacaoProposta) Dispositivo() string {
	if i.Artigo <= 0 {
		return ""
	}
	s := "art. " + strconv.Itoa(i.Artigo)
	if i.Inciso != "" {
		s += ", inciso " + i.Inciso
	}
	if i.Alinea != "" {
		s += ", alínea " + i.Alinea
	}
	if i.Diploma != "" {
		s += ", do " + i.Diploma
	}
	return s
}

// PropostaDelimitacaoPADS é o resultado da etapa 4 no PADS.
type PropostaDelimitacaoPADS struct {
	Peca       segment.Peca        `json:"peca"`
	Acusados   []PessoaProposta    `json:"acusados"`
	Imputacoes []ImputacaoProposta `json:"imputacoes"`
	Pendencias []string            `json:"pendencias"`
}

var naturezas = map[string]engine.Natureza{"LEVE": engine.Leve, "MEDIA": engine.Media, "GRAVE": engine.Grave}

type saidaDelimPADS struct {
	Acusados []struct {
		Nome           string `json:"nome"`
		PostoGraduacao string `json:"posto_graduacao"`
		Matricula      string `json:"matricula"`
		Trecho         string `json:"trecho_literal"`
	} `json:"acusados"`
	Imputacoes []struct {
		Conduta  string `json:"conduta"`
		Diploma  string `json:"diploma"`
		Artigo   int    `json:"artigo"`
		Inciso   string `json:"inciso"`
		Alinea   string `json:"alinea"`
		Natureza string `json:"natureza"`
		Dia      int    `json:"dia"`
		Mes      int    `json:"mes"`
		Ano      int    `json:"ano"`
		Trecho   string `json:"trecho_literal"`
	} `json:"imputacoes"`
}

func schemaDelimPADS() map[string]any {
	return objeto(map[string]any{
		"acusados": lista(objeto(map[string]any{
			"nome": str(120), "posto_graduacao": str(60), "matricula": str(30), "trecho_literal": str(400),
		}), 0, 5),
		"imputacoes": lista(objeto(map[string]any{
			"conduta": str(600), "diploma": str(80), "artigo": inteiro(0, 999), "inciso": str(12), "alinea": str(4),
			"natureza": enum("LEVE", "MEDIA", "GRAVE", "NAO_INFORMADA"),
			"dia":      inteiro(0, 31), "mes": inteiro(0, 12), "ano": inteiro(0, 2100), "trecho_literal": str(800),
		}), 0, 20),
	})
}

// Peça central do fluxo.
func pecaCentral(pecas []segment.Peca, f segment.Funcao) (segment.Peca, bool) {
	for _, p := range pecas {
		if p.Funcao == f {
			return p, true
		}
	}
	return segment.Peca{}, false
}

// tamanhoBloco estima quantas runas cabem num bloco para a etapa.
func tamanhoBloco(x *Executor, reserva int) int {
	livre := x.Motor.Contexto() - reserva - 2600 // instruções + dados + template
	if livre < 600 {
		livre = 600
	}
	return livre * 3
}

// DelimitarPADS extrai acusado(s) e imputações do Memorando Disciplinar
// (etapa 4). O Go confere cada trecho literal, o artigo citado, a natureza
// declarada e a data (regex sobre o trecho; D-09.4).
func DelimitarPADS(ctx context.Context, x *Executor, a *autos.Autos, pecas []segment.Peca, reg *Registro) (PropostaDelimitacaoPADS, error) {
	var r PropostaDelimitacaoPADS
	memo, ok := pecaCentral(pecas, segment.MemorandoDisciplinar)
	if !ok {
		return r, fmt.Errorf("memorando disciplinar não identificado")
	}
	r.Peca = memo
	const reserva = 1500
	max := tamanhoBloco(x, reserva)
	for {
		blocos := Blocos(a, memo, max, reg)
		r.Acusados, r.Imputacoes, r.Pendencias = nil, nil, nil
		err := delimitarBlocosPADS(ctx, x, a, blocos, reserva, &r)
		if policy.IsFatiar(err) && max > 1200 {
			max /= 2
			continue
		}
		if err != nil {
			return r, err
		}
		break
	}
	for i := range r.Imputacoes {
		r.Imputacoes[i].ID = fmt.Sprintf("IMP-%d", i+1)
	}
	for i := range r.Acusados {
		r.Acusados[i].ID = fmt.Sprintf("AC-%d", i+1)
	}
	if len(r.Imputacoes) == 0 {
		r.Pendencias = append(r.Pendencias, "Nenhuma imputação foi identificada com segurança no Memorando Disciplinar.")
	}
	return r, nil
}

func delimitarBlocosPADS(ctx context.Context, x *Executor, a *autos.Autos, blocos []Bloco, reserva int, r *PropostaDelimitacaoPADS) error {
	textoMemo := ""
	for _, b := range blocos {
		textoMemo += b.Texto + "\n"
	}
	for _, b := range blocos {
		var ultimo []ImputacaoProposta
		var acus []PessoaProposta
		err := x.Chamar(ctx, Chamada{
			Etapa: catalog.EtapaDelimitacao, Alvo: "Memorando Disciplinar",
			Evidencias: []policy.Evidencia{b.Evidencia()},
			Tarefa:     "liste as imputações e o acusado do Memorando Disciplinar acima.",
			NomeSchema: "delimitacao_pads", Schema: schemaDelimPADS(), Reserva: reserva,
		}, func(js []byte) error {
			var s saidaDelimPADS
			if err := decodificarEstrito(js, &s); err != nil {
				return err
			}
			ultimo, acus = nil, nil
			var falhas []string
			for k, im := range s.Imputacoes {
				p, ok := localizarNoBloco(a, b, im.Trecho)
				if !ok {
					falhas = append(falhas, fmt.Sprintf("o trecho_literal da imputação %d não aparece literalmente no memorando", k+1))
					continue
				}
				if err := conferirTextoLivre("conduta", im.Conduta, []string{b.Texto}); err != nil {
					falhas = append(falhas, err.Error())
					continue
				}
				prop := ImputacaoProposta{Conduta: strings.TrimSpace(im.Conduta), Diploma: strings.TrimSpace(im.Diploma),
					Artigo: im.Artigo, Inciso: strings.TrimSpace(im.Inciso), Alinea: strings.TrimSpace(im.Alinea), Ponto: p}
				if prop.Artigo > 0 && !artigoNoTexto(textoMemo, prop.Artigo) {
					prop.Pendencias = append(prop.Pendencias, fmt.Sprintf("O art. %d não foi encontrado no memorando; confirme o dispositivo.", prop.Artigo))
					prop.Artigo, prop.Inciso, prop.Alinea = 0, "", ""
				}
				conferirDispositivo(textoMemo, &prop)
				if prop.Artigo == 0 {
					prop.Pendencias = append(prop.Pendencias, "Dispositivo não identificado com segurança; informe-o.")
				}
				if n, ok := naturezas[im.Natureza]; ok && naturezaNoTexto(textoMemo, n) {
					prop.Natureza = n
				} else {
					prop.Pendencias = append(prop.Pendencias, PendenciaNatureza)
				}
				prop.Data, prop.DataOrigem = dataDoTrecho(p.Trecho, im.Dia, im.Mes, im.Ano, p.OCR)
				if prop.Data.Zero() {
					prop.Pendencias = append(prop.Pendencias, "Data do fato não identificada com segurança; informe-a.")
				}
				ultimo = append(ultimo, prop)
			}
			for k, ac := range s.Acusados {
				p, ok := localizarNoBloco(a, b, ac.Trecho)
				if !ok || strings.TrimSpace(ac.Nome) == "" || !strings.Contains(texto.SemAcentos(p.Trecho), texto.SemAcentos(strings.TrimSpace(ac.Nome))) {
					falhas = append(falhas, fmt.Sprintf("o acusado %d não tem trecho literal com o nome dele", k+1))
					continue
				}
				acus = append(acus, PessoaProposta{Nome: strings.TrimSpace(ac.Nome), PostoGraduacao: strings.TrimSpace(ac.PostoGraduacao),
					Matricula: strings.TrimSpace(ac.Matricula), Ponto: p})
			}
			if len(falhas) > 0 {
				return fmt.Errorf("%s", strings.Join(falhas, "; "))
			}
			return nil
		})
		if err != nil && !IsSemSeguranca(err) {
			return err
		}
		if err != nil {
			r.Pendencias = append(r.Pendencias, fmt.Sprintf("Parte do memorando (página %d) não foi lida com segurança pela IA; confira as imputações.", b.PaginaInicial()))
		}
		r.Imputacoes = mesclarImputacoes(r.Imputacoes, ultimo)
		r.Acusados = mesclarPessoas(r.Acusados, acus)
	}
	return nil
}

// Pendências da leitura que o pipeline pode dar por resolvidas.
const (
	PendenciaInciso   = "Inciso não confirmado no memorando; confira o dispositivo."
	PendenciaNatureza = "Natureza da transgressão não declarada no memorando; informe-a."
)

var (
	reRomano = regexp.MustCompile(`^[IVXLCDM]+$`)
	reLetra  = regexp.MustCompile(`^[a-z]$`)
	// Forma de diploma normativo (texto sem acentos, minúsculo).
	reDiploma = regexp.MustCompile(`^(rdpm|rdpm/ro|regulamento\b|estatuto\b|lei\b|decreto\b|codigo\b|cpm\b|cp\b|lc\b|constituicao\b)`)
)

// conferirDispositivo mantém inciso, alínea e diploma só quando aparecem no
// memorando (o modelo às vezes repete o inciso na alínea ou põe a natureza
// no diploma).
func conferirDispositivo(memo string, p *ImputacaoProposta) {
	if p.Artigo == 0 {
		p.Inciso, p.Alinea, p.Diploma = "", "", ""
		return
	}
	p.Inciso = strings.ToUpper(strings.TrimSpace(p.Inciso))
	if p.Inciso != "" {
		re := regexp.MustCompile(`(?i)(\binc(iso)?\.?\s*` + regexp.QuoteMeta(p.Inciso) + `\b|\bart(igo)?\.?\s*` + strconv.Itoa(p.Artigo) + `\s*,\s*` + regexp.QuoteMeta(p.Inciso) + `\b)`)
		if !reRomano.MatchString(p.Inciso) || !re.MatchString(memo) {
			p.Pendencias = append(p.Pendencias, PendenciaInciso)
			p.Inciso = ""
		}
	}
	p.Alinea = strings.ToLower(strings.Trim(strings.TrimSpace(p.Alinea), `"'“”`))
	if p.Alinea != "" {
		re := regexp.MustCompile(`(?i)\bal[ií]nea\s*["'“]?` + regexp.QuoteMeta(p.Alinea) + `\b`)
		if !reLetra.MatchString(p.Alinea) || !re.MatchString(memo) {
			p.Alinea = ""
		}
	}
	if d := strings.TrimSpace(p.Diploma); d != "" && (!reDiploma.MatchString(texto.SemAcentos(d)) || !strings.Contains(texto.SemAcentos(memo), texto.SemAcentos(d))) {
		p.Diploma = ""
	}
}

func artigoNoTexto(s string, art int) bool {
	re := regexp.MustCompile(`(?i)\bart(igo)?s?\.?\s*` + strconv.Itoa(art) + `\b`)
	return re.MatchString(s)
}

func naturezaNoTexto(s string, n engine.Natureza) bool {
	t := texto.SemAcentos(s)
	palavra := map[engine.Natureza]string{engine.Leve: "leve", engine.Media: "media", engine.Grave: "grave"}[n]
	return regexp.MustCompile(`\b` + palavra + `\b`).MatchString(t)
}

// dataDoTrecho aplica a D-09.4: as datas vêm do trecho literal (regex Go).
// Uma só data no trecho → ela. Várias → a da IA, se coincidir com uma delas.
// Nenhuma, ou trecho lido por OCR (D-09.3) → desconhecida (o usuário informa).
func dataDoTrecho(trecho string, dia, mes, ano int, ocr bool) (civil.Data, string) {
	if ocr {
		return civil.Data{}, ""
	}
	ds := autos.DatasNoTexto(trecho, autos.OrigemNativo)
	switch {
	case len(ds) == 1:
		return civil.D(ds[0].Dia, ds[0].Mes, ds[0].Ano), "trecho"
	case len(ds) > 1:
		for _, d := range ds {
			if d.Dia == dia && d.Mes == mes && d.Ano == ano {
				return civil.D(dia, mes, ano), "trecho"
			}
		}
	}
	return civil.Data{}, ""
}

func chaveImputacao(i ImputacaoProposta) string {
	return texto.SemAcentos(i.Ponto.Trecho)
}

func mesclarImputacoes(a, b []ImputacaoProposta) []ImputacaoProposta {
	vistos := map[string]bool{}
	for _, i := range a {
		vistos[chaveImputacao(i)] = true
	}
	for _, i := range b {
		if !vistos[chaveImputacao(i)] {
			a = append(a, i)
			vistos[chaveImputacao(i)] = true
		}
	}
	return a
}

func mesclarPessoas(a, b []PessoaProposta) []PessoaProposta {
	vistos := map[string]bool{}
	for _, p := range a {
		vistos[texto.SemAcentos(p.Nome)] = true
	}
	for _, p := range b {
		if k := texto.SemAcentos(p.Nome); !vistos[k] {
			a = append(a, p)
			vistos[k] = true
		}
	}
	return a
}

// ---- Sindicância Regular ----

// FatoProposto é um fato da portaria.
type FatoProposto struct {
	ID        string `json:"id"`
	Descricao string `json:"descricao"`
	Ponto     Ponto  `json:"ponto"`
}

// PropostaObjetoSR é o resultado da etapa 4 na SR.
type PropostaObjetoSR struct {
	Peca       segment.Peca     `json:"peca"`
	Numero     string           `json:"numero"`
	Autoridade string           `json:"autoridade"`
	Data       civil.Data       `json:"data"`
	Fatos      []FatoProposto   `json:"fatos"`
	Sindicados []PessoaProposta `json:"sindicados"`
	Pendencias []string         `json:"pendencias"`
}

type saidaDelimSR struct {
	Numero     string `json:"numero_portaria"`
	Autoridade string `json:"autoridade"`
	Dia        int    `json:"dia"`
	Mes        int    `json:"mes"`
	Ano        int    `json:"ano"`
	TrechoPort string `json:"trecho_literal_portaria"`
	Fatos      []struct {
		Descricao string `json:"descricao"`
		Trecho    string `json:"trecho_literal"`
	} `json:"fatos"`
	Sindicados []struct {
		Nome           string `json:"nome"`
		PostoGraduacao string `json:"posto_graduacao"`
		Matricula      string `json:"matricula"`
		Trecho         string `json:"trecho_literal"`
	} `json:"sindicados"`
}

func schemaDelimSR() map[string]any {
	return objeto(map[string]any{
		"numero_portaria": str(60), "autoridade": str(160),
		"dia": inteiro(0, 31), "mes": inteiro(0, 12), "ano": inteiro(0, 2100),
		"trecho_literal_portaria": str(400),
		"fatos":                   lista(objeto(map[string]any{"descricao": str(500), "trecho_literal": str(800)}), 0, 15),
		"sindicados": lista(objeto(map[string]any{
			"nome": str(120), "posto_graduacao": str(60), "matricula": str(30), "trecho_literal": str(400),
		}), 0, 20),
	})
}

// DelimitarSR extrai o objeto da Portaria de instauração (etapa 4).
func DelimitarSR(ctx context.Context, x *Executor, a *autos.Autos, pecas []segment.Peca, reg *Registro) (PropostaObjetoSR, error) {
	var r PropostaObjetoSR
	port, ok := pecaCentral(pecas, segment.PortariaSR)
	if !ok {
		return r, fmt.Errorf("portaria de instauração não identificada")
	}
	r.Peca = port
	const reserva = 1800
	max := tamanhoBloco(x, reserva)
	for {
		blocos := Blocos(a, port, max, reg)
		r = PropostaObjetoSR{Peca: port}
		err := delimitarBlocosSR(ctx, x, a, blocos, reserva, &r)
		if policy.IsFatiar(err) && max > 1200 {
			max /= 2
			continue
		}
		if err != nil {
			return r, err
		}
		break
	}
	for i := range r.Fatos {
		r.Fatos[i].ID = fmt.Sprintf("F-%d", i+1)
	}
	for i := range r.Sindicados {
		r.Sindicados[i].ID = fmt.Sprintf("S-%d", i+1)
	}
	if len(r.Fatos) == 0 {
		r.Pendencias = append(r.Pendencias, "Nenhum fato foi identificado com segurança na Portaria.")
	}
	if len(r.Sindicados) == 0 {
		r.Pendencias = append(r.Pendencias, "Nenhum sindicado foi identificado com segurança na Portaria.")
	}
	return r, nil
}

func delimitarBlocosSR(ctx context.Context, x *Executor, a *autos.Autos, blocos []Bloco, reserva int, r *PropostaObjetoSR) error {
	for _, b := range blocos {
		var parcial PropostaObjetoSR
		err := x.Chamar(ctx, Chamada{
			Etapa: catalog.EtapaDelimitacao, Alvo: "Portaria de instauração",
			Evidencias: []policy.Evidencia{b.Evidencia()},
			Tarefa:     "informe a identificação da portaria, os fatos a apurar e os sindicados.",
			NomeSchema: "delimitacao_sr", Schema: schemaDelimSR(), Reserva: reserva,
		}, func(js []byte) error {
			var s saidaDelimSR
			if err := decodificarEstrito(js, &s); err != nil {
				return err
			}
			parcial = PropostaObjetoSR{}
			var falhas []string
			if p, ok := localizarNoBloco(a, b, s.TrechoPort); ok {
				parcial.Numero, parcial.Autoridade = strings.TrimSpace(s.Numero), strings.TrimSpace(s.Autoridade)
				parcial.Data, _ = dataDoTrecho(p.Trecho, s.Dia, s.Mes, s.Ano, p.OCR)
			}
			for k, f := range s.Fatos {
				p, ok := localizarNoBloco(a, b, f.Trecho)
				if !ok {
					falhas = append(falhas, fmt.Sprintf("o trecho_literal do fato %d não aparece literalmente na portaria", k+1))
					continue
				}
				parcial.Fatos = append(parcial.Fatos, FatoProposto{Descricao: strings.TrimSpace(f.Descricao), Ponto: p})
			}
			for k, sd := range s.Sindicados {
				p, ok := localizarNoBloco(a, b, sd.Trecho)
				if !ok || strings.TrimSpace(sd.Nome) == "" || !strings.Contains(texto.SemAcentos(p.Trecho), texto.SemAcentos(strings.TrimSpace(sd.Nome))) {
					falhas = append(falhas, fmt.Sprintf("o sindicado %d não tem trecho literal com o nome dele", k+1))
					continue
				}
				parcial.Sindicados = append(parcial.Sindicados, PessoaProposta{Nome: strings.TrimSpace(sd.Nome),
					PostoGraduacao: strings.TrimSpace(sd.PostoGraduacao), Matricula: strings.TrimSpace(sd.Matricula), Ponto: p})
			}
			if len(falhas) > 0 {
				return fmt.Errorf("%s", strings.Join(falhas, "; "))
			}
			return nil
		})
		if err != nil && !IsSemSeguranca(err) {
			return err
		}
		if err != nil {
			r.Pendencias = append(r.Pendencias, fmt.Sprintf("Parte da portaria (página %d) não foi lida com segurança pela IA; confira o objeto.", b.PaginaInicial()))
		}
		if r.Numero == "" {
			r.Numero, r.Autoridade = parcial.Numero, parcial.Autoridade
		}
		if r.Data.Zero() {
			r.Data = parcial.Data
		}
		vistos := map[string]bool{}
		for _, f := range r.Fatos {
			vistos[texto.SemAcentos(f.Ponto.Trecho)] = true
		}
		for _, f := range parcial.Fatos {
			if !vistos[texto.SemAcentos(f.Ponto.Trecho)] {
				r.Fatos = append(r.Fatos, f)
			}
		}
		r.Sindicados = mesclarPessoas(r.Sindicados, parcial.Sindicados)
	}
	return nil
}

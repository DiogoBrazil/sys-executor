package minuta

import (
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/pads"
	"github.com/DiogoBrazil/sys-executor/internal/rules/engine"
)

// Dados fictícios.
func entradaComPunicao() EntradaPADS {
	par := func(t string) analise.Trecho {
		return analise.Trecho{Paragrafos: []analise.Paragrafo{{Texto: t, Evid: []string{"EV-0001"}}}}
	}
	fat := make([]analise.FatorArt36, 7)
	for i := range fat {
		fat[i] = analise.FatorArt36{Inciso: i + 1, Texto: analise.AusenciaDeElementos}
	}
	fat[2] = analise.FatorArt36{Inciso: 3, Texto: "A falta atingiu a regularidade do serviço.", Evid: []string{"EV-0001"}}
	return EntradaPADS{
		Acusado: "SD PM FULANO FICTÍCIO", Matricula: "000000000", Memorando: "Memorando Disciplinar (SEI nº 0090000001)",
		Global: pads.ProcedenciaParcial, Relatorio: par("A testemunha declarou que o acusado não compareceu ao serviço."),
		Imputacoes: []ImputacaoPADS{
			{ID: "IMP-1", Conduta: "Faltou ao serviço.", Dispositivo: "art. 16, inciso XVI, do RDPM", Natureza: engine.Media, Procedente: true, Fundamentacao: par("A prova confirma a ausência.")},
			{ID: "IMP-2", Conduta: "Chegou atrasado.", Dispositivo: "art. 17, inciso II, do RDPM", Natureza: engine.Leve, Fundamentacao: par("A prova não confirma o atraso.")},
		},
		Sancoes: []SancaoPADS{{ImputacaoID: "IMP-1", Dispositivo: "art. 16, inciso XVI, do RDPM", Natureza: engine.Media, Sancao: engine.Detencao, Dias: 2, Fatores: fat}},
		Pontuacao: engine.ResultadoPontuacao{Concluido: true, ComportamentoAnterior: engine.Bom, ComportamentoFinal: engine.Bom, PontosAnteriores: 25, PontosFinais: 23,
			Descontos: []engine.DescontoAplicado{{Pontos: 2}}},
	}
}

var evidencias = map[string]bool{"EV-0001": true}

// Regra: TRV-VAL-004, TRV-VAL-006, TRV-VAL-007, TRV-VAL-008
func TestValidarPADSAceitaAMinutaComposta(t *testing.T) {
	e := entradaComPunicao()
	if as := ValidarPADS(ComporPADS(e), e, evidencias); len(as) != 0 {
		t.Fatalf("minuta composta devia passar: %+v", as)
	}
	// Absolvição: sem punição, sem cláusula de residência, sem nota de punição.
	a := entradaComPunicao()
	a.Imputacoes[0].Procedente, a.Sancoes, a.Global, a.Pontuacao = false, nil, pads.Improcedencia, engine.ResultadoPontuacao{}
	d := ComporPADS(a)
	if as := ValidarPADS(d, a, evidencias); len(as) != 0 {
		t.Fatalf("absolvição devia passar: %+v", as)
	}
	if txt := d.Texto(); !strings.Contains(txt, "ABSOLVER SD PM FULANO FICTÍCIO") || strings.Contains(txt, "nota de punição") || strings.Contains(txt, "Lei nº 2.484") || strings.Contains(txt, "imponho") {
		t.Fatalf("texto da absolvição:\n%s", txt)
	}
}

// Cada adulteração da minuta tem de ser apanhada pelo validador certo.
// Regra: TRV-VAL-004, TRV-VAL-006, TRV-VAL-007, TRV-VAL-008, TRV-VAL-010, TRV-VAL-014, TRV-EXP-003
func TestValidarPADSApontaDivergencias(t *testing.T) {
	trocar := func(de, para string) func(*Documento, *EntradaPADS) {
		return func(d *Documento, _ *EntradaPADS) {
			for i := range d.Blocos {
				d.Blocos[i].Texto = strings.Replace(d.Blocos[i].Texto, de, para, 1)
			}
		}
	}
	casos := []struct {
		nome, regra string
		mudar       func(*Documento, *EntradaPADS)
	}{
		{"seção fora de ordem", "estrutura", trocar("2. FUNDAMENTAÇÃO", "2. MOTIVAÇÃO")},
		{"resultado trocado no texto", "resultado", trocar("do RDPM: IMPROCEDENTE.", "do RDPM: PROCEDENTE.")},
		{"resultado global trocado", "resultado", trocar("Julgar PARCIALMENTE PROCEDENTE", "Julgar PROCEDENTE")},
		{"dias diferentes do cálculo", "calculo", trocar("2 (dois) dias de DETENÇÃO", "5 (cinco) dias de DETENÇÃO")},
		{"cálculo mudou depois da composição", "calculo", func(_ *Documento, e *EntradaPADS) { e.Sancoes[0].Dias = 3 }},
		{"pontos diferentes do cálculo", "calculo", trocar("de 25 para 23 pontos", "de 25 para 20 pontos")},
		{"natureza trocada", "calculo", trocar("natureza MÉDIA", "natureza GRAVE")},
		{"punição a mais", "calculo", func(d *Documento, _ *EntradaPADS) {
			d.add(Paragrafo, "Além disso, imponho a ele a punição de PRISÃO.")
		}},
		{"cláusula de residência retirada", "texto_fixo", func(d *Documento, _ *EntradaPADS) {
			for i := range d.Blocos {
				if d.Blocos[i].Trecho == "clausula_residencial" {
					d.Blocos[i].Trecho, d.Blocos[i].Texto = "", "."
				}
			}
		}},
		{"nota de punição retirada", "texto_fixo", trocar("nota de punição somente depois", "nota somente depois")},
		{"art. 41, § 3º", "forma", trocar("A prova confirma a ausência.", "A prova confirma a ausência, na forma do art. 41, § 3º, do RDPM.")},
		{"identificador interno no texto", "forma", trocar("A prova confirma a ausência.", "A prova (EV-0001) confirma a ausência.")},
		{"página citada pela IA", "forma", trocar("A prova confirma a ausência.", "A prova de fls. 12 confirma a ausência.")},
		{"assinatura simulada", "forma", trocar("CUMPRA-SE.", "CUMPRA-SE. Documento assinado eletronicamente.")},
		{"evidência inexistente", "evidencia", func(d *Documento, _ *EntradaPADS) {
			for i := range d.Blocos {
				if d.Blocos[i].Trecho == "relatorio" {
					d.Blocos[i].Evid = []string{"EV-9999"}
				}
			}
		}},
		{"parágrafo da IA sem evidência", "evidencia", func(d *Documento, _ *EntradaPADS) {
			for i := range d.Blocos {
				if d.Blocos[i].Trecho == "fundamentacao:IMP-1" {
					d.Blocos[i].Evid = nil
				}
			}
		}},
	}
	for _, c := range casos {
		e := entradaComPunicao()
		d := ComporPADS(e)
		c.mudar(d, &e)
		as := ValidarPADS(d, e, evidencias)
		achou := false
		for _, a := range as {
			achou = achou || a.Regra == c.regra
		}
		if !achou {
			t.Errorf("%s: esperava achado %q, veio %+v", c.nome, c.regra, as)
		}
	}
}

package minuta

import (
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/pads"
	"github.com/DiogoBrazil/sys-executor/internal/domain/sr"
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

func entradaSR() (EntradaSR, *sr.ObjetoSR) {
	tr := func(t string) analise.Trecho {
		return analise.Trecho{Paragrafos: []analise.Paragrafo{{Texto: t, Evid: []string{"EV-1"}}}}
	}
	transg := EnquadramentoSR{Tipo: sr.Transgressao, Referencia: "art. 17, inciso XXXVIII, do RDPM", Texto: "deixar de zelar pelo material"}
	crime := EnquadramentoSR{Tipo: sr.CrimeMilitar, Referencia: "art. 303 do Código Penal Militar", Rubrica: "Peculato"}
	e := EntradaSR{
		Portaria:    "Portaria nº 55/2026 (SEI nº 0012345)",
		Sindicados:  []PessoaSR{{Nome: "CB PM FULANO TESTE", SEI: "0012350"}, {Nome: "SD PM BELTRANO TESTE"}},
		Ofendidos:   []string{"Administração Pública Militar"},
		Testemunhas: []PessoaSR{{Nome: "SGT PM CICRANO TESTE", SEI: "0012360"}},
		Sintese:     tr("A Portaria determina a apuração do extravio de uma arma da reserva."),
		Fatos: []FatoSR{
			{ID: "F-1", Descricao: "extravio de arma", Analise: tr("A testemunha situa o primeiro sindicado na reserva."),
				Conclusoes: []ConclusaoSR{{SindicadoID: "S-1", Sindicado: "CB PM FULANO TESTE", Indicios: true, Enquadramentos: []EnquadramentoSR{crime, transg}},
					{SindicadoID: "S-2", Sindicado: "SD PM BELTRANO TESTE"}}},
			{ID: "F-2", Descricao: "dano em viatura", Analise: tr("Não há prova de quem danificou a viatura."),
				Conclusoes: []ConclusaoSR{{SindicadoID: "S-1", Sindicado: "CB PM FULANO TESTE"}, {SindicadoID: "S-2", Sindicado: "SD PM BELTRANO TESTE"}}},
		},
	}
	obj, err := sr.NovoObjeto("55/2026", []sr.Fato{{ID: "F-1"}, {ID: "F-2"}},
		[]sr.Pessoa{{ID: "S-1", Tipo: sr.Militar}, {ID: "S-2", Tipo: sr.Militar}}, []string{"S-1", "S-2"})
	if err != nil {
		panic(err)
	}
	return e, obj
}

// Regra: SR-SOL-003, SR-SOL-004, SR-SOL-005
func TestComporEValidarSR(t *testing.T) {
	e, obj := entradaSR()
	d := ComporSR(e)
	ev := map[string]bool{"EV-1": true}
	if as := ValidarSR(d, e, obj, ev); len(as) != 0 {
		t.Fatalf("achados na minuta composta: %+v\n%s", as, d.Texto())
	}
	txt := d.Texto()
	for _, quer := range []string{
		"a. Portaria: Portaria nº 55/2026 (SEI nº 0012345).",
		"b. Sindicados: CB PM FULANO TESTE (SEI nº 0012350); SD PM BELTRANO TESTE.",
		"c. Ofendido: Administração Pública Militar.",
		"d. Testemunhas ouvidas: SGT PM CICRANO TESTE (SEI nº 0012360).",
		"Fato 1 – extravio de arma", "Fato 2 – dano em viatura",
		"a) CB PM FULANO TESTE.",
		"Quanto ao fato 1, há indícios da prática de crime militar previsto no art. 303 do Código Penal Militar (peculato); " +
			"há indícios da prática de transgressão disciplinar prevista no art. 17, inciso XXXVIII, do RDPM – “deixar de zelar pelo material”; " +
			"não foram identificados indícios de crime comum.",
		"Quanto ao fato 2, não foram identificados indícios suficientes de crime comum, crime militar ou transgressão disciplinar.",
	} {
		if !strings.Contains(txt, quer) {
			t.Errorf("sem %q:\n%s", quer, txt)
		}
	}
	// Sem perícia identificada: só o rótulo, com a pendência (V4 §23.2).
	if !strings.Contains(txt, "e. Perícias realizadas:\n") || len(d.Pendencias) == 0 {
		t.Errorf("alínea e sem pendência:\n%s\n%v", txt, d.Pendencias)
	}
}

// Um fato e o mesmo resultado para todos: conclusão conjunta (V4 §25.1).
func TestSolucaoSRConjunta(t *testing.T) {
	e, _ := entradaSR()
	e.Fatos = e.Fatos[1:]
	d := ComporSR(e)
	if !strings.Contains(d.Texto(), "Quanto ao fato investigado, em relação ao CB PM FULANO TESTE e ao SD PM BELTRANO TESTE, não foram identificados indícios suficientes") ||
		!strings.Contains(d.Texto(), "examinadas individualmente") {
		t.Fatalf("conclusão conjunta:\n%s", d.Texto())
	}
}

// Regra: SR-SOL-004, SR-SOL-008
func TestValidarSRApontaDivergencias(t *testing.T) {
	ev := map[string]bool{"EV-1": true}
	casos := []struct {
		nome    string
		mudar   func(*Documento, *EntradaSR)
		esperar string
	}{
		{"sem a seção III", func(d *Documento, _ *EntradaSR) {
			for i, b := range d.Blocos {
				if b.Texto == SecaoAnaliseSR {
					d.Blocos = append(d.Blocos[:i], d.Blocos[i+1:]...)
					return
				}
			}
		}, "seções estão fora de ordem"},
		{"inconclusivo na solução", func(d *Documento, _ *EntradaSR) {
			d.Blocos = append(d.Blocos[:len(d.Blocos)-2], append([]Bloco{{Tipo: Paragrafo, Texto: "Quanto ao fato 3, o resultado é inconclusivo.", Origem: Fixo}}, d.Blocos[len(d.Blocos)-2:]...)...)
		}, "inconclusivo"},
		{"providência automática", func(d *Documento, _ *EntradaSR) {
			d.Blocos = append(d.Blocos[:len(d.Blocos)-2], append([]Bloco{{Tipo: Paragrafo, Texto: "Sugiro o arquivamento.", Origem: Fixo}}, d.Blocos[len(d.Blocos)-2:]...)...)
		}, "providência"},
		{"enquadramento confirmado fora da solução", func(_ *Documento, e *EntradaSR) {
			e.Fatos[0].Conclusoes[0].Enquadramentos = append(e.Fatos[0].Conclusoes[0].Enquadramentos,
				EnquadramentoSR{Tipo: sr.CrimeComum, Referencia: "art. 312 do Código Penal"})
		}, "art. 312 do Código Penal"},
		{"ausência com enquadramento", func(_ *Documento, e *EntradaSR) {
			e.Fatos[1].Conclusoes[0].Enquadramentos = []EnquadramentoSR{{Tipo: sr.Transgressao, Referencia: "art. 17, inciso XXXVIII, do RDPM"}}
		}, "ausência de indícios com enquadramento"},
		{"parágrafo da IA sem evidência", func(d *Documento, _ *EntradaSR) {
			for i := range d.Blocos {
				if d.Blocos[i].Origem == IA {
					d.Blocos[i].Evid = nil
					return
				}
			}
		}, "não cita evidência"},
		{"identificador interno", func(d *Documento, _ *EntradaSR) {
			for i := range d.Blocos {
				if d.Blocos[i].Origem == IA {
					d.Blocos[i].Texto += " (EV-1)"
					return
				}
			}
		}, "identificador interno"},
	}
	for _, c := range casos {
		e, obj := entradaSR()
		d := ComporSR(e)
		c.mudar(d, &e)
		var msgs []string
		for _, a := range ValidarSR(d, e, obj, ev) {
			msgs = append(msgs, a.Mensagem)
		}
		if !strings.Contains(strings.Join(msgs, " | "), c.esperar) {
			t.Errorf("%s: esperado %q em %v", c.nome, c.esperar, msgs)
		}
	}
}

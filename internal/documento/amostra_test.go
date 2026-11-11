package documento

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/analise"
	"github.com/DiogoBrazil/sys-executor/internal/domain/pads"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/pdf"
	"github.com/DiogoBrazil/sys-executor/internal/minuta"
	"github.com/DiogoBrazil/sys-executor/internal/rules/engine"
)

// minutaFicticia compõe um julgamento inteiro com dados inventados.
func minutaFicticia() *minuta.Documento {
	par := func(ts ...string) analise.Trecho {
		var t analise.Trecho
		for _, x := range ts {
			t.Paragrafos = append(t.Paragrafos, analise.Paragrafo{Texto: x, Evid: []string{"EV-0001"}})
		}
		return t
	}
	fat := make([]analise.FatorArt36, 7)
	for i := range fat {
		fat[i] = analise.FatorArt36{Inciso: i + 1, Texto: analise.AusenciaDeElementos}
	}
	fat[2] = analise.FatorArt36{Inciso: 3, Texto: "A falta atingiu a regularidade do serviço para o qual o acusado estava escalado.", Evid: []string{"EV-0001"}}
	return minuta.ComporPADS(minuta.EntradaPADS{
		Acusado: "SD PM FULANO FICTÍCIO", Matricula: "000000000", Memorando: "Memorando Disciplinar (SEI nº 0090000001)",
		Global: pads.ProcedenciaParcial,
		Relatorio: par("Segundo o Memorando Disciplinar (SEI nº 0090000001), o acusado teria faltado ao serviço para o qual estava escalado e chegado atrasado à instrução do dia seguinte.",
			"Na defesa prévia (SEI nº 0090000002), o acusado sustentou que estava doente e que apresentaria atestado médico.",
			"No Termo de Inquirição (SEI nº 0090000003), a testemunha declarou que o acusado não compareceu ao quartel no dia dos fatos e não avisou o chefe da guarnição."),
		Imputacoes: []minuta.ImputacaoPADS{
			{ID: "IMP-1", Conduta: "Faltou ao serviço para o qual estava escalado.", Dispositivo: "art. 16, inciso XVI, do RDPM", Natureza: engine.Media, Procedente: true,
				TextoDispositivo: "faltar a qualquer ato de serviço em que deva tomar parte ou assistir;",
				Fundamentacao:    par("A prova testemunhal confirma a ausência ao serviço. O atestado médico mencionado pela defesa não foi juntado aos autos, de modo que a justificativa não está demonstrada.")},
			{ID: "IMP-2", Conduta: "Chegou atrasado à instrução.", Dispositivo: "art. 17, inciso II, do RDPM", Natureza: engine.Leve},
		},
		Sancoes: []minuta.SancaoPADS{{ImputacaoID: "IMP-1", Dispositivo: "art. 16, inciso XVI, do RDPM", Natureza: engine.Media, Sancao: engine.Detencao, Dias: 2, Fatores: fat,
			AtenuantesContadas: []string{"art. 38, I"}}},
		Pontuacao: engine.ResultadoPontuacao{Concluido: true, ComportamentoAnterior: engine.Bom, ComportamentoFinal: engine.Bom, PontosAnteriores: 25, PontosFinais: 23,
			Descontos: []engine.DescontoAplicado{{Pontos: 2}}},
		RotulosArt36: [7]string{"a personalidade do transgressor e seus antecedentes disciplinares;", "os motivos determinantes;", "a natureza dos fatos e/ou dos atos que a envolveram;",
			"a intensidade do dolo ou o grau da culpa;", "o dano ou o perigo de dano;", "as circunstâncias atenuantes e agravantes;", "as conseqüências da transgressão."},
	})
}

// TestAmostra grava o DOCX, o PDF e as páginas do PDF em PNG na pasta de
// SYSEXEC_AMOSTRA, para conferência visual durante o desenvolvimento.
func TestAmostra(t *testing.T) {
	dir := os.Getenv("SYSEXEC_AMOSTRA")
	if dir == "" {
		t.Skip("defina SYSEXEC_AMOSTRA com a pasta de saída")
	}
	doc := minutaFicticia()
	cab := CarregarCabecalho(os.Getenv("SYSEXEC_COMPONENTES"))
	b, err := DOCX(doc, cab)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "amostra.docx"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	ty, err := LocalizarTypst(os.Getenv("SYSEXEC_COMPONENTES"))
	if err != nil {
		t.Skip("sem Typst: só o DOCX foi gravado")
	}
	p, err := ty.PDF(context.Background(), doc, cab, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "amostra.pdf"), p, 0o600); err != nil {
		t.Fatal(err)
	}
	ex := pdf.NovoExtrator()
	defer ex.Fechar()
	d, err := ex.Abrir(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Fechar()
	for i := 0; i < d.Paginas(); i++ {
		im, err := d.Renderizar(context.Background(), i, 80)
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, fmt.Sprintf("amostra-%d.png", i+1)))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, im)
		f.Close()
	}
}

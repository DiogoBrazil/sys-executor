package segment

import (
	"strings"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
)

// pag cria uma página nativa com o texto dado (rodapé SEI opcional).
func pag(n int, texto, sei string) autos.Pagina {
	if sei != "" {
		texto += "\nReferência: Processo nº 0000.000001/2026-01 SEI nº " + sei
	}
	return autos.Pagina{Numero: n, Texto: texto, Situacao: autos.Lida, Origem: autos.OrigemNativo}
}

// Espelho textual da fixture testdata/autos/sei-sintetico.pdf.
func autosSinteticos() *autos.Autos {
	return &autos.Autos{Paginas: []autos.Pagina{
		pag(1, "MEMORANDO DISCIPLINAR Nº 1/2026\nAo SD PM FULANO ALFA FICTÍCIO…", "0090000001"),
		pag(2, "NOTIFICAÇÃO DE INQUIRIÇÃO\nAudiência por videoconferência, com gravação em mídia.", "0090000002"),
		pag(3, "MANIFESTAÇÃO\nO acusado, por seu defensor, apresenta manifestação…", "0090000003"),
		{Numero: 4, Texto: "TERMO DE INQUIRIÇÃO DE TESTEMUNHA\nAs câmeras do quartel registraram a saída e a gravação foi entregue.", Situacao: autos.OCROk, Origem: autos.OrigemOCR},
		pag(5, "ATA DE PRESENÇA POR VIDEOCONFERÊNCIA\nGravação disponível em link externo.", "0090000004"),
		{Numero: 6, Situacao: autos.Vazia},
		pag(7, "TERMO DE INQUIRIÇÃO DE TESTEMUNHA\nA testemunha declarou…", "0090000005"),
		pag(8, "continuação do depoimento… Nada mais disse.", "0090000005"),
		pag(9, "MANIFESTAÇÃO\nA defesa reitera a negativa dos fatos…", "0090000006"),
	}}
}

func TestSegmentarFixtureSEI(t *testing.T) {
	ps := Segmentar(autosSinteticos())
	type esp struct {
		ini, fim int
		sei      string
		f        Funcao
		base     Base
		provavel bool
		externo  bool
	}
	esperado := []esp{
		{1, 1, "0090000001", MemorandoDisciplinar, BaseTitulo, false, false},
		{2, 2, "0090000002", Notificacao, BaseTitulo, false, false},
		{3, 3, "0090000003", DefesaPrevia, BaseCronologia, true, false},     // A01/D1
		{4, 4, "", TermoInquiricao, BaseTitulo, false, true},                // escaneado, externo
		{5, 6, "0090000004", AtaVideoconferencia, BaseTitulo, false, false}, // página em branco junta à anterior
		{7, 8, "0090000005", TermoInquiricao, BaseTitulo, false, false},     // 2 páginas, mesmo SEI
		{9, 9, "0090000006", AlegacoesFinais, BaseCronologia, true, false},  // A02/D2
	}
	if len(ps) != len(esperado) {
		t.Fatalf("peças: %+v", ps)
	}
	for i, e := range esperado {
		p := ps[i]
		if p.PaginaInicial != e.ini || p.PaginaFinal != e.fim || p.SEI != e.sei || p.Funcao != e.f ||
			p.Base != e.base || p.Provavel != e.provavel || p.Externo != e.externo {
			t.Errorf("peça %d: %+v, esperado %+v", i+1, p, e)
		}
	}
}

func TestManifestacaoEntreTermosFicaIndeterminada(t *testing.T) {
	a := &autos.Autos{Paginas: []autos.Pagina{
		pag(1, "MEMORANDO DISCIPLINAR", "1000001"),
		pag(2, "TERMO DE INQUIRIÇÃO", "1000002"),
		pag(3, "Manifestação da defesa", "1000003"),
		pag(4, "TERMO DE DECLARAÇÕES", "1000004"),
	}}
	ps := Segmentar(a)
	if ps[2].Funcao != Indeterminada || ps[2].Base != BaseCronologia {
		t.Fatalf("manifestação entre termos deve ficar indeterminada: %+v", ps[2])
	}
	// Sem termo algum, a cronologia não decide.
	b := &autos.Autos{Paginas: []autos.Pagina{pag(1, "MEMORANDO DISCIPLINAR", "1"+"000001"), pag(2, "MANIFESTAÇÃO", "1000002")}}
	if p := Segmentar(b)[1]; p.Funcao != Indeterminada {
		t.Fatalf("%+v", p)
	}
}

func TestCompatibilidade(t *testing.T) {
	a := autosSinteticos()
	ps := Segmentar(a)
	if c := VerificarCompatibilidade("PADS", ps, a); c.Resultado != Compativel || c.PecaCentral != "PEC-01" {
		t.Fatalf("PADS: %+v", c)
	}
	// T04: autos de SR com tipo PADS → incompatível, sem trocar o tipo.
	sr := &autos.Autos{Paginas: []autos.Pagina{
		pag(1, "PORTARIA Nº 5/2026\nInstaura Sindicância Regular para apurar…", "2000001"),
		pag(2, "TERMO DE DECLARAÇÕES", "2000002"),
	}}
	if c := VerificarCompatibilidade("PADS", Segmentar(sr), sr); c.Resultado != Incompativel || c.TipoAparente != "SR" {
		t.Fatalf("T04: %+v", c)
	}
	if c := VerificarCompatibilidade("SR", Segmentar(sr), sr); c.Resultado != Compativel {
		t.Fatalf("SR: %+v", c)
	}
	// T05: PADS com portaria de SR anexada depois do memorando → continua PADS.
	t05 := &autos.Autos{Paginas: append(append([]autos.Pagina{}, a.Paginas...),
		pag(10, "PORTARIA Nº 9/2025\nInstaura Sindicância Regular (cópia anexada como prova)", "3000001"))}
	if c := VerificarCompatibilidade("PADS", Segmentar(t05), t05); c.Resultado != Compativel {
		t.Fatalf("T05: %+v", c)
	}
	// T06: peça inicial ilegível e sem memorando reconhecível → pede versão legível.
	t06 := &autos.Autos{Paginas: []autos.Pagina{
		{Numero: 1, Texto: "M3M0R4ND0 ?? d1sc...", Situacao: autos.Ilegivel, Origem: autos.OrigemOCR},
		pag(2, "TERMO DE INQUIRIÇÃO", "4000002"),
	}}
	if c := VerificarCompatibilidade("PADS", Segmentar(t06), t06); c.Resultado != PecaIlegivel {
		t.Fatalf("T06: %+v", c)
	}
	vazio := &autos.Autos{Paginas: []autos.Pagina{pag(1, "DESPACHO", "5000001")}}
	if c := VerificarCompatibilidade("PADS", Segmentar(vazio), vazio); c.Resultado != PecaAusente {
		t.Fatalf("ausente: %+v", c)
	}
}

func TestDetectarMidias(t *testing.T) {
	a := autosSinteticos()
	a.Paginas = append(a.Paginas, pag(10, "Juntou-se o áudio da oitiva e o vídeo da câmera.", "9"+"000001"))
	ms := DetectarMidias(a, Segmentar(a))
	termos := []string{}
	for _, m := range ms {
		termos = append(termos, m.Termo)
		r := []rune(a.Paginas[m.Pagina-1].Texto)
		if string(r[m.Inicio:m.Fim]) != m.Termo {
			t.Errorf("posição errada para %q", m.Termo)
		}
	}
	j := strings.Join(termos, "|")
	for _, esperado := range []string{"videoconferência", "gravação", "mídia", "câmeras", "link externo", "áudio", "vídeo", "câmera"} {
		if !strings.Contains(j, esperado) {
			t.Errorf("faltou %q em %s", esperado, j)
		}
	}
	if ms[0].Peca == "" {
		t.Error("mídia sem peça associada")
	}
}

package segment

import (
	"fmt"
	"testing"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
)

const procAtual = "0000.000001/2026-01"

// Estrutura observada num PDF consolidado real do SEI/RO (nomes e números fictícios).
func autosComCarimbos() *autos.Autos {
	type doc struct {
		nome, sei string
		paginas   int
		semParen  bool
		antigo    bool // traz carimbo de outro processo na mesma linha
		corpo     string
	}
	docs := []doc{
		{"PADS CAPA", "1000000001", 2, true, false, "capa"},
		{"PADS Memorando Disciplinar 12", "1000000002", 2, false, false, "texto"},
		{"Termo de Abertura", "1000000003", 1, false, false, "texto"},
		{"Anexo Despacho nº 123 - 2026 - CAPA", "1000000004", 1, false, false, "texto"},
		{"Anexo Documento", "1000000005", 3, false, false, "texto"},
		{"Despacho corregedoria - encaminhamento", "1000000006", 1, false, false, "texto"},
		{"Anexo - Processo anterior", "1000000007", 2, false, true, "texto"},
		{"Defesa Prévia do acusado", "1000000008", 3, false, false, "texto"},
		{"Notificação 56 de Inquirição", "1000000009", 1, false, false, "texto"},
		{"Anexo - Ciência sobre Inquirição de testemunha", "1000000010", 1, false, false, "texto"},
		{"PADS Ata de Presença por Videoconferência", "1000000011", 2, true, false, "texto"},
		{"Notificação 64 - Alegações finais de defesa", "1000000012", 1, false, false, "texto"},
		{"Anexo - Ciência da defesa - Alegações finais de defesa", "1000000013", 1, false, false, "texto"},
		{"Anexo - Alegações Finais de Defesa", "10000014", 3, false, false, "texto"},
		{"Termo de Remessa", "10000015", 1, false, false, "texto"},
	}
	a := &autos.Autos{}
	n := 0
	for _, d := range docs {
		for k := 0; k < d.paginas; k++ {
			n++
			sei := "(" + d.sei + ")"
			if d.semParen {
				sei = d.sei
			}
			linha := fmt.Sprintf("%s %s SEI %s / pg. %d", d.nome, sei, procAtual, n)
			if d.antigo {
				linha = fmt.Sprintf("Anexo - Termo 7 (0099999999) SEI 0000.000009/2025-09 / pg. %d %s", 30+k, linha)
			}
			txt := d.corpo + "\n" + linha
			if n == 11 { // carimbo fora da última linha (página impressa de navegador)
				txt = d.corpo + "\n" + linha + "\n12 de 13\n1"
			}
			a.Paginas = append(a.Paginas, autos.Pagina{Numero: n, Texto: txt, Situacao: autos.Lida, Origem: autos.OrigemNativo, Carimbos: autos.LerCarimbos(txt)})
		}
	}
	return a
}

func TestSegmentarPorCarimbo(t *testing.T) {
	ps := Segmentar(autosComCarimbos())
	esperado := []struct {
		ini, fim int
		f        Funcao
	}{
		{1, 2, Capa}, {3, 4, MemorandoDisciplinar}, {5, 5, Indeterminada}, {6, 6, Despacho}, {7, 9, Anexo}, // "Anexo Despacho…" é despacho anexado
		{10, 10, Despacho}, {11, 12, Anexo}, {13, 15, DefesaPrevia}, {16, 16, Notificacao}, {17, 17, Ciencia},
		{18, 19, AtaVideoconferencia}, {20, 20, Notificacao}, {21, 21, Ciencia}, {22, 24, AlegacoesFinais}, {25, 25, TermoRemessa},
	}
	if len(ps) != len(esperado) {
		for _, p := range ps {
			t.Logf("%+v", p)
		}
		t.Fatalf("peças: %d, esperado %d", len(ps), len(esperado))
	}
	for i, e := range esperado {
		p := ps[i]
		if p.PaginaInicial != e.ini || p.PaginaFinal != e.fim || p.Funcao != e.f {
			t.Errorf("peça %d: pp. %d-%d %s (%q), esperado pp. %d-%d %s", i+1, p.PaginaInicial, p.PaginaFinal, p.Funcao, p.Titulo, e.ini, e.fim, e.f)
		}
		if p.SEI == "" || p.SEI == "0099999999" {
			t.Errorf("peça %d: SEI %q (o carimbo de outro processo não pode valer)", i+1, p.SEI)
		}
	}
	if ps[7].Base != BaseTitulo || ps[7].Provavel {
		t.Errorf("defesa prévia nomeada na árvore não é inferência: %+v", ps[7])
	}
	a := autosComCarimbos()
	if c := VerificarCompatibilidade("PADS", ps, a); c.Resultado != Compativel || c.PecaCentral != "PEC-02" {
		t.Fatalf("compatibilidade: %+v", c)
	}
}

// Manifestação genérica depois das atas de videoconferência → alegações prováveis.
func TestCronologiaComAtaDeVideoconferencia(t *testing.T) {
	var a autos.Autos
	for i, nome := range []string{"PADS Memorando Disciplinar 1", "Manifestação", "PADS Ata de Presença por Videoconferência", "Manifestação"} {
		l := fmt.Sprintf("%s (%d) SEI %s / pg. %d", nome, 2000000001+i, procAtual, i+1)
		a.Paginas = append(a.Paginas, autos.Pagina{Numero: i + 1, Texto: l, Situacao: autos.Lida, Carimbos: autos.LerCarimbos(l)})
	}
	ps := Segmentar(&a)
	if ps[1].Funcao != DefesaPrevia || !ps[1].Provavel || ps[3].Funcao != AlegacoesFinais || !ps[3].Provavel {
		t.Fatalf("%+v", ps)
	}
}

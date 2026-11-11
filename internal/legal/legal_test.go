package legal

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pags(ts ...string) []PaginaTexto {
	out := make([]PaginaTexto, len(ts))
	for i, t := range ts {
		out[i] = PaginaTexto{Numero: i + 1, Texto: t}
	}
	return out
}

func achar(t *testing.T, us []Unidade, art, par, inc, ali string, vigente bool) Unidade {
	t.Helper()
	for _, u := range us {
		if u.Artigo == art && u.Paragrafo == par && u.Inciso == inc && u.Alinea == ali && u.Historica != vigente {
			return u
		}
	}
	t.Fatalf("dispositivo não encontrado: art. %s § %q inciso %q alínea %q (vigente=%v)", art, par, inc, ali, vigente)
	return Unidade{}
}

const normaSintetica = `REGULAMENTO DE TESTE
TÍTULO I
DAS DISPOSIÇÕES GERAIS
CAPÍTULO I
DO JULGAMENTO
Art. 1º Este regulamento é fictício e serve aos testes, nos termos do
Art. 65, inciso V, da Constituição Estadual.
Art. 2º Esgotado o prazo, a autoridade solucionará o caso.
Art. 2º Esgotado o prazo, a autoridade julgará o processo. (Redação dada pelo
Decreto n° 1, de 13/01/2010)
Parágrafo único. A publicação ocorrerá após a decisão tornar-se irrecorrível.
(Acrescido pelo Decreto n° 1, de 13/01/2010)
Seção I
Das Circunstâncias
Art. 3º São circunstâncias que agravam a punição:
I – a reincidência;
II – ter sido praticada a transgressão:
a) durante a execução de serviço;
b) com premeditação.
§ 1º O policial será classificado conforme os pontos.
§ 2º Ingressará no comportamento insuficiente quem cometer duas transgressões
graves em menos de 1 (um) ano.
Art. 3º-A. Dispositivo acrescido. (Acrescido pelo Decreto n° 2, de 1/02/2011)
Art. 4º Ficam extintas as execuções. (Revogado pelo Decreto n° 3, de 2/03/2012)
` + "Art. 5º Compete ao Comandante\x02Geral decidir.\n"

// Cobre: LIB-PARSE-01, LIB-PARSE-02, LIB-PARSE-03, LIB-PARSE-04, LIB-PARSE-05
// Regra: TRV-LIB-003
func TestAnalisarEstrutura(t *testing.T) {
	partes := Analisar(pags(normaSintetica))
	if len(partes) != 1 {
		t.Fatalf("partes = %d, quero 1", len(partes))
	}
	us := partes[0].Unidades
	// A citação "Art. 65, inciso V" no começo da linha não abre artigo.
	if u := achar(t, us, "1", "", "", "", true); !strings.Contains(u.Texto, "Art. 65, inciso V, da Constituição") {
		t.Errorf("art. 1º perdeu a continuação: %q", u.Texto)
	}
	for _, u := range us {
		if u.Artigo == "65" {
			t.Fatalf("citação virou artigo: %+v", u)
		}
	}
	// LIB-PARSE-03: redação dupla — a antiga é histórica, a atual tem a nota.
	if u := achar(t, us, "2", "", "", "", false); !strings.Contains(u.Texto, "solucionará") {
		t.Errorf("redação histórica errada: %q", u.Texto)
	}
	atual := achar(t, us, "2", "", "", "", true)
	if !strings.Contains(atual.Texto, "julgará") || !strings.Contains(atual.Nota, "Redação dada pelo Decreto n° 1") {
		t.Errorf("redação atual errada: %+v", atual)
	}
	// LIB-PARSE-04: parágrafo único.
	if u := achar(t, us, "2", "unico", "", "", true); !strings.Contains(u.Nota, "Acrescido") {
		t.Errorf("parágrafo único sem a nota: %+v", u)
	}
	// LIB-PARSE-02: incisos e alíneas; LIB-PARSE-01: parágrafos numerados.
	achar(t, us, "3", "", "I", "", true)
	if u := achar(t, us, "3", "", "II", "b", true); u.Texto != "com premeditação." {
		t.Errorf("alínea b = %q", u.Texto)
	}
	if u := achar(t, us, "3", "2", "", "", true); !strings.Contains(u.Texto, "graves em menos de 1 (um) ano") {
		t.Errorf("§ 2º sem a continuação: %q", u.Texto)
	}
	if u := achar(t, us, "3", "", "", "", true); u.Contexto != "TÍTULO I — DAS DISPOSIÇÕES GERAIS › CAPÍTULO I — DO JULGAMENTO › Seção I — Das Circunstâncias" {
		t.Errorf("contexto = %q", u.Contexto)
	}
	// O marcador de hífen do pdfium vira hífen.
	if u := achar(t, us, "5", "", "", "", true); u.Texto != "Compete ao Comandante-Geral decidir." {
		t.Errorf("art. 5º = %q", u.Texto)
	}
	// LIB-PARSE-05: artigo acrescido.
	achar(t, us, "3-A", "", "", "", true)
	if u := achar(t, us, "4", "", "", "", true); !u.Revogada {
		t.Errorf("art. 4º devia estar marcado como revogado")
	}
}

// Regra: TRV-LIB-003
func TestAnalisarPartesEDocumentoSemArtigos(t *testing.T) {
	txt := "DECRETO Nº 1, DE 1 DE JANEIRO DE 2000\nArt. 1º Fica aprovado o Regulamento.\nArt. 2º Este Decreto entra em vigor.\n" +
		"REGULAMENTO FICTÍCIO\nArt. 1º O Regulamento é fictício.\nArt. 2º Vale para os testes.\nArt. 3º Fim.\n"
	partes := Analisar(pags(txt))
	if len(partes) != 2 || partes[0].Artigos() != 2 || partes[1].Artigos() != 3 {
		t.Fatalf("partes = %+v", partes)
	}
	if partes[0].Titulo != "DECRETO Nº 1, DE 1 DE JANEIRO DE 2000" || partes[1].Titulo != "REGULAMENTO FICTÍCIO" {
		t.Errorf("títulos = %q e %q", partes[0].Titulo, partes[1].Titulo)
	}
	sem := Analisar(pags("Procedimento operacional padrão: a guarnição encerra a ocorrência no local e registra o boletim.", "x"))
	if len(sem) != 1 || len(sem[0].Unidades) != 1 || sem[0].Unidades[0].Artigo != "" || sem[0].Unidades[0].Pagina != 1 {
		t.Fatalf("documento sem artigos = %+v", sem)
	}
}

func abrir(t *testing.T) *Biblioteca {
	t.Helper()
	b, err := Padrao()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Cobre: T28
// Regra: TRV-LIB-002
func TestAcervoInicialCobreABibliotecaDoProjeto(t *testing.T) {
	m, err := LerManifesto()
	if err != nil {
		t.Fatal(err)
	}
	a, err := AcervoInicial()
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Documentos) != len(m.Documentos) || len(a.Documentos) == 0 {
		t.Fatalf("acervo com %d documentos; manifesto com %d (rode go run ./cmd/bibgen)", len(a.Documentos), len(m.Documentos))
	}
	lib := filepath.Join("..", "..", "docs", "law_library")
	arqs, _ := filepath.Glob(filepath.Join(lib, "*.pdf"))
	if len(arqs) == 0 {
		t.Skip("docs/law_library ausente")
	}
	if len(arqs) != len(a.Documentos) {
		t.Errorf("docs/law_library tem %d PDFs; o acervo tem %d (rode go run ./cmd/bibgen)", len(arqs), len(a.Documentos))
	}
	for _, d := range a.Documentos {
		b, err := os.ReadFile(filepath.Join(lib, d.Arquivo))
		if err != nil {
			t.Errorf("%s: %v", d.ID, err)
			continue
		}
		h := sha256.Sum256(b)
		if hex.EncodeToString(h[:]) != d.SHA256 {
			t.Errorf("%s: o original mudou depois da indexação (rode go run ./cmd/bibgen)", d.ID)
		}
		if d.SemTexto || len(d.Partes) == 0 {
			t.Errorf("%s: sem texto no acervo (original escaneado sem OCR?)", d.ID)
		}
		if RotuloCategoria[d.Categoria] == "" {
			t.Errorf("%s: categoria %q", d.ID, d.Categoria)
		}
	}
}

// Cobre: LIB-HYB-01
// Regra: TRV-LIB-004
func TestBuscaExataEPorPalavras(t *testing.T) {
	b := abrir(t)
	ps, err := b.Buscar("art. 39, parágrafo único, RDPM", 5)
	if err != nil || len(ps) == 0 {
		t.Fatalf("busca exata: %v, %d resultados", err, len(ps))
	}
	p := ps[0]
	if !p.Exata || p.DocumentoID != "rdpm" || p.Artigo != "39" || p.Paragrafo != "unico" || !strings.Contains(p.Texto, "Para efeito de reincidência") {
		t.Errorf("exata = %+v", p)
	}
	if p.Referencia != "art. 39, parágrafo único, RDPM" || p.Versao == "" || p.Pagina == 0 {
		t.Errorf("referência %q, versão %q, página %d", p.Referencia, p.Versao, p.Pagina)
	}
	// Por palavras: o tema leva ao art. 39 do RDPM; a ordem é estável.
	for range 2 {
		ps, err = b.Buscar("reincidência prazo", 5)
		if err != nil || len(ps) == 0 {
			t.Fatalf("busca por palavras: %v, %d resultados", err, len(ps))
		}
		ok := false
		for _, p := range ps {
			ok = ok || (p.DocumentoID == "rdpm" && p.Artigo == "39")
		}
		if !ok {
			t.Errorf("art. 39 do RDPM fora dos 5 primeiros: %v", refs(ps))
		}
	}
	// Plural e acento não impedem o encontro; o tesauro amplia ("pena" → punição).
	if ps, _ = b.Buscar("circunstancias que agravam a pena", 5); len(ps) == 0 || ps[0].Artigo != "39" {
		t.Errorf("agravantes: %v", refs(ps))
	}
	// A cópia do RDPM dentro da Coletânea não repete o mesmo artigo.
	ps, _ = b.Buscar("circunstâncias que atenuam a punição", 8)
	n := 0
	for _, p := range ps {
		if p.Artigo == "38" && strings.Contains(p.Texto, "atenuam") {
			n++
		}
	}
	if n > 1 {
		t.Errorf("art. 38 repetido %d vezes: %v", n, refs(ps))
	}
}

func refs(ps []Passagem) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Referencia
	}
	return out
}

// Regra: TRV-LIB-003, TRV-LIB-004
func TestDispositivoRedacaoAtualECaput(t *testing.T) {
	b := abrir(t)
	// Inciso: traz o texto do inciso e o caput do artigo.
	p, ok := b.Dispositivo(Referencia{Documento: "rdpm", Artigo: "39", Inciso: "VII", Alinea: "a"})
	if !ok || p.Texto != "durante a execução de serviço;" || !strings.Contains(p.Caput, "agravam a punição") {
		t.Errorf("art. 39, VII, a = %+v (ok=%v)", p, ok)
	}
	// Redação dupla (art. 32): só a atual.
	p, ok = b.Dispositivo(Referencia{Documento: "rdpm", Artigo: "32"})
	if !ok || !strings.Contains(p.Texto, "julgará o processo") || strings.Contains(p.Texto, "solucionará o caso") {
		t.Errorf("art. 32 = %q (ok=%v)", p.Texto, ok)
	}
	if !strings.Contains(p.Texto, "Parágrafo único. A publicação da nota de punição") {
		t.Errorf("art. 32 sem o parágrafo único: %q", p.Texto)
	}
	// Estatuto, art. 29: preceito de ética usado no enquadramento.
	p, ok = b.Dispositivo(Referencia{Documento: "estatuto", Artigo: "29", Inciso: "III"})
	if !ok || !strings.Contains(p.Texto, "dignidade da pessoa humana") {
		t.Errorf("Estatuto art. 29, III = %+v (ok=%v)", p, ok)
	}
	if _, ok := b.Dispositivo(Referencia{Documento: "rdpm", Artigo: "999"}); ok {
		t.Error("artigo inexistente encontrado")
	}
	r, ok := b.InterpretarReferencia("art. 17, inciso V, do RDPM")
	if !ok || r != (Referencia{Documento: "rdpm", Artigo: "17", Inciso: "V"}) {
		t.Errorf("referência = %+v", r)
	}
	r, _ = b.InterpretarReferencia("Art. 29, XVII, do Estatuto dos Policiais Militares (Decreto-Lei nº 09-A/1982)")
	if r != (Referencia{Documento: "estatuto", Artigo: "29", Inciso: "XVII"}) {
		t.Errorf("referência = %+v", r)
	}
	r, _ = b.InterpretarReferencia("art. 41, § 3º, do RDPM")
	if r != (Referencia{Documento: "rdpm", Artigo: "41", Paragrafo: "3"}) {
		t.Errorf("referência = %+v", r)
	}
}

// Cobre: LIB-D01-01
// Regra: TRV-LIB-012
func TestNotaDeVigenciaFiltraOArt41Par3(t *testing.T) {
	b := abrir(t)
	ref := Referencia{Documento: "rdpm", Artigo: "41", Paragrafo: "3"}
	if _, ok := b.Dispositivo(ref); ok {
		t.Error("art. 41, § 3º oferecido à redação")
	}
	n, ok := b.Nota(ref)
	if !ok || n.Status != StatusRevogadoPeloUsuario || n.FonteRevogadora != "" || !strings.Contains(n.Origem, "D-01") {
		t.Errorf("nota = %+v (ok=%v)", n, ok)
	}
	// O artigo inteiro vem sem o § 3º, e os demais parágrafos ficam.
	p, ok := b.Dispositivo(Referencia{Documento: "rdpm", Artigo: "41"})
	if !ok || strings.Contains(p.Texto, "Ficam extintas as execuções") || !strings.Contains(p.Texto, "§ 2º") {
		t.Errorf("art. 41 = %q", p.Texto)
	}
	// A busca por palavras também não o traz, nem pela cópia da Coletânea.
	ps, err := b.Buscar("extintas execuções medidas privativas restritivas liberdade caráter disciplinar", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		if strings.Contains(p.Texto, "Ficam extintas as execuções de medidas privativas") {
			t.Errorf("trecho filtrado apareceu em %s", p.Referencia)
		}
	}
}

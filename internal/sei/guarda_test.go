package sei

import (
	"net/url"
	"testing"

	"github.com/chromedp/cdproto/network"
)

// SEG13-07 / SEG-SEI-01 (unidade): só consulta, geração e download passam.
func TestGuarda(t *testing.T) {
	g := NovaGuarda("sei.sistemas.ro.gov.br", false, AcoesLeitura, []string{"acao_permitida"})
	casos := []struct {
		url  string
		tipo network.ResourceType
		ok   bool
	}{
		{"https://sei.sistemas.ro.gov.br/sip/login.php?sigla_sistema=SEI", network.ResourceTypeDocument, true},
		{"https://sei.sistemas.ro.gov.br/sei/inicializar.php", network.ResourceTypeDocument, true},
		{"https://sei.sistemas.ro.gov.br/sei/controlador.php?acao=procedimento_trabalhar&infra_hash=x", network.ResourceTypeDocument, true},
		{"https://sei.sistemas.ro.gov.br/sei/controlador.php?acao=procedimento_gerar_pdf", network.ResourceTypeDocument, true},
		{"https://sei.sistemas.ro.gov.br/sei/controlador.php?acao=exibir_arquivo", network.ResourceTypeDocument, true},
		{"https://sei.sistemas.ro.gov.br/sei/controlador_ajax.php?acao_ajax=acao_permitida", network.ResourceTypeXHR, true},
		{"https://sei.sistemas.ro.gov.br/infra_css/infra.css", network.ResourceTypeStylesheet, true},
		{"https://sei.sistemas.ro.gov.br/imagens/pdf.svg", network.ResourceTypeImage, true},
		// Bloqueios:
		{"https://sei.sistemas.ro.gov.br/sei/controlador.php?acao=procedimento_excluir", network.ResourceTypeDocument, false},
		{"https://sei.sistemas.ro.gov.br/sei/controlador.php?acao=documento_assinar", network.ResourceTypeXHR, false},
		{"https://sei.sistemas.ro.gov.br/sei/controlador.php?acao=procedimento_concluir", network.ResourceTypeImage, false}, // ação disfarçada de imagem
		{"https://sei.sistemas.ro.gov.br/sei/controlador.php", network.ResourceTypeDocument, false},
		{"https://sei.sistemas.ro.gov.br/sei/controlador_ajax.php?acao_ajax=outra", network.ResourceTypeXHR, false},
		{"https://sei.sistemas.ro.gov.br/sei/qualquer.php", network.ResourceTypeImage, false},
		{"https://sei.sistemas.ro.gov.br/sei/qualquer.php", network.ResourceTypeDocument, false},
		{"http://sei.sistemas.ro.gov.br/sip/login.php", network.ResourceTypeDocument, false},
		{"https://outro.exemplo/sip/login.php", network.ResourceTypeDocument, false},
		{"https://sei.sistemas.ro.gov.br/x", network.ResourceTypePing, false},
		{"file:///C:/Windows/win.ini", network.ResourceTypeDocument, false},
	}
	for _, c := range casos {
		u, _ := url.Parse(c.url)
		if got := g.Permitir(u, c.tipo); got != c.ok {
			t.Errorf("%s (%s): %v, esperado %v", c.url, c.tipo, got, c.ok)
		}
	}
	if len(g.Bloqueios()) != 11 {
		t.Errorf("bloqueios registrados: %v", g.Bloqueios())
	}
	// Sem acao_ajax configurada (padrão até o valor ser informado): nada passa.
	g2 := NovaGuarda("sei.sistemas.ro.gov.br", false, AcoesLeitura, AcoesAjax)
	u, _ := url.Parse("https://sei.sistemas.ro.gov.br/sei/controlador_ajax.php?acao_ajax=qualquer")
	if g2.Permitir(u, network.ResourceTypeXHR) {
		t.Error("acao_ajax não configurada deve ser bloqueada")
	}
}

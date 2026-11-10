package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestForma(t *testing.T) {
	casos := map[string]string{
		"Termo de Inquirição - SD PM FULANO (0012345)": "Termo de Inquirição - AA AA AA (#######)",
		"Memorando Disciplinar 101/2026":               "Memorando Disciplinar ###/####",
		"Ofício nº 9":                                  "Ofício nº #",
		"Despacho da Corregedoria":                     "Despacho da Aa",
	}
	for in, quer := range casos {
		if got := forma(in); got != quer {
			t.Errorf("forma(%q) = %q, quer %q", in, got, quer)
		}
	}
}

// Página fictícia no formato observado na F0 (frmProcedimentoPdf, rdoTipo,
// selDocumentosApenas, hdnDocumentosApenas). Nomes e números inventados.
const telaFicticia = `<!doctype html><html><body>
<form id="frmProcedimentoPdf" method="post" action="controlador.php?acao=procedimento_gerar_pdf&infra_hash=h1">
<input type="radio" name="rdoTipo" id="optTodos" value="T" checked onclick="trocar('T');"> <label for="optTodos">Todos os documentos disponíveis</label>
<input type="radio" name="rdoTipo" id="optApenas" value="A" onclick="trocar('A');"> <label for="optApenas">Apenas selecionados</label>
<select id="selDocumentosApenas" name="selDocumentosApenas" multiple size="8">
<option value="11112222">Memorando Disciplinar 101 (0010000)</option>
<option value="11113333">Termo de Inquirição - SD PM ANTÔNIO FICTÍCIO (0010001)</option>
</select>
<table id="tblDocumentos"><tr><th>Documento</th><th>Tipo</th></tr>
<tr><td><input type="checkbox" id="chkInfraItem0" name="chkInfraItem0" value="11112222" title="Memorando 0010000"></td><td>Memorando Disciplinar</td></tr>
<tr><td><input type="checkbox" id="chkInfraItem1" name="chkInfraItem1" value="11113333"></td><td>Termo de Inquirição de BENTO IMAGINÁRIO</td></tr></table>
<input type="hidden" name="hdnDocumentosApenas" id="hdnDocumentosApenas" value="11112222,11113333">
<button type="button" name="btnGerar" value="Gerar" onclick="gerar();">Gerar</button>
</form>
<script>
function trocar(t) { document.getElementById('selDocumentosApenas').disabled = (t != 'A'); }
function gerar() {
  if (document.getElementById('optApenas').checked && document.getElementById('hdnDocumentosApenas').value == '') { alert('Nenhum documento selecionado para ANTÔNIO 123.'); return; }
  document.getElementById('hdnFlagGerar').value = '1';
}
</script></body></html>`

// O JavaScript do mapeamento roda num navegador real, sem erro, e não
// devolve nenhum nome nem número da página (só formas e palavras da lista).
func TestMapaGerarNaoVazaNemQuebra(t *testing.T) {
	chrome := os.Getenv("SEIMAPA_CHROME")
	if chrome == "" {
		chrome = os.ExpandEnv("$HOME/.cache/sys-executor-dev/bin/chrome/chrome-linux64/chrome")
	}
	if _, err := os.Stat(chrome); err != nil {
		t.Skip("Chrome de desenvolvimento ausente (defina SEIMAPA_CHROME)")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(telaFicticia))
	}))
	defer srv.Close()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(chrome),
		chromedp.Flag("disable-gpu", true), chromedp.NoSandbox) // nunca GPU nesta máquina
	actx, cancelA := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelA()
	ctx, cancel := chromedp.NewContext(actx)
	defer cancel()
	ctx, cancelT := context.WithTimeout(ctx, 60*time.Second)
	defer cancelT()
	if _, err := chromedp.Run(ctx, chromedp.Navigate(srv.URL+"/sei/controlador.php?acao=procedimento_gerar_pdf")); err != nil {
		t.Fatal(err)
	}
	raw, err := chromedp.Run(ctx, chromedp.Evaluate[string](mapaGerarJS))
	if err != nil {
		t.Fatal(err)
	}
	var quadros []map[string]any
	if err := json.Unmarshal([]byte(raw), &quadros); err != nil || len(quadros) != 1 {
		t.Fatalf("saída inesperada: %v %s", err, raw)
	}
	for _, proibido := range []string{"FULANO", "ANTÔNIO", "FICTÍCIO", "BENTO", "IMAGINÁRIO", "0010000", "11112222", "123"} {
		if strings.Contains(raw, proibido) {
			t.Errorf("saída contém %q: %s", proibido, raw)
		}
	}
	for _, esperado := range []string{"selDocumentosApenas", `"multiplo":true`, `"opcoes":2`, "(#######)", "########,########", "chkInfraItem#", "function gerar()", "function trocar(t)", "Memorando Disciplinar"} {
		if !strings.Contains(raw, esperado) {
			t.Errorf("saída sem %q: %s", esperado, raw)
		}
	}
}

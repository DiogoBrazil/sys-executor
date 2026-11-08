// Package sei obtém o PDF consolidado do processo no SEI/RO dirigindo o
// Microsoft Edge instalado (chromedp; ADR-008; mapeamento real na F0-08).
//
// Regras de segurança (V4 §§1.2, 2.1, 13, 33):
//   - o cliente não tem nenhum método de escrita no SEI;
//   - uma guarda de navegação (CDP Fetch) bloqueia todo host que não seja o do
//     SEI e toda ação (acao=) que não seja de consulta, geração do PDF ou
//     download (deny-by-default);
//   - credenciais e códigos chegam como []byte, são usados uma vez e zerados;
//   - nunca marca "Não usar 2FA neste dispositivo" nem aciona "Desativar 2FA";
//   - perfil do navegador e download ficam na pasta protegida da sessão.
package sei

import (
	"net/url"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/network"
)

// AcoesLeitura são as ações do controlador.php necessárias para consultar o
// processo e gerar o PDF (mapeadas na F0-08). Qualquer outra é bloqueada.
var AcoesLeitura = []string{
	"procedimento_controlar",
	"protocolo_pesquisa_rapida",
	"procedimento_trabalhar",
	"usuario_validar_acesso",
	"procedimento_visualizar",
	"arvore_visualizar",
	"arvore_processar_html",
	"procedimento_gerar_pdf",
	"exibir_arquivo",
}

// AcoesAjax são os valores de acao_ajax permitidos em controlador_ajax.php.
// O usuário confirmou (08/11/2026) que o ícone PDF e o botão "Gerar" não
// usam controlador_ajax.php: nenhum valor é necessário, todos são bloqueados.
var AcoesAjax = []string{}

// Guarda decide se uma requisição do navegador pode seguir.
type Guarda struct {
	host         string
	permitirHTTP bool
	acoes        map[string]bool
	acoesAjax    map[string]bool

	mu        sync.Mutex
	bloqueios []string // descrições sem dados (host/caminho/acao), para diagnóstico em memória
}

// NovaGuarda cria a guarda para o host do SEI. permitirHTTP só é usado pelo
// simulador de testes.
func NovaGuarda(host string, permitirHTTP bool, acoes, acoesAjax []string) *Guarda {
	g := &Guarda{host: strings.ToLower(host), permitirHTTP: permitirHTTP, acoes: map[string]bool{}, acoesAjax: map[string]bool{}}
	for _, a := range acoes {
		g.acoes[a] = true
	}
	for _, a := range acoesAjax {
		g.acoesAjax[a] = true
	}
	return g
}

// Permitir aplica as regras. Recursos estáticos (estilo, script, imagem,
// fonte) do próprio host do SEI passam; navegações e chamadas de dados só se
// estiverem na lista.
func (g *Guarda) Permitir(u *url.URL, tipo network.ResourceType) bool {
	ok := g.decidir(u, tipo)
	if !ok {
		g.mu.Lock()
		acao := u.Query().Get("acao")
		if acao == "" {
			acao = u.Query().Get("acao_ajax")
		}
		g.bloqueios = append(g.bloqueios, u.Host+u.Path+" "+acao)
		g.mu.Unlock()
	}
	return ok
}

func (g *Guarda) decidir(u *url.URL, tipo network.ResourceType) bool {
	switch u.Scheme {
	case "https":
	case "http":
		if !g.permitirHTTP {
			return false
		}
	case "data", "blob":
		return true // conteúdo embutido na própria página, sem rede
	default:
		return false
	}
	if strings.ToLower(u.Host) != g.host {
		return false
	}
	q := u.Query()
	// Controladores: a lista de ações vale para QUALQUER tipo de recurso (uma
	// <img src="controlador.php?acao=..."> também executaria a ação).
	switch u.Path {
	case "/sei/controlador.php":
		return g.acoes[q.Get("acao")]
	case "/sei/controlador_ajax.php":
		return g.acoesAjax[q.Get("acao_ajax")]
	}
	switch tipo {
	case network.ResourceTypeStylesheet, network.ResourceTypeScript, network.ResourceTypeImage, network.ResourceTypeFont:
		return !strings.HasSuffix(strings.ToLower(u.Path), ".php") // estáticos; nada de script de servidor
	case network.ResourceTypeDocument, network.ResourceTypeXHR, network.ResourceTypeFetch, network.ResourceTypeOther:
	default:
		return false // ping, beacon, websocket, etc.
	}
	switch u.Path {
	case "/sip/login.php", "/sei/inicializar.php":
		return true
	}
	return false
}

// Bloqueios devolve as requisições barradas (diagnóstico; sem dados do processo).
func (g *Guarda) Bloqueios() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string(nil), g.bloqueios...)
}

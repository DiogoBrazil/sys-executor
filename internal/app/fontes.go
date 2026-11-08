package app

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
	"github.com/DiogoBrazil/sys-executor/internal/pipeline"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// PrefixoAutos é o caminho (no servidor de assets do app) do PDF dos autos.
const PrefixoAutos = "/autos/"

// FontesService dá acesso aos autos da tarefa ativa (visualizador, realce, busca).
type FontesService struct {
	g *workflow.Gerenciador

	mu     sync.Mutex
	tarefa string // ID da tarefa à qual o token pertence
	token  string
}

// NovaFontesService cria o serviço.
func NovaFontesService(g *workflow.Gerenciador) *FontesService { return &FontesService{g: g} }

// RespostaEndereco devolve o endereço do PDF da tarefa ativa.
type RespostaEndereco struct {
	Endereco string              `json:"endereco"`
	Erro     *workflow.ErroVisao `json:"erro"`
}

// EnderecoPDF gera (ou reaproveita) o endereço do PDF desta tarefa. O token é
// aleatório e vale só para a tarefa ativa.
func (s *FontesService) EnderecoPDF() RespostaEndereco {
	v := s.g.Visao()
	if v == nil || v.Terminal || s.g.Sessao() == nil {
		return RespostaEndereco{Erro: erroVisao(erros.Novo(erros.TarefaSemTarefa))}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tarefa != v.ID || s.token == "" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		s.tarefa, s.token = v.ID, hex.EncodeToString(b)
	}
	return RespostaEndereco{Endereco: PrefixoAutos + s.token + ".pdf"}
}

// Middleware serve o PDF dos autos sem cache. Qualquer outro caminho segue
// para o servidor de assets padrão.
func (s *FontesService) Middleware(proximo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, PrefixoAutos) {
			proximo.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		token := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, PrefixoAutos), ".pdf")
		v := s.g.Visao()
		s.mu.Lock()
		valido := token != "" && token == s.token && v != nil && v.ID == s.tarefa && !v.Terminal
		s.mu.Unlock()
		sessao := s.g.Sessao()
		if !valido || sessao == nil || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		b, err := sessao.Ler(workflow.NomeAutos)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(b)
	})
}

// Caixa em pontos, origem no canto superior esquerdo.
type Caixa struct {
	X0 float64 `json:"x0"`
	Y0 float64 `json:"y0"`
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
}

// RespostaDestaque traz o tamanho da página e as caixas do trecho (UI-SRC-01).
type RespostaDestaque struct {
	Pagina  int                 `json:"pagina"`
	Largura float64             `json:"largura"`
	Altura  float64             `json:"altura"`
	Caixas  []Caixa             `json:"caixas"`
	Erro    *workflow.ErroVisao `json:"erro"`
}

func (s *FontesService) dados() *pipeline.Dados {
	d, _ := s.g.Dados().(*pipeline.Dados)
	if d == nil || d.Autos == nil {
		return nil
	}
	return d
}

// Destaque devolve as caixas do intervalo [inicio, fim) da página (runas).
// Sem caixas (ex.: posição desconhecida), a interface abre só a página.
func (s *FontesService) Destaque(pagina, inicio, fim int) RespostaDestaque {
	d := s.dados()
	if d == nil {
		return RespostaDestaque{Erro: erroVisao(erros.Novo(erros.TarefaSemTarefa))}
	}
	if pagina < 1 || pagina > len(d.Autos.Paginas) {
		return RespostaDestaque{Erro: erroVisao(erros.Novo(erros.TarefaOrdemInvalida))}
	}
	p := d.Autos.Paginas[pagina-1]
	r := RespostaDestaque{Pagina: pagina, Largura: p.Largura, Altura: p.Altura, Caixas: []Caixa{}}
	if fim > inicio {
		for _, c := range p.Caixas(inicio, fim) {
			r.Caixas = append(r.Caixas, Caixa{X0: c.X0, Y0: c.Y0, X1: c.X1, Y1: c.Y1})
		}
	}
	return r
}

// RespostaBusca traz as ocorrências encontradas nos autos.
type RespostaBusca struct {
	Ocorrencias []autos.Ocorrencia  `json:"ocorrencias"`
	Erro        *workflow.ErroVisao `json:"erro"`
}

// Buscar procura nos autos da tarefa ativa (índice em memória).
func (s *FontesService) Buscar(consulta string) RespostaBusca {
	d := s.dados()
	if d == nil || d.Indice == nil {
		return RespostaBusca{Erro: erroVisao(erros.Novo(erros.TarefaSemTarefa))}
	}
	oc, err := d.Indice.Buscar(consulta, 30)
	if err != nil {
		return RespostaBusca{Erro: erroVisao(erros.Com(erros.InternoInesperado, err))}
	}
	if oc == nil {
		oc = []autos.Ocorrencia{}
	}
	return RespostaBusca{Ocorrencias: oc}
}

func erroVisao(e *erros.Erro) *workflow.ErroVisao {
	return &workflow.ErroVisao{Codigo: string(e.Codigo), Titulo: e.Titulo, Mensagem: e.Mensagem, Acao: e.Acao}
}

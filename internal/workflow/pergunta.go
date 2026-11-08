package workflow

// Pergunta pontual que pausa a tarefa (V4 §31.2). Cada opção leva a um
// destino explícito: "" volta à etapa que perguntou (que lê a resposta),
// RecebendoUpload pede outro PDF, Cancelada encerra a tarefa.
type Pergunta struct {
	ID       string
	Titulo   string
	Contexto string
	Opcoes   []Opcao
}

// Opcao de resposta.
type Opcao struct {
	ID      string
	Rotulo  string
	Destino Estado
}

// PerguntaVisao é a pergunta enviada à interface.
type PerguntaVisao struct {
	ID       string       `json:"id"`
	Titulo   string       `json:"titulo"`
	Contexto string       `json:"contexto"`
	Opcoes   []OpcaoVisao `json:"opcoes"`
}

// OpcaoVisao é uma opção exibida.
type OpcaoVisao struct {
	ID     string `json:"id"`
	Rotulo string `json:"rotulo"`
}

func (p *Pergunta) visao() *PerguntaVisao {
	if p == nil {
		return nil
	}
	v := &PerguntaVisao{ID: p.ID, Titulo: p.Titulo, Contexto: p.Contexto}
	for _, o := range p.Opcoes {
		v.Opcoes = append(v.Opcoes, OpcaoVisao{ID: o.ID, Rotulo: o.Rotulo})
	}
	return v
}

// ResumoAutos é o que a interface mostra sobre os autos lidos (cobertura,
// peças e mídias). Não contém texto dos autos.
type ResumoAutos struct {
	TotalPaginas    int            `json:"totalPaginas"`
	Paginas         []PaginaResumo `json:"paginas"`
	Pecas           []PecaResumo   `json:"pecas"`
	Midias          []MidiaResumo  `json:"midias"`
	Compatibilidade string         `json:"compatibilidade"`
	PecaCentral     string         `json:"pecaCentral"`
}

// PaginaResumo é uma linha do mapa de cobertura.
type PaginaResumo struct {
	Numero    int     `json:"numero"`
	Situacao  string  `json:"situacao"`
	Origem    string  `json:"origem"`
	Confianca float64 `json:"confianca"`
	Peca      string  `json:"peca"`
}

// PecaResumo é uma peça identificada.
type PecaResumo struct {
	ID            string `json:"id"`
	Titulo        string `json:"titulo"`
	Funcao        string `json:"funcao"`
	Rotulo        string `json:"rotulo"`
	PaginaInicial int    `json:"paginaInicial"`
	PaginaFinal   int    `json:"paginaFinal"`
	SEI           string `json:"sei"`
	Provavel      bool   `json:"provavel"`
	Externo       bool   `json:"externo"`
	Situacao      string `json:"situacao"` // pior situação de leitura entre as páginas
}

// MidiaResumo é uma menção a mídia não incorporada aos autos.
type MidiaResumo struct {
	Pagina int    `json:"pagina"`
	Peca   string `json:"peca"`
	Termo  string `json:"termo"`
	Inicio int    `json:"inicio"`
	Fim    int    `json:"fim"`
}

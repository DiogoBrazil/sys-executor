package app

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/credencial"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/prefs"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// EventoIA leva a visão das configurações de IA à interface.
const EventoIA = "ia:visao"

// URLChavesAIStudio: onde o usuário obtém a chave (V4 §47; aberto pelo Go).
const URLChavesAIStudio = "https://aistudio.google.com/api-keys"

// tempoRede: limite das operações de configuração (listar, validar, testar).
const tempoRede = 60 * time.Second

// esperaConfig: espera máxima entre tentativas nessas operações; acima disso
// (ex.: cota por minuto do nível gratuito), o erro aparece logo na tela.
const esperaConfig = 5 * time.Second

// FonteGemini implementa pipeline.FonteMotor com a chave guardada (DPAPI) e
// o modelo escolhido nas preferências (TRV-IA-011).
type FonteGemini struct {
	Credenciais credencial.Loja
	Loja        *prefs.Loja
	// Pensamento: nível de raciocínio enviado ao modelo ("" = padrão do modelo).
	Pensamento string

	// Injetáveis nos testes; nil = Gemini API real.
	NovoMotor func(context.Context, inference.ConfigGemini) (MotorComUso, error)
	Listar    func(context.Context, inference.ConfigGemini) ([]inference.ModeloGemini, error)
	Testar    func(context.Context, inference.ConfigGemini) error

	mu     sync.Mutex
	ultimo MotorComUso
}

// MotorComUso é o motor da tarefa com o consumo acumulado.
type MotorComUso interface {
	inference.Motor
	Uso() inference.Uso
	Fechar()
}

func (f *FonteGemini) novoMotor() func(context.Context, inference.ConfigGemini) (MotorComUso, error) {
	if f.NovoMotor != nil {
		return f.NovoMotor
	}
	return func(ctx context.Context, c inference.ConfigGemini) (MotorComUso, error) {
		m, err := inference.NovoMotorGemini(ctx, c)
		if err != nil {
			return nil, err
		}
		return m, nil
	}
}

func (f *FonteGemini) listar() func(context.Context, inference.ConfigGemini) ([]inference.ModeloGemini, error) {
	if f.Listar != nil {
		return f.Listar
	}
	return inference.ListarModelos
}

func (f *FonteGemini) testar() func(context.Context, inference.ConfigGemini) error {
	if f.Testar != nil {
		return f.Testar
	}
	return inference.TestarModelo
}

// Alvo confere a configuração sem rede e sem decifrar a chave
// (pipeline.FonteMotor).
func (f *FonteGemini) Alvo() (inference.Alvo, error) {
	m, err := f.Modelo()
	if err != nil {
		return inference.Alvo{}, err
	}
	return inference.Alvo{Provedor: inference.ProvedorGemini, Rotulo: inference.RotuloProvedor(inference.ProvedorGemini), Modelo: m}, nil
}

// Modelo confere a configuração sem rede e sem decifrar a chave.
func (f *FonteGemini) Modelo() (string, error) {
	if f.Credenciais == nil || !f.Credenciais.Existe() {
		return "", erros.Novo(erros.IAChaveAusente)
	}
	m := f.Loja.Obter().ModeloGemini
	if m == "" {
		return "", erros.Com(erros.IAModeloIndisponivel, errors.New("nenhum modelo escolhido"))
	}
	return m, nil
}

// lerChave decifra a chave; o chamador zera a cópia depois de usar.
func (f *FonteGemini) lerChave() ([]byte, error) {
	if f.Credenciais == nil {
		return nil, erros.Novo(erros.IAChaveAusente)
	}
	c, err := f.Credenciais.Ler()
	switch {
	case errors.Is(err, credencial.ErrAusente):
		return nil, erros.Novo(erros.IAChaveAusente)
	case err != nil:
		return nil, erros.Com(erros.IACredencial, err)
	}
	return c, nil
}

// Abrir cria o cliente da tarefa e confere o modelo no provedor.
func (f *FonteGemini) Abrir(ctx context.Context, alvo inference.Alvo, atividade func(string), inspecionar func([]byte) error) (inference.Motor, func(), error) {
	if alvo.Provedor != inference.ProvedorGemini || alvo.Modelo == "" {
		return nil, nil, erros.Com(erros.InternoInesperado, errors.New("alvo de IA não é a Gemini API"))
	}
	// O motor abre exatamente o modelo autorizado na tarefa (V4 §48.1), não
	// o que estiver nas preferências agora.
	modelo := alvo.Modelo
	chave, err := f.lerChave()
	if err != nil {
		return nil, nil, err
	}
	defer credencial.Zerar(chave)
	atividade("Conectando à Gemini API")
	m, err := f.novoMotor()(ctx, inference.ConfigGemini{Chave: chave, Modelo: modelo, Pensamento: f.Pensamento, Inspecionar: inspecionar})
	if err != nil {
		return nil, nil, err
	}
	f.mu.Lock()
	f.ultimo = m
	f.mu.Unlock()
	return m, m.Fechar, nil
}

// Uso da última tarefa (memória; nunca gravado).
func (f *FonteGemini) Uso() (inference.Uso, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ultimo == nil {
		return inference.Uso{}, false
	}
	return f.ultimo.Uso(), true
}

// Situações da configuração exibidas na tela.
const (
	StatusSemChave      = "sem_chave"
	StatusNaoVerificada = "nao_verificada"
	StatusValida        = "valida"
	StatusInvalida      = "invalida"
	StatusSemConexao    = "sem_conexao"
)

// ModeloIA é um modelo na tela, com o resultado do último teste.
type ModeloIA struct {
	inference.ModeloGemini
	Teste     string              `json:"teste"` // "", "ok", "falhou"
	TesteErro *workflow.ErroVisao `json:"testeErro"`
}

// UsoIA resume o consumo da última tarefa (sem preço: não inventamos valores).
type UsoIA struct {
	Modelo           string `json:"modelo"`
	Chamadas         int    `json:"chamadas"`
	Falhas           int    `json:"falhas"`
	TokensEntrada    int    `json:"tokensEntrada"`
	TokensSaida      int    `json:"tokensSaida"`
	TokensPensamento int    `json:"tokensPensamento"`
	Segundos         int    `json:"segundos"`
}

// VisaoIA é o que a tela de configurações de IA recebe. NUNCA contém a
// chave nem parte dela (TRV-SEG-009).
type VisaoIA struct {
	ChaveCadastrada   bool                `json:"chaveCadastrada"`
	Status            string              `json:"status"`
	Modelos           []ModeloIA          `json:"modelos"`
	ModeloSelecionado string              `json:"modeloSelecionado"`
	ModeloForaDaLista bool                `json:"modeloForaDaLista"`
	Ocupado           bool                `json:"ocupado"`
	Uso               *UsoIA              `json:"uso"`
	URLChaves         string              `json:"urlChaves"`
	Erro              *workflow.ErroVisao `json:"erro"`
}

// IAService expõe as configurações de IA (substitui a tela Modelos local).
type IAService struct {
	Fonte    *FonteGemini
	Emitir   func(*VisaoIA)
	AbrirURL func(string) error

	mu      sync.Mutex
	status  string
	modelos []inference.ModeloGemini
	testes  map[string]ModeloIA
	ocupado bool
	erro    *erros.Erro
}

func (s *IAService) emitir(v VisaoIA) VisaoIA {
	if s.Emitir != nil {
		c := v
		s.Emitir(&c)
	}
	return v
}

// Estado monta a visão sem acessar a rede.
func (s *IAService) Estado() VisaoIA {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.visaoLocked()
}

func (s *IAService) visaoLocked() VisaoIA {
	v := VisaoIA{URLChaves: URLChavesAIStudio, Ocupado: s.ocupado, Status: s.status,
		ModeloSelecionado: s.Fonte.Loja.Obter().ModeloGemini}
	v.ChaveCadastrada = s.Fonte.Credenciais != nil && s.Fonte.Credenciais.Existe()
	if !v.ChaveCadastrada {
		v.Status = StatusSemChave
	} else if v.Status == "" || v.Status == StatusSemChave {
		v.Status = StatusNaoVerificada
	}
	achou := false
	for _, m := range s.modelos {
		mi := ModeloIA{ModeloGemini: m}
		if t, ok := s.testes[m.ID]; ok {
			mi.Teste, mi.TesteErro = t.Teste, t.TesteErro
		}
		v.Modelos = append(v.Modelos, mi)
		achou = achou || m.ID == v.ModeloSelecionado
	}
	v.ModeloForaDaLista = v.ModeloSelecionado != "" && len(s.modelos) > 0 && !achou
	if u, ok := s.Fonte.Uso(); ok {
		v.Uso = &UsoIA{Modelo: u.Modelo, Chamadas: u.Chamadas, Falhas: u.Falhas, TokensEntrada: u.TokensEntrada,
			TokensSaida: u.TokensSaida, TokensPensamento: u.TokensPensamento, Segundos: int(u.Duracao.Seconds())}
	}
	if s.erro != nil {
		v.Erro = workflow.VisaoErro(s.erro)
	}
	return v
}

// iniciar marca a operação em andamento (uma por vez).
func (s *IAService) iniciar() bool {
	s.mu.Lock()
	if s.ocupado {
		s.mu.Unlock()
		return false
	}
	s.ocupado, s.erro = true, nil
	v := s.visaoLocked()
	s.mu.Unlock()
	s.emitir(v)
	return true
}

func (s *IAService) concluir(f func()) VisaoIA {
	s.mu.Lock()
	f()
	s.ocupado = false
	v := s.visaoLocked()
	s.mu.Unlock()
	return s.emitir(v)
}

func (s *IAService) ocupadoAgora() VisaoIA {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.visaoLocked()
	v.Erro = workflow.VisaoErro(erros.Novo(erros.TarefaOrdemInvalida))
	return v
}

// Chaves da Gemini API: letras, dígitos, "-", "_" e ".", tamanho razoável.
var reChave = regexp.MustCompile(`^[A-Za-z0-9_.\-]{20,200}$`)

// statusDoErro traduz o erro de rede/credencial na situação da tela.
func statusDoErro(err error) string {
	var e *erros.Erro
	if errors.As(err, &e) {
		switch e.Codigo {
		case erros.IAChaveInvalida, erros.IAPermissaoNegada:
			return StatusInvalida
		case erros.IASemConexao, erros.IATempoEsgotado, erros.IAIndisponivel:
			return StatusSemConexao
		}
	}
	return StatusNaoVerificada
}

// SalvarChave valida a chave na Gemini API (listagem de modelos, sem gerar
// texto) e só então a guarda com DPAPI. Chave recusada não é guardada.
func (s *IAService) SalvarChave(chave string) VisaoIA {
	b := []byte(strings.TrimSpace(chave))
	defer credencial.Zerar(b)
	if !reChave.Match(b) {
		s.mu.Lock()
		s.erro = erros.Com(erros.IAChaveInvalida, errors.New("formato de chave inválido"))
		v := s.visaoLocked()
		s.mu.Unlock()
		return s.emitir(v)
	}
	if !s.iniciar() {
		return s.ocupadoAgora()
	}
	ctx, cancel := context.WithTimeout(context.Background(), tempoRede)
	defer cancel()
	ms, err := s.Fonte.listar()(ctx, inference.ConfigGemini{Chave: b, EsperaMax: esperaConfig})
	if err == nil {
		if e := s.Fonte.Credenciais.Guardar(b); e != nil {
			err = erros.Com(erros.IACredencial, e)
		}
	}
	return s.concluir(func() {
		if err != nil {
			// A chave nova não foi guardada; a situação da chave anterior (se
			// houver) não muda. A mensagem explica o motivo.
			s.erro = erros.De(err)
			return
		}
		s.status, s.modelos, s.testes = StatusValida, ms, map[string]ModeloIA{}
	})
}

// RemoverChave apaga a chave guardada.
func (s *IAService) RemoverChave() VisaoIA {
	err := s.Fonte.Credenciais.Remover()
	s.mu.Lock()
	if err != nil {
		s.erro = erros.Com(erros.IACredencial, err)
	} else {
		s.erro, s.status, s.modelos, s.testes = nil, StatusSemChave, nil, nil
	}
	v := s.visaoLocked()
	s.mu.Unlock()
	return s.emitir(v)
}

// Verificar testa a conexão com a chave guardada e atualiza a lista de modelos.
func (s *IAService) Verificar() VisaoIA {
	if !s.iniciar() {
		return s.ocupadoAgora()
	}
	chave, err := s.Fonte.lerChave()
	var ms []inference.ModeloGemini
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), tempoRede)
		ms, err = s.Fonte.listar()(ctx, inference.ConfigGemini{Chave: chave, EsperaMax: esperaConfig})
		cancel()
		credencial.Zerar(chave)
	}
	return s.concluir(func() {
		if err != nil {
			s.erro, s.status = erros.De(err), statusDoErro(err)
			return
		}
		s.status, s.modelos = StatusValida, ms
		if s.testes == nil {
			s.testes = map[string]ModeloIA{}
		}
	})
}

// SelecionarModelo grava o modelo escolhido (V4 §47.5).
func (s *IAService) SelecionarModelo(id string) VisaoIA {
	s.mu.Lock()
	conhecido := len(s.modelos) == 0
	for _, m := range s.modelos {
		conhecido = conhecido || m.ID == id
	}
	s.mu.Unlock()
	var err error
	if !conhecido {
		err = erros.Com(erros.IAModeloIndisponivel, errors.New("modelo fora da lista da chave"))
	} else if _, e := s.Fonte.Loja.Atualizar(func(p *prefs.Preferencias) {
		// Escolher um modelo Gemini é escolher a Gemini API como provedor
		// (escolha explícita, V4 §48.1).
		p.ModeloGemini, p.ProvedorIA = id, prefs.ProvedorGemini
	}); e != nil {
		err = erros.Com(erros.IAModeloIndisponivel, e)
	}
	s.mu.Lock()
	s.erro = nil
	if err != nil {
		s.erro = erros.De(err)
	}
	v := s.visaoLocked()
	s.mu.Unlock()
	return s.emitir(v)
}

// TestarModelo faz uma geração mínima, sem dados dos autos, para saber se a
// chave tem acesso e cota para o modelo.
func (s *IAService) TestarModelo(id string) VisaoIA {
	if !s.iniciar() {
		return s.ocupadoAgora()
	}
	chave, err := s.Fonte.lerChave()
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), tempoRede)
		err = s.Fonte.testar()(ctx, inference.ConfigGemini{Chave: chave, Modelo: id, Pensamento: s.Fonte.Pensamento, EsperaMax: esperaConfig})
		cancel()
		credencial.Zerar(chave)
	}
	return s.concluir(func() {
		if s.testes == nil {
			s.testes = map[string]ModeloIA{}
		}
		r := ModeloIA{Teste: "ok"}
		if err != nil {
			r = ModeloIA{Teste: "falhou", TesteErro: workflow.VisaoErro(erros.De(err))}
			if st := statusDoErro(err); st == StatusInvalida {
				s.status = st
			}
		}
		s.testes[id] = r
	})
}

// AbrirPaginaChaves abre o Google AI Studio no navegador padrão.
func (s *IAService) AbrirPaginaChaves() VisaoIA {
	var err error
	if s.AbrirURL != nil {
		err = s.AbrirURL(URLChavesAIStudio)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.erro = erros.Com(erros.InternoInesperado, err)
	}
	return s.visaoLocked()
}

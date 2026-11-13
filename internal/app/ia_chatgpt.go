package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/DiogoBrazil/sys-executor/internal/chatgpt"
	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/prefs"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// ContaChatGPT é a conexão com a conta (chatgpt.Conta; substituível nos testes).
type ContaChatGPT interface {
	inference.FonteToken
	Estado() (chatgpt.Estado, error)
	Entrar(ctx context.Context) error
	Desconectar(ctx context.Context) (confirmada bool, err error)
}

// FonteChatGPT implementa a fonte do provedor ChatGPT (plano do usuário;
// V4 §48.3, §48.7). Nenhum modelo é fixo no código.
type FonteChatGPT struct {
	Conta ContaChatGPT
	Loja  *prefs.Loja
	// Esforco: reasoning.effort ("" = padrão do modelo).
	Esforco string

	// Injetáveis nos testes; nil = Responses API real.
	NovoMotor func(inference.ConfigChatGPT) (MotorComUso, error)
	Listar    func(context.Context, inference.ConfigChatGPT) ([]inference.ModeloChatGPT, error)
	Testar    func(context.Context, inference.ConfigChatGPT) error

	mu     sync.Mutex
	ultimo MotorComUso
}

func (f *FonteChatGPT) novoMotor() func(inference.ConfigChatGPT) (MotorComUso, error) {
	if f.NovoMotor != nil {
		return f.NovoMotor
	}
	return func(c inference.ConfigChatGPT) (MotorComUso, error) { return inference.NovoMotorChatGPT(c) }
}

func (f *FonteChatGPT) listar() func(context.Context, inference.ConfigChatGPT) ([]inference.ModeloChatGPT, error) {
	if f.Listar != nil {
		return f.Listar
	}
	return inference.ListarModelosChatGPT
}

func (f *FonteChatGPT) testar() func(context.Context, inference.ConfigChatGPT) error {
	if f.Testar != nil {
		return f.Testar
	}
	return inference.TestarModeloChatGPT
}

// Alvo confere conta, permissão do plano e modelo, sem rede.
func (f *FonteChatGPT) Alvo() (inference.Alvo, error) {
	st, err := f.Conta.Estado()
	switch {
	case err != nil:
		return inference.Alvo{}, err
	case st.Reconectar:
		return inference.Alvo{}, erros.Novo(erros.IASessaoExpirada)
	case !st.Conectada:
		return inference.Alvo{}, erros.Novo(erros.IAContaDesconectada)
	case !st.PlanoAutorizado:
		return inference.Alvo{}, erros.Novo(erros.IAPlanoNaoAutorizado)
	}
	m := f.Loja.Obter().ModeloChatGPT
	if m == "" {
		return inference.Alvo{}, erros.Com(erros.IAModeloIndisponivel, errors.New("nenhum modelo escolhido em Configurações do assistente"))
	}
	return inference.Alvo{Provedor: inference.ProvedorChatGPT, Rotulo: inference.RotuloProvedor(inference.ProvedorChatGPT), Modelo: m}, nil
}

// Abrir prepara o motor para o alvo autorizado.
func (f *FonteChatGPT) Abrir(ctx context.Context, alvo inference.Alvo, atividade func(string), inspecionar func([]byte) error) (inference.Motor, func(), error) {
	if alvo.Provedor != inference.ProvedorChatGPT || alvo.Modelo == "" {
		return nil, nil, erros.Com(erros.InternoInesperado, errors.New("alvo de IA não é o ChatGPT"))
	}
	atividade("Conectando ao assistente")
	if _, err := f.Conta.Token(ctx); err != nil {
		return nil, nil, err
	}
	m, err := f.novoMotor()(inference.ConfigChatGPT{Modelo: alvo.Modelo, Tokens: f.Conta, Esforco: f.Esforco, Inspecionar: inspecionar})
	if err != nil {
		return nil, nil, err
	}
	f.mu.Lock()
	f.ultimo = m
	f.mu.Unlock()
	return m, m.Fechar, nil
}

// Uso da última tarefa com o ChatGPT (memória; nunca gravado).
func (f *FonteChatGPT) Uso() (inference.Uso, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ultimo == nil {
		return inference.Uso{}, false
	}
	return f.ultimo.Uso(), true
}

// ---- tela ----

// ModeloChatGPTIA é um modelo na tela, com o resultado do último teste.
type ModeloChatGPTIA struct {
	inference.ModeloChatGPT
	Teste     string              `json:"teste"`
	TesteErro *workflow.ErroVisao `json:"testeErro"`
}

// VisaoChatGPT é a parte do ChatGPT na tela de configurações. Nunca contém
// tokens (TRV-SEG-014).
type VisaoChatGPT struct {
	Disponivel        bool              `json:"disponivel"`
	Conectada         bool              `json:"conectada"`
	Email             string            `json:"email"`
	PlanoAutorizado   bool              `json:"planoAutorizado"`
	Reconectar        bool              `json:"reconectar"`
	Entrando          bool              `json:"entrando"`
	Modelos           []ModeloChatGPTIA `json:"modelos"`
	ModeloSelecionado string            `json:"modeloSelecionado"`
	ModeloForaDaLista bool              `json:"modeloForaDaLista"`
	Uso               *UsoIA            `json:"uso"`
	URLUso            string            `json:"urlUso"`
}

// URLUsoChatGPT: onde o usuário acompanha o uso do plano e os aplicativos
// conectados (Configurações do ChatGPT).
const URLUsoChatGPT = "https://chatgpt.com/#settings"

func (s *IAService) visaoChatGPTLocked() *VisaoChatGPT {
	if s.ChatGPT == nil {
		return &VisaoChatGPT{}
	}
	v := &VisaoChatGPT{Disponivel: true, URLUso: URLUsoChatGPT, ModeloSelecionado: s.ChatGPT.Loja.Obter().ModeloChatGPT}
	if st, err := s.ChatGPT.Conta.Estado(); err == nil {
		v.Conectada, v.Email, v.PlanoAutorizado, v.Reconectar, v.Entrando = st.Conectada, st.Email, st.PlanoAutorizado, st.Reconectar, st.Entrando
	}
	v.Entrando = v.Entrando || s.cancelarEntrada != nil
	achou := false
	for _, m := range s.cModelos {
		mi := ModeloChatGPTIA{ModeloChatGPT: m}
		if t, ok := s.cTestes[m.ID]; ok {
			mi.Teste, mi.TesteErro = t.Teste, t.TesteErro
		}
		v.Modelos = append(v.Modelos, mi)
		achou = achou || m.ID == v.ModeloSelecionado
	}
	v.ModeloForaDaLista = v.ModeloSelecionado != "" && len(s.cModelos) > 0 && !achou
	if u, ok := s.ChatGPT.Uso(); ok {
		v.Uso = &UsoIA{Modelo: u.Modelo, Chamadas: u.Chamadas, Falhas: u.Falhas, TokensEntrada: u.TokensEntrada,
			TokensSaida: u.TokensSaida, TokensPensamento: u.TokensPensamento, Segundos: int(u.Duracao.Seconds())}
	}
	return v
}

func (s *IAService) semChatGPT() VisaoIA {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.visaoLocked()
	v.Erro = workflow.VisaoErro(erros.Novo(erros.IAContaDesconectada))
	return v
}

// SelecionarProvedor grava o serviço de IA usado nas análises (escolha
// explícita, V4 §48.1; não há troca automática).
func (s *IAService) SelecionarProvedor(provedor string) VisaoIA {
	var err error
	switch provedor {
	case prefs.ProvedorGemini, prefs.ProvedorChatGPT:
		if provedor == prefs.ProvedorChatGPT && s.ChatGPT == nil {
			return s.semChatGPT()
		}
		if _, e := s.Fonte.Loja.Atualizar(func(p *prefs.Preferencias) { p.ProvedorIA = provedor }); e != nil {
			err = erros.Com(erros.InternoInesperado, e)
		}
	default:
		err = erros.Novo(erros.TarefaOrdemInvalida)
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

// EntrarChatGPT abre o login no navegador do sistema e volta na hora; o
// resultado chega pelo evento ia:visao. Uma entrada por vez.
func (s *IAService) EntrarChatGPT() VisaoIA {
	if s.ChatGPT == nil {
		return s.semChatGPT()
	}
	s.mu.Lock()
	if s.cancelarEntrada != nil {
		v := s.visaoLocked()
		s.mu.Unlock()
		return v
	}
	ctx, cancelar := context.WithCancel(context.Background())
	s.cancelarEntrada, s.erro, s.aviso = cancelar, nil, ""
	v := s.visaoLocked()
	s.mu.Unlock()
	go func() {
		defer cancelar()
		err := s.ChatGPT.Conta.Entrar(ctx)
		var lista []inference.ModeloChatGPT
		if err == nil {
			lctx, c := context.WithTimeout(context.Background(), tempoRede)
			lista, err = s.ChatGPT.listar()(lctx, inference.ConfigChatGPT{Tokens: s.ChatGPT.Conta})
			c()
		}
		s.mu.Lock()
		s.cancelarEntrada = nil
		if err != nil {
			s.erro = erros.De(err)
		} else {
			s.cModelos, s.cTestes = lista, map[string]ModeloChatGPTIA{}
		}
		vv := s.visaoLocked()
		s.mu.Unlock()
		s.emitir(vv)
	}()
	return s.emitir(v)
}

// CancelarEntradaChatGPT interrompe a espera pelo navegador.
func (s *IAService) CancelarEntradaChatGPT() VisaoIA {
	s.mu.Lock()
	if s.cancelarEntrada != nil {
		s.cancelarEntrada()
	}
	v := s.visaoLocked()
	s.mu.Unlock()
	return v
}

// DesconectarChatGPT revoga a sessão (quando possível) e apaga a credencial.
func (s *IAService) DesconectarChatGPT() VisaoIA {
	if s.ChatGPT == nil {
		return s.semChatGPT()
	}
	if !s.iniciar() {
		return s.ocupadoAgora()
	}
	ctx, c := context.WithTimeout(context.Background(), 30*time.Second)
	confirmada, err := s.ChatGPT.Conta.Desconectar(ctx)
	c()
	return s.concluir(func() {
		s.cModelos, s.cTestes = nil, nil
		if err != nil {
			s.erro = erros.De(err)
		}
		if !confirmada {
			s.aviso = "A conta foi desconectada neste computador, mas o ChatGPT não confirmou o encerramento da sessão. Desconecte o sys-executor também nas configurações do ChatGPT (Aplicativos conectados)."
		}
	})
}

// AtualizarModelosChatGPT lista os modelos da conta (sem gerar texto).
func (s *IAService) AtualizarModelosChatGPT() VisaoIA {
	if s.ChatGPT == nil {
		return s.semChatGPT()
	}
	if !s.iniciar() {
		return s.ocupadoAgora()
	}
	ctx, c := context.WithTimeout(context.Background(), tempoRede)
	lista, err := s.ChatGPT.listar()(ctx, inference.ConfigChatGPT{Tokens: s.ChatGPT.Conta})
	c()
	return s.concluir(func() {
		if err != nil {
			s.erro = erros.De(err)
			return
		}
		s.cModelos, s.cTestes = lista, map[string]ModeloChatGPTIA{}
	})
}

// SelecionarModeloChatGPT grava o modelo e escolhe o ChatGPT como serviço.
func (s *IAService) SelecionarModeloChatGPT(id string) VisaoIA {
	if s.ChatGPT == nil {
		return s.semChatGPT()
	}
	s.mu.Lock()
	conhecido := false
	for _, m := range s.cModelos {
		conhecido = conhecido || m.ID == id
	}
	s.mu.Unlock()
	var err error
	if !conhecido {
		err = erros.Com(erros.IAModeloIndisponivel, errors.New("modelo fora da lista da conta"))
	} else if _, e := s.ChatGPT.Loja.Atualizar(func(p *prefs.Preferencias) {
		p.ModeloChatGPT, p.ProvedorIA = id, prefs.ProvedorChatGPT
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

// TestarModeloChatGPT faz uma geração mínima, sem dados dos autos.
func (s *IAService) TestarModeloChatGPT(id string) VisaoIA {
	if s.ChatGPT == nil {
		return s.semChatGPT()
	}
	if !s.iniciar() {
		return s.ocupadoAgora()
	}
	ctx, c := context.WithTimeout(context.Background(), tempoRede)
	err := s.ChatGPT.testar()(ctx, inference.ConfigChatGPT{Modelo: id, Tokens: s.ChatGPT.Conta})
	c()
	return s.concluir(func() {
		if s.cTestes == nil {
			s.cTestes = map[string]ModeloChatGPTIA{}
		}
		r := ModeloChatGPTIA{Teste: "ok"}
		if err != nil {
			r.Teste, r.TesteErro = "falhou", workflow.VisaoErro(erros.De(err))
		}
		s.cTestes[id] = r
	})
}

// AbrirUsoChatGPT abre as configurações do ChatGPT (uso do plano e
// aplicativos conectados) no navegador do sistema.
func (s *IAService) AbrirUsoChatGPT() VisaoIA {
	if s.AbrirURL != nil {
		_ = s.AbrirURL(URLUsoChatGPT)
	}
	return s.Estado()
}

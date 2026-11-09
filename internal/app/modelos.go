package app

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/inference"
	"github.com/DiogoBrazil/sys-executor/internal/prefs"
	"github.com/DiogoBrazil/sys-executor/internal/workflow"
)

// EventoModelos leva o andamento de download/importação à interface.
const EventoModelos = "modelos:visao"

// Executaveis localiza os builds do llama-server.
type Executaveis struct {
	CPU func() (string, error)
	GPU func() (string, error) // build Vulkan; erro = não instalado
}

// MotorLocal é a FonteMotor do pipeline: escolhe o modelo, confere a
// memória e sobe o sidecar (V4 §§37–38).
type MotorLocal struct {
	Modelos *inference.GerenciadorModelos
	Loja    *prefs.Loja
	Exe     Executaveis
	// Hardware (testes substituem).
	Hardware func() (inference.Hardware, error)
}

func (m *MotorLocal) hardware() (inference.Hardware, error) {
	if m.Hardware != nil {
		return m.Hardware()
	}
	return inference.DetectarHardware()
}

// Escolher devolve o modelo e o plano que a tarefa usaria agora.
func (m *MotorLocal) Escolher() (inference.Instalado, inference.Plano, error) {
	inst, err := m.Modelos.Listar()
	if err != nil {
		return inference.Instalado{}, inference.Plano{}, erros.Com(erros.ArquivoLeitura, err)
	}
	var geradores []inference.Instalado
	for _, i := range inst {
		if i.Modelo.Papel == inference.PapelGerador {
			geradores = append(geradores, i)
		}
	}
	if len(geradores) == 0 {
		return inference.Instalado{}, inference.Plano{}, erros.Novo(erros.ModeloAusente)
	}
	hw, err := m.hardware()
	if err != nil {
		return inference.Instalado{}, inference.Plano{}, erros.Com(erros.InternoInesperado, err)
	}
	porID := map[string]inference.Instalado{}
	for _, i := range geradores {
		porID[i.Modelo.ID] = i
	}
	if manual := m.Loja.Obter().ModeloManual; manual != "" {
		if _, ok := porID[manual]; ok {
			p, _, err := inference.PlanejarManual(hw, m.Modelos.Catalogo, geradores, manual)
			return porID[manual], p, err
		}
	}
	r, err := inference.Recomendar(hw, m.Modelos.Catalogo, geradores)
	if err != nil {
		return inference.Instalado{}, inference.Plano{}, err
	}
	if !r.Instalado {
		// Nenhum instalado cabe; a recomendação seria baixar outro (T15).
		return inference.Instalado{}, r.Plano, erros.Com(erros.ModeloSemMemoria, fmt.Errorf("nenhum modelo instalado cabe; recomendado %s", r.Plano.Modelo.ID))
	}
	return porID[r.Plano.Modelo.ID], r.Plano, nil
}

// Abrir implementa pipeline.FonteMotor.
func (m *MotorLocal) Abrir(ctx context.Context, atividade func(string)) (inference.Motor, func(), error) {
	inst, plano, err := m.Escolher()
	if err != nil {
		return nil, nil, err
	}
	liberar := m.Modelos.Reservar(inst.Modelo.ID)
	cfg := inference.ConfigServidor{Modelo: inst.Caminho, Contexto: plano.Contexto, Threads: plano.Threads}
	if plano.BytesLivres > uint64(inst.Modelo.Tamanho) {
		cfg.LimiteMemoria = plano.BytesLivres - uint64(inst.Modelo.Tamanho) // protege contra estouro (D-09.2)
	}
	// T14: GPU só se a sonda segura já passou; falhando, segue em CPU.
	if m.Loja.Obter().GPU.Estado == prefs.GPUEstavel && m.Exe.GPU != nil {
		if exe, err := m.Exe.GPU(); err == nil {
			atividade("Carregando o modelo de IA na placa de vídeo")
			c := cfg
			c.Executavel, c.CamadasGPU = exe, 99
			if s, err := inference.IniciarServidor(ctx, c); err == nil {
				return s, func() { s.Fechar(); liberar() }, nil
			}
			if ctx.Err() != nil {
				liberar()
				return nil, nil, ctx.Err()
			}
		}
	}
	exe, err := m.Exe.CPU()
	if err != nil {
		liberar()
		return nil, nil, err
	}
	cfg.Executavel = exe
	atividade("Carregando o modelo de IA (" + inst.Modelo.Nome + ")")
	s, err := inference.IniciarServidor(ctx, cfg)
	if err != nil {
		liberar()
		return nil, nil, err
	}
	return s, func() { s.Fechar(); liberar() }, nil
}

// ---- serviço da tela Modelos ----

// ModeloVisao é um modelo listado na interface.
type ModeloVisao struct {
	ID          string `json:"id"`
	Nome        string `json:"nome"`
	Perfil      string `json:"perfil"`
	Papel       string `json:"papel"`
	Tamanho     int64  `json:"tamanho"`
	Licenca     string `json:"licenca"`
	Testado     bool   `json:"testado"`
	Nota        string `json:"nota"`
	Instalado   bool   `json:"instalado"`
	Recomendado bool   `json:"recomendado"`
	Selecionado bool   `json:"selecionado"`
	Cabe        bool   `json:"cabe"`
	Contexto    int    `json:"contexto"`
}

// AndamentoModelos é o progresso de download ou importação.
type AndamentoModelos struct {
	ModeloID string `json:"modeloId"`
	Acao     string `json:"acao"` // baixando | importando
	Feito    int64  `json:"feito"`
	Total    int64  `json:"total"`
}

// VisaoModelos é a tela Modelos.
type VisaoModelos struct {
	Hardware  inference.Hardware  `json:"hardware"`
	Modelos   []ModeloVisao       `json:"modelos"`
	Avisos    []string            `json:"avisos"`
	Manual    string              `json:"manual"`
	GPU       string              `json:"gpu"`
	GPUs      []inference.GPU     `json:"gpus"`
	Andamento *AndamentoModelos   `json:"andamento"`
	Pronto    bool                `json:"pronto"` // há modelo instalado que cabe
	Servidor  bool                `json:"servidor"`
	Erro      *workflow.ErroVisao `json:"erro"`
}

// ModelosService expõe o gerenciador de modelos (T27; MOD-01…04).
type ModelosService struct {
	Motor   *MotorLocal
	Emitir  func(*VisaoModelos)
	Seletor func() (string, error) // diálogo para escolher um .gguf

	mu        sync.Mutex
	andamento *AndamentoModelos
	cancelar  context.CancelFunc
	ultimoErr *erros.Erro
	gpus      []inference.GPU
}

func (s *ModelosService) emitir() {
	if s.Emitir != nil {
		v := s.Estado()
		s.Emitir(&v)
	}
}

// Estado monta a visão da tela.
func (s *ModelosService) Estado() VisaoModelos {
	m := s.Motor
	v := VisaoModelos{Manual: m.Loja.Obter().ModeloManual, GPU: string(m.Loja.Obter().GPU.Estado)}
	s.mu.Lock()
	if s.andamento != nil {
		a := *s.andamento
		v.Andamento = &a
	}
	v.Erro, v.GPUs = workflow.VisaoErro(s.ultimoErr), s.gpus
	s.mu.Unlock()
	if _, err := m.Exe.CPU(); err == nil {
		v.Servidor = true
	}
	hw, err := m.hardware()
	if err != nil {
		v.Erro = workflow.VisaoErro(erros.Com(erros.InternoInesperado, err))
		return v
	}
	v.Hardware = hw
	inst, _ := m.Modelos.Listar()
	porID := map[string]inference.Instalado{}
	for _, i := range inst {
		porID[i.Modelo.ID] = i
	}
	rec, errRec := inference.Recomendar(hw, m.Modelos.Catalogo, inst)
	v.Avisos = rec.Avisos
	if errRec != nil {
		v.Avisos = append(v.Avisos, erros.De(errRec).Mensagem)
	}
	add := func(md inference.Modelo, i *inference.Instalado) {
		mv := ModeloVisao{ID: md.ID, Nome: md.Nome, Perfil: md.Perfil, Papel: md.Papel, Tamanho: md.Tamanho, Licenca: md.Licenca,
			Testado: md.Testado, Nota: md.Nota, Instalado: i != nil, Selecionado: v.Manual == md.ID,
			Recomendado: errRec == nil && rec.Plano.Modelo.ID == md.ID}
		var meta *inference.MetadadosGGUF
		if i != nil {
			meta = &i.Meta
		}
		if p, err := inference.Planejar(hw, md, meta); err == nil {
			mv.Cabe, mv.Contexto = true, p.Contexto
			if i != nil && md.Papel == inference.PapelGerador {
				v.Pronto = true
			}
		}
		v.Modelos = append(v.Modelos, mv)
	}
	for _, md := range m.Modelos.Catalogo.Modelos {
		if i, ok := porID[md.ID]; ok {
			add(md, &i)
		} else {
			add(md, nil)
		}
	}
	for _, i := range inst {
		if _, doCat := m.Modelos.Catalogo.PorID(i.Modelo.ID); !doCat {
			i := i
			add(i.Modelo, &i)
		}
	}
	return v
}

func (s *ModelosService) iniciar(id, acao string, f func(ctx context.Context, prog inference.Progresso) error) VisaoModelos {
	s.mu.Lock()
	if s.andamento != nil {
		s.mu.Unlock()
		v := s.Estado()
		v.Erro = workflow.VisaoErro(erros.Com(erros.TarefaOrdemInvalida, errors.New("download em andamento")))
		return v
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.andamento, s.cancelar, s.ultimoErr = &AndamentoModelos{ModeloID: id, Acao: acao}, cancel, nil
	s.mu.Unlock()
	go func() {
		defer cancel()
		err := f(ctx, func(feito, total int64) {
			s.mu.Lock()
			if s.andamento != nil {
				s.andamento.Feito, s.andamento.Total = feito, total
			}
			s.mu.Unlock()
			s.emitir()
		})
		s.mu.Lock()
		s.andamento, s.cancelar = nil, nil
		if err != nil && !errors.Is(err, context.Canceled) {
			s.ultimoErr = erros.De(err)
		}
		s.mu.Unlock()
		s.emitir()
	}()
	return s.Estado()
}

// Baixar inicia o download orientado de um modelo do catálogo (T27).
func (s *ModelosService) Baixar(id string) VisaoModelos {
	return s.iniciar(id, "baixando", func(ctx context.Context, prog inference.Progresso) error {
		_, err := s.Motor.Modelos.Baixar(ctx, id, prog)
		return err
	})
}

// Importar copia um GGUF escolhido pelo usuário (MOD-03).
func (s *ModelosService) Importar() VisaoModelos {
	if s.Seletor == nil {
		return s.Estado()
	}
	caminho, err := s.Seletor()
	if err != nil || caminho == "" {
		return s.Estado()
	}
	return s.iniciar("", "importando", func(ctx context.Context, prog inference.Progresso) error {
		_, err := s.Motor.Modelos.Importar(ctx, caminho, prog)
		return err
	})
}

// CancelarAndamento interrompe o download ou a importação (o parcial do
// download fica para retomar).
func (s *ModelosService) CancelarAndamento() VisaoModelos {
	s.mu.Lock()
	if s.cancelar != nil {
		s.cancelar()
	}
	s.mu.Unlock()
	return s.Estado()
}

// Remover apaga um modelo instalado (MOD-04).
func (s *ModelosService) Remover(id string) VisaoModelos {
	err := s.Motor.Modelos.Remover(id)
	if err == nil && s.Motor.Loja.Obter().ModeloManual == id {
		_, err = s.Motor.Loja.Atualizar(func(p *prefs.Preferencias) { p.ModeloManual = "" })
	}
	v := s.Estado()
	if err != nil {
		v.Erro = workflow.VisaoErro(erros.De(err))
	}
	return v
}

// Selecionar fixa um modelo (seleção manual avançada); "" volta ao automático.
func (s *ModelosService) Selecionar(id string) VisaoModelos {
	_, err := s.Motor.Loja.Atualizar(func(p *prefs.Preferencias) { p.ModeloManual = id })
	v := s.Estado()
	if err != nil {
		v.Erro = workflow.VisaoErro(erros.Com(erros.ArquivoEscrita, err))
	}
	return v
}

// TestarGPU roda a sonda segura (avançado; D-09.1): grava a sentinela,
// lista os dispositivos e faz uma carga curta com o menor gerador
// instalado. Queda do app durante a sonda deixa a GPU marcada instável.
func (s *ModelosService) TestarGPU() VisaoModelos {
	m := s.Motor
	falha := func(err error) VisaoModelos {
		_, _ = m.Loja.Atualizar(func(p *prefs.Preferencias) { p.GPU = prefs.GPU{Estado: prefs.GPUInstavel} })
		v := s.Estado()
		v.Erro = workflow.VisaoErro(erros.De(err))
		return v
	}
	if m.Exe.GPU == nil {
		return falha(erros.Com(erros.ModeloGPUInstavel, inference.ErrSemGPU))
	}
	exe, err := m.Exe.GPU()
	if err != nil {
		return falha(erros.Com(erros.ModeloGPUInstavel, err))
	}
	inst, _ := m.Modelos.Listar()
	var menor *inference.Instalado
	for k := range inst {
		if inst[k].Modelo.Papel == inference.PapelGerador && (menor == nil || inst[k].Modelo.Tamanho < menor.Modelo.Tamanho) {
			menor = &inst[k]
		}
	}
	if menor == nil {
		v := s.Estado()
		v.Erro = workflow.VisaoErro(erros.Novo(erros.ModeloAusente))
		return v
	}
	if _, err := m.Loja.Atualizar(func(p *prefs.Preferencias) { p.GPU.SondaEmAndamento = true }); err != nil {
		v := s.Estado()
		v.Erro = workflow.VisaoErro(erros.Com(erros.ArquivoEscrita, err))
		return v
	}
	ctx := context.Background()
	gs, err := inference.ListarDispositivos(ctx, exe)
	if err == nil && len(gs) == 0 {
		err = erros.Com(erros.ModeloGPUInstavel, errors.New("nenhuma GPU utilizável"))
	}
	if err == nil {
		err = inference.SondarGPU(ctx, exe, menor.Caminho)
	}
	if err != nil {
		return falha(err)
	}
	s.mu.Lock()
	s.gpus = gs
	s.mu.Unlock()
	_, _ = m.Loja.Atualizar(func(p *prefs.Preferencias) { p.GPU = prefs.GPU{Estado: prefs.GPUEstavel} })
	return s.Estado()
}

// UsarSoCPU desativa a GPU (o usuário prefere CPU).
func (s *ModelosService) UsarSoCPU() VisaoModelos {
	_, _ = s.Motor.Loja.Atualizar(func(p *prefs.Preferencias) { p.GPU = prefs.GPU{Estado: prefs.GPUDesativada} })
	return s.Estado()
}

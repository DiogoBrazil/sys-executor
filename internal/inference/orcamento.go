package inference

import (
	"fmt"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

const (
	mib = 1 << 20
	gib = 1 << 30

	// MargemSistema reserva memória para o app, o WebView2, o OCR e o índice
	// dos autos (plano §9: "+ app + margem").
	MargemSistema = 1536 * mib

	// ContextoMinimo absoluto: abaixo disso as etapas não cabem nem fatiadas.
	ContextoMinimo = 4096

	contextoReferencia = 8192 // contexto das medições de privado_mb
)

// Plano é a configuração escolhida para executar um modelo.
type Plano struct {
	Modelo         Modelo `json:"modelo"`
	Contexto       int    `json:"contexto"`
	FaixaMin       int    `json:"faixaMin"`
	FaixaMax       int    `json:"faixaMax"`
	Reduzido       bool   `json:"reduzido"` // abaixo da faixa por falta de memória livre
	Threads        int    `json:"threads"`
	CamadasGPU     int    `json:"camadasGPU"`
	BytesEstimados uint64 `json:"bytesEstimados"`
	BytesLivres    uint64 `json:"bytesLivres"`
}

// FaixaContexto é a faixa inicial da V4 §38.3 pela RAM total.
func FaixaContexto(ramTotal uint64) (min, max int) {
	switch {
	case ramTotal < 12*gib:
		return 4096, 8192
	case ramTotal < 24*gib:
		return 8192, 16384
	}
	return 16384, 32768
}

// estimar soma pesos (mmap) + pico privado (com o KV ajustado ao contexto) +
// margem do sistema (D-09.2).
func estimar(m Modelo, meta *MetadadosGGUF, ctx int) uint64 {
	privado := uint64(m.PrivadoMB) * mib
	if meta != nil && meta.Camadas > 0 {
		ref, alvo := uint64(meta.BytesKV(contextoReferencia)), uint64(meta.BytesKV(ctx))
		if ref < privado {
			privado = privado - ref + alvo
		}
	} else if ctx > contextoReferencia {
		privado = privado * uint64(ctx) / contextoReferencia // sem metadados: proporcional (conservador)
	}
	return uint64(m.Tamanho) + privado + MargemSistema
}

// Planejar escolhe o maior contexto da faixa que cabe na memória livre; se
// nada da faixa couber, reduz até ContextoMinimo (V4 §38.3: memória
// efetivamente disponível). Nada cabendo → MOD-003.
func Planejar(hw Hardware, m Modelo, meta *MetadadosGGUF) (Plano, error) {
	fmin, fmax := FaixaContexto(hw.RAMTotal)
	p := Plano{Modelo: m, FaixaMin: fmin, FaixaMax: fmax, Threads: hw.NucleosFisicos, BytesLivres: hw.RAMLivre}
	if p.Threads < 1 {
		p.Threads = 1
	}
	limite := fmax
	if meta != nil && meta.ContextoTreino > 0 && int(meta.ContextoTreino) < limite {
		limite = int(meta.ContextoTreino)
	}
	for ctx := limite; ctx >= ContextoMinimo; ctx /= 2 {
		if est := estimar(m, meta, ctx); est <= hw.RAMLivre {
			p.Contexto, p.BytesEstimados, p.Reduzido = ctx, est, ctx < fmin
			return p, nil
		}
	}
	p.BytesEstimados = estimar(m, meta, ContextoMinimo)
	return p, erros.Com(erros.ModeloSemMemoria, fmt.Errorf("%s: estimado %d MiB, livre %d MiB",
		m.ID, p.BytesEstimados/mib, hw.RAMLivre/mib))
}

// Instalado é um modelo presente na pasta de modelos.
type Instalado struct {
	Modelo  Modelo
	Caminho string
	Meta    MetadadosGGUF
}

// Recomendacao é a escolha automática (V4 §37).
type Recomendacao struct {
	Plano     Plano  `json:"plano"`
	Instalado bool   `json:"instalado"`
	Caminho   string `json:"-"`
	// Melhor é um modelo testado melhor que cabe, mas não está instalado.
	Melhor       *Modelo  `json:"melhor,omitempty"`
	Alternativas []Plano  `json:"alternativas"`
	Avisos       []string `json:"avisos"`
}

// Recomendar escolhe, entre os geradores com perfil testado que cabem nesta
// máquina, o melhor já instalado; sem nenhum instalado, o melhor para baixar
// (T27). Nenhum cabendo → MOD-006 (T15). GGUF importado fora do catálogo só
// entra por seleção manual.
func Recomendar(hw Hardware, cat CatalogoModelos, inst []Instalado) (Recomendacao, error) {
	r := Recomendacao{Avisos: hw.Avisos()}
	porID := map[string]Instalado{}
	for _, i := range inst {
		porID[i.Modelo.ID] = i
	}
	var viaveis []Plano
	for _, m := range cat.Geradores() {
		if !m.Testado {
			continue
		}
		var meta *MetadadosGGUF
		if i, ok := porID[m.ID]; ok {
			meta = &i.Meta
		}
		if p, err := Planejar(hw, m, meta); err == nil {
			viaveis = append(viaveis, p)
		}
	}
	if len(viaveis) == 0 {
		return r, erros.Com(erros.ModeloHardwareFraco, fmt.Errorf("livre %d MiB, total %d MiB", hw.RAMLivre/mib, hw.RAMTotal/mib))
	}
	escolhido := -1
	for k, p := range viaveis {
		if _, ok := porID[p.Modelo.ID]; ok {
			escolhido = k
			break
		}
	}
	if escolhido < 0 {
		r.Plano = viaveis[0]
	} else {
		r.Plano, r.Instalado, r.Caminho = viaveis[escolhido], true, porID[viaveis[escolhido].Modelo.ID].Caminho
		if escolhido > 0 {
			m := viaveis[0].Modelo
			r.Melhor = &m
		}
	}
	for _, p := range viaveis {
		if p.Modelo.ID != r.Plano.Modelo.ID {
			r.Alternativas = append(r.Alternativas, p)
		}
	}
	if r.Plano.Reduzido {
		r.Avisos = append(r.Avisos, fmt.Sprintf("Pouca memória livre: o contexto foi reduzido para %d tokens; autos longos serão analisados em mais partes.", r.Plano.Contexto))
	}
	return r, nil
}

// PlanejarManual é a seleção avançada (V4 §37): qualquer instalado, inclusive
// importado sem perfil testado. Se não couber, devolve MOD-003 e as
// alternativas que cabem (V4 §38.4, T15).
func PlanejarManual(hw Hardware, cat CatalogoModelos, inst []Instalado, id string) (Plano, []Plano, error) {
	var alvo *Instalado
	for k := range inst {
		if inst[k].Modelo.ID == id {
			alvo = &inst[k]
		}
	}
	if alvo == nil {
		return Plano{}, nil, erros.Com(erros.ModeloAusente, fmt.Errorf("modelo %s não instalado", id))
	}
	p, err := Planejar(hw, alvo.Modelo, &alvo.Meta)
	if err == nil {
		return p, nil, nil
	}
	var alt []Plano
	for _, i := range inst {
		if i.Modelo.ID == id || i.Modelo.Papel != PapelGerador {
			continue
		}
		if q, e := Planejar(hw, i.Modelo, &i.Meta); e == nil {
			alt = append(alt, q)
		}
	}
	return p, alt, err
}

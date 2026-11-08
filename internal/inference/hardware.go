package inference

import (
	"runtime"

	"golang.org/x/sys/cpu"
)

// GPU detectada por um backend utilizável (só após consentimento; D-08).
type GPU struct {
	Nome      string `json:"nome"`
	Backend   string `json:"backend"` // ex.: Vulkan
	VRAMTotal uint64 `json:"vramTotal"`
	VRAMLivre uint64 `json:"vramLivre"`
}

// Hardware é o que o orçamento considera (V4 §§37, 38.1).
type Hardware struct {
	NucleosFisicos int    `json:"nucleosFisicos"`
	NucleosLogicos int    `json:"nucleosLogicos"`
	AVX2           bool   `json:"avx2"`
	RAMTotal       uint64 `json:"ramTotal"`
	RAMLivre       uint64 `json:"ramLivre"`
	GPUs           []GPU  `json:"gpus"`
}

// DetectarHardware lê CPU e memória. GPUs nunca são sondadas aqui: a sonda
// é opcional, manual e com sentinela (D-08, D-09.1).
func DetectarHardware() (Hardware, error) {
	h := Hardware{NucleosLogicos: runtime.NumCPU(), AVX2: cpu.X86.HasAVX2}
	var err error
	h.RAMTotal, h.RAMLivre, err = memoriaSistema()
	if err != nil {
		return h, err
	}
	h.NucleosFisicos = nucleosFisicos()
	if h.NucleosFisicos <= 0 {
		h.NucleosFisicos = (h.NucleosLogicos + 1) / 2
	}
	return h, nil
}

// Avisos de perfil mínimo (V4 §38.1: 4 núcleos físicos, AVX2 preferencial).
// Não bloqueiam: o orçamento de memória decide a viabilidade.
func (h Hardware) Avisos() []string {
	var a []string
	if h.NucleosFisicos < 4 {
		a = append(a, "O processador tem menos de 4 núcleos físicos; a análise pode ficar muito lenta.")
	}
	if !h.AVX2 {
		a = append(a, "O processador não tem AVX2; a análise pode ficar muito lenta.")
	}
	return a
}

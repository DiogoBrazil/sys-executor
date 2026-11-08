package inference

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
)

// Limites contra arquivo malicioso ou corrompido.
const (
	maxTextoGGUF  = 1 << 20    // 1 MiB por string
	maxItensGGUF  = 1 << 24    // itens por array
	maxChavesGGUF = 1 << 16    // pares chave/valor
	magicGGUF     = 0x46554747 // "GGUF" little-endian
)

// ErrNaoGGUF: o arquivo não é um GGUF válido.
var ErrNaoGGUF = errors.New("arquivo não é um modelo GGUF válido")

// MetadadosGGUF são os campos do cabeçalho usados pelo gerenciador.
type MetadadosGGUF struct {
	Versao         uint32
	Tensores       uint64
	Arquitetura    string
	Nome           string
	Licenca        string
	TipoArquivo    uint32 // general.file_type (15 = Q4_K_M, 7 = Q8_0…)
	Camadas        uint32 // <arq>.block_count
	ContextoTreino uint32 // <arq>.context_length
	Embedding      uint32 // <arq>.embedding_length
	CabecasAtencao uint32 // <arq>.attention.head_count
	CabecasKV      uint32 // <arq>.attention.head_count_kv
	DimChave       uint32 // <arq>.attention.key_length
	DimValor       uint32 // <arq>.attention.value_length
	IntervaloAtenc uint32 // <arq>.full_attention_interval (modelos híbridos)
	Pooling        uint32 // <arq>.pooling_type (embeddings)
	TemplateDeChat bool
}

// LerGGUF lê o cabeçalho (metadados) sem carregar tensores.
func LerGGUF(r io.Reader) (MetadadosGGUF, error) {
	br := &leitorGGUF{r: bufio.NewReaderSize(r, 1<<16)}
	var m MetadadosGGUF
	if br.u32() != magicGGUF {
		return m, ErrNaoGGUF
	}
	m.Versao = br.u32()
	if br.err == nil && (m.Versao < 2 || m.Versao > 3) {
		return m, fmt.Errorf("%w: versão %d não suportada", ErrNaoGGUF, m.Versao)
	}
	m.Tensores = br.u64()
	nkv := br.u64()
	if br.err != nil {
		return m, fmt.Errorf("%w: %v", ErrNaoGGUF, br.err)
	}
	if nkv > maxChavesGGUF || m.Tensores == 0 {
		return m, fmt.Errorf("%w: cabeçalho inconsistente", ErrNaoGGUF)
	}
	kv := map[string]any{}
	for i := uint64(0); i < nkv && br.err == nil; i++ {
		k := br.texto()
		t := br.u32()
		v := br.valor(t, 0)
		if br.err == nil {
			kv[k] = v
		}
	}
	if br.err != nil {
		return m, fmt.Errorf("%w: %v", ErrNaoGGUF, br.err)
	}
	m.Arquitetura, _ = kv["general.architecture"].(string)
	if m.Arquitetura == "" {
		return m, fmt.Errorf("%w: sem general.architecture", ErrNaoGGUF)
	}
	m.Nome, _ = kv["general.name"].(string)
	m.Licenca, _ = kv["general.license"].(string)
	m.TipoArquivo = inteiro(kv["general.file_type"])
	a := m.Arquitetura + "."
	m.Camadas = inteiro(kv[a+"block_count"])
	m.ContextoTreino = inteiro(kv[a+"context_length"])
	m.Embedding = inteiro(kv[a+"embedding_length"])
	m.CabecasAtencao = inteiro(kv[a+"attention.head_count"])
	m.CabecasKV = inteiro(kv[a+"attention.head_count_kv"])
	if m.CabecasKV == 0 {
		m.CabecasKV = m.CabecasAtencao
	}
	m.DimChave = inteiro(kv[a+"attention.key_length"])
	m.DimValor = inteiro(kv[a+"attention.value_length"])
	if m.DimChave == 0 && m.CabecasAtencao > 0 {
		m.DimChave = m.Embedding / m.CabecasAtencao
	}
	if m.DimValor == 0 {
		m.DimValor = m.DimChave
	}
	m.IntervaloAtenc = inteiro(kv[a+"full_attention_interval"])
	m.Pooling = inteiro(kv[a+"pooling_type"])
	_, m.TemplateDeChat = kv["tokenizer.chat_template"].(string)
	return m, nil
}

// CamadasComKV é o número de camadas que guardam KV-cache. Modelos híbridos
// (atenção completa a cada N camadas) só guardam KV nessas camadas; as demais
// têm estado recorrente de tamanho fixo.
func (m MetadadosGGUF) CamadasComKV() uint32 {
	if m.IntervaloAtenc > 1 {
		return (m.Camadas + m.IntervaloAtenc - 1) / m.IntervaloAtenc
	}
	return m.Camadas
}

// BytesKV estima o KV-cache para n_ctx tokens em F16 (2 bytes por elemento).
func (m MetadadosGGUF) BytesKV(nCtx int) int64 {
	porToken := int64(m.CamadasComKV()) * int64(m.CabecasKV) * int64(m.DimChave+m.DimValor) * 2
	return porToken * int64(nCtx)
}

func inteiro(v any) uint32 {
	switch x := v.(type) {
	case uint32:
		return x
	case int32:
		if x > 0 {
			return uint32(x)
		}
	case uint64:
		if x <= math.MaxUint32 {
			return uint32(x)
		}
	case int64:
		if x > 0 && x <= math.MaxUint32 {
			return uint32(x)
		}
	case uint16:
		return uint32(x)
	case uint8:
		return uint32(x)
	}
	return 0
}

type leitorGGUF struct {
	r   *bufio.Reader
	err error
	buf [8]byte
}

func (l *leitorGGUF) ler(n int) []byte {
	if l.err != nil {
		return l.buf[:n]
	}
	_, l.err = io.ReadFull(l.r, l.buf[:n])
	return l.buf[:n]
}

func (l *leitorGGUF) u32() uint32 { return binary.LittleEndian.Uint32(l.ler(4)) }
func (l *leitorGGUF) u64() uint64 { return binary.LittleEndian.Uint64(l.ler(8)) }

func (l *leitorGGUF) texto() string {
	n := l.u64()
	if l.err != nil {
		return ""
	}
	if n > maxTextoGGUF {
		l.err = errors.New("string grande demais")
		return ""
	}
	b := make([]byte, n)
	_, l.err = io.ReadFull(l.r, b)
	return strings.ToValidUTF8(string(b), "�")
}

func (l *leitorGGUF) pular(n uint64) {
	if l.err == nil {
		_, l.err = l.r.Discard(int(n))
	}
}

// valor lê um valor do tipo t. Arrays são descartados (só o tamanho importa).
func (l *leitorGGUF) valor(t uint32, prof int) any {
	switch t {
	case 0:
		return l.ler(1)[0]
	case 1:
		return int8(l.ler(1)[0])
	case 2:
		return binary.LittleEndian.Uint16(l.ler(2))
	case 3:
		return int16(binary.LittleEndian.Uint16(l.ler(2)))
	case 4:
		return l.u32()
	case 5:
		return int32(l.u32())
	case 6:
		return math.Float32frombits(l.u32())
	case 7:
		return l.ler(1)[0] != 0
	case 8:
		return l.texto()
	case 9:
		if prof > 2 {
			l.err = errors.New("array aninhado demais")
			return nil
		}
		et := l.u32()
		n := l.u64()
		if l.err != nil {
			return nil
		}
		if n > maxItensGGUF {
			l.err = errors.New("array grande demais")
			return nil
		}
		if tam := tamanhoFixo(et); tam > 0 {
			l.pular(n * uint64(tam))
			return nil
		}
		for i := uint64(0); i < n && l.err == nil; i++ {
			l.valor(et, prof+1)
		}
		return nil
	case 10:
		return l.u64()
	case 11:
		return int64(l.u64())
	case 12:
		return math.Float64frombits(l.u64())
	}
	l.err = fmt.Errorf("tipo de valor desconhecido %d", t)
	return nil
}

func tamanhoFixo(t uint32) int {
	switch t {
	case 0, 1, 7:
		return 1
	case 2, 3:
		return 2
	case 4, 5, 6:
		return 4
	case 10, 11, 12:
		return 8
	}
	return 0
}

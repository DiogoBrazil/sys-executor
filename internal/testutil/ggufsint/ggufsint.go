// Pacote ggufsint monta cabeçalhos GGUF sintéticos para testes (sem pesos reais).
package ggufsint

import (
	"bytes"
	"encoding/binary"
)

// Montar monta um cabeçalho GGUF v3 mínimo (sem tensores reais).
func Montar(arq string, chat bool, pooling uint32, extra int) []byte {
	var b bytes.Buffer
	w := func(v any) { binary.Write(&b, binary.LittleEndian, v) }
	str := func(s string) { w(uint64(len(s))); b.WriteString(s) }
	kvs := 0 // os pares são escritos primeiro; o cabeçalho, depois
	kvStr := func(k, v string) { str(k); w(uint32(8)); str(v); kvs++ }
	kvU32 := func(k string, v uint32) { str(k); w(uint32(4)); w(v); kvs++ }
	kvStr("general.architecture", arq)
	kvStr("general.name", "Sintético "+arq)
	kvStr("general.license", "apache-2.0")
	kvU32("general.file_type", 15)
	kvU32(arq+".block_count", 8)
	kvU32(arq+".context_length", 32768)
	kvU32(arq+".embedding_length", 512)
	kvU32(arq+".attention.head_count", 8)
	kvU32(arq+".attention.head_count_kv", 2)
	if pooling != 0 {
		kvU32(arq+".pooling_type", pooling)
	}
	// Array de strings (como o vocabulário) e array numérico: devem ser pulados.
	str("tokenizer.ggml.tokens")
	w(uint32(9))
	w(uint32(8))
	w(uint64(3))
	str("a")
	str("bb")
	str("ccc")
	kvs++
	str("tokenizer.ggml.scores")
	w(uint32(9))
	w(uint32(6))
	w(uint64(4))
	for i := 0; i < 4; i++ {
		w(float32(i))
	}
	kvs++
	if chat {
		kvStr("tokenizer.chat_template", "{{ messages }}")
	}
	pares := b.Bytes()
	b = bytes.Buffer{}
	w(uint32(0x46554747))
	w(uint32(3))
	w(uint64(1))
	w(uint64(kvs))
	b.Write(pares)
	b.Write(bytes.Repeat([]byte{0}, extra)) // "pesos"
	return b.Bytes()
}

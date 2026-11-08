package session

import "sync"

// Segredo guarda um dado sigiloso (senha do SEI, OTP) em []byte, nunca em
// string, para que possa ser zerado após o uso (V4 §33). A limpeza de memória
// gerenciada é melhor esforço: o Go pode ter copiado os bytes antes (ex.: na
// desserialização vinda da interface); por isso o segredo vive o mínimo possível.
type Segredo struct {
	mu sync.Mutex
	b  []byte
}

// NovoSegredo assume a posse de b (o chamador não deve reutilizá-lo).
func NovoSegredo(b []byte) *Segredo { return &Segredo{b: b} }

// Usar empresta os bytes a f. f não pode guardar a fatia.
func (s *Segredo) Usar(f func([]byte) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return f(s.b)
}

// Vazio informa se o segredo já foi zerado ou nunca teve conteúdo.
func (s *Segredo) Vazio() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.b) == 0
}

// Zerar sobrescreve e descarta os bytes. Idempotente.
func (s *Segredo) Zerar() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.b {
		s.b[i] = 0
	}
	s.b = nil
}

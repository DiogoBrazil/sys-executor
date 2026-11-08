package workflow

import (
	"context"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
	"github.com/DiogoBrazil/sys-executor/internal/session"
)

// TipoPedido identifica o que a etapa precisa que o usuário informe.
type TipoPedido string

const (
	PedidoCredenciais    TipoPedido = "CREDENCIAIS"    // usuário, senha, órgão e número do processo
	PedidoOTP            TipoPedido = "OTP"            // código de autenticação em dois fatores
	PedidoReautenticacao TipoPedido = "REAUTENTICACAO" // senha de novo (processo sigiloso, V4 §13)
)

// Pedido de dados ao usuário durante uma etapa (SEI).
type Pedido struct {
	ID   string
	Tipo TipoPedido
	// Alerta da tentativa anterior (ex.: senha não aceita), sem segredos.
	Alerta *erros.Erro
}

// PedidoVisao é o pedido exibido na interface (nunca contém valores digitados).
type PedidoVisao struct {
	ID     string     `json:"id"`
	Tipo   TipoPedido `json:"tipo"`
	Alerta *ErroVisao `json:"alerta"`
}

// RespostaPedido é o que o usuário informou. Campos comuns em Campos; o
// segredo (senha ou código) em []byte, zerado por quem o usar (V4 §33).
type RespostaPedido struct {
	Campos  map[string]string
	Segredo *session.Segredo
}

// Solicitar mostra o pedido e espera a resposta do usuário (ou o cancelamento).
func (c *Contexto) Solicitar(ctx context.Context, p Pedido) (RespostaPedido, error) {
	ch := make(chan RespostaPedido, 1)
	c.g.mu.Lock()
	if c.g.t != c.t || c.t.estado.Terminal() {
		c.g.mu.Unlock()
		return RespostaPedido{}, context.Canceled
	}
	if p.ID == "" {
		p.ID = novoID()
	}
	c.t.pedido, c.t.respostaPedido = &p, ch
	c.g.emitirLocked()
	c.g.mu.Unlock()
	defer func() {
		c.g.mu.Lock()
		if c.t.pedido != nil && c.t.pedido.ID == p.ID {
			c.t.pedido, c.t.respostaPedido = nil, nil
		}
		c.g.mu.Unlock()
	}()
	select {
	case r := <-ch:
		return r, nil
	case <-ctx.Done():
		return RespostaPedido{}, ctx.Err()
	}
}

// Avancar muda o estado exibido durante uma etapa longa (ex.: as fases do
// SEI). Só aceita transições válidas da tabela.
func (c *Contexto) Avancar(e Estado) error {
	c.g.mu.Lock()
	defer c.g.mu.Unlock()
	if c.g.t != c.t || c.t.estado.Terminal() {
		return context.Canceled
	}
	if c.t.estado == e {
		return nil
	}
	if err := c.g.mover(e); err != nil {
		return err
	}
	c.t.erro = nil
	c.g.emitirLocked()
	return nil
}

// Fornecer entrega a resposta ao pedido pendente. Se não houver pedido com
// esse ID, o segredo é zerado e o comando é rejeitado.
func (g *Gerenciador) Fornecer(pedidoID string, r RespostaPedido) (*Visao, error) {
	g.mu.Lock()
	t := g.t
	if t == nil || t.pedido == nil || t.pedido.ID != pedidoID || t.respostaPedido == nil {
		g.mu.Unlock()
		if r.Segredo != nil {
			r.Segredo.Zerar()
		}
		return nil, erros.Novo(erros.TarefaOrdemInvalida)
	}
	t.respostaPedido <- r
	t.pedido, t.respostaPedido = nil, nil
	return g.emitirUnlock()
}

package workflow

// Documentos do SEI (V4 §48.4, revisão de 10/11/2026; D-11 E2 revisado):
// o PDF de todos os documentos, gerado pelo SEI, é separado pelo carimbo em
// um documento por número SEI. A lista (título e número SEI) só existe no
// computador; o usuário escolhe quais documentos entram em cada peça do
// cadastro. Os não escolhidos não entram nos autos da tarefa, não vão à IA e
// são apagados com a sessão. Nenhuma ação nova no SEI.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/DiogoBrazil/sys-executor/internal/erros"
)

// DocumentoSEI é um documento separado do PDF do SEI (o conteúdo fica
// cifrado na sessão; a interface vê só estes metadados).
type DocumentoSEI struct {
	ID       string `json:"id"`  // DOC-nn
	SEI      string `json:"sei"` // número SEI do carimbo ("" se a página não tinha)
	Titulo   string `json:"titulo"`
	Paginas  int    `json:"paginas"`
	Sugestao string `json:"sugestao"` // tipo de peça sugerido pelo título (PecaMemorando…); o usuário confere
	Papel    string `json:"papel"`    // depoimento: papel sugerido pelo título
	Usado    string `json:"usado"`    // ID da peça em que foi anexado ("" = não escolhido)
	sha      string
}

// nomeLogico na sessão cifrada.
func (d DocumentoSEI) nomeLogico() string { return "sei-doc-" + d.ID + ".pdf" }

// DocumentoSEIEntrada é o que o pipeline entrega depois de separar o PDF.
type DocumentoSEIEntrada struct {
	SEI, Titulo, Sugestao, Papel string
	Paginas                      int
	Dados                        []byte
}

// ReceberDocumentosSEI guarda os documentos separados do PDF do SEI (estado
// SEI_BAIXANDO) e prepara o cadastro de peças. A etapa devolve depois
// RECEBENDO_UPLOAD, onde o usuário escolhe os documentos.
func (c *Contexto) ReceberDocumentosSEI(ds []DocumentoSEIEntrada) error {
	g := c.g
	g.mu.Lock()
	defer g.mu.Unlock()
	t := c.t
	if g.t != t || t.estado != SEIBaixando {
		return erros.Novo(erros.TarefaOrdemInvalida)
	}
	if len(ds) == 0 {
		return erros.Com(erros.SEIDownloadIncompleto, errors.New("nenhum documento no PDF do SEI"))
	}
	// Uma nova obtenção substitui a lista anterior (os arquivos já anexados
	// às peças são cópias próprias e ficam).
	for _, d := range t.documentosSEI {
		_ = t.sessao.ApagarLogico(d.nomeLogico())
	}
	t.documentosSEI = nil
	for i, e := range ds {
		if err := conferirPDF(e.Dados); err != nil {
			return err
		}
		h := sha256.Sum256(e.Dados)
		d := DocumentoSEI{ID: fmt.Sprintf("DOC-%02d", i+1), SEI: e.SEI, Titulo: limpar(e.Titulo, 160), Paginas: e.Paginas,
			Sugestao: e.Sugestao, Papel: e.Papel, sha: hex.EncodeToString(h[:])}
		if err := t.sessao.Gravar(d.nomeLogico(), e.Dados); err != nil {
			return err
		}
		t.documentosSEI = append(t.documentosSEI, d)
	}
	if t.cadastro == nil {
		t.cadastro = novoCadastro(t.tipo)
	}
	t.cadastro.Concluido = false
	return nil
}

// documentosVisao marca em que peça cada documento foi anexado.
func (t *tarefa) documentosVisao() []DocumentoSEI {
	if len(t.documentosSEI) == 0 {
		return nil
	}
	out := make([]DocumentoSEI, len(t.documentosSEI))
	copy(out, t.documentosSEI)
	if t.cadastro == nil {
		return out
	}
	for i := range out {
		for _, p := range t.cadastro.Pecas {
			for _, a := range p.Arquivos {
				if a.sha == out[i].sha {
					out[i].Usado = p.ID
				}
			}
		}
	}
	return out
}

// AnexarDocumentosSEI acrescenta a uma peça os documentos do SEI escolhidos,
// na ordem dada. O mesmo documento não entra em duas peças (ARQ-004). O
// título do SEI fica só no nome local do arquivo (nunca vai à IA).
func (g *Gerenciador) AnexarDocumentosSEI(pecaID string, ids []string) (*Visao, error) {
	g.mu.Lock()
	if err := g.emCadastro(); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	t := g.t
	if len(ids) == 0 {
		g.mu.Unlock()
		return nil, erros.Novo(erros.TarefaOrdemInvalida)
	}
	var as []ArquivoEntrada
	var seis []string
	for _, id := range ids {
		var d *DocumentoSEI
		for i := range t.documentosSEI {
			if t.documentosSEI[i].ID == id {
				d = &t.documentosSEI[i]
			}
		}
		if d == nil {
			g.mu.Unlock()
			return nil, erros.Novo(erros.TarefaOrdemInvalida)
		}
		b, err := t.sessao.Ler(d.nomeLogico())
		if err != nil {
			g.mu.Unlock()
			return nil, err
		}
		nome := d.Titulo
		if d.SEI != "" {
			nome = "SEI " + d.SEI + " – " + d.Titulo
		}
		as = append(as, ArquivoEntrada{Nome: nome, Dados: b, Origem: "SEI"})
		seis = append(seis, d.SEI)
	}
	contar := g.contarPaginas
	if contar == nil {
		contar = func([]byte) (int, error) { return 0, nil }
	}
	if err := t.cadastro.anexar(pecaID, as, contar, t.sessao.Gravar); err != nil {
		g.mu.Unlock()
		return nil, err
	}
	// Número SEI da peça: o do documento, quando ela tem um só e o usuário
	// ainda não informou outro.
	if p := t.cadastro.peca(pecaID); p != nil && p.SEI == "" && len(p.Arquivos) == 1 && len(seis) == 1 {
		p.SEI = seis[0]
	}
	return g.emitirUnlock()
}

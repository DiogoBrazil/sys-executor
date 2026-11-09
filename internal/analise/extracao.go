package analise

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/DiogoBrazil/sys-executor/internal/ingest/autos"
	"github.com/DiogoBrazil/sys-executor/internal/ingest/segment"
	"github.com/DiogoBrazil/sys-executor/internal/policy"
	"github.com/DiogoBrazil/sys-executor/internal/rules/catalog"
	"github.com/DiogoBrazil/sys-executor/internal/texto"
)

// Funções documentais que não trazem prova (só trâmite) não são extraídas.
var semProva = map[segment.Funcao]bool{
	segment.Capa: true, segment.Notificacao: true, segment.Ciencia: true, segment.TermoJuntada: true,
	segment.TermoRemessa: true, segment.Procuracao: true, segment.MemorandoDisciplinar: true, segment.PortariaSR: true,
}

// Termos que caracterizam mídia (mesma família de segment.DetectarMidias).
var reMidia = regexp.MustCompile(`\b(video|videos|audio|audios|gravac|gravad|imagem|imagens|foto|fotos|fotograf|camera|cameras|midia|midias|cd|dvd|pen ?drive|link|arquivo digital|whatsapp|mensagem de voz|filmag)`)

// PecaComProva diz se a peça entra na extração de pontos (etapa 5).
func PecaComProva(p segment.Peca) bool { return !semProva[p.Funcao] }

// PessoaNoAto é uma pessoa e o papel dela num ato (SR-EST-010; plano §6).
type PessoaNoAto struct {
	Nome    string `json:"nome"`
	Papel   string `json:"papel"`
	Militar string `json:"militar"` // SIM, NAO, INCERTO
	Posto   string `json:"posto"`
	Peca    string `json:"peca"`
	Ponto   Ponto  `json:"ponto"`
}

// Achado é um item fora da análise principal (mídia ausente, conduta
// diversa, fato novo, militar novo, perícia).
type Achado struct {
	Tipo      string `json:"tipo"`
	Descricao string `json:"descricao"`
	Extra     string `json:"extra"`
	Ponto     Ponto  `json:"ponto"`
}

// Tipos de achado.
const (
	AchadoMidia          = "MIDIA_AUSENTE"
	AchadoCondutaDiversa = "CONDUTA_DIVERSA" // PADS: só registro interno (I-05)
	AchadoFatoNovo       = "FATO_NOVO"       // SR: exige autorização
	AchadoMilitarNovo    = "MILITAR_NOVO"    // SR: exige autorização
	AchadoPericia        = "PERICIA"
)

// Extracao é o resultado da etapa 5 para uma peça.
type Extracao struct {
	Peca       segment.Peca  `json:"peca"`
	Pontos     []Ponto       `json:"pontos"`
	Pessoas    []PessoaNoAto `json:"pessoas"`
	Achados    []Achado      `json:"achados"`
	Descartes  int           `json:"descartes"` // itens cujo trecho não conferiu
	Pendencias []string      `json:"pendencias"`
}

// AlvoExtracao é uma imputação (PADS) ou um fato (SR) para o enum do schema.
type AlvoExtracao struct {
	ID        string `json:"id"`
	Descricao string `json:"descricao"`
}

type saidaExtracao struct {
	Pontos []struct {
		Alvo    string `json:"alvo_id"`
		Pessoa  string `json:"pessoa"`
		Quem    string `json:"quem"`
		Trecho  string `json:"trecho_literal"`
		Sentido string `json:"sentido"`
		Resumo  string `json:"resumo"`
	} `json:"pontos"`
	Pessoas []struct {
		Nome    string `json:"nome"`
		Papel   string `json:"papel"`
		Militar string `json:"militar"`
		Posto   string `json:"posto_graduacao"`
		Trecho  string `json:"trecho_literal"`
	} `json:"pessoas"`
	Midias []struct {
		Descricao string `json:"descricao"`
		Trecho    string `json:"trecho_literal"`
	} `json:"midias"`
	ForaDoObjeto []struct {
		Tipo      string `json:"tipo"`
		Descricao string `json:"descricao"`
		Trecho    string `json:"trecho_literal"`
	} `json:"fora_do_objeto"`
	Pericias []struct {
		Descricao string `json:"descricao"`
		Laudo     string `json:"numero_laudo"`
		Trecho    string `json:"trecho_literal"`
	} `json:"pericias"`
}

var (
	papeisPADS = []string{"ACUSADO", "TESTEMUNHA", "DECLARANTE", "OFENDIDO", "DEFENSOR", "ENCARREGADO", "PERITO", "OUTRO"}
	papeisSR   = []string{"SINDICADO", "TESTEMUNHA", "DECLARANTE", "OFENDIDO", "PERITO", "ENCARREGADO", "OUTRO"}
)

func schemaExtracao(fluxo string, alvos []string) map[string]any {
	pt := map[string]any{"alvo_id": enum(alvos...), "quem": str(120), "trecho_literal": str(500), "resumo": str(300),
		"sentido": enum("ACUSACAO", "DEFESA", "NEUTRO"), "pessoa": str(120)}
	pessoa := map[string]any{"nome": str(120), "posto_graduacao": str(60), "trecho_literal": str(300),
		"militar": enum("SIM", "NAO", "INCERTO")}
	fora := enum("CONDUTA_DIVERSA")
	props := map[string]any{
		"midias": lista(objeto(map[string]any{"descricao": str(200), "trecho_literal": str(300)}), 0, 10),
	}
	if fluxo == policy.FluxoSR {
		delete(pt, "sentido")
		pessoa["papel"] = enum(papeisSR...)
		fora = enum(AchadoFatoNovo, AchadoMilitarNovo)
		props["pericias"] = lista(objeto(map[string]any{"descricao": str(200), "numero_laudo": str(40), "trecho_literal": str(300)}), 0, 10)
	} else {
		delete(pt, "pessoa")
		pessoa["papel"] = enum(papeisPADS...)
	}
	props["pontos"] = lista(objeto(pt), 0, 25)
	props["pessoas"] = lista(objeto(pessoa), 0, 15)
	props["fora_do_objeto"] = lista(objeto(map[string]any{"tipo": fora, "descricao": str(300), "trecho_literal": str(400)}), 0, 10)
	return objeto(props)
}

// ExtrairPeca extrai os pontos de prova de uma peça (etapa 5), fatiando a
// peça em blocos que caibam no contexto (F4-06). secao é o objeto congelado
// (IMPUTACOES_CONGELADAS no PADS; OBJETO_SR na SR).
func ExtrairPeca(ctx context.Context, x *Executor, a *autos.Autos, p segment.Peca, secao policy.Secao, alvos []AlvoExtracao,
	flags map[string]bool, reg *Registro) (Extracao, error) {
	r := Extracao{Peca: p}
	ids := make([]string, len(alvos))
	for i, al := range alvos {
		ids[i] = al.ID
	}
	const reserva = 2200
	max := tamanhoBloco(x, reserva)
	for {
		r = Extracao{Peca: p}
		err := extrairBlocos(ctx, x, a, Blocos(a, p, max, reg), secao, ids, flags, reserva, reg, &r)
		if policy.IsFatiar(err) && max > 1200 {
			max /= 2
			continue
		}
		return r, err
	}
}

func extrairBlocos(ctx context.Context, x *Executor, a *autos.Autos, blocos []Bloco, secao policy.Secao, ids []string,
	flags map[string]bool, reserva int, reg *Registro, r *Extracao) error {
	validos := map[string]bool{}
	for _, id := range ids {
		validos[id] = true
	}
	papeis, fora := papeisPADS, []string{AchadoCondutaDiversa}
	if x.Fluxo == policy.FluxoSR {
		papeis, fora = papeisSR, []string{AchadoFatoNovo, AchadoMilitarNovo}
	}
	for _, b := range blocos {
		// Itens que conferem em qualquer tentativa são acumulados; o reenvio
		// pede só a correção dos demais.
		var parcial Extracao
		descartes := 0
		vistos := map[string]bool{}
		novo := func(tipo string, p Ponto) bool {
			k := fmt.Sprintf("%s|%d|%d|%d", tipo, p.Pagina, p.Inicio, p.Fim)
			if vistos[k] {
				return false
			}
			vistos[k] = true
			return true
		}
		err := x.Chamar(ctx, Chamada{
			Etapa: catalog.EtapaExtracao, Alvo: fmt.Sprintf("%s (pág. %d)", b.Peca.Titulo, b.PaginaInicial()), Flags: flags,
			Secoes: []policy.Secao{secao}, Evidencias: []policy.Evidencia{b.Evidencia()},
			Tarefa:     "extraia os pontos de prova da peça acima.",
			NomeSchema: "extracao", Schema: schemaExtracao(x.Fluxo, ids), Reserva: reserva,
		}, func(js []byte) error {
			var s saidaExtracao
			if err := decodificarEstrito(js, &s); err != nil {
				return err
			}
			descartes = 0
			var falhas []string
			falha := func(f string, a ...any) {
				descartes++
				if len(falhas) < 6 {
					falhas = append(falhas, fmt.Sprintf(f, a...))
				}
			}
			for k, pt := range s.Pontos {
				if !validos[pt.Alvo] {
					falha("o ponto %d cita alvo_id inexistente %q", k+1, pt.Alvo)
					continue
				}
				if x.Fluxo == policy.FluxoPADS && !em(pt.Sentido, "ACUSACAO", "DEFESA", "NEUTRO") {
					falha("o ponto %d tem sentido inválido %q", k+1, pt.Sentido)
					continue
				}
				p, ok := localizarNoBloco(a, b, pt.Trecho)
				if !ok {
					falha("o trecho_literal do ponto %d não aparece literalmente no documento", k+1)
					continue
				}
				if err := conferirTextoLivre("resumo", pt.Resumo, []string{b.Texto}); err != nil {
					falha("ponto %d: %v", k+1, err)
					continue
				}
				p.Alvo, p.Quem, p.Resumo, p.Sentido, p.Pessoa = pt.Alvo, strings.TrimSpace(pt.Quem), strings.TrimSpace(pt.Resumo), pt.Sentido, strings.TrimSpace(pt.Pessoa)
				if novo("ponto"+p.Alvo, p) {
					parcial.Pontos = append(parcial.Pontos, p)
				}
			}
			for k, ps := range s.Pessoas {
				p, ok := localizarNoBloco(a, b, ps.Trecho)
				if !ok || strings.TrimSpace(ps.Nome) == "" {
					falha("a pessoa %d não tem trecho literal no documento", k+1)
					continue
				}
				if !em(ps.Papel, papeis...) || !em(ps.Militar, "SIM", "NAO", "INCERTO", "") {
					falha("a pessoa %d tem papel ou indicação de militar inválidos", k+1)
					continue
				}
				if novo("pessoa"+texto.SemAcentos(ps.Nome), Ponto{}) {
					parcial.Pessoas = append(parcial.Pessoas, PessoaNoAto{Nome: strings.TrimSpace(ps.Nome), Papel: ps.Papel,
						Militar: ps.Militar, Posto: strings.TrimSpace(ps.Posto), Peca: b.Peca.ID, Ponto: p})
				}
			}
			for k, m := range s.Midias {
				p, ok := localizarNoBloco(a, b, m.Trecho)
				if !ok {
					falha("a mídia %d não tem trecho literal no documento", k+1)
					continue
				}
				if !reMidia.MatchString(texto.SemAcentos(p.Trecho)) {
					falha("o item %d de midias não menciona mídia (vídeo, áudio, gravação, imagem, link…); deixe a lista vazia se não houver", k+1)
					continue
				}
				if novo(AchadoMidia, p) {
					parcial.Achados = append(parcial.Achados, Achado{Tipo: AchadoMidia, Descricao: strings.TrimSpace(m.Descricao), Ponto: p})
				}
			}
			for k, f := range s.ForaDoObjeto {
				if !em(f.Tipo, fora...) {
					falha("o item fora do objeto %d tem tipo inválido %q", k+1, f.Tipo)
					continue
				}
				p, ok := localizarNoBloco(a, b, f.Trecho)
				if !ok {
					falha("o item fora do objeto %d não tem trecho literal no documento", k+1)
					continue
				}
				if novo(f.Tipo, p) {
					parcial.Achados = append(parcial.Achados, Achado{Tipo: f.Tipo, Descricao: strings.TrimSpace(f.Descricao), Ponto: p})
				}
			}
			for k, pe := range s.Pericias {
				p, ok := localizarNoBloco(a, b, pe.Trecho)
				if !ok {
					falha("a perícia %d não tem trecho literal no documento", k+1)
					continue
				}
				if novo(AchadoPericia, p) {
					parcial.Achados = append(parcial.Achados, Achado{Tipo: AchadoPericia, Descricao: strings.TrimSpace(pe.Descricao), Extra: strings.TrimSpace(pe.Laudo), Ponto: p})
				}
			}
			if len(falhas) > 0 {
				return fmt.Errorf("%s (os demais itens já foram aceitos; corrija só estes)", strings.Join(falhas, "; "))
			}
			return nil
		})
		if err != nil && !IsSemSeguranca(err) {
			return err
		}
		if err != nil {
			// Itens que conferiram ficam; os demais são descartados e a
			// limitação aparece (nunca se aceita trecho não verificado).
			r.Pendencias = append(r.Pendencias, fmt.Sprintf("Na peça %s (pág. %d), %d item(ns) da IA não conferiram com o documento e foram descartados.",
				b.Peca.Titulo, b.PaginaInicial(), descartes))
			r.Descartes += descartes
		}
		for _, p := range parcial.Pontos {
			p.ID = reg.Novo()
			r.Pontos = append(r.Pontos, p)
		}
		r.Pessoas = append(r.Pessoas, parcial.Pessoas...)
		r.Achados = append(r.Achados, parcial.Achados...)
	}
	r.Pontos = dedupPontos(r.Pontos)
	return nil
}

func dedupPontos(ps []Ponto) []Ponto {
	vistos := map[string]bool{}
	var out []Ponto
	for _, p := range ps {
		k := fmt.Sprintf("%s|%d|%d|%d", p.Alvo, p.Pagina, p.Inicio, p.Fim)
		if vistos[k] {
			continue
		}
		vistos[k] = true
		out = append(out, p)
	}
	return out
}

// MesmaPessoa compara nomes normalizados (sem acento e caixa).
func MesmaPessoa(a, b string) bool {
	na, nb := strings.Join(strings.Fields(texto.SemAcentos(a)), " "), strings.Join(strings.Fields(texto.SemAcentos(b)), " ")
	return na != "" && (na == nb || strings.Contains(na, nb) || strings.Contains(nb, na))
}

func em(v string, vs ...string) bool {
	for _, x := range vs {
		if v == x {
			return true
		}
	}
	return false
}

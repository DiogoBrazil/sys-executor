// Pacote pseudonimo troca dados pessoais por marcadores antes do envio à IA
// e restaura a grafia exata na resposta (V4 §47.3, D-10, TRV-SEG-011).
//
// É MELHOR ESFORÇO, não anonimização: a detecção é local e determinística
// (sem modelo de linguagem) e se apoia em âncoras — posto/graduação, papel
// processual, rótulos ("CPF", "RG nº", "matrícula", "residente na"…) e
// formatos com dígito verificador. Nomes sem âncora, grafias quebradas por
// OCR ou hifenização podem passar, e o contexto pode permitir reidentificação.
//
// Cada grafia distinta ganha o próprio marcador; grafias que só diferem em
// caixa ou acentos são a mesma pessoa e compartilham o número
// ([PESSOA-03], [PESSOA-03.2]). Assim a restauração devolve exatamente o
// texto original, e a conferência literal dos trechos citados pela IA
// continua funcionando sem mudança.
package pseudonimo

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Tipo de dado pessoal.
type Tipo string

const (
	Pessoa     Tipo = "PESSOA"
	CPF        Tipo = "CPF"
	RG         Tipo = "RG"
	Matricula  Tipo = "MATRICULA"
	Telefone   Tipo = "TELEFONE"
	Email      Tipo = "EMAIL"
	CEP        Tipo = "CEP"
	Endereco   Tipo = "ENDERECO"
	Nascimento Tipo = "NASCIMENTO"
	OAB        Tipo = "OAB"
)

var tipos = []Tipo{Pessoa, CPF, RG, Matricula, Telefone, Email, CEP, Endereco, Nascimento, OAB}

// achado é uma ocorrência detectada: a grafia exata e o tipo.
type achado struct {
	tipo   Tipo
	forma  string
	inicio int // posição em bytes no texto (para testes e ordem)
}

const (
	maiuscula = `A-ZÀ-ÖØ-Þ`
	minuscula = `a-zà-öø-ÿ`
	// Uma palavra de nome: "Silva", "SILVA", "D'Ávila" não é coberto.
	palavra  = `(?:[` + maiuscula + `][` + minuscula + `]+|[` + maiuscula + `]{2,})`
	conector = `(?:da|de|do|das|dos|e|DA|DE|DO|DAS|DOS|E)`
	// Até 7 palavras, com conectores; espaços podem incluir uma quebra de linha.
	nome = palavra + `(?:[ \t]{1,3}\n?[ \t]*(?:` + conector + `[ \t]{1,3}\n?[ \t]*)?` + palavra + `){0,6}`
	// Posto/graduação da PM/BM (âncora de nome). Alternativas mais longas primeiro.
	posto = `(?:Ten(?:ente)?[ -]?Cel|TEN[ -]?CEL|Tenente[- ]Coronel|TENENTE[- ]CORONEL|TC|Cel|CEL|Coronel|CORONEL|Maj|MAJ|Major|MAJOR|` +
		`Cap|CAP|Capit[ãa]o|CAPIT[ÃA]O|[12][º°oª]?[ \t]?Ten|[12][º°oª]?[ \t]?TEN|Ten|TEN|Tenente|TENENTE|Asp(?:[ \t]?Of)?|ASP(?:[ \t]?OF)?|Aspirante|ASPIRANTE|` +
		`Subten|SUBTEN|Subtenente|SUBTENENTE|ST|[123][º°oª]?[ \t]?Sgt|[123][º°oª]?[ \t]?SGT|Sgt|SGT|Sargento|SARGENTO|Cb|CB|Cabo|CABO|Sd|SD|Soldado|SOLDADO)`
	quadro = `(?:(?:QOPM|QPPM|QOBM|QPBM|QOAPM|QOEPM|QPEPM)[ \t]+)?(?:PM|BM|PMRO|BMRO|RR)?\.?`
	// Rótulo de parente na qualificação do termo.
	parente = `(?i:pai|m[ãa]e|genitor(?:a|es)?|filia[çc][ãa]o|c[ôo]njuge|companheir[oa]|espos[oa]|convivente)`
	// Matrícula opcional entre o posto e o nome ("Sd PM 100012345 FULANO").
	matEntre = `(?:(?:RE|Mat\.?|MAT\.?|[Mm]atr[íi]cula)?[ \t]*(?:n[º°o.]*[ \t]*)?(\d[\d.\-/]{3,}\d)[ \t]+)?`
)

var (
	rePosto = regexp.MustCompile(`\b` + posto + `\.?[ \t]+` + quadro + `[ \t]*` + matEntre + `(` + nome + `)`)

	rePapel = regexp.MustCompile(`(?i:\b(?:testemunhas?|declarantes?|depoentes?|v[íi]timas?|acusad[oa]s?|sindicad[oa]s?|investigad[oa]s?|` +
		`ofendid[oa]s?|comunicantes?|denunciantes?|noticiantes?|advogad[oa]s?|defensor(?:a|es)?|procurador(?:a|es)?|` +
		`senhor(?:a)?|sr\.?|sra\.?|srta\.?|dr\.?|dra\.?|doutor(?:a)?|cidad[ãa]os?|cidad[ãa]s?|requerente|interessad[oa]|` +
		`representante|motorista|condutor(?:a)?|propriet[áa]ri[oa]|nome(?:[ \t]+completo)?[ \t]*:))` +
		`[ \t]*[:,]?[ \t]*(?:(?:o|a)[ \t]+)?(?:(?i:dr|dra|sr|sra|srta)\.?[ \t]+)?(` + nome + `)`)

	reFiliacao = regexp.MustCompile(`(?i:\bfilh[oa][ \t]+de)[ \t]+(` + nome + `)(?:[ \t]+(?i:e)[ \t]+(?:(?i:de)[ \t]+)?(` + nome + `))?`)
	// Parentes da qualificação (pai, mãe, cônjuge…). O OCR troca o ":" por
	// ".", ";" ou "-", ou o perde. No início da linha, aceita qualquer um
	// deles ou só o espaço; no meio da linha, exige ":", ";" ou hífen, para
	// que "o pai Fulano" de um relato não vire âncora. O nome continua
	// exigindo inicial maiúscula ("Mãe: não declarada" não casa).
	reParenteLinha = regexp.MustCompile(`(?m)^[ \t]*` + parente + `[ \t]*(?:[:.;\-–][ \t]*)?(` + nome + `)`)
	reParente      = regexp.MustCompile(`\b` + parente + `[ \t]*[:;\-–][ \t]*(` + nome + `)`)
	reConvivio     = regexp.MustCompile(`(?i:\bcasad[oa][ \t]+com|\buni[ãa]o[ \t]+est[áa]vel[ \t]+com|\bconvive[ \t]+com|\bvive[ \t]+maritalmente[ \t]+com)` +
		`[ \t]+(?:(?i:o|a)[ \t]+)?(?:(?i:sr|sra)\.?[ \t]+)?(` + nome + `)`)

	// Bloco de assinatura: linha só com nome em maiúsculas seguida de linha
	// com posto, cargo ou identificação.
	reAssinatura = regexp.MustCompile(`(?m)^[ \t]*([` + maiuscula + `]{2,}(?:[ \t]+(?:DA|DE|DO|DAS|DOS|E|[` + maiuscula + `]{2,})){1,6})[ \t]*\r?\n[ \t]*` +
		`(?:` + posto + `\b|(?i:encarregad[oa]|sindicante|comandante|corregedor(?:a)?\b|escriv[ãa]o|testemunha|declarante|acusad[oa]|` +
		`sindicad[oa]|advogad[oa]|defensor(?:a)?|presidente|membro|mat(?:r[íi]cula)?\.?|RE\b|CPF|OAB))`)

	reCPF       = regexp.MustCompile(`\b\d{3}\.?\d{3}\.?\d{3}[-.]?\d{2}\b`)
	reCPFRotulo = regexp.MustCompile(`(?i:\bCPF)[ \t]*(?:n[º°o.]*[ \t]*)?[:\-]?[ \t]*(\d[\d.\-]{9,13}\d)`)
	reRG        = regexp.MustCompile(`(?i:\b(?:RG|R\.G\.|registro[ \t]+geral|carteira[ \t]+de[ \t]+identidade|c[ée]dula[ \t]+de[ \t]+identidade))[ \t]*(?:n[º°o.]*[ \t]*)?[:\-]?[ \t]*(\d[\d.\-]{3,14}[\dXx])`)
	reMat       = regexp.MustCompile(`(?:\b(?i:matr[íi]cula|mat\.)|\bRE\b)[ \t]*(?:n[º°o.]*[ \t]*)?[:\-]?[ \t]*(\d[\d.\-/]{3,14}\d)`)
	reOAB       = regexp.MustCompile(`\bOAB[ \t]*/?[ \t]*[A-Z]{2}[ \t]*(?:n[º°o.]*[ \t]*)?[:\-]?[ \t]*(\d[\d.]{2,8}\d?[A-Z]?)`)
	reTelefone  = regexp.MustCompile(`\(\d{2}\)[ \t]?9?\d{4}-?\d{4}\b`)
	reTelRotulo = regexp.MustCompile(`(?i:\b(?:telefone|celular|tel\.|fone|whats\s?app|contato))[ \t]*(?:n[º°o.]*[ \t]*)?[:\-]?[ \t]*(\(?\d{2}\)?[ \t]?9?\d{4}[ \t-]?\d{4})`)
	reEmail     = regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`)
	reCEP       = regexp.MustCompile(`(?:(?i:\bCEP)[ \t]*(?:n[º°o.]*[ \t]*)?[:\-]?[ \t]*(\d{2}\.?\d{3}-?\d{3})|\b(\d{5}-\d{3})\b)`)
	reNasc      = regexp.MustCompile(`(?i:\b(?:nascid[oa][ \t]+em|data[ \t]+de[ \t]+nascimento|nasc\.|d\.?[ \t]?n\.))[ \t]*[:\-]?[ \t]*(\d{1,2}[/.\-]\d{1,2}[/.\-]\d{2,4})`)
	reEndereco  = regexp.MustCompile(`(?i:\b(?:residente(?:[ \t]+e[ \t]+domiciliad[oa])?|domiciliad[oa]|endere[çc]o(?:[ \t]+residencial)?[ \t]*:?|reside))` +
		`[ \t]+(?i:(?:na|no|à|a|em)[ \t]+)?((?i:rua|r\.|avenida|av\.|travessa|tv\.|alameda|rodovia|estrada|linha|ramal|beco|pra[çc]a|quadra|conjunto|condom[íi]nio|setor|bairro)\b[^\n;]{2,100}?)` +
		`(?:[.;\n]|,[ \t]*(?i:cep|telefone|tel\.|celular|e-?mail|portador|nesta|neste|nascid))`)
)

// paradas: palavras capitalizadas que não são nomes de pessoas (instituição,
// cargo, documento, lugar comum). O nome é cortado na primeira delas.
var paradas = map[string]bool{}

func init() {
	for _, p := range strings.Fields(`polícia policia militar militares estado rondônia rondonia governo secretaria batalhão batalhao bpm
		companhia cia pelotão pelotao comando comandante subcomandante corregedor corregedora corregedoria chefe diretor diretoria
		encarregado encarregada sindicante escrivão escrivao presidente membro auditor auditoria delegado delegada juiz juíza promotor
		promotora defensor defensora advogado advogada secretário secretario governador senhor senhora excelentíssimo excelentissimo
		ilustríssimo ilustrissimo memorando portaria processo termo declarações declaracoes depoimento ofício oficio despacho relatório
		relatorio sindicância sindicancia regular disciplinar apuratório apuratorio sumário sumario regulamento lei decreto resolução
		resolucao diretriz art artigo inciso parágrafo paragrafo alínea alinea que aos às as os no na nos nas em ao o a um uma ele ela
		eles elas sim não nao porto velho brasil república republica federativa geral central sei pm bm pmro rdpm cpm cpp cp cf ccb
		testemunha declarante depoente vítima vitima acusado acusada sindicado sindicada investigado investigada comunicante
		interessado interessada requerente motorista condutor proprietário proprietario viatura vtr operação operacao ocorrência
		ocorrencia boletim bo unidade opm cpo cpi grupamento esquadrão esquadrao rua avenida travessa bairro centro cidade município
		municipio janeiro fevereiro março marco abril maio junho julho agosto setembro outubro novembro dezembro segunda terça terca
		quarta quinta sexta sábado sabado domingo dia data hora horas ano anos documento documentos anexo anexos assinatura
		assinado assinada eletronicamente conforme referente nome completo matrícula matricula cpf rg oab qopm qppm qobm qpbm cel
		ten cap maj sgt cb sd asp subten st tc coronel tenente capitão capitao major sargento cabo soldado aspirante subtenente
		ministério ministerio público publico justiça justica tribunal vara comarca fórum forum cartório cartorio segurança seguranca
		pública trânsito transito detran ciopor sesdec sepog jus gov dr dra sr sra srta exmo exma ilmo ilma
		defesa prévia previa alegações alegacoes finais inquirição inquiricao interrogatório interrogatorio
		qualificação qualificacao laudo certidão certidao notificação notificacao citação citacao intimação
		intimacao manifestação manifestacao parecer solução solucao mandado exame perícia pericia pericial ata
		audiência audiencia comunicação comunicacao informação informacao encaminhamento juntada conclusão
		conclusao decisão decisao julgamento homologação homologacao imputação imputacao imputações imputacoes
		fato fatos objeto escala serviço servico instauração instauracao abertura vista prazo
		vossa vosso senhoria excelência excelencia magnificência magnificencia fica ficam
		estatuto policiais policial decreto-lei código codigo constituição constituicao penal civil federal
		estadual complementar emenda normativa instrução instrucao portarias regimento interno
		transgressão transgressao transgressões transgressoes disciplina infração infracao infrações infracoes autos
		escolaridade ensino fundamental médio medio superior incompleto completa incompleta analfabeto analfabeta
		profissão profissao naturalidade nacionalidade religião religiao sexo cor raça raca
		união uniao estável estavel solteiro solteira casado casada divorciado divorciada viúvo viuvo viúva viuva
		separado separada filiação filiacao pai mãe mae genitor genitora genitores cônjuge conjuge companheiro companheira
		esposa esposo convivente desconhecido desconhecida ignorado ignorada falecido falecida
		igreja cristã crista evangélica evangelica católica catolica assembleia templo paróquia paroquia
		escola colégio colegio municipal hospital delegacia prefeitura associação associacao sindicato
		fazenda sítio sitio chácara chacara linha ramal rodovia distrito placa veículo veiculo motocicleta moto carro
		caminhonete automóvel automovel honda yamaha suzuki fiat volkswagen chevrolet toyota hyundai renault nissan
		mitsubishi peugeot citroën citroen jeep kawasaki dafra shineray haojue hilux strada palio titan biz bros
		unisp ciop patrulha patrulhamento radiopatrulha tático tatico tática tatica força forca rotam bope choque canil
		ronda ostensivo ostensiva guarnição guarnicao equipe`) {
		paradas[p] = true
	}
}

var reRomano = regexp.MustCompile(`^[IVXLCDM]+$`)

// aparar corta o nome capturado na primeira palavra que não é nome e remove
// conectores soltos no fim. Devolve "" se não sobrar nome.
var rePalavra = regexp.MustCompile(`\S+`)

func aparar(s string) string {
	palavras := 0
	ultimoNome := 0
	for _, ix := range rePalavra.FindAllStringIndex(s, -1) {
		w := s[ix[0]:ix[1]]
		if eConector(w) {
			continue
		}
		lw := strings.ToLower(w)
		if paradas[lw] || paradas[semAcento(lw)] || reRomano.MatchString(w) {
			break
		}
		palavras++
		ultimoNome = ix[1]
	}
	if palavras == 0 {
		return ""
	}
	return s[:ultimoNome]
}

func eConector(w string) bool {
	switch w {
	case "da", "de", "do", "das", "dos", "e", "DA", "DE", "DO", "DAS", "DOS", "E":
		return true
	}
	return false
}

// semAcento normaliza para comparação (caixa e acentos).
func semAcento(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	r, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	return r
}

// chavePessoa: grafias que só diferem em caixa, acentos e espaços são a
// mesma pessoa.
func chavePessoa(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(semAcento(s))), " ")
}

// cpfValido confere os dígitos verificadores.
func cpfValido(s string) bool {
	var d []int
	for _, r := range s {
		if r >= '0' && r <= '9' {
			d = append(d, int(r-'0'))
		}
	}
	if len(d) != 11 {
		return false
	}
	iguais := true
	for _, x := range d[1:] {
		if x != d[0] {
			iguais = false
		}
	}
	if iguais {
		return false
	}
	for k := 9; k <= 10; k++ {
		soma := 0
		for i := 0; i < k; i++ {
			soma += d[i] * (k + 1 - i)
		}
		dv := (soma * 10) % 11
		if dv == 10 {
			dv = 0
		}
		if dv != d[k] {
			return false
		}
	}
	return true
}

// detectar devolve os achados de um texto.
func detectar(t string) []achado {
	var out []achado
	add := func(tp Tipo, s string, ini int) {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, achado{tipo: tp, forma: s, inicio: ini})
		}
	}
	grupo := func(re *regexp.Regexp, tp Tipo, g int) {
		for _, m := range re.FindAllStringSubmatchIndex(t, -1) {
			if m[2*g] >= 0 {
				add(tp, t[m[2*g]:m[2*g+1]], m[2*g])
			}
		}
	}
	nomes := func(re *regexp.Regexp, gs ...int) {
		for _, m := range re.FindAllStringSubmatchIndex(t, -1) {
			for _, g := range gs {
				if m[2*g] < 0 {
					continue
				}
				n := aparar(t[m[2*g]:m[2*g+1]])
				if n == "" {
					continue
				}
				// "Fulano Silva e Beltrano Costa": duas pessoas, dois marcadores.
				for _, parte := range dividirE(n) {
					add(Pessoa, parte, m[2*g]+strings.Index(n, parte))
				}
			}
		}
	}
	// Identificadores primeiro (o nome pode vir logo depois do número).
	for _, m := range reCPF.FindAllStringIndex(t, -1) {
		if s := t[m[0]:m[1]]; cpfValido(s) {
			add(CPF, s, m[0])
		}
	}
	grupo(reCPFRotulo, CPF, 1)
	grupo(reRG, RG, 1)
	grupo(reMat, Matricula, 1)
	grupo(reOAB, OAB, 1)
	for _, m := range reTelefone.FindAllStringIndex(t, -1) {
		add(Telefone, t[m[0]:m[1]], m[0])
	}
	grupo(reTelRotulo, Telefone, 1)
	for _, m := range reEmail.FindAllStringIndex(t, -1) {
		add(Email, t[m[0]:m[1]], m[0])
	}
	grupo(reCEP, CEP, 1)
	grupo(reCEP, CEP, 2)
	grupo(reNasc, Nascimento, 1)
	grupo(reEndereco, Endereco, 1)
	// Matrícula entre posto e nome.
	grupo(rePosto, Matricula, 1)
	nomes(rePosto, 2)
	nomes(rePapel, 1)
	nomes(reFiliacao, 1, 2)
	nomes(reParenteLinha, 1)
	nomes(reParente, 1)
	nomes(reConvivio, 1)
	nomes(reAssinatura, 1)
	return out
}

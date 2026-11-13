package pseudonimo

import (
	_ "embed"
	"strings"
	"sync"
	"unicode"
)

// Vocabulário de palavras comuns (pedido do usuário, 12/11/2026): reduz os
// falsos "possíveis nomes" da barreira ("Auto de Prisão", "Local de
// Trabalho", "Orientação Sexual"). Uma sequência só deixa de ser suspeita se
// TODAS as palavras dela forem comuns; uma palavra fora do vocabulário
// ("Rudinei", "Linhares") basta para continuar perguntando. Gerado do acervo
// da biblioteca por cmd/vocabgen.

//go:embed dados/vocabulario.txt
var vocabularioTxt string

var (
	vocabOnce sync.Once
	vocab     map[string]bool
)

func vocabulario() map[string]bool {
	vocabOnce.Do(func() {
		vocab = map[string]bool{}
		for _, l := range strings.Split(vocabularioTxt, "\n") {
			if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
				vocab[l] = true
			}
		}
	})
	return vocab
}

// ChaveVocabulario normaliza a palavra (minúsculas, sem acento).
func ChaveVocabulario(w string) string { return strings.ToLower(semAcento(w)) }

// sobrenomesComuns: sobrenomes frequentes que também são palavras da língua.
// Nunca entram no vocabulário, para que "José Pinto Rocha" continue suspeito.
var sobrenomesComuns = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`abreu aguiar alegria almeida amaral amor andrade aquino araujo arruda assis aurora azevedo
		bahia bandeira barbosa barreto barros bastos batista bento bezerra bispo bonfim borges braga branco brasil brito bueno
		cabral caldas calixto camargo campelo campos cardoso carneiro carvalho castro cavalcante cerqueira chagas coelho conceicao
		cordeiro correia costa couto cruz cunha dantas delgado diniz domingos dourado duarte esperanca estrela fagundes faria farias
		feitosa felix fernandes ferraz ferreira figueiredo flores fonseca fontes fortes franca franco freire freitas frota galvao
		gama garcia gaspar gloria godoy goes gomes goncalves gouveia graca guedes guerra guimaraes jardim jesus junior justo lacerda
		leal leite lemos lima linhares lobo lopes louro luz macedo machado magalhaes maia malta marinho marques martins mata matos
		medeiros meireles melo mendes menezes mesquita miranda monte monteiro morais moreira mota moura muniz nascimento negreiros
		neto neves nobre nogueira novais nunes oliveira pacheco paiva palmeira paz pedrosa peixoto pereira pimenta pinheiro pinto
		pires porto prado prates queiroz ramos rangel rebelo rego reis ribeiro rocha rodrigues rosa rosario sales salgado sampaio
		santana santiago santos sardinha serra silva silveira simoes soares sobral sousa souza tavares teixeira teles torres toledo
		valadares vale valente vasconcelos veloso viana vidal vieira vilela xavier
		clara vitoria celeste serena socorro dores fatima aparecida lourdes penha gracas mercedes assuncao piedade
		natividade anjos prazeres remedios carmo luzia trindade sacramento ascensao encarnacao caridade evangelista
		bondade felicidade constancia florencia`) {
		sobrenomesComuns[w] = true
	}
}

// palavrasExtras: marcas, aplicativos e termos comuns nos autos que não
// estão nas leis (não são dados pessoais).
var palavrasExtras = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`google gmail meet whatsapp instagram facebook telegram youtube outlook hotmail yahoo
		tiktok twitter zoom teams microsoft windows android iphone apple samsung motorola xiaomi lg positivo nokia
		ford fiat chevrolet volkswagen honda toyota hyundai renault nissan yamaha suzuki jeep mitsubishi peugeot
		citroen kia bmw mercedes audi volvo dafra shineray kawasaki
		gol palio uno onix hilux corolla civic strada saveiro fiesta sandero logan duster renegade compass toro
		spin cobalt prisma celta corsa kwid mobi argo cronos polo virtus voyage fox titan fan biz bros xre factor
		fazer nmax ranger amarok frontier tracker creta kicks versa sentra march etios yaris city fit ecosport
		montana hrv wrv cruze s10 hb20 flex sedan hatch pickup caminhonete motocicleta moto
		pistola revolver espingarda carabina fuzil taurus glock imbel rossi cbc municao carregador
		crack cocaina maconha skunk ecstasy entorpecente
		adendo anexo nucleo promotoria camera embalagem jardim condominio residencial bairro setor quadra lote
		rua avenida rodovia estrada distrito linha ramal travessa vila conjunto loteamento
		ocorrencia boletim relatorio oficio despacho certidao memorando portaria termo auto laudo parecer
		informacao requerimento notificacao intimacao mandado alvara decisao sentenca acordao
		pdf sei seip seirp cpf rg cnh cnpj renavam placa chassi`) {
		palavrasExtras[w] = true
	}
}

// SobrenomeComum informa se a palavra (já normalizada) é sobrenome comum.
func SobrenomeComum(w string) bool { return sobrenomesComuns[w] }

// minusculasDoTexto: palavras (3 letras ou mais) que aparecem escritas em
// minúsculas no próprio texto conferido; ali elas são palavras comuns. Só
// contam palavras soltas, feitas só de letras: e-mails, endereços web e
// identificadores ("fulano.silva@…", "fulano_silva") trazem nomes em
// minúsculas e são ignorados por inteiro.
func minusculasDoTexto(t string) map[string]bool {
	out := map[string]bool{}
	for _, f := range strings.Fields(t) {
		w := strings.TrimFunc(f, func(r rune) bool { return strings.ContainsRune(`.,;:!?()[]{}"'“”‘’«»`, r) })
		if len([]rune(w)) < 3 || strings.ToLower(w) != w {
			continue
		}
		soLetras := true
		for _, r := range w {
			soLetras = soLetras && unicode.IsLetter(r)
		}
		if soLetras {
			out[ChaveVocabulario(w)] = true
		}
	}
	return out
}

// sufixosComuns: terminações típicas de substantivo ou adjetivo comum
// ("Investigação", "Liberdade", "Procedimento", "Ocorrência", "Embalagem",
// "Inaceitável"). Só em palavras de 6 letras ou mais; os nomes e sobrenomes
// com essas terminações ("Conceição", "Trindade", "Sacramento") estão em
// sobrenomesComuns e continuam suspeitos.
var sufixosComuns = []string{"cao", "coes", "dade", "dades", "mento", "mentos", "encia", "encias", "ancia", "ancias",
	"agem", "agens", "ismo", "ismos", "avel", "aveis", "ivel", "iveis", "orio", "orios", "oria", "orias"}

func sufixoComum(k string) bool {
	if len([]rune(k)) < 6 {
		return false
	}
	for _, s := range sufixosComuns {
		if strings.HasSuffix(k, s) {
			return true
		}
	}
	return false
}

// soPalavrasComuns: todas as palavras da sequência (fora os conectores e as
// de uma ou duas letras, como "Ka" ou "HB") são palavras comuns (do
// vocabulário, da lista extra, de terminação típica ou escritas em
// minúsculas no próprio texto), e nenhuma é sobrenome comum.
func soPalavrasComuns(seg string, doTexto map[string]bool) bool {
	v := vocabulario()
	n := 0
	for _, w := range strings.Fields(seg) {
		k := ChaveVocabulario(w)
		switch k {
		case "da", "de", "do", "das", "dos", "e":
			continue
		}
		if len([]rune(k)) <= 2 {
			continue // "Ka", "HB", "S1": não é nome de pessoa sozinho
		}
		if !(v[k] || palavrasExtras[k] || doTexto[k] || sufixoComum(k)) || sobrenomesComuns[k] {
			return false
		}
		n++
	}
	return n > 0
}

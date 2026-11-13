// Cenários da galeria: a forma das visões do Go, com nomes fictícios.
const ETAPAS_PADS = ["Obtendo os autos", "Extraindo texto e OCR", "Classificando peças", "Delimitando o objeto", "Analisando provas",
  "Consultando fontes jurídicas", "Calculando comportamento e sanção", "Elaborando a minuta", "Validando a minuta", "Aguardando sua revisão"];

function etapas(ativa: number, nomes = ETAPAS_PADS) {
  return nomes.map((rotulo, i) => ({ chave: "e" + i, rotulo, situacao: i < ativa ? "concluida" : i === ativa ? "ativa" : "pendente" }));
}
function visao(estado: string, extra: Record<string, unknown> = {}) {
  return { id: "t1", tipo: "PADS", entrada: "UPLOAD", estado, descricao: "Analisando as provas", terminal: false, etapas: etapas(4),
    progresso: null, erro: null, aviso: null, pergunta: null, pedido: null, autos: null, parada: false, retomavel: false, atividade: "",
    painel: null, respostas: [], sequencia: 1, cadastro: null, pendenciasCadastro: [], documentosSei: [], ...extra };
}
const peca = (id: string, tipo: string, extra: Record<string, unknown> = {}) =>
  ({ id, tipo, titulo: "", descricao: "", papel: "", pessoaIds: [], data: "", sei: "", naoApresentada: false, arquivos: [], ...extra });
const arq = (id: string, nome: string, paginas: number) => ({ id, nome, paginas, hash: "a1b2c3", origem: "UPLOAD" });

const cadPADS = {
  tipo: "PADS", acusadoId: "P1", portaria: "", sindicados: [], ofendidos: [], semOfendido: false, concluido: false,
  pessoas: [{ id: "P1", nome: "CARLOS INVENTADO DA SILVA", postoGrad: "CB QPPM", grafias: [], crianca: false },
    { id: "P2", nome: "JOEL FICTÍCIO", postoGrad: "SD PM", grafias: [], crianca: false }],
  fatos: [{ id: "FATO-01", resumo: "", data: "06/07/2025", enquadramentos: [{ dispositivoId: "RDPM-17-XI", natureza: "" }] }],
  pecas: [peca("PEC-01", "MEMORANDO", { arquivos: [arq("A1", "memorando.pdf", 2)] }), peca("PEC-02", "DEFESA_PREVIA", { arquivos: [arq("A2", "defesa-previa.pdf", 2)] }),
    peca("PEC-03", "ALEGACOES_FINAIS"), peca("PEC-04", "DEPOIMENTO", { papel: "TESTEMUNHA", pessoaIds: ["P2"], data: "04/09/2026", arquivos: [arq("A3", "termo-inquiricao.pdf", 2)] })],
};
const cadSR = {
  tipo: "SR", acusadoId: "", portaria: "77/2026, Corregedor-Geral, 05/06/2026", sindicados: ["S1", "S2"], ofendidos: ["O1"], semOfendido: false, concluido: false,
  pessoas: [{ id: "S1", nome: "HÉLIO FICTÍCIO", postoGrad: "3º SGT PM", grafias: [], crianca: false }, { id: "S2", nome: "ÍCARO SUPOSTO", postoGrad: "CB PM", grafias: [], crianca: false },
    { id: "O1", nome: "Administração Pública", postoGrad: "", grafias: [], crianca: false }],
  fatos: [], pecas: [peca("PEC-01", "PORTARIA", { arquivos: [arq("A1", "portaria.pdf", 1)] })],
};

const resumoAutos = {
  totalPaginas: 13, compatibilidade: "COMPATIVEL", pecaCentral: "PEC-01",
  paginas: Array.from({ length: 13 }, (_, i) => ({ numero: i + 1, situacao: i === 4 || i === 5 ? "OCR_BAIXO" : "LIDA", origem: i === 4 || i === 5 ? "OCR" : "NATIVO", confianca: 80, peca: "PEC-01" })),
  pecas: [
    { id: "PEC-01", titulo: "MEMORANDO DISCIPLINAR", funcao: "MEMORANDO_DISCIPLINAR", rotulo: "Memorando Disciplinar", paginaInicial: 1, paginaFinal: 2, sei: "0073088120", provavel: false, externo: false, situacao: "LIDA" },
    { id: "PEC-02", titulo: "DEFESA PRÉVIA", funcao: "DEFESA_PREVIA", rotulo: "Defesa prévia", paginaInicial: 3, paginaFinal: 4, sei: "0073240862", provavel: false, externo: false, situacao: "LIDA" },
    { id: "PEC-03", titulo: "TERMO DE INQUIRIÇÃO", funcao: "TERMO_INQUIRICAO", rotulo: "Termo de inquirição", paginaInicial: 5, paginaFinal: 6, sei: "0076305748", provavel: false, externo: false, situacao: "OCR_BAIXO" },
    { id: "PEC-04", titulo: "ALEGAÇÕES FINAIS", funcao: "ALEGACOES_FINAIS", rotulo: "Alegações finais", paginaInicial: 11, paginaFinal: 13, sei: "0077208327", provavel: false, externo: false, situacao: "LIDA" },
  ],
  midias: [{ pagina: 5, peca: "PEC-03", termo: "vídeos", inicio: 0, fim: 5 }],
};
const painel = { secoes: [
  { titulo: "Objeto do procedimento", itens: [{ titulo: "Acusado", situacao: "confirmado", linhas: ["CB QPPM CARLOS INVENTADO DA SILVA"], fontes: [] },
    { titulo: "IMP-1", situacao: "congelada", linhas: ["empregar força ou medida desnecessária em ato de serviço, ainda que não resulte dano.", "art. 17, inciso XI, do RDPM — natureza grave — fato em 06/07/2025"], fontes: [] }] },
  { titulo: "Leitura das peças", itens: [{ titulo: "Termo de inquirição — JOEL FICTÍCIO", situacao: "lida", linhas: ["Relata que viu o acusado conter a pessoa abordada."], fontes: [{ rotulo: "Termo de inquirição", pagina: 9, inicio: 0, fim: 10, trecho: "t" }] }] },
  { titulo: "Propostas do assistente (a confirmar)", itens: [{ titulo: "IMP-1", situacao: "proposta: improcedente", linhas: ["Fundamento: a testemunha relata contenção proporcional."], fontes: [] }] },
] };
const respostas = [
  { perguntaId: "envio", titulo: "Autorizar o envio ao assistente", resumo: "envio ao assistente autorizado", revisavel: false },
  { perguntaId: "delimitacao", titulo: "Confirme as imputações do Memorando Disciplinar", resumo: "1 imputação(ões); acusado: CB QPPM CARLOS INVENTADO DA SILVA", revisavel: true },
  { perguntaId: "midias", titulo: "Mídias citadas e não incluídas no PDF", resumo: "0 de 1 mídia(s) descrita(s)", revisavel: true },
];

const f = (rotulo: string, pagina: number) => ({ rotulo, pagina, inicio: 0, fim: 20, trecho: "trecho citado" });
const perguntaResultado = {
  id: "resultado:IMP-1", titulo: "Resultado da imputação IMP-1", contexto: "Confira a proposta e as fontes. O resultado é seu: o assistente apenas sugere.", revisavel: true,
  grupos: [{ id: "IMP-1", titulo: "IMP-1 — art. 17, inciso XI, do RDPM",
    descricao: ["empregar força ou medida desnecessária em ato de serviço, ainda que não resulte dano; desferir tapas na pessoa abordada durante o atendimento de ocorrência.",
      "Proposta do assistente: improcedente.",
      "Fundamento: A materialidade de ao menos um tapa está amparada pelo depoimento de JOEL FICTÍCIO, que afirmou: “em reação imediata, desferiu um tapa que atingiu o rosto dela.” [EV-0018]",
      "Fundamento: Quanto à dinâmica e à necessidade da força, a testemunha relatou que a abordada “continuava a empurrá-lo e agredi-lo”. Esses relatos favorecem a hipótese de reação a agressões. [EV-0018, EV-0021]",
      "Fundamento: A testemunha declarou: “não observei excesso por parte do militar” e avaliou que a atuação “foi proporcional e necessária para cessar as agressões.” [EV-0019, EV-0020]",
      "Tese da defesa: A defesa afirma que os atos foram reação às agressões e à contenção física, sem caráter ofensivo, retaliatório ou punitivo. [EV-0003, EV-0004]",
      "Tese da defesa: Nas alegações finais, sustenta que não há prova segura da força desnecessária e pede que a dúvida não fundamente sanção. [EV-0023, EV-0024]"],
    alertas: ["Lacuna: Não há registro audiovisual apresentado para análise.", "Lacuna: A prova não esclarece se houve outros tapas além do relatado.", "Diligência sugerida: Ouvir a pessoa abordada sobre a dinâmica dos fatos."],
    fontes: [f("Depoimento de testemunha — JOEL FICTÍCIO (04/09/2026) (EV-0018)", 10), f("Depoimento de testemunha — JOEL FICTÍCIO (04/09/2026) (EV-0021)", 9),
      f("Depoimento de testemunha — JOEL FICTÍCIO (04/09/2026) (EV-0019)", 10), f("Depoimento de testemunha — JOEL FICTÍCIO (04/09/2026) (EV-0020)", 10),
      f("Defesa prévia (EV-0003)", 3), f("Defesa prévia (EV-0004)", 3), f("Defesa prévia (EV-0005)", 3), f("Defesa prévia (EV-0006)", 3), f("Defesa prévia (EV-0007)", 4),
      f("Alegações finais (EV-0023)", 11), f("Alegações finais (EV-0024)", 11), f("Alegações finais (EV-0025)", 12)] }],
  campos: [{ id: "resultado", rotulo: "Resultado", tipo: "ESCOLHA", obrigatorio: true, grupo: "IMP-1", valor: "",
      opcoes: [{ id: "PROCEDENTE", rotulo: "Procedente" }, { id: "IMPROCEDENTE", rotulo: "Improcedente" }, { id: "PARCIAL", rotulo: "Procedente em parte" }] },
    { id: "justificativa", rotulo: "Justificativa (se divergir da proposta)", tipo: "TEXTO_LONGO", obrigatorio: false, grupo: "IMP-1", valor: "" }],
  opcoes: [{ id: "confirmar", rotulo: "Confirmar", validar: true }],
};
const perguntaHistorico = {
  id: "historico", titulo: "Histórico disciplinar recente",
  contexto: "Para conferir a regressão especial de comportamento (RDPM, art. 70), informe as transgressões anteriores reconhecidas em decisão disciplinar nos 12 meses anteriores ao fato. Não inclua acusações pendentes.",
  grupos: [1, 2, 3].map((n) => ({ id: "t" + n, titulo: "Transgressão anterior " + n, descricao: [], alertas: [], fontes: [] })),
  campos: [{ id: "ha", rotulo: "Há transgressões anteriores nesse período?", tipo: "SIM_NAO", obrigatorio: true, valor: "" },
    ...[1, 2, 3].flatMap((n) => [
      { id: "nat" + n, rotulo: "Natureza", tipo: "ESCOLHA", obrigatorio: true, grupo: "t" + n, valor: "", mostrarSe: { campo: "ha", valor: "sim" },
        opcoes: [{ id: "LEVE", rotulo: "Leve" }, { id: "MEDIA", rotulo: "Média" }, { id: "GRAVE", rotulo: "Grave" }] },
      { id: "data" + n, rotulo: "Data da decisão", tipo: "DATA", obrigatorio: true, grupo: "t" + n, valor: "", mostrarSe: { campo: "ha", valor: "sim" } }])],
  opcoes: [{ id: "confirmar", rotulo: "Confirmar", validar: true }],
};
const perguntaEnvio = {
  id: "envio", titulo: "Autorizar o envio ao assistente",
  contexto: "Para analisar os autos, o JURISPM enviará ao assistente trechos do texto extraído das 6 peça(s) identificadas (13 página(s)). O arquivo PDF não é enviado.\n\nAntes do envio, os dados pessoais reconhecidos (6 pessoas, 3 CPFs, 3 RGs) serão trocados por marcadores, e a resposta volta com os dados originais. É um melhor esforço.\n\nSem a sua autorização, nada é enviado.",
  grupos: [{ id: "revisao", titulo: "Possíveis nomes não reconhecidos", descricao: ["Estes trechos parecem nomes de pessoas, mas não têm posto, papel processual ou rótulo que permitam reconhecê-los com segurança. Decida cada um."], alertas: [], fontes: [] }],
  campos: [
    { id: "suspeito_0", rotulo: "Kauan Ficticio Ramos", tipo: "ESCOLHA", obrigatorio: true, grupo: "revisao", valor: "", fontes: [f("pág. 5", 5)],
      opcoes: [{ id: "ocultar", rotulo: "É nome de pessoa ou dado pessoal: ocultar" }, { id: "manter", rotulo: "Não é dado pessoal: enviar como está" }] },
    { id: "crianca", rotulo: "Os autos contêm dados pessoais de criança menor de 13 anos?", tipo: "SIM_NAO", obrigatorio: true, valor: "",
      ajuda: "Os termos do serviço configurado para o assistente proíbem enviar esses dados, mesmo com os nomes ocultados. Se houver, esse serviço fica bloqueado nesta tarefa: escolha outro em Configurações do assistente." },
    { id: "ocultar", rotulo: "Outros nomes ou dados a ocultar (opcional)", tipo: "TEXTO_LONGO", obrigatorio: false, valor: "", ajuda: "Um por linha." }],
  opcoes: [{ id: "autorizar", rotulo: "Autorizar o envio", validar: true }, { id: "nao", rotulo: "Não enviar" }],
};
const perguntaDelim = {
  id: "confirmar_delimitacao", titulo: "Confirme as imputações do Memorando Disciplinar",
  contexto: "O assistente leu o Memorando Disciplinar e propôs as condutas e datas abaixo. Confira cada uma no trecho original e corrija o que for preciso. O enquadramento é seu.",
  grupos: [{ id: "acusado", titulo: "Acusado", descricao: [], alertas: [], fontes: [] },
    { id: "IMP-1", titulo: "IMP-1", descricao: ["empregar força desproporcional ao desferir tapas durante o atendimento de uma ocorrência policial.", "Dispositivo citado no Memorando (para conferência): art. 17, do RDPM."],
      alertas: ["O Memorando cita só o art. 17; o inciso XI é o informado no seu cadastro. Confira."], fontes: [f("Memorando", 1)] }],
  campos: [{ id: "acusado", rotulo: "Acusado", tipo: "TEXTO", obrigatorio: true, grupo: "acusado", valor: "CB QPPM CARLOS INVENTADO DA SILVA", fontes: [f("Memorando", 1)] },
    { id: "incluir_IMP-1", rotulo: "Incluir esta imputação", tipo: "SIM_NAO", obrigatorio: true, grupo: "IMP-1", valor: "sim" },
    { id: "conduta_IMP-1", rotulo: "Conduta imputada", tipo: "TEXTO_LONGO", obrigatorio: true, grupo: "IMP-1", valor: "empregar força desproporcional ao desferir tapas durante o atendimento de uma ocorrência policial.", mostrarSe: { campo: "incluir_IMP-1", valor: "sim" } },
    { id: "data_IMP-1", rotulo: "Data do fato", tipo: "DATA", obrigatorio: true, grupo: "IMP-1", valor: "06/07/2025", ajuda: "DD/MM/AAAA", mostrarSe: { campo: "incluir_IMP-1", valor: "sim" } }],
  opcoes: [{ id: "confirmar", rotulo: "Confirmar imputações", validar: true }, { id: "cancelar", rotulo: "Cancelar a tarefa" }],
};

const ia = {
  chaveCadastrada: true, status: "ok", modelos: [{ id: "gemini-3.6-flash", nome: "Gemini 3.6 Flash", descricao: "", estagio: "estavel", entradaMax: 1048576, saidaMax: 65536, pensamento: true, teste: "", testeErro: null }],
  modeloSelecionado: "gemini-3.6-flash", modeloForaDaLista: false, ocupado: false, uso: null, urlChaves: "", erro: null, provedor: "CHATGPT", aviso: "",
  chatgpt: { disponivel: true, conectada: true, email: "usuario@exemplo.invalid", planoAutorizado: true, reconectar: false, entrando: false,
    modelos: [{ id: "gpt-6-luna", nome: "gpt-6-luna", descricao: "" }], modeloSelecionado: "gpt-6-luna", modeloForaDaLista: false, uso: null, urlUso: "" },
};

const docSEI = (id: string, sei: string, titulo: string, sugestao: string, paginas = 2) => ({ id, sei, titulo, paginas, sugestao, papel: "", usado: "" });

// Minuta sintética (pessoas e fatos fictícios).
const fonteTermo = { rotulo: "Termo de Inquirição, pág. 2", pagina: 2, inicio: 0, fim: 40, trecho: "A testemunha declarou que o acusado não compareceu." };
const bl = (tipo: string, texto: string, origem = "FIXO", fontes: unknown[] = []) => ({ tipo, texto, origem, pendente: texto.includes("[A PREENCHER]"), fontes });
const minuta = {
  validada: true, camposPendentes: 6, versao: 2, achados: [],
  ajustes: [{ pedido: "No relatório, diga que a testemunha é sargento.", aplicado: true, resultado: "Ajuste aplicado em: relatório (relato dos fatos, da defesa e das provas). Os demais trechos não foram alterados." }],
  pendencias: ["Sem pontos de prova sobre a imputação IMP-2, a fundamentação fica a preencher."],
  blocos: [
    bl("TITULO", "JULGAMENTO DE PADS"),
    bl("CABECALHO", "Autoridade disciplinar: [A PREENCHER]"),
    bl("CABECALHO", "Acusado(a): CB PM CARLOS INVENTADO"),
    bl("CABECALHO", "Referência: PADS nº [A PREENCHER]"),
    bl("SECAO", "1. RELATÓRIO"),
    bl("PARAGRAFO", "Trata-se do Processo Administrativo Disciplinar Sumário (PADS) nº [A PREENCHER], instaurado por meio do Memorando Disciplinar incluído nos autos, em relação a CB PM CARLOS INVENTADO."),
    bl("PARAGRAFO", "Conforme o Memorando Disciplinar, foram imputadas as seguintes transgressões disciplinares ao(à) acusado(a):"),
    bl("ITEM", "I – tomar parte em manifestação coletiva de crítica a ato de superior (art. 17, inciso V, do RDPM: “tomar parte em qualquer manifestação coletiva, seja ela de caráter reivindicatório, de crítica ou de apoio a atos de superior”); e"),
    bl("ITEM", "II – deixar de comunicar a impossibilidade de comparecer ao serviço (art. 15, inciso III, do RDPM)."),
    bl("PARAGRAFO", "Ao(à) acusado(a) foi assegurado o exercício da ampla defesa e do contraditório, nos termos do art. 5º, inciso LV, da Constituição Federal e do art. 4º do RDPM."),
    bl("PARAGRAFO", "No Termo de Inquirição incluído nos autos, a testemunha 3º SGT PM JOÃO FICTÍCIO declarou que o acusado participou da reunião no pátio da unidade e leu em voz alta a nota de crítica ao comandante.", "IA", [fonteTermo]),
    bl("PARAGRAFO", "Nas alegações finais, a defesa sustenta que o acusado apenas passava pelo local e não aderiu à manifestação.", "IA", [{ ...fonteTermo, rotulo: "Alegações finais, pág. 4", pagina: 4 }]),
    bl("SECAO", "2. FUNDAMENTAÇÃO"),
    bl("SUBTITULO", "Quanto à imputação do art. 17, inciso V, do RDPM"),
    bl("PARAGRAFO", "Os dois depoimentos colhidos são coerentes entre si quanto à presença do acusado e à leitura da nota. A versão da defesa não encontra apoio em nenhuma das peças.", "IA", [fonteTermo]),
    bl("SUBTITULO", "Quanto à imputação do art. 15, inciso III, do RDPM"),
    bl("PARAGRAFO", "[A PREENCHER] — fundamentação da imputação do art. 15, inciso III, do RDPM."),
    bl("SECAO", "3. DECISÃO"),
    bl("PARAGRAFO", "Dito isso, RESOLVO:"),
    bl("PARAGRAFO", "Julgar PARCIALMENTE PROCEDENTE a acusação descrita no Memorando Disciplinar incluído nos autos, nos seguintes termos:"),
    bl("ITEM", "I – imputação do art. 17, inciso V, do RDPM: PROCEDENTE; e"),
    bl("ITEM", "II – imputação do art. 15, inciso III, do RDPM: IMPROCEDENTE."),
    bl("SUBTITULO", "Quanto à transgressão do art. 17, inciso V, do RDPM"),
    bl("PARAGRAFO", "Em conformidade com o art. 36, incisos I a VII, do RDPM, verifico o seguinte:"),
    bl("ITEM", "I – A personalidade do transgressor e seus antecedentes disciplinares: Não há elementos nos autos sobre este fator."),
    bl("ITEM", "II – Os motivos determinantes: A manifestação foi motivada por discordância quanto à escala de serviço.", "IA", [fonteTermo]),
    bl("PARAGRAFO", "Assim, com base no art. 41, inciso III, e no art. 58, inciso II, alínea c, do RDPM, e consideradas as circunstâncias acima, imponho a CB PM CARLOS INVENTADO a punição disciplinar de 1 (um) dia de PRISÃO, por ter praticado a transgressão disciplinar do art. 17, inciso V, do RDPM."),
    bl("PARAGRAFO", "Como próximas providências:"),
    bl("ITEM", "I – Publicar este julgamento em Boletim Interno; e"),
    bl("ITEM", "II – Manter os documentos do presente PADS em arquivo."),
    bl("FECHO", "CUMPRA-SE."),
    bl("FECHO", "[A PREENCHER] (local), data conforme a assinatura eletrônica."),
    bl("ASSINATURA", "[A PREENCHER] (nome, posto e função da autoridade disciplinar)"),
  ],
};

const perguntaDosimetria = {
  id: "calculo", titulo: "Confira a dosimetria e o comportamento proposto", revisavel: true,
  contexto: "O cálculo é feito pelo aplicativo, não pelo assistente, a partir das suas respostas. Confira a memória de cálculo. A quantidade de dias é uma proposta: ajuste, se for o caso. Para mudar um dado de entrada, use \"Alterar\" em \"Suas respostas\".",
  grupos: [{ id: "s_IMP-1", titulo: "IMP-1 — prisão", descricao: ["Natureza grave: prisão (RDPM, art. 58, II).", "Atenuantes contadas: art. 38, I.", "Agravantes contadas: art. 39, III.",
      "Regra aplicada: 1 atenuante(s) × 1 agravante(s): quantidades iguais → 2 dias."], alertas: [], fontes: [] },
    ...[["Pontuação e comportamento atuais", "31 pontos, comportamento ótimo (informados/confirmados pelo usuário)."], ["Imputações procedentes consideradas", "IMP-1: natureza grave, fato em 05/04/2026, sanção prisão."],
      ["Histórico disciplinar anterior utilizado", "anterior 1: natureza grave, cometida em 05/10/2025, reconhecida em decisão disciplinar."], ["Equivalências e janela temporal", "Equivalência: 8 unidade(s) = 2 graves."],
      ["Regressão especial ou desconto ordinário", "Regressão especial (art. 70): pontuação fixa de 20 pontos."], ["Pontuação e comportamento propostos", "Propõe-se 20 pontos, comportamento bom. A alteração depende do ato administrativo competente."],
      ["Fundamento legal e ressalvas", "RDPM, arts. 70 e 71."]].map(([t, l], i) => ({ id: "m" + (i + 1), titulo: `${i + 1}. ${t}`, descricao: [l], alertas: [], fontes: [] }))],
  campos: [{ id: "dias_IMP-1", rotulo: "Dias de prisão", tipo: "NUMERO", obrigatorio: true, grupo: "s_IMP-1", valor: "2", min: 1, max: 30, ajuda: "Proposta: 2 dia(s)." }],
  opcoes: [{ id: "confirmar", rotulo: "Confirmar", validar: true }],
};
const perguntaDelim2 = { ...perguntaDelim,
  grupos: [...perguntaDelim.grupos, { id: "IMP-2", titulo: "IMP-2", descricao: ["deixou de registrar o abastecimento da viatura no livro próprio", "Natureza declarada no Memorando: leve."], alertas: [], fontes: [f("Memorando", 1)] }],
  campos: [...perguntaDelim.campos,
    { id: "diploma_IMP-1", rotulo: "Enquadramento: diploma", tipo: "ESCOLHA", obrigatorio: true, grupo: "IMP-1", valor: "RDPM", mostrarSe: { campo: "incluir_IMP-1", valor: "sim" },
      opcoes: [{ id: "RDPM", rotulo: "RDPM (arts. 15 a 17)" }, { id: "ESTATUTO", rotulo: "Estatuto (arts. 29 e 32)" }] },
    { id: "disp_rdpm_IMP-1", rotulo: "Dispositivo do RDPM", tipo: "BUSCA", obrigatorio: true, grupo: "IMP-1", valor: "RDPM-17-XI", mostrarSe: { campo: "diploma_IMP-1", valor: "RDPM" },
      opcoes: [{ id: "RDPM-17-XI", rotulo: "art. 17, XI — empregar força ou medida desnecessária em ato de serviço (Grave)" }] },
    { id: "incluir_IMP-2", rotulo: "Incluir esta imputação", tipo: "SIM_NAO", obrigatorio: true, grupo: "IMP-2", valor: "sim" },
    { id: "diploma_IMP-2", rotulo: "Enquadramento: diploma", tipo: "ESCOLHA", obrigatorio: true, grupo: "IMP-2", valor: "RDPM", mostrarSe: { campo: "incluir_IMP-2", valor: "sim" },
      opcoes: [{ id: "RDPM", rotulo: "RDPM (arts. 15 a 17)" }, { id: "ESTATUTO", rotulo: "Estatuto (arts. 29 e 32)" }] },
    { id: "disp_rdpm_IMP-2", rotulo: "Dispositivo do RDPM", tipo: "BUSCA", obrigatorio: true, grupo: "IMP-2", valor: "", mostrarSe: { campo: "diploma_IMP-2", valor: "RDPM" },
      opcoes: [{ id: "RDPM-15-III", rotulo: "art. 15, III — deixar de participar em tempo hábil (Leve)" }] },
    { id: "conduta_IMP-2", rotulo: "Conduta imputada", tipo: "TEXTO_LONGO", obrigatorio: true, grupo: "IMP-2", valor: "deixou de registrar o abastecimento da viatura no livro próprio", mostrarSe: { campo: "incluir_IMP-2", valor: "sim" } },
    { id: "data_IMP-2", rotulo: "Data do fato", tipo: "DATA", obrigatorio: true, grupo: "IMP-2", valor: "05/04/2026", mostrarSe: { campo: "incluir_IMP-2", valor: "sim" } }],
};

export const cenarios: Record<string, { titulo: string; visao: any; ia?: any; config?: boolean; altura?: number; clicar?: string }> = {
  inicio: { titulo: "Início", visao: null, altura: 900 },
  entrada: { titulo: "Entrada dos autos", visao: visao("TIPO_ESCOLHIDO", { entrada: "", etapas: [] }), altura: 900 },
  "cadastro-memorando": { titulo: "Cadastro PADS: Memorando", visao: visao("RECEBENDO_UPLOAD", { cadastro: cadPADS, pendenciasCadastro: ["Anexe as alegações finais ou marque como não apresentadas."] }), altura: 1500 },
  "cadastro-sei": { titulo: "Cadastro PADS pelo SEI", visao: visao("RECEBENDO_UPLOAD", { entrada: "SEI", cadastro: cadPADS,
    documentosSei: [docSEI("D1", "73088120", "PADS Memorando Disciplinar 3", "MEMORANDO"), docSEI("D2", "73240862", "Anexo - Defesa Prévia", "DEFESA_PREVIA"), docSEI("D3", "73100379", "Certidão 94", "OUTRO", 1)] }), altura: 1500 },
  "cadastro-sr": { titulo: "Cadastro SR: Portaria", visao: visao("RECEBENDO_UPLOAD", { tipo: "SR", cadastro: cadSR }), altura: 1300 },
  processando: { titulo: "Processando (progresso)", visao: visao("EXTRAINDO", { etapas: etapas(1), descricao: "Extraindo texto e OCR", atividade: "Lendo a página 7 de 13", progresso: { atual: 7, total: 13, unidade: "páginas" }, autos: resumoAutos }), altura: 1000 },
  analisando: { titulo: "Analisando (assistente, robô)", visao: visao("ANALISANDO", { etapas: etapas(4), descricao: "Analisando as provas.", atividade: "Lendo Depoimento de testemunha — JOEL FICTÍCIO", progresso: { atual: 1, total: 3, unidade: "peças lidas" }, autos: resumoAutos, painel, respostas }), altura: 900 },
  "analisando-respostas": { titulo: "Gaveta Suas respostas", visao: visao("ANALISANDO", { etapas: etapas(4), descricao: "Analisando as provas.", autos: resumoAutos, painel, respostas }), altura: 900, clicar: "abrir-respostas" },
  "analisando-analise": { titulo: "Gaveta Análise", visao: visao("ANALISANDO", { etapas: etapas(4), descricao: "Analisando as provas.", autos: resumoAutos, painel, respostas }), altura: 900, clicar: "abrir-analise" },
  "autos-lidos": { titulo: "Modal Autos lidos", visao: visao("ANALISANDO", { etapas: etapas(4), descricao: "Analisando as provas.", autos: resumoAutos, painel, respostas }), altura: 900, clicar: "abrir-autos-lidos" },
  dosimetria: { titulo: "Dosimetria", visao: visao("AGUARDANDO_ESCLARECIMENTO", { etapas: etapas(6), pergunta: perguntaDosimetria, autos: resumoAutos, painel, respostas }), altura: 1000 },
  "dosimetria-memoria": { titulo: "Modal memória de cálculo", visao: visao("AGUARDANDO_ESCLARECIMENTO", { etapas: etapas(6), pergunta: perguntaDosimetria, autos: resumoAutos, painel, respostas }), altura: 1000, clicar: "abrir-memoria" },
  "delimitacao-2": { titulo: "Confirmar duas imputações", visao: visao("AGUARDANDO_ESCLARECIMENTO", { etapas: etapas(3), pergunta: perguntaDelim2, autos: resumoAutos, respostas: respostas.slice(0, 1) }), altura: 1100 },
  "historico-sim": { titulo: "Histórico disciplinar (sim)", visao: visao("AGUARDANDO_ESCLARECIMENTO", { pergunta: { ...perguntaHistorico, campos: perguntaHistorico.campos.map((c: any) => (c.id === "ha" ? { ...c, valor: "sim" } : c)) }, autos: resumoAutos, painel, respostas }), altura: 1000 },
  "cadastro-modal": { titulo: "Cadastro PADS: modal do Memorando", visao: visao("RECEBENDO_UPLOAD", { cadastro: cadPADS }), altura: 1000, clicar: "passo-memorando" },
  punicao: { titulo: "Punição anterior (pergunta curta)", visao: visao("AGUARDANDO_ESCLARECIMENTO", { etapas: etapas(4), autos: resumoAutos, painel, respostas, pergunta: {
    id: "reincidencia:GRAVE", titulo: "Punição anterior de natureza grave", contexto: "O acusado possui punição anterior de natureza grave? A reincidência depende da data de TÉRMINO do cumprimento da punição anterior.",
    opcoes: [{ id: "confirmar", rotulo: "Confirmar", validar: true }], grupos: [],
    campos: [{ id: "possui", rotulo: "Possui punição anterior de natureza grave?", tipo: "SIM_NAO", obrigatorio: true, valor: "sim" },
      { id: "termino", rotulo: "Data de término do cumprimento", tipo: "DATA", valor: "", ajuda: "DD/MM/AAAA. Sem a data, a reincidência não é reconhecida.", mostrarSe: { campo: "possui", valor: "sim" } },
      { id: "hora", rotulo: "Hora do término (só se for no mesmo dia do fato)", tipo: "TEXTO", valor: "", ajuda: "HH:MM", mostrarSe: { campo: "possui", valor: "sim" } }] } }), altura: 800 },
  "cadastro-depoimentos": { titulo: "Cadastro PADS: depoimentos", visao: visao("RECEBENDO_UPLOAD", { cadastro: { ...cadPADS, pecas: [...cadPADS.pecas, peca("PEC-05", "DEPOIMENTO", { papel: "DECLARANTE", pessoaIds: ["P2"] })] } }), altura: 1100, clicar: "passo-depoimentos" },
  "cadastro-conferencia": { titulo: "Cadastro PADS: conferência", visao: visao("RECEBENDO_UPLOAD", { cadastro: cadPADS, pendenciasCadastro: ["Anexe as alegações finais ou marque como não apresentadas."] }), altura: 1000, clicar: "passo-conferencia" },
  envio: { titulo: "Autorizar o envio", visao: visao("AGUARDANDO_CONFIRMACAO", { etapas: etapas(3), pergunta: perguntaEnvio, autos: resumoAutos }), altura: 1500 },
  delimitacao: { titulo: "Confirmar imputações", visao: visao("AGUARDANDO_ESCLARECIMENTO", { etapas: etapas(3), pergunta: perguntaDelim, autos: resumoAutos, respostas: respostas.slice(0, 1) }), altura: 1500 },
  resultado: { titulo: "Resultado da imputação", visao: visao("AGUARDANDO_ESCLARECIMENTO", { pergunta: perguntaResultado, autos: resumoAutos, painel, respostas }), altura: 2200 },
  historico: { titulo: "Histórico disciplinar", visao: visao("AGUARDANDO_ESCLARECIMENTO", { pergunta: perguntaHistorico, autos: resumoAutos, painel, respostas }), altura: 1200 },
  parada: { titulo: "Parada TAR-004", visao: visao("CONSULTANDO_FONTES", { etapas: etapas(5), parada: true, erro: { codigo: "TAR-004", titulo: "t", mensagem: "m", acao: "a" }, autos: resumoAutos, painel, respostas }), altura: 1400 },
  minuta: { titulo: "Minuta em revisão", visao: visao("AGUARDANDO_REVISAO", { etapas: etapas(9), descricao: "A minuta está pronta para sua revisão.", autos: resumoAutos, painel, respostas, minuta }), altura: 1100 },
  "minuta-cheia": { titulo: "Minuta em tela cheia", visao: visao("AGUARDANDO_REVISAO", { etapas: etapas(9), descricao: "A minuta está pronta para sua revisão.", autos: resumoAutos, painel, respostas, minuta }), altura: 900, clicar: "minuta-tela-cheia" },
  finalizada: { titulo: "Tarefa finalizada", visao: visao("FINALIZADA", { terminal: true, etapas: etapas(10) }), altura: 900 },
  "erro-ia": { titulo: "Erro do assistente (retomável)", visao: visao("ANALISANDO", { parada: true, retomavel: true, erro: { codigo: "IA-003", titulo: "Limite de uso do plano atingido", mensagem: "O limite de uso do seu plano foi atingido.", acao: "Aguarde a renovação do limite ou escolha outro serviço em Configurações do assistente." }, autos: resumoAutos }), altura: 1000 },
  config: { titulo: "Configurações do assistente", visao: null, config: true, altura: 1200 },
};
export const iaPadrao = ia;

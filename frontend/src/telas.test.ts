import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";

// Backend simulado: a FSM real é testada em Go; aqui só a interface.
const chamadas: string[] = [];
let visaoAtual: any = null;
const { modelosBase } = vi.hoisted(() => ({
  modelosBase: {
    hardware: { nucleosFisicos: 6, nucleosLogicos: 12, avx2: true, ramTotal: 16 * 2 ** 30, ramLivre: 12 * 2 ** 30, gpus: [] },
    modelos: [
      { id: "qwen3.5-4b-q4km", nome: "Qwen3.5-4B (Q4_K_M)", perfil: "padrao", papel: "gerador", tamanho: 2.7 * 2 ** 30, licenca: "Apache-2.0", testado: true, nota: "", instalado: false, recomendado: true, selecionado: false, cabe: true, contexto: 16384 },
      { id: "qwen3.5-2b-q4km", nome: "Qwen3.5-2B (Q4_K_M)", perfil: "basico", papel: "gerador", tamanho: 1.2 * 2 ** 30, licenca: "Apache-2.0", testado: true, nota: "", instalado: false, recomendado: false, selecionado: false, cabe: true, contexto: 16384 },
    ],
    avisos: [], manual: "", gpu: "desconhecida", gpus: [], andamento: null, pronto: false, servidor: true, erro: null,
  },
}));
let modelosAtual: any = modelosBase;

function visao(estado: string, extra: Record<string, unknown> = {}) {
  return {
    id: "t1", tipo: "PADS", entrada: "", estado, descricao: "Descrição de " + estado, terminal: false,
    etapas: [
      { chave: "obter", rotulo: "Obtendo os autos", situacao: "concluida" },
      { chave: "extrair", rotulo: "Extraindo texto e OCR", situacao: "ativa" },
      { chave: "analisar", rotulo: "Analisando provas", situacao: "pendente" },
    ],
    progresso: null, erro: null, aviso: null, ...extra,
  };
}

function resp(v: any, erro: any = null) {
  visaoAtual = v;
  return Promise.resolve({ visao: v, erro });
}

vi.mock("./api", () => ({
  api: {
    atual: () => { chamadas.push("atual"); return resp(visaoAtual); },
    iniciar: () => { chamadas.push("iniciar"); return resp(visao("NOVA", { tipo: "" })); },
    escolherTipo: (t: string) => { chamadas.push("tipo:" + t); return resp(visao("TIPO_ESCOLHIDO", { tipo: t })); },
    escolherEntrada: (e: string) => {
      chamadas.push("entrada:" + e);
      if (e === "SEI") {
        return resp(visaoAtual, { codigo: "TAR-004", titulo: "Etapa ainda não disponível", mensagem: "m", acao: "a" });
      }
      return resp(visao("RECEBENDO_UPLOAD", { tipo: visaoAtual.tipo, entrada: e }));
    },
    voltarParaEntrada: () => { chamadas.push("voltar"); return resp(visao("TIPO_ESCOLHIDO")); },
    selecionarPDF: () => { chamadas.push("pdf"); return resp(visao("EXTRAINDO", { tipo: visaoAtual.tipo })); },
    cancelar: () => { chamadas.push("cancelar"); return resp(visao("CANCELADA", { terminal: true })); },
    descartar: () => { chamadas.push("descartar"); return resp(null); },
    aoMudar: () => () => {},
  },
  responder: (p: string, o: string, vals: Record<string, string> = {}) => {
    chamadas.push(`responder:${p}:${o}:${JSON.stringify(vals)}`);
    return resp(visao("DELIMITANDO", { parada: true, erro: { codigo: "TAR-004", titulo: "t", mensagem: "m", acao: "a" } }));
  },
  tarefa: {
    revisar: (id: string) => { chamadas.push("revisar:" + id); return resp(visao("DELIMITANDO")); },
    retomar: () => { chamadas.push("retomar"); return resp(visao("DELIMITANDO")); },
  },
  modelos: {
    estado: () => { chamadas.push("modelos"); return Promise.resolve(modelosAtual); },
    baixar: (id: string) => { chamadas.push("baixar:" + id); return Promise.resolve({ ...modelosAtual, andamento: { modeloId: id, acao: "baixando", feito: 2 ** 30, total: 2 * 2 ** 30 } }); },
    importar: () => { chamadas.push("importar"); return Promise.resolve(modelosAtual); },
    cancelar: () => Promise.resolve(modelosAtual),
    remover: (id: string) => { chamadas.push("remover:" + id); return Promise.resolve(modelosAtual); },
    selecionar: (id: string) => { chamadas.push("selecionar:" + id); return Promise.resolve(modelosAtual); },
    testarGPU: () => { chamadas.push("testarGPU"); return Promise.resolve(modelosAtual); },
    usarSoCPU: () => Promise.resolve(modelosAtual),
    aoMudar: () => () => {},
  },
  credenciais: {
    informar: (id: string, u: string, senha: string, o: string, p: string) => {
      chamadas.push(`credenciais:${id}:${u}:${senha}:${o}:${p}`);
      return resp(visao("SEI_OTP", { pedido: { id: "p2", tipo: "OTP", alerta: null } }));
    },
    segredo: (id: string, valor: string) => {
      chamadas.push(`segredo:${id}:${valor}`);
      return resp(visao("SEI_LOCALIZANDO"));
    },
  },
  fontes: {
    enderecoPDF: () => Promise.resolve({ endereco: "", erro: { codigo: "TAR-003", titulo: "", mensagem: "", acao: "" } }),
    destaque: () => Promise.resolve({ pagina: 1, largura: 0, altura: 0, caixas: [], erro: null }),
    buscar: () => Promise.resolve({ ocorrencias: [], erro: null }),
  },
}));

import App from "./App.vue";
import { estado } from "./estado";

beforeEach(() => {
  chamadas.length = 0;
  visaoAtual = null;
  modelosAtual = modelosBase;
  estado.visao = null;
  estado.erro = null;
  estado.ocupado = false;
  estado.modelos = null;
  estado.verModelos = false;
});

describe("T01 — ordem das escolhas", () => {
  it("primeira tela oferece só PADS e SR; a entrada vem depois", async () => {
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="tipo-PADS"]').exists()).toBe(true);
    expect(w.find('[data-teste="tipo-SR"]').exists()).toBe(true);
    expect(w.find('[data-teste="entrada-SEI"]').exists()).toBe(false);
    expect(w.find('[data-teste="entrada-UPLOAD"]').exists()).toBe(false);

    await w.find('[data-teste="tipo-SR"]').trigger("click");
    await flushPromises();
    expect(chamadas).toEqual(["atual", "modelos", "iniciar", "tipo:SR"]);
    expect(w.find('[data-teste="entrada-SEI"]').exists()).toBe(true);
    expect(w.find('[data-teste="entrada-UPLOAD"]').exists()).toBe(true);
    expect(w.find('[data-teste="faixa-tipo"]').text()).toContain("Relatório de Sindicância Regular");
  });

  it("SEI indisponível mostra aviso e mantém a escolha do tipo", async () => {
    visaoAtual = visao("TIPO_ESCOLHIDO");
    const w = mount(App);
    await flushPromises();
    await w.find('[data-teste="entrada-SEI"]').trigger("click");
    await flushPromises();
    expect(w.find('[data-teste="aviso-erro"]').text()).toContain("Etapa ainda não disponível");
    expect(w.find('[data-teste="faixa-tipo"]').text()).toContain("Julgamento de PADS");
  });
});

describe("T02 — upload sem credencial", () => {
  it("enviar PDF escolhe UPLOAD e abre o seletor, sem campo de senha", async () => {
    visaoAtual = visao("TIPO_ESCOLHIDO");
    const w = mount(App);
    await flushPromises();
    expect(w.find('input[type="password"]').exists()).toBe(false);
    await w.find('[data-teste="entrada-UPLOAD"]').trigger("click");
    await flushPromises();
    expect(chamadas).toEqual(["atual", "modelos", "entrada:UPLOAD", "pdf"]);
    expect(w.find('input[type="password"]').exists()).toBe(false);
    expect(w.find('[data-teste="linha-do-tempo"]').exists()).toBe(true);
  });
});

describe("T07 — tarefa em execução", () => {
  it("mostra status e etapas, sem chatbot nem texto em streaming", async () => {
    visaoAtual = visao("EXTRAINDO");
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="estado-atual"]').text()).toContain("Descrição de EXTRAINDO");
    expect(w.findAll('[data-teste="linha-do-tempo"] li')).toHaveLength(3);
    expect(w.find("textarea").exists()).toBe(false);
    expect(w.find('input[type="text"]').exists()).toBe(false);
    // Sem métrica real: indicação de atividade, nunca percentual.
    expect(w.find('[data-teste="atividade"]').exists()).toBe(true);
    expect(w.find('[role="progressbar"]').exists()).toBe(false);
    expect(w.text()).not.toMatch(/%/);
  });

  it("com métrica real mostra a barra e a contagem", async () => {
    visaoAtual = visao("EXTRAINDO", { progresso: { atual: 30, total: 120, unidade: "páginas" } });
    const w = mount(App);
    await flushPromises();
    const barra = w.find('[role="progressbar"]');
    expect(barra.attributes("aria-valuenow")).toBe("25");
    expect(w.find('[data-teste="progresso"]').text()).toContain("30 de 120 páginas");
  });

  it("cancelar exige confirmação e depois volta ao início", async () => {
    visaoAtual = visao("EXTRAINDO");
    const w = mount(App);
    await flushPromises();
    await w.find('[data-teste="cancelar"]').trigger("click");
    expect(chamadas).not.toContain("cancelar");
    await w.find('[data-teste="confirmar-cancelar"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("cancelar");
    await w.find('[data-teste="voltar-inicio"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("descartar");
    expect(w.find('[data-teste="tipo-PADS"]').exists()).toBe(true);
  });

  it("falha mostra o erro do catálogo e permite recomeçar", async () => {
    visaoAtual = visao("FALHA", {
      terminal: true,
      erro: { codigo: "TAR-004", titulo: "Etapa ainda não disponível", mensagem: "Esta etapa…", acao: "Inicie…" },
    });
    const w = mount(App);
    await flushPromises();
    expect(w.text()).toContain("A tarefa não pôde continuar");
    expect(w.find('[data-teste="aviso-erro"]').text()).toContain("Etapa ainda não disponível");
    expect(w.find('[data-teste="voltar-inicio"]').exists()).toBe(true);
  });
});

const resumo = {
  totalPaginas: 9,
  compatibilidade: "COMPATIVEL",
  pecaCentral: "PEC-01",
  paginas: [
    { numero: 1, situacao: "LIDA", origem: "NATIVO", confianca: 0, peca: "PEC-01" },
    { numero: 2, situacao: "OCR_BAIXO", origem: "OCR", confianca: 60, peca: "PEC-02" },
    { numero: 3, situacao: "VAZIA", origem: "", confianca: 0, peca: "PEC-02" },
  ],
  pecas: [
    { id: "PEC-01", titulo: "MEMORANDO DISCIPLINAR", funcao: "MEMORANDO_DISCIPLINAR", rotulo: "Memorando Disciplinar", paginaInicial: 1, paginaFinal: 1, sei: "0090000001", provavel: false, externo: false, situacao: "LIDA" },
    { id: "PEC-02", titulo: "MANIFESTAÇÃO", funcao: "DEFESA_PREVIA", rotulo: "Defesa prévia", paginaInicial: 2, paginaFinal: 3, sei: "", provavel: true, externo: true, situacao: "OCR_BAIXO" },
  ],
  midias: [
    { pagina: 2, peca: "PEC-02", termo: "gravação", inicio: 10, fim: 18 },
    { pagina: 3, peca: "PEC-02", termo: "Vídeo", inicio: 1, fim: 6 },
    { pagina: 3, peca: "PEC-02", termo: "vídeo", inicio: 20, fim: 25 },
  ],
};

describe("Fase 2 — autos, perguntas e parada", () => {
  it("pergunta de compatibilidade mostra opções e envia a resposta (T04)", async () => {
    visaoAtual = visao("AGUARDANDO_ESCLARECIMENTO", {
      pergunta: { id: "compatibilidade", titulo: "Os autos parecem ser de Relatório de Sindicância Regular", contexto: "…", opcoes: [
        { id: "outro", rotulo: "Enviar outro PDF" }, { id: "continuar", rotulo: "Continuar como Julgamento de PADS" }, { id: "cancelar", rotulo: "Cancelar a tarefa" },
      ] },
    });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="pergunta"]').text()).toContain("Relatório de Sindicância Regular");
    expect(w.find('[data-teste="faixa-tipo"]').text()).toContain("Julgamento de PADS");
    await w.find('[data-teste="opcao-continuar"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("responder:compatibilidade:continuar:{}");
    expect(w.find('[data-teste="parada"]').exists()).toBe(true);
  });

  it("painel mostra peças, cobertura, páginas incertas e mídias; parada é explícita", async () => {
    visaoAtual = visao("DELIMITANDO", { parada: true, autos: resumo, erro: { codigo: "TAR-004", titulo: "t", mensagem: "m", acao: "a" } });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="parada"]').exists()).toBe(true);
    expect(w.find('[data-teste="estado-atual"]').exists()).toBe(false);
    const pecas = w.find('[data-teste="pecas"]').text();
    expect(pecas).toContain("Memorando Disciplinar");
    expect(pecas).toContain("peça central");
    expect(pecas).toContain("provável, pela ordem dos autos");
    expect(pecas).toContain("documento externo");
    expect(w.find('[data-teste="cobertura"]').text()).toContain("1 OCR de baixa qualidade");
    expect(w.find('[data-teste="paginas-problema"]').text()).toContain("2");
    const midias = w.find('[data-teste="midias"]');
    expect(midias.findAll("li")).toHaveLength(1); // agrupadas por peça
    expect(midias.text()).toContain("Defesa prévia");
    expect(midias.text()).toContain("gravação, vídeo · 3 menções");
    expect(w.find('[data-teste="cancelar"]').exists()).toBe(true);
  });

  it("PDF protegido volta ao envio com o alerta, mantendo o tipo", async () => {
    visaoAtual = visao("RECEBENDO_UPLOAD", { erro: { codigo: "PDF-004", titulo: "PDF protegido por senha", mensagem: "m", acao: "a" } });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="aguardando-pdf"]').exists()).toBe(true);
    expect(w.find('[data-teste="erro-arquivo"]').text()).toContain("PDF protegido por senha");
    expect(w.find('[data-teste="faixa-tipo"]').text()).toContain("Julgamento de PADS");
  });
});

describe("Fase 2 — SEI", () => {
  it("credenciais e código vão ao backend e os campos secretos são limpos (SEG13-01)", async () => {
    visaoAtual = visao("SEI_LOGIN", { entrada: "SEI", pedido: { id: "p1", tipo: "CREDENCIAIS", alerta: null } });
    const w = mount(App);
    await flushPromises();
    const f = w.find('[data-teste="form-credenciais"]');
    const campos = f.findAll("input");
    await campos[0].setValue("fulano");
    await campos[1].setValue("segredo-123");
    await campos[3].setValue("0000.000001/2026-01");
    expect((campos[2].element as HTMLInputElement).value).toBe("PM");
    await f.trigger("submit");
    await flushPromises();
    expect(chamadas).toContain("credenciais:p1:fulano:segredo-123:PM:0000.000001/2026-01");
    const otp = w.find('[data-teste="form-otp"]');
    expect(otp.exists()).toBe(true);
    await w.find('[data-teste="campo-codigo"]').setValue("654321");
    await otp.trigger("submit");
    await flushPromises();
    expect(chamadas).toContain("segredo:p2:654321");
    expect(w.find('[data-teste="sei-andamento"]').text()).toContain("Descrição de SEI_LOCALIZANDO");
    expect(w.html()).not.toContain("segredo-123");
    expect(w.html()).not.toContain("654321");
  });

  it("reautenticação de sigiloso pede só a senha (SEG13-02) e mostra o alerta", async () => {
    visaoAtual = visao("SEI_REAUTENTICACAO", {
      entrada: "SEI",
      pedido: { id: "p3", tipo: "REAUTENTICACAO", alerta: { codigo: "SEI-002", titulo: "Usuário ou senha não aceitos", mensagem: "m", acao: "a" } },
    });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="form-reautenticacao"]').text()).toContain("sigiloso");
    expect(w.find('[data-teste="form-reautenticacao"]').findAll("input")).toHaveLength(1);
    expect(w.find('[data-teste="alerta-sei"]').text()).toContain("Usuário ou senha não aceitos");
  });

  it("T03: sem internet volta à escolha com o aviso e oferece o envio do PDF", async () => {
    visaoAtual = visao("TIPO_ESCOLHIDO", { tipo: "SR", erro: { codigo: "REDE-001", titulo: "Sem conexão com a internet", mensagem: "m", acao: "Envie o PDF completo do processo." } });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="erro-arquivo"]').text()).toContain("Sem conexão com a internet");
    expect(w.find('[data-teste="entrada-UPLOAD"]').exists()).toBe(true);
    expect(w.find('[data-teste="faixa-tipo"]').text()).toContain("Relatório de Sindicância Regular");
  });
});

const perguntaForm = {
  id: "confirmar_delimitacao", titulo: "Confirme as imputações", contexto: "c",
  opcoes: [{ id: "confirmar", rotulo: "Confirmar imputações", validar: true }, { id: "cancelar", rotulo: "Cancelar", validar: false }],
  grupos: [{ id: "IMP-1", titulo: "IMP-1", descricao: ["deixar de comparecer"], alertas: ["Data do fato não identificada"],
    fontes: [{ rotulo: "Memorando", pagina: 1, inicio: 10, fim: 40, trecho: "no dia…" }] }],
  campos: [
    { id: "incluir_IMP-1", rotulo: "Incluir", tipo: "SIM_NAO", opcoes: [], obrigatorio: true, valor: "sim", ajuda: "", grupo: "IMP-1", min: 0, max: 0, fontes: [], mostrarSe: null },
    { id: "data_IMP-1", rotulo: "Data do fato", tipo: "DATA", opcoes: [], obrigatorio: true, valor: "", ajuda: "", grupo: "IMP-1", min: 0, max: 0, fontes: [], mostrarSe: { campo: "incluir_IMP-1", valor: "sim" } },
  ],
};

describe("F4 — perguntas com campos (T08)", () => {
  it("valida campo obrigatório visível, mostra a fonte e envia os valores", async () => {
    visaoAtual = visao("AGUARDANDO_ESCLARECIMENTO", { pergunta: perguntaForm });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="grupo-IMP-1"]').text()).toContain("Data do fato não identificada");
    expect(w.find('[data-teste="fonte-IMP-1-0"]').text()).toContain("pág. 1");
    await w.find('[data-teste="opcao-confirmar"]').trigger("click");
    await flushPromises();
    expect(chamadas.some((c) => c.startsWith("responder"))).toBe(false); // data obrigatória vazia
    await w.find('[data-teste="campo-data_IMP-1"]').setValue("31/02/2026");
    await w.find('[data-teste="opcao-confirmar"]').trigger("click");
    await flushPromises();
    expect(chamadas.some((c) => c.startsWith("responder"))).toBe(false); // data inexistente
    await w.find('[data-teste="campo-data_IMP-1"]').setValue("07/11/2026");
    await w.find('[data-teste="opcao-confirmar"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain('responder:confirmar_delimitacao:confirmar:{"incluir_IMP-1":"sim","data_IMP-1":"07/11/2026"}');
  });

  it("campo condicional some e não é enviado", async () => {
    visaoAtual = visao("AGUARDANDO_ESCLARECIMENTO", { pergunta: perguntaForm });
    const w = mount(App);
    await flushPromises();
    await w.find('[data-teste="campo-incluir_IMP-1-nao"]').setValue();
    expect(w.find('[data-teste="campo-data_IMP-1"]').exists()).toBe(false);
    await w.find('[data-teste="opcao-confirmar"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain('responder:confirmar_delimitacao:confirmar:{"incluir_IMP-1":"nao"}');
  });

  it("opção sem validação (cancelar) não exige campos", async () => {
    visaoAtual = visao("AGUARDANDO_ESCLARECIMENTO", { pergunta: perguntaForm });
    const w = mount(App);
    await flushPromises();
    await w.find('[data-teste="opcao-cancelar"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("responder:confirmar_delimitacao:cancelar:{}");
  });
});

describe("F4 — revisão, retomada e painel", () => {
  it("parada recuperável (MOD-001): tentar novamente e abrir modelos", async () => {
    visaoAtual = visao("DELIMITANDO", { parada: true, retomavel: true, erro: { codigo: "MOD-001", titulo: "Modelo de IA não instalado", mensagem: "m", acao: "a" } });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="parada"]').exists()).toBe(false);
    await w.find('[data-teste="abrir-modelos"]').trigger("click");
    await flushPromises();
    expect(w.find('[data-teste="tela-modelos"]').exists()).toBe(true);
    await w.find('[data-teste="fechar-modelos"]').trigger("click");
    await flushPromises();
    await w.find('[data-teste="retomar"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("retomar");
  });

  it("respostas dadas podem ser alteradas; painel mostra fontes", async () => {
    visaoAtual = visao("CONSULTANDO_FONTES", {
      parada: true, erro: { codigo: "TAR-004", titulo: "t", mensagem: "m", acao: "a" },
      respostas: [{ perguntaId: "resultado:IMP-1", titulo: "Resultado da imputação IMP-1", resumo: "procedente", revisavel: true }],
      painel: { secoes: [{ titulo: "Propostas da IA (a confirmar)", itens: [{ titulo: "IMP-1", situacao: "proposta: procedente", linhas: ["Fundamento: x"], fontes: [{ rotulo: "Termo", pagina: 2, inicio: 0, fim: 5, trecho: "t" }] }] }] },
    });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="parada"]').text()).toContain("biblioteca jurídica");
    expect(w.find('[data-teste="painel-analise"]').text()).toContain("Termo — pág. 2");
    await w.find('[data-teste="revisar-resultado:IMP-1"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("revisar:resultado:IMP-1");
  });
});

describe("T27 — modelos não instalados", () => {
  it("primeiro uso oferece instalar o modelo recomendado", async () => {
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="primeiro-uso"]').exists()).toBe(true);
    await w.find('[data-teste="instalar-modelo"]').trigger("click");
    await flushPromises();
    const tela = w.find('[data-teste="tela-modelos"]');
    expect(tela.text()).toContain("Recomendado para este computador");
    await w.find('[data-teste="baixar-qwen3.5-4b-q4km"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("baixar:qwen3.5-4b-q4km");
    expect(w.find('[data-teste="andamento"]').text()).toContain("1 de 2 GB");
  });

  it("teste da placa de vídeo exige confirmação (D-08)", async () => {
    const w = mount(App);
    await flushPromises();
    await w.find('[data-teste="link-modelos"]').trigger("click");
    await flushPromises();
    await w.find('[data-teste="testar-gpu"]').trigger("click");
    expect(chamadas).not.toContain("testarGPU");
    await w.find('[data-teste="confirmar-gpu"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("testarGPU");
  });

  it("com modelo instalado não há aviso de primeiro uso", async () => {
    modelosAtual = { ...modelosBase, pronto: true };
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="primeiro-uso"]').exists()).toBe(false);
  });
});

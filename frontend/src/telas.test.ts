import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";

// Backend simulado: a FSM real é testada em Go; aqui só a interface.
const chamadas: string[] = [];
let visaoAtual: any = null;

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
  responder: (p: string, o: string) => {
    chamadas.push(`responder:${p}:${o}`);
    return resp(visao("DELIMITANDO", { parada: true, erro: { codigo: "TAR-004", titulo: "t", mensagem: "m", acao: "a" } }));
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
  estado.visao = null;
  estado.erro = null;
  estado.ocupado = false;
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
    expect(chamadas).toEqual(["atual", "iniciar", "tipo:SR"]);
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
    expect(chamadas).toEqual(["atual", "entrada:UPLOAD", "pdf"]);
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
  midias: [{ pagina: 2, peca: "PEC-02", termo: "gravação", inicio: 10, fim: 18 }],
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
    expect(chamadas).toContain("responder:compatibilidade:continuar");
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
    expect(w.find('[data-teste="midias"]').text()).toContain("gravação");
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

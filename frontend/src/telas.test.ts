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

import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";

// Backend simulado: a FSM real é testada em Go; aqui só a interface.
const chamadas: string[] = [];
let visaoAtual: any = null;
// Ouvinte do evento tarefa:visao e gancho para emitir eventos durante um comando.
const ganchos = vi.hoisted(() => ({ ouvinte: null as null | ((v: any) => void), antesDeResponder: null as null | (() => void) }));
const { iaBase } = vi.hoisted(() => ({
  iaBase: {
    chaveCadastrada: false, status: "sem_chave", modelos: [], modeloSelecionado: "", modeloForaDaLista: false,
    ocupado: false, uso: null, urlChaves: "https://aistudio.google.com/api-keys", erro: null,
    provedor: "", aviso: "",
    chatgpt: { disponivel: true, conectada: false, email: "", planoAutorizado: false, reconectar: false, entrando: false,
      modelos: [], modeloSelecionado: "", modeloForaDaLista: false, uso: null, urlUso: "https://chatgpt.com/#settings" },
  },
}));
const iaModelos = [
  { id: "gemini-3.6-flash", nome: "Gemini 3.6 Flash", descricao: "", estagio: "estavel", entradaMax: 1048576, saidaMax: 65536, pensamento: true, teste: "", testeErro: null },
  { id: "gemini-3.8-flash-preview", nome: "Gemini 3.8 Flash (prévia)", descricao: "", estagio: "previa", entradaMax: 1048576, saidaMax: 65536, pensamento: true, teste: "", testeErro: null },
];
let iaAtual: any = iaBase;

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

// Cadastro inicial do PADS, como o Go o cria (peças fixas da modalidade).
function cadastroPADS(extra: Record<string, unknown> = {}) {
  const peca = (id: string, tipo: string) => ({ id, tipo, titulo: "", descricao: "", papel: "", pessoaIds: [], data: "", sei: "", naoApresentada: false, arquivos: [] });
  return {
    tipo: "PADS", pessoas: [], acusadoId: "", fatos: [], portaria: "", sindicados: [], ofendidos: [], semOfendido: false,
    pecas: [peca("PEC-01", "MEMORANDO"), peca("PEC-02", "DEFESA_PREVIA"), peca("PEC-03", "ALEGACOES_FINAIS")], concluido: false, ...extra,
  };
}
// O "Go" simulado do cadastro: IDs provisórios viram definitivos.
function salvarSimulado(c: any) {
  let n = 10;
  const novo: Record<string, string> = {};
  const real = (id: string) => (id.startsWith("NOVO-") ? (novo[id] ??= `PES-${n++}`) : id);
  const pessoas = c.pessoas.map((p: any) => ({ ...p, id: real(p.id) }));
  const pecas = c.pecas.map((p: any) => ({ ...p, id: p.id.startsWith("NOVO-") ? `PEC-${n++}` : p.id, pessoaIds: p.pessoaIds.map(real) }));
  const fatos = c.fatos.map((f: any, i: number) => ({ ...f, id: f.id.startsWith("NOVO-") ? `FATO-${n + i}` : f.id }));
  return { ...c, pessoas, pecas, fatos, acusadoId: real(c.acusadoId) };
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
      return resp(visao("RECEBENDO_UPLOAD", { tipo: visaoAtual.tipo, entrada: e, cadastro: cadastroPADS(), pendenciasCadastro: ["Anexe o Memorando Disciplinar."] }));
    },
    voltarParaEntrada: () => { chamadas.push("voltar"); return resp(visao("TIPO_ESCOLHIDO")); },
    selecionarPDF: () => { chamadas.push("pdf"); return resp(visao("EXTRAINDO", { tipo: visaoAtual.tipo })); },
    cancelar: () => { chamadas.push("cancelar"); return resp(visao("CANCELADA", { terminal: true })); },
    descartar: () => { chamadas.push("descartar"); return resp(null); },
    aoMudar: (cb: (v: any) => void) => { ganchos.ouvinte = cb; return () => {}; },
  },
  responder: (p: string, o: string, vals: Record<string, string> = {}) => {
    chamadas.push(`responder:${p}:${o}:${JSON.stringify(vals)}`);
    if (ganchos.antesDeResponder) {
      ganchos.antesDeResponder();
      return Promise.resolve({ visao: visao("DELIMITANDO", { sequencia: 6 }), erro: null });
    }
    return resp(visao("DELIMITANDO", { parada: true, erro: { codigo: "TAR-004", titulo: "t", mensagem: "m", acao: "a" } }));
  },
  tarefa: {
    revisar: (id: string) => { chamadas.push("revisar:" + id); return resp(visao("DELIMITANDO")); },
    retomar: () => { chamadas.push("retomar"); return resp(visao("DELIMITANDO")); },
  },
  ia: {
    estado: () => { chamadas.push("ia"); return Promise.resolve(iaAtual); },
    salvarChave: (c: string) => {
      chamadas.push("salvarChave:" + c.length);
      iaAtual = { ...iaAtual, chaveCadastrada: true, status: "valida", modelos: iaModelos };
      return Promise.resolve(iaAtual);
    },
    removerChave: () => { chamadas.push("removerChave"); iaAtual = { ...iaBase }; return Promise.resolve(iaAtual); },
    verificar: () => { chamadas.push("verificar"); return Promise.resolve(iaAtual); },
    selecionarModelo: (id: string) => { chamadas.push("selecionar:" + id); iaAtual = { ...iaAtual, modeloSelecionado: id }; return Promise.resolve(iaAtual); },
    testarModelo: (id: string) => {
      chamadas.push("testar:" + id);
      iaAtual = { ...iaAtual, modelos: iaAtual.modelos.map((m: any) => (m.id === id ? { ...m, teste: "ok" } : m)) };
      return Promise.resolve(iaAtual);
    },
    abrirPaginaChaves: () => { chamadas.push("abrirPaginaChaves"); return Promise.resolve(iaAtual); },
    selecionarProvedor: (p: string) => { chamadas.push("provedor:" + p); iaAtual = { ...iaAtual, provedor: p }; return Promise.resolve(iaAtual); },
    entrarChatGPT: () => {
      chamadas.push("entrarChatGPT");
      iaAtual = { ...iaAtual, chatgpt: { ...iaAtual.chatgpt, conectada: true, planoAutorizado: true, email: "fulano@exemplo.invalid",
        modelos: [{ id: "gpt-a", nome: "GPT A", contexto: 500000, raciocinio: ["low", "high"], raciocinioPadrao: "low", teste: "", testeErro: null }] } };
      return Promise.resolve(iaAtual);
    },
    cancelarEntradaChatGPT: () => { chamadas.push("cancelarEntrada"); return Promise.resolve(iaAtual); },
    desconectarChatGPT: () => { chamadas.push("desconectar"); iaAtual = { ...iaAtual, chatgpt: { ...iaBase.chatgpt } }; return Promise.resolve(iaAtual); },
    atualizarModelosChatGPT: () => { chamadas.push("atualizarModelosChatGPT"); return Promise.resolve(iaAtual); },
    selecionarModeloChatGPT: (id: string) => {
      chamadas.push("chatgpt-modelo:" + id);
      iaAtual = { ...iaAtual, provedor: "CHATGPT", chatgpt: { ...iaAtual.chatgpt, modeloSelecionado: id } };
      return Promise.resolve(iaAtual);
    },
    testarModeloChatGPT: (id: string) => { chamadas.push("chatgpt-testar:" + id); return Promise.resolve(iaAtual); },
    abrirUsoChatGPT: () => { chamadas.push("abrirUso"); return Promise.resolve(iaAtual); },
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
  cadastro: {
    salvar: (c: any) => {
      chamadas.push("salvar:" + JSON.stringify({ pessoas: c.pessoas, acusadoId: c.acusadoId, fatos: c.fatos }));
      return resp({ ...visaoAtual, cadastro: salvarSimulado(c) });
    },
    anexar: (id: string) => {
      chamadas.push("anexar:" + id);
      const cad = visaoAtual.cadastro;
      const pecas = cad.pecas.map((p: any) => (p.id === id ? { ...p, arquivos: [...p.arquivos, { id: "ARQ-90", nome: "memorando.pdf", paginas: 3, hash: "abc", origem: "UPLOAD" }] } : p));
      return resp({ ...visaoAtual, cadastro: { ...cad, pecas }, pendenciasCadastro: [] });
    },
    anexarSEI: (id: string, docs: string[]) => {
      chamadas.push(`anexarSEI:${id}:${docs.join(",")}`);
      const cad = visaoAtual.cadastro;
      const documentosSei = visaoAtual.documentosSei.map((d: any) => (docs.includes(d.id) ? { ...d, usado: id } : d));
      const pecas = cad.pecas.map((p: any) => (p.id === id ? { ...p, arquivos: [...p.arquivos, { id: "ARQ-91", nome: "SEI 0010000 – Memorando 101", paginas: 1, hash: "def", origem: "SEI" }] } : p));
      return resp({ ...visaoAtual, cadastro: { ...cad, pecas }, documentosSei });
    },
    remover: (p: string, a: string) => { chamadas.push(`remover:${p}:${a}`); return resp(visaoAtual); },
    mover: (p: string, a: string, d: number) => { chamadas.push(`mover:${p}:${a}:${d}`); return resp(visaoAtual); },
    concluir: () => { chamadas.push("concluir"); return resp(visao("VERIFICANDO")); },
    catalogo: () => Promise.resolve([
      { id: "RDPM-17-V", diploma: "RDPM", rotulo: "art. 17, V — utilizar viatura sem autorização", natureza: "GRAVE" },
      { id: "EST-29-I", diploma: "ESTATUTO", rotulo: "art. 29, I — amar a verdade", natureza: "" },
    ]),
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
  ganchos.antesDeResponder = null;
  chamadas.length = 0;
  visaoAtual = null;
  iaAtual = iaBase;
  estado.visao = null;
  estado.erro = null;
  estado.ocupado = false;
  estado.ia = null;
  estado.verConfigIA = false;
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
    expect(chamadas).toEqual(["atual", "ia", "iniciar", "tipo:SR"]);
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

describe("T02 — cadastro de peças sem credencial", () => {
  it("cadastrar peças escolhe UPLOAD e abre o assistente, sem campo de senha", async () => {
    visaoAtual = visao("TIPO_ESCOLHIDO");
    const w = mount(App);
    await flushPromises();
    expect(w.find('input[type="password"]').exists()).toBe(false);
    await w.find('[data-teste="entrada-UPLOAD"]').trigger("click");
    await flushPromises();
    expect(chamadas).toEqual(["atual", "ia", "entrada:UPLOAD"]);
    expect(w.find('[data-teste="montagem"]').exists()).toBe(true);
    expect(w.find('input[type="password"]').exists()).toBe(false);
  });
});

describe("Cadastro de peças (V4 §48.4)", () => {
  it("SEI: documentos separados do PDF são escolhidos por peça; a sugestão vem primeiro", async () => {
    const doc = (id: string, sei: string, titulo: string, sugestao: string) => ({ id, sei, titulo, paginas: 1, sugestao, papel: "", usado: "" });
    visaoAtual = visao("RECEBENDO_UPLOAD", { entrada: "SEI", cadastro: cadastroPADS(), pendenciasCadastro: [],
      documentosSei: [doc("DOC-01", "0010001", "Despacho 5", "OUTRO"), doc("DOC-02", "0010000", "Memorando 101", "MEMORANDO")] });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="aviso-sei"]').text()).toContain("2 documento(s)");
    const sel = w.find('[data-teste="arq-memorando-sei"]');
    const opcoes = sel.findAll("option").map((o) => o.element.value);
    expect(opcoes).toEqual(["", "DOC-02", "DOC-01"]); // o Memorando sugerido primeiro
    await sel.setValue("DOC-02");
    await w.find('[data-teste="arq-memorando-anexar-sei"]').trigger("click");
    await flushPromises();
    expect(chamadas.at(-1)).toBe("anexarSEI:PEC-01:DOC-02");
    expect(w.find('[data-teste="arq-memorando-arquivo"]').text()).toContain("Memorando 101");
    // O documento escolhido sai das listas; o outro continua disponível.
    expect(w.find('[data-teste="arq-memorando-sei"]').findAll("option").map((o) => o.element.value)).toEqual(["", "DOC-01"]);
  });

  it("grava acusado e enquadramento antes de anexar; o Estatuto exige a natureza", async () => {
    visaoAtual = visao("RECEBENDO_UPLOAD", { entrada: "UPLOAD", cadastro: cadastroPADS(), pendenciasCadastro: ["Anexe o Memorando Disciplinar."] });
    const w = mount(App);
    await flushPromises();
    await w.find('[data-teste="acusado-posto"]').setValue("CB PM");
    await w.find('[data-teste="acusado-nome"]').setValue("Carlos Inventado");
    await w.find('[data-teste="enq-0-mais"]').trigger("click");
    await w.find('[data-teste="enq-0-rdpm-0"]').trigger("change");
    await w.find('[data-teste="enq-0-disp-0-filtro"]').setValue("viatura");
    await w.find('[data-teste="enq-0-disp-0-opcao-RDPM-17-V"]').trigger("click");
    expect(w.find('[data-teste="enq-0-nat-0"]').exists()).toBe(false); // RDPM: natureza do artigo
    await w.find('[data-teste="enq-0-mais"]').trigger("click");
    await w.find('[data-teste="enq-0-est-1"]').trigger("change");
    await w.find('[data-teste="enq-0-disp-1-filtro"]').setValue("verdade");
    await w.find('[data-teste="enq-0-disp-1-opcao-EST-29-I"]').trigger("click");
    expect(w.find('[data-teste="enq-0-nat-1"]').exists()).toBe(true);
    await w.find('[data-teste="arq-memorando-anexar"]').trigger("click");
    await flushPromises();
    const salvo = JSON.parse(chamadas.find((c) => c.startsWith("salvar:"))!.slice(7));
    expect(salvo.pessoas).toEqual([{ id: salvo.acusadoId, nome: "Carlos Inventado", postoGrad: "CB PM", crianca: false, grafias: [] }]);
    // Sem a natureza, o dispositivo do Estatuto não vai ao Go (fica na tela).
    expect(salvo.fatos[0].enquadramentos).toEqual([{ dispositivoId: "RDPM-17-V", natureza: "" }]);
    expect(chamadas.at(-1)).toBe("anexar:PEC-01");
    expect(w.find('[data-teste="arq-memorando-arquivo"]').text()).toContain("memorando.pdf");
    expect(w.find('[data-teste="enq-0-nat-1"]').exists()).toBe(true); // linha incompleta preservada
    expect(w.find('[data-teste="acusado-nome"]').element as HTMLInputElement).toHaveProperty("value", "Carlos Inventado");
  });

  it("depoimentos, conferência com pendências e conclusão", async () => {
    visaoAtual = visao("RECEBENDO_UPLOAD", { entrada: "UPLOAD", cadastro: cadastroPADS(), pendenciasCadastro: ["Informe o acusado."] });
    const w = mount(App);
    await flushPromises();
    await w.find('[data-teste="passo-depoimentos"]').trigger("click");
    await flushPromises();
    await w.find('[data-teste="novo-depoimento"]').trigger("click");
    await w.find('[data-teste="depoente-nome-0"]').setValue("Diego Suposto");
    await w.find('[data-teste="papel-0-DECLARANTE"]').setValue(true);
    await w.find('[data-teste="arq-dep-0-anexar"]').trigger("click");
    await flushPromises();
    // A peça nova ganha ID no Go antes de receber o arquivo.
    expect(chamadas.at(-1)).toMatch(/^anexar:PEC-\d+$/);
    await w.find('[data-teste="passo-conferencia"]').trigger("click");
    await flushPromises();
    expect(w.find('[data-teste="conferencia-pecas"]').text()).toContain("Depoimento (declarante): Diego Suposto");
    expect(w.findAll('[data-teste="pessoa"]')).toHaveLength(1);
    await w.find('[data-teste="concluir-cadastro"]').trigger("click");
    await flushPromises();
    expect(chamadas.at(-1)).toBe("concluir");
  });

  it("depoente: nome direto; reaproveitar pessoa só quando já há outra cadastrada", async () => {
    visaoAtual = visao("RECEBENDO_UPLOAD", { entrada: "UPLOAD", cadastro: cadastroPADS() });
    const w = mount(App);
    await flushPromises();
    await w.find('[data-teste="passo-depoimentos"]').trigger("click");
    await flushPromises();
    await w.find('[data-teste="novo-depoimento"]').trigger("click");
    expect(w.find('[data-teste="depoente-0"]').exists()).toBe(false); // ninguém para reaproveitar
    await w.find('[data-teste="depoente-posto-0"]').setValue("Sd PM");
    await w.find('[data-teste="depoente-nome-0"]').setValue("Diego Suposto");
    await w.find('[data-teste="novo-depoimento"]').trigger("click");
    const sel = w.find('[data-teste="depoente-1"]');
    expect(sel.text()).toContain("Sim: Sd PM Diego Suposto");
    expect(sel.text()).not.toContain("sem nome");
    const opt = sel.findAll("option")[1].element as HTMLOptionElement;
    await sel.setValue(opt.value);
    expect(w.find('[data-teste="depoente-reuso-1"]').exists()).toBe(true);
    expect(w.find('[data-teste="depoente-nome-1"]').element as HTMLInputElement).toHaveProperty("value", "Diego Suposto");
    await sel.setValue("");
    expect(w.find('[data-teste="depoente-nome-1"]').element as HTMLInputElement).toHaveProperty("value", "");
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
    visaoAtual = visao("RECEBENDO_UPLOAD", { cadastro: cadastroPADS(), erro: { codigo: "PDF-004", titulo: "PDF protegido por senha", mensagem: "m", acao: "a" } });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="montagem"]').exists()).toBe(true);
    expect(w.find('[data-teste="erro-cadastro"]').text()).toContain("PDF protegido por senha");
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
  it("parada recuperável (IA-007): tentar novamente e abrir configurações de IA", async () => {
    visaoAtual = visao("DELIMITANDO", { parada: true, retomavel: true, erro: { codigo: "IA-007", titulo: "Sem conexão com a Gemini API", mensagem: "m", acao: "a" } });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="parada"]').exists()).toBe(false);
    await w.find('[data-teste="abrir-config-ia"]').trigger("click");
    await flushPromises();
    expect(w.find('[data-teste="tela-config-ia"]').exists()).toBe(true);
    await w.find('[data-teste="fechar-config-ia"]').trigger("click");
    await flushPromises();
    await w.find('[data-teste="retomar"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("retomar");
  });

  it("parada com erro não retomável mostra o erro, não o aviso de fim de fase", async () => {
    visaoAtual = visao("ANALISANDO", { parada: true, retomavel: false, erro: { codigo: "IA-005", titulo: "Limite de uso da Gemini API atingido", mensagem: "m", acao: "a" } });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="parada"]').exists()).toBe(false);
    expect(w.find('[data-teste="parada-erro"]').text()).toContain("Limite de uso");
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

// T27 (emenda V4 §47): primeiro uso = cadastrar a chave da Gemini API e escolher o modelo.
describe("T27 — configuração da Gemini API", () => {
  it("primeiro uso leva às configurações de IA; a chave some do campo e nunca é exibida", async () => {
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="primeiro-uso"]').text()).toContain("Gemini API");
    await w.find('[data-teste="configurar-ia"]').trigger("click");
    await flushPromises();
    await w.find('[data-teste="aba-gemini"]').trigger("click");
    const tela = w.find('[data-teste="tela-config-ia"]');
    expect(tela.find('[data-teste="situacao-chave"]').text()).toContain("Nenhuma chave");
    const campo = w.find('[data-teste="campo-chave"]');
    expect(campo.attributes("type")).toBe("password");
    expect(campo.attributes("autocomplete")).toBe("off");
    const chave = "AIzaSyCHAVE_DE_TESTE_SENTINELA_123456";
    await campo.setValue(chave);
    await w.find("form").trigger("submit");
    await flushPromises();
    expect(chamadas).toContain("salvarChave:" + chave.length);
    expect((w.find('[data-teste="campo-chave"]').element as HTMLInputElement).value).toBe("");
    expect(w.html()).not.toContain("SENTINELA");
    expect(w.find('[data-teste="situacao-chave"]').text()).toContain("aceita pelo Google");
    expect(w.find('[data-teste="modelo-gemini-3.8-flash-preview"]').text()).toContain("Prévia");
    await w.find('[data-teste="usar-gemini-3.6-flash"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("selecionar:gemini-3.6-flash");
    expect(w.find('[data-teste="modelo-gemini-3.6-flash"]').text()).toContain("Em uso");
    await w.find('[data-teste="testar-gemini-3.6-flash"]').trigger("click");
    await flushPromises();
    expect(w.find('[data-teste="teste-ok-gemini-3.6-flash"]').exists()).toBe(true);
    await w.find('[data-teste="obter-chave"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("abrirPaginaChaves");
  });

  it("remover a chave exige confirmação", async () => {
    iaAtual = { ...iaBase, chaveCadastrada: true, status: "nao_verificada", modeloSelecionado: "gemini-3.6-flash" };
    const w = mount(App);
    await flushPromises();
    await w.find('[data-teste="link-config-ia"]').trigger("click");
    await flushPromises();
    expect(w.find('[data-teste="painel-gemini"]').exists()).toBe(true); // provedor Gemini abre a aba Gemini
    await w.find('[data-teste="remover-chave"]').trigger("click");
    expect(chamadas).not.toContain("removerChave");
    await w.find('[data-teste="confirmar-remover-chave"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("removerChave");
  });

  it("com chave e modelo configurados não há aviso de primeiro uso", async () => {
    iaAtual = { ...iaBase, chaveCadastrada: true, status: "nao_verificada", modeloSelecionado: "gemini-3.6-flash" };
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="primeiro-uso"]').exists()).toBe(false);
  });
});

// A resposta de um comando pode chegar depois dos eventos que a etapa emitiu
// em seguida (ex.: revisão da barreira de privacidade, sem rede): a tela não
// pode voltar para a visão mais antiga.
describe("Ordem das visões", () => {
  it("descarta a resposta do comando mais antiga que o evento já aplicado", async () => {
    const { carregar, responder } = await import("./estado");
    visaoAtual = visao("AGUARDANDO_ESCLARECIMENTO", { sequencia: 5, pergunta: { id: "delimitacao", titulo: "t", contexto: "", opcoes: [], campos: [], grupos: [] } });
    await carregar();
    expect(estado.visao?.sequencia).toBe(5);
    const nova = visao("AGUARDANDO_ESCLARECIMENTO", { sequencia: 7, pergunta: { id: "revisao_privacidade_1", titulo: "Revisar", contexto: "", opcoes: [], campos: [], grupos: [] } });
    ganchos.antesDeResponder = () => ganchos.ouvinte?.(nova);
    await responder("delimitacao", "confirmar", {});
    await flushPromises();
    expect(estado.visao?.sequencia).toBe(7);
    expect(estado.visao?.pergunta?.id).toBe("revisao_privacidade_1");
    // visão mais nova vinda depois continua sendo aplicada
    ganchos.ouvinte?.(visao("ANALISANDO", { sequencia: 8 }));
    expect(estado.visao?.estado).toBe("ANALISANDO");
    // outra tarefa (id diferente) sempre substitui
    ganchos.ouvinte?.({ ...visao("NOVA"), id: "t2", sequencia: 1 });
    expect(estado.visao?.id).toBe("t2");
  });
});

// V4 §48.3 (Termos do Sign in with ChatGPT): aviso de privacidade antes da
// conexão; botão só em texto; escolha explícita do serviço.
describe("ChatGPT nas configurações de IA", () => {
  it("o aviso de privacidade precede a conexão; entrar, escolher e testar o modelo", async () => {
    const w = mount(App);
    await flushPromises();
    await w.find('[data-teste="link-config-ia"]').trigger("click");
    await flushPromises();
    const painel = w.find('[data-teste="painel-chatgpt"]');
    expect(painel.exists()).toBe(true); // aba padrão sem serviço escolhido
    expect(painel.find('[data-teste="aviso-privacidade"]').text()).toContain("menor de 13 anos");
    const entrar = w.find('[data-teste="entrar-chatgpt"]');
    expect(entrar.text()).toBe("Entrar com ChatGPT");
    expect(entrar.find("img, svg").exists()).toBe(false);
    expect(entrar.attributes("disabled")).toBeDefined();
    await entrar.trigger("click");
    expect(chamadas).not.toContain("entrarChatGPT");
    await w.find('[data-teste="li-aviso"]').setValue(true);
    await w.find('[data-teste="entrar-chatgpt"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("entrarChatGPT");
    expect(w.find('[data-teste="conta-conectada"]').text()).toContain("fulano@exemplo.invalid");
    await w.find('[data-teste="chatgpt-usar-gpt-a"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("chatgpt-modelo:gpt-a");
    expect(w.find('[data-teste="usando-plano"]').exists()).toBe(true);
    expect(w.find('[data-teste="chatgpt-modelo-gpt-a"]').text()).toContain("Em uso");
    await w.find('[data-teste="chatgpt-testar-gpt-a"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("chatgpt-testar:gpt-a");
    await w.find('[data-teste="desconectar"]').trigger("click");
    expect(chamadas).not.toContain("desconectar");
    await w.find('[data-teste="confirmar-desconectar"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain("desconectar");
    expect(w.find('[data-teste="aviso-privacidade"]').exists()).toBe(true);
  });

  it("o serviço é escolhido explicitamente; ChatGPT pronto dispensa o aviso de primeiro uso", async () => {
    iaAtual = { ...iaBase, provedor: "CHATGPT", chatgpt: { ...iaBase.chatgpt, conectada: true, planoAutorizado: true, modeloSelecionado: "gpt-a" } };
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="primeiro-uso"]').exists()).toBe(false);
    await w.find('[data-teste="link-config-ia"]').trigger("click");
    await flushPromises();
    await w.find('[data-teste="provedor-GEMINI"] input').setValue(true);
    await flushPromises();
    expect(chamadas).toContain("provedor:GEMINI");
  });

  it("conta conectada sem permissão do plano mostra o alerta e oferece entrar de novo", async () => {
    iaAtual = { ...iaBase, provedor: "CHATGPT", chatgpt: { ...iaBase.chatgpt, conectada: true, planoAutorizado: false } };
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="primeiro-uso"]').exists()).toBe(true);
    await w.find('[data-teste="link-config-ia"]').trigger("click");
    await flushPromises();
    expect(w.find('[data-teste="plano-nao-autorizado"]').exists()).toBe(true);
    expect(w.find('[data-teste="chatgpt-sem-lista"]').exists()).toBe(false);
  });
});

// Enquadramento manual (V4 §48.5): diploma → dispositivo (lista com filtro)
// → natureza só no Estatuto; condições encadeadas.
describe("Enquadramento no formulário das imputações", () => {
  const c = (id: string, extra: Record<string, unknown>) => ({ id, rotulo: id, tipo: "TEXTO", opcoes: [], obrigatorio: true, valor: "",
    ajuda: "", grupo: "IMP-1", min: 0, max: 0, fontes: [], mostrarSe: null, ...extra });
  const form = {
    id: "confirmar_delimitacao", titulo: "Confirme", contexto: "c",
    opcoes: [{ id: "confirmar", rotulo: "Confirmar", validar: true }],
    grupos: [{ id: "IMP-1", titulo: "IMP-1", descricao: ["Dispositivo citado no Memorando (para conferência): art. 16, inciso XVI, do RDPM."], alertas: [], fontes: [] }],
    campos: [
      c("incluir_IMP-1", { tipo: "SIM_NAO", valor: "sim" }),
      c("diploma_IMP-1", { tipo: "ESCOLHA", opcoes: [{ id: "RDPM", rotulo: "RDPM" }, { id: "ESTATUTO", rotulo: "Estatuto" }], mostrarSe: { campo: "incluir_IMP-1", valor: "sim" } }),
      c("disp_rdpm_IMP-1", { tipo: "BUSCA", mostrarSe: { campo: "diploma_IMP-1", valor: "RDPM" }, opcoes: [
        { id: "RDPM-16-XLIII", rotulo: "art. 16, XLIII — chegar atrasado à atividade para a qual esteja escalado (média)" },
        { id: "RDPM-17-XXXIX", rotulo: "art. 17, XXXIX — dirigir-se de maneira desrespeitosa ou desatenciosa a subordinado, par ou superior hierárquico (grave)" },
      ] }),
      c("disp_est_IMP-1", { tipo: "BUSCA", mostrarSe: { campo: "diploma_IMP-1", valor: "ESTATUTO" }, opcoes: [
        { id: "EST-32-IV", rotulo: "art. 32, IV — a disciplina e o respeito à hierarquia" },
      ] }),
      c("natureza_IMP-1", { tipo: "ESCOLHA", opcoes: [{ id: "LEVE", rotulo: "Leve" }, { id: "MEDIA", rotulo: "Média" }, { id: "GRAVE", rotulo: "Grave" }], mostrarSe: { campo: "diploma_IMP-1", valor: "ESTATUTO" } }),
    ],
  };

  it("o dispositivo é escolhido pelo usuário, com filtro; natureza só no Estatuto", async () => {
    visaoAtual = visao("AGUARDANDO_ESCLARECIMENTO", { pergunta: form });
    const w = mount(App);
    await flushPromises();
    expect(w.find('[data-teste="campo-disp_rdpm_IMP-1"]').exists()).toBe(false); // sem diploma, sem dispositivo
    expect(w.find('[data-teste="grupo-IMP-1"]').text()).toContain("para conferência");
    await w.find('[data-teste="campo-diploma_IMP-1-RDPM"]').setValue(true);
    await flushPromises();
    expect(w.find('[data-teste="campo-natureza_IMP-1-LEVE"]').exists()).toBe(false); // RDPM: natureza do artigo
    await w.find('[data-teste="campo-disp_rdpm_IMP-1-filtro"]').setValue("desatencios");
    await flushPromises();
    expect(w.findAll('[data-teste^="campo-disp_rdpm_IMP-1-opcao-"]').length).toBe(1);
    await w.find('[data-teste="campo-disp_rdpm_IMP-1-opcao-RDPM-17-XXXIX"]').trigger("click");
    await flushPromises();
    expect(w.find('[data-teste="campo-disp_rdpm_IMP-1-escolhida"]').text()).toContain("art. 17, XXXIX");
    await w.find('[data-teste="opcao-confirmar"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain('responder:confirmar_delimitacao:confirmar:{"incluir_IMP-1":"sim","diploma_IMP-1":"RDPM","disp_rdpm_IMP-1":"RDPM-17-XXXIX"}');
  });

  it("Estatuto exige a natureza; incluir = não esconde toda a cadeia", async () => {
    visaoAtual = visao("AGUARDANDO_ESCLARECIMENTO", { pergunta: form });
    const w = mount(App);
    await flushPromises();
    await w.find('[data-teste="campo-diploma_IMP-1-ESTATUTO"]').setValue(true);
    await flushPromises();
    await w.find('[data-teste="campo-disp_est_IMP-1-filtro"]').setValue("hierarquia");
    await w.find('[data-teste="campo-disp_est_IMP-1-opcao-EST-32-IV"]').trigger("click");
    await w.find('[data-teste="opcao-confirmar"]').trigger("click");
    await flushPromises();
    expect(chamadas.some((x) => x.startsWith("responder"))).toBe(false); // natureza obrigatória
    await w.find('[data-teste="campo-natureza_IMP-1-MEDIA"]').setValue(true);
    await w.find('[data-teste="campo-incluir_IMP-1-nao"]').setValue(true);
    await flushPromises();
    expect(w.find('[data-teste="campo-disp_est_IMP-1"]').exists()).toBe(false); // pai oculto → filho oculto
    expect(w.find('[data-teste="campo-natureza_IMP-1-MEDIA"]').exists()).toBe(false);
    await w.find('[data-teste="opcao-confirmar"]').trigger("click");
    await flushPromises();
    expect(chamadas).toContain('responder:confirmar_delimitacao:confirmar:{"incluir_IMP-1":"nao"}');
  });
});

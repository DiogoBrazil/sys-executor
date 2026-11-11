<script setup lang="ts">
// Assistente de cadastro das peças (V4 §48.4; E6): peça por peça, com vários
// arquivos por peça. PADS: Memorando (acusado, fatos e enquadramentos),
// defesa prévia, depoimentos, alegações finais e outros documentos. SR:
// Portaria (sindicados e ofendidos), depoimentos e outros documentos.
// Nada é persistido aqui: o cadastro vive na sessão da tarefa, no Go.
import { computed, onMounted, ref, watch } from "vue";
import {
  anexarArquivos, anexarDocumentosSEI, catalogoEnquadramento, concluirCadastro, estado, moverArquivo, removerArquivo, salvarCadastro,
} from "../estado";
import type { Cadastro, DispositivoVisao } from "../api";
import type { PecaCadastro } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";
import ArquivosPeca from "../components/ArquivosPeca.vue";
import EditorEnquadramento, { type LinhaEnquadramento } from "../components/EditorEnquadramento.vue";
import AvisoErro from "../components/AvisoErro.vue";
import Icone from "../components/ui/Icone.vue";
import Janela from "../components/ui/Janela.vue";

interface PessoaLocal { id: string; nome: string; postoGrad: string; grafias: string; crianca: boolean }
interface FatoLocal { id: string; resumo: string; data: string; linhas: LinhaEnquadramento[] }
interface Local {
  tipo: string; pessoas: PessoaLocal[]; acusadoId: string; fatos: FatoLocal[];
  portaria: string; sindicados: string[]; ofendidos: string[]; semOfendido: boolean; pecas: PecaCadastro[];
}

const catalogo = ref<DispositivoVisao[]>([]);
onMounted(async () => { catalogo.value = await catalogoEnquadramento(); });

let provisorio = 0;
const novoId = () => `NOVO-${++provisorio}`;
const diplomaDe = (id: string) => (id.startsWith("RDPM-") ? "RDPM" : id.startsWith("EST-") ? "ESTATUTO" : "");

function doServidor(c: Cadastro): Local {
  const l: Local = {
    tipo: c.tipo,
    pessoas: (c.pessoas ?? []).map((p) => ({ id: p.id, nome: p.nome, postoGrad: p.postoGrad, grafias: (p.grafias ?? []).join("; "), crianca: p.crianca })),
    acusadoId: c.acusadoId,
    fatos: (c.fatos ?? []).map((f) => ({
      id: f.id, resumo: f.resumo, data: f.data,
      linhas: (f.enquadramentos ?? []).map((e) => ({ diploma: diplomaDe(e.dispositivoId), dispositivoId: e.dispositivoId, natureza: e.natureza })),
    })),
    portaria: c.portaria, sindicados: [...(c.sindicados ?? [])], ofendidos: [...(c.ofendidos ?? [])], semOfendido: c.semOfendido,
    pecas: (c.pecas ?? []).map((p) => ({ ...p, pessoaIds: [...(p.pessoaIds ?? [])], arquivos: [...(p.arquivos ?? [])] })),
  };
  if (l.tipo === "PADS" && !l.acusadoId) {
    const id = novoId();
    l.pessoas.unshift({ id, nome: "", postoGrad: "", grafias: "", crianca: false });
    l.acusadoId = id;
  }
  if (l.tipo === "PADS" && l.fatos.length === 0) l.fatos.push({ id: novoId(), resumo: "", data: "", linhas: [] });
  // Depoimento sem pessoa (o nome ainda não tinha sido digitado quando o
  // cadastro foi salvo): volta com uma pessoa em branco, para os campos.
  for (const p of l.pecas) {
    if (p.tipo === "DEPOIMENTO" && p.pessoaIds.length === 0) {
      const id = novoId();
      l.pessoas.push({ id, nome: "", postoGrad: "", grafias: "", crianca: false });
      p.pessoaIds = [id];
    }
  }
  return l;
}

const local = ref<Local>(doServidor(estado.visao!.cadastro!));
let sujo = false;
const marcar = () => { sujo = true; };

// A visão nova do Go (depois de salvar ou anexar) é a referência: IDs
// definitivos e arquivos. Linhas de enquadramento ainda incompletas ficam.
watch(() => estado.visao?.cadastro, (c) => {
  if (!c) return;
  const antes = local.value;
  const novo = doServidor(c);
  antes.fatos.forEach((f, i) => {
    const incompletas = f.linhas.filter((x) => !x.dispositivoId || (x.diploma === "ESTATUTO" && !x.natureza));
    if (novo.fatos[i]) novo.fatos[i].linhas.push(...incompletas.filter((x) => !novo.fatos[i].linhas.some((y) => y.dispositivoId && y.dispositivoId === x.dispositivoId)));
  });
  local.value = novo;
  sujo = false;
});

// Para o Go: pessoas sem nome saem (e as referências a elas); linhas de
// enquadramento só completas.
function paraServidor(): Cadastro {
  const l = local.value;
  const validas = new Set(l.pessoas.filter((p) => p.nome.trim()).map((p) => p.id));
  const ref = (id: string) => (validas.has(id) ? id : "");
  return {
    tipo: l.tipo,
    pessoas: l.pessoas.filter((p) => validas.has(p.id)).map((p) => ({
      id: p.id, nome: p.nome, postoGrad: p.postoGrad, crianca: p.crianca,
      grafias: p.grafias.split(/[;\n]/).map((g) => g.trim()).filter(Boolean),
    })),
    acusadoId: ref(l.acusadoId),
    fatos: l.fatos.map((f) => ({
      id: f.id, resumo: f.resumo, data: f.data.trim(),
      enquadramentos: f.linhas.filter((x) => x.dispositivoId && (x.diploma !== "ESTATUTO" || x.natureza))
        .map((x) => ({ dispositivoId: x.dispositivoId, natureza: x.diploma === "ESTATUTO" ? x.natureza : "" })),
    })),
    portaria: l.portaria,
    sindicados: l.sindicados.filter((id) => validas.has(id)),
    ofendidos: l.ofendidos.filter((id) => validas.has(id)),
    semOfendido: l.semOfendido,
    pecas: l.pecas.map((p) => ({ ...p, pessoaIds: p.pessoaIds.filter((id) => validas.has(id)) })),
    concluido: false,
  } as unknown as Cadastro;
}

const paraSalvar = () => (sujo ? paraServidor() : null);
async function salvar() {
  if (sujo) await salvarCadastro(paraServidor());
}

// ---- passos ----
const PASSOS_PADS = [
  { id: "memorando", rotulo: "Memorando e acusado" },
  { id: "defesa", rotulo: "Defesa prévia" },
  { id: "depoimentos", rotulo: "Depoimentos" },
  { id: "alegacoes", rotulo: "Alegações finais" },
  { id: "outros", rotulo: "Outros documentos" },
  { id: "conferencia", rotulo: "Conferência" },
];
const PASSOS_SR = [
  { id: "portaria", rotulo: "Portaria e envolvidos" },
  { id: "depoimentos", rotulo: "Depoimentos" },
  { id: "outros", rotulo: "Outros documentos" },
  { id: "conferencia", rotulo: "Conferência" },
];
const passos = computed(() => (local.value.tipo === "SR" ? PASSOS_SR : PASSOS_PADS));
const passo = ref(0);
const atual = computed(() => passos.value[passo.value].id);
async function irPara(i: number) {
  if (i < 0 || i >= passos.value.length) return;
  await salvar();
  passo.value = i;
}

// ---- peças ----
const pecaDo = (tipo: string) => local.value.pecas.find((p) => p.tipo === tipo);
const doTipo = (tipo: string) => local.value.pecas.filter((p) => p.tipo === tipo);
const pessoa = (id: string) => local.value.pessoas.find((p) => p.id === id);

function novaPeca(tipo: string, extra: Partial<PecaCadastro> = {}) {
  local.value.pecas.push({ id: novoId(), tipo, titulo: "", descricao: "", papel: "", pessoaIds: [], data: "", sei: "", naoApresentada: false, arquivos: [], ...extra } as PecaCadastro);
  marcar();
}
function retirarPeca(p: PecaCadastro) {
  local.value.pecas = local.value.pecas.filter((x) => x !== p);
  marcar();
}
function novaPessoa(): PessoaLocal {
  const p = { id: novoId(), nome: "", postoGrad: "", grafias: "", crianca: false };
  local.value.pessoas.push(p);
  marcar();
  return p;
}
function novoDepoimento() {
  const papel = "TESTEMUNHA";
  novaPeca("DEPOIMENTO", { papel, pessoaIds: [novaPessoa().id] });
}
// Reaproveitar uma pessoa já cadastrada (outra oitiva; na SR, sindicado ou
// ofendido da Portaria). "" volta a ser uma pessoa nova, em branco.
function trocarDepoente(p: PecaCadastro, id: string) {
  p.pessoaIds = id ? [id] : [novaPessoa().id];
  marcar();
}
function compartilhada(id: string, p: PecaCadastro): boolean {
  return local.value.pecas.some((x) => x !== p && x.pessoaIds.includes(id)) ||
    local.value.sindicados.includes(id) || local.value.ofendidos.includes(id);
}
const rotuloPessoa = (ps: PessoaLocal) => [ps.postoGrad, ps.nome].map((x) => x.trim()).filter(Boolean).join(" ");
function opcoesReuso(p: PecaCadastro): PessoaLocal[] {
  const atual = p.pessoaIds[0] ?? "";
  return depoentes.value.filter((ps) => ps.nome.trim() && (ps.id !== atual || compartilhada(atual, p)));
}
// ---- Pessoas repetidas e ofendido institucional (teste da SR, 10/11/2026) ----
const chaveNome = (t: string) => t.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase().replace(/\s+/g, " ").trim();
const reInstitucional = /\b(administracao|estado|fazenda|policia|municipio|prefeitura|secretaria|uniao|governo|ministerio|corregedoria|batalhao)\b/;
const institucional = (ps: PessoaLocal) => reInstitucional.test(chaveNome(ps.nome));
function referencias(id: string): number {
  const l = local.value;
  return l.pecas.filter((x) => x.pessoaIds.includes(id)).length + (l.acusadoId === id ? 1 : 0) +
    (l.sindicados.includes(id) ? 1 : 0) + (l.ofendidos.includes(id) ? 1 : 0);
}
function papelCadastrado(id: string): string {
  const l = local.value;
  if (l.acusadoId === id) return "acusado";
  if (l.sindicados.includes(id)) return "sindicado";
  if (l.ofendidos.includes(id)) return "ofendido";
  return "depoente";
}
// Outra pessoa já cadastrada com o mesmo nome do depoente digitado.
function homonimo(p: PecaCadastro): PessoaLocal | undefined {
  const atual = pessoa(p.pessoaIds[0] ?? "");
  const k = atual ? chaveNome(atual.nome) : "";
  if (!k) return undefined;
  return local.value.pessoas.find((x) => x.id !== atual!.id && chaveNome(x.nome) === k &&
    (local.value.tipo !== "PADS" || x.id !== local.value.acusadoId));
}
function usarMesma(p: PecaCadastro, outra: PessoaLocal) {
  const antiga = p.pessoaIds[0];
  trocarDepoente(p, outra.id);
  if (antiga && referencias(antiga) === 0) local.value.pessoas = local.value.pessoas.filter((x) => x.id !== antiga);
}
// Avisos da conferência (não impedem concluir).
const avisosConferencia = computed(() => {
  const l = local.value;
  const out: string[] = [];
  for (const p of l.pecas) {
    if (p.tipo !== "DEPOIMENTO" || l.tipo !== "SR") continue;
    const ps = pessoa(p.pessoaIds[0] ?? "");
    if (ps?.nome.trim() && l.sindicados.includes(ps.id) && p.papel !== "SINDICADO") {
      out.push(`${rotuloPessoa(ps)} é sindicado, mas o depoimento está como ${rotuloPapel(p.papel).toLowerCase()}. Confira o papel.`);
    }
  }
  const vistos = new Map<string, number>();
  for (const ps of l.pessoas) if (ps.nome.trim()) vistos.set(chaveNome(ps.nome), (vistos.get(chaveNome(ps.nome)) ?? 0) + 1);
  for (const [k, n] of vistos) {
    if (n > 1) {
      const ps = l.pessoas.find((x) => chaveNome(x.nome) === k)!;
      out.push(`${rotuloPessoa(ps)} foi cadastrado ${n} vezes. Se é a mesma pessoa, use "Usar a mesma" no depoimento.`);
    }
  }
  return out;
});
const valorReuso = (p: PecaCadastro) => (p.pessoaIds[0] && compartilhada(p.pessoaIds[0], p) ? p.pessoaIds[0] : "");
function acrescentarEnvolvido(lista: "sindicados" | "ofendidos") {
  local.value[lista].push(novaPessoa().id);
  if (lista === "ofendidos") local.value.semOfendido = false;
}
function retirarEnvolvido(lista: "sindicados" | "ofendidos", id: string) {
  local.value[lista] = local.value[lista].filter((x) => x !== id);
  marcar();
}
function novoFato() {
  local.value.fatos.push({ id: novoId(), resumo: "", data: "", linhas: [] });
  marcar();
}

// Comandos de arquivo: os metadados editados são gravados antes.
async function comPeca(p: PecaCadastro, acao: (c: Cadastro | null, pecaID: string) => Promise<unknown>) {
  if (!p.id.startsWith("NOVO-")) {
    await acao(paraSalvar(), p.id);
    return;
  }
  // Peça nova, ainda sem ID definitivo: grava e anexa à peça na mesma
  // posição (o Go mantém a ordem das peças).
  const i = local.value.pecas.indexOf(p);
  if (!(await salvarCadastro(paraServidor()))) return;
  const real = estado.visao?.cadastro?.pecas?.[i];
  if (real) await acao(null, real.id);
}
// Loader do anexo (pedido de 11/11/2026): da escolha dos arquivos até a
// cópia e a contagem das páginas.
const anexando = ref("");
async function comLoader(p: PecaCadastro, f: () => Promise<unknown>) {
  anexando.value = p.id;
  try {
    await f();
  } finally {
    anexando.value = "";
  }
}
const anexar = (p: PecaCadastro) => comLoader(p, () => comPeca(p, anexarArquivos));
const anexarSei = (p: PecaCadastro, doc: string) => comLoader(p, () => comPeca(p, (c, id) => anexarDocumentosSEI(c, id, [doc])));
// Documentos separados do PDF do SEI (V4 §48.4, revisão de 10/11/2026).
const documentosSei = computed(() => estado.visao?.documentosSei ?? []);
const seiNaoEscolhidos = computed(() => documentosSei.value.filter((d) => !d.usado).length);
const remover = (p: PecaCadastro, a: string) => removerArquivo(paraSalvar(), p.id, a);
const mover = (p: PecaCadastro, a: string, d: number) => moverArquivo(paraSalvar(), p.id, a, d);
const concluir = () => concluirCadastro(paraSalvar());

const pendencias = computed(() => estado.visao?.pendenciasCadastro ?? []);
// No PADS o acusado não é testemunha nem declarante: o interrogatório dele
// entra em "Outros documentos" (decisão pendente do plano D-11, §8.1).
const depoentes = computed(() => local.value.pessoas.filter((p) => local.value.tipo !== "PADS" || p.id !== local.value.acusadoId));
const PAPEIS_PADS = [{ id: "TESTEMUNHA", rotulo: "Testemunha" }, { id: "DECLARANTE", rotulo: "Declarante" }];
const PAPEIS_SR = [...PAPEIS_PADS, { id: "SINDICADO", rotulo: "Sindicado" }];
const papeis = computed(() => (local.value.tipo === "SR" ? PAPEIS_SR : PAPEIS_PADS));
const rotuloPapel = (id: string) => PAPEIS_SR.find((p) => p.id === id)?.rotulo ?? "";
// Situação de cada passo, mostrada no seletor de passos.
const nArq = (p?: PecaCadastro) => p?.arquivos?.length ?? 0;
const situacaoPassos = computed<Record<string, { ok: boolean; info: string; alerta?: boolean }>>(() => {
  const l = local.value;
  const ac = pessoa(l.acusadoId);
  const enq = l.fatos.some((f) => f.linhas.some((x) => x.dispositivoId));
  const peca = (t: string) => pecaDo(t);
  const apresentada = (t: string) => {
    const p = peca(t);
    return { ok: !!p && (p.naoApresentada || nArq(p) > 0), info: p?.naoApresentada ? "não apresentada" : nArq(p) ? `${nArq(p)} arq.` : "" };
  };
  const dep = doTipo("DEPOIMENTO").length;
  const out = doTipo("OUTRO").length;
  return {
    memorando: { ok: nArq(peca("MEMORANDO")) > 0 && !!ac?.nome.trim() && enq, info: nArq(peca("MEMORANDO")) ? `${nArq(peca("MEMORANDO"))} arq.` : "" },
    portaria: { ok: nArq(peca("PORTARIA")) > 0 && l.sindicados.length > 0, info: nArq(peca("PORTARIA")) ? `${nArq(peca("PORTARIA"))} arq.` : "" },
    defesa: apresentada("DEFESA_PREVIA"),
    alegacoes: apresentada("ALEGACOES_FINAIS"),
    depoimentos: { ok: dep > 0, info: dep ? String(dep) : "" },
    outros: { ok: out > 0, info: out ? String(out) : "" },
    conferencia: { ok: pendencias.value.length === 0, info: pendencias.value.length ? `${pendencias.value.length} pend.` : "", alerta: pendencias.value.length > 0 },
  };
});

// ---- cartões das peças e modal de cada uma (protótipo aprovado, 11/11/2026) ----
const modalAberto = ref(false);
const pecasDosPassos = computed(() => passos.value.filter((p) => p.id !== "conferencia"));
const opcional = (id: string) => id === "outros";
const obrigatorios = computed(() => pecasDosPassos.value.filter((p) => !opcional(p.id)));
const concluidos = computed(() => obrigatorios.value.filter((p) => situacaoPassos.value[p.id]?.ok).length);
const proximo = computed(() => obrigatorios.value.find((p) => !situacaoPassos.value[p.id]?.ok)?.id ?? "");
const faltam = computed(() => obrigatorios.value.filter((p) => !situacaoPassos.value[p.id]?.ok).map((p) => p.rotulo.toLowerCase()));
const ICONES: Record<string, "pdf" | "documento" | "pessoa" | "pasta"> = {
  memorando: "pdf", portaria: "pdf", defesa: "documento", alegacoes: "documento", depoimentos: "pessoa", outros: "pasta",
};
const plural = (n: number, s: string, p: string) => `${n} ${n === 1 ? s : p}`;
function infoCartao(id: string): string {
  const l = local.value;
  switch (id) {
    case "memorando": {
      const ac = pessoa(l.acusadoId);
      const imps = l.fatos.reduce((n, f) => n + f.linhas.filter((x) => x.dispositivoId).length, 0);
      const partes = [nArq(pecaDo("MEMORANDO")) ? plural(nArq(pecaDo("MEMORANDO")), "PDF", "PDFs") : "Nenhum PDF"];
      if (ac?.nome.trim()) partes.push(rotuloPessoa(ac));
      partes.push(plural(imps, "imputação", "imputações"));
      return partes.join(" · ");
    }
    case "portaria":
      return `${nArq(pecaDo("PORTARIA")) ? plural(nArq(pecaDo("PORTARIA")), "PDF", "PDFs") : "Nenhum PDF"} · ${plural(l.sindicados.length, "sindicado", "sindicados")}`;
    case "defesa":
    case "alegacoes": {
      const p = pecaDo(id === "defesa" ? "DEFESA_PREVIA" : "ALEGACOES_FINAIS");
      if (p?.naoApresentada) return id === "defesa" ? "Não apresentada" : "Não apresentadas";
      return nArq(p) ? plural(nArq(p), "PDF", "PDFs") : "Anexe o PDF ou marque como não apresentada.";
    }
    case "depoimentos": {
      const ds = doTipo("DEPOIMENTO");
      return ds.length ? `${plural(ds.length, "oitiva", "oitivas")} · ${plural(ds.reduce((n, p) => n + nArq(p), 0), "PDF", "PDFs")}` : "Nenhum depoimento cadastrado.";
    }
    case "outros": {
      const n = doTipo("OUTRO").length;
      return n ? plural(n, "documento", "documentos") : l.tipo === "PADS" ? "Ofícios, certidões, interrogatório do acusado…" : "Ofícios, certidões, relatórios…";
    }
  }
  return "";
}
async function abrirPasso(i: number) {
  await irPara(i);
  modalAberto.value = passos.value[i]?.id !== "conferencia";
}
async function continuar() {
  await irPara(passo.value + 1);
  if (atual.value === "conferencia") modalAberto.value = false;
}
async function fecharModal() {
  await salvar();
  modalAberto.value = false;
}
async function voltarPecas() {
  await salvar();
  modalAberto.value = false;
  passo.value = 0;
}
const PASSO_DO_TIPO: Record<string, string> = {
  MEMORANDO: "memorando", PORTARIA: "portaria", DEFESA_PREVIA: "defesa", ALEGACOES_FINAIS: "alegacoes", DEPOIMENTO: "depoimentos", OUTRO: "outros",
};
const editarPeca = (p: PecaCadastro) => abrirPasso(passos.value.findIndex((x) => x.id === PASSO_DO_TIPO[p.tipo]));
const tituloCadastro = computed(() => (local.value.tipo === "SR" ? "Cadastre as peças do procedimento" : "Cadastre as peças do processo"));
const totalPaginasAutos = computed(() => local.value.pecas.reduce((n, p) => n + totalPaginas(p), 0));
const totalImputacoes = computed(() => local.value.fatos.reduce((n, f) => n + f.linhas.filter((x) => x.dispositivoId).length, 0));
const envolvidos = computed(() => local.value.pessoas.filter((x) => x.nome.trim()));
function papelDe(ps: PessoaLocal): string {
  const l = local.value;
  if (l.acusadoId === ps.id) return "acusado";
  if (l.sindicados.includes(ps.id)) return "sindicado";
  if (l.ofendidos.includes(ps.id)) return "ofendido";
  const dep = l.pecas.find((p) => p.tipo === "DEPOIMENTO" && p.pessoaIds.includes(ps.id));
  return dep ? rotuloPapel(dep.papel).toLowerCase() || "depoente" : "";
}

// Textos de ajuda longos: abrem num modal ("Como funciona?").
const ajuda = ref<"" | "geral" | "fatos" | "depoimentos" | "envolvidos" | "ofendidos">("");
const TITULOS_AJUDA: Record<string, string> = {
  geral: "Como funciona o cadastro das peças", fatos: "Fatos e enquadramento", depoimentos: "Depoimentos",
  envolvidos: "Envolvidos e privacidade", ofendidos: "Ofendidos",
};

const totalPaginas = (p: PecaCadastro) => (p.arquivos ?? []).reduce((s, a) => s + a.paginas, 0);
const rotuloTipo: Record<string, string> = {
  MEMORANDO: "Memorando Disciplinar", PORTARIA: "Portaria de instauração", DEFESA_PREVIA: "Defesa prévia",
  ALEGACOES_FINAIS: "Alegações finais", DEPOIMENTO: "Depoimento", OUTRO: "Outro documento",
};
function descricaoPeca(p: PecaCadastro): string {
  if (p.tipo === "DEPOIMENTO") {
    const quem = p.pessoaIds.map((id) => pessoa(id)?.nome).filter(Boolean).join(", ") || "depoente não informado";
    return `Depoimento (${rotuloPapel(p.papel).toLowerCase()}): ${quem}`;
  }
  if (p.tipo === "OUTRO") return p.titulo || "Outro documento (sem nome)";
  return rotuloTipo[p.tipo] ?? p.tipo;
}
</script>

<template>
  <section class="montagem" data-teste="montagem">
    <!-- Cartões das peças -->
    <template v-if="atual !== 'conferencia'">
      <header class="cab">
        <div class="titulo">
          <h2>{{ tituloCadastro }}</h2>
          <p>Clique em cada peça para anexar os PDFs e preencher os dados. O tipo que você indica é o que vale.</p>
        </div>
        <button type="button" class="botao ajuda-botao" data-teste="ajuda-cadastro" @click="ajuda = 'geral'">
          <Icone nome="ajuda" :tamanho="16" />Como funciona?
        </button>
      </header>
      <div v-if="documentosSei.length" class="callout info" data-teste="aviso-sei">
        <Icone nome="info" class="callout-icone" />
        <p>
          O PDF do processo foi baixado do SEI e separado em {{ documentosSei.length }} documento(s). Em cada peça, use
          "Documento do SEI" para escolher os que a compõem (o tipo sugerido vem do título e deve ser conferido). Os documentos não
          escolhidos não entram nos autos, não são enviados à IA e são apagados ao final.
        </p>
      </div>

      <div class="progresso" aria-label="Peças concluídas">
        <div class="trilho"><span :style="{ width: (obrigatorios.length ? (concluidos / obrigatorios.length) * 100 : 0) + '%' }"></span></div>
        <span><strong>{{ concluidos }} de {{ obrigatorios.length }}</strong> peças concluídas</span>
      </div>

      <div class="cartoes">
        <button v-for="(p, i) in pecasDosPassos" :key="p.id" type="button" class="cartao-peca"
          :class="{ feito: situacaoPassos[p.id]?.ok, proxima: proximo === p.id, opcional: opcional(p.id) && !situacaoPassos[p.id]?.ok }"
          :disabled="estado.ocupado" :data-teste="'passo-' + p.id" @click="abrirPasso(i)">
          <span v-if="situacaoPassos[p.id]?.ok" class="marca-ok" aria-label="Concluída"><Icone nome="check" :tamanho="18" /></span>
          <span v-else-if="proximo === p.id" class="etiqueta pendente">Pendente</span>
          <span v-else-if="opcional(p.id)" class="etiqueta">Opcional</span>
          <span class="ic-peca"><Icone :nome="ICONES[p.id] ?? 'documento'" :tamanho="24" /></span>
          <strong class="nome-peca">{{ p.rotulo }}</strong>
          <span class="info-peca">{{ infoCartao(p.id) }}</span>
          <span class="acao-peca" :class="{ forte: proximo === p.id }">{{ situacaoPassos[p.id]?.ok ? "Editar" : "Abrir" }}</span>
        </button>
        <div class="cartao-peca conferir" :class="{ pronto: !faltam.length }">
          <span class="ic-peca ok"><Icone nome="check" :tamanho="24" /></span>
          <strong class="nome-peca">Conferir e processar</strong>
          <span class="info-peca" :class="{ falta: faltam.length }">
            {{ faltam.length ? `Falta: ${faltam.join(", ")}.` : "Tudo pronto para a conferência." }}
          </span>
          <button type="button" class="botao" :class="{ primario: !faltam.length }" :disabled="estado.ocupado" data-teste="passo-conferencia"
            @click="abrirPasso(passos.length - 1)">Ir para a conferência</button>
        </div>
      </div>
    </template>

    <!-- Conferência -->
    <template v-else>
      <div class="conf-cab">
        <button type="button" class="botao discreto voltar" :disabled="estado.ocupado" data-teste="voltar-pecas" @click="voltarPecas">
          <Icone nome="chevron" :tamanho="16" class="volta" />Voltar às peças
        </button>
        <h2>Confira antes de processar</h2>
        <p>Revise as peças e os envolvidos. Os nomes e as outras grafias são trocados por marcadores antes de qualquer envio à IA.</p>
      </div>
      <div class="conf">
        <div class="coluna">
          <section class="bloco">
            <h3>Peças dos autos</h3>
            <div class="rolagem">
              <table class="tabela">
                <thead><tr><th>Peça</th><th>Arquivos</th><th>Páginas</th><th>Situação</th><th></th></tr></thead>
                <tbody data-teste="conferencia-pecas">
                  <tr v-for="p in local.pecas" :key="p.id">
                    <td><strong>{{ descricaoPeca(p) }}</strong></td>
                    <td>{{ p.naoApresentada ? "—" : p.arquivos.length }}</td>
                    <td>{{ p.naoApresentada ? "—" : totalPaginas(p) }}</td>
                    <td :class="p.naoApresentada ? 'suave' : p.arquivos.length ? 'ok' : 'falta'">
                      {{ p.naoApresentada ? "não apresentada" : p.arquivos.length ? "✓ com PDF" : "sem PDF" }}
                    </td>
                    <td class="dir"><button type="button" class="botao discreto" :disabled="estado.ocupado" @click="editarPeca(p)">Editar</button></td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p v-if="documentosSei.length" class="ajuda" data-teste="conferencia-sei">
              Documentos do SEI: {{ documentosSei.length - seiNaoEscolhidos }} escolhido(s) e {{ seiNaoEscolhidos }} não escolhido(s), que não
              entram nos autos.
            </p>
          </section>
          <section class="bloco">
            <div class="cab-bloco">
              <h3>Envolvidos</h3>
              <button type="button" class="botao ajuda-botao" @click="ajuda = 'envolvidos'"><Icone nome="ajuda" :tamanho="16" />Como funciona?</button>
            </div>
            <div class="rolagem">
              <table class="tabela">
                <thead><tr><th>Pessoa</th><th>Papel</th><th class="largo">Outras grafias</th><th>Menor de 13 anos</th></tr></thead>
                <tbody>
                  <tr v-for="p in envolvidos" :key="p.id" data-teste="pessoa">
                    <td><strong>{{ rotuloPessoa(p) }}</strong></td>
                    <td class="suave">{{ papelDe(p) }}</td>
                    <td><input v-model="p.grafias" :aria-label="'Outras grafias de ' + p.nome" placeholder="Ex.: Cabo Inventado; Inventado" @input="marcar" /></td>
                    <td>
                      <label v-if="!institucional(p)" class="marca"><input v-model="p.crianca" type="checkbox" data-teste="crianca" @change="marcar" /> Sim</label>
                      <span v-else class="selo alerta">Instituição?</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
        </div>
        <aside class="pronto" :class="{ pendente: pendencias.length }" aria-label="Antes de processar">
          <span class="ic-pronto"><Icone :nome="pendencias.length ? 'alerta' : 'check'" :tamanho="26" /></span>
          <h3>{{ pendencias.length ? "Antes de processar" : "O cadastro está completo" }}</h3>
          <ul v-if="pendencias.length" class="pendencias" data-teste="pendencias">
            <li v-for="p in pendencias" :key="p">{{ p }}</li>
          </ul>
          <ul v-if="avisosConferencia.length" class="avisos-conf" data-teste="avisos-conferencia">
            <li v-for="a in avisosConferencia" :key="a">{{ a }}</li>
          </ul>
          <dl class="numeros">
            <dt>Peças</dt><dd>{{ local.pecas.length }}</dd>
            <dt>Páginas</dt><dd>{{ totalPaginasAutos }}</dd>
            <dt>Envolvidos</dt><dd>{{ envolvidos.length }}</dd>
            <template v-if="local.tipo === 'PADS'"><dt>Imputações</dt><dd>{{ totalImputacoes }}</dd></template>
          </dl>
          <button type="button" class="botao primario grande" :disabled="estado.ocupado" data-teste="concluir-cadastro" @click="concluir">Concluir e processar</button>
          <p class="ajuda">Nada é enviado à IA sem a sua autorização, que vem a seguir.</p>
        </aside>
      </div>
    </template>

    <AvisoErro v-if="estado.erro" :erro="estado.erro" />
    <AvisoErro v-else-if="estado.visao?.erro" :erro="estado.visao.erro" data-teste="erro-cadastro" />

    <!-- Modal de cada peça -->
    <Janela :aberta="modalAberto" :titulo="passos[passo]?.rotulo ?? ''" largura="g" teste="modal-peca" @fechar="fecharModal">
      <p class="sub-modal">Peça {{ passo + 1 }} de {{ pecasDosPassos.length }}<template v-if="opcional(atual)"> · opcional</template></p>

      <!-- PADS: Memorando, acusado, fatos e enquadramentos -->
      <div v-if="atual === 'memorando'" class="corpo">
        <div class="duas">
          <section class="parte">
            <h3>PDFs do Memorando Disciplinar</h3>
            <ArquivosPeca v-if="pecaDo('MEMORANDO')" :peca="pecaDo('MEMORANDO')!" :ocupado="estado.ocupado" teste="arq-memorando"
              :carregando="anexando === pecaDo('MEMORANDO')!.id" :documentos="documentosSei"
              @anexar="anexar(pecaDo('MEMORANDO')!)" @anexar-sei="(d) => anexarSei(pecaDo('MEMORANDO')!, d)"
              @remover="(a) => remover(pecaDo('MEMORANDO')!, a)" @mover="(a, d) => mover(pecaDo('MEMORANDO')!, a, d)" />
          </section>
          <section v-if="pessoa(local.acusadoId)" class="parte">
            <h3>Acusado</h3>
            <label>Posto/graduação <input v-model="pessoa(local.acusadoId)!.postoGrad" data-teste="acusado-posto" @input="marcar" /></label>
            <label>Nome completo <input v-model="pessoa(local.acusadoId)!.nome" data-teste="acusado-nome" @input="marcar" /></label>
            <p class="ajuda">Um único acusado. O nome é conferido com a leitura do Memorando e ocultado antes de qualquer envio à IA.</p>
          </section>
        </div>
        <section class="parte separada">
          <div class="cab-bloco">
            <h3>Fatos e enquadramento</h3>
            <button type="button" class="botao ajuda-botao" @click="ajuda = 'fatos'"><Icone nome="ajuda" :tamanho="16" />Como funciona?</button>
          </div>
          <p class="ajuda">Para cada fato do Memorando, escolha o(s) dispositivo(s) da acusação. Cada dispositivo vira uma imputação.</p>
          <div v-for="(f, i) in local.fatos" :key="f.id" class="fato" :data-teste="'fato-' + i">
            <div class="linha-fato">
              <label>Fato {{ i + 1 }} · resumo (opcional) <input v-model="f.resumo" :data-teste="'fato-resumo-' + i" @input="marcar" /></label>
              <label>Data do fato <input v-model="f.data" placeholder="DD/MM/AAAA" :data-teste="'fato-data-' + i" @input="marcar" /></label>
              <button v-if="local.fatos.length > 1" type="button" class="botao icone" :aria-label="'Retirar o fato ' + (i + 1)" title="Retirar este fato"
                @click="local.fatos.splice(i, 1); marcar()"><Icone nome="lixeira" /></button>
            </div>
            <EditorEnquadramento :linhas="f.linhas" :catalogo="catalogo" :teste="'enq-' + i" @mudou="marcar" />
          </div>
          <div><button type="button" class="botao tracejado" data-teste="novo-fato" @click="novoFato"><Icone nome="mais" :tamanho="16" />Acrescentar fato</button></div>
        </section>
      </div>

      <!-- SR: Portaria, sindicados e ofendidos -->
      <div v-if="atual === 'portaria'" class="corpo duas">
        <section class="parte">
          <h3>Portaria de instauração</h3>
          <label>Identificação (número, autoridade e data) <input v-model="local.portaria" data-teste="portaria-ident" @input="marcar" /></label>
          <ArquivosPeca v-if="pecaDo('PORTARIA')" :peca="pecaDo('PORTARIA')!" :ocupado="estado.ocupado" teste="arq-portaria"
            :carregando="anexando === pecaDo('PORTARIA')!.id" :documentos="documentosSei"
            @anexar="anexar(pecaDo('PORTARIA')!)" @anexar-sei="(d) => anexarSei(pecaDo('PORTARIA')!, d)"
            @remover="(a) => remover(pecaDo('PORTARIA')!, a)" @mover="(a, d) => mover(pecaDo('PORTARIA')!, a, d)" />
        </section>
        <div class="coluna">
          <section v-for="lista in (['sindicados', 'ofendidos'] as const)" :key="lista" class="parte">
            <div class="cab-bloco">
              <h3>{{ lista === "sindicados" ? "Sindicados" : "Ofendidos" }}</h3>
              <button v-if="lista === 'ofendidos'" type="button" class="botao ajuda-botao" @click="ajuda = 'ofendidos'"><Icone nome="ajuda" :tamanho="16" />Como funciona?</button>
            </div>
            <p v-if="lista === 'ofendidos'" class="ajuda">Só pessoas. Se o ofendido é a Administração ou uma instituição, não a cadastre aqui.</p>
            <div v-for="id in local[lista]" :key="id" class="linha2" :data-teste="lista + '-item'">
              <template v-if="pessoa(id)">
                <label>Posto/graduação <input v-model="pessoa(id)!.postoGrad" @input="marcar" /></label>
                <label>Nome <input v-model="pessoa(id)!.nome" :data-teste="lista + '-nome'" @input="marcar" /></label>
                <button type="button" class="botao icone" aria-label="Retirar" title="Retirar" @click="retirarEnvolvido(lista, id)"><Icone nome="lixeira" /></button>
                <p v-if="lista === 'ofendidos' && institucional(pessoa(id)!)" class="alerta-campo" data-teste="ofendido-institucional">
                  Parece uma instituição, e não uma pessoa. Se for, retire-a e marque "Não há ofendido individualizado".
                </p>
              </template>
            </div>
            <label v-if="lista === 'ofendidos' && local.ofendidos.length === 0" class="marca">
              <input v-model="local.semOfendido" type="checkbox" data-teste="sem-ofendido" @change="marcar" /> Não há ofendido individualizado
            </label>
            <div><button type="button" class="botao tracejado" :data-teste="'novo-' + lista" @click="acrescentarEnvolvido(lista)">
              <Icone nome="mais" :tamanho="16" />{{ lista === "sindicados" ? "Acrescentar sindicado" : "Acrescentar ofendido" }}</button></div>
          </section>
        </div>
      </div>

      <!-- Defesa prévia e alegações finais (PADS) -->
      <div v-for="t in (['defesa', 'alegacoes'] as const)" v-show="atual === t" :key="t" class="corpo estreito">
        <template v-if="pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')">
          <label class="marca destaque-marca">
            <input v-model="pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!.naoApresentada" type="checkbox" :data-teste="'nao-' + t" @change="marcar" />
            {{ t === "defesa" ? "A defesa prévia não foi apresentada" : "As alegações finais não foram apresentadas" }}
          </label>
          <ArquivosPeca v-if="!pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!.naoApresentada"
            :peca="pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!" :ocupado="estado.ocupado" :teste="'arq-' + t"
            :carregando="anexando === pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!.id" :documentos="documentosSei"
            @anexar="anexar(pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!)"
            @anexar-sei="(d) => anexarSei(pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!, d)"
            @remover="(a) => remover(pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!, a)"
            @mover="(a, d) => mover(pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!, a, d)" />
        </template>
      </div>

      <!-- Depoimentos: uma linha por oitiva -->
      <div v-if="atual === 'depoimentos'" class="corpo">
        <div class="cab-bloco">
          <p class="ajuda">Cada oitiva é uma peça: papel, posto e nome de quem depôs, e o PDF do termo.
            <template v-if="local.tipo === 'PADS'">O interrogatório do acusado vai em "Outros documentos".</template></p>
          <button type="button" class="botao ajuda-botao" @click="ajuda = 'depoimentos'"><Icone nome="ajuda" :tamanho="16" />Como funciona?</button>
        </div>
        <div class="rolagem">
          <div class="tabela-dep">
            <div v-if="doTipo('DEPOIMENTO').length" class="linha-dep cab" aria-hidden="true">
              <span>#</span><span>Papel</span><span>Posto</span><span>Nome do depoente</span><span>Data da oitiva</span><span>PDF do termo</span><span></span>
            </div>
            <section v-for="(p, i) in doTipo('DEPOIMENTO')" :key="p.id" class="dep" :class="{ aviso: homonimo(p) }" data-teste="depoimento">
              <div class="linha-dep">
                <strong class="num">{{ i + 1 }}</strong>
                <fieldset class="papel">
                  <legend class="visualmente-oculto">Papel do depoente {{ i + 1 }}</legend>
                  <div class="segmentado">
                    <label v-for="o in papeis" :key="o.id" :class="{ marcado: p.papel === o.id }">
                      <input v-model="p.papel" type="radio" :value="o.id" :data-teste="'papel-' + i + '-' + o.id" @change="marcar" /> {{ o.rotulo }}
                    </label>
                  </div>
                </fieldset>
                <template v-if="pessoa(p.pessoaIds[0])">
                  <input v-model="pessoa(p.pessoaIds[0])!.postoGrad" :aria-label="'Posto do depoente ' + (i + 1)" placeholder="Posto" :data-teste="'depoente-posto-' + i" @input="marcar" />
                  <input v-model="pessoa(p.pessoaIds[0])!.nome" :aria-label="'Nome do depoente ' + (i + 1)" placeholder="Nome completo" :data-teste="'depoente-nome-' + i" @input="marcar" />
                </template>
                <template v-else><span></span><span></span></template>
                <input v-model="p.data" :aria-label="'Data da oitiva ' + (i + 1)" placeholder="DD/MM/AAAA" @input="marcar" />
                <ArquivosPeca :peca="p" :ocupado="estado.ocupado" :teste="'arq-dep-' + i" compacto :carregando="anexando === p.id"
                  :documentos="documentosSei" @anexar="anexar(p)" @anexar-sei="(d) => anexarSei(p, d)" @remover="(a) => remover(p, a)" @mover="(a, d) => mover(p, a, d)" />
                <button type="button" class="botao icone" :aria-label="'Retirar o depoimento ' + (i + 1)" title="Retirar este depoimento" @click="retirarPeca(p)"><Icone nome="lixeira" /></button>
              </div>
              <div v-if="homonimo(p)" class="sugestao" :data-teste="'depoente-homonimo-' + i">
                <span>Esta pessoa já está cadastrada como {{ papelCadastrado(homonimo(p)!.id) }}.</span>
                <button type="button" class="botao" :data-teste="'depoente-usar-mesma-' + i" @click="usarMesma(p, homonimo(p)!)">Usar a mesma</button>
              </div>
              <div v-if="opcoesReuso(p).length" class="reuso">
                <label>Já cadastrada em outra parte? (opcional)
                  <select :value="valorReuso(p)" :data-teste="'depoente-' + i" @change="trocarDepoente(p, ($event.target as HTMLSelectElement).value)">
                    <option value="">Não: é a pessoa digitada acima</option>
                    <option v-for="ps in opcoesReuso(p)" :key="ps.id" :value="ps.id">Sim: {{ rotuloPessoa(ps) }}</option>
                  </select>
                </label>
                <p v-if="valorReuso(p)" class="ajuda" :data-teste="'depoente-reuso-' + i">
                  Mesma pessoa de outra parte do cadastro: o que você alterar no nome ou no posto vale para as duas.
                </p>
              </div>
            </section>
          </div>
        </div>
        <div><button type="button" class="botao tracejado" data-teste="novo-depoimento" @click="novoDepoimento"><Icone nome="mais" :tamanho="16" />Acrescentar depoimento</button></div>
      </div>

      <!-- Outros documentos -->
      <div v-if="atual === 'outros'" class="corpo">
        <p class="ajuda">Documentos que não são as peças anteriores (ofícios, certidões, relatórios, fotografias em PDF…).</p>
        <section v-for="(p, i) in doTipo('OUTRO')" :key="p.id" class="linha-outro" data-teste="outro">
          <label>Nome do documento <input v-model="p.titulo" :data-teste="'outro-titulo-' + i" @input="marcar" /></label>
          <ArquivosPeca :peca="p" :ocupado="estado.ocupado" :teste="'arq-outro-' + i" compacto :carregando="anexando === p.id"
            :documentos="documentosSei" @anexar="anexar(p)" @anexar-sei="(d) => anexarSei(p, d)" @remover="(a) => remover(p, a)" @mover="(a, d) => mover(p, a, d)" />
          <button type="button" class="botao icone" :aria-label="'Retirar o documento ' + (i + 1)" title="Retirar este documento" @click="retirarPeca(p)"><Icone nome="lixeira" /></button>
        </section>
        <div><button type="button" class="botao tracejado" data-teste="novo-outro" @click="novaPeca('OUTRO')"><Icone nome="mais" :tamanho="16" />Acrescentar documento</button></div>
      </div>

      <footer class="rodape-modal">
        <span class="info-rodape" :class="{ alerta: anexando }">{{ anexando ? "Aguarde o PDF terminar de carregar." : infoCartao(atual) }}</span>
        <button type="button" class="botao" :disabled="estado.ocupado || passo === 0" data-teste="passo-anterior" @click="irPara(passo - 1)">
          <Icone nome="chevron" :tamanho="16" class="volta" />Anterior
        </button>
        <button type="button" class="botao primario" :disabled="estado.ocupado" data-teste="passo-proximo" @click="continuar">
          Continuar: {{ passos[passo + 1]?.rotulo ?? "" }}<Icone nome="chevron" :tamanho="16" />
        </button>
      </footer>
    </Janela>

    <Janela :aberta="!!ajuda" :titulo="TITULOS_AJUDA[ajuda] ?? ''" @fechar="ajuda = ''">
      <div class="texto-ajuda">
        <template v-if="ajuda === 'geral'">
          <p>Clique em cada peça para anexar os arquivos PDF dela (pode haver mais de um) e preencher os dados. Os arquivos são copiados de
            forma cifrada para a área temporária desta tarefa e apagados ao final.</p>
          <p>O tipo que você indica para cada peça é o que vale: o conteúdo só é conferido.</p>
          <p>A peça concluída fica com o sinal verde. Quando não faltar nada, vá para a conferência e processe.</p>
        </template>
        <template v-else-if="ajuda === 'fatos'">
          <p>Para cada fato do Memorando, escolha o(s) dispositivo(s) da acusação. Cada dispositivo vira uma imputação.</p>
          <p>As imputações do mesmo fato são conexas: vale a mais grave, e as outras viram agravantes.</p>
          <p>O enquadramento é seu: o aplicativo não sugere dispositivo e, depois da leitura do Memorando, confere a sua escolha com o
            dispositivo citado nele.</p>
        </template>
        <template v-else-if="ajuda === 'depoimentos'">
          <p>Cada oitiva é uma peça. Digite o posto e o nome de quem depôs.</p>
          <p>Se a mesma pessoa já foi cadastrada (outra oitiva<template v-if="local.tipo === 'SR'">, sindicado ou ofendido</template>),
            escolha-a em "Já cadastrada em outra parte?".</p>
          <p v-if="local.tipo === 'PADS'">O interrogatório do acusado entra em "Outros documentos".</p>
        </template>
        <template v-else-if="ajuda === 'envolvidos'">
          <p>Os nomes e as outras grafias informadas (apelidos, abreviações, separados por ponto e vírgula) são trocados por marcadores antes
            de qualquer envio à IA e restaurados na resposta, só neste computador.</p>
        </template>
        <template v-else-if="ajuda === 'ofendidos'">
          <p>Cadastre aqui só pessoas. Se o ofendido é a Administração ou uma instituição, não a cadastre: deixe a lista vazia e marque
            "Não há ofendido individualizado".</p>
        </template>
      </div>
    </Janela>
  </section>
</template>

<style scoped>
.montagem { display: grid; gap: 1.4rem; }
.cab { display: flex; flex-wrap: wrap; align-items: flex-end; justify-content: space-between; gap: 0.8rem; }
.cab .titulo { display: grid; gap: 0.3rem; }
.cab h2, .conf-cab h2 { font-size: 1.9rem; }
.cab p, .conf-cab p, .ajuda { color: var(--tinta-suave); }
.ajuda { font-size: var(--t-sm); }

.progresso { display: flex; align-items: center; gap: 0.9rem; font-size: var(--t-sm); color: var(--tinta-suave); }
.progresso strong { color: var(--tinta); }
.trilho { flex: 1; height: 8px; border-radius: 999px; background: #dfe3dc; overflow: hidden; }
.trilho span { display: block; height: 100%; background: var(--ok); border-radius: 999px; transition: width var(--dur-3) var(--ease); }

.cartoes { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(100%, 18rem), 1fr)); gap: 1.1rem; }
.cartao-peca {
  position: relative;
  text-align: left;
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
  min-height: 10.5rem;
  padding: 1.35rem;
  background: var(--superficie);
  border: 1.5px solid var(--linha);
  border-radius: 14px;
  box-shadow: var(--sombra-1);
  color: inherit;
  transition: border-color var(--dur-1) var(--ease), box-shadow var(--dur-2) var(--ease), transform var(--dur-2) var(--ease);
}
button.cartao-peca:hover:not(:disabled) { border-color: var(--acento); box-shadow: var(--sombra-2); transform: translateY(-2px); }
.cartao-peca.feito { border-color: #c3dfc9; }
.cartao-peca.proxima { border: 2px solid var(--acento); box-shadow: 0 12px 28px -14px rgba(122, 46, 58, 0.45); }
.cartao-peca.opcional { border-style: dashed; border-color: var(--linha-forte); }
.cartao-peca.conferir { background: var(--superficie-2); }
.cartao-peca.conferir.pronto { background: var(--superficie); border-color: #c3dfc9; }
.marca-ok { position: absolute; top: 1rem; right: 1rem; width: 1.9rem; height: 1.9rem; border-radius: 50%; background: var(--ok); color: #fff; display: grid; place-items: center; }
.etiqueta { position: absolute; top: 1rem; right: 1rem; font-size: var(--t-xs); color: var(--tinta-suave); border: 1px solid var(--linha); border-radius: 999px; padding: 0.05rem 0.6rem; }
.etiqueta.pendente { color: var(--alerta); background: var(--alerta-claro); border-color: var(--alerta-linha); font-weight: 600; }
.ic-peca { width: 2.75rem; height: 2.75rem; border-radius: 10px; background: var(--acento-claro); color: var(--acento); display: grid; place-items: center; }
.ic-peca.ok { background: var(--ok-claro); color: var(--ok); }
.nome-peca { font-size: 1.05rem; }
.info-peca { font-size: var(--t-sm); color: var(--tinta-suave); }
.info-peca.falta { color: var(--alerta); }
.acao-peca { margin-top: auto; font-size: var(--t-sm); font-weight: 600; color: var(--acento); }
.acao-peca.forte { align-self: flex-start; background: var(--acento); color: #fff; border-radius: var(--r-sm); padding: 0.3rem 0.8rem; }
.cartao-peca.conferir .botao { margin-top: auto; align-self: flex-start; }

/* modal */
.sub-modal { font-size: var(--t-sm); color: var(--tinta-suave); margin-bottom: 0.9rem; }
.corpo { display: grid; gap: 1.1rem; }
.corpo.estreito { max-width: 46rem; }
.duas { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1.4rem; align-items: start; }
.coluna { display: grid; gap: 1.1rem; align-content: start; }
.parte { display: grid; gap: 0.7rem; align-content: start; }
.parte.separada { border-top: 1px solid var(--linha); padding-top: 1.1rem; }
.parte h3, .bloco h3 { font-family: var(--sans); font-size: var(--t-md); font-weight: 600; }
.cab-bloco { display: flex; align-items: center; justify-content: space-between; gap: 0.8rem; flex-wrap: wrap; }
label { display: grid; gap: 0.3rem; font-size: var(--t-sm); font-weight: 600; }
label input, label select { font-weight: 400; }
label.marca { display: flex; gap: 0.5rem; align-items: center; font-weight: 400; }
.destaque-marca { border: 1px solid var(--linha); border-radius: var(--r-sm); padding: 0.75rem 0.9rem; background: var(--superficie-2); }
.fato { display: grid; gap: 0.7rem; padding: 0.95rem; border: 1px solid var(--linha); border-radius: var(--r-md); background: #fbfbf9; }
.linha-fato { display: grid; grid-template-columns: minmax(0, 1fr) 11rem auto; gap: 0.7rem; align-items: end; }
.linha2 { display: grid; grid-template-columns: minmax(8rem, 11rem) minmax(0, 1fr) auto; gap: 0.6rem; align-items: end; }
.alerta-campo { grid-column: 1 / -1; color: var(--alerta); font-size: var(--t-sm); }
.botao.tracejado { border: 1.5px dashed var(--linha-forte); background: transparent; color: var(--acento); font-weight: 600; }
.botao.tracejado:hover:not(:disabled) { border-color: var(--acento); background: var(--acento-claro); }
.rolagem { overflow-x: auto; }
.tabela-dep { min-width: 960px; display: grid; gap: 0.6rem; }
.linha-dep {
  display: grid;
  grid-template-columns: 1.6rem 13.5rem 7.5rem minmax(0, 1fr) 8.5rem 16rem 2.6rem;
  gap: 0.6rem;
  align-items: center;
}
.linha-dep.cab { font-size: var(--t-xs); font-weight: 600; color: var(--tinta-suave); padding: 0 0.8rem; }
.dep { border: 1px solid var(--linha); border-radius: var(--r-md); background: #fbfbf9; padding: 0.7rem 0.8rem; display: grid; gap: 0.6rem; }
.dep.aviso { border-color: var(--info); }
.num { color: var(--tinta-suave); }
.papel { border: 0; margin: 0; padding: 0; min-width: 0; }
.papel .segmentado > label { padding-inline: 0.6rem; font-weight: 400; }
.papel .segmentado > label.marcado { font-weight: 600; }
.sugestao { display: flex; align-items: center; gap: 0.8rem; flex-wrap: wrap; background: var(--info-claro); border: 1px solid var(--info-linha); border-radius: var(--r-sm); padding: 0.5rem 0.8rem; font-size: var(--t-sm); }
.sugestao span { flex: 1; }
.reuso { display: grid; gap: 0.3rem; max-width: 34rem; }
.linha-outro { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1.2fr) 2.6rem; gap: 0.8rem; align-items: end; border: 1px solid var(--linha); border-radius: var(--r-md); background: #fbfbf9; padding: 0.8rem; }
.rodape-modal {
  position: sticky;
  bottom: -1.4rem;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 0.7rem;
  flex-wrap: wrap;
  margin: 1.2rem -1.3rem -1.4rem;
  padding: 0.85rem 1.3rem;
  background: var(--superficie-2);
  border-top: 1px solid var(--linha);
}
.info-rodape { flex: 1; font-size: var(--t-sm); color: var(--tinta-suave); }
.info-rodape.alerta { color: var(--alerta); }
.volta { transform: rotate(180deg); }

/* conferência */
.conf-cab { display: grid; gap: 0.3rem; justify-items: start; }
.voltar { padding-left: 0; }
.conf { display: grid; grid-template-columns: minmax(0, 1fr) 20rem; gap: 1.4rem; align-items: start; }
.bloco { display: grid; gap: 0.8rem; padding: 1.2rem 1.3rem; background: var(--superficie); border: 1px solid var(--linha); border-radius: 14px; box-shadow: var(--sombra-1); }
.tabela { width: 100%; border-collapse: collapse; font-size: var(--t-sm); min-width: 34rem; }
.tabela th { text-align: left; font-size: var(--t-xs); color: var(--tinta-suave); font-weight: 600; padding: 0.4rem 0.6rem; border-bottom: 1px solid var(--linha); }
.tabela th.largo { width: 40%; }
.tabela td { padding: 0.5rem 0.6rem; border-bottom: 1px solid #ecefe9; vertical-align: middle; }
.tabela tr:last-child td { border-bottom: 0; }
.tabela td input { min-height: 2.5rem; padding-block: 0.4rem; }
.tabela .ok { color: var(--ok); }
.tabela .falta { color: var(--alerta); }
.tabela .suave { color: var(--tinta-suave); }
.tabela .dir { text-align: right; }
.pronto {
  position: sticky; top: 1rem;
  display: grid; gap: 0.8rem; padding: 1.4rem;
  background: var(--superficie); border: 1px solid var(--linha); border-top: 4px solid var(--ok); border-radius: 14px; box-shadow: var(--sombra-1);
}
.pronto.pendente { border-top-color: var(--alerta); }
.pronto h3 { font-size: var(--t-lg); }
.ic-pronto { width: 3rem; height: 3rem; border-radius: 50%; background: var(--ok-claro); color: var(--ok); display: grid; place-items: center; }
.pendente .ic-pronto { background: var(--alerta-claro); color: var(--alerta); }
.pendencias { margin: 0; padding-left: 1.1rem; color: var(--alerta); display: grid; gap: 0.3rem; font-size: var(--t-sm); }
.avisos-conf { margin: 0; padding: 0.6rem 0.8rem 0.6rem 1.8rem; background: var(--info-claro); border: 1px solid var(--info-linha); border-radius: var(--r-sm); display: grid; gap: 0.25rem; font-size: var(--t-sm); }
.numeros { margin: 0; display: grid; grid-template-columns: 1fr auto; gap: 0.3rem 0.8rem; font-size: var(--t-sm); }
.numeros dt { color: var(--tinta-suave); }
.numeros dd { margin: 0; font-weight: 600; }
.botao.grande { min-height: 3rem; }

@media (max-width: 1000px) {
  .conf { grid-template-columns: 1fr; }
  .pronto { position: static; }
  .duas { grid-template-columns: 1fr; }
}
@media (max-width: 640px) {
  .linha-fato, .linha2, .linha-outro { grid-template-columns: 1fr; }
}
</style>

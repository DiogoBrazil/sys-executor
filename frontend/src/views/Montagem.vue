<script setup lang="ts">
// Assistente de cadastro das peças (V4 §48.4; E6): peça por peça, com vários
// arquivos por peça. PADS: Memorando (acusado, fatos e enquadramentos),
// defesa prévia, depoimentos, alegações finais e outros documentos. SR:
// Portaria (sindicados e ofendidos), depoimentos e outros documentos.
// Nada é persistido aqui: o cadastro vive na sessão da tarefa, no Go.
import { computed, onMounted, ref, watch } from "vue";
import {
  anexarArquivos, anexarDocumentosSEI, catalogoEnquadramento, concluirCadastro, estado, moverArquivo, removerArquivo, salvarCadastro, voltarParaEntrada,
} from "../estado";
import type { Cadastro, DispositivoVisao } from "../api";
import type { PecaCadastro } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";
import ArquivosPeca from "../components/ArquivosPeca.vue";
import EditorEnquadramento, { type LinhaEnquadramento } from "../components/EditorEnquadramento.vue";
import AvisoErro from "../components/AvisoErro.vue";

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
const anexar = (p: PecaCadastro) => comPeca(p, anexarArquivos);
const anexarSei = (p: PecaCadastro, doc: string) => comPeca(p, (c, id) => anexarDocumentosSEI(c, id, [doc]));
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
    <header>
      <h2>Cadastre as peças do procedimento</h2>
      <p>
        Indique cada peça e anexe os arquivos PDF dela (pode haver mais de um). Os arquivos são copiados de forma cifrada para a área
        temporária desta tarefa e apagados ao final. O tipo que você indica para cada peça é o que vale: o conteúdo só é conferido.
      </p>
      <p v-if="documentosSei.length" class="aviso-sei" data-teste="aviso-sei">
        O PDF do processo foi baixado do SEI e separado em {{ documentosSei.length }} documento(s). Em cada peça, use
        "Documento do SEI" para escolher os que a compõem (o tipo sugerido vem do título e deve ser conferido). Os documentos não
        escolhidos não entram nos autos, não são enviados à IA e são apagados ao final.
      </p>
    </header>

    <nav class="passos" aria-label="Passos do cadastro">
      <button v-for="(p, i) in passos" :key="p.id" type="button" class="passo" :class="{ atual: i === passo }"
        :aria-current="i === passo ? 'step' : undefined" :disabled="estado.ocupado" :data-teste="'passo-' + p.id" @click="irPara(i)">
        <span class="num">{{ i + 1 }}</span> {{ p.rotulo }}
      </button>
    </nav>

    <!-- PADS: Memorando, acusado, fatos e enquadramentos -->
    <div v-if="atual === 'memorando'" class="corpo">
      <section class="bloco">
        <h3>Memorando Disciplinar</h3>
        <ArquivosPeca v-if="pecaDo('MEMORANDO')" :peca="pecaDo('MEMORANDO')!" :ocupado="estado.ocupado" teste="arq-memorando"
          :documentos="documentosSei" @anexar="anexar(pecaDo('MEMORANDO')!)" @anexar-sei="(d) => anexarSei(pecaDo('MEMORANDO')!, d)" @remover="(a) => remover(pecaDo('MEMORANDO')!, a)" @mover="(a, d) => mover(pecaDo('MEMORANDO')!, a, d)" />
      </section>
      <section v-if="pessoa(local.acusadoId)" class="bloco">
        <h3>Acusado</h3>
        <div class="linha2">
          <label>Posto/graduação <input v-model="pessoa(local.acusadoId)!.postoGrad" data-teste="acusado-posto" @input="marcar" /></label>
          <label class="largo">Nome completo <input v-model="pessoa(local.acusadoId)!.nome" data-teste="acusado-nome" @input="marcar" /></label>
        </div>
        <p class="ajuda">O PADS tem um único acusado. O nome é conferido com a leitura do Memorando e ocultado antes de qualquer envio à IA.</p>
      </section>
      <section class="bloco">
        <h3>Fatos e enquadramento</h3>
        <p class="ajuda">
          Para cada fato do Memorando, escolha o(s) dispositivo(s) da acusação. Cada dispositivo vira uma imputação; as do mesmo fato
          são conexas (vale a mais grave, e as outras viram agravantes). O enquadramento é seu: o aplicativo não sugere dispositivo e,
          depois da leitura do Memorando, confere a sua escolha com o dispositivo citado nele.
        </p>
        <div v-for="(f, i) in local.fatos" :key="f.id" class="fato" :data-teste="'fato-' + i">
          <div class="linha2">
            <label class="largo">Fato {{ i + 1 }}: resumo (opcional) <input v-model="f.resumo" :data-teste="'fato-resumo-' + i" @input="marcar" /></label>
            <label>Data do fato (opcional) <input v-model="f.data" placeholder="DD/MM/AAAA" :data-teste="'fato-data-' + i" @input="marcar" /></label>
          </div>
          <EditorEnquadramento :linhas="f.linhas" :catalogo="catalogo" :teste="'enq-' + i" @mudou="marcar" />
          <div v-if="local.fatos.length > 1">
            <button type="button" class="botao discreto" @click="local.fatos.splice(i, 1); marcar()">Retirar este fato</button>
          </div>
        </div>
        <div><button type="button" class="botao" data-teste="novo-fato" @click="novoFato">Acrescentar fato</button></div>
      </section>
    </div>

    <!-- SR: Portaria, sindicados e ofendidos -->
    <div v-if="atual === 'portaria'" class="corpo">
      <section class="bloco">
        <h3>Portaria de instauração</h3>
        <label class="largo">Identificação (número, autoridade e data) <input v-model="local.portaria" data-teste="portaria-ident" @input="marcar" /></label>
        <ArquivosPeca v-if="pecaDo('PORTARIA')" :peca="pecaDo('PORTARIA')!" :ocupado="estado.ocupado" teste="arq-portaria"
          :documentos="documentosSei" @anexar="anexar(pecaDo('PORTARIA')!)" @anexar-sei="(d) => anexarSei(pecaDo('PORTARIA')!, d)" @remover="(a) => remover(pecaDo('PORTARIA')!, a)" @mover="(a, d) => mover(pecaDo('PORTARIA')!, a, d)" />
      </section>
      <section v-for="lista in (['sindicados', 'ofendidos'] as const)" :key="lista" class="bloco">
        <h3>{{ lista === "sindicados" ? "Sindicados" : "Ofendidos" }}</h3>
        <p v-if="lista === 'ofendidos'" class="ajuda">
          Só pessoas. Se o ofendido é a Administração ou uma instituição, não a cadastre aqui: deixe a lista vazia e marque
          "Não há ofendido individualizado".
        </p>
        <div v-for="id in local[lista]" :key="id" class="linha2" :data-teste="lista + '-item'">
          <template v-if="pessoa(id)">
            <label>Posto/graduação <input v-model="pessoa(id)!.postoGrad" @input="marcar" /></label>
            <label class="largo">Nome <input v-model="pessoa(id)!.nome" :data-teste="lista + '-nome'" @input="marcar" /></label>
            <button type="button" class="botao discreto" @click="retirarEnvolvido(lista, id)">Retirar</button>
            <p v-if="lista === 'ofendidos' && institucional(pessoa(id)!)" class="alerta-campo" data-teste="ofendido-institucional">
              Parece uma instituição, e não uma pessoa. Se for, retire-a e marque "Não há ofendido individualizado".
            </p>
          </template>
        </div>
        <label v-if="lista === 'ofendidos' && local.ofendidos.length === 0" class="marca">
          <input v-model="local.semOfendido" type="checkbox" data-teste="sem-ofendido" @change="marcar" /> Não há ofendido individualizado
        </label>
        <div><button type="button" class="botao" :data-teste="'novo-' + lista" @click="acrescentarEnvolvido(lista)">
          {{ lista === "sindicados" ? "Acrescentar sindicado" : "Acrescentar ofendido" }}</button></div>
      </section>
    </div>

    <!-- Defesa prévia e alegações finais (PADS) -->
    <div v-for="t in (['defesa', 'alegacoes'] as const)" v-show="atual === t" :key="t" class="corpo">
      <section v-if="pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')" class="bloco">
        <h3>{{ t === "defesa" ? "Defesa prévia" : "Alegações finais" }}</h3>
        <label class="marca">
          <input v-model="pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!.naoApresentada" type="checkbox" :data-teste="'nao-' + t" @change="marcar" />
          {{ t === "defesa" ? "A defesa prévia não foi apresentada" : "As alegações finais não foram apresentadas" }}
        </label>
        <ArquivosPeca v-if="!pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!.naoApresentada"
          :peca="pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!" :ocupado="estado.ocupado" :teste="'arq-' + t"
          :documentos="documentosSei" @anexar="anexar(pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!)" @anexar-sei="(d) => anexarSei(pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!, d)"
          @remover="(a) => remover(pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!, a)"
          @mover="(a, d) => mover(pecaDo(t === 'defesa' ? 'DEFESA_PREVIA' : 'ALEGACOES_FINAIS')!, a, d)" />
      </section>
    </div>

    <!-- Depoimentos -->
    <div v-if="atual === 'depoimentos'" class="corpo">
      <p class="ajuda">
        Cada oitiva é uma peça. Digite o posto e o nome de quem depôs. Se a mesma pessoa já foi cadastrada (outra oitiva<template v-if="local.tipo === 'SR'">, sindicado ou ofendido</template>), escolha-a em "Já cadastrada em outra parte?".
        <template v-if="local.tipo === 'PADS'">O interrogatório do acusado entra em "Outros documentos".</template>
      </p>
      <section v-for="(p, i) in doTipo('DEPOIMENTO')" :key="p.id" class="bloco" data-teste="depoimento">
        <h3>Depoimento {{ i + 1 }}</h3>
        <fieldset class="papel">
          <legend>Papel do depoente</legend>
          <label v-for="o in papeis" :key="o.id"><input v-model="p.papel" type="radio" :value="o.id" :data-teste="'papel-' + i + '-' + o.id" @change="marcar" /> {{ o.rotulo }}</label>
        </fieldset>
        <div v-if="pessoa(p.pessoaIds[0])" class="linha2">
          <label>Posto/graduação (se militar) <input v-model="pessoa(p.pessoaIds[0])!.postoGrad" :data-teste="'depoente-posto-' + i" @input="marcar" /></label>
          <label class="largo">Nome do depoente <input v-model="pessoa(p.pessoaIds[0])!.nome" :data-teste="'depoente-nome-' + i" @input="marcar" /></label>
        </div>
        <div v-if="homonimo(p)" class="callout info sugestao" :data-teste="'depoente-homonimo-' + i">
          <p>Esta pessoa já está cadastrada como {{ papelCadastrado(homonimo(p)!.id) }}.</p>
          <div><button type="button" class="botao" :data-teste="'depoente-usar-mesma-' + i" @click="usarMesma(p, homonimo(p)!)">Usar a mesma</button></div>
        </div>
        <label v-if="opcoesReuso(p).length" class="reuso">Já cadastrada em outra parte? (opcional)
          <select :value="valorReuso(p)" :data-teste="'depoente-' + i" @change="trocarDepoente(p, ($event.target as HTMLSelectElement).value)">
            <option value="">Não: é a pessoa digitada acima</option>
            <option v-for="ps in opcoesReuso(p)" :key="ps.id" :value="ps.id">Sim: {{ rotuloPessoa(ps) }}</option>
          </select>
        </label>
        <p v-if="valorReuso(p)" class="ajuda" :data-teste="'depoente-reuso-' + i">
          Mesma pessoa de outra parte do cadastro: o que você alterar no nome ou no posto vale para as duas.
        </p>
        <label>Data da oitiva (opcional) <input v-model="p.data" placeholder="DD/MM/AAAA" @input="marcar" /></label>
        <ArquivosPeca :peca="p" :ocupado="estado.ocupado" :teste="'arq-dep-' + i"
          :documentos="documentosSei" @anexar="anexar(p)" @anexar-sei="(d) => anexarSei(p, d)" @remover="(a) => remover(p, a)" @mover="(a, d) => mover(p, a, d)" />
        <div><button type="button" class="botao discreto" @click="retirarPeca(p)">Retirar este depoimento</button></div>
      </section>
      <div><button type="button" class="botao" data-teste="novo-depoimento" @click="novoDepoimento">Acrescentar depoimento</button></div>
    </div>

    <!-- Outros documentos -->
    <div v-if="atual === 'outros'" class="corpo">
      <p class="ajuda">Documentos que não são as peças acima (ofícios, certidões, relatórios, fotografias em PDF…).</p>
      <section v-for="(p, i) in doTipo('OUTRO')" :key="p.id" class="bloco" data-teste="outro">
        <label class="largo">Nome do documento <input v-model="p.titulo" :data-teste="'outro-titulo-' + i" @input="marcar" /></label>
        <ArquivosPeca :peca="p" :ocupado="estado.ocupado" :teste="'arq-outro-' + i"
          :documentos="documentosSei" @anexar="anexar(p)" @anexar-sei="(d) => anexarSei(p, d)" @remover="(a) => remover(p, a)" @mover="(a, d) => mover(p, a, d)" />
        <div><button type="button" class="botao discreto" @click="retirarPeca(p)">Retirar este documento</button></div>
      </section>
      <div><button type="button" class="botao" data-teste="novo-outro" @click="novaPeca('OUTRO')">Acrescentar documento</button></div>
    </div>

    <!-- Conferência -->
    <div v-if="atual === 'conferencia'" class="corpo">
      <section class="bloco">
        <h3>Peças</h3>
        <ul class="resumo" data-teste="conferencia-pecas">
          <li v-for="p in local.pecas" :key="p.id">
            <strong>{{ descricaoPeca(p) }}</strong>:
            <span v-if="p.naoApresentada">não apresentada</span>
            <span v-else>{{ p.arquivos.length }} arquivo(s), {{ totalPaginas(p) }} página(s)</span>
          </li>
        </ul>
        <p v-if="documentosSei.length" class="ajuda" data-teste="conferencia-sei">
          Documentos do SEI: {{ documentosSei.length - seiNaoEscolhidos }} escolhido(s) e {{ seiNaoEscolhidos }} não escolhido(s), que não
          entram nos autos.
        </p>
      </section>
      <section class="bloco">
        <h3>Envolvidos</h3>
        <p class="ajuda">
          Os nomes e as outras grafias informadas (apelidos, abreviações, separados por ponto e vírgula) são trocados por marcadores antes
          de qualquer envio à IA e restaurados na resposta, só neste computador.
        </p>
        <div v-for="p in local.pessoas.filter((x) => x.nome.trim())" :key="p.id" class="pessoa" data-teste="pessoa">
          <strong>{{ [p.postoGrad, p.nome].filter(Boolean).join(" ") }}</strong>
          <label class="largo">Outras grafias <input v-model="p.grafias" placeholder="Ex.: Cabo Inventado; Inventado" @input="marcar" /></label>
          <label v-if="!institucional(p)" class="marca"><input v-model="p.crianca" type="checkbox" data-teste="crianca" @change="marcar" /> Menor de 13 anos</label>
          <span v-else class="selo alerta">Instituição?</span>
        </div>
      </section>
      <section class="bloco">
        <h3>Antes de processar</h3>
        <ul v-if="pendencias.length" class="pendencias" data-teste="pendencias">
          <li v-for="p in pendencias" :key="p">{{ p }}</li>
        </ul>
        <p v-else class="ok">O cadastro está completo.</p>
        <ul v-if="avisosConferencia.length" class="avisos-conf" data-teste="avisos-conferencia">
          <li v-for="a in avisosConferencia" :key="a">{{ a }}</li>
        </ul>
        <div class="acoes">
          <button type="button" class="botao primario" :disabled="estado.ocupado" data-teste="concluir-cadastro" @click="concluir">Concluir e processar</button>
        </div>
      </section>
    </div>

    <AvisoErro v-if="estado.erro" :erro="estado.erro" />
    <AvisoErro v-else-if="estado.visao?.erro" :erro="estado.visao.erro" data-teste="erro-cadastro" />

    <footer class="navegar">
      <button type="button" class="botao" :disabled="estado.ocupado || passo === 0" data-teste="passo-anterior" @click="irPara(passo - 1)">Anterior</button>
      <button v-if="passo < passos.length - 1" type="button" class="botao primario" :disabled="estado.ocupado" data-teste="passo-proximo" @click="irPara(passo + 1)">Próximo</button>
      <button type="button" class="botao discreto" :disabled="estado.ocupado" @click="voltarParaEntrada">Escolher outro caminho</button>
    </footer>
  </section>
</template>

<style scoped>
.aviso-sei { border-left: 3px solid var(--linha-forte); padding-left: 0.6rem; }
.montagem { display: grid; gap: 1.2rem; }
header h2 { font-size: var(--t-lg); }
header p, .ajuda { color: var(--tinta-suave); max-width: 72ch; }
.passos { display: flex; flex-wrap: wrap; gap: 0.4rem; }
.passo { font: inherit; font-size: var(--t-sm); border: 1px solid var(--linha); background: var(--superficie); border-radius: 999px; padding: 0.3rem 0.8rem; cursor: pointer; }
.passo.atual { border-color: var(--tinta); background: var(--tinta); color: #fff; }
.num { font-weight: 600; margin-right: 0.2rem; }
.corpo { display: grid; gap: 1rem; }
.bloco { display: grid; gap: 0.7rem; padding: 1.1rem 1.2rem; background: var(--superficie); border: 1px solid var(--linha); border-radius: var(--raio); }
.bloco h3 { font-family: var(--sans); font-size: var(--t-md); font-weight: 600; }
.fato { display: grid; gap: 0.6rem; padding-top: 0.6rem; border-top: 1px solid var(--linha); }
label { display: grid; gap: 0.25rem; font-size: var(--t-sm); }
label.marca { display: flex; gap: 0.5rem; align-items: center; }
.linha2 { display: grid; grid-template-columns: minmax(10rem, 14rem) 1fr auto; gap: 0.6rem; align-items: end; }
.largo { min-width: 0; }
.papel { border: 0; margin: 0; padding: 0; display: flex; gap: 1rem; flex-wrap: wrap; }
.papel legend { font-size: var(--t-sm); font-weight: 600; margin-bottom: 0.25rem; }
.papel label { display: flex; gap: 0.35rem; align-items: center; }
.resumo, .pendencias { margin: 0; padding-left: 1.2rem; display: grid; gap: 0.25rem; }
.pendencias { color: var(--alerta); }
.avisos-conf { margin: 0; padding: 0.6rem 0.8rem 0.6rem 1.8rem; background: var(--info-claro); border: 1px solid var(--info-linha); border-radius: var(--r-sm); display: grid; gap: 0.25rem; }
.alerta-campo { grid-column: 1 / -1; color: var(--alerta); font-size: var(--t-sm); }
.sugestao { align-items: center; }
.ok { color: var(--ok); }
.pessoa { display: grid; grid-template-columns: minmax(12rem, 1fr) 2fr auto; gap: 0.6rem; align-items: end; }
.acoes, .navegar { display: flex; gap: 0.8rem; flex-wrap: wrap; }
@media (max-width: 720px) { .linha2, .pessoa { grid-template-columns: 1fr; } }
</style>

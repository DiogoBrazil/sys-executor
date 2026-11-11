<script setup lang="ts">
// Cartão da pergunta. Formatos (protótipo aprovado em 11/11/2026):
// - confirmação das imputações: uma linha por imputação, em tabela;
// - grupos numerados ("Transgressão anterior 1…5"): linhas acrescentadas uma a uma;
// - dosimetria (grupos s_*): painéis de sanção, dias e comportamento proposto,
//   com a memória de cálculo (m1…m7) num modal;
// - demais: seções com os campos lado a lado.
// Os campos, a validação e o que vai ao Go são os mesmos em todos os formatos.
import { computed, nextTick, reactive, ref, watch } from "vue";
import { estado, responder } from "../estado";
import type { Campo, Fonte, GrupoCampos, PerguntaVisao } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";
import CampoPergunta from "./CampoPergunta.vue";
import Icone from "./ui/Icone.vue";
import Janela from "./ui/Janela.vue";

const props = defineProps<{ pergunta: PerguntaVisao; etapa?: string }>();
const emit = defineEmits<{ abrir: [pagina: number, inicio: number, fim: number] }>();

const valores = reactive<Record<string, string>>({});
const invalidos = ref<Record<string, string>>({});
const raiz = ref<HTMLElement | null>(null);
const verContexto = ref(false);
const verMemoria = ref(false);
const mostrarRep = ref(1);

const campos = computed(() => props.pergunta.campos ?? []);
const grupos = computed(() => props.pergunta.grupos ?? []);
const doGrupo = (g: GrupoCampos) => campos.value.filter((c) => c.grupo === g.id);
const temGrupo = (c: Campo) => !!c.grupo && grupos.value.some((g) => g.id === c.grupo);

// Visibilidade encadeada (como no Go).
function visivel(c: Campo, prof = 0): boolean {
  if (!c.mostrarSe) return true;
  if (prof > 8 || valores[c.mostrarSe.campo] !== c.mostrarSe.valor) return false;
  const pai = campos.value.find((x) => x.id === c.mostrarSe!.campo);
  return !pai || visivel(pai, prof + 1);
}
const temConteudo = (g: GrupoCampos) => !!(g.descricao?.length || g.alertas?.length || g.fontes?.length);
const grupoVisivel = (g: GrupoCampos) => temConteudo(g) || doGrupo(g).some((c) => visivel(c));

// ---- memória de cálculo (modal) ----
const ehMemoria = (g: GrupoCampos) => /^m\d+$/.test(g.id) && !doGrupo(g).length;
const memoria = computed(() => grupos.value.filter(ehMemoria));
const propostaMemoria = computed(() => memoria.value.find((g) => /propost/i.test(g.titulo)));

// ---- dosimetria ----
const ehSancao = (g: GrupoCampos) => /^s_/.test(g.id);
const sancoes = computed(() => grupos.value.filter(ehSancao));
const RE_CIRC = /^(Atenuantes|Agravantes) contadas:\s*(.+)$/;
function partesSancao(g: GrupoCampos) {
  const [imp, sancao] = g.titulo.split(" — ");
  const linhas = g.descricao ?? [];
  const outras = linhas.filter((l) => !RE_CIRC.test(l));
  return {
    imp: imp ?? g.titulo,
    sancao: sancao ? sancao.charAt(0).toUpperCase() + sancao.slice(1) : "",
    principal: outras[0] ?? "",
    notas: outras.slice(1),
    circ: linhas.filter((l) => RE_CIRC.test(l)).map((l) => {
      const m = RE_CIRC.exec(l)!;
      return { tipo: m[1] === "Atenuantes" ? "atenuante" : "agravante", rotulo: m[1], texto: m[2].replace(/\.$/, "") };
    }),
  };
}
const diasDe = (g: GrupoCampos) => doGrupo(g).find((c) => c.tipo === "NUMERO");
function somar(c: Campo, d: number) {
  let v = (Number(valores[c.id] || 0) || 0) + d;
  if (c.min) v = Math.max(c.min, v);
  if (c.max) v = Math.min(c.max, v);
  valores[c.id] = String(v);
}
const propostaResumo = computed(() => {
  const t = (propostaMemoria.value?.descricao ?? []).join(" ");
  const m = /Propõe-se (\d+) pontos?, comportamento ([\p{L}]+)/u.exec(t);
  return m ? `${m[2].charAt(0).toUpperCase() + m[2].slice(1)} · ${m[1]} pontos` : "";
});

// ---- grupos numerados, acrescentados um a um ----
const rep = computed(() => {
  const mapa = new Map<string, GrupoCampos[]>();
  for (const g of grupos.value) {
    const m = /^(.*\S)\s+(\d+)$/.exec(g.titulo);
    if (!m || temConteudo(g) || ehMemoria(g) || ehSancao(g) || !doGrupo(g).length) continue;
    mapa.set(m[1], [...(mapa.get(m[1]) ?? []), g]);
  }
  for (const [prefixo, gs] of mapa) if (gs.length >= 3) return { prefixo, grupos: gs };
  return null;
});
const repVisivel = computed(() => !!rep.value && doGrupo(rep.value.grupos[0]).some((c) => visivel(c)));
function preenchidos(): number {
  if (!rep.value) return 1;
  let n = 1;
  rep.value.grupos.forEach((g, i) => {
    if (doGrupo(g).some((c) => (valores[c.id] ?? "").trim())) n = i + 1;
  });
  return n;
}
function acrescentar() {
  if (rep.value && mostrarRep.value < rep.value.grupos.length) mostrarRep.value++;
}
// Retira a linha k: as seguintes sobem uma posição.
function retirarLinha(k: number) {
  const gs = rep.value!.grupos;
  for (let i = k; i < gs.length; i++) {
    const atuais = doGrupo(gs[i]);
    const prox = i + 1 < gs.length ? doGrupo(gs[i + 1]) : [];
    atuais.forEach((c, j) => (valores[c.id] = prox[j] ? (valores[prox[j].id] ?? "") : ""));
  }
  mostrarRep.value = Math.max(1, mostrarRep.value - 1);
}
const rotuloAcrescentar = computed(() => (rep.value?.prefixo ?? "").toLowerCase().replace(/ anterior$/, ""));

// ---- confirmação das imputações (tabela) ----
const modoTabela = computed(() => props.pergunta.id === "confirmar_delimitacao");
const grupoAcusado = computed(() => (modoTabela.value ? grupos.value.find((g) => g.id === "acusado") : undefined));
const linhasImp = computed(() => (modoTabela.value ? grupos.value.filter((g) => g.id !== "acusado" && grupoVisivel(g)) : []));
const COLUNAS: Record<string, string[]> = {
  conduta: ["conduta_"],
  dispositivo: ["diploma_", "disp_rdpm_", "disp_est_", "natureza_"],
  data: ["data_"],
  incluir: ["incluir_"],
};
const TODAS = Object.values(COLUNAS).flat();
const naColuna = (g: GrupoCampos, col: string) => doGrupo(g).filter((c) => visivel(c) && COLUNAS[col].some((p) => c.id === p + g.id));
const foraDasColunas = (g: GrupoCampos) => doGrupo(g).filter((c) => visivel(c) && !TODAS.some((p) => c.id === p + g.id));
const naturezaLida = (g: GrupoCampos) => {
  for (const d of g.descricao ?? []) {
    const m = /Natureza declarada no Memorando: ([\p{L}]+)/u.exec(d);
    if (m) return m[1].toLowerCase();
  }
  return "";
};
// Linhas da leitura da IA, menos a conduta (já no campo) e a natureza (selo).
const notasImp = (g: GrupoCampos) => {
  const conduta = doGrupo(g).find((c) => c.id === "conduta_" + g.id)?.valor ?? "";
  return (g.descricao ?? []).filter((d) => d !== conduta && !/^Natureza declarada no Memorando/.test(d));
};
const incluida = (g: GrupoCampos) => valores["incluir_" + g.id] === "sim";

// ---- demais grupos e campos soltos ----
const principais = computed(() =>
  modoTabela.value ? [] : grupos.value.filter((g) => !ehMemoria(g) && !ehSancao(g) && !rep.value?.grupos.includes(g) && grupoVisivel(g)),
);
const soltos = computed(() => campos.value.filter((c) => !temGrupo(c)));
// Campo solto de que dependem campos de grupo (ex.: "Há transgressões?") vem antes.
const controla = (c: Campo) => campos.value.some((x) => temGrupo(x) && x.mostrarSe?.campo === c.id);
const soltosAntes = computed(() => soltos.value.filter(controla));
const soltosDepois = computed(() => soltos.value.filter((c) => !controla(c)));

// Explicação longa: duas linhas e o texto inteiro num modal (a autorização de
// envio fica inteira: é o texto do consentimento).
const contextoLongo = computed(() => props.pergunta.id !== "envio_ia" && (props.pergunta.contexto ?? "").length > 260);

// Trechos citados: os quatro primeiros e "mais n".
const LIMITE_FONTES = 4;
const fontesAbertas = reactive(new Set<string>());
const fontesDe = (chave: string, fs: Fonte[]) => (fontesAbertas.has(chave) ? fs : fs.slice(0, LIMITE_FONTES));
const rotuloFonte = (f: Fonte) => `${f.rotulo || "Autos"} — pág. ${f.pagina}`;

function rotulado(t: string): { rotulo: string; texto: string } | null {
  const m = /^([A-ZÀ-Ú][^:.\n]{1,38}):\s+([\s\S]+)$/.exec(t);
  return m ? { rotulo: m[1], texto: m[2] } : null;
}
const definir = (c: Campo) => (v: string) => (valores[c.id] = v);
const repassar = (p: number, i: number, f: number) => emit("abrir", p, i, f);

function iniciar() {
  for (const k of Object.keys(valores)) delete valores[k];
  for (const c of props.pergunta.campos ?? []) valores[c.id] = c.valor ?? "";
  invalidos.value = {};
  verContexto.value = false;
  verMemoria.value = false;
  fontesAbertas.clear();
  mostrarRep.value = preenchidos();
}
watch(() => props.pergunta.id, iniciar, { immediate: true });

const reData = /^(\d{2})\/(\d{2})\/(\d{4})$/;
function dataValida(s: string): boolean {
  const m = reData.exec(s.trim());
  if (!m) return false;
  const [d, mes, a] = [Number(m[1]), Number(m[2]), Number(m[3])];
  const dt = new Date(a, mes - 1, d);
  return dt.getFullYear() === a && dt.getMonth() === mes - 1 && dt.getDate() === d;
}

function validar(): boolean {
  const erros: Record<string, string> = {};
  for (const c of campos.value) {
    if (!visivel(c)) continue;
    const v = (valores[c.id] ?? "").trim();
    if (!v) {
      if (c.obrigatorio) erros[c.id] = "Obrigatório.";
      continue;
    }
    if (c.tipo === "DATA" && !dataValida(v)) erros[c.id] = "Use DD/MM/AAAA.";
    if (c.tipo === "NUMERO" && (!/^-?\d+$/.test(v) || ((c.min || c.max) && (Number(v) < c.min || Number(v) > c.max)))) {
      erros[c.id] = `Número entre ${c.min} e ${c.max}.`;
    }
  }
  invalidos.value = erros;
  return Object.keys(erros).length === 0;
}

async function irAoPrimeiroErro() {
  await nextTick();
  const el = raiz.value?.querySelector<HTMLElement>(".campo.invalido, .painel-dos.invalido");
  el?.scrollIntoView?.({ block: "center", behavior: "smooth" });
  el?.querySelector<HTMLElement>("input, select, textarea")?.focus({ preventScroll: true });
}

async function escolher(opcaoID: string, validarCampos: boolean) {
  if (validarCampos && !validar()) {
    await irAoPrimeiroErro();
    return;
  }
  const enviados: Record<string, string> = {};
  if (validarCampos) {
    for (const c of campos.value) if (visivel(c)) enviados[c.id] = valores[c.id] ?? "";
  }
  await responder(props.pergunta.id, opcaoID, enviados);
}
</script>

<template>
  <section ref="raiz" class="pergunta" role="group" :aria-labelledby="'perg-' + pergunta.id" data-teste="pergunta">
    <span v-if="etapa" class="aba-etapa" aria-hidden="true">{{ etapa }}</span>
    <header class="cabeca" :class="{ comAcusado: grupoAcusado }">
      <div class="titulos">
        <h2 :id="'perg-' + pergunta.id">{{ pergunta.titulo }}</h2>
        <p class="contexto" :class="{ cortado: contextoLongo }">{{ pergunta.contexto }}</p>
        <div v-if="contextoLongo">
          <button type="button" class="botao ajuda-botao" data-teste="ler-contexto" @click="verContexto = true">
            <Icone nome="info" :tamanho="16" />Ler a explicação completa
          </button>
        </div>
      </div>
      <!-- Imputações: o acusado numa linha compacta, ao lado do título. -->
      <div v-if="grupoAcusado" class="acusado" :data-teste="'grupo-' + grupoAcusado.id">
        <template v-for="c in doGrupo(grupoAcusado)" :key="c.id">
          <CampoPergunta v-if="visivel(c)" :campo="c" :model-value="valores[c.id] ?? ''" :nome="pergunta.id + c.id"
            :erro="invalidos[c.id]" @update:model-value="definir(c)($event)" @abrir="repassar" />
        </template>
        <ul v-if="grupoAcusado.alertas?.length" class="alertas" role="list">
          <li v-for="(a, i) in grupoAcusado.alertas" :key="i"><Icone nome="alerta" :tamanho="16" />{{ a }}</li>
        </ul>
      </div>
    </header>

    <!-- Confirmação das imputações: uma linha por imputação -->
    <div v-if="modoTabela" class="rolagem">
      <div class="tabela-imp" role="table" aria-label="Imputações">
        <div class="linha-imp cab" role="row">
          <span role="columnheader">Imputação</span><span role="columnheader">Conduta imputada</span>
          <span role="columnheader">Dispositivo (escolha você)</span><span role="columnheader">Data do fato</span><span role="columnheader">Incluir</span>
        </div>
        <div v-for="g in linhasImp" :key="g.id" class="linha-imp" :class="{ excluida: !incluida(g), comErro: doGrupo(g).some((c) => invalidos[c.id]) }"
          role="row" :data-teste="'grupo-' + g.id">
          <div class="cel-imp" role="cell">
            <strong>{{ g.titulo }}</strong>
            <span v-if="naturezaLida(g)" class="natureza" :class="naturezaLida(g)">{{ naturezaLida(g) }}</span>
            <button v-for="(f, i) in g.fontes ?? []" :key="i" type="button" class="chip fonte" :title="f.trecho"
              :data-teste="'fonte-' + g.id + '-' + i" @click="emit('abrir', f.pagina, f.inicio, f.fim)">
              <Icone nome="pdf" :tamanho="14" /><span class="visualmente-oculto">Ver no PDF: </span>{{ rotuloFonte(f) }}
            </button>
          </div>
          <div class="cel" role="cell">
            <CampoPergunta v-for="c in naColuna(g, 'conduta')" :key="c.id" :campo="c" :model-value="valores[c.id] ?? ''" :nome="pergunta.id + c.id"
              :erro="invalidos[c.id]" sem-rotulo :linhas="2" @update:model-value="definir(c)($event)" />
            <p v-if="!incluida(g)" class="nota">Não será julgada.</p>
            <p v-for="(d, i) in notasImp(g)" :key="i" class="nota">{{ d }}</p>
          </div>
          <div class="cel" role="cell">
            <CampoPergunta v-for="c in naColuna(g, 'dispositivo')" :key="c.id" :campo="c" :model-value="valores[c.id] ?? ''" :nome="pergunta.id + c.id"
              :erro="invalidos[c.id]" :sem-rotulo="!c.id.startsWith('natureza_')" segmentado @update:model-value="definir(c)($event)" />
          </div>
          <div class="cel" role="cell">
            <CampoPergunta v-for="c in naColuna(g, 'data')" :key="c.id" :campo="c" :model-value="valores[c.id] ?? ''" :nome="pergunta.id + c.id"
              :erro="invalidos[c.id]" sem-rotulo @update:model-value="definir(c)($event)" />
          </div>
          <div class="cel" role="cell">
            <CampoPergunta v-for="c in naColuna(g, 'incluir')" :key="c.id" :campo="c" :model-value="valores[c.id] ?? ''" :nome="pergunta.id + c.id"
              :erro="invalidos[c.id]" sem-rotulo segmentado @update:model-value="definir(c)($event)" />
          </div>
          <div v-if="g.alertas?.length || foraDasColunas(g).length" class="cel-extra" role="cell">
            <ul v-if="g.alertas?.length" class="alertas" role="list">
              <li v-for="(a, i) in g.alertas" :key="i"><Icone nome="alerta" :tamanho="16" />{{ a }}</li>
            </ul>
            <CampoPergunta v-for="c in foraDasColunas(g)" :key="c.id" :campo="c" :model-value="valores[c.id] ?? ''" :nome="pergunta.id + c.id"
              :erro="invalidos[c.id]" @update:model-value="definir(c)($event)" />
          </div>
        </div>
      </div>
    </div>

    <div v-if="soltosAntes.some((c) => visivel(c))" class="faixa-pergunta">
      <template v-for="c in soltosAntes" :key="'a' + c.id">
        <CampoPergunta v-if="visivel(c)" :campo="c" :model-value="valores[c.id] ?? ''" :nome="pergunta.id + c.id" prefixo="s-"
          :erro="invalidos[c.id]" @update:model-value="definir(c)($event)" />
      </template>
    </div>

    <!-- Grupos numerados: linhas acrescentadas uma a uma -->
    <div v-if="rep && repVisivel" class="repetidos">
      <div class="tabela-rep" role="table" :aria-label="rep.prefixo">
        <div class="linha-rep cab" role="row" :style="{ '--n': doGrupo(rep.grupos[0]).length }">
          <span role="columnheader">#</span>
          <span v-for="c in doGrupo(rep.grupos[0])" :key="c.id" role="columnheader">{{ c.rotulo }}</span>
          <span role="columnheader"></span>
        </div>
        <template v-for="(g, k) in rep.grupos" :key="g.id">
          <div v-if="k < mostrarRep" class="linha-rep" role="row" :style="{ '--n': doGrupo(g).length }" :data-teste="'grupo-' + g.id">
            <strong class="num" role="cell">{{ k + 1 }}</strong>
            <div v-for="c in doGrupo(g)" :key="c.id" role="cell">
              <CampoPergunta v-if="visivel(c)" :campo="c" :model-value="valores[c.id] ?? ''" :nome="pergunta.id + c.id"
                :erro="invalidos[c.id]" sem-rotulo segmentado @update:model-value="definir(c)($event)" />
            </div>
            <button v-if="mostrarRep > 1" type="button" class="botao icone" role="cell" :aria-label="'Retirar ' + g.titulo.toLowerCase()"
              :title="'Retirar ' + g.titulo.toLowerCase()" @click="retirarLinha(k)"><Icone nome="lixeira" /></button>
            <span v-else role="cell"></span>
          </div>
        </template>
      </div>
      <div class="acrescentar">
        <button v-if="mostrarRep < rep.grupos.length" type="button" class="botao tracejado" data-teste="acrescentar-linha" @click="acrescentar">
          <Icone nome="mais" :tamanho="16" />Acrescentar {{ rotuloAcrescentar }}
        </button>
        <span class="nota">{{ mostrarRep }} de até {{ rep.grupos.length }}</span>
      </div>
    </div>

    <!-- Dosimetria: painéis -->
    <div v-if="sancoes.length" class="dosimetria">
      <div class="paineis">
      <template v-for="g in sancoes" :key="g.id">
        <div class="painel-dos" :data-teste="'grupo-' + g.id">
          <span class="rotulo-painel">{{ partesSancao(g).imp }} · sanção</span>
          <strong class="grande">{{ partesSancao(g).sancao }}</strong>
          <span class="nota">{{ partesSancao(g).principal }}</span>
        </div>
        <div v-if="diasDe(g)" class="painel-dos destaque" :class="{ invalido: invalidos[diasDe(g)!.id] }">
          <label class="rotulo-painel" :for="'c-' + diasDe(g)!.id">{{ diasDe(g)!.rotulo }}</label>
          <div class="passo">
            <button type="button" class="botao" aria-label="Menos um dia" @click="somar(diasDe(g)!, -1)">−</button>
            <input :id="'c-' + diasDe(g)!.id" v-model="valores[diasDe(g)!.id]" inputmode="numeric" :data-teste="'campo-' + diasDe(g)!.id" />
            <button type="button" class="botao" aria-label="Mais um dia" @click="somar(diasDe(g)!, 1)">+</button>
          </div>
          <span class="nota">{{ diasDe(g)!.ajuda }} De {{ diasDe(g)!.min }} a {{ diasDe(g)!.max }}.</span>
          <p v-if="invalidos[diasDe(g)!.id]" class="msg" role="alert">{{ invalidos[diasDe(g)!.id] }}</p>
        </div>
      </template>
      <div v-if="propostaMemoria" class="painel-dos" data-teste="memoria-resumo">
        <span class="rotulo-painel">Comportamento proposto</span>
        <strong v-if="propostaResumo" class="grande">{{ propostaResumo }}</strong>
        <span v-for="(d, i) in propostaMemoria.descricao" :key="i" class="nota">{{ d }}</span>
      </div>
      </div>
      <div class="circunstancias">
        <template v-for="g in sancoes" :key="'c' + g.id">
          <span v-for="(c, i) in partesSancao(g).circ" :key="i" class="circ" :class="c.tipo">{{ c.rotulo }} · {{ c.texto }}</span>
          <span v-for="(n, i) in partesSancao(g).notas" :key="'n' + i" class="nota">
            <template v-if="rotulado(n)"><strong>{{ rotulado(n)!.rotulo }}:</strong> {{ rotulado(n)!.texto }}</template>
            <template v-else>{{ n }}</template>
          </span>
        </template>
        <span class="espaco"></span>
        <button v-if="memoria.length" type="button" class="botao" data-teste="abrir-memoria" @click="verMemoria = true">
          <Icone nome="documento" />Ver a memória de cálculo ({{ memoria.length }} itens)
        </button>
      </div>
    </div>

    <section v-for="g in principais" :key="g.id" class="grupo" :data-teste="'grupo-' + g.id">
      <h3>{{ g.titulo }}</h3>
      <template v-for="(d, i) in g.descricao" :key="i">
        <p v-if="rotulado(d)" class="desc"><strong>{{ rotulado(d)!.rotulo }}:</strong> {{ rotulado(d)!.texto }}</p>
        <p v-else class="desc">{{ d }}</p>
      </template>
      <ul v-if="g.alertas?.length" class="alertas" role="list">
        <li v-for="(a, i) in g.alertas" :key="i"><Icone nome="alerta" :tamanho="16" />{{ a }}</li>
      </ul>
      <div v-if="g.fontes?.length" class="fontes">
        <span class="rotulo-fontes">Trechos citados</span>
        <button v-for="(f, i) in fontesDe(g.id, g.fontes)" :key="i" type="button" class="chip fonte" :title="f.trecho"
          :data-teste="'fonte-' + g.id + '-' + i" @click="emit('abrir', f.pagina, f.inicio, f.fim)">
          <Icone nome="pdf" :tamanho="14" /><span class="visualmente-oculto">Ver no PDF: </span>{{ rotuloFonte(f) }}
        </button>
        <button v-if="g.fontes.length > LIMITE_FONTES && !fontesAbertas.has(g.id)" type="button" class="botao discreto mais"
          @click="fontesAbertas.add(g.id)">mais {{ g.fontes.length - LIMITE_FONTES }}</button>
      </div>
      <div v-if="doGrupo(g).some((c) => visivel(c))" class="campos">
        <template v-for="c in doGrupo(g)" :key="c.id">
          <CampoPergunta v-if="visivel(c)" :campo="c" :model-value="valores[c.id] ?? ''" :nome="pergunta.id + c.id"
            :erro="invalidos[c.id]" @update:model-value="definir(c)($event)" @abrir="repassar" />
        </template>
      </div>
    </section>

    <div v-if="memoria.length && !sancoes.length" class="memoria" data-teste="memoria-resumo">
      <div v-if="propostaMemoria" class="proposta">
        <strong>{{ propostaMemoria.titulo.replace(/^\d+\.\s*/, "") }}</strong>
        <p v-for="(d, i) in propostaMemoria.descricao" :key="i">{{ d }}</p>
      </div>
      <button type="button" class="botao" data-teste="abrir-memoria" @click="verMemoria = true">
        <Icone nome="documento" />Ver a memória de cálculo ({{ memoria.length }} itens)
      </button>
    </div>

    <div v-if="soltosDepois.some((c) => visivel(c))" class="campos soltos">
      <template v-for="c in soltosDepois" :key="'s' + c.id">
        <CampoPergunta v-if="visivel(c)" :campo="c" :model-value="valores[c.id] ?? ''" :nome="pergunta.id + c.id" prefixo="s-"
          :erro="invalidos[c.id]" @update:model-value="definir(c)($event)" @abrir="repassar" />
      </template>
    </div>

    <div class="barra-acao opcoes">
      <button
        v-for="(o, i) in pergunta.opcoes"
        :key="o.id"
        class="botao"
        :class="{ primario: i === 0 }"
        :disabled="estado.ocupado"
        :data-teste="'opcao-' + o.id"
        @click="escolher(o.id, o.validar)"
      >
        {{ o.rotulo }}
      </button>
      <span v-if="Object.keys(invalidos).length" class="msg-barra" role="status">
        <Icone nome="alerta" :tamanho="16" />{{ Object.keys(invalidos).length }} campo(s) a corrigir
      </span>
    </div>

    <Janela v-if="contextoLongo" :aberta="verContexto" :titulo="pergunta.titulo" @fechar="verContexto = false">
      <p class="texto-ajuda">{{ pergunta.contexto }}</p>
    </Janela>

    <Janela v-if="memoria.length" :aberta="verMemoria" titulo="Memória de cálculo" largura="g" teste="janela-memoria" @fechar="verMemoria = false">
      <div class="itens-memoria">
        <section v-for="g in memoria" :key="g.id" class="item-memoria" :data-teste="'grupo-' + g.id">
          <h3>{{ g.titulo }}</h3>
          <p v-for="(d, i) in g.descricao" :key="i">{{ d }}</p>
        </section>
      </div>
    </Janela>
  </section>
</template>

<style scoped>
.pergunta {
  position: relative;
  margin-top: 1.5rem;
  border: 1px solid var(--linha);
  border-top: 3px solid var(--acento);
  border-radius: var(--r-pasta);
  background: var(--superficie);
  box-shadow: var(--sombra-2);
  padding: 1.3rem 1.5rem 0;
  display: grid;
  gap: 1.1rem;
}
.aba-etapa {
  position: absolute; top: -3px; left: -1px; transform: translateY(-100%);
  background: var(--acento); color: #fff; font-size: var(--t-xs); font-weight: 600;
  padding: 0.25rem 0.9rem 0.2rem; border-radius: 8px 8px 0 0;
  max-width: 80%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.cabeca { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 0.8rem 1.6rem; }
.titulos { display: grid; gap: 0.4rem; flex: 1 1 32rem; min-width: 0; }
.pergunta h2 { font-size: var(--t-lg); }
.contexto { max-width: 110ch; color: var(--tinta-suave); white-space: pre-line; }
.contexto.cortado { display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.comAcusado .contexto.cortado { -webkit-line-clamp: 1; line-clamp: 1; }
.acusado { flex: 0 1 26rem; display: grid; gap: 0.4rem; }

/* tabela das imputações */
.rolagem { overflow-x: auto; }
.tabela-imp { min-width: 960px; border: 1px solid #c9cfc6; border-radius: var(--r-md); overflow: hidden; }
.linha-imp {
  display: grid;
  grid-template-columns: 9rem minmax(0, 1.25fr) minmax(0, 1.35fr) 9.5rem 7.5rem;
  gap: 0.6rem 0.9rem;
  padding: 0.85rem 1rem;
  align-items: start;
  border-top: 1px solid var(--linha);
}
.linha-imp.cab { border-top: 0; background: var(--superficie-2); padding-block: 0.55rem; font-size: var(--t-xs); font-weight: 600; color: var(--tinta-suave); }
.linha-imp.excluida { background: var(--superficie-2); }
.linha-imp.excluida .cel-imp strong { color: var(--tinta-suave); }
.linha-imp.comErro { background: #fffafa; }
.cel, .cel-imp { display: grid; gap: 0.4rem; align-content: start; min-width: 0; }
.cel-imp { padding-top: 0.45rem; justify-items: start; }
.cel-imp strong { color: var(--acento); font-size: var(--t-md); }
.cel-extra { grid-column: 1 / -1; display: grid; gap: 0.5rem; }
.natureza { font-size: var(--t-xs); font-weight: 600; border-radius: 6px; padding: 0 0.45rem; border: 1px solid var(--linha); color: var(--tinta-suave); background: var(--superficie); }
.natureza.grave { background: var(--erro-claro); color: var(--erro); border-color: var(--erro-linha); }
.natureza.média { background: var(--alerta-claro); color: var(--alerta); border-color: var(--alerta-linha); }
.nota { font-size: var(--t-xs); color: var(--tinta-suave); }
.linha-imp :deep(.campo) { flex: none; }
.linha-imp :deep(.l-data input) { width: 100%; }
.linha-imp :deep(.l-data) { max-width: none; }
.linha-imp :deep(.segmentado > label) { padding-inline: 0.7rem; }

/* faixa de pergunta curta (ex.: "Há transgressões?") */
.faixa-pergunta {
  display: flex; flex-wrap: wrap; align-items: flex-end; gap: 1rem 1.5rem;
  border: 1px solid #c9cfc6; border-radius: var(--r-md); background: var(--superficie-2); padding: 0.9rem 1.1rem;
}

/* linhas repetíveis */
.repetidos { display: grid; gap: 0.7rem; }
.tabela-rep { border: 1px solid #c9cfc6; border-radius: var(--r-md); overflow: hidden; }
.linha-rep {
  display: grid;
  grid-template-columns: 2.5rem repeat(var(--n), minmax(0, 1fr)) 2.6rem;
  gap: 0.9rem;
  align-items: center;
  padding: 0.7rem 1rem;
  border-top: 1px solid var(--linha);
}
.linha-rep.cab { border-top: 0; background: var(--superficie-2); padding-block: 0.5rem; font-size: var(--t-xs); font-weight: 600; color: var(--tinta-suave); }
.linha-rep .num { color: var(--tinta-suave); }
.linha-rep :deep(.l-data input) { width: 12rem; max-width: 100%; }
.acrescentar { display: flex; align-items: center; gap: 0.9rem; flex-wrap: wrap; }
.botao.tracejado { border: 1.5px dashed var(--linha-forte); background: transparent; color: var(--acento); font-weight: 600; }
.botao.tracejado:hover { border-color: var(--acento); background: var(--acento-claro); }

/* dosimetria */
.dosimetria { display: grid; gap: 0.9rem; }
.paineis { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 15rem), 1fr)); gap: 0.9rem; }
.painel-dos { border: 1px solid #c9cfc6; border-radius: var(--r-md); padding: 0.95rem 1.1rem; display: grid; gap: 0.25rem; align-content: start; }
.painel-dos.destaque { border: 2px solid var(--acento); background: #fdf8f8; }
.painel-dos.invalido { border-color: var(--erro); }
.rotulo-painel { font-size: var(--t-xs); font-weight: 600; color: var(--tinta-suave); text-transform: uppercase; letter-spacing: 0.02em; }
.destaque .rotulo-painel { color: var(--acento); }
.grande { font-family: var(--serif); font-size: 1.6rem; font-weight: 600; line-height: 1.2; }
.passo { display: flex; align-items: center; gap: 0.5rem; }
.passo .botao { width: 2.9rem; padding: 0; font-size: 1.25rem; }
.passo input { width: 4.5rem; text-align: center; font-size: 1.4rem; font-weight: 600; }
.circunstancias { display: flex; flex-wrap: wrap; gap: 0.5rem 0.6rem; align-items: center; border-top: 1px solid var(--linha); padding-top: 0.8rem; }
.circ { font-size: var(--t-xs); border-radius: 999px; padding: 0.15rem 0.65rem; border: 1px solid; }
.circ.atenuante { background: var(--ok-claro); color: var(--ok); border-color: #c3dfc9; }
.circ.agravante { background: var(--erro-claro); color: var(--erro); border-color: var(--erro-linha); }
.espaco { flex: 1; }

/* seções comuns */
.grupo { display: grid; gap: 0.6rem; border-top: 1px solid var(--linha); padding-top: 0.9rem; min-width: 0; }
.grupo h3 { font-size: var(--t-md); font-family: var(--sans); font-weight: 600; }
.desc { max-width: 110ch; }
.desc strong { font-weight: 600; }
.alertas { margin: 0; padding: 0.55rem 0.8rem; list-style: none; background: var(--alerta-claro); border: 1px solid var(--alerta-linha); color: var(--alerta); border-radius: var(--r-sm); display: grid; gap: 0.3rem; font-size: var(--t-sm); }
.alertas li { display: flex; gap: 0.5rem; align-items: flex-start; }
.alertas li .icone { margin-top: 0.15rem; flex: none; }
.fontes { display: flex; flex-wrap: wrap; gap: 0.35rem; align-items: center; }
.rotulo-fontes { font-size: var(--t-xs); font-weight: 600; color: var(--tinta-suave); margin-right: 0.2rem; }
.fonte { font-size: var(--t-xs); padding: 0.12rem 0.55rem; }
.fonte .icone { color: var(--acento); }
.mais { font-size: var(--t-xs); }
.campos { display: flex; flex-wrap: wrap; gap: 0.9rem 1.3rem; align-items: flex-start; }

.memoria {
  display: flex; flex-wrap: wrap; gap: 0.8rem 1.2rem; align-items: center; justify-content: space-between;
  border: 1px solid var(--linha); border-radius: var(--r-sm); padding: 0.8rem 1rem; background: var(--acento-claro);
}
.proposta { display: grid; gap: 0.2rem; flex: 1 1 24rem; }
.itens-memoria { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 22rem), 1fr)); gap: 0.8rem; }
.item-memoria { border: 1px solid var(--linha); border-radius: var(--r-sm); padding: 0.8rem 1rem; display: grid; gap: 0.35rem; background: var(--superficie-2); align-content: start; }
.item-memoria h3 { font-family: var(--sans); font-size: var(--t-sm); font-weight: 600; }
.item-memoria p { font-size: var(--t-sm); }
.msg { color: var(--erro); font-size: var(--t-sm); }

.opcoes { margin: 0.1rem -1.5rem 0; padding-inline: 1.5rem; border-radius: 0 0 var(--r-md) var(--r-md); }
.msg-barra { display: inline-flex; gap: 0.35rem; align-items: center; color: var(--erro); font-size: var(--t-sm); margin-left: auto; }

@media (max-width: 640px) {
  .pergunta { padding: 1.1rem 1rem 0; }
  .opcoes { margin-inline: -1rem; padding-inline: 1rem; }
  .linha-rep { grid-template-columns: 2rem 1fr 2.6rem; }
  .linha-rep > :nth-child(n+3):not(:last-child) { grid-column: 2; }
}
</style>

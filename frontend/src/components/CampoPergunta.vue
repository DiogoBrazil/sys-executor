<script setup lang="ts">
// Um campo de pergunta (todos os tipos do Go). Usado no formulário comum e
// nas tabelas (imputações, histórico), onde o rótulo fica só para leitores
// de tela (semRotulo). Os data-teste são os mesmos de antes.
import { computed } from "vue";
import type { Campo, Fonte } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";
import CampoBusca from "./CampoBusca.vue";
import Icone from "./ui/Icone.vue";

const props = withDefaults(
  defineProps<{ campo: Campo; modelValue: string; nome: string; prefixo?: string; erro?: string; semRotulo?: boolean; linhas?: number; segmentado?: boolean }>(),
  { prefixo: "c-", erro: "", semRotulo: false, linhas: 3, segmentado: false },
);
const emit = defineEmits<{ "update:modelValue": [v: string]; abrir: [pagina: number, inicio: number, fim: number] }>();

const c = computed(() => props.campo);
const id = computed(() => props.prefixo + c.value.id);
const opcoes = computed(() => {
  if (c.value.tipo === "SIM_NAO" && !(c.value.opcoes ?? []).length) {
    return [
      { id: "sim", rotulo: "Sim" },
      { id: "nao", rotulo: "Não" },
    ];
  }
  return c.value.opcoes ?? [];
});
const emRadio = computed(() => c.value.tipo === "SIM_NAO" || (c.value.tipo === "ESCOLHA" && opcoes.value.length <= 4));
// Opções curtas: controle segmentado; longas: cartões de escolha, um por linha.
const curtas = computed(() => props.segmentado || opcoes.value.reduce((n, o) => n + o.rotulo.length, 0) <= 40);
// Largura pelo tipo: datas e números estreitos; texto longo e busca, a linha toda.
const largura = computed(() => {
  const t = c.value.tipo;
  if (t === "DATA") return "data";
  if (t === "NUMERO" || /HH:MM/.test(c.value.ajuda ?? "")) return "curto";
  if (t === "TEXTO_LONGO" || t === "BUSCA" || (emRadio.value && !curtas.value)) return "largo";
  if (t === "TEXTO" || t === "ESCOLHA") return "medio";
  return "auto";
});
const rotuloFonte = (f: Fonte) => `${f.rotulo || "Autos"} — pág. ${f.pagina}`;
const valor = (e: Event) => (e.target as HTMLInputElement).value;
</script>

<template>
  <div class="campo" :class="['l-' + largura, { invalido: erro }]">
    <fieldset v-if="emRadio">
      <legend class="rotulo" :class="{ 'visualmente-oculto': semRotulo }">{{ c.rotulo }}<span v-if="c.obrigatorio" aria-hidden="true"> *</span></legend>
      <div :class="curtas ? 'segmentado' : 'escolhas'">
        <label v-for="o in opcoes" :key="o.id" :class="{ marcado: modelValue === o.id }">
          <input type="radio" :class="{ sobre: !curtas }" :name="nome" :value="o.id" :checked="modelValue === o.id"
            :data-teste="'campo-' + c.id + '-' + o.id" @change="emit('update:modelValue', o.id)" />
          {{ o.rotulo }}
        </label>
      </div>
    </fieldset>
    <template v-else>
      <label class="rotulo" :class="{ 'visualmente-oculto': semRotulo }" :for="id">{{ c.rotulo }}<span v-if="c.obrigatorio" aria-hidden="true"> *</span></label>
      <select v-if="c.tipo === 'ESCOLHA'" :id="id" :value="modelValue" :data-teste="'campo-' + c.id" @change="emit('update:modelValue', valor($event))">
        <option value="">Escolha…</option>
        <option v-for="o in opcoes" :key="o.id" :value="o.id">{{ o.rotulo }}</option>
      </select>
      <CampoBusca v-else-if="c.tipo === 'BUSCA'" :id="id" :model-value="modelValue" :opcoes="opcoes" :teste="'campo-' + c.id"
        @update:model-value="(v: string) => emit('update:modelValue', v)" />
      <textarea v-else-if="c.tipo === 'TEXTO_LONGO'" :id="id" :value="modelValue" :rows="linhas" :data-teste="'campo-' + c.id"
        @input="emit('update:modelValue', valor($event))"></textarea>
      <input
        v-else
        :id="id"
        :value="modelValue"
        :inputmode="c.tipo === 'NUMERO' ? 'numeric' : undefined"
        :placeholder="c.tipo === 'DATA' ? 'DD/MM/AAAA' : /HH:MM/.test(c.ajuda ?? '') ? 'HH:MM' : undefined"
        :data-teste="'campo-' + c.id"
        @input="emit('update:modelValue', valor($event))"
      />
    </template>
    <p v-if="c.ajuda && c.ajuda !== 'DD/MM/AAAA' && c.ajuda !== 'HH:MM'" class="ajuda">{{ c.ajuda }}</p>
    <div v-if="c.fontes?.length" class="fontes">
      <button v-for="(f, i) in c.fontes" :key="i" type="button" class="chip fonte" :title="f.trecho" @click="emit('abrir', f.pagina, f.inicio, f.fim)">
        <Icone nome="pdf" :tamanho="14" /><span class="visualmente-oculto">Ver no PDF: </span>{{ rotuloFonte(f) }}
      </button>
    </div>
    <p v-if="erro" class="msg" role="alert">{{ erro }}</p>
  </div>
</template>

<style scoped>
.campo { display: grid; gap: 0.35rem; align-content: start; min-width: 0; }
.campo fieldset { border: 0; padding: 0; margin: 0; min-width: 0; }
.campo legend { margin-bottom: 0.4rem; padding: 0; }
.rotulo { font-weight: 600; font-size: var(--t-sm); color: var(--tinta); }
.l-data, .l-curto { flex: 0 1 auto; max-width: 24rem; }
.l-data input { width: 11.5rem; }
.l-curto input { width: 9rem; }
.l-medio { flex: 1 1 20rem; }
.l-medio input, .l-medio select { max-width: 34rem; }
.l-largo { flex: 1 1 100%; }
.l-auto { flex: 0 0 auto; }
.ajuda { font-size: var(--t-xs); color: var(--tinta-suave); }
.msg { color: var(--erro); font-size: var(--t-sm); }
.invalido input:not([type="radio"]), .invalido select, .invalido textarea { border-color: var(--erro); border-width: 2px; box-shadow: 0 0 0 3px rgba(155, 28, 28, 0.12); }
.invalido .segmentado, .invalido .escolhas > label { border-color: var(--erro); }
.fontes { display: flex; flex-wrap: wrap; gap: 0.35rem; }
.fonte { font-size: var(--t-xs); padding: 0.12rem 0.55rem; }
.fonte .icone { color: var(--acento); }
</style>

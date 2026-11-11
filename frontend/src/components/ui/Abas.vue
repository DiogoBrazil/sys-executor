<script setup lang="ts">
// Abas acessíveis (role=tablist, setas do teclado). Os painéis são slots
// nomeados pelo id da aba e ficam no DOM (v-show), só um visível.
import { nextTick, onMounted, ref, watch } from "vue";

export interface Aba { id: string; rotulo: string; contador?: number | null; teste?: string }
const props = defineProps<{ abas: Aba[]; modelValue: string }>();
const emit = defineEmits<{ "update:modelValue": [id: string] }>();

const lista = ref<HTMLElement | null>(null);
const barra = ref({ left: 0, width: 0 });
function medir() {
  const el = lista.value?.querySelector<HTMLElement>(`[data-aba="${props.modelValue}"]`);
  if (el) barra.value = { left: el.offsetLeft, width: el.offsetWidth };
}
onMounted(medir);
watch(() => [props.modelValue, props.abas.map((a) => a.contador).join()], () => nextTick(medir));

function teclar(e: KeyboardEvent, i: number) {
  const n = props.abas.length;
  let j = -1;
  if (e.key === "ArrowRight") j = (i + 1) % n;
  else if (e.key === "ArrowLeft") j = (i - 1 + n) % n;
  else if (e.key === "Home") j = 0;
  else if (e.key === "End") j = n - 1;
  if (j < 0) return;
  e.preventDefault();
  emit("update:modelValue", props.abas[j].id);
  nextTick(() => lista.value?.querySelector<HTMLElement>(`[data-aba="${props.abas[j].id}"]`)?.focus());
}
</script>

<template>
  <div class="abas">
    <div ref="lista" class="lista" role="tablist">
      <button v-for="(a, i) in abas" :key="a.id" type="button" role="tab" class="aba" :class="{ atual: a.id === modelValue }"
        :data-aba="a.id" :data-teste="a.teste" :aria-selected="a.id === modelValue" :tabindex="a.id === modelValue ? 0 : -1"
        :aria-controls="'painel-' + a.id" @click="emit('update:modelValue', a.id)" @keydown="teclar($event, i)">
        {{ a.rotulo }}<span v-if="a.contador" class="conta">{{ a.contador }}</span>
      </button>
      <span class="barra" :style="{ transform: `translateX(${barra.left}px)`, width: barra.width + 'px' }" aria-hidden="true"></span>
    </div>
    <div v-for="a in abas" v-show="a.id === modelValue" :id="'painel-' + a.id" :key="a.id" role="tabpanel" class="painel">
      <slot :name="a.id" />
    </div>
  </div>
</template>

<style scoped>
.abas { display: grid; gap: 0.9rem; }
.lista { position: relative; display: flex; gap: 0.2rem; border-bottom: 1px solid var(--linha); overflow-x: auto; }
.aba {
  border: 0; background: none; padding: 0.6rem 0.85rem; font-weight: 500; color: var(--tinta-suave);
  display: inline-flex; align-items: center; gap: 0.45rem; white-space: nowrap; border-radius: var(--r-sm) var(--r-sm) 0 0;
  transition: color var(--dur-1) var(--ease), background var(--dur-1) var(--ease);
}
.aba:hover { color: var(--tinta); background: rgba(255, 255, 255, 0.5); }
.aba.atual { color: var(--tinta); font-weight: 600; }
.conta { font-size: var(--t-xs); background: var(--superficie); border: 1px solid var(--linha); border-radius: 999px; padding: 0 0.45rem; color: var(--tinta-suave); }
.barra { position: absolute; bottom: -1px; left: 0; height: 3px; border-radius: 3px 3px 0 0; background: var(--acento); transition: transform var(--dur-3) var(--ease), width var(--dur-3) var(--ease); }
</style>

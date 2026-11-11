<script setup lang="ts">
// Seção que abre e fecha com altura animada (grid 0fr → 1fr). O conteúdo
// fica no DOM mesmo fechado (inert), para leitura por testes e para não
// perder o estado dos campos.
import { ref, useId, watch } from "vue";
import Icone from "./Icone.vue";

const props = withDefaults(defineProps<{ titulo: string; contador?: number | null; aberto?: boolean; teste?: string }>(), {
  contador: null, aberto: false, teste: undefined,
});
const emit = defineEmits<{ alternou: [aberto: boolean] }>();
const aberta = ref(props.aberto);
watch(() => props.aberto, (v) => (aberta.value = v));
function alternar() {
  aberta.value = !aberta.value;
  emit("alternou", aberta.value);
}
const id = "rec-" + useId();
</script>

<template>
  <section class="recolhivel" :class="{ aberta }" :data-teste="teste">
    <button type="button" class="cab" :aria-expanded="aberta" :aria-controls="id" @click="alternar">
      <Icone nome="chevron" :tamanho="16" class="seta" />
      <span class="titulo"><slot name="titulo">{{ titulo }}</slot></span>
      <span v-if="contador !== null" class="contador">{{ contador }}</span>
    </button>
    <div :id="id" class="corpo" :inert="!aberta || undefined">
      <div class="dentro"><slot /></div>
    </div>
  </section>
</template>

<style scoped>
.recolhivel { display: grid; }
.cab {
  display: flex; align-items: center; gap: 0.45rem;
  border: 0; background: none; padding: 0.35rem 0; text-align: left;
  font-weight: 600; font-size: var(--t-sm); color: var(--tinta);
}
.cab:hover .titulo { text-decoration: underline; text-underline-offset: 3px; }
.seta { transition: transform var(--dur-2) var(--ease); color: var(--tinta-suave); }
.aberta .seta { transform: rotate(90deg); }
.contador {
  font-size: var(--t-xs); font-weight: 600; color: var(--tinta-suave);
  background: var(--superficie-2); border: 1px solid var(--linha); border-radius: 999px; padding: 0 0.45rem;
}
.corpo { display: grid; grid-template-rows: 0fr; transition: grid-template-rows var(--dur-3) var(--ease), opacity var(--dur-2) var(--ease); opacity: 0; }
.aberta .corpo { grid-template-rows: 1fr; opacity: 1; }
.dentro { overflow: hidden; min-height: 0; }
.aberta .dentro { padding-top: 0.4rem; }
</style>

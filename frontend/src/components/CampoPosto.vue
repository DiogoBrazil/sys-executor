<script setup lang="ts">
// Posto/graduação da PMRO em lista (pedido do usuário, 12/11/2026). Um valor
// antigo fora da lista (cadastro anterior) continua como opção, para não se
// perder ao abrir a tela.
import { computed } from "vue";
import { POSTOS } from "../postos";

const props = defineProps<{ modelValue: string; rotulo?: string; teste?: string }>();
const emit = defineEmits<{ "update:modelValue": [v: string] }>();
const opcoes = computed(() => (props.modelValue && !POSTOS.includes(props.modelValue) ? [props.modelValue, ...POSTOS] : POSTOS));
</script>

<template>
  <select :value="modelValue" :aria-label="rotulo ?? 'Posto/graduação'" :data-teste="teste"
    @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)">
    <option value="">Selecione</option>
    <option v-for="o in opcoes" :key="o" :value="o">{{ o }}</option>
  </select>
</template>

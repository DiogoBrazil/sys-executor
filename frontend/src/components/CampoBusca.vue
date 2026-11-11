<script setup lang="ts">
// Escolha numa lista longa com filtro por digitação (ex.: dispositivos do
// catálogo de enquadramento). O valor é sempre o ID de uma das opções.
import { computed, ref } from "vue";

type Opcao = { id: string; rotulo: string };
const props = defineProps<{ id: string; opcoes: Opcao[]; teste: string }>();
const valor = defineModel<string>({ default: "" });
const filtro = ref("");
const MAX = 12;

const normal = (s: string) => s.normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase();
const escolhida = computed(() => props.opcoes.find((o) => o.id === valor.value) ?? null);
const encontradas = computed(() => {
  const termos = normal(filtro.value).split(/\s+/).filter(Boolean);
  const lista = termos.length ? props.opcoes.filter((o) => termos.every((t) => normal(o.rotulo).includes(t))) : props.opcoes;
  return lista.slice(0, MAX);
});
const total = computed(() => {
  const termos = normal(filtro.value).split(/\s+/).filter(Boolean);
  return termos.length ? props.opcoes.filter((o) => termos.every((t) => normal(o.rotulo).includes(t))).length : props.opcoes.length;
});

function escolher(o: Opcao) {
  valor.value = o.id;
  filtro.value = "";
}
</script>

<template>
  <div class="busca" :data-teste="teste">
    <div v-if="escolhida" class="escolhida" :data-teste="teste + '-escolhida'">
      <span>{{ escolhida.rotulo }}</span>
      <button type="button" class="botao discreto" :data-teste="teste + '-trocar'" @click="valor = ''">Trocar</button>
    </div>
    <template v-else>
      <input
        :id="id"
        v-model="filtro"
        type="search"
        autocomplete="off"
        placeholder="Digite para filtrar (artigo, inciso ou palavra)"
        :data-teste="teste + '-filtro'"
      />
      <ul class="lista" role="listbox">
        <li v-for="o in encontradas" :key="o.id">
          <button type="button" role="option" :data-teste="teste + '-opcao-' + o.id" @click="escolher(o)">{{ o.rotulo }}</button>
        </li>
      </ul>
      <p v-if="total > encontradas.length" class="mais">Mostrando {{ encontradas.length }} de {{ total }}; continue digitando para filtrar.</p>
      <p v-if="!total" class="mais">Nenhum dispositivo com esse texto.</p>
    </template>
  </div>
</template>

<style scoped>
.busca { display: grid; gap: 0.35rem; }

.lista { list-style: none; margin: 0; padding: 0; max-height: 16rem; overflow: auto; border: 1px solid var(--linha); border-radius: var(--raio); }
.lista button { display: block; width: 100%; text-align: left; font: inherit; font-size: var(--t-sm); background: none; border: 0; border-bottom: 1px solid var(--linha); padding: 0.4rem 0.6rem; cursor: pointer; }
.lista button:hover, .lista button:focus { background: var(--papel); }
.escolhida { display: flex; gap: 0.6rem; align-items: center; justify-content: space-between; border: 1px solid var(--ok); border-radius: var(--raio); padding: 0.45rem 0.6rem; }
.mais { font-size: var(--t-xs); color: var(--tinta-suave); }
</style>

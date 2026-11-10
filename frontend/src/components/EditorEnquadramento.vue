<script setup lang="ts">
// Enquadramento manual de um fato (V4 §48.5; E7): cada dispositivo vira uma
// imputação. RDPM arts. 15–17 com natureza do artigo; Estatuto arts. 29 e 32
// com natureza atribuída pelo usuário. O aplicativo não sugere dispositivo.
import { computed } from "vue";
import type { DispositivoVisao } from "../api";
import CampoBusca from "./CampoBusca.vue";

export interface LinhaEnquadramento {
  diploma: string;
  dispositivoId: string;
  natureza: string;
}

const props = defineProps<{ linhas: LinhaEnquadramento[]; catalogo: DispositivoVisao[]; teste: string }>();
const emit = defineEmits<{ mudou: [] }>();

const NATUREZAS = [
  { id: "LEVE", rotulo: "Leve" },
  { id: "MEDIA", rotulo: "Média" },
  { id: "GRAVE", rotulo: "Grave" },
];
const rotuloNatureza = (n: string) => NATUREZAS.find((x) => x.id === n)?.rotulo ?? "";

const opcoes = computed(() => ({
  RDPM: props.catalogo.filter((d) => d.diploma === "RDPM").map((d) => ({ id: d.id, rotulo: `${d.rotulo} (${rotuloNatureza(d.natureza)})` })),
  ESTATUTO: props.catalogo.filter((d) => d.diploma === "ESTATUTO").map((d) => ({ id: d.id, rotulo: d.rotulo })),
}));

function trocarDiploma(l: LinhaEnquadramento, d: string) {
  if (l.diploma !== d) {
    l.diploma = d;
    l.dispositivoId = "";
    l.natureza = "";
  }
  emit("mudou");
}
function acrescentar() {
  props.linhas.push({ diploma: "", dispositivoId: "", natureza: "" });
  emit("mudou");
}
function retirar(i: number) {
  props.linhas.splice(i, 1);
  emit("mudou");
}
</script>

<template>
  <div class="enq" :data-teste="teste">
    <div v-for="(l, i) in linhas" :key="i" class="linha" :data-teste="teste + '-linha'">
      <fieldset class="diploma">
        <legend>Dispositivo {{ i + 1 }}: diploma</legend>
        <label><input type="radio" :name="teste + '-dip-' + i" value="RDPM" :checked="l.diploma === 'RDPM'" :data-teste="teste + '-rdpm-' + i" @change="trocarDiploma(l, 'RDPM')" /> RDPM (arts. 15 a 17)</label>
        <label><input type="radio" :name="teste + '-dip-' + i" value="ESTATUTO" :checked="l.diploma === 'ESTATUTO'" :data-teste="teste + '-est-' + i" @change="trocarDiploma(l, 'ESTATUTO')" /> Estatuto (arts. 29 e 32)</label>
      </fieldset>
      <CampoBusca
        v-if="l.diploma"
        :id="teste + '-disp-' + i"
        v-model="l.dispositivoId"
        :opcoes="l.diploma === 'RDPM' ? opcoes.RDPM : opcoes.ESTATUTO"
        :teste="teste + '-disp-' + i"
        @update:model-value="emit('mudou')"
      />
      <label v-if="l.diploma === 'ESTATUTO'" class="natureza">
        Natureza atribuída
        <select v-model="l.natureza" :data-teste="teste + '-nat-' + i" @change="emit('mudou')">
          <option value="">Escolha…</option>
          <option v-for="n in NATUREZAS" :key="n.id" :value="n.id">{{ n.rotulo }}</option>
        </select>
        <small>O Estatuto não define a natureza: atribua-a pela semelhança de intensidade com as transgressões do RDPM.</small>
      </label>
      <div><button type="button" class="botao discreto" @click="retirar(i)">Retirar este dispositivo</button></div>
    </div>
    <div>
      <button type="button" class="botao" :data-teste="teste + '-mais'" @click="acrescentar">Acrescentar dispositivo</button>
    </div>
  </div>
</template>

<style scoped>
.enq { display: grid; gap: 0.8rem; }
.linha { display: grid; gap: 0.5rem; padding: 0.8rem; border: 1px solid var(--linha); border-radius: var(--raio); background: #fff; }
.diploma { border: 0; padding: 0; margin: 0; display: flex; gap: 1.2rem; flex-wrap: wrap; }
.diploma legend { font-weight: 600; margin-bottom: 0.3rem; }
.natureza { display: grid; gap: 0.25rem; max-width: 34rem; }
.natureza select { font: inherit; padding: 0.35rem; max-width: 14rem; }
.natureza small { color: var(--tinta-suave); }
</style>

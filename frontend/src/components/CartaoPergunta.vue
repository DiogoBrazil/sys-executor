<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { estado, responder } from "../estado";
import type { Campo, Fonte, GrupoCampos, PerguntaVisao } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";

const props = defineProps<{ pergunta: PerguntaVisao }>();
const emit = defineEmits<{ abrir: [pagina: number, inicio: number, fim: number] }>();

const valores = reactive<Record<string, string>>({});
const invalidos = ref<Record<string, string>>({});

function iniciar() {
  for (const k of Object.keys(valores)) delete valores[k];
  for (const c of props.pergunta.campos ?? []) valores[c.id] = c.valor ?? "";
  invalidos.value = {};
}
watch(() => props.pergunta.id, iniciar, { immediate: true });

const campos = computed(() => props.pergunta.campos ?? []);
const grupos = computed(() => props.pergunta.grupos ?? []);
const soltos = computed(() => campos.value.filter((c) => !c.grupo || !grupos.value.some((g) => g.id === c.grupo)));
const doGrupo = (g: GrupoCampos) => campos.value.filter((c) => c.grupo === g.id);

function visivel(c: Campo): boolean {
  return !c.mostrarSe || valores[c.mostrarSe.campo] === c.mostrarSe.valor;
}

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

async function escolher(opcaoID: string, validarCampos: boolean) {
  if (validarCampos && !validar()) return;
  const enviados: Record<string, string> = {};
  if (validarCampos) {
    for (const c of campos.value) if (visivel(c)) enviados[c.id] = valores[c.id] ?? "";
  }
  await responder(props.pergunta.id, opcaoID, enviados);
}

function opcoesDe(c: Campo) {
  if (c.tipo === "SIM_NAO" && !(c.opcoes ?? []).length) {
    return [
      { id: "sim", rotulo: "Sim" },
      { id: "nao", rotulo: "Não" },
    ];
  }
  return c.opcoes ?? [];
}

function rotuloFonte(f: Fonte) {
  return `${f.rotulo || "Autos"} — pág. ${f.pagina}`;
}
</script>

<template>
  <section class="pergunta" role="group" :aria-labelledby="'perg-' + pergunta.id" data-teste="pergunta">
    <h2 :id="'perg-' + pergunta.id">{{ pergunta.titulo }}</h2>
    <p class="contexto">{{ pergunta.contexto }}</p>

    <section v-for="g in grupos" :key="g.id" class="grupo" :data-teste="'grupo-' + g.id">
      <h3>{{ g.titulo }}</h3>
      <p v-for="(d, i) in g.descricao" :key="i" class="desc">{{ d }}</p>
      <ul v-if="g.alertas?.length" class="alertas" role="list">
        <li v-for="(a, i) in g.alertas" :key="i">{{ a }}</li>
      </ul>
      <div v-if="g.fontes?.length" class="fontes">
        <button
          v-for="(f, i) in g.fontes"
          :key="i"
          type="button"
          class="fonte"
          :title="f.trecho"
          :data-teste="'fonte-' + g.id + '-' + i"
          @click="emit('abrir', f.pagina, f.inicio, f.fim)"
        >
          Ver no PDF: {{ rotuloFonte(f) }}
        </button>
      </div>
      <template v-for="c in doGrupo(g)" :key="c.id">
        <div v-if="visivel(c)" class="campo" :class="{ invalido: invalidos[c.id] }">
          <template v-if="c.tipo === 'SIM_NAO' || (c.tipo === 'ESCOLHA' && opcoesDe(c).length <= 4)">
            <fieldset>
              <legend class="rotulo">{{ c.rotulo }}<span v-if="c.obrigatorio" aria-hidden="true"> *</span></legend>
              <label v-for="o in opcoesDe(c)" :key="o.id" class="radio">
                <input v-model="valores[c.id]" type="radio" :name="pergunta.id + c.id" :value="o.id" :data-teste="'campo-' + c.id + '-' + o.id" />
                {{ o.rotulo }}
              </label>
            </fieldset>
          </template>
          <template v-else>
            <label class="rotulo" :for="'c-' + c.id">{{ c.rotulo }}<span v-if="c.obrigatorio" aria-hidden="true"> *</span></label>
            <select v-if="c.tipo === 'ESCOLHA'" :id="'c-' + c.id" v-model="valores[c.id]" :data-teste="'campo-' + c.id">
              <option value="">Escolha…</option>
              <option v-for="o in opcoesDe(c)" :key="o.id" :value="o.id">{{ o.rotulo }}</option>
            </select>
            <textarea v-else-if="c.tipo === 'TEXTO_LONGO'" :id="'c-' + c.id" v-model="valores[c.id]" rows="3" :data-teste="'campo-' + c.id"></textarea>
            <input
              v-else
              :id="'c-' + c.id"
              v-model="valores[c.id]"
              :inputmode="c.tipo === 'NUMERO' ? 'numeric' : undefined"
              :placeholder="c.tipo === 'DATA' ? 'DD/MM/AAAA' : undefined"
              :data-teste="'campo-' + c.id"
            />
          </template>
          <p v-if="c.ajuda" class="ajuda">{{ c.ajuda }}</p>
          <div v-if="c.fontes?.length" class="fontes">
            <button v-for="(f, i) in c.fontes" :key="i" type="button" class="fonte" @click="emit('abrir', f.pagina, f.inicio, f.fim)">
              Ver no PDF: {{ rotuloFonte(f) }}
            </button>
          </div>
          <p v-if="invalidos[c.id]" class="msg" role="alert">{{ invalidos[c.id] }}</p>
        </div>
      </template>
    </section>

    <template v-for="c in soltos" :key="'s' + c.id">
      <div v-if="visivel(c)" class="campo" :class="{ invalido: invalidos[c.id] }">
        <fieldset v-if="c.tipo === 'SIM_NAO' || (c.tipo === 'ESCOLHA' && opcoesDe(c).length <= 4)">
          <legend class="rotulo">{{ c.rotulo }}<span v-if="c.obrigatorio" aria-hidden="true"> *</span></legend>
          <label v-for="o in opcoesDe(c)" :key="o.id" class="radio">
            <input v-model="valores[c.id]" type="radio" :name="pergunta.id + c.id" :value="o.id" :data-teste="'campo-' + c.id + '-' + o.id" />
            {{ o.rotulo }}
          </label>
        </fieldset>
        <template v-else>
          <label class="rotulo" :for="'s-' + c.id">{{ c.rotulo }}<span v-if="c.obrigatorio" aria-hidden="true"> *</span></label>
          <select v-if="c.tipo === 'ESCOLHA'" :id="'s-' + c.id" v-model="valores[c.id]" :data-teste="'campo-' + c.id">
            <option value="">Escolha…</option>
            <option v-for="o in opcoesDe(c)" :key="o.id" :value="o.id">{{ o.rotulo }}</option>
          </select>
          <textarea v-else-if="c.tipo === 'TEXTO_LONGO'" :id="'s-' + c.id" v-model="valores[c.id]" rows="3" :data-teste="'campo-' + c.id"></textarea>
          <input v-else :id="'s-' + c.id" v-model="valores[c.id]" :placeholder="c.tipo === 'DATA' ? 'DD/MM/AAAA' : undefined" :data-teste="'campo-' + c.id" />
        </template>
        <p v-if="c.ajuda" class="ajuda">{{ c.ajuda }}</p>
        <p v-if="invalidos[c.id]" class="msg" role="alert">{{ invalidos[c.id] }}</p>
      </div>
    </template>

    <div class="opcoes">
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
    </div>
  </section>
</template>

<style scoped>
.pergunta {
  border: 2px solid var(--tinta);
  border-radius: var(--raio);
  background: var(--superficie);
  padding: 1.3rem 1.4rem;
  display: grid;
  gap: 0.9rem;
}
.pergunta h2 { font-size: var(--t-lg); }
.contexto { max-width: 68ch; color: var(--tinta-suave); white-space: pre-line; }
.campo-solto:empty { display: none; }
.grupo {
  border-top: 1px solid var(--linha);
  padding-top: 0.9rem;
  display: grid;
  gap: 0.6rem;
}
.grupo h3 { font-size: var(--t-md); font-family: var(--sans); font-weight: 600; }
.desc { max-width: 72ch; }
.alertas { margin: 0; padding: 0.6rem 0.8rem 0.6rem 1.8rem; background: var(--alerta-claro); color: var(--alerta); border-radius: var(--raio); }
.fontes { display: flex; flex-wrap: wrap; gap: 0.4rem; }
.fonte {
  border: 1px solid var(--linha-forte);
  background: var(--papel);
  border-radius: 999px;
  padding: 0.15rem 0.7rem;
  font-size: var(--t-sm);
  cursor: pointer;
  color: var(--tinta);
}
.fonte:hover, .fonte:focus-visible { border-color: var(--tinta); }
.campo { display: grid; gap: 0.3rem; max-width: 640px; }
.campo fieldset { border: 0; padding: 0; margin: 0; display: flex; flex-wrap: wrap; gap: 0.4rem 1.2rem; }
.campo legend { margin-bottom: 0.3rem; }
.rotulo { font-weight: 600; font-size: var(--t-sm); }
.radio { display: inline-flex; gap: 0.35rem; align-items: center; }
.campo input:not([type="radio"]), .campo select, .campo textarea {
  font: inherit;
  padding: 0.45rem 0.6rem;
  border: 1px solid var(--linha-forte);
  border-radius: var(--raio);
  background: #fff;
  color: var(--tinta);
}
.campo textarea { resize: vertical; }
.invalido input:not([type="radio"]), .invalido select, .invalido textarea { border-color: var(--erro); }
.ajuda { font-size: var(--t-xs); color: var(--tinta-suave); }
.msg { color: var(--erro); font-size: var(--t-sm); }
.opcoes { display: flex; flex-wrap: wrap; gap: 0.7rem; margin-top: 0.3rem; }
</style>

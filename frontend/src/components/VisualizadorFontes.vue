<script setup lang="ts">
// Visualizador dos autos (UI-SRC-01): abre a página e realça o trecho pelas
// posições do pdfium. Sem posições, abre só a página (V4 §16.4).
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import * as pdfjs from "pdfjs-dist";
import workerUrl from "pdfjs-dist/build/pdf.worker.min.mjs?url";
import { fontes } from "../api";

pdfjs.GlobalWorkerOptions.workerSrc = workerUrl;

const props = defineProps<{ pagina: number; inicio: number; fim: number }>();
const emit = defineEmits<{ fechar: [] }>();

interface Realce {
  x: number;
  y: number;
  w: number;
  h: number;
}

const canvas = ref<HTMLCanvasElement | null>(null);
const area = ref<HTMLDivElement | null>(null);
const fecharBtn = ref<HTMLButtonElement | null>(null);
const paginaAtual = ref(props.pagina);
const total = ref(0);
const realces = ref<Realce[]>([]);
const erro = ref("");
const consulta = ref("");
const resultados = ref<{ pagina: number; inicio: number; fim: number; trecho: string }[]>([]);
const buscou = ref(false);

let doc: pdfjs.PDFDocumentProxy | null = null;
let carga: pdfjs.PDFDocumentLoadingTask | null = null;
let tarefaRender: pdfjs.RenderTask | null = null;
let escala = 1;
// Último trecho mostrado, para redesenhar quando a largura da área muda
// (ex.: a lista de resultados da busca aparece) sem perder o realce.
let atual = { n: props.pagina, inicio: props.inicio, fim: props.fim };
let observador: ResizeObserver | null = null;
let larguraAnterior = 0;
let espera: ReturnType<typeof setTimeout> | undefined;

async function carregar() {
  const end = await fontes.enderecoPDF();
  if (end.erro) {
    erro.value = end.erro.mensagem;
    return;
  }
  carga = pdfjs.getDocument({
    url: end.endereco,
    enableXfa: false,
    standardFontDataUrl: "/pdfjs/standard_fonts/",
    wasmUrl: "/pdfjs/wasm/",
  });
  doc = await carga.promise;
  total.value = doc.numPages;
}

async function mostrar(n: number, inicio = 0, fim = 0) {
  if (!doc || n < 1 || n > total.value) return;
  atual = { n, inicio, fim };
  paginaAtual.value = n;
  const pagina = await doc.getPage(n);
  const base = pagina.getViewport({ scale: 1 });
  const largura = Math.max(320, (area.value?.clientWidth ?? 800) - 32);
  escala = largura / base.width;
  const vp = pagina.getViewport({ scale: escala });
  const dpr = window.devicePixelRatio || 1;
  const c = canvas.value!;
  c.width = Math.floor(vp.width * dpr);
  c.height = Math.floor(vp.height * dpr);
  c.style.width = `${Math.floor(vp.width)}px`;
  c.style.height = `${Math.floor(vp.height)}px`;
  tarefaRender?.cancel();
  tarefaRender = pagina.render({ canvas: c, viewport: vp, transform: dpr !== 1 ? [dpr, 0, 0, dpr, 0, 0] : undefined });
  try {
    await tarefaRender.promise;
  } catch {
    return; // renderização substituída por outra
  }
  realces.value = [];
  if (fim > inicio) {
    const d = await fontes.destaque(n, inicio, fim);
    if (!d.erro && d.largura > 0) {
      const k = vp.width / d.largura;
      realces.value = (d.caixas ?? []).map((cx) => ({
        x: cx.x0 * k - 2,
        y: cx.y0 * k - 2,
        w: (cx.x1 - cx.x0) * k + 4,
        h: (cx.y1 - cx.y0) * k + 4,
      }));
      await nextTick();
      area.value?.querySelector(".realce")?.scrollIntoView({ block: "center" });
    }
  } else {
    area.value?.scrollTo({ top: 0 });
  }
}

async function buscar() {
  buscou.value = true;
  const r = await fontes.buscar(consulta.value);
  resultados.value = r.erro ? [] : (r.ocorrencias ?? []);
}

function tecla(e: KeyboardEvent) {
  if (e.key === "Escape") emit("fechar");
}

onMounted(async () => {
  document.addEventListener("keydown", tecla);
  if (area.value && typeof ResizeObserver !== "undefined") {
    observador = new ResizeObserver(() => {
      const w = area.value?.clientWidth ?? 0;
      if (Math.abs(w - larguraAnterior) < 4) return;
      larguraAnterior = w;
      clearTimeout(espera);
      espera = setTimeout(() => void mostrar(atual.n, atual.inicio, atual.fim), 120);
    });
    observador.observe(area.value);
  }
  fecharBtn.value?.focus();
  try {
    await carregar();
    await mostrar(props.pagina, props.inicio, props.fim);
  } catch {
    erro.value = "Não foi possível abrir os autos no visualizador.";
  }
});

watch(
  () => [props.pagina, props.inicio, props.fim],
  ([p, i, f]) => mostrar(p, i, f),
);

onBeforeUnmount(() => {
  document.removeEventListener("keydown", tecla);
  observador?.disconnect();
  clearTimeout(espera);
  tarefaRender?.cancel();
  void carga?.destroy();
  carga = null;
  doc = null;
});
</script>

<template>
  <div class="fundo" @click.self="emit('fechar')">
    <div class="janela" role="dialog" aria-modal="true" aria-label="Autos do processo" data-teste="visualizador">
      <header class="barra">
        <div class="nav">
          <button class="botao" :disabled="paginaAtual <= 1" @click="mostrar(paginaAtual - 1)">Página anterior</button>
          <span aria-live="polite">Página {{ paginaAtual }} de {{ total || "…" }}</span>
          <button class="botao" :disabled="paginaAtual >= total" @click="mostrar(paginaAtual + 1)">Próxima página</button>
        </div>
        <form class="busca" role="search" @submit.prevent="buscar">
          <label class="visualmente-oculto" for="busca-autos">Buscar nos autos</label>
          <input id="busca-autos" v-model="consulta" type="search" placeholder="Buscar nos autos" />
          <button class="botao" type="submit">Buscar</button>
        </form>
        <button ref="fecharBtn" class="botao" @click="emit('fechar')">Fechar</button>
      </header>

      <div class="corpo">
        <aside v-if="buscou" class="resultados" aria-label="Resultados da busca">
          <p v-if="!resultados.length">Nada encontrado.</p>
          <button v-for="(r, i) in resultados" :key="i" class="resultado" @click="mostrar(r.pagina, r.inicio, r.fim)">
            <strong>p. {{ r.pagina }}</strong>
            <span>{{ r.trecho }}</span>
          </button>
        </aside>
        <div ref="area" class="area">
          <p v-if="erro" class="aviso erro">{{ erro }}</p>
          <div class="folha">
            <canvas ref="canvas"></canvas>
            <span
              v-for="(r, i) in realces"
              :key="i"
              class="realce"
              :style="{ left: r.x + 'px', top: r.y + 'px', width: r.w + 'px', height: r.h + 'px' }"
            ></span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.fundo {
  position: fixed;
  inset: 0;
  background: rgba(28, 37, 48, 0.55);
  display: grid;
  place-items: center;
  z-index: 10;
}
.janela {
  width: min(1100px, 96vw);
  height: 92vh;
  background: var(--papel);
  border-radius: var(--raio);
  display: grid;
  grid-template-rows: auto 1fr;
  overflow: hidden;
  box-shadow: 0 18px 50px rgba(0, 0, 0, 0.3);
}
.barra {
  display: flex;
  flex-wrap: wrap;
  gap: 0.8rem;
  align-items: center;
  justify-content: space-between;
  padding: 0.7rem 1rem;
  background: var(--superficie);
  border-bottom: 1px solid var(--linha);
}
.nav, .busca { display: flex; gap: 0.5rem; align-items: center; }
.busca input { min-width: 16rem; width: auto; }
.corpo { display: flex; min-height: 0; }
.resultados {
  width: 290px;
  overflow: auto;
  border-right: 1px solid var(--linha);
  padding: 0.6rem;
  display: grid;
  gap: 0.4rem;
  align-content: start;
}
.resultado {
  text-align: left;
  border: 1px solid var(--linha);
  border-radius: var(--raio);
  background: var(--superficie);
  padding: 0.5rem;
  font-size: var(--t-sm);
  display: grid;
  gap: 0.2rem;
}
.area { flex: 1; min-width: 0; overflow: auto; padding: 1rem; }
.folha { position: relative; width: fit-content; margin: 0 auto; box-shadow: 0 1px 6px rgba(0, 0, 0, 0.15); background: #fff; }
canvas { display: block; }
.realce {
  position: absolute;
  background: rgba(255, 214, 0, 0.35);
  outline: 2px solid rgba(200, 150, 0, 0.9);
  border-radius: 2px;
  pointer-events: none;
}
</style>

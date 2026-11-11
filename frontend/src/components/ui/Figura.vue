<script setup lang="ts">
// Figuras animadas da espera (decisão do usuário, 11/11/2026): o robô quando
// a IA trabalha e as engrenagens nas etapas feitas no próprio computador.
// SVG próprio, sem dependência; a cor segue o procedimento (--acento).
// Com "reduzir movimento" ligado no sistema, ficam paradas (estilo.css).

defineProps<{ tipo: "robo" | "engrenagem"; tamanho?: number }>();

// Engrenagem de n dentes, centrada em (0,0): contorno trapezoidal simples.
function engrenagem(n: number, rExt: number, rInt: number): string {
  const pts: string[] = [];
  const passo = (Math.PI * 2) / n;
  for (let i = 0; i < n; i++) {
    const a = i * passo;
    const p = (ang: number, r: number) => `${(Math.cos(ang) * r).toFixed(2)},${(Math.sin(ang) * r).toFixed(2)}`;
    pts.push(p(a - passo * 0.32, rInt), p(a - passo * 0.18, rExt), p(a + passo * 0.18, rExt), p(a + passo * 0.32, rInt));
  }
  return "M" + pts.join("L") + "Z";
}
const grande = engrenagem(10, 30, 23);
const pequena = engrenagem(8, 19.5, 14);
</script>

<template>
  <svg v-if="tipo === 'robo'" class="figura robo" :width="tamanho ?? 96" :height="tamanho ?? 96" viewBox="0 0 120 120"
    aria-hidden="true" focusable="false">
    <ellipse class="sombra" cx="60" cy="112" rx="26" ry="4" />
    <g class="corpo-robo">
      <line x1="60" y1="22" x2="60" y2="11" class="traco" />
      <circle cx="60" cy="9" r="5" class="antena" />
      <rect x="28" y="22" width="64" height="44" rx="14" class="cabeca" />
      <rect x="22" y="38" width="6" height="14" rx="3" class="orelha" />
      <rect x="92" y="38" width="6" height="14" rx="3" class="orelha" />
      <rect x="36" y="30" width="48" height="28" rx="9" class="visor" />
      <g class="olhos">
        <circle cx="50" cy="44" r="4.5" class="olho" />
        <circle cx="70" cy="44" r="4.5" class="olho" />
      </g>
      <rect x="38" y="70" width="44" height="30" rx="9" class="tronco" />
      <rect x="45" y="78" width="7" height="7" rx="2" class="luz l1" />
      <rect x="56.5" y="78" width="7" height="7" rx="2" class="luz l2" />
      <rect x="68" y="78" width="7" height="7" rx="2" class="luz l3" />
      <rect x="45" y="90" width="30" height="3" rx="1.5" class="ranhura" />
      <rect x="27" y="74" width="9" height="20" rx="4.5" class="braco esq" />
      <rect x="84" y="74" width="9" height="20" rx="4.5" class="braco dir" />
    </g>
  </svg>

  <svg v-else class="figura engrenagens" :width="tamanho ?? 96" :height="tamanho ?? 96" viewBox="0 0 120 120"
    aria-hidden="true" focusable="false">
    <g transform="translate(46 54)">
      <g class="gira horario">
        <path :d="grande" class="dente" />
        <circle r="9" class="furo" />
      </g>
    </g>
    <g transform="translate(88.5 84)">
      <g class="gira anti">
        <path :d="pequena" class="dente claro" />
        <circle r="6" class="furo" />
      </g>
    </g>
    <circle cx="92" cy="26" r="3" class="faisca f1" />
    <circle cx="104" cy="40" r="2.2" class="faisca f2" />
  </svg>
</template>

<style scoped>
.figura { flex: none; overflow: visible; }

/* ---- robô ---- */
.sombra { fill: rgba(27, 36, 48, 0.12); animation: sombra 2.8s var(--ease) infinite; transform-origin: 60px 112px; }
.corpo-robo { animation: flutua 2.8s var(--ease) infinite; }
.traco { stroke: var(--tinta); stroke-width: 3; stroke-linecap: round; }
.antena { fill: var(--acento); animation: pulsa-antena 1.4s ease-in-out infinite; transform-origin: 60px 9px; }
.cabeca, .tronco { fill: var(--superficie); stroke: var(--tinta); stroke-width: 3; }
.orelha, .braco { fill: var(--acento); }
.visor { fill: var(--tinta); }
.olho { fill: #fff; }
.olhos { animation: pisca 4.2s infinite; transform-origin: 60px 44px; }
.luz { fill: var(--linha); animation: luz 1.5s infinite; }
.l2 { animation-delay: 0.25s; }
.l3 { animation-delay: 0.5s; }
.ranhura { fill: var(--linha); }
.braco.esq { transform-origin: 31px 76px; animation: aceno 2.8s var(--ease) infinite; }
.braco.dir { transform-origin: 89px 76px; animation: aceno 2.8s var(--ease) infinite reverse; }

@keyframes flutua { 0%, 100% { transform: translateY(0); } 50% { transform: translateY(-4px); } }
@keyframes sombra { 0%, 100% { transform: scaleX(1); opacity: 1; } 50% { transform: scaleX(0.86); opacity: 0.7; } }
@keyframes pulsa-antena { 0%, 100% { transform: scale(1); opacity: 1; } 50% { transform: scale(1.35); opacity: 0.6; } }
@keyframes pisca { 0%, 92%, 100% { transform: scaleY(1); } 95% { transform: scaleY(0.1); } }
@keyframes luz { 0%, 100% { fill: var(--linha); } 40% { fill: var(--acento); } }
@keyframes aceno { 0%, 100% { transform: rotate(0deg); } 50% { transform: rotate(8deg); } }

/* ---- engrenagens ---- */
.dente { fill: var(--tinta); }
.dente.claro { fill: var(--acento); }
.furo { fill: var(--superficie); }
.gira { transform-box: fill-box; transform-origin: center; }
.horario { animation: gira 5s linear infinite; }
.anti { animation: gira 3.85s linear infinite reverse; }
.faisca { fill: var(--acento); opacity: 0; animation: faisca 1.8s ease-out infinite; }
.f2 { animation-delay: 0.9s; }
@keyframes gira { to { transform: rotate(360deg); } }
@keyframes faisca { 0% { opacity: 0; transform: translate(0, 0); } 30% { opacity: 1; } 100% { opacity: 0; transform: translate(6px, -6px); } }
</style>

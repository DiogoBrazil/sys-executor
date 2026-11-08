import { createApp } from "vue";
import "@fontsource/source-serif-4/latin-400.css";
import "@fontsource/source-serif-4/latin-600.css";
import "@fontsource/ibm-plex-sans/latin-400.css";
import "@fontsource/ibm-plex-sans/latin-500.css";
import "@fontsource/ibm-plex-sans/latin-600.css";
import "./estilo.css";
import App from "./App.vue";

// Menu de contexto do navegador desativado (não é um site; evita "Salvar como", "Inspecionar").
document.addEventListener("contextmenu", (e) => e.preventDefault());

createApp(App).mount("#app");

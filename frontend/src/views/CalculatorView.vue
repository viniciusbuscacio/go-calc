<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from "vue";
import { Calculate } from "../../wailsjs/go/main/App";
import { ClipboardGetText, ClipboardSetText } from "../../wailsjs/runtime/runtime";
import { busyInc, busyDec } from "../store";

type Kind = "digit" | "op" | "fn" | "equals";

interface Key {
  label: string;
  kind: Kind;
  token?: string;
  action?: "equals" | "clear" | "backspace";
  wide?: boolean;
}

const keys: Key[] = [
  { label: "%", kind: "fn", token: "%" },
  { label: "C", kind: "fn", action: "clear" },
  { label: "⌫", kind: "fn", action: "backspace" },
  { label: "÷", kind: "op", token: " ÷ " },

  { label: "7", kind: "digit", token: "7" },
  { label: "8", kind: "digit", token: "8" },
  { label: "9", kind: "digit", token: "9" },
  { label: "×", kind: "op", token: " × " },

  { label: "4", kind: "digit", token: "4" },
  { label: "5", kind: "digit", token: "5" },
  { label: "6", kind: "digit", token: "6" },
  { label: "−", kind: "op", token: " − " },

  { label: "1", kind: "digit", token: "1" },
  { label: "2", kind: "digit", token: "2" },
  { label: "3", kind: "digit", token: "3" },
  { label: "+", kind: "op", token: " + " },

  { label: "0", kind: "digit", token: "0", wide: true },
  { label: ".", kind: "digit", token: "." },
  { label: "=", kind: "equals", action: "equals" },
];

const expr = ref("");
const formula = ref("");
const errorMsg = ref("");
const justEvaluated = ref(false);

const display = computed(() => {
  if (errorMsg.value) return errorMsg.value;
  if (expr.value.trim() === "") return "0";
  return expr.value;
});

const displayIsError = computed(() => errorMsg.value !== "");

function reset() {
  expr.value = "";
  formula.value = "";
  errorMsg.value = "";
  justEvaluated.value = false;
}

function feed(token: string, chainFromResult: boolean) {
  if (errorMsg.value) reset();
  if (justEvaluated.value) {
    if (!chainFromResult) expr.value = "";
    justEvaluated.value = false;
    formula.value = "";
  }
  expr.value += token;
}

async function equals() {
  const current = expr.value.trim();
  if (current === "") return;
  busyInc();
  try {
    const result = await Calculate(expr.value);
    formula.value = `${current} =`;
    expr.value = result;
    justEvaluated.value = true;
    errorMsg.value = "";
  } catch (e) {
    errorMsg.value = typeof e === "string" ? e : "error";
    expr.value = "";
    justEvaluated.value = false;
  } finally {
    busyDec();
  }
}

function backspaceChar() {
  if (errorMsg.value) {
    reset();
    return;
  }
  if (justEvaluated.value) return;
  const trimmed = expr.value.replace(/\s+$/, "");
  if (/[+−×÷]$/.test(trimmed)) {
    expr.value = trimmed.slice(0, -1).replace(/\s+$/, "");
  } else {
    expr.value = trimmed.slice(0, -1);
  }
}

function press(key: Key) {
  switch (key.action) {
    case "equals":
      equals();
      break;
    case "clear":
      reset();
      break;
    case "backspace":
      backspaceChar();
      break;
    default:
      feed(key.token ?? "", key.kind === "op" || key.token === "%");
  }
}

// --- right-click Copy / Cut / Paste on the display -----------------------
const menu = reactive({ open: false, x: 0, y: 0 });
const canCopy = computed(() => !errorMsg.value && expr.value.trim() !== "");

function openMenu(e: MouseEvent) {
  menu.x = Math.min(e.clientX, window.innerWidth - 156);
  menu.y = Math.min(e.clientY, window.innerHeight - 132);
  menu.open = true;
}
function closeMenu() {
  menu.open = false;
}

async function copyValue() {
  if (!canCopy.value) return;
  try {
    await ClipboardSetText(display.value);
  } catch {
    /* clipboard unavailable; ignore */
  }
}
async function cutValue() {
  if (!canCopy.value) return;
  await copyValue();
  reset();
}
async function pasteValue() {
  let text = "";
  try {
    text = await ClipboardGetText();
  } catch {
    return;
  }
  // keep only characters the engine understands
  const cleaned = text.replace(/[^0-9.,+\-*/×÷−%()\s]/g, "");
  if (!cleaned) return;
  if (errorMsg.value) reset();
  if (justEvaluated.value) {
    expr.value = "";
    justEvaluated.value = false;
    formula.value = "";
  }
  expr.value += cleaned;
}

function onGlobalPointer(e: MouseEvent) {
  const t = e.target as HTMLElement | null;
  if (menu.open && !t?.closest?.(".ctx-menu")) closeMenu();
}

function onKey(e: KeyboardEvent) {
  const k = e.key;
  if (e.ctrlKey || e.metaKey) {
    const lk = k.toLowerCase();
    if (lk === "c") return void (e.preventDefault(), copyValue());
    if (lk === "x") return void (e.preventDefault(), cutValue());
    if (lk === "v") return void (e.preventDefault(), pasteValue());
    return;
  }
  if (k >= "0" && k <= "9") return feed(k, false);
  if (k === "." || k === ",") return feed(".", false);
  if (k === "+") return feed(" + ", true);
  if (k === "-") return feed(" − ", true);
  if (k === "*") return feed(" × ", true);
  if (k === "%") return feed("%", true);
  if (k === "/") {
    e.preventDefault();
    return feed(" ÷ ", true);
  }
  if (k === "(" || k === ")") return feed(k, false);
  if (k === "Enter" || k === "=") {
    e.preventDefault();
    return equals();
  }
  if (k === "Backspace") return backspaceChar();
  if (k === "Escape") return menu.open ? closeMenu() : reset();
}

onMounted(() => {
  window.addEventListener("keydown", onKey);
  window.addEventListener("mousedown", onGlobalPointer);
  window.addEventListener("scroll", closeMenu, true);
  window.addEventListener("resize", closeMenu);
  window.addEventListener("blur", closeMenu);
});
onUnmounted(() => {
  window.removeEventListener("keydown", onKey);
  window.removeEventListener("mousedown", onGlobalPointer);
  window.removeEventListener("scroll", closeMenu, true);
  window.removeEventListener("resize", closeMenu);
  window.removeEventListener("blur", closeMenu);
});
</script>

<template>
  <div class="view view--calc">
    <section class="screen" @contextmenu.prevent="openMenu">
      <div class="formula" data-testid="formula">{{ formula }}</div>
      <div
        class="value"
        :class="{ error: displayIsError }"
        data-testid="display"
      >
        {{ display }}
      </div>
    </section>

    <Teleport to="body">
      <div
        v-if="menu.open"
        class="ctx-menu"
        :style="{ left: menu.x + 'px', top: menu.y + 'px' }"
        @contextmenu.prevent
      >
        <button class="ctx-item" :disabled="!canCopy" @click="cutValue(); closeMenu()">Cut</button>
        <button class="ctx-item" :disabled="!canCopy" @click="copyValue(); closeMenu()">Copy</button>
        <button class="ctx-item" @click="pasteValue(); closeMenu()">Paste</button>
      </div>
    </Teleport>

    <section class="pad">
      <button
        v-for="(key, i) in keys"
        :key="i"
        class="key"
        :class="[`key--${key.kind}`, { 'key--wide': key.wide }]"
        :data-testid="`key-${key.label}`"
        @click="press(key)"
      >
        <svg
          v-if="key.action === 'backspace'"
          class="key-icon"
          viewBox="0 0 24 24"
          fill="currentColor"
          aria-label="Apagar"
        >
          <path d="M18.75 4C20.483 4 21.8992 5.35645 21.9949 7.06558L22 7.25V16.75C22 18.483 20.6435 19.8992 18.9344 19.9949L18.75 20H10.2488C9.48467 20 8.74732 19.7308 8.16441 19.2436L8.00936 19.1053L3.01367 14.3553C1.71288 13.1185 1.66102 11.0613 2.89784 9.76055L3.01367 9.64472L8.00936 4.89472C8.56313 4.36818 9.28296 4.05515 10.0412 4.00663L10.2488 4H18.75ZM18.75 5.5H10.2488C9.85605 5.5 9.47644 5.63205 9.16975 5.87227L9.04295 5.98177L4.04726 10.7318L3.98489 10.7941C3.35809 11.4534 3.34595 12.4733 3.93064 13.1463L4.04726 13.2682L9.04295 18.0182C9.32758 18.2889 9.69368 18.4547 10.0815 18.492L10.2488 18.5H18.75C19.6682 18.5 20.4212 17.7929 20.4942 16.8935L20.5 16.75V7.25C20.5 6.33183 19.7929 5.57881 18.8935 5.5058L18.75 5.5ZM11.4462 8.39705L11.5303 8.46967L14.0001 10.939L16.4697 8.46967C16.7626 8.17678 17.2374 8.17678 17.5303 8.46967C17.7966 8.73594 17.8208 9.1526 17.6029 9.44621L17.5303 9.53033L15.0611 12L17.5303 14.4697C17.8232 14.7626 17.8232 15.2374 17.5303 15.5303C17.2641 15.7966 16.8474 15.8208 16.5538 15.6029L16.4697 15.5303L14.0001 13.061L11.5303 15.5303C11.2374 15.8232 10.7626 15.8232 10.4697 15.5303C10.2034 15.2641 10.1792 14.8474 10.397 14.5538L10.4697 14.4697L12.9391 12L10.4697 9.53033C10.1768 9.23744 10.1768 8.76256 10.4697 8.46967C10.7359 8.2034 11.1526 8.1792 11.4462 8.39705Z" />
        </svg>
        <template v-else>{{ key.label }}</template>
      </button>
    </section>
  </div>
</template>

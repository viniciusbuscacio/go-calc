import { reactive } from "vue";
import { GetSettings, SetTheme, SetOpacity } from "../wailsjs/go/main/App";

export type View = "calc" | "options" | "api";

// Minimal shared UI state. A single reactive object is enough of a "router"
// for a small single-window app and reuses cleanly across the framework.
export const ui = reactive({
  view: "calc" as View,
  theme: "dark",
  opacity: 100,
  // Number of in-flight async operations triggered from the UI (e.g. an
  // equals → Calculate round-trip). The UI bridge waits until this reaches 0
  // before reading the screen back, so it never samples a half-settled state.
  busy: 0,
});

export function go(view: View) {
  ui.view = view;
}

// busyInc/busyDec bracket any async handler that changes visible state, so the
// UI bridge knows when the screen has finished updating. Call busyInc()
// synchronously before the first await, and busyDec() in a finally block.
export function busyInc() {
  ui.busy++;
}

export function busyDec() {
  ui.busy = Math.max(0, ui.busy - 1);
}

export function applyTheme(theme: string) {
  const t = theme === "light" ? "light" : "dark";
  ui.theme = t;
  document.documentElement.dataset.theme = t;
}

export function applyOpacity(percent: number) {
  const clamped = Math.min(100, Math.max(20, percent));
  ui.opacity = clamped;
  document.documentElement.style.setProperty("--bg-alpha", String(clamped / 100));
}

export async function loadSettings() {
  try {
    const s = await GetSettings();
    applyTheme(s.theme || "dark");
    applyOpacity(s.opacity || 100);
  } catch {
    applyTheme("dark");
    applyOpacity(100);
  }
}

export async function setTheme(theme: string) {
  applyTheme(theme);
  busyInc();
  try {
    await SetTheme(theme);
  } catch {
    /* best effort; UI already reflects the change */
  } finally {
    busyDec();
  }
}

export async function setOpacity(percent: number) {
  applyOpacity(percent);
  busyInc();
  try {
    await SetOpacity(Math.round(percent));
  } catch {
    /* best effort */
  } finally {
    busyDec();
  }
}

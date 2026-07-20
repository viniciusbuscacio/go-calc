import { reactive } from "vue";
import {
  GetSettings,
  GetVersion,
  GetAPIStatus,
  SetTheme,
  SetOpacity,
  GetUpdateInfo,
  CheckForUpdates,
  InstallUpdate,
  SkipUpdateVersion,
  RemindUpdateLater,
  SetUpdateAutoCheck,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";

export type View = "calc" | "options" | "api" | "installer";

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
  // App version, fetched once at startup. Loading it here (not on the About
  // panel's mount) keeps the version element present the moment the panel
  // renders, so a UI-state snapshot never races the fetch.
  version: "",
});

// Live REST server status, mirrored from Go (api:state event). Drives the
// titlebar indicator: a green dot only while a port is actually open.
export const api = reactive({
  running: false,
  port: 0,
  url: "",
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
    update.autoCheck = s.updateAutoCheck === true;
  } catch {
    applyTheme("dark");
    applyOpacity(100);
  }
  try {
    ui.version = await GetVersion();
  } catch {
    /* leave version empty if the backend isn't reachable */
  }
  try {
    // Snapshot from Go (a startup auto-check may already have run) + live
    // updates from here on.
    initUpdateEvents();
    applyUpdateState(await GetUpdateInfo());
  } catch {
    /* updater state stays at its defaults */
  }
  try {
    EventsOn("api:state", (s: object) => Object.assign(api, s));
    Object.assign(api, await GetAPIStatus());
  } catch {
    /* indicator stays hidden if the backend isn't reachable */
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

// ---- in-app updater --------------------------------------------------------
// Mirror of the Go-side UpdateInfo (update.go). Go owns every rule (what is
// newer, when to notify); this store only renders it. Kept in sync two ways:
// explicit awaited calls below, plus the "update:state" event Go broadcasts on
// every change (auto-check results, install progress).

export interface UpdateState {
  checking: boolean;
  installing: boolean;
  progress: string; // downloading | verifying | applying
  available: boolean;
  version: string;
  notes: string;
  current: string;
  checkedAt: string;
  error: string;
  notify: boolean;
}

export const update = reactive<UpdateState & { autoCheck: boolean; seen: boolean }>({
  checking: false,
  installing: false,
  progress: "",
  available: false,
  version: "",
  notes: "",
  current: "",
  checkedAt: "",
  error: "",
  notify: false,
  // Local additions: the auto-check preference, and whether the user has
  // checked manually this session (a manual check shows the result card even
  // for a version that was skipped/snoozed before).
  autoCheck: false,
  seen: false,
});

function applyUpdateState(s: UpdateState) {
  Object.assign(update, s);
}

export function initUpdateEvents() {
  EventsOn("update:state", (s: UpdateState) => applyUpdateState(s));
}

export async function checkForUpdates() {
  update.seen = true;
  busyInc();
  try {
    applyUpdateState(await CheckForUpdates());
  } catch {
    /* the Go side reported what it could via the event */
  } finally {
    busyDec();
  }
}

// installUpdate hands over to Go: on success the app restarts itself and this
// call never resolves.
export async function installUpdate() {
  busyInc();
  try {
    await InstallUpdate();
  } catch {
    /* failure state (update.error) arrives via the update:state event */
  } finally {
    busyDec();
  }
}

export async function skipUpdate() {
  update.seen = false;
  busyInc();
  try {
    await SkipUpdateVersion();
  } catch {
    /* best effort */
  } finally {
    busyDec();
  }
}

export async function remindUpdateLater() {
  update.seen = false;
  busyInc();
  try {
    await RemindUpdateLater();
  } catch {
    /* best effort */
  } finally {
    busyDec();
  }
}

export async function setUpdateAutoCheck(on: boolean) {
  update.autoCheck = on;
  busyInc();
  try {
    await SetUpdateAutoCheck(on);
  } catch {
    /* best effort */
  } finally {
    busyDec();
  }
}

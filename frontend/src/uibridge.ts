import { nextTick } from "vue";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { UIAck } from "../wailsjs/go/main/App";
import { ui } from "./store";

// The Go REST server sends a "ui:command"; we perform it against the REAL DOM
// (click the button, press the key, type), let Vue re-render, then report the
// resulting on-screen state back via UIAck. This is what lets an external agent
// actually operate the UI.

interface UICommand {
  id: string;
  type: "state" | "press" | "key" | "input";
  testid?: string;
  key?: string;
  value?: string;
}

function el(testid: string): HTMLElement | null {
  return document.querySelector(`[data-testid="${testid}"]`);
}

function text(testid: string): string | undefined {
  const node = el(testid);
  return node ? (node.textContent ?? "").trim() : undefined;
}

// A snapshot of what is currently on screen, read from the rendered DOM.
function collectState() {
  const controls = Array.from(document.querySelectorAll("[data-testid]"))
    .map((e) => e.getAttribute("data-testid"))
    .filter((v): v is string => !!v);

  const state: Record<string, unknown> = {
    view: ui.view,
    theme: ui.theme,
    opacity: ui.opacity,
    controls, // every testid currently clickable on screen
  };

  const display = text("display");
  if (display !== undefined) state.display = display;
  const formula = text("formula");
  if (formula !== undefined) state.formula = formula;
  const status = text("status");
  if (status !== undefined) state.serverStatus = status;

  const allowlist = el("allowlist");
  if (allowlist) {
    state.allowlist = Array.from(
      allowlist.querySelectorAll("td.mono"),
    ).map((td) => (td.textContent ?? "").trim());
  }

  const inputs: Record<string, string> = {};
  document.querySelectorAll("input[data-testid]").forEach((n) => {
    const t = n.getAttribute("data-testid");
    if (t) inputs[t] = (n as HTMLInputElement).value;
  });
  if (Object.keys(inputs).length) state.inputs = inputs;

  return state;
}

// A structured, application-level failure the Go side maps to an HTTP status.
interface UIError {
  code: "unknown_testid" | "disabled_control";
  message: string;
}

function isDisabled(node: HTMLElement): boolean {
  return (
    node.hasAttribute("disabled") ||
    (node as HTMLButtonElement).disabled === true ||
    node.getAttribute("aria-disabled") === "true"
  );
}

// settle waits for the DOM to finish updating: one microtask flush (nextTick)
// for synchronous changes, then — if a handler started async work — until the
// shared busy counter drains, bounded so a stuck promise can't hang the bridge.
async function settle() {
  await nextTick();
  const deadline = performance.now() + 2500;
  while (ui.busy > 0 && performance.now() < deadline) {
    await new Promise((r) => window.setTimeout(r, 10));
  }
  await nextTick();
}

// perform runs the command and returns a structured error when the target
// control is missing or disabled, so the caller learns exactly what went wrong.
async function perform(cmd: UICommand): Promise<UIError | undefined> {
  let error: UIError | undefined;

  if (cmd.type === "press") {
    const node = cmd.testid ? el(cmd.testid) : null;
    if (!node) {
      error = { code: "unknown_testid", message: `unknown testid: ${cmd.testid}` };
    } else if (isDisabled(node)) {
      error = { code: "disabled_control", message: `control is disabled: ${cmd.testid}` };
    } else {
      node.click();
    }
  } else if (cmd.type === "key" && cmd.key) {
    window.dispatchEvent(
      new KeyboardEvent("keydown", { key: cmd.key, bubbles: true }),
    );
  } else if (cmd.type === "input") {
    const node = cmd.testid
      ? (el(cmd.testid) as HTMLInputElement | null)
      : null;
    if (!node) {
      error = { code: "unknown_testid", message: `unknown testid: ${cmd.testid}` };
    } else if (isDisabled(node)) {
      error = { code: "disabled_control", message: `control is disabled: ${cmd.testid}` };
    } else {
      // Use the native value setter so the framework's reactive tracking sees
      // the change, then fire the events v-model listens to.
      const setter = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value",
      )?.set;
      setter ? setter.call(node, cmd.value ?? "") : (node.value = cmd.value ?? "");
      node.dispatchEvent(new Event("input", { bubbles: true }));
      node.dispatchEvent(new Event("change", { bubbles: true }));
    }
  }

  await settle();
  return error;
}

// Commands are serialized: a busy bridge queues the next one so two DOM
// mutations never overlap and corrupt each other's read-back.
let queue: Promise<void> = Promise.resolve();

async function run(cmd: UICommand) {
  let error: UIError | undefined;
  try {
    error = await perform(cmd);
  } catch {
    /* still report whatever is on screen */
  }
  const state = collectState();
  if (error) state.error = error;
  try {
    await UIAck(cmd.id, JSON.stringify(state));
  } catch {
    /* nothing we can do; the Go side will time out */
  }
}

function handle(cmd: UICommand) {
  queue = queue.then(() => run(cmd));
}

export function initUIBridge() {
  EventsOn("ui:command", (cmd: UICommand) => {
    handle(cmd);
  });
}

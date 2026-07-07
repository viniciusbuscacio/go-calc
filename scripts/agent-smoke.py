#!/usr/bin/env python3
"""End-to-end smoke test for the go-Calc agent control plane.

It discovers the API key and port from the app's settings file (or takes them
via --base-url / --api-key), then exercises every layer an agent depends on:

  * /v1/health           reachability + auth
  * /v1/ax               the app descriptor / accessibility tree contract
  * /v1/calc             direct computation (incl. big ints and decimals)
  * /v1/ui/*             driving the REAL UI (press, key, state)
  * structured errors    invalid_json, missing_field, unknown_testid,
                         disabled_control, calculation_error
  * AX <-> DOM coverage   every testid advertised in /v1/ax is reachable on screen

Run it while the app is open with the REST server started:

    python scripts/agent-smoke.py
    python scripts/agent-smoke.py --base-url http://127.0.0.1:8737 --api-key <key>

Exit code is 0 when every check passes, 1 otherwise.
"""

from __future__ import annotations

import argparse
import json
import os
import platform
import sys
import urllib.error
import urllib.request

APP_DIR = "go-calc"
SETTINGS_FILE = "settings.json"


def settings_path() -> str:
    home = os.path.expanduser("~")
    system = platform.system()
    if system == "Windows":
        base = os.environ.get("APPDATA") or os.path.join(home, "AppData", "Roaming")
    elif system == "Darwin":
        base = os.path.join(home, "Library", "Application Support")
    else:
        base = os.environ.get("XDG_CONFIG_HOME") or os.path.join(home, ".config")
    return os.path.join(base, APP_DIR, SETTINGS_FILE)


def load_settings() -> dict:
    try:
        with open(settings_path(), "r", encoding="utf-8") as fh:
            return json.load(fh)
    except (OSError, ValueError):
        return {}


class Client:
    def __init__(self, base: str, key: str) -> None:
        self.base = base.rstrip("/")
        self.key = key

    def call(self, method: str, path: str, body=None, key=None):
        """Return (status, parsed_json). HTTP errors are returned, not raised."""
        headers = {"X-API-Key": self.key if key is None else key}
        data = None
        if body is not None:
            data = json.dumps(body).encode("utf-8")
            headers["Content-Type"] = "application/json"
        req = urllib.request.Request(
            self.base + path, data=data, headers=headers, method=method
        )
        try:
            with urllib.request.urlopen(req, timeout=15) as resp:
                raw = resp.read().decode("utf-8")
                return resp.status, (json.loads(raw) if raw else None)
        except urllib.error.HTTPError as err:
            raw = err.read().decode("utf-8")
            try:
                return err.code, json.loads(raw)
            except ValueError:
                return err.code, raw
        except urllib.error.URLError as err:
            print(f"  ! could not reach {self.base}{path}: {err}")
            return None, None


class Checks:
    def __init__(self) -> None:
        self.passed = 0
        self.failed = 0

    def ok(self, name: str, cond: bool, detail: str = "") -> bool:
        mark = "PASS" if cond else "FAIL"
        line = f"[{mark}] {name}"
        if detail and not cond:
            line += f"  -> {detail}"
        print(line)
        if cond:
            self.passed += 1
        else:
            self.failed += 1
        return cond


def err_code(payload) -> str:
    if isinstance(payload, dict) and isinstance(payload.get("error"), dict):
        return payload["error"].get("code", "")
    return ""


def collect_ax_testids(node: dict, out: set) -> None:
    tid = node.get("testid")
    if tid:
        out.add(tid)
    for child in node.get("children", []) or []:
        collect_ax_testids(child, out)


def press(c: Client, testid: str):
    return c.call("POST", "/v1/ui/press", {"testid": testid})


def key(c: Client, k: str):
    return c.call("POST", "/v1/ui/key", {"key": k})


def display(state) -> str:
    return state.get("display", "") if isinstance(state, dict) else ""


def controls(state) -> set:
    if isinstance(state, dict) and isinstance(state.get("controls"), list):
        return set(state["controls"])
    return set()


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--base-url", help="e.g. http://127.0.0.1:8737")
    ap.add_argument("--api-key", help="X-API-Key value")
    args = ap.parse_args()

    s = load_settings()
    port = s.get("apiPort", 8737)
    base = args.base_url or f"http://127.0.0.1:{port}"
    api_key = args.api_key or s.get("apiKey", "")
    if not api_key:
        print("No API key found. Pass --api-key or open the app once so it writes settings.json.")
        return 2

    print(f"Target: {base}")
    c = Client(base, api_key)
    chk = Checks()

    # --- health + auth -------------------------------------------------------
    st, body = c.call("GET", "/v1/health")
    if not chk.ok("health reachable", st == 200, f"status={st}"):
        print("Server not reachable — is the app open with the REST server started?")
        return 1
    chk.ok("health body ok", isinstance(body, dict) and body.get("status") == "ok", str(body))

    st, body = c.call("GET", "/v1/health", key="wrong-key")
    chk.ok("auth rejects bad key (401 unauthorized)",
           st == 401 and err_code(body) == "unauthorized", f"status={st} body={body}")

    # --- ax contract ---------------------------------------------------------
    st, ax = c.call("GET", "/v1/ax")
    chk.ok("ax reachable", st == 200, f"status={st}")
    chk.ok("ax has schemaVersion", isinstance(ax, dict) and "schemaVersion" in ax)
    chk.ok("ax has version", isinstance(ax, dict) and bool(ax.get("version")))
    chk.ok("ax advertises capabilities",
           isinstance(ax, dict) and "calc" in (ax.get("capabilities") or []))
    chk.ok("ax documents error codes",
           isinstance(ax, dict) and any(e.get("code") == "unknown_testid"
                                        for e in (ax.get("errors") or [])))

    ax_testids: set = set()
    if isinstance(ax, dict) and isinstance(ax.get("axTree"), dict):
        collect_ax_testids(ax["axTree"], ax_testids)
    chk.ok("ax tree exposes testids", len(ax_testids) > 10, f"count={len(ax_testids)}")

    # --- direct compute ------------------------------------------------------
    st, body = c.call("POST", "/v1/calc", {"expression": "2 + 3 * 4"})
    chk.ok("calc respects precedence (2 + 3 * 4 = 14)",
           st == 200 and body.get("result") == "14", str(body))

    st, body = c.call("POST", "/v1/calc", {"expression": "9007199254740992 + 1"})
    chk.ok("calc exact beyond 2^53",
           st == 200 and body.get("result") == "9007199254740993", str(body))

    st, body = c.call("POST", "/v1/calc", {"expression": "0.1 + 0.2"})
    chk.ok("calc exact decimals (0.1 + 0.2 = 0.3)",
           st == 200 and body.get("result") == "0.3", str(body))

    st, body = c.call("POST", "/v1/calc", {})
    chk.ok("calc empty body -> 400 missing_field",
           st == 400 and err_code(body) == "missing_field", f"status={st} body={body}")

    st, body = c.call("POST", "/v1/calc", {"expression": "1 / 0"})
    chk.ok("calc div-by-zero -> 422 calculation_error",
           st == 422 and err_code(body) == "calculation_error", f"status={st} body={body}")

    # --- drive the real UI ---------------------------------------------------
    st, state = c.call("GET", "/v1/ui/state")
    chk.ok("ui state reachable on calc view",
           st == 200 and isinstance(state, dict) and state.get("view") == "calc",
           f"status={st}")

    press(c, "key-C")
    press(c, "key-7")
    _, state = press(c, "key-9")
    chk.ok("pressing real buttons appends (7,9 -> '79')", display(state) == "79",
           f"display={display(state)!r}")

    press(c, "key-C")
    for ch in ["7", "*", "6"]:
        key(c, ch)
    _, state = key(c, "Enter")
    chk.ok("keyboard drive + evaluate (7 * 6 = 42)", display(state) == "42",
           f"display={display(state)!r}")

    press(c, "key-C")
    for ch in ["(", "2", "+", "3", ")", "*", "4"]:
        key(c, ch)
    _, state = key(c, "Enter")
    chk.ok("parentheses via keys ((2 + 3) * 4 = 20)", display(state) == "20",
           f"display={display(state)!r}")
    press(c, "key-C")

    st, body = press(c, "key-does-not-exist")
    chk.ok("unknown testid -> 404 unknown_testid",
           st == 404 and err_code(body) == "unknown_testid", f"status={st} body={body}")

    st, body = key(c, "")
    chk.ok("empty key -> 400 missing_field",
           st == 400 and err_code(body) == "missing_field", f"status={st} body={body}")

    # --- disabled control ----------------------------------------------------
    press(c, "open-settings")
    press(c, "nav-api")
    st, body = press(c, "add-ip")  # 'New IP' is empty, so Add is disabled
    chk.ok("disabled control -> 409 disabled_control",
           st == 409 and err_code(body) == "disabled_control", f"status={st} body={body}")

    # --- AX <-> DOM coverage -------------------------------------------------
    seen: set = set()
    _, state = c.call("GET", "/v1/ui/state")
    seen |= controls(state)          # api view
    _, state = press(c, "back")
    seen |= controls(state)          # options view
    _, state = press(c, "back")
    seen |= controls(state)          # calc view
    missing = sorted(t for t in ax_testids if t not in seen)
    chk.ok("every ax testid is reachable on screen", not missing,
           f"missing={missing}")

    print(f"\n{chk.passed} passed, {chk.failed} failed")
    return 0 if chk.failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())

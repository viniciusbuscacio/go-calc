# Agent API — the control plane

go-Calc exposes a small HTTP API so an AI agent (or any script) can discover the
app and operate it end-to-end. It is **off by default**; start it from
**Settings → REST API Server**.

Every request needs the header `X-API-Key: <key>` (shown, copyable, in that same
panel) and must come from an IP in the allowlist (default `127.0.0.1/32`). See
[security.md](security.md).

Base URL is shown in the panel; by default `http://127.0.0.1:8737`.

## Start here: `GET /v1/ax`

One request tells the agent everything. The response is a JSON document:

| Field | Meaning |
| --- | --- |
| `schemaVersion` | integer contract version of this document — bumped on any breaking change |
| `version` | app/framework version |
| `app`, `description` | what this is |
| `howToUse` | prose instructions (precedence, exact arithmetic, how errors and risk work) |
| `capabilities` | e.g. `["calc","ui.state","ui.press","ui.key","ui.input"]` |
| `api` | the endpoint list with example bodies |
| `errors` | every error `code`, its HTTP `status`, and its meaning |
| `axTree` | the accessibility tree: every view and control |

### The accessibility tree

Each node describes one view or control:

```json
{
  "role": "button",
  "name": "7",
  "testid": "key-7",
  "action": "append digit 7",
  "keyboard": "7",
  "risk": "safe"
}
```

- **`testid`** is how you address the control in `/v1/ui/*`.
- **`keyboard`** is the physical key that does the same thing (useful for
  `/v1/ui/key`, and the only way to enter parentheses — they have no button).
- **`risk`** tells you how careful to be before pressing (see below).
- Views carry `openedBy` / `id` so you know how to reach controls that are not on
  the current screen.

### Risk levels

| Risk | Meaning | Examples |
| --- | --- | --- |
| `safe` | no lasting effect | digits, operators, theme, copy public text |
| `navigation` | only moves between views | `open-settings`, `nav-api`, `back` |
| `external` | reaches outside the app | `open-github` (opens a browser) |
| `sensitive` | changes exposure or reveals a secret | `toggle-server`, `add-ip`, `rotate-key`, `copy-key` |
| `destructive` | irreversible / closes the app | `window-close` |

An agent should treat anything above `navigation` as needing intent or
confirmation.

## Compute directly: `POST /v1/calc`

```
POST /v1/calc   {"expression":"2 + 3 * 4"}   ->   200 {"result":"14"}
```

Accepts the UI glyphs too (`×`, `÷`, `−`) and `,` as a decimal separator.
Arithmetic is exact. An unparseable or undefined expression (e.g. `1 / 0`)
returns `422 calculation_error`.

## Drive the real UI: `/v1/ui/*`

These operate the actual frontend and return the resulting on-screen **state**
(the same shape `GET /v1/ui/state` returns): current `view`, `theme`, `opacity`,
the list of `controls` currently on screen, and view-specific fields like
`display`, `formula`, `serverStatus`, `allowlist`, `inputs`.

| Method | Path | Body | Does |
| --- | --- | --- | --- |
| `GET` | `/v1/ui/state` | — | read what is on screen |
| `POST` | `/v1/ui/press` | `{"testid":"key-7"}` | click a control |
| `POST` | `/v1/ui/key` | `{"key":"Enter"}` | send a keystroke to the app |
| `POST` | `/v1/ui/input` | `{"testid":"new-ip","value":"10.0.0.0/24"}` | type into a field (empty `value` clears it) |

The bridge waits for the UI to finish updating — including any Go round-trip
triggered by the action — before it reads the state back, so the state you
receive reflects the completed action, not a half-rendered frame. Commands are
serialized, so it is safe to fire them back-to-back.

### Example: press `7 × 6 =` on the real keypad

```bash
curl -X POST $BASE/v1/ui/press -H "X-API-Key: $KEY" -d '{"testid":"key-7"}'
curl -X POST $BASE/v1/ui/key   -H "X-API-Key: $KEY" -d '{"key":"*"}'
curl -X POST $BASE/v1/ui/press -H "X-API-Key: $KEY" -d '{"testid":"key-6"}'
curl -X POST $BASE/v1/ui/key   -H "X-API-Key: $KEY" -d '{"key":"Enter"}'
# {"view":"calc","display":"42","formula":"7 × 6 =", ...}
```

## Errors

Every error has the same shape; branch on `code`, not on the message text:

```json
{ "error": { "code": "unknown_testid", "message": "unknown testid: key-x", "status": 404 } }
```

| `code` | HTTP | When |
| --- | --- | --- |
| `invalid_json` | 400 | body is not valid JSON |
| `missing_field` | 400 | a required field (`expression` / `testid` / `key`) was empty or absent |
| `unauthorized` | 401 | invalid or missing `X-API-Key` |
| `forbidden` | 403 | client IP not in the allowlist |
| `unknown_testid` | 404 | no control on screen has that `testid` |
| `method_not_allowed` | 405 | wrong HTTP method (these endpoints are POST) |
| `disabled_control` | 409 | the control exists but is currently disabled |
| `calculation_error` | 422 | the expression could not be evaluated |
| `ui_timeout` | 503 | the UI did not respond in time |

The authoritative list is always the `errors` array in `GET /v1/ax` for the
running version.

## Verifying the contract

`scripts/agent-smoke.py` exercises all of the above — health, auth, the `/v1/ax`
fields, direct compute (including big integers and exact decimals), driving the
UI, every structured error, and a check that **every `testid` advertised in
`/v1/ax` is reachable on screen**. Run it with the app open and the server
started:

```bash
python scripts/agent-smoke.py
```

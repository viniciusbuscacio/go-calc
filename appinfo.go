package main

// appInfo builds the app descriptor / accessibility tree served at GET /v1/ax.
// It tells an automated client what the app is, how to use it, and — via the
// axTree — every view and control with its role, testid, action, keyboard
// shortcut and risk level, so an agent knows exactly where to "click" and which
// actions to treat with care.

// axSchemaVersion is the contract version of the /v1/ax document. Bump it on any
// breaking change to the shape below so clients can detect drift.
const axSchemaVersion = 1

// appVersion is the app/framework version reported in /v1/ax. It is a var, not
// a const, so release builds can inject the tag version at link time via
// `-ldflags "-X main.appVersion=..."` (see .github/workflows/release.yml).
var appVersion = "0.1.0"

// Risk levels classify how careful a client should be before invoking a control:
//
//	safe        no lasting effect (digits, operators, theme, copy public text)
//	navigation  only moves between views
//	external    reaches outside the app (opens a browser)
//	sensitive   changes security/exposure or reveals a secret (server, allowlist, key)
//	destructive irreversible or closes the app
const (
	riskSafe        = "safe"
	riskNavigation  = "navigation"
	riskExternal    = "external"
	riskSensitive   = "sensitive"
	riskDestructive = "destructive"
)

type axNode struct {
	Role        string   `json:"role"`
	Name        string   `json:"name"`
	ID          string   `json:"id,omitempty"`
	Testid      string   `json:"testid,omitempty"`
	Description string   `json:"description,omitempty"`
	Action      string   `json:"action,omitempty"`
	Keyboard    string   `json:"keyboard,omitempty"`
	Risk        string   `json:"risk,omitempty"`
	OpenedBy    string   `json:"openedBy,omitempty"`
	Children    []axNode `json:"children,omitempty"`
}

type apiEndpoint struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Body    any    `json:"body,omitempty"`
	Returns any    `json:"returns,omitempty"`
	Auth    string `json:"auth"`
}

// errorInfo documents one stable error code an automated client may receive.
type errorInfo struct {
	Code    string `json:"code"`
	Status  int    `json:"status"`
	Meaning string `json:"meaning"`
}

type appInfoDTO struct {
	SchemaVersion int           `json:"schemaVersion"`
	Version       string        `json:"version"`
	App           string        `json:"app"`
	Description   string        `json:"description"`
	HowToUse      string        `json:"howToUse"`
	Capabilities  []string      `json:"capabilities"`
	API           []apiEndpoint `json:"api"`
	Errors        []errorInfo   `json:"errors"`
	AXTree        axNode        `json:"axTree"`
}

func calcKeyNodes() []axNode {
	keys := []struct{ label, action, keyboard string }{
		{"%", "append percent (postfix, divides the number by 100)", "%"},
		{"C", "clear the whole expression", "Escape"},
		{"⌫", "delete the last token", "Backspace"},
		{"÷", "append division", "/"},
		{"7", "append digit 7", "7"},
		{"8", "append digit 8", "8"},
		{"9", "append digit 9", "9"},
		{"×", "append multiplication", "*"},
		{"4", "append digit 4", "4"},
		{"5", "append digit 5", "5"},
		{"6", "append digit 6", "6"},
		{"−", "append subtraction", "-"},
		{"1", "append digit 1", "1"},
		{"2", "append digit 2", "2"},
		{"3", "append digit 3", "3"},
		{"+", "append addition", "+"},
		{"0", "append digit 0", "0"},
		{".", "append the decimal separator", "."},
		{"=", "evaluate the expression", "Enter"},
	}
	nodes := make([]axNode, 0, len(keys))
	for _, k := range keys {
		nodes = append(nodes, axNode{
			Role:     "button",
			Name:     k.label,
			Testid:   "key-" + k.label,
			Action:   k.action,
			Keyboard: k.keyboard,
			Risk:     riskSafe,
		})
	}
	return nodes
}

func (a *App) appInfo() any {
	calcChildren := append([]axNode{
		{Role: "textbox", Name: "Display", Testid: "display", Description: "Shows the current expression or the result", Action: "read-only"},
		{Role: "text", Name: "Formula", Testid: "formula", Description: "Shows the evaluated expression after '='"},
	}, calcKeyNodes()...)

	return appInfoDTO{
		SchemaVersion: axSchemaVersion,
		Version:       appVersion,
		App:           "go-Calc",
		Description:   "A Windows 11-style calculator built with Go + Wails and TypeScript. Also a template for cross-platform desktop apps.",
		HowToUse: "Build a full arithmetic expression and press '=' or Enter to evaluate. Operator precedence is respected, so 2 + 3 × 4 = 14. " +
			"Parentheses have NO on-screen button — send them with /v1/ui/key ('(' and ')') or a physical keyboard. " +
			"Arithmetic is exact (math/big rationals): integers beyond 2^53 and sums like 0.1 + 0.2 are precise. " +
			"To compute directly, POST the expression to /v1/calc. To operate the actual UI (press real buttons, read the screen), use the " +
			"/v1/ui/* endpoints: press a control by its testid, send a key, or type into an input — each returns the resulting on-screen state. " +
			"Every control carries a 'risk' level (safe, navigation, external, sensitive, destructive) — check it before pressing; e.g. window-close is destructive. " +
			"Errors are structured as {\"error\":{\"code\",\"message\",\"status\"}}; branch on 'code' (see the errors list). " +
			"Pressing an unknown testid returns code unknown_testid (404); a disabled control returns disabled_control (409). The testids are listed in this axTree. " +
			"The title-bar gear (open-settings) opens Settings; each panel has a back button. " +
			"Settings also hosts the in-app updater: press 'update-check' to query GitHub Releases (risk external) and read the outcome at GET /v1/update; " +
			"'update-install' (risk destructive) downloads, verifies and applies the new version, then RESTARTS the app — the API goes away mid-call.",
		Capabilities: []string{"calc", "ui.state", "ui.press", "ui.key", "ui.input", "updates"},
		Errors: []errorInfo{
			{Code: "invalid_json", Status: 400, Meaning: "request body is not valid JSON"},
			{Code: "missing_field", Status: 400, Meaning: "a required field (expression / testid / key) was empty or absent"},
			{Code: "unauthorized", Status: 401, Meaning: "invalid or missing X-API-Key header"},
			{Code: "forbidden", Status: 403, Meaning: "the client IP is not in the allowlist"},
			{Code: "unknown_testid", Status: 404, Meaning: "no control on screen has that testid"},
			{Code: "method_not_allowed", Status: 405, Meaning: "wrong HTTP method for this endpoint (see the api list: /v1/ui/state, /v1/health and /v1/ax are GET; the rest are POST)"},
			{Code: "disabled_control", Status: 409, Meaning: "the control exists but is currently disabled"},
			{Code: "calculation_error", Status: 422, Meaning: "the expression could not be evaluated"},
			{Code: "ui_timeout", Status: 503, Meaning: "the UI did not respond in time"},
		},
		API: []apiEndpoint{
			{Method: "POST", Path: "/v1/calc", Body: map[string]string{"expression": "2 + 3 * 4"}, Returns: map[string]string{"result": "14"}, Auth: "X-API-Key header"},
			{Method: "GET", Path: "/v1/health", Returns: map[string]string{"status": "ok"}, Auth: "X-API-Key header"},
			{Method: "GET", Path: "/v1/ax", Returns: "this document (app info + accessibility tree)", Auth: "X-API-Key header"},
			{Method: "GET", Path: "/v1/update", Returns: "last update-check snapshot: {checking, installing, available, version, notes, current, checkedAt, error, notify}", Auth: "X-API-Key header"},
			{Method: "GET", Path: "/v1/ui/state", Returns: "current on-screen state", Auth: "X-API-Key header"},
			{Method: "POST", Path: "/v1/ui/press", Body: map[string]string{"testid": "key-7"}, Returns: "resulting on-screen state", Auth: "X-API-Key header"},
			{Method: "POST", Path: "/v1/ui/key", Body: map[string]string{"key": "Enter"}, Returns: "resulting on-screen state", Auth: "X-API-Key header"},
			{Method: "POST", Path: "/v1/ui/input", Body: map[string]string{"testid": "new-ip", "value": "10.0.0.0/24"}, Returns: "resulting on-screen state", Auth: "X-API-Key header"},
		},
		AXTree: axNode{
			Role: "application",
			Name: "go-Calc",
			Children: []axNode{
				{
					Role:        "toolbar",
					Name:        "Title bar",
					Description: "Always visible, on every view.",
					Children: []axNode{
						{Role: "button", Name: "Settings", Testid: "open-settings", Action: "open the Settings view", Risk: riskNavigation},
						{Role: "button", Name: "Minimize", Testid: "window-minimize", Action: "minimize the window", Risk: riskSafe},
						{Role: "button", Name: "Maximize", Testid: "window-maximize", Action: "maximize/restore the window", Risk: riskSafe},
						{Role: "button", Name: "Close", Testid: "window-close", Action: "close the app", Risk: riskDestructive},
					},
				},
				{
					Role:        "view",
					Name:        "Calculator",
					ID:          "calc",
					Description: "Main screen. Buttons append to the expression; '=' evaluates it. Parentheses have no button — use /v1/ui/key.",
					Children:    calcChildren,
				},
				{
					Role:        "view",
					Name:        "Settings",
					ID:          "options",
					OpenedBy:    "open-settings",
					Description: "Opened by the gear button in the title bar.",
					Children: []axNode{
						{Role: "button", Name: "Back", Testid: "back", Action: "return to the calculator", Risk: riskNavigation},
						{Role: "switch", Name: "Dark mode", Testid: "theme-switch", Action: "toggle between dark and light theme", Risk: riskSafe},
						{Role: "slider", Name: "Transparency", Testid: "opacity-slider", Action: "set window opacity from 20% to 100%", Risk: riskSafe},
						{Role: "button", Name: "REST API Server", Testid: "nav-api", Action: "open the REST API server settings", Risk: riskNavigation},
						{Role: "switch", Name: "Automatic update checks", Testid: "update-autocheck", Action: "toggle checking GitHub for a newer release once a day on launch (off by default; checking calls the network)", Risk: riskSafe},
						{Role: "button", Name: "Check for updates", Testid: "update-check", Action: "ask GitHub Releases for a newer version right now; the result (including 'notify') is also served at GET /v1/update", Risk: riskExternal},
						{Role: "status", Name: "Update status", Testid: "update-status", Description: "outcome of the last update check: up to date, update available, or the error"},
						{Role: "text", Name: "Release notes", Testid: "update-notes", Description: "the newer version's release notes; present only while an update is available"},
						{Role: "button", Name: "Install and restart", Testid: "update-install", Action: "download the new version, verify its checksum, replace the app and restart it; present only while an update is available", Risk: riskDestructive},
						{Role: "button", Name: "Skip this version", Testid: "update-skip", Action: "silence this particular version (a newer one will notify again); present only while an update is available", Risk: riskSafe},
						{Role: "button", Name: "Remind me later", Testid: "update-later", Action: "snooze the update notice for 7 days; present only while an update is available", Risk: riskSafe},
						{Role: "link", Name: "GitHub", Testid: "open-github", Action: "open the project on GitHub in the default browser", Risk: riskExternal},
						{Role: "text", Name: "Version", Testid: "app-version", Description: "the app version (About section)"},
					},
				},
				{
					Role:        "view",
					Name:        "REST API Server",
					ID:          "api",
					Description: "Opened from Settings → REST API Server.",
					Children: []axNode{
						{Role: "button", Name: "Back", Testid: "back", Action: "return to Settings", Risk: riskNavigation},
						{Role: "button", Name: "Start/Stop", Testid: "toggle-server", Action: "start or stop the REST server", Risk: riskSensitive},
						{Role: "status", Name: "Server status", Testid: "status", Description: "shows Running or Stopped"},
						{Role: "text", Name: "Server error", Testid: "server-error", Description: "why the last server operation failed; rendered only after a failure"},
						{Role: "button", Name: "Shuffle port", Testid: "shuffle-port", Action: "pick a random free port (8700-8799) and restart the server if running", Risk: riskSensitive},
						{Role: "switch", Name: "Start automatically", Testid: "autostart", Action: "toggle starting the server on app launch", Risk: riskSensitive},
						{Role: "switch", Name: "Use HTTPS", Testid: "use-https", Action: "toggle HTTPS (self-signed) vs plain HTTP; restarts the server if running", Risk: riskSensitive},
						{Role: "table", Name: "Allowed IPs", Testid: "allowlist", Description: "CIDR allowlist controlling who may call the API"},
						{Role: "button", Name: "Remove IP", Testid: "remove-<cidr>", Description: "one per allowlist row; the testid embeds the CIDR, e.g. remove-127.0.0.1/32", Action: "remove that CIDR from the allowlist", Risk: riskSensitive},
						{Role: "textbox", Name: "New IP", Testid: "new-ip", Action: "type a CIDR (e.g. 192.168.0.0/24) to allow", Risk: riskSafe},
						{Role: "button", Name: "Add IP", Testid: "add-ip", Action: "add the typed CIDR to the allowlist", Risk: riskSensitive},
						{Role: "text", Name: "IP error", Testid: "ip-error", Description: "why the typed CIDR was rejected; rendered only after a failure"},
						{Role: "text", Name: "Agent instructions", Testid: "agent-instructions", Description: "copy-paste snippet: base URL, key, and starting endpoints"},
						{Role: "button", Name: "Copy instructions", Testid: "copy-instructions", Action: "copy the agent instructions", Risk: riskSafe},
						{Role: "text", Name: "Access key", Testid: "api-key", Description: "the API key (masked)"},
						{Role: "button", Name: "Copy key", Testid: "copy-key", Action: "copy the API key", Risk: riskSensitive},
						{Role: "button", Name: "Rotate key", Testid: "rotate-key", Action: "generate a new API key", Risk: riskSensitive},
						{Role: "text", Name: "Certificate pin", Testid: "fingerprint", Description: "the TLS public-key pin (shortened); visible only while serving HTTPS"},
						{Role: "button", Name: "Copy pin", Testid: "copy-fingerprint", Action: "copy the full TLS public-key pin (sha256//...)", Risk: riskSafe},
						{Role: "text", Name: "Test command", Testid: "curl-example", Description: "a ready-to-run curl with the pin baked in; visible only while serving HTTPS"},
						{Role: "button", Name: "Copy test command", Testid: "copy-curl", Action: "copy the ready-to-run curl example", Risk: riskSafe},
					},
				},
			},
		},
	}
}

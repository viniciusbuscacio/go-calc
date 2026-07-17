package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"

	"github.com/viniciusbuscacio/go-calc/internal/apiserver"
	"github.com/viniciusbuscacio/go-calc/internal/calc"
	"github.com/viniciusbuscacio/go-calc/internal/settings"
	updater "github.com/viniciusbuscacio/go-updates"
)

// API port range the shuffle button picks from.
const (
	portRangeBase = 8700
	portRangeSpan = 100 // 8700..8799
)

// App is the thin Wails adapter. Business logic lives in internal/*; App just
// wires it to the frontend and owns process-level state (settings + server).
type App struct {
	ctx context.Context
	// mu guards cfg: Wails-bound methods and the REST server's UI handlers run
	// on different goroutines. The rule to avoid deadlocks: lock → copy/mutate
	// cfg → unlock, and only then call anything slow (settings.Save, server
	// start/stop). APIAllowlist is copy-on-write (never mutated in place), so a
	// shallow copy of cfg is safe to read without the lock.
	mu     sync.Mutex
	cfg    settings.Settings
	server *apiserver.Server
	ui     *uiBridge
	// In-app updater state (see update.go): the last check's snapshot and the
	// release it found, kept so Install doesn't need to re-check.
	updState   UpdateInfo
	updRelease *updater.Release
}

func NewApp() *App {
	a := &App{}
	a.ui = newUIBridge(a)
	a.server = apiserver.New(calc.Evaluate, a.appInfo, a.ui)
	a.server.HandleExtra("/v1/update", a.handleUpdate)
	return a
}

// UIAck is called by the frontend to report the resulting screen state after
// executing a ui:command. It is bound to JS by Wails.
func (a *App) UIAck(id string, state string) {
	a.ui.ack(id, json.RawMessage(state))
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cfg := settings.Load()
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	go fixTaskbarIcon(appTitle)
	// Sweep the ".old" binary a previous self-update parked, then — if the
	// user opted in — check for a newer release in the background.
	a.updateClient().CleanupOld()
	go a.maybeAutoCheck()
	if cfg.APIAutoStart {
		_ = a.startServer()
	}
}

// Calculate evaluates a full arithmetic expression.
func (a *App) Calculate(expression string) (string, error) {
	return calc.Evaluate(expression)
}

// ---- Settings ----

// snapshot returns a copy of the current settings, safe to use without the
// lock (see the copy-on-write note on App.mu).
func (a *App) snapshot() settings.Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

func (a *App) GetSettings() settings.Settings {
	return a.snapshot()
}

func (a *App) SetTheme(theme string) error {
	if theme != "light" {
		theme = "dark"
	}
	a.mu.Lock()
	a.cfg.Theme = theme
	cfg := a.cfg
	a.mu.Unlock()
	return settings.Save(cfg)
}

func (a *App) SetOpacity(percent int) error {
	if percent < 20 {
		percent = 20
	}
	if percent > 100 {
		percent = 100
	}
	a.mu.Lock()
	a.cfg.Opacity = percent
	cfg := a.cfg
	a.mu.Unlock()
	return settings.Save(cfg)
}

// ---- REST API server ----

// APIStatus is the snapshot the frontend renders.
type APIStatus struct {
	Running     bool   `json:"running"`
	Port        int    `json:"port"`
	URL         string `json:"url"`
	TLS         bool   `json:"tls"`
	Fingerprint string `json:"fingerprint"` // public-key pin, set while TLS is running
}

// apiURL builds the URL clients should call for the given settings. The HTTPS
// scheme is a direct user choice (the "Use HTTPS" toggle), independent of the
// bind address.
func apiURL(cfg settings.Settings) string {
	host := "127.0.0.1"
	if apiserver.BindHost(cfg.APIAllowlist) == "0.0.0.0" {
		host = apiserver.OutboundIP()
	}
	scheme := "http"
	if cfg.APIHTTPS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, host, cfg.APIPort)
}

func (a *App) status() APIStatus {
	cfg := a.snapshot()
	return APIStatus{
		Running:     a.server.Running(),
		Port:        cfg.APIPort,
		URL:         apiURL(cfg),
		TLS:         cfg.APIHTTPS,
		Fingerprint: a.server.Fingerprint(),
	}
}

func (a *App) startServer() error {
	dir, err := settings.ConfigDir()
	if err != nil {
		return err
	}
	cfg := a.snapshot()
	return a.server.Start(apiserver.Config{
		Port:      cfg.APIPort,
		Key:       cfg.APIKey,
		Allowlist: cfg.APIAllowlist,
		TLS:       cfg.APIHTTPS,
		CertDir:   dir,
	})
}

// applyIfRunning restarts the server so config changes (key, allowlist) take
// effect immediately while it is running. The error is returned so callers can
// surface a server that failed to come back up instead of silently showing it
// as running.
func (a *App) applyIfRunning() error {
	if !a.server.Running() {
		return nil
	}
	if err := a.server.Stop(); err != nil {
		return err
	}
	return a.startServer()
}

func (a *App) StartAPIServer() (APIStatus, error) {
	if err := a.startServer(); err != nil {
		return a.status(), err
	}
	return a.status(), nil
}

func (a *App) StopAPIServer() (APIStatus, error) {
	if err := a.server.Stop(); err != nil {
		return a.status(), err
	}
	return a.status(), nil
}

func (a *App) GetAPIStatus() APIStatus {
	return a.status()
}

// ShuffleAPIPort picks a random FREE port in 8700–8799 (different from the
// current one), persists it, and restarts the server if running. It probes for
// a free port so pressing the button actually escapes an occupied port.
func (a *App) ShuffleAPIPort() (APIStatus, error) {
	// Probe for the port outside the lock (it binds sockets), then commit it.
	cur := a.snapshot()
	port := pickFreePort(cur.APIPort, apiserver.BindHost(cur.APIAllowlist))
	a.mu.Lock()
	a.cfg.APIPort = port
	cfg := a.cfg
	a.mu.Unlock()
	if err := settings.Save(cfg); err != nil {
		return a.status(), err
	}
	if err := a.applyIfRunning(); err != nil {
		return a.status(), err
	}
	return a.status(), nil
}

// pickFreePort returns a random bindable port in the range, avoiding exclude.
// It falls back to any random port ≠ exclude if none probe as free.
func pickFreePort(exclude int, host string) int {
	for i := 0; i < 40; i++ {
		p := portRangeBase + rand.IntN(portRangeSpan)
		if p == exclude {
			continue
		}
		if ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, p)); err == nil {
			_ = ln.Close()
			return p
		}
	}
	next := exclude
	for next == exclude {
		next = portRangeBase + rand.IntN(portRangeSpan)
	}
	return next
}

func (a *App) SetAPIAutoStart(v bool) error {
	a.mu.Lock()
	a.cfg.APIAutoStart = v
	cfg := a.cfg
	a.mu.Unlock()
	return settings.Save(cfg)
}

// SetHTTPS chooses the transport (HTTPS when true, plain HTTP when false), then
// restarts the server if running so the change (scheme + fingerprint) applies
// immediately.
func (a *App) SetHTTPS(v bool) (APIStatus, error) {
	a.mu.Lock()
	a.cfg.APIHTTPS = v
	cfg := a.cfg
	a.mu.Unlock()
	if err := settings.Save(cfg); err != nil {
		return a.status(), err
	}
	if err := a.applyIfRunning(); err != nil {
		return a.status(), err
	}
	return a.status(), nil
}

// GetAPIFingerprint returns the public-key pin while the TLS server is running
// (empty otherwise). A client pins it: curl --pinnedpubkey sha256//<fingerprint>.
func (a *App) GetAPIFingerprint() string {
	return a.server.Fingerprint()
}

func (a *App) GetAllowlist() []string {
	return a.snapshot().APIAllowlist
}

func (a *App) AddAllowlistEntry(entry string) ([]string, error) {
	normalized, err := apiserver.NormalizeCIDR(entry)
	if err != nil {
		return a.snapshot().APIAllowlist, err
	}
	a.mu.Lock()
	for _, e := range a.cfg.APIAllowlist {
		if e == normalized {
			list := a.cfg.APIAllowlist
			a.mu.Unlock()
			return list, nil
		}
	}
	// Copy-on-write: build a fresh slice instead of appending in place, so
	// snapshots handed out earlier stay valid without the lock.
	next := append(append([]string(nil), a.cfg.APIAllowlist...), normalized)
	a.cfg.APIAllowlist = next
	cfg := a.cfg
	a.mu.Unlock()
	if err := settings.Save(cfg); err != nil {
		return next, err
	}
	if err := a.applyIfRunning(); err != nil {
		return next, err
	}
	return next, nil
}

func (a *App) RemoveAllowlistEntry(entry string) ([]string, error) {
	a.mu.Lock()
	next := make([]string, 0, len(a.cfg.APIAllowlist))
	for _, e := range a.cfg.APIAllowlist {
		if e != entry {
			next = append(next, e)
		}
	}
	a.cfg.APIAllowlist = next
	cfg := a.cfg
	a.mu.Unlock()
	if err := settings.Save(cfg); err != nil {
		return next, err
	}
	if err := a.applyIfRunning(); err != nil {
		return next, err
	}
	return next, nil
}

func (a *App) GetAPIKey() string {
	return a.snapshot().APIKey
}

func (a *App) RotateAPIKey() (string, error) {
	a.mu.Lock()
	a.cfg.APIKey = settings.GenerateKey()
	cfg := a.cfg
	a.mu.Unlock()
	if err := settings.Save(cfg); err != nil {
		return cfg.APIKey, err
	}
	if err := a.applyIfRunning(); err != nil {
		return cfg.APIKey, err
	}
	return cfg.APIKey, nil
}

func (a *App) GetAPIURL() string {
	return apiURL(a.snapshot())
}

// GetVersion returns the app version so the frontend can show it (Settings →
// About). Single source of truth: the same appVersion reported in /v1/ax.
func (a *App) GetVersion() string {
	return appVersion
}

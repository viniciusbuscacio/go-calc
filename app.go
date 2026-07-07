package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net"

	"github.com/viniciusbuscacio/go-calc/internal/apiserver"
	"github.com/viniciusbuscacio/go-calc/internal/calc"
	"github.com/viniciusbuscacio/go-calc/internal/settings"
)

// API port range the shuffle button picks from.
const (
	portRangeBase = 8700
	portRangeSpan = 100 // 8700..8799
)

// App is the thin Wails adapter. Business logic lives in internal/*; App just
// wires it to the frontend and owns process-level state (settings + server).
type App struct {
	ctx    context.Context
	cfg    settings.Settings
	server *apiserver.Server
	ui     *uiBridge
}

func NewApp() *App {
	a := &App{}
	a.ui = newUIBridge(a)
	a.server = apiserver.New(calc.Evaluate, a.appInfo, a.ui)
	return a
}

// UIAck is called by the frontend to report the resulting screen state after
// executing a ui:command. It is bound to JS by Wails.
func (a *App) UIAck(id string, state string) {
	a.ui.ack(id, json.RawMessage(state))
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cfg = settings.Load()
	go fixTaskbarIcon(appTitle)
	if a.cfg.APIAutoStart {
		_ = a.startServer()
	}
}

// Calculate evaluates a full arithmetic expression.
func (a *App) Calculate(expression string) (string, error) {
	return calc.Evaluate(expression)
}

// ---- Settings ----

func (a *App) GetSettings() settings.Settings {
	return a.cfg
}

func (a *App) SetTheme(theme string) error {
	if theme != "light" {
		theme = "dark"
	}
	a.cfg.Theme = theme
	return settings.Save(a.cfg)
}

func (a *App) SetOpacity(percent int) error {
	if percent < 20 {
		percent = 20
	}
	if percent > 100 {
		percent = 100
	}
	a.cfg.Opacity = percent
	return settings.Save(a.cfg)
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

// useTLS reports whether the server should serve HTTPS. It is a direct user
// choice (the "Use HTTPS" toggle), independent of the bind address.
func (a *App) useTLS() bool {
	return a.cfg.APIHTTPS
}

func (a *App) apiURL() string {
	host := "127.0.0.1"
	if apiserver.BindHost(a.cfg.APIAllowlist) == "0.0.0.0" {
		host = apiserver.OutboundIP()
	}
	scheme := "http"
	if a.useTLS() {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, host, a.cfg.APIPort)
}

func (a *App) status() APIStatus {
	return APIStatus{
		Running:     a.server.Running(),
		Port:        a.cfg.APIPort,
		URL:         a.apiURL(),
		TLS:         a.useTLS(),
		Fingerprint: a.server.Fingerprint(),
	}
}

func (a *App) startServer() error {
	dir, err := settings.ConfigDir()
	if err != nil {
		return err
	}
	return a.server.Start(apiserver.Config{
		Port:      a.cfg.APIPort,
		Key:       a.cfg.APIKey,
		Allowlist: a.cfg.APIAllowlist,
		TLS:       a.useTLS(),
		CertDir:   dir,
	})
}

// applyIfRunning restarts the server so config changes (key, allowlist) take
// effect immediately while it is running.
func (a *App) applyIfRunning() {
	if a.server.Running() {
		_ = a.server.Stop()
		_ = a.startServer()
	}
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
	host := apiserver.BindHost(a.cfg.APIAllowlist)
	a.cfg.APIPort = pickFreePort(a.cfg.APIPort, host)
	if err := settings.Save(a.cfg); err != nil {
		return a.status(), err
	}
	a.applyIfRunning()
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
	a.cfg.APIAutoStart = v
	return settings.Save(a.cfg)
}

// SetHTTPS chooses the transport (HTTPS when true, plain HTTP when false), then
// restarts the server if running so the change (scheme + fingerprint) applies
// immediately.
func (a *App) SetHTTPS(v bool) (APIStatus, error) {
	a.cfg.APIHTTPS = v
	if err := settings.Save(a.cfg); err != nil {
		return a.status(), err
	}
	a.applyIfRunning()
	return a.status(), nil
}

// GetAPIFingerprint returns the public-key pin while the TLS server is running
// (empty otherwise). A client pins it: curl --pinnedpubkey sha256//<fingerprint>.
func (a *App) GetAPIFingerprint() string {
	return a.server.Fingerprint()
}

func (a *App) GetAllowlist() []string {
	return a.cfg.APIAllowlist
}

func (a *App) AddAllowlistEntry(entry string) ([]string, error) {
	normalized, err := apiserver.NormalizeCIDR(entry)
	if err != nil {
		return a.cfg.APIAllowlist, err
	}
	for _, e := range a.cfg.APIAllowlist {
		if e == normalized {
			return a.cfg.APIAllowlist, nil
		}
	}
	a.cfg.APIAllowlist = append(a.cfg.APIAllowlist, normalized)
	if err := settings.Save(a.cfg); err != nil {
		return a.cfg.APIAllowlist, err
	}
	a.applyIfRunning()
	return a.cfg.APIAllowlist, nil
}

func (a *App) RemoveAllowlistEntry(entry string) ([]string, error) {
	next := make([]string, 0, len(a.cfg.APIAllowlist))
	for _, e := range a.cfg.APIAllowlist {
		if e != entry {
			next = append(next, e)
		}
	}
	a.cfg.APIAllowlist = next
	if err := settings.Save(a.cfg); err != nil {
		return next, err
	}
	a.applyIfRunning()
	return next, nil
}

func (a *App) GetAPIKey() string {
	return a.cfg.APIKey
}

func (a *App) RotateAPIKey() (string, error) {
	a.cfg.APIKey = settings.GenerateKey()
	if err := settings.Save(a.cfg); err != nil {
		return a.cfg.APIKey, err
	}
	a.applyIfRunning()
	return a.cfg.APIKey, nil
}

func (a *App) GetAPIURL() string {
	return a.apiURL()
}

// GetVersion returns the app version so the frontend can show it (Settings →
// About). Single source of truth: the same appVersion reported in /v1/ax.
func (a *App) GetVersion() string {
	return appVersion
}

// Package settings persists user preferences to a JSON file in the OS config
// dir. It is framework-agnostic (no Wails) so it can be reused across apps.
package settings

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Settings struct {
	Theme        string   `json:"theme"`        // "dark" | "light"
	Opacity      int      `json:"opacity"`      // window opacity, 20..100
	APIAutoStart bool     `json:"apiAutoStart"` // start REST server on app launch
	APIPort      int      `json:"apiPort"`
	APIKey       string   `json:"apiKey"`
	APIAllowlist []string `json:"apiAllowlist"` // CIDRs, e.g. "127.0.0.1/32"
	APIHTTPS     bool     `json:"apiHttps"`     // serve HTTPS instead of HTTP

	// In-app updater. AutoCheck is opt-in: the app makes no network call the
	// user didn't ask for. Timestamps are RFC3339; empty means "never"/"unset".
	UpdateAutoCheck  bool   `json:"updateAutoCheck"`      // check GitHub once a day on launch
	UpdateSkipped    string `json:"updateSkippedVersion"` // tag the user chose to skip (e.g. "v0.2.0")
	UpdateLaterUntil string `json:"updateLaterUntil"`     // "Later" cooldown: no notify before this instant
	UpdateLastCheck  string `json:"updateLastAutoCheck"`  // last automatic check, for the once-a-day throttle
}

const (
	appDir      = "go-calc"
	fileName    = "settings.json"
	defaultPort = 8737
)

var mu sync.Mutex

func Default() Settings {
	return Settings{
		Theme:        "dark",
		Opacity:      100,
		APIAutoStart: false,
		APIPort:      defaultPort,
		APIKey:       GenerateKey(),
		APIAllowlist: []string{"127.0.0.1/32"},
		APIHTTPS:     false,
	}
}

// ConfigDir returns the app's per-user config directory (e.g. the folder that
// holds settings.json and the TLS key). It does not create the directory.
func ConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appDir), nil
}

func filePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

// Load reads settings, returning sensible defaults (and persisting them) on the
// first run or if the file is missing/corrupt.
func Load() Settings {
	mu.Lock()
	defer mu.Unlock()

	p, err := filePath()
	if err != nil {
		return Default()
	}
	data, err := os.ReadFile(p)
	if err != nil {
		s := Default()
		_ = save(s)
		return s
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		// Corrupt JSON: keep the bad file for inspection and persist fresh
		// defaults, mirroring the missing-file path — otherwise every launch
		// would generate (and hand out) a brand-new API key.
		_ = os.Rename(p, p+".corrupt")
		s = Default()
		_ = save(s)
		return s
	}

	d := Default()
	if s.Theme == "" {
		s.Theme = d.Theme
	}
	if s.Opacity < 20 || s.Opacity > 100 {
		s.Opacity = d.Opacity
	}
	if s.APIPort < 1 || s.APIPort > 65535 {
		s.APIPort = d.APIPort
	}
	if s.APIKey == "" {
		s.APIKey = GenerateKey()
	}
	if s.APIAllowlist == nil {
		s.APIAllowlist = d.APIAllowlist
	}
	return s
}

func Save(s Settings) error {
	mu.Lock()
	defer mu.Unlock()
	return save(s)
}

func save(s Settings) error {
	p, err := filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// GenerateKey returns a random 48-char hex API key.
func GenerateKey() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

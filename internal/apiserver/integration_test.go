package apiserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestServerEndToEnd(t *testing.T) {
	s := New(func(expr string) (string, error) {
		if expr == "2+2" {
			return "4", nil
		}
		return "", fmt.Errorf("expressao invalida")
	}, nil, nil)
	cfg := Config{Port: 18737, Key: "secret", Allowlist: []string{"127.0.0.1/32"}}
	if err := s.Start(cfg); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	base := "http://127.0.0.1:18737"

	// Authenticated calc.
	req, _ := http.NewRequest(http.MethodPost, base+"/v1/calc", strings.NewReader(`{"expression":"2+2"}`))
	req.Header.Set("X-API-Key", "secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("calc request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("calc status = %d, quer 200", resp.StatusCode)
	}
	var out calcResponse
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	if out.Result != "4" {
		t.Errorf("result = %q, quer \"4\"", out.Result)
	}

	// Missing key is rejected.
	noKey, _ := http.NewRequest(http.MethodGet, base+"/v1/health", nil)
	r2, err := http.DefaultClient.Do(noKey)
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	r2.Body.Close()
	if r2.StatusCode != http.StatusUnauthorized {
		t.Errorf("no-key status = %d, quer 401", r2.StatusCode)
	}

	// Health with key.
	ok, _ := http.NewRequest(http.MethodGet, base+"/v1/health", nil)
	ok.Header.Set("X-API-Key", "secret")
	r3, err := http.DefaultClient.Do(ok)
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	r3.Body.Close()
	if r3.StatusCode != http.StatusOK {
		t.Errorf("health status = %d, quer 200", r3.StatusCode)
	}
}

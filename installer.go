package main

import (
	_ "embed"
)

// The embedded install wizard ("the downloaded exe IS the installer").
// Mechanics live in the shared github.com/viniciusbuscacio/go-installer
// library; this app draws the wizard (InstallerView.vue) and decides the
// launch mode. Real implementation in install_windows.go — the wizard is
// Windows-only; macOS ships a DMG and Linux has its CLI install.

// projectURL is the app's public home: the license screen links it and the
// Apps & Features entry lists it.
const projectURL = "https://github.com/viniciusbuscacio/go-calc"

// licenseText is the MIT license shown on the wizard's license screen.
//
//go:embed LICENSE
var licenseText string

// InstallerState is everything the wizard needs to draw itself. Mode ""
// means a normal app run (installed, portable, or a non-Windows OS) and
// keeps the wizard hidden.
type InstallerState struct {
	Mode    string `json:"mode"` // "" | "wizard" | "uninstall"
	Dir     string `json:"dir"`  // destination folder (default or user-chosen)
	Version string `json:"version"`
	URL     string `json:"url"`
	License string `json:"license"`
}

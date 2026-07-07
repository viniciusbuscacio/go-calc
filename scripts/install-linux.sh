#!/usr/bin/env bash
# install-linux.sh — install go-Calc into the current user's desktop.
#
# Copies the built binary, the app icon, and a .desktop launcher into the
# per-user XDG locations so the icon shows up in the dock/taskbar and app grid
# (needed on Linux, especially GNOME/Wayland, where the window's app_id is
# matched to a .desktop file rather than a runtime window icon).
#
#   ./scripts/install-linux.sh            # install (build first with: go run ./tools/build)
#   ./scripts/install-linux.sh --uninstall
#
# No root required; installs under ~/.local.
set -euo pipefail

APP_ID="go-calc"                       # matches linux.Options.ProgramName / StartupWMClass
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_SRC="$REPO_ROOT/build/bin/go-calc"
ICON_SRC="$REPO_ROOT/build/appicon.png"
DESKTOP_SRC="$REPO_ROOT/build/linux/go-calc.desktop"

BIN_DST="$HOME/.local/bin/$APP_ID"
ICON_DST="$HOME/.local/share/icons/hicolor/512x512/apps/$APP_ID.png"
DESKTOP_DST="$HOME/.local/share/applications/$APP_ID.desktop"

uninstall() {
  rm -f "$BIN_DST" "$ICON_DST" "$DESKTOP_DST"
  update-desktop-database "$HOME/.local/share/applications" 2>/dev/null || true
  gtk-update-icon-cache -f -t "$HOME/.local/share/icons/hicolor" 2>/dev/null || true
  echo "go-Calc uninstalled from ~/.local"
}

if [[ "${1:-}" == "--uninstall" ]]; then
  uninstall
  exit 0
fi

if [[ ! -x "$BIN_SRC" ]]; then
  echo "error: $BIN_SRC not found. Build it first:" >&2
  echo "  go run ./tools/build" >&2
  exit 1
fi

install -Dm755 "$BIN_SRC"  "$BIN_DST"
install -Dm644 "$ICON_SRC" "$ICON_DST"

install -d "$(dirname "$DESKTOP_DST")"
sed -e "s|__EXEC__|$BIN_DST|" -e "s|__ICON__|$APP_ID|" "$DESKTOP_SRC" > "$DESKTOP_DST"
chmod 644 "$DESKTOP_DST"

update-desktop-database "$HOME/.local/share/applications" 2>/dev/null || true
gtk-update-icon-cache -f -t "$HOME/.local/share/icons/hicolor" 2>/dev/null || true

echo "Installed go-Calc:"
echo "  binary : $BIN_DST"
echo "  icon   : $ICON_DST"
echo "  launcher: $DESKTOP_DST"
echo
echo "Launch it from your app grid (search 'go-Calc') so the dock icon binds"
echo "to the .desktop file. If ~/.local/bin isn't on PATH, add it to run 'go-calc'."

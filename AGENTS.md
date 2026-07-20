# go-calc — agent notes

Calculator of the [go-apps](https://github.com/viniciusbuscacio/go-apps)
family; it was the model project of the mini-framework.

**Before changing anything, read the family spec:**
[`go-apps.spec`](https://github.com/viniciusbuscacio/go-apps/blob/main/go-apps.spec)
(local sibling checkout: `../go-apps/go-apps.spec`). UI patterns and assets:
[go-design](https://github.com/viniciusbuscacio/go-design)
(`../go-design/README.md`). go-notepad is the family's visual reference.

App specifics:

- Engine in `internal/calc` (pure Go); `app.go` is the Wails adapter.
- REST API port: family-shared range **8000–8999**, random default per
  install; domain endpoint `POST /v1/calc`.
- Smoke suite: `go run ./tools/smoke` with the app open and the server on.
- Gate before commit: `go vet ./...`, `go test ./...`, `wails build`, smoke.

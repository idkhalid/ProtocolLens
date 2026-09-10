# Release Checklist

For v0.1.0 preparation only. Do not tag or publish until explicitly approved.

- [ ] Clean git status reviewed
- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `go build ./...`
- [ ] `go test -race ./...` where supported
- [ ] `cd web && npm ci && npm run lint && npm run typecheck && npm run build`
- [ ] `cd browser && npm ci && npm run typecheck && npm run build`
- [ ] Windows and Linux Go builds/tests pass in CI
- [ ] Replay default disabled verified
- [ ] Local capture default disabled verified
- [ ] Public bind with local capture rejected
- [ ] HAR fixtures smoke tested
- [ ] Safe replay smoke tested only with explicit local enablement
- [ ] Generators verified network-inert
- [ ] Optional local capture smoke tested on loopback
- [ ] Benchmark smoke tested with safe GET only
- [ ] README and changelog reviewed
- [ ] Release build uses `-ldflags "-X protocollens/internal/version.Version=v0.1.0"`
- [ ] Tag `v0.1.0`
- [ ] GitHub release notes copied from `CHANGELOG.md`

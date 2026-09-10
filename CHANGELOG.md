# Changelog

## v0.1.0 - 2026-09-10

### Added

- HAR 1.2 import with endpoint aggregation.
- Session artifact detection for Cookie, Set-Cookie, Bearer, CSRF, and API key signals.
- Deterministic dependency inference from response JSON into later request paths, query values, JSON bodies, and form bodies.
- Workflow graph construction from captured request instances and dependency evidence.
- Developer Workbench UI for overview, endpoints, sessions, dependencies, workflow, replay, capture, and benchmark views.
- Safe HTTP Replay, disabled by default.
- Client generation for cURL, Python httpx, and Go net/http.
- Optional local Playwright capture adapter and local Capture Workbench.
- Browser-to-HTTP benchmark using bounded sequential safe replay runs.

### Security / Safety

- Imported sensitive header values are redacted during normalization.
- Replay is disabled by default and uses destination, DNS/IP, redirect, timeout, request-size, response-size, and concurrency guards when enabled.
- Local capture is disabled by default, requires loopback server bind and local Host/Origin/RemoteAddr checks, and does not trust forwarded headers.
- Benchmarks reuse Safe Replay, cap runs at 5, run sequentially, and require acknowledgement for repeated non-idempotent methods.
- Replay, capture, benchmark, and generated client outputs are not persisted as histories.

### Known Limitations

- HAR analysis is observational evidence, not a complete model of application semantics.
- Browser-to-HTTP benchmark results are directional and reflect current replay transport/network conditions, not lab-grade performance measurements.
- Browser capture is local-only and does not automate login flows, CAPTCHA solving, scripting, or stealth behavior.
- No capture, replay, or benchmark history is stored.
- The Go server is API-only; the React workbench is built and served separately for v0.1.0.

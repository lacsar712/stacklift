# StackLift

Tower crane **moment-percent limiting** service with slew/boom interlock, wind anemometer gust hold windows, and an embedded operator web console (Go 1.22).

Coordinates rig pose snapshots, duty-mode FSM side effects, load-cell moment validation, and wind confirmation windows driven by a process clock so operator pauses do not alone close interlock timers.

## Packages

| Package | Role |
|---------|------|
| `internal/crane` | Per-rig orchestration and fleet grouping |
| `internal/boom` | Luff/jib angle driver and pose projection |
| `internal/slew` | Slew coordinator, plant, emitter |
| `internal/load` | Load cell checks and moment-percent engine |
| `internal/wind` | Anemometer classification and gust sentinels |
| `internal/interlock` | Axis limits, wind hold window, slew/boom matrix |
| `internal/fsm` | Duty mode machine with motion side effects |
| `internal/store` | Rig pose store with deep-copy snapshots |
| `internal/counter` | Moment integrators and cycle tallies |
| `internal/hook` | Hook block height sensor |
| `internal/alarm` | Operator alarm bus |
| `internal/telemetry` | Event ring |
| `internal/journal` | Audit journal writer |
| `internal/api` | HTTP control API |
| `internal/web` | Embedded operator UI |
| `internal/app` | Process wiring |
| `internal/config` | Site and limit configuration |
| `internal/clock` | Process clock for confirm windows |
| `internal/model` | Domain types |

## Run

```bash
go test ./...
go run ./cmd/stacklift
go run ./cmd/stacklift -demo
```

Default listen address: `:8092`.

## API

- `GET /api/health` — liveness
- `GET /api/status` — rig status bundle
- `GET /api/poses` — pose snapshot
- `POST /api/slew` — `{"rig":"TC-01","delta_deg":5}`
- `POST /api/boom` — `{"rig":"TC-01","delta_deg":2}`
- `POST /api/emergency` — `{"rig":"TC-01"}`

Static operator console is served from embedded `internal/web/static/`.

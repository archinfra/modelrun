# ModelRun Platform Architecture Redesign

## 1. Core judgement

The current platform mixes four very different workloads into one storage and execution path:

1. control-plane configuration
2. runtime command execution
3. live logs / step output
4. monitoring and metrics collection

That is the root reason the product feels slow.

The most important architecture rule for the redesign is:

- SQLite stores control data and run summaries
- runtime logs are append-only files plus in-memory tail buffers
- metrics are current-state cache plus time-series backend
- command execution is stream-first, not database-first

## 2. What is wrong with the current design

### 2.1 SQLite is carrying hot-path runtime traffic

Today `Store.Update()` clones the entire `domain.Data`, then `save()` deletes the whole `documents` table and writes every collection back again.

That makes SQLite a bottleneck for:

- every deployment status update
- every server test result
- every config change
- every log append summary

This pattern is acceptable for small config CRUD, but not for runtime state.

### 2.2 One document store is being used for everything

Current persistent state includes:

- servers
- deployments
- tasks
- action templates
- bootstrap configs
- pipeline step templates
- deployment logs

The problem is not SQLite itself. The problem is that the current access pattern is "load all / clone all / rewrite all".

### 2.3 Monitoring path is mixed with control path

Exporter data, server test data, and runtime resource data are being treated like configuration snapshots. That means a frequent update to observability can disturb the same store used by deployment CRUD.

### 2.4 Logs are not modeled as logs

Runtime logs want:

- ordered append
- tail reads
- streaming to UI
- later replay

SQLite can store summaries, but it should not be the primary sink for line-by-line runtime logs.

### 2.5 UI fetch shape is too coarse

Large pages currently fetch multiple heavy datasets together. Even after some frontend optimization, the deeper problem remains:

- the backend returns too much coupled state
- some pages poll data that is not actively being viewed
- runtime and configuration state are not isolated

## 3. Target architecture

The platform should be split into four subsystems.

## 3.1 Control plane

Purpose:

- projects
- servers
- jump hosts
- models
- deployments
- action templates
- bootstrap templates
- pipeline templates
- run summaries

Storage:

- SQLite is acceptable in the short term
- PostgreSQL is the right long-term target if the platform becomes multi-user or high-concurrency

Important rule:

- control-plane writes must be row-level upserts
- never rewrite the whole database for one update

Recommended tables:

- `projects`
- `servers`
- `jump_hosts`
- `models`
- `deployments`
- `deployment_runs`
- `deployment_run_steps`
- `action_templates`
- `bootstrap_templates`
- `pipeline_step_templates`
- `artifacts`

What should NOT be here:

- line-by-line runtime stdout/stderr
- every exporter sample
- websocket session state

## 3.2 Runtime execution plane

Purpose:

- SSH or agent-based command execution
- deployment pipeline orchestration
- step-by-step event emission
- cancellation
- replay of what happened

Runtime model:

- one `deployment_run` creates one execution session
- each server has a `run_worker`
- each pipeline step emits events

Event types:

- `run_started`
- `step_started`
- `step_stdout`
- `step_stderr`
- `step_progress`
- `step_failed`
- `step_completed`
- `run_completed`
- `run_cancelled`

Storage for runtime execution:

- in-memory tail buffer for websocket streaming
- append-only log files on disk for replay
- summary rows in SQLite on step end / run end

Recommended local file layout:

- `/var/lib/modelrun/runs/<run_id>/manifest.json`
- `/var/lib/modelrun/runs/<run_id>/<server_id>/<step_id>.stdout.log`
- `/var/lib/modelrun/runs/<run_id>/<server_id>/<step_id>.stderr.log`
- `/var/lib/modelrun/runs/<run_id>/<server_id>/<step_id>.meta.json`

Why this matters:

- stream writes become cheap
- replay is exact
- "还原现场" becomes natural
- database pressure stays low

## 3.3 Observability plane

Purpose:

- node metrics
- GPU / NPU telemetry
- exporter health
- model process state

This must be separated from SSH execution.

Recommended data sources:

- `node_exporter` for host metrics
- `npu_exporter` for Ascend NPU metrics
- optional container metrics endpoint
- optional runtime health endpoints

Recommended design:

- Prometheus scrapes exporters
- backend queries Prometheus for dashboard and detail pages
- backend maintains an in-memory "last known status cache" for fast UI summaries

If Prometheus is too heavy for the first phase:

- keep only a current-state cache in memory
- use exporter HTTP scraping directly from backend
- do not write every sample to SQLite
- persist only periodic snapshots or alert-worthy changes

What should go into SQLite:

- exporter installation status
- last successful scrape time
- last error summary
- last known health state

What should NOT go into SQLite:

- every CPU / memory / NPU sample
- every second-by-second time series point

## 3.4 Realtime delivery plane

Purpose:

- websocket updates to the UI
- step logs
- progress
- server status changes

Recommended design:

- backend event bus in memory
- per-run subscriptions
- ring buffer per stream for reconnect replay
- append-only file persistence for durable replay

The websocket server should not read from SQLite for hot updates.

Instead:

1. command runner emits event
2. event bus publishes to websocket subscribers immediately
3. tail buffer stores the latest N lines
4. file sink appends durable logs
5. SQLite receives only summary state updates

## 4. SSH versus agent

## 4.1 What should still use SSH

SSH is fine for:

- pipeline command execution
- bootstrap installs
- ad hoc diagnostics
- low-frequency file inspection

SSH is NOT ideal for:

- continuous metrics collection
- high-frequency status polling
- large-fanout command orchestration at scale

## 4.2 When to introduce an agent

A lightweight agent becomes worth it when you need:

- persistent session management
- command queues on remote nodes
- resume after backend restart
- faster fanout to many servers
- local scraping / local file tailing / local process inspection
- better jump-host abstraction

Recommended judgement for ModelRun:

- keep SSH for deployment execution in phase 1
- do not use SSH for monitoring
- introduce an optional edge agent in phase 2

That agent should expose:

- command execution
- file tail
- process state
- exporter discovery
- local runtime inventory

## 5. Exact storage boundaries

This is the most important decision set.

## 5.1 SQLite should store

- users / projects / servers / jump hosts
- deployment definitions
- templates and bootstrap definitions
- deployment run summaries
- step summaries
- final error summary
- artifact paths
- last known exporter health

## 5.2 Filesystem should store

- stdout / stderr logs
- raw command transcript
- rendered pipeline scripts
- runtime generated config files
- exporter installation output
- run manifests

## 5.3 In-memory cache should store

- current websocket subscribers
- latest server health cache
- latest metrics snapshot cache
- per-step tail logs
- active run state machine

## 5.4 Time-series backend should store

- host CPU / memory / disk / network
- NPU memory / utilization / temperature / HBM / process metrics
- optional inference throughput metrics

Preferred:

- Prometheus

Optional long-term:

- VictoriaMetrics / Thanos if scale increases

## 6. API redesign

Current APIs should be split by workload type.

## 6.1 Control APIs

- `GET /api/servers`
- `POST /api/deployments`
- `PATCH /api/pipeline-step-templates/:id`

These read and write SQLite-backed control data.

## 6.2 Run APIs

- `POST /api/deployment-runs`
- `POST /api/deployment-runs/:id/cancel`
- `GET /api/deployment-runs/:id`
- `GET /api/deployment-runs/:id/steps`
- `GET /api/deployment-runs/:id/artifacts`

These operate on run summaries and runtime artifacts.

## 6.3 Log APIs

- `GET /api/deployment-runs/:id/logs?serverId=&stepId=&cursor=`
- `GET /api/deployment-runs/:id/logs/tail?...`

These read append-only log files, not SQLite rows.

## 6.4 Metrics APIs

- `GET /api/servers/:id/health`
- `GET /api/servers/:id/metrics/current`
- `GET /api/servers/:id/metrics/range?...`
- `GET /api/deployments/:id/metrics/current`

These read from cache or Prometheus, not from deployment config documents.

## 6.5 Realtime APIs

- `WS /ws/runs/:id`
- `WS /ws/servers/:id`

Separate channels keep large pages from subscribing to irrelevant traffic.

## 7. UI redesign rules

The frontend should follow these rules.

## 7.1 Never open a page by loading everything

Each screen should request only what it needs.

Examples:

- action template page should not fetch runtime logs
- logs page should not fetch all pipeline templates
- deployment run page should not fetch every server resource card repeatedly

## 7.2 Logs must be virtualized

Large log views should use:

- incremental append
- fixed-height rows or chunk virtualization
- cursor-based history loading

## 7.3 Metrics should be snapshot plus detail

The list page should use current-state summaries only.

The detail page can fetch:

- time series
- exporter raw metrics
- process details

## 7.4 Command preview and execution state must be separated

Preview is configuration.

Execution output is runtime.

They should not be tied to the same refresh cycle.

## 8. Recommended implementation phases

## Phase 1: Stop the biggest performance mistakes

1. Replace whole-database rewrite storage with row-level tables and upserts
2. Remove live logs from SQLite hot path entirely
3. Store run logs as files plus in-memory tail buffers
4. Split Config Center / Deployment / Logs / Metrics APIs
5. Make exporter metrics current-state cache only

Expected effect:

- obvious responsiveness improvement
- lower write amplification
- better replay of failures

## Phase 2: Introduce a run/event model

1. Add `deployment_runs` and `deployment_run_steps`
2. Move from deployment-centric task state to run-centric event state
3. Add file-backed event artifacts
4. Add websocket replay from in-memory ring buffers

Expected effect:

- much cleaner runtime model
- better cancellation / restart / retry semantics

## Phase 3: Monitoring plane separation

1. Integrate Prometheus scraping for node_exporter and npu_exporter
2. Backend reads metrics from Prometheus query API
3. Keep only health summary in SQLite

Expected effect:

- monitoring no longer interferes with deployment control path

## Phase 4: Optional edge agent

1. Introduce per-server optional agent
2. Keep SSH as fallback
3. Move local diagnostics and log tail to agent

Expected effect:

- better scale
- cleaner jump-host traversal
- stronger run continuity

## 9. Immediate architectural decisions

These are my final recommendations.

### Decision 1

Keep SQLite only for control plane and summaries.

### Decision 2

Do not store runtime line logs in SQLite.

### Decision 3

Do not store every exporter sample in SQLite.

### Decision 4

Use append-only files for runtime logs and artifacts.

### Decision 5

Use in-memory buffers plus websocket for live log streaming.

### Decision 6

Use exporter scraping or Prometheus for metrics.

### Decision 7

Keep SSH for deployments now, but remove SSH from continuous monitoring.

### Decision 8

Restructure all persistence from "single JSON-like document store" to "normalized row-level tables".

## 10. What I would do next in this repository

If we start implementation immediately, I would do it in this order:

1. replace store layer with row-level repositories
2. add `deployment_runs` and `deployment_run_steps`
3. move deployment logs to file-backed storage
4. add runtime ring buffers and file replay API
5. split metrics cache from server config persistence
6. make Config Center, Logs, Metrics separate backend and frontend slices

That path gives the biggest performance win with the least architectural regret.

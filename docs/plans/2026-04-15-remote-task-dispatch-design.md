# Remote Task Dispatch Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a generic remote task dispatch capability that can send shell commands, remote script URLs, and built-in operational presets to one, many, or all robots.

**Architecture:** Keep deployment tasks untouched and introduce a separate `RemoteTask` model. Reuse the existing SSH collector and jump-host support for execution, add a dedicated backend dispatcher plus REST API, and expose the workflow through a new frontend task center page.

**Tech Stack:** Go backend, SQLite document store, React + TypeScript frontend, existing SSH collector, polling-based UI refresh.

---

### Task 1: Backend remote task domain and persistence

**Files:**
- Modify: `backend/internal/domain/types.go`
- Modify: `backend/internal/store/store.go`

**Step 1: Add `RemoteTask`, `RemoteTaskRun`, and `RemoteTaskPreset` types**

Define task scope, execution type, command preview, per-server run result, and timestamps.

**Step 2: Persist `remote_tasks` alongside existing document collections**

Load, save, clone, and initialize remote task slices in the SQLite-backed document store.

**Step 3: Verify persistence behavior**

Run: `go test ./backend/internal/...`
Expected: backend packages compile and store-backed API tests keep passing.

### Task 2: Backend execution and API

**Files:**
- Create: `backend/internal/collect/remote_exec.go`
- Create: `backend/internal/dispatch/executor.go`
- Create: `backend/internal/dispatch/presets.go`
- Create: `backend/internal/api/remote_tasks.go`
- Modify: `backend/internal/api/api.go`
- Modify: `backend/internal/api/api_test.go`

**Step 1: Add generic SSH command execution support**

Return stdout, stderr, exit code, and duration while preserving existing mock-server behavior.

**Step 2: Build a dedicated remote task dispatcher**

Resolve target robots for `all`, `project`, or `selected` scope, expand presets, and fan out execution in background goroutines.

**Step 3: Expose REST endpoints**

Add preset discovery, task creation, and task detail/list endpoints.

**Step 4: Add backend tests**

Cover the create-and-complete mock task flow and preset command generation.

### Task 3: Frontend task center

**Files:**
- Create: `src/components/TaskDispatchManager.tsx`
- Modify: `src/App.tsx`
- Modify: `src/components/Layout.tsx`
- Modify: `src/types/index.ts`

**Step 1: Add remote task types to the frontend model layer**

Model scopes, execution modes, task records, preset field schemas, and per-run results.

**Step 2: Build a task dispatch page**

Support raw command entry, script URL entry, preset selection with dynamic fields, and target-scope selection.

**Step 3: Show task history and per-robot results**

Poll the backend list endpoint, expand task cards, and display command/output/error for each robot run.

### Task 4: Docs and verification

**Files:**
- Modify: `backend/README.md`
- Modify: `API.md`
- Create: `docs/plans/2026-04-15-remote-task-dispatch-design.md`

**Step 1: Document the new remote task API**

Capture supported execution modes, endpoints, and preset examples.

**Step 2: Run validation**

Run:

```bash
cd backend && go test ./...
npm.cmd run typecheck
npm.cmd run build
```

Expected:

- backend tests pass
- TypeScript typecheck passes
- production frontend build succeeds

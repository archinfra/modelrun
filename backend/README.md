# ModelRun Backend

Go backend for the ModelDeploy/ModelRun UI.

## Run

```powershell
cd backend
go run ./cmd/modelrun
```

The service listens on `:8080` by default and writes SQLite state to
`backend/data/modelrun.db`.

## Environment

- `MODELRUN_ADDR`: listen address, for example `:8080`
- `MODELRUN_DATA`: SQLite data file path
- `MODELRUN_STATIC_DIR`: optional directory for built frontend assets

## Main endpoints

- `GET /api/health`
- `GET|POST /api/projects`
- `GET /api/projects/{id}/summary`
- `GET|POST /api/servers`
- `POST /api/servers/{id}/test`
- `GET /api/servers/{id}/resources`
- `GET|POST /api/models`
- `POST /api/models/scan`
- `GET /api/models/search?source=modelscope&q=qwen`
- `GET|POST /api/deployments`
- `POST /api/deployments/{id}/start`
- `POST /api/deployments/{id}/stop`
- `GET /api/deployments/{id}/logs`
- `GET /api/deployments/{id}/metrics`
- `GET /api/tasks?deploymentId={id}`
- `GET /ws`

The deployment executor currently simulates the SSH/Docker workflow and emits
progress, status, log, and metric messages over WebSocket. The API and storage
layers are shaped around the frontend TypeScript types so the real SSH/Docker
executor can be swapped in later without changing the UI contract.

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
- `GET /api/servers/{id}/gpu`
- `GET /api/servers/{id}/npu-exporter`
- `POST /api/servers/{id}/npu-exporter/install`
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

`POST /api/servers/{id}/test`, `GET /api/servers/{id}/resources`, and
`GET /api/servers/{id}/gpu` use real SSH collection. The collector supports
direct SSH and SSH through a server marked as a jump host. It reads Linux system
files plus `nvidia-smi`, `npu-smi info`, optional NPU Exporter Prometheus
metrics, and `docker --version`.

The deployment executor currently simulates the Docker workflow and emits
progress, status, log, and metric messages over WebSocket. Server discovery is
real; deployment execution can be replaced with a real SSH/Docker executor
without changing the UI contract.

See `../docs/server-collection.md` for NPU prerequisites, exporter guidance,
jump host setup, and troubleshooting.

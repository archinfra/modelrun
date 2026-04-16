# ModelRun Data Source, Config Center, and Pipeline Guide

## 1. How to disable mock data

The backend now treats remote collection and SSH execution as mock **only** when the environment variable below is explicitly enabled:

```bash
MODELRUN_FAKE_CONNECT=1
```

If this variable is not set, the backend will try real SSH collection and real remote command execution.

Important changes in this revision:

- The old host-name shortcut such as `mock-*` no longer triggers fake collection.
- The frontend model page no longer falls back to built-in mock model cards.
- The backend exposes `/api/system/status` so the UI can show:
  - whether mock collection is enabled
  - which collections are persisted in SQLite
  - which endpoints still contain demo content

## 2. What is persisted and what is still demo

### Persisted in SQLite

These collections are stored in `backend/data/modelrun.db` through the `documents` table:

- `projects`
- `servers`
- `jumpHosts`
- `models`
- `deployments`
- `tasks`
- `remoteTasks`
- `actionTemplates`
- `bootstrapConfigs`
- `logs`

### Still demo-only

The only remaining demo catalog in this revision is:

- `GET /api/models/search`

This endpoint is still a built-in catalog for search suggestions. It is **not** the source of truth for deployed models. Real model records come from:

- `GET /api/models`
- `POST /api/models`

and are persisted to SQLite.

## 3. Where server information comes from now

For a configured server, the backend gathers information through SSH and optional exporters.

### 3.1 Base host information

Collected over SSH by the backend:

- SSH connectivity
- Docker version
- CPU, memory, disk
- GPU or NPU device information

### 3.2 GPU information

For NVIDIA-style GPU nodes, the backend primarily collects device data directly over SSH.

### 3.3 NPU information

For Ascend NPU nodes, the backend can read NPU metrics from an exporter endpoint configured on the target node:

- default endpoint: `http://127.0.0.1:9101/metrics`
- alternate fallback endpoint: `http://127.0.0.1:8082/metrics`

The backend parses Prometheus metrics exposed by `npu_exporter` and maps them to:

- utilization
- temperature
- power
- HBM total/used/free
- health status

## 4. Do you need `npu_exporter` and `node_exporter`

### `npu_exporter`

For Ascend NPU metrics, **yes, it is strongly recommended**.

Why:

- direct SSH can confirm that the host is reachable
- direct SSH can read some base information
- but NPU telemetry such as utilization, HBM usage, and health is best exposed through `npu_exporter`
- this also aligns with the exporter metric model you shared from the Ascend collector source

So the backend now treats `npu_exporter` as a first-class bootstrap service.

### `node_exporter`

`node_exporter` is not mandatory for the current host summary pages, because the backend can still collect basic CPU, memory, and disk stats over SSH.

But it is recommended when you want:

- Prometheus-standard host metrics
- longer-term observability integration
- unified host-level metrics across clusters

So `node_exporter` is also seeded as a first-class bootstrap service in the new config center.

## 5. Config Center design

This revision adds a backend-managed Config Center with two persisted object types.

### 5.1 Action Templates

Action templates are reusable remote execution definitions. They are persisted and editable through:

- `GET /api/action-templates`
- `POST /api/action-templates`
- `PATCH /api/action-templates/:id`
- `DELETE /api/action-templates/:id`

Built-in defaults seeded on startup:

- `install_node_exporter`
- `install_npu_exporter`
- `install_modelscope_cli`
- `install_huggingface_cli`
- `docker_pull_image`
- `docker_restart_service`

The remote task dispatch page now reads its preset list from these persisted templates instead of hardcoded read-only presets.

Compatibility is kept for the previous preset id:

- `docker_install_npu_exporter`

### 5.2 Bootstrap Configs

Bootstrap configs describe service initialization items that should exist in the backend by default.

Persisted endpoints:

- `GET /api/bootstrap-configs`
- `POST /api/bootstrap-configs`
- `PATCH /api/bootstrap-configs/:id`
- `DELETE /api/bootstrap-configs/:id`

Built-in defaults seeded on startup:

- `Node Exporter`
- `NPU Exporter`
- `ModelScope CLI`
- `Hugging Face CLI`

Each bootstrap config references an action template and carries default arguments like image name, port, or endpoint.

## 6. Jump host behavior

Jump host support remains in place and is used by:

- server connectivity test
- host resource collection
- NPU exporter probing
- remote task dispatch
- deployment pipeline execution

Behavior:

1. If a server has `useJumpHost=true`, the backend first connects to the jump host over SSH.
2. The backend then opens an SSH tunnel from the jump host to the target server's internal IP.
3. All collection and command execution happens through that chained connection.

This means internal-only IP nodes can now participate in:

- inventory
- exporter checks
- task dispatch
- deployment pipelines

## 7. New pipeline console

The old next-step wizard has been replaced by a single-page pipeline console.

### 7.1 What changed

Old behavior:

- step-by-step wizard
- next/back navigation
- mostly simulated deployment execution

New behavior:

- single-page pipeline configuration
- pipeline board layout
- per-stage status
- expandable stage details
- per-server command and log visibility

### 7.2 Supported framework presets

Seeded pipeline templates:

- `tei`
- `vllm-ascend`
- `mindie`

Exposed through:

- `GET /api/pipeline-templates`

### 7.3 Default stage model

The backend now builds deployment tasks using these stages:

1. `prepare_model`
2. `pull_image`
3. `launch_runtime`
4. `verify_service`

This intentionally matches your requirement that:

- container startup
- optional Ray bootstrap
- framework configuration and service startup

should be treated as **one managed runtime launch action**, even if the UI still allows operators to inspect the detail separately.

### 7.4 Runtime auto-restart behavior

The generated runtime launch stage now:

- writes a managed launch script under the deployment work directory
- recreates the container
- starts the framework inside the container
- uses `--restart unless-stopped`

So after a container restart, the framework service comes back with the same launch logic rather than depending on a manual follow-up step.

## 8. Framework notes

### TEI

The TEI template prepares the model, pulls the image, launches `text-embeddings-router`, and verifies the service endpoint.

### vLLM Ascend

The `vllm-ascend` template:

- supports optional Ray bootstrap
- starts Ray head/worker inside the managed launch stage when enabled
- starts `vllm serve` in the same managed runtime action
- exposes OpenAI-compatible service verification

### MindIE

The `mindie` template:

- prepares a managed `config.json`
- recreates the container
- starts `mindieservice_daemon` from the managed launch stage

Because MindIE deployment details can vary by image packaging and local environment, operators should still validate image contents and config compatibility on their target environment.

## 9. Operational APIs added in this revision

- `GET /api/system/status`
- `GET /api/action-templates`
- `POST /api/action-templates`
- `PATCH /api/action-templates/:id`
- `DELETE /api/action-templates/:id`
- `GET /api/bootstrap-configs`
- `POST /api/bootstrap-configs`
- `PATCH /api/bootstrap-configs/:id`
- `DELETE /api/bootstrap-configs/:id`
- `GET /api/pipeline-templates`

## 10. Recommended production checks

Before using this in production, verify:

1. The backend process environment does not set `MODELRUN_FAKE_CONNECT=1`.
2. `npu_exporter` is installed and reachable on Ascend servers.
3. `modelscope` or `huggingface-cli` is available on target nodes if model download is remote.
4. Docker privileges are available for the SSH user, or passwordless `sudo` is configured.
5. For multi-node `vllm-ascend`, the chosen Ray head node and NIC are correct.
6. For `mindie`, the runtime image layout and generated config are validated against the target environment.

## 11. Official references used while adapting runtime defaults

- Hugging Face Text Embeddings Inference docs: [https://huggingface.co/docs/text-embeddings-inference/main/en/index](https://huggingface.co/docs/text-embeddings-inference/main/en/index)
- vLLM Ascend quickstart docs: [https://docs.vllm.ai/projects/ascend/en/main/quick_start.html](https://docs.vllm.ai/projects/ascend/en/main/quick_start.html)
- MindIE service deployment docs: [https://www.hiascend.com/document](https://www.hiascend.com/document)

# ModelRun

ModelRun packages a React console and Go API server for managing model
deployment workflows.

## Container Image

GitHub Actions builds and publishes the all-in-one image to GitHub Container
Registry on every push to `main`, and also on pushed git tags. When Aliyun
registry credentials are configured at the org or repo level, the same image
tags are pushed to Aliyun Container Registry in the same workflow run.

```bash
docker pull ghcr.io/archinfra/modelrun:latest
docker run --rm -p 8080:8080 -v modelrun-data:/var/lib/modelrun ghcr.io/archinfra/modelrun:latest
```

Open `http://localhost:8080` after the container starts. The same image includes
the `modelscope` CLI:

```bash
docker run --rm ghcr.io/archinfra/modelrun:latest modelscope --help
```

To enable Aliyun image publishing in Actions, configure these variables and
secrets at the GitHub organization or repository level:

- `ALIYUN_REGISTRY`
  Example: `registry.cn-hangzhou.aliyuncs.com`
- `ALIYUN_REGISTRY_NAMESPACE`
  Example: your organization namespace in Aliyun ACR
- `ALIYUN_IMAGE_NAME` (optional)
  Defaults to the GitHub repository name when omitted
- `ALIYUN_REGISTRY_USERNAME`
- `ALIYUN_REGISTRY_PASSWORD`

After that, branch pushes publish `branch`, `sha-*`, and `latest` tags, and git
tag pushes publish the matching git tag to both GHCR and Aliyun ACR.

Runtime paths:

- `MODELRUN_DATA=/var/lib/modelrun/modelrun.db`
- `MODELRUN_STATIC_DIR=/app/dist`
- `MODELSCOPE_CACHE=/var/lib/modelrun/modelscope`

## Server Collection

Server status, GPU/NPU inventory, NPU Exporter metrics, and Docker details are
collected by the Go backend over SSH. The frontend no longer fabricates GPU data
for newly added servers. Jump hosts are supported by marking one server as
`isJumpHost` and setting other internal servers to `useJumpHost`.

See [docs/server-collection.md](docs/server-collection.md) for the exact SSH
commands, NPU exporter guidance, jump host flow, and troubleshooting notes.

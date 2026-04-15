# ModelRun

ModelRun packages a React console and Go API server for managing model
deployment workflows.

## Container Image

GitHub Actions builds and publishes the all-in-one image to GitHub Container
Registry on every push to `main`.

```bash
docker pull ghcr.io/archinfra/modelrun:latest
docker run --rm -p 8080:8080 -v modelrun-data:/var/lib/modelrun ghcr.io/archinfra/modelrun:latest
```

Open `http://localhost:8080` after the container starts. The same image includes
the `modelscope` CLI:

```bash
docker run --rm ghcr.io/archinfra/modelrun:latest modelscope --help
```

Runtime paths:

- `MODELRUN_DATA=/var/lib/modelrun/modelrun.db`
- `MODELRUN_STATIC_DIR=/app/dist`
- `MODELSCOPE_CACHE=/var/lib/modelrun/modelscope`

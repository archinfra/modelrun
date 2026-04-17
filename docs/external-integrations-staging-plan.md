# External Integrations Staging Plan

## Goal

Keep the platform usable first.

For now:

- Keep backend support for Netdata and NPU exporter.
- Keep server-side data structures and APIs.
- Hide unstable or heavy external integrations from the main frontend flow.
- Prefer simple installation presets with safe defaults.

## Current Decision

### Netdata

- Backend support stays in place.
- Server data keeps `netdataEndpoint`, `netdataStatus`, and `netdataLastCheck`.
- Frontend dashboard embedding is intentionally hidden for now.
- Installation switches to Docker mode so rollout is predictable across servers.

### NPU Exporter

- The default built-in image is `swr.cn-south-1.myhuaweicloud.com/ascendhub/npu-exporter:v7.3.0`.
- The recommended default runtime args are:
  - `listenIP=0.0.0.0`
  - `port=8082`
  - `containerMode=docker`
- The backend still supports endpoint fallback probing:
  - `http://127.0.0.1:8082/metrics`
  - `http://127.0.0.1:9101/metrics`

## Why Hide Some Frontend Entries

The platform's main value right now is:

- server management
- task dispatch
- deployment pipeline
- runtime troubleshooting

Embedded third-party dashboards are useful, but they also introduce:

- extra render cost
- iframe instability
- more network dependencies
- more failure modes during bring-up

So the current rule is:

- backend capability can stay
- frontend exposure should be conservative

## Task Dispatch Defaults

Task dispatch presets must always provide visible defaults for operator-facing fields.

For `docker_install_npu_exporter`, the UI should show:

- image
- container name
- listen IP
- port

Blank values in the draft should fall back to preset defaults instead of rendering as empty.

## Recommended NPU Exporter Docker Shape

The current platform assumption is a host-network deployment with explicit listener args.

Example shape:

```sh
docker run -d \
  --name modelrun-npu-exporter \
  --restart unless-stopped \
  --network host \
  --privileged \
  -v /dev:/dev \
  -v /usr/local/Ascend:/usr/local/Ascend:ro \
  -v /usr/local/dcmi:/usr/local/dcmi:ro \
  -v /sys:/sys:ro \
  -v /tmp:/tmp \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /etc/localtime:/etc/localtime:ro \
  swr.cn-south-1.myhuaweicloud.com/ascendhub/npu-exporter:v7.3.0 \
  -ip=0.0.0.0 \
  -port=8082 \
  -containerMode=docker
```

## Next Steps

1. Keep verifying 7.3.0 metric compatibility against real hosts.
2. Surface backend diagnostic messages clearly in the server UI.
3. Re-open Netdata frontend integration only after the core workflow is stable.
4. Add a lightweight operator guide for exporter troubleshooting.

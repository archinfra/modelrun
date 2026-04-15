# 服务器采集、NPU 与跳板机说明

ModelRun 的服务器信息采集走后端 SSH 实时探测，不再由前端 mock 数据生成。前端点击“采集信息”时会调用：

- `POST /api/servers/{id}/test`：建立 SSH，采集加速卡、系统资源、Docker 版本，并更新 SQLite 中的服务器状态。
- `GET /api/servers/{id}/gpu`：只采集 GPU/NPU 加速卡信息。
- `GET /api/servers/{id}/resources`：只采集 CPU、内存、磁盘资源。
- `GET /api/servers/{id}/npu-exporter`：通过 SSH 从目标机本地访问 NPU Exporter `/metrics`，检查 exporter 是否可用。
- `POST /api/servers/{id}/npu-exporter/install`：通过 SSH 下发 NPU Exporter 安装命令。

## 后端信息来源

后端会使用服务器配置里的 SSH 信息连接目标机器。目标机器需要是 Linux 环境，且 SSH 用户至少能读取 `/proc`、执行 `df`，并能运行对应硬件命令。

当前采集来源如下：

| 信息 | 来源 |
| --- | --- |
| NVIDIA GPU | `nvidia-smi --query-gpu=index,name,memory.total,memory.used,memory.free,utilization.gpu,temperature.gpu,power.draw,power.limit --format=csv,noheader,nounits` |
| NVIDIA 驱动 | `nvidia-smi --query-gpu=driver_version --format=csv,noheader,nounits` |
| CUDA 版本 | `nvidia-smi` 输出中的 `CUDA Version` |
| Ascend NPU | `npu-smi info` |
| Ascend NPU Exporter | 默认从目标机访问 `http://127.0.0.1:9101/metrics`，解析 `npu_chip_info_*` Prometheus 指标 |
| CPU 核数 | `getconf _NPROCESSORS_ONLN` 或 `nproc` |
| CPU 使用率 | `/proc/stat`，间隔 1 秒采样两次计算 |
| 内存 | `/proc/meminfo` 的 `MemTotal` 与 `MemAvailable` |
| 根分区磁盘 | `df -Pm /` |
| Docker 版本 | `docker --version` |

如果目标机器没有 NVIDIA GPU，也没有 Ascend NPU，采集仍会成功，但加速卡列表为空。前端会显示“未采集”或 0 张加速卡，而不会再伪造 A100 数据。

## NPU 是否必须安装 exporter

不必须。

ModelRun 当前做的是“资产与部署前检查”级别的信息获取，只要目标机器安装了 Ascend 驱动/CANN，并且 SSH 用户能直接执行 `npu-smi info`，后端就可以识别 NPU 型号、健康状态、温度、功耗、AI Core 利用率和 HBM 使用情况。

结合 Ascend `npu-exporter` 源码看，它本质上是一个运行在 NPU 节点本机的 Prometheus exporter：collector 里通过 DCMI/devmanager 读取 NPU、HBM、DDR、HCCS、RoCE、PCIe、光模块等指标，再暴露成 `npu_chip_info_*`、`container_npu_*` 这类 Prometheus 文本指标。它依赖目标机的 Ascend 驱动库、设备文件、容器/K8s 上下文，不适合把源码直接编进 ModelRun 管理端。

ModelRun 采用的方案是：

- 轻量场景：不装 exporter，直接 SSH 执行 `npu-smi info`。
- 监控场景：在 NPU 节点安装 exporter，ModelRun 通过 SSH 访问目标机本地 `/metrics` 并解析核心指标。
- 安装动作：由 ModelRun 通过 SSH 下发 Docker 或自定义安装命令，不把 exporter 源码嵌入 ModelRun。

建议安装 `npu_exporter` 或类似 Prometheus exporter 的场景：

- 你需要连续监控曲线，而不是点击一次采集一次。
- 你需要告警、历史趋势、Grafana 面板。
- 你需要更细的 NPU 指标，比如链路、芯片错误计数、进程级指标。

不安装 exporter 的限制：

- ModelRun 只能在调用 API 时通过 SSH 现场采样。
- 没有历史时间序列。
- 当前网络速率字段暂时返回 0，后续可以接入 exporter 或 `/proc/net/dev` 增量采样。

已接入的 exporter 指标：

| ModelRun 字段 | Exporter 指标 |
| --- | --- |
| NPU 名称 | `npu_chip_info_name` 的 `name` 或其他指标的 `modelName` label |
| 利用率 | `npu_chip_info_utilization` 或 `npu_chip_info_overall_utilization` |
| 温度 | `npu_chip_info_temperature` |
| 功耗 | `npu_chip_info_power` |
| HBM 总量 | `npu_chip_info_hbm_total_memory` |
| HBM 已用 | `npu_chip_info_hbm_used_memory` |
| 健康状态 | `npu_chip_info_health_status`，值为 `1` 视为 OK |

## NPU 目标机检查

在目标服务器上先确认：

```bash
which npu-smi
npu-smi info
```

如果交互式登录能执行，但 ModelRun 采集失败，通常是非交互 SSH 环境没有加载同样的 PATH。可以选择一种方式处理：

```bash
sudo ln -s /usr/local/Ascend/driver/tools/npu-smi /usr/local/bin/npu-smi
```

或者在 SSH 用户的 shell profile 里补齐 Ascend 工具路径。

## NPU Exporter 检查与安装

默认 endpoint 是：

```text
http://127.0.0.1:9101/metrics
```

这个地址是从目标服务器本机访问的，不是从浏览器或 ModelRun 容器直接访问。通过跳板机连接内网服务器时，后端仍会先 SSH 到目标服务器，然后在目标服务器上执行 `curl` 或 `wget` 访问本地 exporter。

检查 exporter：

```bash
curl -X GET http://127.0.0.1:8080/api/servers/{serverId}/npu-exporter
```

如果你已经有 exporter 镜像，可以让 ModelRun 通过 SSH 下发 Docker 安装命令：

```bash
curl -X POST http://127.0.0.1:8080/api/servers/{serverId}/npu-exporter/install \
  -H 'Content-Type: application/json' \
  -d '{
    "mode": "docker",
    "image": "your-registry/npu-exporter:v6.0.0",
    "port": 9101
  }'
```

后端下发的 Docker 命令会使用：

```bash
docker rm -f modelrun-npu-exporter || true
docker run -d --name modelrun-npu-exporter --restart unless-stopped --network host --privileged \
  -v /dev:/dev \
  -v /usr/local/Ascend:/usr/local/Ascend:ro \
  -v /etc/localtime:/etc/localtime:ro \
  your-registry/npu-exporter:v6.0.0
```

如果你的安装方式不是 Docker，可以直接传自定义命令：

```bash
curl -X POST http://127.0.0.1:8080/api/servers/{serverId}/npu-exporter/install \
  -H 'Content-Type: application/json' \
  -d '{
    "mode": "command",
    "endpoint": "http://127.0.0.1:9101/metrics",
    "command": "systemctl restart npu-exporter || /usr/local/bin/npu-exporter >/var/log/npu-exporter.log 2>&1 &"
  }'
```

建议生产环境优先把 exporter 镜像放到你自己的内网镜像仓库。ModelRun 不内置第三方 exporter 镜像名称，避免拉错版本或引入不可控供应链。

## 跳板机连接方式

可以把任意服务器标记为跳板机，然后让其他服务器填写内网 IP 并通过它转发 SSH。

配置步骤：

1. 在“服务器管理”添加一台公网可访问的服务器。
2. 勾选“这台服务器可作为跳板机”。
3. 再添加内网服务器，主机地址填写内网 IP。
4. 勾选“通过跳板机连接”，选择刚才那台跳板机。
5. 点击“采集信息”验证链路。

后端实际链路：

```text
ModelRun 后端 -> SSH 跳板机 -> SSH 目标内网服务器
```

要求：

- ModelRun 后端所在机器能访问跳板机的 SSH 端口。
- 跳板机能访问目标服务器的内网 IP 与 SSH 端口。
- 跳板机和目标服务器可以使用不同的用户名、密码或私钥。
- 目标服务器不能选择自己作为自己的跳板机。

服务器 JSON 示例：

```json
{
  "name": "bastion-01",
  "host": "203.0.113.10",
  "sshPort": 22,
  "username": "root",
  "authType": "key",
  "privateKey": "-----BEGIN OPENSSH PRIVATE KEY-----\n...",
  "isJumpHost": true,
  "useJumpHost": false
}
```

内网 NPU 服务器 JSON 示例：

```json
{
  "name": "npu-node-01",
  "host": "10.10.1.23",
  "sshPort": 22,
  "username": "root",
  "authType": "password",
  "password": "******",
  "useJumpHost": true,
  "jumpHostId": "server_xxxxxx"
}
```

## 认证与安全边界

当前支持：

- 密码认证。
- OpenSSH 私钥认证。
- 带 passphrase 的私钥，`password` 字段会作为 passphrase 尝试。
- keyboard-interactive 密码认证。

注意：

- 服务器密码和私钥会随服务器配置保存在 SQLite 的 JSON payload 里。生产环境请限制 `MODELRUN_DATA` 所在目录权限，并优先使用权限收敛的专用 SSH 用户。
- 当前 SSH host key 使用 `InsecureIgnoreHostKey`，适合内网快速落地。生产环境建议后续接入 `known_hosts` 校验，防止中间人攻击。
- SSH 用户不需要 root，但需要能执行 `nvidia-smi`、`npu-smi`、`docker --version`，并读取 `/proc/stat`、`/proc/meminfo`、`df -Pm /`。

## 常见问题

`password or privateKey is required`

服务器或跳板机没有配置密码，也没有配置私钥。

`jumpHostId is required when useJumpHost is true`

启用了跳板机连接，但没有选择跳板机。

`jump host not found`

目标服务器配置的 `jumpHostId` 在当前后端存储里不存在。重新保存目标服务器，选择现有跳板机。

`ssh connect ...`

ModelRun 后端到目标服务器或跳板机网络不通，检查防火墙、安全组、端口和 SSH 服务。

`jump host dial target ...`

后端能连上跳板机，但跳板机访问不了目标内网 IP 或端口。

`npu-smi: command not found`

目标机没有安装 Ascend 驱动/CANN，或 SSH 非交互环境 PATH 没有包含 `npu-smi`。

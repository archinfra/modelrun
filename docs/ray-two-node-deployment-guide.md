# Ray 双机部署说明

本文说明 ModelRun 里 vLLM Ascend + Ray 的双机部署方式，以及 `Qwen/Qwen3.5-397B-A17B` 这类大模型应该如何理解参数。

## 1. Ray 在这里负责什么

在当前实现里，Ray 只负责多机资源组网和 worker 管理，不负责替代 vLLM 自身的模型参数配置。

启用 Ray 后：

- `head` 节点负责启动 Ray 集群，并承载 `vllm serve`
- `worker` 节点只负责执行 `ray start --address ...` 加入集群
- worker 不会重复执行 `vllm serve`
- 对外 API 入口只保留 head 节点

这和单机模式有一个关键差别：单机是每台机器都拉起服务；Ray 模式下只有 head 暴露推理服务，worker 只提供算力。

## 2. 前端现在需要配置什么

在部署流水线页面，启用 Ray 后需要重点配置以下内容：

1. 参与部署的服务器列表
2. 哪一台是 `Ray Head`
3. `Ray 通信网卡名`
4. `Ray 端口` 与 `Dashboard 端口`
5. 每台机器自己的 `node IP`
6. 每台机器自己的 `可见设备`
7. vLLM 的 `TP/PP` 参数

其中第 5 和第 6 项现在已经支持按服务器单独填写。

## 3. 两台服务器为什么参数不一样

因为 Ray head 和 worker 的启动职责不同。

### Head 节点

head 会执行类似下面的命令：

```bash
ray start --head --port 6379 --dashboard-host 0.0.0.0 --dashboard-port 8265 --node-ip-address <head-node-ip>
vllm serve /model --distributed-executor-backend ray ...
```

### Worker 节点

worker 会执行类似下面的命令：

```bash
ray start --address <head-node-ip>:6379 --node-ip-address <worker-node-ip>
```

worker 不会再执行 `vllm serve`，它只需要保持容器常驻并加入 Ray 集群。

## 4. `node IP` 和 `通信网卡名` 分别是什么

这两个概念不要混掉。

- `node IP`：该节点实际参与 Ray/HCCL 通信时使用的 IP
- `通信网卡名`：该 IP 对应的网卡名，例如 `eth0`、`bond0`

当前后端会把：

- `node IP` 下发给 `HCCL_IF_IP` 和 `ray start --node-ip-address`
- `通信网卡名` 下发给 `HCCL_SOCKET_IFNAME`、`GLOO_SOCKET_IFNAME`、`TP_SOCKET_IFNAME`

如果两台机器的对外管理 IP 和组网 IP 不一样，应该在每台服务器卡片里把 `node IP` 改成真正参与组网的内网地址。

## 5. TP / PP 应该怎么理解

在双机场景里，一个常见起步思路是：

- `TP = 每台机器参与推理的卡数`
- `PP = 机器台数`

例如：

- 两台 910B
- 每台 8 张 NPU

通常可以先从下面这组参数开始：

```text
TP = 8
PP = 2
Enable Expert Parallel = true
```

这只是起步值，不是绝对值。后续还要结合显存占用、吞吐和实际框架兼容性继续调。

## 6. 关于 `Qwen/Qwen3.5-397B-A17B`

这个模型要特别注意版本和资源级别。

按照 vLLM Ascend 当前教程：

- 原始 BF16 版 `Qwen/Qwen3.5-397B-A17B` 更接近 `4 x Atlas A2` 或 `2 x Atlas A3`
- `2 x Atlas A2` 对应的是量化版 `Qwen3.5-397B-A17B-w8a8`

所以如果你现在手里是两台 910B / A2 服务器：

- 更稳妥的选择是量化版
- 如果坚持原始 BF16 版，资源大概率不够，需要额外机器

## 7. 当前后端已经做的处理

这次改动后，后端行为变成了：

1. Ray 模式下自动区分 head / worker
2. `vllm serve` 会显式带上 `--distributed-executor-backend ray`
3. worker 不再重复启动推理服务
4. worker 的验证步骤改成容器内执行 `ray status`
5. 对外 endpoint 只保留 head
6. 支持每台机器独立设置：
   - `node IP`
   - `visibleDevices`
   - `ray start` 额外参数

## 8. 失败后如何修改并重跑

部署流水线页面现在支持：

1. 在“最近部署”里点击 `载入编辑`
2. 修改 Ray / TP / PP / 路径 / 模型参数
3. 点击 `保存并重跑`

如果只是想沿用原参数重新跑一次，也可以直接点 `直接重跑`。

## 9. 推荐阅读

- vLLM Ascend Ray 教程: https://docs.vllm.ai/projects/ascend/zh-cn/main/tutorials/features/ray.html
- vLLM Ascend Qwen3.5-397B-A17B 教程: https://docs.vllm.ai/projects/ascend/en/main/tutorials/models/Qwen3.5-397B-A17B.html

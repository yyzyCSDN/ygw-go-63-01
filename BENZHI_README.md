# ModelRouter

ModelRouter 是一个自包含的模型服务路由与版本网关示例项目。它提供模型注册、
版本发布、别名路由、灰度放量、健康实例路由、并发配额与失败回退等能力，并带
一个简单的 Web 控制台页面。

## 构建

```bash
go build -mod=vendor ./...
go test -mod=vendor ./...
go vet -mod=vendor ./...
```

依赖已 vendor 化，构建全程离线（GOPROXY=off）。

## 运行

```bash
go run ./cmd/modelrouter
```

默认监听 `:8080`，可用环境变量调整：

- `MODELROUTER_ADDR`：监听地址，默认 `:8080`
- `MODELROUTER_TIMEOUT_MS`：单次转发超时毫秒数，默认 5000
- `MODELROUTER_QUOTA`：并发配额，默认 64
- `MODELROUTER_GRAY_PERIOD`：灰度窗口周期，默认 100
- `MODELROUTER_HEALTH_MS`：健康检查间隔毫秒数，默认 10000
- `MODELROUTER_SEED`：启动目录 JSON（可选）

## 接口

- `GET /v1/healthz`：健康检查
- `GET /v1/console`：控制台页面
- `POST /v1/models/{model}/infer`：模型推理路由
- `POST /v1/admin/register`：注册模型版本与实例
- `POST /v1/admin/publish`：发布版本
- `POST /v1/admin/alias`：配置版本别名
- `POST /v1/admin/retire`：退役版本
- `POST /v1/admin/sync`：从上游同步模型目录
- `GET /v1/admin/models`：模型与路由状态
- `GET /v1/admin/instances`：健康实例列表
- `GET /v1/metrics`：指标快照

## 示例种子目录

```json
{
  "models": [
    {
      "name": "resnet",
      "version": "v1",
      "instances": [
        {"id": "resnet-a", "addr": "127.0.0.1:9101"}
      ]
    }
  ]
}
```

将上述 JSON 放入环境变量 `MODELROUTER_SEED` 后启动，网关即可对 `resnet`
进行路由。

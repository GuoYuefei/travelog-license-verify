# travelog-license-verify

**Travelog 许可证验证 Go 库** — 用于验证 Travelog License Server 生成的 `.lic` 许可证文件。

支持**离线验证**（Ed25519 签名）、**在线激活/心跳**、**Chi 中间件**。

## 安装

```bash
go get home.naturgift.fun/aiwork/travelog-license-verify
```

> ⚠️ 这是一个私有仓库，Go 需要额外配置才能下载：
>
> ```bash
> # 1. 告诉 Go 这个域名不走代理
> go env -w GOPRIVATE=home.naturgift.fun
>
> # 2. 配置 Git 用 SSH 认证
> git config --global url."ssh://git@home.naturgift.fun:2222/".insteadOf "https://home.naturgift.fun/"
> ```

## 快速开始

### 最简用法（一行验证）

```go
package main

import (
    "fmt"
    "home.naturgift.fun/aiwork/travelog-license-verify"
)

func main() {
    lic, err := verify.VerifyFileWithKeyFile("license.lic", "public.pem")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Licensed to: %s (valid: %v)\n", lic.CustomerName, lic.IsValid())
}
```

### 离线验证 .lic 文件

```go
// 从 PEM 文件加载公钥
pubKey, err := verify.LoadPublicKey("public.pem")

// 验证 .lic 文件（签名 + 过期 + 吊销）
lic, err := verify.VerifyFile("license.lic", pubKey)

// 检查状态
if lic.IsExpired() {
    fmt.Println("许可证已过期")
}
if lic.IsRevoked() {
    fmt.Println("许可证已被吊销")
}
if lic.IsValid() {
    fmt.Println("许可证有效")
}

// 检查功能开关
if lic.IsFeatureEnabled("export_pdf") {
    fmt.Println("支持 PDF 导出")
}

// 读取配置参数（类型安全）
storage, _ := verify.GetCapabilityInt(lic.Capabilities, "storage_gb")
fmt.Printf("存储容量: %d GB\n", storage)
```

### 在线验证（HTTP 客户端）

```go
client := verify.NewClient("http://localhost:9443")

// 验证许可证状态
result, err := client.Verify(ctx, "license-key-xxx")

// 设备激活
result, err := client.Activate(ctx, verify.ActivateRequest{
    LicenseKey:        "license-key-xxx",
    DeviceFingerprint: "cpu+mobo+mac-hash",
    Hostname:          "my-pc",
    Platform:          "windows",
})

// 心跳保活
result, err := client.Heartbeat(ctx, "license-key-xxx", "device-fingerprint")
```

### Chi 中间件

```go
import (
    "github.com/go-chi/chi/v5"
    "home.naturgift.fun/aiwork/travelog-license-verify"
)

r := chi.NewRouter()

// 注册中间件：自动验证每个请求的许可证
r.Use(verify.Middleware(
    verify.WithLicensePath("license.lic"),
    verify.WithPublicKey(pubKey),
))

// 在 handler 中获取已验证的许可证
r.Get("/api/status", func(w http.ResponseWriter, r *http.Request) {
    lic := verify.FromContext(r.Context())
    // ...
})
```

## API 概览

### 核心验证

| 函数 | 说明 |
|------|------|
| `VerifyBytes(data, pubKey)` | 解析字节数组，验证 Ed25519 签名，返回 License |
| `VerifyFile(path, pubKey)` | 读取 .lic 文件并验证 |
| `VerifyFileWithKeyFile(licPath, keyPath)` | 从文件加载 .lic 和公钥，一步验证 |
| `Decode(data)` | 仅解析 .lic 格式，不验签（返回 LicenseFile）|
| `Verify(pub, data, sig)` | 底层 Ed25519 验签 |
| `ParsePublicKey(pemData)` | PEM → Ed25519 公钥 |
| `LoadPublicKey(path)` | 从文件加载 PEM 公钥 |
| `GenerateKey()` | 生成 Ed25519 密钥对 |

### License 方法

| 方法 | 说明 |
|------|------|
| `IsValid()` | 未过期且未吊销 |
| `IsExpired()` | 检查过期时间（0 表示永不过期）|
| `IsRevoked()` | 检查是否被吊销 |
| `IsFeatureEnabled(name)` | 检查功能开关 |
| `GetCapability(key)` | 获取配置参数值 |

### 类型安全 Capability 读取

| 函数 | 说明 |
|------|------|
| `GetCapabilityString(caps, key)` | 读取字符串值 |
| `GetCapabilityInt(caps, key)` | 读取整数值 |
| `GetCapabilityBool(caps, key)` | 读取布尔值 |

### HTTP 客户端

| 方法 | 说明 |
|------|------|
| `NewClient(serverURL)` | 创建客户端 |
| `Verify(ctx, licenseKey)` | 在线验证许可证 |
| `Activate(ctx, req)` | 设备激活 |
| `Heartbeat(ctx, licenseKey, deviceFP)` | 心跳保活 |

### 中间件选项

| 选项 | 说明 |
|------|------|
| `WithLicensePath(path)` | .lic 文件路径 |
| `WithPublicKey(pub)` | Ed25519 公钥 |
| `WithCacheDuration(d)` | 缓存时长（减少磁盘 I/O）|
| `WithErrorHandler(fn)` | 自定义验签失败处理 |
| `WithInvalidHandler(fn)` | 自定义过期/吊销处理 |

## 许可证文件格式 (.lic)

此库完全兼容 Travelog License Server 生成的 `.lic` 文件：

```
-----BEGIN TRAVELOG LICENSE-----
format: travelog-license-v1
algorithm: Ed25519
signer_key: <base64 公钥>
payload: <base64 JSON 许可证数据>
signature: <Ed25519 签名>
-----END TRAVELOG LICENSE-----
```

- 签名算法：Ed25519（Go 标准库 `crypto/ed25519`）
- 签名覆盖：JSON payload 的原始字节（非整个文本文件）
- 零外部依赖：核心验签仅使用 Go 标准库

## 关于依赖

- **核心验证**：零外部依赖（仅 `crypto/ed25519`, `encoding/base64`, `encoding/json`, `encoding/pem`）
- **HTTP 客户端**：仅 `net/http` 标准库
- **Chi 中间件**：兼容标准 `net/http` 和 `go-chi/chi/v5`

## 测试

```bash
# 运行所有测试
go test ./... -v -count=1

# 先生成测试数据
go run testdata/gen_testdata.go

# 再运行测试
go test ./... -v -count=1
```

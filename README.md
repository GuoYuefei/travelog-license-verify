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

### 解析与验证

| 函数 / 方法 | 说明 |
|---|---|
| `Decode(data)` | 解析 .lic 字节 → `LicenseFile`（仅解析信封，不验签）|
| `LicenseFile.Verify(pub)` | 对已解析的 LicenseFile 验签（pub=nil 则使用内嵌公钥）|
| `LicenseFile.ParsePayload()` | 将 payload JSON 解析为 `License` 结构体 |
| `VerifyBytes(data, pubKey)` | Decode + Verify + ParsePayload 一步完成 |
| `VerifyFile(path, pubKey)` | 读取 .lic 文件并验证 |
| `VerifyFileWithKeyFile(licPath, keyPath)` | 从文件加载 .lic 和公钥，一步验证 |
| `Verify(pub, data, sig)` | 底层 Ed25519 签名验证 |

### 密钥管理

| 函数 | 说明 |
|---|---|
| `ParsePublicKey(pemData)` | PEM → Ed25519 公钥（支持 `"ED25519 PUBLIC KEY"` 和 `"PUBLIC KEY"` 两种类型）|
| `ParsePublicKeyRaw(raw)` | 原始 32 字节 → Ed25519 公钥 |
| `LoadPublicKey(path)` | 从文件加载 PEM 公钥 |
| `PublicKeyToPEM(pub)` | Ed25519 公钥 → `"ED25519 PUBLIC KEY"` PEM |
| `GenerateKey()` | 生成 Ed25519 密钥对 |
| `VerifyKeyMatch(given, expected)` | 常量时间比较两个公钥是否一致 |

### License 方法

| 方法 | 说明 |
|---|---|
| `IsValid()` | 未过期且未吊销 |
| `IsExpired()` | ExpiresAt > 0 且已超过（0 表示永不过期）|
| `IsRevoked()` | RevokedAt > 0 表示已吊销 |
| `IsFeatureEnabled(name)` | 检查 `Features` 中某项是否启用 |
| `GetCapability(key)` | 从 `Capabilities` map 取任意类型值 |
| `CapabilityKeys()` | 返回排序后的所有 capability key 列表 |
| `RangeCapabilities(fn)` | 按键名排序遍历所有 capability |

### 类型安全 Capability 读取

| 函数 | 说明 |
|---|---|
| `GetCapability(caps, key)` | 通用读取（返回 `any, bool`）|
| `GetCapabilityString(caps, key)` | 读取字符串值 |
| `GetCapabilityInt(caps, key)` | 读取整数值（自动处理 JSON `float64` → `int`）|
| `GetCapabilityBool(caps, key)` | 读取布尔值 |

### 常量与错误

| 符号 | 值 / 说明 |
|---|---|
| `LicenseTypeTrial` / `Standard` / `Enterprise` | 许可证类型常量 |
| `PublicKeySize` | Ed25519 公钥字节数（32）|
| `DefaultHTTPTimeout` | HTTP 客户端默认超时（30s）|
| `ErrInvalidFormat` | .lic 文件格式无效 |
| `ErrMissingField` | .lic 缺少必填字段 |
| `ErrInvalidPublicKey` | 公钥格式无效 |
| `ErrInvalidSignature` | 签名验证失败 |
| `ErrKeyMismatch` | 提供的公钥与 .lic 内嵌密钥不匹配 |

### HTTP 客户端

**创建客户端：**

| 函数 | 说明 |
|---|---|
| `NewClient(serverURL)` | 创建客户端（默认 30s 超时）|
| `NewClientWithHTTP(serverURL, httpClient)` | 使用自定义 `http.Client` 创建 |

**API 端点调用：**

| 方法 | 说明 | 路径 |
|---|---|---|
| `Verify(ctx, licenseKey)` | 在线验证许可证状态 | `GET /api/v1/client/verify/{key}` |
| `Activate(ctx, req)` | 设备激活 | `POST /api/v1/client/activate` |
| `Heartbeat(ctx, licenseKey, deviceFP)` | 心跳保活 | `POST /api/v1/client/heartbeat` |

**便捷方法（自动获取本机指纹）：**

| 方法 | 说明 |
|---|---|
| `ActivateLocalDevice(ctx, licenseKey)` | 自动检测指纹 + 主机名 + 平台，再激活 |
| `HeartbeatLocalDevice(ctx, licenseKey)` | 自动检测指纹后发送心跳 |

**辅助函数：**

| 函数 | 说明 |
|---|---|
| `Hostname()` | 返回系统主机名 |
| `Platform()` | 返回 `runtime.GOOS`（windows/linux/darwin 等）|

**响应类型字段：**

| 类型 | 主要字段 |
|---|---|
| `VerifyResult` | Valid, Expired, Revoked, ExpiresAt, MaxDevices, ActiveDevices, Product, LicenseType, CustomerName, Features, Capabilities, Error |
| `ActivateResult` | Status, Device, Error |
| `HeartbeatResult` | Status, LicenseExpired, LicenseRevoked, ExpiresAt, Device, Error |

### 设备指纹

| 函数 | 说明 |
|---|---|
| `DeviceFingerprint(ctx)` | 生成硬件绑定的设备标识（SHA-256(CPU+主板+MAC)）|
| `MustDeviceFingerprint(ctx)` | 同上，失败时 panic |

### 中间件

| 函数 / 选项 | 说明 |
|---|---|
| `Middleware(opts...)` | 创建 HTTP 中间件（兼容 `net/http` 和 `chi`）|
| `FromContext(ctx)` | 从 request context 取出已验证的 `License` |
| `WithLicensePath(path)` | .lic 文件路径（必填） |
| `WithPublicKey(pub)` | Ed25519 公钥（nil 则使用内嵌公钥）|
| `WithCacheDuration(d)` | 缓存时长；0 表示不缓存，每次请求重新读取 |
| `WithErrorHandler(fn)` | 自定义验签失败处理（默认返回 500）|
| `WithInvalidHandler(fn)` | 自定义过期/吊销处理（默认返回 403）|

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
# 1. 先生成测试数据（.lic 文件和密钥对）
go run testdata/gen_testdata.go

# 2. 运行所有测试（-count=1 禁用缓存）
go test ./... -v -count=1
```

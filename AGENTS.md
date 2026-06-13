# travelog-license-verify

Go 许可证验证库（package `verify`），用于解析和验证 Travelog License Server 生成的 `.lic` 文件。

## 关键事实

### 项目结构

- 包名：`verify`（import 路径很长但包名很短，引用时用 `verify.XXX`）
- Go 1.26.1，依赖极少：仅 `testify`（测试）和 `machineid`（指纹）
- 私有仓库 `home.naturgift.fun`，需要配置 `GOPRIVATE` 和 Git SSH 替代

### 核心验证流程

```
.lic 文件 → Decode() → LicenseFile → Verify(公钥) → ParsePayload() → License
                          ↓ 
                    VerifyBytes() / VerifyFile() 一步到位
```

- 签名算法为 Ed25519（标准库 `crypto/ed25519`），签名覆盖 **JSON payload 原始字节**，非整个文件
- `.lic` 文件是 PEM 风格信封格式，见 `verify.go` 中 `Decode()`
- `VerifyBytes(data, nil)` — 传 nil 公钥则使用 `.lic` 内嵌公钥验证
- `ExpiresAt == 0` 表示永不过期；`RevokedAt > 0` 表示已吊销

### 测试

- 测试与生产代码同包（`package verify`），可访问 `helpers_test.go` 中的辅助函数
- **必须先生成测试数据**：`go run testdata/gen_testdata.go`
- 运行所有测试：`go test ./... -v -count=1`
- `count=1` 禁用缓存，建议始终加上
- `helpers_test.go` 包含 `generateTestKey()`, `makeValidLicense()`, `makeExpiredLicense()`, `generateTestLic()` 等

### 公钥

- `ParsePublicKey()` 同时支持 `"ED25519 PUBLIC KEY"` 和 `"PUBLIC KEY"` 两种 PEM 类型
- `LoadPublicKey()` 从 PEM 文件加载
- `GenerateKey()` 生成 Ed25519 密钥对
- `PublicKeyToPEM()` 编码为 `"ED25519 PUBLIC KEY"` PEM

### HTTP 客户端

- 客户端 API 路径：`/api/v1/client/verify/{key}`, `/api/v1/client/activate`, `/api/v1/client/heartbeat`
- 默认超时 30s，可用 `NewClientWithHTTP()` 自定义 HTTP client
- 服务器返回 JSON，即使 HTTP 状态码非 2xx 也尝试解析 body（见 `client.go`）

### 中间件

- 兼容 `net/http` 和 `chi`
- 默认：验证失败返回 500，过期/吊销返回 403
- 缓存使用 `sync.RWMutex` + double-check locking，`CacheDuration=0` 不缓存
- `FromContext(ctx)` 从 context 取已验证的 License

### Capabilities

- JSON 中数字解码为 `float64`，使用 `GetCapabilityInt()` 自动处理类型转换
- 迭代能力：`CapabilityKeys()` 返回排序后的 key 列表，`RangeCapabilities()` 按序回调

### 触发词

- `"/license"`, `"license verify"`, `"许可证验证"` → 此仓库

### 编译产物

- `*.exe`, `*.out`, `*.test` 在 `.gitignore` 中，提交前注意清理

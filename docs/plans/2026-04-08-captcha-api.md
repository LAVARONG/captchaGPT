# Captcha API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 构建一个可部署的 Go 在线 API 服务，提供 `POST /api/getCode` 接口，接收验证码图片 Base64 与识别约束信息，调用 NVIDIA 托管的 `google/gemma-4-31b-it` 多模态能力完成验证码识别，并以 JSON 返回结果。

**Architecture:** 服务采用分层结构：HTTP 路由层负责鉴权、限流、请求校验和统一错误输出；应用服务层负责 Base64 解码、临时图片保存、提示词构造、上游大模型请求和结果解析；基础设施层负责配置加载、日志、HTTP 客户端、临时文件和限流器。接口设计尽量贴近 OpenAI 风格的消息输入与 JSON 输出，方便未来切换到其他兼容大模型 API。

**Tech Stack:** Go 1.24+, Gin 或 Chi、`godotenv`、`golang.org/x/time/rate`、标准库 `net/http`、`context`、`encoding/base64`、`mime`、`image/*`、`testing`、Docker

---

### Task 1: 初始化项目骨架与基础配置

**Files:**
- Create: `I:/CaptchaGPT/go.mod`
- Create: `I:/CaptchaGPT/cmd/api/main.go`
- Create: `I:/CaptchaGPT/internal/config/config.go`
- Create: `I:/CaptchaGPT/internal/server/server.go`
- Create: `I:/CaptchaGPT/internal/server/router.go`
- Create: `I:/CaptchaGPT/.env.example`
- Create: `I:/CaptchaGPT/.gitignore`
- Create: `I:/CaptchaGPT/README.md`

**Step 1: 写失败前的配置加载测试**

```go
func TestLoadConfig_UsesEnvAndDefaults(t *testing.T) {
    t.Setenv("PORT", "8080")
    t.Setenv("MODEL_NAME", "")
    cfg, err := config.Load()
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if cfg.ModelName != "google/gemma-4-31b-it" {
        t.Fatalf("expected default model, got %s", cfg.ModelName)
    }
}
```

**Step 2: 运行测试确认失败**

Run: `go test ./...`
Expected: FAIL，提示 `config.Load` 或相关包不存在

**Step 3: 写最小实现**

```go
type Config struct {
    Port             string
    ModelName        string
    NVIDIAAPIKey     string
    UserAPIKey       string
    RateLimitRPS     float64
    RateLimitBurst   int
    RequestTimeoutS  int
    MaxImageBytes    int64
    TempDir          string
    UpstreamBaseURL  string
}

func Load() (Config, error) {
    _ = godotenv.Load()
    cfg := Config{
        Port:            getEnv("PORT", "8080"),
        ModelName:       getEnv("MODEL_NAME", "google/gemma-4-31b-it"),
        NVIDIAAPIKey:    os.Getenv("NVIDIA_API_KEY"),
        UserAPIKey:      os.Getenv("USER_API_KEY"),
        RateLimitRPS:    getEnvFloat("RATE_LIMIT_RPS", 2),
        RateLimitBurst:  getEnvInt("RATE_LIMIT_BURST", 5),
        RequestTimeoutS: getEnvInt("REQUEST_TIMEOUT_SECONDS", 45),
        MaxImageBytes:   getEnvInt64("MAX_IMAGE_BYTES", 5<<20),
        TempDir:         getEnv("TEMP_IMAGE_DIR", os.TempDir()),
        UpstreamBaseURL: getEnv("UPSTREAM_BASE_URL", "https://integrate.api.nvidia.com/v1"),
    }
    return cfg, cfg.Validate()
}
```

**Step 4: 再次运行测试确认通过**

Run: `go test ./...`
Expected: PASS

**Step 5: 提交**

```bash
git add go.mod cmd/api/main.go internal/config/config.go internal/server/server.go internal/server/router.go .env.example .gitignore README.md
git commit -m "chore: bootstrap captcha api service"
```

### Task 2: 定义 OpenAI 风格请求与响应契约

**Files:**
- Create: `I:/CaptchaGPT/internal/api/types.go`
- Create: `I:/CaptchaGPT/internal/api/types_test.go`
- Modify: `I:/CaptchaGPT/README.md`

**Step 1: 写失败测试，覆盖请求字段与 JSON 反序列化**

```go
func TestDecodeCaptchaRequest_WithHints(t *testing.T) {
    body := `{
      "image_base64":"ZmFrZQ==",
      "captcha":{"length":4,"charset":"numeric","case_sensitive":false},
      "client_request_id":"req_123"
    }`
    var req api.CaptchaRequest
    if err := json.Unmarshal([]byte(body), &req); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if req.Captcha.Length != 4 {
        t.Fatalf("expected length=4")
    }
}
```

**Step 2: 运行测试确认失败**

Run: `go test ./internal/api -v`
Expected: FAIL，提示类型未定义

**Step 3: 写最小实现**

```go
type CaptchaHints struct {
    Length         int      `json:"length,omitempty"`
    Charset        string   `json:"charset,omitempty"` // numeric|alpha|alphanumeric|custom
    AllowedChars   string   `json:"allowed_chars,omitempty"`
    CaseSensitive  bool     `json:"case_sensitive,omitempty"`
    Pattern        string   `json:"pattern,omitempty"` // 如 "AANNN"
    Language       string   `json:"language,omitempty"`
    ExtraRules     []string `json:"extra_rules,omitempty"`
}

type CaptchaRequest struct {
    ImageBase64      string       `json:"image_base64"`
    Captcha          CaptchaHints `json:"captcha"`
    ClientRequestID  string       `json:"client_request_id,omitempty"`
}

type CaptchaResponse struct {
    ID      string         `json:"id"`
    Object  string         `json:"object"`
    Created int64          `json:"created"`
    Model   string         `json:"model"`
    Result  CaptchaResult  `json:"result"`
    Error   *APIError      `json:"error,omitempty"`
}
```

**Step 4: 运行测试确认通过**

Run: `go test ./internal/api -v`
Expected: PASS

**Step 5: 提交**

```bash
git add internal/api/types.go internal/api/types_test.go README.md
git commit -m "feat: define captcha api request and response schema"
```

### Task 3: 实现 Authorization 鉴权与统一错误模型

**Files:**
- Create: `I:/CaptchaGPT/internal/middleware/auth.go`
- Create: `I:/CaptchaGPT/internal/middleware/errors.go`
- Create: `I:/CaptchaGPT/internal/middleware/auth_test.go`
- Modify: `I:/CaptchaGPT/internal/server/router.go`

**Step 1: 写失败测试，覆盖缺失 Header、格式错误、Key 错误**

```go
func TestAuthMiddleware(t *testing.T) {
    cases := []struct{
        name string
        auth string
        code int
    }{
        {"missing", "", 401},
        {"bad prefix", "Token abc", 401},
        {"wrong key", "Bearer wrong", 403},
        {"ok", "Bearer test-key", 200},
    }
}
```

**Step 2: 运行测试确认失败**

Run: `go test ./internal/middleware -v`
Expected: FAIL

**Step 3: 写最小实现**

```go
func RequireAPIKey(expected string) gin.HandlerFunc {
    return func(c *gin.Context) {
        header := strings.TrimSpace(c.GetHeader("Authorization"))
        if header == "" {
            WriteError(c, http.StatusUnauthorized, "missing_authorization", "Authorization header is required")
            c.Abort()
            return
        }
        token := strings.TrimPrefix(header, "Bearer ")
        if token == header {
            WriteError(c, http.StatusUnauthorized, "invalid_authorization_format", "Use Bearer <key>")
            c.Abort()
            return
        }
        if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(token)), []byte(expected)) != 1 {
            WriteError(c, http.StatusForbidden, "invalid_api_key", "API key is invalid")
            c.Abort()
            return
        }
        c.Next()
    }
}
```

**Step 4: 运行测试确认通过**

Run: `go test ./internal/middleware -v`
Expected: PASS

**Step 5: 提交**

```bash
git add internal/middleware/auth.go internal/middleware/errors.go internal/middleware/auth_test.go internal/server/router.go
git commit -m "feat: add api key auth and error envelope"
```

### Task 4: 实现限流、中间件链与请求上下文超时

**Files:**
- Create: `I:/CaptchaGPT/internal/middleware/ratelimit.go`
- Create: `I:/CaptchaGPT/internal/middleware/requestid.go`
- Create: `I:/CaptchaGPT/internal/middleware/recovery.go`
- Create: `I:/CaptchaGPT/internal/middleware/ratelimit_test.go`
- Modify: `I:/CaptchaGPT/internal/server/router.go`
- Modify: `I:/CaptchaGPT/internal/server/server.go`

**Step 1: 写失败测试，覆盖 burst 外请求被拒绝**

```go
func TestRateLimiter_RejectsOverflow(t *testing.T) {
    limiter := middleware.NewRateLimiter(rate.Limit(1), 1)
    // 连续请求两次，第二次应返回 429
}
```

**Step 2: 运行测试确认失败**

Run: `go test ./internal/middleware -run RateLimiter -v`
Expected: FAIL

**Step 3: 写最小实现**

```go
func NewRateLimiter(r rate.Limit, burst int) gin.HandlerFunc {
    limiter := rate.NewLimiter(r, burst)
    return func(c *gin.Context) {
        if !limiter.Allow() {
            WriteError(c, http.StatusTooManyRequests, "rate_limit_exceeded", "Too many requests")
            c.Abort()
            return
        }
        c.Next()
    }
}
```

同时在服务入口加上 `http.Server` 级别读写超时，以及请求级 `context.WithTimeout`。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/middleware -run RateLimiter -v`
Expected: PASS

**Step 5: 提交**

```bash
git add internal/middleware/ratelimit.go internal/middleware/requestid.go internal/middleware/recovery.go internal/middleware/ratelimit_test.go internal/server/router.go internal/server/server.go
git commit -m "feat: add rate limit and request timeout middleware"
```

### Task 5: 实现 Base64 校验、图片落盘与临时文件清理

**Files:**
- Create: `I:/CaptchaGPT/internal/imageutil/decode.go`
- Create: `I:/CaptchaGPT/internal/imageutil/decode_test.go`
- Modify: `I:/CaptchaGPT/internal/api/types.go`

**Step 1: 写失败测试，覆盖 data URL、纯 base64、超大小、非法内容**

```go
func TestDecodeAndSaveImage(t *testing.T) {
    path, meta, err := imageutil.DecodeAndSave(ctx, tempDir, validPNGBase64, 1024*1024)
    if err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
    if meta.MIMEType != "image/png" {
        t.Fatalf("expected png, got %s", meta.MIMEType)
    }
}
```

**Step 2: 运行测试确认失败**

Run: `go test ./internal/imageutil -v`
Expected: FAIL

**Step 3: 写最小实现**

```go
func DecodeAndSave(ctx context.Context, dir, encoded string, maxBytes int64) (string, Meta, error) {
    raw, mimeType, err := normalizeAndDecode(encoded)
    if err != nil {
        return "", Meta{}, ErrInvalidImage
    }
    if int64(len(raw)) > maxBytes {
        return "", Meta{}, ErrImageTooLarge
    }
    if err := validateImage(raw); err != nil {
        return "", Meta{}, ErrUnsupportedImage
    }
    path := filepath.Join(dir, uuid.NewString()+extensionFromMIME(mimeType))
    if err := os.WriteFile(path, raw, 0o600); err != nil {
        return "", Meta{}, err
    }
    return path, Meta{MIMEType: mimeType, SizeBytes: int64(len(raw))}, nil
}
```

并确保请求完成后 `defer os.Remove(path)` 清理临时文件。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/imageutil -v`
Expected: PASS

**Step 5: 提交**

```bash
git add internal/imageutil/decode.go internal/imageutil/decode_test.go internal/api/types.go
git commit -m "feat: add base64 image decoding and temp file storage"
```

### Task 6: 设计提示词构造器，利用验证码约束提高识别率

**Files:**
- Create: `I:/CaptchaGPT/internal/prompt/prompt.go`
- Create: `I:/CaptchaGPT/internal/prompt/prompt_test.go`

**Step 1: 写失败测试，确认提示词包含长度、字符集与严格输出要求**

```go
func TestBuildCaptchaPrompt(t *testing.T) {
    prompt := prompt.Build(api.CaptchaHints{
        Length: 4,
        Charset: "numeric",
        Pattern: "NNNN",
    })
    for _, expected := range []string{
        "exactly 4 characters",
        "digits only",
        "return only the captcha text",
    } {
        if !strings.Contains(prompt, expected) {
            t.Fatalf("missing %q", expected)
        }
    }
}
```

**Step 2: 运行测试确认失败**

Run: `go test ./internal/prompt -v`
Expected: FAIL

**Step 3: 写最小实现**

```go
func Build(h api.CaptchaHints) string {
    rules := []string{
        "You are solving a captcha from an image.",
        "Return only the captcha text.",
        "Do not explain your reasoning.",
        "If uncertain, return your best guess as plain text only.",
    }
    if h.Length > 0 {
        rules = append(rules, fmt.Sprintf("The captcha contains exactly %d characters.", h.Length))
    }
    switch h.Charset {
    case "numeric":
        rules = append(rules, "The captcha contains digits only.")
    case "alpha":
        rules = append(rules, "The captcha contains letters only.")
    case "alphanumeric":
        rules = append(rules, "The captcha contains only letters and digits.")
    }
    if h.Pattern != "" {
        rules = append(rules, "Expected pattern: "+h.Pattern)
    }
    if h.AllowedChars != "" {
        rules = append(rules, "Allowed characters: "+h.AllowedChars)
    }
    return strings.Join(rules, "\n")
}
```

**Step 4: 运行测试确认通过**

Run: `go test ./internal/prompt -v`
Expected: PASS

**Step 5: 提交**

```bash
git add internal/prompt/prompt.go internal/prompt/prompt_test.go
git commit -m "feat: add captcha prompt builder with recognition hints"
```

### Task 7: 实现 NVIDIA 上游客户端，并保持 OpenAI 兼容抽象

**Files:**
- Create: `I:/CaptchaGPT/internal/upstream/client.go`
- Create: `I:/CaptchaGPT/internal/upstream/nvidia_client.go`
- Create: `I:/CaptchaGPT/internal/upstream/types.go`
- Create: `I:/CaptchaGPT/internal/upstream/nvidia_client_test.go`

**Step 1: 写失败测试，验证请求组装和响应解析**

```go
func TestNVIDIAClient_RecognizeCaptcha(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if got := r.Header.Get("Authorization"); got != "Bearer nvidia-key" {
            t.Fatalf("unexpected auth header: %s", got)
        }
        // 校验 model 和 messages 结构
        _, _ = w.Write([]byte(`{
          "id":"chatcmpl_test",
          "choices":[{"message":{"content":"7K3A"}}]
        }`))
    }))
}
```

**Step 2: 运行测试确认失败**

Run: `go test ./internal/upstream -v`
Expected: FAIL

**Step 3: 写最小实现**

```go
type VisionClient interface {
    RecognizeCaptcha(ctx context.Context, req RecognizeRequest) (RecognizeResult, error)
}

type RecognizeRequest struct {
    Model      string
    Prompt     string
    ImagePath  string
    MIMEType   string
}

func (c *NVIDIAClient) RecognizeCaptcha(ctx context.Context, req RecognizeRequest) (RecognizeResult, error) {
    imageDataURL, err := fileToDataURL(req.ImagePath, req.MIMEType)
    if err != nil {
        return RecognizeResult{}, err
    }
    body := ChatCompletionRequest{
        Model: req.Model,
        Messages: []Message{
            {
                Role: "user",
                Content: []ContentPart{
                    {Type: "text", Text: req.Prompt},
                    {Type: "image_url", ImageURL: ImageURL{URL: imageDataURL}},
                },
            },
        },
        Temperature: 0.1,
    }
    // POST /chat/completions
}
```

要求把上游请求/响应封装成 OpenAI 风格结构，而不是把 NVIDIA 特定字段散落在业务层。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/upstream -v`
Expected: PASS

**Step 5: 提交**

```bash
git add internal/upstream/client.go internal/upstream/nvidia_client.go internal/upstream/types.go internal/upstream/nvidia_client_test.go
git commit -m "feat: add nvidia vision client with openai style payloads"
```

### Task 8: 编排识别服务，处理常见 API 异常与结果清洗

**Files:**
- Create: `I:/CaptchaGPT/internal/service/captcha_service.go`
- Create: `I:/CaptchaGPT/internal/service/captcha_service_test.go`
- Create: `I:/CaptchaGPT/internal/service/errors.go`

**Step 1: 写失败测试，覆盖这些场景**

```go
func TestCaptchaService_Recognize(t *testing.T) {
    // 1. base64 非法 -> 400
    // 2. 图片超限 -> 413
    // 3. 上游超时 -> 504
    // 4. 上游 401/429/500 -> 映射为 502/503
    // 5. 模型返回空串 -> 502
    // 6. 成功时清洗换行和多余说明
}
```

**Step 2: 运行测试确认失败**

Run: `go test ./internal/service -v`
Expected: FAIL

**Step 3: 写最小实现**

```go
func (s *CaptchaService) Recognize(ctx context.Context, req api.CaptchaRequest) (api.CaptchaResponse, int, error) {
    path, meta, err := s.decoder.DecodeAndSave(ctx, s.tempDir, req.ImageBase64, s.maxImageBytes)
    if err != nil {
        return api.CaptchaResponse{}, mapDecodeError(err), err
    }
    defer os.Remove(path)

    prompt := s.promptBuilder.Build(req.Captcha)
    result, err := s.visionClient.RecognizeCaptcha(ctx, upstream.RecognizeRequest{
        Model: s.modelName,
        Prompt: prompt,
        ImagePath: path,
        MIMEType: meta.MIMEType,
    })
    if err != nil {
        return api.CaptchaResponse{}, mapUpstreamError(err), err
    }

    text := sanitizeCaptcha(result.Text)
    if text == "" {
        return api.CaptchaResponse{}, http.StatusBadGateway, ErrEmptyModelResponse
    }
    return newSuccessResponse(text, s.modelName), http.StatusOK, nil
}
```

**Step 4: 运行测试确认通过**

Run: `go test ./internal/service -v`
Expected: PASS

**Step 5: 提交**

```bash
git add internal/service/captcha_service.go internal/service/captcha_service_test.go internal/service/errors.go
git commit -m "feat: orchestrate captcha recognition service with error mapping"
```

### Task 9: 暴露 `POST /api/getCode` 路由与健康检查接口

**Files:**
- Create: `I:/CaptchaGPT/internal/handler/captcha_handler.go`
- Create: `I:/CaptchaGPT/internal/handler/captcha_handler_test.go`
- Modify: `I:/CaptchaGPT/internal/server/router.go`

**Step 1: 写失败测试，覆盖成功响应与错误响应 JSON**

```go
func TestPostGetCode(t *testing.T) {
    req := httptest.NewRequest(http.MethodPost, "/api/getCode", strings.NewReader(`{"image_base64":"ZmFrZQ=="}`))
    req.Header.Set("Authorization", "Bearer test-key")
    req.Header.Set("Content-Type", "application/json")
    // 断言 object/model/result/error/request_id 字段
}
```

**Step 2: 运行测试确认失败**

Run: `go test ./internal/handler -v`
Expected: FAIL

**Step 3: 写最小实现**

```go
func (h *CaptchaHandler) GetCode(c *gin.Context) {
    var req api.CaptchaRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        middleware.WriteError(c, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
        return
    }
    resp, status, err := h.service.Recognize(c.Request.Context(), req)
    if err != nil {
        middleware.WriteAPIErrorResponse(c, status, resp)
        return
    }
    c.JSON(http.StatusOK, resp)
}
```

同时增加 `GET /healthz` 返回基础健康状态，便于部署探针使用。

**Step 4: 运行测试确认通过**

Run: `go test ./internal/handler -v`
Expected: PASS

**Step 5: 提交**

```bash
git add internal/handler/captcha_handler.go internal/handler/captcha_handler_test.go internal/server/router.go
git commit -m "feat: expose captcha recognition endpoint and health check"
```

### Task 10: 补全部署、容器化与运维参数

**Files:**
- Create: `I:/CaptchaGPT/Dockerfile`
- Create: `I:/CaptchaGPT/.dockerignore`
- Create: `I:/CaptchaGPT/docker-compose.yml`
- Modify: `I:/CaptchaGPT/.env.example`
- Modify: `I:/CaptchaGPT/README.md`

**Step 1: 写失败前的部署检查清单**

```text
需要明确：
1. 默认监听端口
2. 环境变量名称和说明
3. 容器启动命令
4. 健康检查路径
```

**Step 2: 手动检查缺口**

Run: `rg -n "NVIDIA_API_KEY|USER_API_KEY|getCode|healthz" README.md .env.example`
Expected: 缺少完整部署说明

**Step 3: 写最小实现**

`.env.example` 至少包含：

```dotenv
PORT=8080
MODEL_NAME=google/gemma-4-31b-it
NVIDIA_API_KEY=
USER_API_KEY=
UPSTREAM_BASE_URL=https://integrate.api.nvidia.com/v1
REQUEST_TIMEOUT_SECONDS=45
RATE_LIMIT_RPS=2
RATE_LIMIT_BURST=5
MAX_IMAGE_BYTES=5242880
TEMP_IMAGE_DIR=./tmp
LOG_LEVEL=info
```

README 需要包含：
- 鉴权方式：`Authorization: Bearer <USER_API_KEY>`
- 请求示例
- 成功返回示例
- 常见错误码
- Docker 运行示例

**Step 4: 验证部署说明完整**

Run: `rg -n "Authorization|POST /api/getCode|RATE_LIMIT_RPS|Docker" README.md .env.example`
Expected: 能搜到全部关键字段

**Step 5: 提交**

```bash
git add Dockerfile .dockerignore docker-compose.yml .env.example README.md
git commit -m "docs: add deployment and environment configuration"
```

### Task 11: 端到端测试、冒烟测试与发布前检查

**Files:**
- Create: `I:/CaptchaGPT/test/e2e/get_code_test.go`
- Create: `I:/CaptchaGPT/scripts/smoke-test.ps1`
- Modify: `I:/CaptchaGPT/README.md`

**Step 1: 写失败测试，模拟完整成功链路**

```go
func TestGetCodeE2E(t *testing.T) {
    // 启动测试 server
    // mock 上游 NVIDIA 接口
    // 发送真实 JSON 请求
    // 校验 200、response.id、result.text、model
}
```

**Step 2: 运行测试确认失败**

Run: `go test ./test/e2e -v`
Expected: FAIL

**Step 3: 写最小实现**

新增：
- E2E 用例
- PowerShell 冒烟脚本，读取本地图片并转 Base64 发请求
- README 的“发布前检查”段落

**Step 4: 跑完整验证**

Run: `go test ./...`
Expected: PASS

Run: `powershell -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1`
Expected: 返回 JSON，包含识别出的验证码文本

**Step 5: 提交**

```bash
git add test/e2e/get_code_test.go scripts/smoke-test.ps1 README.md
git commit -m "test: add e2e coverage and smoke test script"
```

### Task 12: 预留未来模型迁移能力

**Files:**
- Modify: `I:/CaptchaGPT/internal/upstream/client.go`
- Modify: `I:/CaptchaGPT/internal/config/config.go`
- Modify: `I:/CaptchaGPT/README.md`

**Step 1: 写失败前的接口约束检查**

```go
func TestNewVisionClient_SupportsProviderSelection(t *testing.T) {
    cfg := config.Config{
        UpstreamProvider: "nvidia",
    }
    client, err := upstream.NewVisionClient(cfg)
    if err != nil || client == nil {
        t.Fatalf("expected provider client")
    }
}
```

**Step 2: 运行测试确认失败**

Run: `go test ./internal/upstream -run Provider -v`
Expected: FAIL

**Step 3: 写最小实现**

增加：
- `UPSTREAM_PROVIDER=nvidia`
- `NewVisionClient(cfg)` 工厂方法
- README 的“迁移到其他模型 API”说明，强调只需替换 provider 适配层

**Step 4: 运行测试确认通过**

Run: `go test ./internal/upstream -run Provider -v`
Expected: PASS

**Step 5: 提交**

```bash
git add internal/upstream/client.go internal/config/config.go README.md
git commit -m "refactor: prepare provider abstraction for future llm migration"
```

## API 设计草案

**Endpoint**

`POST /api/getCode`

**Headers**

```http
Authorization: Bearer <USER_API_KEY>
Content-Type: application/json
```

**Request**

```json
{
  "image_base64": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
  "captcha": {
    "length": 4,
    "charset": "alphanumeric",
    "allowed_chars": "ABCDEFGHJKLMNPQRSTUVWXYZ23456789",
    "case_sensitive": false,
    "pattern": "AANN",
    "language": "en",
    "extra_rules": [
      "ignore background lines",
      "focus on foreground characters only"
    ]
  },
  "client_request_id": "req_demo_001"
}
```

**Success Response**

```json
{
  "id": "cap_01hrz0x8w6k7m4q2n9",
  "object": "captcha.result",
  "created": 1775606400,
  "model": "google/gemma-4-31b-it",
  "result": {
    "text": "7K3A"
  }
}
```

**Error Response**

```json
{
  "id": "cap_01hrz0x8w6k7m4q2n9",
  "object": "error",
  "created": 1775606400,
  "model": "google/gemma-4-31b-it",
  "error": {
    "code": "invalid_image_base64",
    "message": "image_base64 must be a valid base64-encoded image",
    "type": "invalid_request_error",
    "request_id": "req_demo_001"
  }
}
```

## 异常处理清单

- `400 Bad Request`：JSON 非法、缺少 `image_base64`、提示参数非法、Base64 不可解码
- `401 Unauthorized`：缺少 `Authorization`
- `403 Forbidden`：用户 Key 错误
- `413 Payload Too Large`：图片超出 `MAX_IMAGE_BYTES`
- `415 Unsupported Media Type`：不是支持的图片格式
- `429 Too Many Requests`：触发本地限流或上游限流
- `502 Bad Gateway`：上游模型返回空内容、格式异常、非预期响应
- `503 Service Unavailable`：上游服务暂时不可用
- `504 Gateway Timeout`：请求 NVIDIA API 超时
- `500 Internal Server Error`：本地未捕获异常

## 实现建议

- 优先选 `Gin`，因为写中间件和 JSON 错误响应会更快；如果更偏极简也可以换 `Chi`
- 临时图片建议保存到配置目录，并用 UUID 文件名，权限 `0600`
- 提示词要强制“只输出验证码文本，不要解释”，并结合位数、字符集、pattern 压缩输出空间
- 上游调用尽量保持 `chat/completions` 风格消息体，后续切 OpenAI 兼容供应商时改动最小
- 日志中不要打印原始图片 Base64、用户密钥或 NVIDIA 密钥
- 后续若验证码量大，可以增加 Redis 维度的按 key 限流，而不是只做单机内存限流

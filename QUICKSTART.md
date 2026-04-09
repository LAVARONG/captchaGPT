# 使用指南

本文档包含 captchaGPT 的完整使用说明：从下载启动到 API 调用、参数说明、返回格式及多语言示例。

## 目录

- [下载与安装](#下载与安装)
- [配置](#配置)
- [启动服务](#启动服务)
- [API 接口](#api-接口)
  - [接口地址](#接口地址)
  - [请求头](#请求头)
  - [请求参数](#请求参数)
  - [charset 取值说明](#charset-取值说明)
  - [成功返回](#成功返回)
  - [错误返回](#错误返回)
  - [错误码一览](#错误码一览)
- [调用示例](#调用示例)
  - [smoke-test 脚本（推荐）](#smoke-test-脚本推荐)
  - [cURL](#curl)
  - [PowerShell](#powershell)
  - [Python](#python)
- [常见问题](#常见问题)

---

## 下载与安装

从 [Releases](https://github.com/your-repo/captchaGPT/releases) 页面下载对应系统的压缩包，解压即可使用。

**Windows 包内容：**

```text
captchaGPT-windows-amd64/
├── captchaGPT.exe
├── start.bat
├── .env.example
├── smoke-test.ps1
├── sample.png
├── README.md
└── QUICKSTART.md
```

**Linux 包内容：**

```text
captchaGPT-linux-amd64/
├── captchaGPT
├── .env.example
├── sample.png
├── README.md
└── QUICKSTART.md
```

## 配置

将 `.env.example` 复制为 `.env`，填入以下两个必填项：

```dotenv
NVIDIA_API_KEY=你的_NVIDIA_API_KEY
USER_API_KEY=你提供给调用方的服务密钥
```

其他配置项均有合理默认值，一般无需修改。完整配置项说明请参阅 [README.md](README.md#环境变量)。

> **注意：** 不要将真实 `.env` 上传到版本控制。每个使用者都应该自行创建 `.env`。

如果你想测试关闭 thinking 后的速度，可以在 `.env` 里加：

```dotenv
ENABLE_THINKING=false
```

## 启动服务

### Windows

双击 `start.bat`，或在终端执行：

```powershell
.\start.bat
```

### Linux

```bash
chmod +x ./captchaGPT
./captchaGPT
```

启动成功后，服务默认监听 `http://127.0.0.1:8080`。

默认情况下，程序启动后会自动测试一次上游大模型接口，发送简单问候语，并在日志中显示：

- 是否请求成功
- 模型回复内容
- 响应耗时（毫秒）
- 当前是否开启 thinking

如果你不想启用启动自检，可以在 `.env` 中加入：

```dotenv
STARTUP_SELF_TEST=false
```

可通过健康检查确认服务状态：

```http
GET /healthz
```

---

## API 接口

### 接口地址

```http
POST /api/getCode
```

### 请求头

| Header | 值 | 说明 |
| --- | --- | --- |
| `Authorization` | `Bearer <USER_API_KEY>` | 必填，服务密钥鉴权 |
| `Content-Type` | `application/json` | 必填 |

### 请求参数

请求体为 JSON 格式：

```json
{
  "image_base64": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
  "captcha": {
    "length": 4,
    "charset": "alphanumeric",
    "allowed_chars": "ABCDEFGHJKLMNPQRSTUVWXYZ23456789",
    "case_sensitive": false,
    "language": "en",
    "extra_rules": [
      "ignore background lines",
      "focus on foreground characters only"
    ]
  },
  "client_request_id": "req_demo_001"
}
```

**字段说明：**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `image_base64` | `string` | 是 | 验证码图片，支持纯 Base64 或 `data:image/...;base64,...` 格式 |
| `captcha` | `object` | 否 | 验证码约束条件，提供越多识别越准确 |
| `captcha.task` | `string` | 否 | 任务类型，`text` 表示普通字符验证码，`math` 表示算术验证码 |
| `captcha.length` | `int` | 否 | 验证码字符数，如 `4` |
| `captcha.charset` | `string` | 否 | 字符类型，见下方 [charset 取值说明](#charset-取值说明) |
| `captcha.allowed_chars` | `string` | 否 | 允许的字符集合，可用于排除易混淆字符（如 `0/O`、`1/I`） |
| `captcha.case_sensitive` | `bool` | 否 | 是否区分大小写，默认 `false` |
| `captcha.language` | `string` | 否 | 字符语言/脚本，如 `en` |
| `captcha.extra_rules` | `string[]` | 否 | 额外识别提示，如"忽略背景线" |
| `client_request_id` | `string` | 否 | 客户端自定义请求 ID，便于日志追踪 |

> **提示：** `captcha` 对象中的所有字段均为可选。不传时服务会使用通用提示词进行识别。传入约束条件可以显著提高识别准确率。

### task 取值说明

| 值 | 含义 | 返回内容 |
| --- | --- | --- |
| `text` | 普通字符验证码 | 直接返回识别出的字符 |
| `math` | 算术验证码 | 返回最终计算结果的数字 |

**算术验证码示例：**

图片内容如果是：

- `20 - 18 = ?`
- `九乘六等于?`

可以这样请求：

```json
{
  "image_base64": "...",
  "captcha": {
    "task": "math"
  }
}
```

预期返回：

```json
{
  "result": {
    "text": "2",
    "duration_ms": 1842
  }
}
```

### charset 取值说明

| 值 | 含义 | 示例 |
| --- | --- | --- |
| `numeric` | 纯数字 | `4831` |
| `alpha` | 纯字母 | `AbCd` |
| `alphanumeric` | 字母 + 数字混合 | `A7C2` |
| `custom` | 自定义字符范围，需配合 `allowed_chars` 使用 | `ABCDEFG123456` |

**典型用法示例：**

已知验证码为 4 位纯数字：

```json
{
  "image_base64": "...",
  "captcha": { "length": 4, "charset": "numeric" }
}
```

已知验证码字符仅来自特定集合：

```json
{
  "image_base64": "...",
  "captcha": { "charset": "custom", "allowed_chars": "ABCDEFG123456" }
}
```

不确定验证码格式，只传图片（也可以）：

```json
{
  "image_base64": "..."
}
```

### 成功返回

```json
{
  "id": "cap_17e988afe298f8d69e58ce4a",
  "object": "captcha.result",
  "created": 1775606400,
  "result": {
    "text": "7K3A",
    "duration_ms": 1842
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | 服务端生成的响应 ID |
| `object` | `string` | 固定为 `captcha.result` |
| `created` | `int64` | Unix 时间戳 |
| `result.text` | `string` | 识别出的验证码文本 |
| `result.duration_ms` | `int64` | 本次识别耗时，单位毫秒 |

### 错误返回

```json
{
  "id": "cap_17e988afe298f8d69e58ce4a",
  "object": "error",
  "created": 1775606400,
  "error": {
    "code": "invalid_image_base64",
    "message": "image_base64 must be a valid base64-encoded image",
    "type": "invalid_request_error",
    "request_id": "req_demo_001"
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `object` | `string` | 固定为 `error` |
| `error.code` | `string` | 机器可读错误码 |
| `error.message` | `string` | 错误描述 |
| `error.type` | `string` | 错误类型 |
| `error.request_id` | `string` | 请求追踪 ID |

### 错误码一览

**客户端错误（4xx）：**

| 错误码 | 说明 |
| --- | --- |
| `missing_authorization` | 缺少 Authorization 头 |
| `invalid_authorization_format` | Authorization 格式不正确 |
| `invalid_api_key` | API Key 无效 |
| `invalid_json` | 请求体 JSON 格式错误 |
| `missing_image_base64` | 缺少 image_base64 字段 |
| `invalid_image_base64` | Base64 解码失败 |
| `image_too_large` | 图片超过大小限制 |
| `unsupported_image_format` | 不支持的图片格式 |
| `rate_limit_exceeded` | 请求频率超限 |

**服务端/上游错误（5xx）：**

| 错误码 | 说明 |
| --- | --- |
| `upstream_auth_failed` | 上游 API 鉴权失败 |
| `upstream_rate_limited` | 上游 API 频率限制 |
| `upstream_timeout` | 上游 API 超时 |
| `upstream_unavailable` | 上游 API 不可用 |
| `upstream_request_failed` | 上游 API 请求失败 |
| `empty_model_response` | 模型返回空结果 |
| `internal_server_error` | 内部错误 |

---

## 调用示例

### smoke-test 脚本（推荐）

Release 包中附带了 `smoke-test.ps1` 和 `sample.png`，这是最简单的测试方式：

```powershell
powershell -ExecutionPolicy Bypass -File .\smoke-test.ps1 -ImagePath .\sample.png -ApiKey your-user-key
```

脚本会自动读取图片、转换 Base64、发送请求并输出结果。

如果服务不在本机：

```powershell
powershell -ExecutionPolicy Bypass -File .\smoke-test.ps1 -ImagePath .\sample.png -BaseUrl http://192.168.1.10:8080 -ApiKey your-user-key
```

如果是算术验证码，比如 `20-18=?` 或 “九乘六等于?”，可以直接这样测：

```powershell
powershell -ExecutionPolicy Bypass -File .\smoke-test.ps1 -ImagePath .\math-sample.png -ApiKey your-user-key -Task math
```

### cURL

```bash
curl -X POST "http://127.0.0.1:8080/api/getCode" \
  -H "Authorization: Bearer your-user-key" \
  -H "Content-Type: application/json" \
  -d '{
    "image_base64": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
    "captcha": {
      "length": 4,
      "charset": "alphanumeric"
    }
  }'
```

### PowerShell

手动将图片转为 Base64 并发送请求：

```powershell
$bytes = [System.IO.File]::ReadAllBytes((Resolve-Path ".\sample.png"))
$base64 = [System.Convert]::ToBase64String($bytes)

$body = @{
  image_base64 = $base64
  captcha = @{
    length = 4
    charset = "alphanumeric"
  }
} | ConvertTo-Json -Depth 4

Invoke-RestMethod `
  -Method Post `
  -Uri "http://127.0.0.1:8080/api/getCode" `
  -Headers @{ Authorization = "Bearer your-user-key" } `
  -ContentType "application/json" `
  -Body $body
```

### Python

```python
import base64
import requests

# 读取本地图片并转为 Base64
with open("sample.png", "rb") as f:
    image_base64 = base64.b64encode(f.read()).decode()

resp = requests.post(
    "http://127.0.0.1:8080/api/getCode",
    headers={
        "Authorization": "Bearer your-user-key",
        "Content-Type": "application/json",
    },
    json={
        "image_base64": image_base64,
        "captcha": {
            "length": 4,
            "charset": "alphanumeric",
        },
    },
    timeout=45,
)

print(resp.status_code)
print(resp.json())
```

---

## 常见问题

### 程序启动后提示缺少 Key

`.env` 文件不存在或未填写 `NVIDIA_API_KEY` / `USER_API_KEY`。请检查 `.env` 是否在可执行文件同目录。

### 为什么传 Base64 而不是直接传图片文件

API 层接收 Base64 更通用，方便各种语言和环境调用。如果觉得麻烦，使用 `smoke-test.ps1` 脚本可以直接传图片路径，脚本会自动转换。

### captcha 参数不传行不行

可以。`captcha` 对象完全可选，不传时服务使用通用提示词识别。但提供长度、字符集等约束可以明显提高准确率。

### 支持哪些图片格式

支持 PNG、JPEG、GIF、BMP、WebP。图片大小默认限制 5MB（可通过 `MAX_IMAGE_BYTES` 调整）。

### 识别不准怎么办

- 尽量提供 `captcha.length` 和 `captcha.charset`
- 如果知道具体字符范围，使用 `captcha.allowed_chars` 排除易混淆字符
- 利用 `captcha.extra_rules` 补充识别提示，如"忽略背景干扰线"

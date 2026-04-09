param(
  [string]$ImagePath = ".\sample.png",
  [string]$BaseUrl = "http://localhost:8080",
  [string]$ApiKey = "",
  [string]$Task = "text"
)

if (-not (Test-Path $ImagePath)) {
  throw "图片不存在: $ImagePath"
}

if ([string]::IsNullOrWhiteSpace($ApiKey)) {
  throw "请提供 -ApiKey"
}

$Task = $Task.Trim().ToLower()
if ($Task -notin @("text", "math")) {
  throw "Task 只支持 text 或 math"
}

$bytes = [System.IO.File]::ReadAllBytes((Resolve-Path $ImagePath))
$base64 = [System.Convert]::ToBase64String($bytes)

$body = @{
  image_base64 = $base64
  captcha = @{
    task = $Task
  }
  client_request_id = "smoke_test"
} | ConvertTo-Json -Depth 4

Invoke-RestMethod `
  -Method Post `
  -Uri "$BaseUrl/api/getCode" `
  -Headers @{ Authorization = "Bearer $ApiKey" } `
  -ContentType "application/json" `
  -Body $body

param(
  [string]$Version = "v0.1.0",
  [string]$OutputDir = ".\release"
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "未找到 go 命令，请先安装 Go 并确保 go 在 PATH 中。"
}

$releaseRoot = Join-Path $root $OutputDir
$buildRoot = Join-Path $releaseRoot "build"
$windowsDir = Join-Path $buildRoot "captchaGPT-windows-amd64"
$linuxDir = Join-Path $buildRoot "captchaGPT-linux-amd64"

New-Item -ItemType Directory -Force -Path $windowsDir | Out-Null
New-Item -ItemType Directory -Force -Path $linuxDir | Out-Null

Write-Host "[1/6] 编译 Windows amd64..."
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o (Join-Path $windowsDir "captchaGPT.exe") .\cmd\api

Write-Host "[2/6] 编译 Linux amd64..."
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o (Join-Path $linuxDir "captchaGPT") .\cmd\api

Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host "[3/6] 复制发布文件..."
$sharedFiles = @(
  ".env.example",
  "QUICKSTART.md",
  "README.md",
  "sample.png"
)

foreach ($file in $sharedFiles) {
  Copy-Item -Path (Join-Path $root $file) -Destination $windowsDir -Force
  Copy-Item -Path (Join-Path $root $file) -Destination $linuxDir -Force
}

Copy-Item -Path (Join-Path $root "start.bat") -Destination $windowsDir -Force
Copy-Item -Path (Join-Path $root "scripts\smoke-test.ps1") -Destination (Join-Path $windowsDir "smoke-test.ps1") -Force

Write-Host "[4/6] 打包压缩文件..."
New-Item -ItemType Directory -Force -Path $releaseRoot | Out-Null
$windowsZip = Join-Path $releaseRoot ("captchaGPT-windows-amd64-" + $Version + ".zip")
$linuxTar = Join-Path $releaseRoot ("captchaGPT-linux-amd64-" + $Version + ".tar.gz")

if (Test-Path $windowsZip) {
  Remove-Item $windowsZip -Force
}
Compress-Archive -Path (Join-Path $windowsDir "*") -DestinationPath $windowsZip -Force

if (Test-Path $linuxTar) {
  Remove-Item $linuxTar -Force
}
tar -czf $linuxTar -C $buildRoot "captchaGPT-linux-amd64"

Write-Host "[5/6] 生成完成"
Write-Host "Windows 包: $windowsZip"
Write-Host "Linux 包:   $linuxTar"
Write-Host "构建目录:   $buildRoot"

Write-Host "[6/6] 可上传到 GitHub Releases"

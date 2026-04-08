@echo off
setlocal
cd /d "%~dp0"

if not exist ".env" (
  echo [ERROR] .env file not found
  echo.
  echo Copy .env.example to .env and set:
  echo   NVIDIA_API_KEY
  echo   USER_API_KEY
  echo.
  pause
  exit /b 1
)

if not exist "captchaGPT.exe" (
  echo [ERROR] captchaGPT.exe not found
  echo.
  echo Required files:
  echo   captchaGPT.exe
  echo   .env
  echo.
  pause
  exit /b 1
)

echo [INFO] Starting captchaGPT.exe
echo [INFO] Working directory: %cd%
echo.
"%~dp0captchaGPT.exe"
set EXIT_CODE=%ERRORLEVEL%
echo.
echo [INFO] Process exited with code: %EXIT_CODE%
pause
exit /b %EXIT_CODE%

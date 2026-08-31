@echo off
setlocal enabledelayedexpansion

echo ============================================
echo   CyberMind One-Click Deploy (Windows)
echo ============================================
echo.

REM ---- 1. Check Go ----
where go >nul 2>nul
if errorlevel 1 (
    echo [ERROR] Go not found. Install Go 1.25+ first: https://go.dev/dl/
    pause
    exit /b 1
)
for /f "tokens=3" %%v in ('go version') do echo [OK] %%v

REM ---- 2. Prepare config ----
if not exist config.yaml (
    echo [INFO] config.yaml not found, generating from config.example.yaml ...
    copy config.example.yaml config.yaml >nul
    echo [OK] config.yaml generated. Fill in your LLM api_key in config.yaml.
) else (
    echo [OK] config.yaml exists.
)

REM ---- 3. Download Go deps ----
echo [INFO] Downloading Go dependencies ...
go mod download
if errorlevel 1 (
    echo [ERROR] go mod download failed. Try: go env -w GOPROXY=https://goproxy.cn,direct
    pause
    exit /b 1
)

REM ---- 4. Build ----
echo [INFO] Building cybermind.exe ...
go build -o cybermind.exe cmd/server/main.go
if errorlevel 1 (
    echo [ERROR] Build failed.
    pause
    exit /b 1
)
echo [OK] Build complete.

REM ---- 5. Start server in a minimized window + open browser ----
echo [INFO] Starting server at http://127.0.0.1:8080 ...
start "CyberMind" /min cmd /c "cybermind.exe -config config.yaml"
timeout /t 3 /nobreak >nul
start "" http://127.0.0.1:8080/

echo.
echo [OK] CyberMind is running. Close the minimized "CyberMind" window to stop it.
echo      Web console: http://127.0.0.1:8080/
echo.
pause

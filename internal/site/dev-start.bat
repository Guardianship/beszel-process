@echo off
chcp 65001 >nul 2>&1

echo ========================================
echo    Beszel Dev Environment
echo ========================================
echo.

:: Resolve project root path
for %%I in ("%~dp0..\..") do set "ROOT=%%~fI"

:: Ensure dev SSH key is in hub's data directory
if not exist "%ROOT%\internal\cmd\hub\beszel_data" mkdir "%ROOT%\internal\cmd\hub\beszel_data"
copy /Y "%ROOT%\dev_beszel_key" "%ROOT%\internal\cmd\hub\beszel_data\id_ed25519" >nul 2>&1

echo [1/2] Starting Hub backend (port 8090)...
start "Beszel Hub" cmd /k "cd /d %ROOT%\internal\cmd\hub && set ENV=dev && go run . serve --http 0.0.0.0:8090"

timeout /t 10 /nobreak >nul

echo [2/2] Starting Agent (port 8091)...
start "Beszel Agent" cmd /k "cd /d %ROOT% && set KEY_FILE=%ROOT%\dev_beszel_key.pub && go run ./internal/cmd/agent --listen 8091"

echo.
echo ========================================
echo  All services started!
echo    Hub      : http://localhost:8090
echo    Agent    : localhost:8091
echo ========================================
echo.
echo If you need to modify frontend code:
echo   1. Run: cd internal/site && npm run dev
echo   2. Restart Hub with -tags development
echo   3. Access via http://localhost:5173
echo.
echo Close the terminal windows to stop.
pause

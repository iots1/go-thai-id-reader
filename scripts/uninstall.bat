@echo off
chcp 65001 >nul
setlocal EnableDelayedExpansion

:: Go Thai ID API - Windows Uninstaller
:: Run as Administrator

echo ============================================
echo   Go Thai ID API - Windows Uninstaller
echo ============================================
echo.

:: Check for admin rights
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Please run this script as Administrator
    echo Right-click and select "Run as administrator"
    pause
    exit /b 1
)

set "APP_NAME=go-thai-id-api"
set "INSTALL_DIR=%ProgramFiles%\GoThaiIDAPI"
set "SERVICE_NAME=GoThaiIDAPI"

echo [WARNING] This will remove Go Thai ID API from your system.
echo.
set /p CONFIRM="Are you sure? (Y/N): "
if /i not "%CONFIRM%"=="Y" (
    echo Uninstall cancelled.
    pause
    exit /b 0
)

echo.
echo [STEP 1/4] Stopping service...
sc query %SERVICE_NAME% >nul 2>&1
if %errorlevel% equ 0 (
    sc stop %SERVICE_NAME% >nul 2>&1
    timeout /t 2 /nobreak >nul
    sc delete %SERVICE_NAME% >nul 2>&1
    echo [OK] Service removed
) else (
    echo [OK] No service found
)

echo [STEP 2/4] Killing running processes...
taskkill /F /IM %APP_NAME%.exe >nul 2>&1
echo [OK] Processes terminated

echo [STEP 3/4] Removing files...
if exist "%INSTALL_DIR%" (
    rmdir /S /Q "%INSTALL_DIR%"
    echo [OK] Installation directory removed
) else (
    echo [OK] Directory not found
)

echo [STEP 4/4] Removing startup shortcut...
set "SHORTCUT=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup\%APP_NAME%.lnk"
if exist "%SHORTCUT%" (
    del /F "%SHORTCUT%"
    echo [OK] Startup shortcut removed
) else (
    echo [OK] No shortcut found
)

:: Remove firewall rule
echo [INFO] Removing firewall rule...
netsh advfirewall firewall delete rule name="Go Thai ID API" >nul 2>&1
echo [OK] Firewall rule removed

echo.
echo ============================================
echo   Uninstallation Complete!
echo ============================================
echo.

pause

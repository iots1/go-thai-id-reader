@echo off
chcp 65001 >nul
setlocal EnableDelayedExpansion

:: Go Thai ID API - Windows Installer
:: Run as Administrator

echo ============================================
echo   Go Thai ID API - Windows Installer
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

:: Configuration
set "APP_NAME=go-thai-id-api"
set "INSTALL_DIR=%ProgramFiles%\GoThaiIDAPI"
set "SERVICE_NAME=GoThaiIDAPI"

:: Find the executable in current directory
set "EXE_FILE="
for %%f in (*.exe) do (
    if /i "%%~nf" neq "install" (
        set "EXE_FILE=%%f"
    )
)

if not defined EXE_FILE (
    echo [ERROR] Cannot find go-thai-id-api executable
    echo Please place this script in the same folder as the .exe file
    pause
    exit /b 1
)

echo [INFO] Found executable: %EXE_FILE%
echo [INFO] Install directory: %INSTALL_DIR%
echo.

:: Stop existing service if running
echo [STEP 1/4] Stopping existing service...
sc query %SERVICE_NAME% >nul 2>&1
if %errorlevel% equ 0 (
    sc stop %SERVICE_NAME% >nul 2>&1
    timeout /t 2 /nobreak >nul
    sc delete %SERVICE_NAME% >nul 2>&1
    echo [OK] Existing service removed
) else (
    echo [OK] No existing service found
)

:: Create installation directory
echo [STEP 2/4] Creating installation directory...
if not exist "%INSTALL_DIR%" (
    mkdir "%INSTALL_DIR%"
)
echo [OK] Directory created

:: Copy files
echo [STEP 3/4] Copying files...
copy /Y "%EXE_FILE%" "%INSTALL_DIR%\%APP_NAME%.exe" >nul
if %errorlevel% neq 0 (
    echo [ERROR] Failed to copy executable
    pause
    exit /b 1
)
echo [OK] Files copied

:: Create Windows Service using sc
echo [STEP 4/4] Creating Windows service...
sc create %SERVICE_NAME% binPath= "\"%INSTALL_DIR%\%APP_NAME%.exe\"" start= auto DisplayName= "Go Thai ID API"
if %errorlevel% neq 0 (
    echo [WARNING] Could not create service, will create startup shortcut instead
    goto :CreateShortcut
)

sc description %SERVICE_NAME% "Thai National ID Card Reader API Service"
sc start %SERVICE_NAME%

echo [OK] Service created and started
goto :Done

:CreateShortcut
:: Create startup shortcut as fallback
echo [INFO] Creating startup shortcut...
set "STARTUP_DIR=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"
set "SHORTCUT=%STARTUP_DIR%\%APP_NAME%.lnk"

powershell -Command "$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut('%SHORTCUT%'); $s.TargetPath = '%INSTALL_DIR%\%APP_NAME%.exe'; $s.WorkingDirectory = '%INSTALL_DIR%'; $s.Save()"

:: Start the application now
start "" "%INSTALL_DIR%\%APP_NAME%.exe"
echo [OK] Application added to startup and started

:Done
echo.
echo ============================================
echo   Installation Complete!
echo ============================================
echo.
echo API URL: http://localhost:8080/api/read
echo Install Location: %INSTALL_DIR%
echo.
echo The service will start automatically on boot.
echo.

:: Add firewall rule
echo [INFO] Adding firewall rule...
netsh advfirewall firewall delete rule name="Go Thai ID API" >nul 2>&1
netsh advfirewall firewall add rule name="Go Thai ID API" dir=in action=allow protocol=tcp localport=8080 >nul 2>&1
echo [OK] Firewall rule added
echo.

pause

@echo off
setlocal enabledelayedexpansion
set "PATH=%SystemRoot%\System32;%SystemRoot%;%SystemRoot%\System32\Wbem;%PATH%"


:: --- Auto-elevate (UAC) ---
net session >nul 2>&1
if %errorlevel%==0 goto :gotAdmin

echo Requesting Administrator privileges...
echo Set UAC = CreateObject^("Shell.Application"^) > "%temp%\pt_elevate.vbs"
echo UAC.ShellExecute "%~s0", "", "", "runas", 1 >> "%temp%\pt_elevate.vbs"
"%temp%\pt_elevate.vbs"
del "%temp%\pt_elevate.vbs" >nul 2>&1
exit /b

:gotAdmin
cd /d "%~dp0"
:: --- Elevated ---

set "PROXY_NAME=Home-Reality"
set "PROXY_HOST=play.rowstonsoftware.com"
set "PROXY_PORT=443"
set "PROXY_UUID=a750d63d-3348-4f8b-be6a-be17606aff60"
set "PROXY_FLOW=xtls-rprx-vision"
set "PROXY_SNI=dl.google.com"
set "PROXY_FP=chrome"
set "PROXY_PBK=npglY0nkJZ16gJ-AmWkt30hf1zgikjNRN9BhPUOKVnQ"
set "PROXY_SID=0123456789abcdef"
set "LOCAL_PORT=2080"
set "LOCAL_PROXY=127.0.0.1:%LOCAL_PORT%"
set "SINGBOX_JSON=%~dp0sing-box-config.json"
set "XRAY_JSON=%~dp0xray-config.json"
set "CORE_LOG=%~dp0proxy_core.log"

set "CORE="
set "CORE_TYPE="
if exist "%~dp0sing-box.exe" (
    set "CORE=%~dp0sing-box.exe"
    set "CORE_TYPE=singbox"
)
if not defined CORE if exist "%~dp0nekobox_core.exe" (
    set "CORE=%~dp0nekobox_core.exe"
    set "CORE_TYPE=singbox"
)
if not defined CORE if exist "%~dp0xray.exe" (
    set "CORE=%~dp0xray.exe"
    set "CORE_TYPE=xray"
)
if not defined CORE if exist "%~dp0nekoray_core.exe" (
    set "CORE=%~dp0nekoray_core.exe"
    set "CORE_TYPE=xray"
)

:menu
cls
echo.
echo ========================================
echo   PROXY TOGGLE - %PROXY_NAME%
echo   VLESS Reality
echo ========================================
echo.

set "PROXY_ENABLED=0"
%SystemRoot%\System32\tasklist.exe /FI "IMAGENAME eq sing-box.exe" 2>nul | %SystemRoot%\System32\findstr.exe /I "sing-box.exe" >nul && set "PROXY_ENABLED=1"
%SystemRoot%\System32\tasklist.exe /FI "IMAGENAME eq nekobox_core.exe" 2>nul | %SystemRoot%\System32\findstr.exe /I "nekobox_core.exe" >nul && set "PROXY_ENABLED=1"
%SystemRoot%\System32\tasklist.exe /FI "IMAGENAME eq nekoray_core.exe" 2>nul | %SystemRoot%\System32\findstr.exe /I "nekoray_core.exe" >nul && set "PROXY_ENABLED=1"
%SystemRoot%\System32\tasklist.exe /FI "IMAGENAME eq xray.exe" 2>nul | %SystemRoot%\System32\findstr.exe /I "xray.exe" >nul && set "PROXY_ENABLED=1"

if %PROXY_ENABLED% equ 1 (
    color 0A
    echo   Status: Connected
) else (
    color 0C
    echo   Status: Disconnected
)
color 07
echo.
echo   Host:  %PROXY_HOST%:%PROXY_PORT%
echo   Local: %LOCAL_PROXY%
if defined CORE (
    echo   Core:  %CORE%
    echo   Type:  %CORE_TYPE%
) else (
    echo   Core:  NOT FOUND
)
echo.
echo ========================================
echo.

if %PROXY_ENABLED% equ 0 (
    echo   [1]  ENABLE PROXY
    echo   [2]  Show URI / paths
    echo   [3]  Exit
    echo.
    set /p choice="Enter choice [1-3]: "
    if "!choice!"=="1" goto enable_proxy
    if "!choice!"=="2" goto show_config
    if "!choice!"=="3" goto exit_app
) else (
    echo   [1]  Already running
    echo   [2]  DISABLE PROXY
    echo   [3]  Show URI / paths
    echo   [4]  Exit
    echo.
    set /p choice="Enter choice [1-4]: "
    if "!choice!"=="1" goto menu
    if "!choice!"=="2" goto disable_proxy
    if "!choice!"=="3" goto show_config
    if "!choice!"=="4" goto exit_app
)
goto menu

:enable_proxy
cls
echo.
echo Enabling %PROXY_NAME% ...
echo.

if not defined CORE call :get_core
if not defined CORE (
    echo [!] No core binary in this folder and the automatic download failed.
    echo.
    echo     Recommended: download official sing-box for Windows and
    echo     place sing-box.exe next to this script.
    echo     https://github.com/SagerNet/sing-box/releases
    echo.
    echo     nekobox_core.exe from NekoBox also works.
    echo     Plain nekoray_core.exe often cannot run standalone.
    echo.
    pause
    goto menu
)

echo [*] Stopping old cores...
%SystemRoot%\System32\taskkill.exe /F /IM sing-box.exe >nul 2>&1
%SystemRoot%\System32\taskkill.exe /F /IM nekobox_core.exe >nul 2>&1
%SystemRoot%\System32\taskkill.exe /F /IM nekoray_core.exe >nul 2>&1
%SystemRoot%\System32\taskkill.exe /F /IM xray.exe >nul 2>&1

echo [*] Setting system proxy to %LOCAL_PROXY% ...
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v ProxyServer /d "%LOCAL_PROXY%" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v ProxyEnable /d 1 /f >nul 2>&1

echo [*] Starting core: %CORE%
echo     type: %CORE_TYPE%
echo.

del "%CORE_LOG%" >nul 2>&1

if /i "%CORE_TYPE%"=="singbox" (
    if not exist "%SINGBOX_JSON%" (
        echo [!] Missing sing-box-config.json
        pause
        goto menu
    )
    echo     "%CORE%" run -c "%SINGBOX_JSON%"
    start "proxy-core" /MIN cmd /c ""%CORE%" run -c "%SINGBOX_JSON%" > "%CORE_LOG%" 2>&1"
) else (
    if not exist "%XRAY_JSON%" (
        echo [!] Missing xray-config.json
        pause
        goto menu
    )
    echo     Trying several Xray CLI forms...
    REM Form 1: run -c
    start "proxy-core" /MIN cmd /c ""%CORE%" run -c "%XRAY_JSON%" > "%CORE_LOG%" 2>&1"
    ping -n 3 127.0.0.1 >nul
    set "RUNNING=0"
    call :check_running
    if !RUNNING! equ 1 goto :started_ok

    REM Form 2: -config
    start "proxy-core" /MIN cmd /c ""%CORE%" -config "%XRAY_JSON%" > "%CORE_LOG%" 2>&1"
    ping -n 3 127.0.0.1 >nul
    call :check_running
    if !RUNNING! equ 1 goto :started_ok

    REM Form 3: run -config
    start "proxy-core" /MIN cmd /c ""%CORE%" run -config "%XRAY_JSON%" > "%CORE_LOG%" 2>&1"
    ping -n 3 127.0.0.1 >nul
    call :check_running
    if !RUNNING! equ 1 goto :started_ok

    goto :start_failed
)

ping -n 3 127.0.0.1 >nul
set "RUNNING=0"
call :check_running
if %RUNNING% equ 1 goto :started_ok

:start_failed
echo.
echo [!] Core did not stay running.
echo.
if exist "%CORE_LOG%" (
    echo --- proxy_core.log ---
    type "%CORE_LOG%"
    echo ----------------------
)
echo.
echo nekoray_core.exe from old NekoRay builds often only works inside
echo the NekoRay GUI ^(gRPC^), not as a standalone CLI.
echo.
echo FIX: Put one of these next to this script instead:
echo   1. sing-box.exe   ^(best^)  https://github.com/SagerNet/sing-box/releases
echo   2. nekobox_core.exe from a recent NekoBox package
echo.
echo Or use NekoRay GUI: import Home-Reality.uri, Start, enable System Proxy.
echo.
pause
goto menu

:started_ok
echo.
echo [+] Core is running.
echo [+] System proxy: %LOCAL_PROXY%
echo [+] Test IP: https://www.whatismyipaddress.com/
echo.
pause
goto menu

:get_core
echo [*] sing-box not found - downloading it now ^(first run, about 35 MB^)...
echo.
%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0get-singbox.ps1"
echo.
if exist "%~dp0sing-box.exe" (
    set "CORE=%~dp0sing-box.exe"
    set "CORE_TYPE=singbox"
)
exit /b

:check_running
set "RUNNING=0"
%SystemRoot%\System32\tasklist.exe /FI "IMAGENAME eq sing-box.exe" 2>nul | %SystemRoot%\System32\findstr.exe /I "sing-box.exe" >nul && set "RUNNING=1"
%SystemRoot%\System32\tasklist.exe /FI "IMAGENAME eq nekobox_core.exe" 2>nul | %SystemRoot%\System32\findstr.exe /I "nekobox_core.exe" >nul && set "RUNNING=1"
%SystemRoot%\System32\tasklist.exe /FI "IMAGENAME eq nekoray_core.exe" 2>nul | %SystemRoot%\System32\findstr.exe /I "nekoray_core.exe" >nul && set "RUNNING=1"
%SystemRoot%\System32\tasklist.exe /FI "IMAGENAME eq xray.exe" 2>nul | %SystemRoot%\System32\findstr.exe /I "xray.exe" >nul && set "RUNNING=1"
exit /b

:disable_proxy
cls
echo.
echo Disabling proxy...
%SystemRoot%\System32\taskkill.exe /F /IM sing-box.exe >nul 2>&1
%SystemRoot%\System32\taskkill.exe /F /IM nekobox_core.exe >nul 2>&1
%SystemRoot%\System32\taskkill.exe /F /IM nekoray_core.exe >nul 2>&1
%SystemRoot%\System32\taskkill.exe /F /IM xray.exe >nul 2>&1
%SystemRoot%\System32\taskkill.exe /F /IM nekoray.exe >nul 2>&1
%SystemRoot%\System32\taskkill.exe /F /IM nekobox.exe >nul 2>&1

reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v ProxyEnable /d 0 /f >nul 2>&1
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v ProxyServer /f >nul 2>&1

echo [+] Disabled.
echo.
pause
goto menu

:show_config
cls
echo.
echo ========================================
echo   %PROXY_NAME%
echo ========================================
echo.
echo Core:         %CORE%
echo Type:         %CORE_TYPE%
echo sing-box cfg: %SINGBOX_JSON%
echo xray cfg:     %XRAY_JSON%
echo log:          %CORE_LOG%
echo.
echo URI:
echo vless://%PROXY_UUID%@%PROXY_HOST%:%PROXY_PORT%?encryption=none^&flow=%PROXY_FLOW%^&security=reality^&sni=%PROXY_SNI%^&fp=%PROXY_FP%^&pbk=%PROXY_PBK%^&sid=%PROXY_SID%^&type=tcp#%PROXY_NAME%
echo.
pause
goto menu

:exit_app
exit /b 0

@echo off
echo Installing APIGO service...

:: Get current directory
set CURRENT_DIR=%~dp0
set SERVICE_NAME=APIGO
set EXE_PATH=%CURRENT_DIR%m.exe

:: Stop and remove service (if exists)
echo Checking if service exists...
sc query %SERVICE_NAME% > nul
if %errorlevel% equ 0 (
    echo Service already exists, removing...
    %CURRENT_DIR%nssm.exe stop %SERVICE_NAME%
    %CURRENT_DIR%nssm.exe remove %SERVICE_NAME% confirm
)

:: Install service
echo Installing service...
%CURRENT_DIR%nssm.exe install %SERVICE_NAME% "%EXE_PATH%"
%CURRENT_DIR%nssm.exe set %SERVICE_NAME% DisplayName "APIGO Service"
%CURRENT_DIR%nssm.exe set %SERVICE_NAME% Description "SQL-based API Service"
%CURRENT_DIR%nssm.exe set %SERVICE_NAME% AppDirectory "%CURRENT_DIR%"
%CURRENT_DIR%nssm.exe set %SERVICE_NAME% AppExit Default Restart
%CURRENT_DIR%nssm.exe set %SERVICE_NAME% AppStdout "%CURRENT_DIR%apigo.log"
%CURRENT_DIR%nssm.exe set %SERVICE_NAME% AppStderr "%CURRENT_DIR%apigo.err"
%CURRENT_DIR%nssm.exe set %SERVICE_NAME% Start SERVICE_AUTO_START

:: Start service
echo Starting service...
%CURRENT_DIR%nssm.exe start %SERVICE_NAME%

echo APIGO service installation completed.
pause 
@echo off
echo ======================================
echo       APIGO 服务安装工具
echo       版本: v1.0
echo ======================================
echo.

:: 检查是否以管理员身份运行
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo 错误: 请以管理员身份运行此脚本！
    echo 请右键点击此脚本，选择"以管理员身份运行"。
    pause
    exit /b 1
)

:: 检查编译后的文件是否存在
if not exist build\m.exe (
    echo 错误: 未找到可执行文件 build\m.exe
    echo 请先运行 build.bat 编译项目。
    pause
    exit /b 1
)

echo 正在安装 APIGO 服务...

:: 获取当前目录的绝对路径
cd /d %~dp0
set CURRENT_DIR=%CD%

:: 使用nssm安装服务
nssm.exe install APIGO "%CURRENT_DIR%\build\m.exe"
nssm.exe set APIGO DisplayName "APIGO API服务"
nssm.exe set APIGO Description "APIGO REST API服务 - SQL模板引擎"
nssm.exe set APIGO AppDirectory "%CURRENT_DIR%\build"
nssm.exe set APIGO AppStdout "%CURRENT_DIR%\build\stdout.log"
nssm.exe set APIGO AppStderr "%CURRENT_DIR%\build\stderr.log"
nssm.exe set APIGO AppRotateFiles 1
nssm.exe set APIGO AppRotateBytes 10485760
nssm.exe set APIGO Start SERVICE_AUTO_START

:: 启动服务
echo 正在启动 APIGO 服务...
net start APIGO

:: 检查服务是否成功启动
sc query APIGO | find "RUNNING" > nul
if %errorlevel% equ 0 (
    echo.
    echo ========== 安装成功 ==========
    echo APIGO 服务已成功安装并启动。
    echo 服务名称: APIGO
    echo 可执行文件: %CURRENT_DIR%\build\m.exe
    echo 日志文件: %CURRENT_DIR%\build\stdout.log
    echo.
    echo 您可以通过以下命令管理服务:
    echo   启动: net start APIGO
    echo   停止: net stop APIGO
    echo   卸载: uninstall.bat
    echo ===============================
) else (
    echo.
    echo 警告: 服务安装成功，但启动失败。
    echo 请检查配置文件和日志，然后手动启动服务。
    echo 手动启动命令: net start APIGO
)

echo.
pause 
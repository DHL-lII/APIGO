@echo off
echo ======================================
echo       APIGO 服务卸载工具
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

:: 检查服务是否存在
sc query APIGO > nul
if %errorlevel% neq 0 (
    echo 服务 APIGO 不存在或已被卸载。
    pause
    exit /b 0
)

echo 正在停止 APIGO 服务...
net stop APIGO > nul 2>&1

:: 等待服务停止
timeout /t 2 > nul

echo 正在卸载 APIGO 服务...
nssm.exe remove APIGO confirm

echo.
echo ========== 卸载完成 ==========
echo APIGO 服务已成功卸载。
echo ==============================

echo.
pause 
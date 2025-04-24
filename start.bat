@echo off
echo ======================================
echo       APIGO 开发测试启动工具
echo       版本: v1.0
echo ======================================
echo.

:: 检查可执行文件是否存在
if not exist build\m.exe (
    echo 可执行文件不存在，尝试编译...
    call build.bat
    if %errorlevel% neq 0 (
        echo 编译失败，无法启动程序！
        pause
        exit /b %errorlevel%
    )
)

echo 正在启动 APIGO...
echo 按 Ctrl+C 停止服务
echo.

cd build
m.exe

pause 
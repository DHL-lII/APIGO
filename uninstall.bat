@echo off
echo 正在卸载APIGO服务...

:: 获取当前目录
set CURRENT_DIR=%~dp0
set SERVICE_NAME=APIGO

:: 检查服务是否存在
echo 检查服务是否存在...
sc query %SERVICE_NAME% > nul
if %errorlevel% neq 0 (
    echo 服务不存在，无需卸载。
    goto END
)

:: 停止服务
echo 正在停止服务...
%CURRENT_DIR%nssm.exe stop %SERVICE_NAME%

:: 删除服务
echo 正在删除服务...
%CURRENT_DIR%nssm.exe remove %SERVICE_NAME% confirm

echo APIGO服务已卸载。

:END
pause 
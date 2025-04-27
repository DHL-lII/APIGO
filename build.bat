@echo off
echo ======================================
echo       APIGO Build Tool
echo       Version: v1.0
echo ======================================
echo.

:: Set environment variables
set GOARCH=amd64
set GOOS=windows
set CGO_ENABLED=1

echo Downloading dependencies...
go mod tidy

:: Create build directory
if not exist build mkdir build

echo Building APIGO...
go build -o build/m.exe src/m.go

:: Check build result
if %errorlevel% neq 0 (
    echo Build failed, please check error messages!
    exit /b %errorlevel%
)

echo Copying configuration files...
copy src\m.json build\m.json

echo Copying test pages...
copy index.html build\index.html

echo.
echo ========== Build Complete ==========
echo Executable: build\m.exe
echo Config: build\m.json
echo Test Pages: build\index.html
echo.
echo You can run install.bat to install as Windows service
echo ================================

echo.
pause 
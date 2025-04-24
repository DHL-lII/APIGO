@echo off
echo APIGO Building script
echo ===============================

:: Set environment variables
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0

:: Create output directory
if not exist build mkdir build

echo Building main program m.go...
go build -o build/m.exe src/m.go src/nx.go src/wx.go
if %errorlevel% neq 0 (
    echo Main program build failed!
    goto :error
) else (
    echo Main program build success: build/m.exe
)

echo Building tray program tary.go...
go build -o build/tary.exe tary.go
if %errorlevel% neq 0 (
    echo Tray program build failed!
    goto :error
) else (
    echo Tray program build success: build/tary.exe
)

echo Copying configuration file...
copy src\m.json build\m.json
if %errorlevel% neq 0 (
    echo Configuration file copy failed!
    goto :error
) else (
    echo Configuration file copy success: build/m.json
)

echo Copying icon file...
copy favicon.ico build\favicon.ico
if %errorlevel% neq 0 (
    echo Icon file copy failed!
    goto :error
) else (
    echo Icon file copy success: build/favicon.ico
)

echo Copying script files...
copy install.bat build\install.bat
copy uninstall.bat build\uninstall.bat
copy test.html build\test.html
if %errorlevel% neq 0 (
    echo Script files copy failed!
    goto :error
) else (
    echo Script files copy success
)

echo Build and copy completed!
echo All files have been generated to build directory
goto :eof

:error
echo Error occurred during build, please check error messages.
exit /b 1 
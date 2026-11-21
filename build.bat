@echo off
rem Сборка установщика на Windows: нужен Go 1.22+ (https://go.dev/dl/).
cd /d "%~dp0"
set GOOS=windows
set GOARCH=amd64
go build -trimpath -ldflags "-H windowsgui -s -w" -o CapCut-Wiggle-Setup.exe .
if errorlevel 1 (pause & exit /b 1)
echo Готово: CapCut-Wiggle-Setup.exe
pause

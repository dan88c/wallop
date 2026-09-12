@echo off
setlocal
if defined WALLOP_ROOT (
  set ROOT=%WALLOP_ROOT%
) else (
  set ROOT=%~dp0
)
if exist "%ROOT%bin\wallop.exe" (
  "%ROOT%bin\wallop.exe" %*
  exit /b %ERRORLEVEL%
)
where go >nul 2>nul
if errorlevel 1 (
  echo wallop.exe missing. Install Go 1.22+ or run: py -3 scripts\bootstrap.py --build
  exit /b 2
)
mkdir "%ROOT%bin" 2>nul
go build -o "%ROOT%bin\wallop.exe" ./cmd/harness
if errorlevel 1 exit /b 1
cd /d "%ROOT%core-operator"
go build -o "%ROOT%bin\wallop.exe" ./cmd/harness
cd /d "%ROOT%"
"%ROOT%bin\wallop.exe" %*

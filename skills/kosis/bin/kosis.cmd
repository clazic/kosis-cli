@echo off
chcp 65001 >nul
setlocal

set "SKILL_DIR=%~dp0.."
set "BIN=%SKILL_DIR%\apps\kosis.exe"

if exist "%BIN%" (
  "%BIN%" %*
  exit /b %ERRORLEVEL%
)

echo kosis 바이너리가 없습니다. 자동 설치를 시작합니다... 1>&2
powershell -NoProfile -ExecutionPolicy Bypass -File "%SKILL_DIR%\scripts\install-binary.ps1"

if exist "%BIN%" (
  "%BIN%" %*
  exit /b %ERRORLEVEL%
) else (
  echo 오류: 바이너리 설치에 실패했습니다. 1>&2
  echo 수동 설치: https://github.com/clazic/kosis-cli/releases 1>&2
  exit /b 1
)

@echo off
echo Updating Context from GitHub...
echo This is an optional maintenance script and not part of normal offline runtime.

:: 1. Pull the latest code from GitHub
git pull

:: 2. If git pull failed, pause so you can see why
if %errorlevel% neq 0 (
    echo ❌ Git Pull Failed!
    pause
    exit /b
)

:: 3. Run tests before installing the updated binary
echo Running test suite...
go test ./...

if %errorlevel% neq 0 (
    echo ❌ Tests FAILED. Build/install aborted.
    pause
    exit /b
)

:: 4. Run your build script
call BuildToToolsDir.bat

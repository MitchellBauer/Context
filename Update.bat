@echo off
echo Updating Context...

:: 1. Pull the latest code from GitHub
git pull

:: 2. If git pull failed, pause so you can see why
if %errorlevel% neq 0 (
    echo ❌ Git Pull Failed!
    pause
    exit /b
)

:: 3. Run your build script
call BuildToToolsDir.bat
@echo off
:: Set code page to UTF-8 to support emojis
chcp 65001 >nul

echo Building Context tool...

:: 1. Build directly into the tools folder
go build -o C:\tools\context.exe context.go

if %errorlevel% neq 0 (
    echo ❌ Build FAILED. See errors above.
    pause
) else (
    :: 2. Copy config.json to tools folder, overwriting automatically (/Y)
    copy /Y config.json C:\tools\config.json >nul
    
    if %errorlevel% neq 0 (
        echo ⚠️  Build passed, but config.json was NOT found or copied.
    ) else (
        echo ✅ Build and Config Update Success! Installed to C:\tools
    )
    
    timeout /t 2 >nul
)
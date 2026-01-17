@echo off
:: Set code page to UTF-8 to support emojis
chcp 65001 >nul

echo Building Context tool...

:: Build directly into the tools folder
go build -o C:\tools\context.exe context.go

if %errorlevel% neq 0 (
    echo ❌ Build FAILED. See errors above.
    pause
) else (
    echo ✅ Build Success! context.exe updated in C:\tools
    timeout /t 2 >nul
)
@echo off
echo ========================================
echo Building Video Generation System
echo ========================================
echo.

echo [Step 1/3] Building frontend...
cd frontend
call npm run build
if %ERRORLEVEL% NEQ 0 (
    echo.
    echo [ERROR] Frontend build failed!
    pause
    exit /b 1
)

echo.
echo [Step 2/3] Copying frontend to backend...
cd ..
if exist backend\dist rmdir /s /q backend\dist
xcopy /E /I /Y frontend\dist backend\dist >nul
if %ERRORLEVEL% NEQ 0 (
    echo.
    echo [ERROR] Failed to copy frontend files!
    pause
    exit /b 1
)

echo.
echo [Step 3/3] Compiling Go backend...
cd backend
go build -o videogen.exe .
if %ERRORLEVEL% NEQ 0 (
    echo.
    echo [ERROR] Backend build failed!
    pause
    exit /b 1
)

echo.
echo ========================================
echo [SUCCESS] Build completed!
echo ========================================
echo.
echo Output: backend\videogen.exe
echo.
echo To run the application:
echo   1. Edit backend\config.json and add your API key
echo   2. Double-click backend\videogen.exe
echo   3. Browser will open automatically
echo.
pause

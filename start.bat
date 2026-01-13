@echo off
echo ========================================
echo Video Generation System Startup
echo ========================================
echo.

REM Check if videogen.exe exists (Requirement 8.1, 8.2)
if not exist "backend\videogen.exe" (
    echo [ERROR] Backend executable not found!
    echo Please run build.bat first to compile the backend.
    echo.
    pause
    exit /b 1
)

echo [OK] Backend executable found.
echo.

REM Start backend server (Requirement 8.3)
echo Starting backend server...
start "Video Generation Backend" cmd /c "cd backend && videogen.exe"

REM Wait for backend to initialize
timeout /t 2 /nobreak > nul

REM Start frontend dev server (Requirement 8.4)
echo Starting frontend dev server...
start "Video Generation Frontend" cmd /c "cd frontend && npm run dev"

REM Wait for frontend to initialize
echo Waiting for services to start...
timeout /t 5 /nobreak > nul

REM Open browser (Requirement 8.5)
echo Opening browser...
start http://localhost:5173

echo.
echo ========================================
echo System started successfully!
echo ========================================
echo.
echo Backend:  http://localhost:8080
echo Frontend: http://localhost:5173
echo.
echo Close this window to keep services running.
echo To stop services, close the Backend and Frontend windows.
echo.
pause

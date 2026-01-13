# Requirements Document

## Introduction

本项目是一个视频生成系统，包含 React 前端 UI 和 Go 后端服务。系统通过调用第三方 API (vectorengine.ai) 生成视频，使用 SQLite 数据库记录生成历史，支持视频文件的本地存储和管理。前端提供美观的视频创作界面，后端负责任务调度、状态轮询和文件管理。

## Glossary

- **Video Generation System (视频生成系统)**: 整个应用系统，包含前端和后端
- **Backend Service (后端服务)**: Go 语言编写的 HTTP API 服务
- **Frontend Application (前端应用)**: React + TypeScript 编写的 Web 界面
- **Generation Task (生成任务)**: 一次视频生成请求，包含提示词、参考图片和参数设置
- **Task Status (任务状态)**: 任务的当前状态，包括 pending、processing、completed、failed
- **Progress (进度)**: 任务完成的百分比 (0-100)
- **SQLite Database (SQLite 数据库)**: 用于持久化存储任务记录的本地数据库
- **Output Directory (输出目录)**: 后端存储生成视频文件的目录 (backend/output)
- **VectorEngine API**: 第三方视频生成 API 服务

## Requirements

### Requirement 1

**User Story:** As a user, I want to submit video generation requests through the UI, so that I can create AI-generated videos based on my prompts and reference images.

#### Acceptance Criteria

1. WHEN a user enters a prompt and clicks the send button THEN the Video Generation System SHALL create a new generation task and submit it to the backend
2. WHEN a user uploads a reference image THEN the Video Generation System SHALL include the image URL in the generation request
3. WHEN a user selects duration (10s/15s) and orientation (portrait/landscape) THEN the Video Generation System SHALL apply these settings to the generation request
4. IF a user submits an empty prompt without a reference image THEN the Video Generation System SHALL prevent the submission and display the send button as disabled

### Requirement 2

**User Story:** As a user, I want to see real-time progress of my video generation tasks, so that I know how long to wait for completion.

#### Acceptance Criteria

1. WHEN a generation task is submitted THEN the Backend Service SHALL poll the VectorEngine API every 3 seconds for status updates
2. WHEN the Backend Service receives a status update THEN the Video Generation System SHALL store the current status and progress in the SQLite Database
3. WHEN the Frontend Application requests task status THEN the Backend Service SHALL return the current status and progress percentage
4. WHILE a task is processing THEN the Frontend Application SHALL display a loading indicator with the current progress

### Requirement 3

**User Story:** As a user, I want my generated videos to be saved locally, so that I can access them later without re-downloading.

#### Acceptance Criteria

1. WHEN a video generation task completes successfully THEN the Backend Service SHALL download the video from the returned URL
2. WHEN downloading a video THEN the Backend Service SHALL save the file to the Output Directory with a unique filename
3. WHEN a video is saved THEN the Backend Service SHALL update the SQLite Database with the local file path
4. WHEN the Frontend Application requests video history THEN the Backend Service SHALL return records including local file paths

### Requirement 4

**User Story:** As a user, I want to view my generation history, so that I can see all my past creations and their details.

#### Acceptance Criteria

1. WHEN the Frontend Application loads THEN the Video Generation System SHALL fetch and display all historical generation records
2. WHEN displaying history THEN the Frontend Application SHALL show video thumbnail, prompt, duration, orientation, and creation time
3. WHEN a user clicks on a history item THEN the Frontend Application SHALL allow video playback from the local file

### Requirement 5

**User Story:** As a user, I want to delete generated videos, so that I can manage storage space and remove unwanted content.

#### Acceptance Criteria

1. WHEN a user requests to delete a video THEN the Backend Service SHALL remove the video file from the Output Directory
2. WHEN a video file is deleted THEN the Backend Service SHALL remove the corresponding record from the SQLite Database
3. WHEN deletion is successful THEN the Frontend Application SHALL remove the item from the displayed history
4. IF a video file does not exist during deletion THEN the Backend Service SHALL still remove the database record and return success

### Requirement 6

**User Story:** As a developer, I want the backend to be written in Go with proper data persistence, so that the system is performant and maintains state across restarts.

#### Acceptance Criteria

1. WHEN the Backend Service starts THEN the Backend Service SHALL initialize the SQLite Database and create required tables if they do not exist
2. WHEN the Backend Service handles API requests THEN the Backend Service SHALL use proper JSON serialization for request and response bodies
3. WHEN the Backend Service serializes task records THEN the Backend Service SHALL include all fields: id, prompt, image_url, duration, orientation, status, progress, video_url, local_path, created_at, updated_at
4. WHEN the Backend Service deserializes task records THEN the Backend Service SHALL parse JSON and populate the corresponding struct fields

### Requirement 7

**User Story:** As a developer, I want the Go backend to be compiled into a standalone executable, so that the system can run on any Windows computer without requiring Go installation.

#### Acceptance Criteria

1. WHEN building the Backend Service THEN the build process SHALL compile the Go code into a Windows executable (videogen.exe)
2. WHEN the executable is built THEN the executable SHALL be self-contained and require no external Go runtime
3. WHEN the executable runs THEN the Backend Service SHALL create the Output Directory if it does not exist
4. WHEN the executable runs THEN the Backend Service SHALL initialize the SQLite Database file in the backend directory

### Requirement 8

**User Story:** As a user, I want a one-click startup script, so that I can run the entire system on any Windows computer without manual setup.

#### Acceptance Criteria

1. WHEN the startup script executes THEN the script SHALL check if the Backend Service executable exists
2. IF the executable does not exist THEN the script SHALL display an error message and exit
3. WHEN the executable exists THEN the script SHALL start the Backend Service executable
4. WHEN the Backend Service is running THEN the script SHALL start the Frontend Application using npm
5. WHEN both services are running THEN the script SHALL open the default browser to the Frontend Application URL

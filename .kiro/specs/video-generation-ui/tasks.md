# Implementation Plan

- [x] 1. Set up backend project structure and core models





  - [x] 1.1 Create Go data models and types


    - Create `models.go` with Task struct and request/response types
    - Include JSON tags for all fields
    - _Requirements: 6.3_
  - [x] 1.2 Write property test for JSON serialization round-trip






    - **Property 6: Task JSON serialization round-trip**
    - **Validates: Requirements 6.2, 6.3, 6.4**

- [x] 2. Implement SQLite database layer






  - [x] 2.1 Create database initialization and schema

    - Create `db.go` with database connection and table creation
    - Implement InitDB function that creates tasks table if not exists
    - _Requirements: 6.1_


  - [x] 2.2 Implement CRUD operations for tasks







    - CreateTask, GetTask, GetAllTasks, UpdateTask, DeleteTask functions
    - _Requirements: 2.2, 3.3, 5.2_
  - [ ] 2.3 Write property test for task creation preserves fields



    - **Property 1: Task creation preserves all input fields**
    - **Validates: Requirements 1.1, 1.2, 1.3**
  - [ ]* 2.4 Write property test for task status updates
    - **Property 3: Task status updates are persisted and retrievable**
    - **Validates: Requirements 2.2, 2.3, 3.3, 3.4**
  - [ ]* 2.5 Write property test for task deletion
    - **Property 5: Task deletion removes database record**
    - **Validates: Requirements 5.1, 5.2**

- [x] 3. Implement VectorEngine API client





  - [x] 3.1 Create VectorEngine API client


    - Create `vectorengine.go` with CreateVideoTask and QueryTaskStatus functions
    - Implement HTTPS connection to api.vectorengine.ai
    - _Requirements: 2.1_
  - [x] 3.2 Implement video download functionality


    - Download video from URL and save to output directory
    - Generate unique filename using task ID and timestamp
    - _Requirements: 3.1, 3.2_
  - [ ]* 3.3 Write property test for unique filename generation
    - **Property 4: Video filenames are unique per task**
    - **Validates: Requirements 3.2**

- [x] 4. Implement HTTP API handlers





  - [x] 4.1 Create HTTP server and router


    - Create `main.go` with HTTP server setup
    - Configure CORS for frontend access
    - _Requirements: 6.2_

  - [x] 4.2 Implement POST /api/tasks handler

    - Validate request (prompt or image required)
    - Create task in database and start background processing
    - _Requirements: 1.1, 1.2, 1.3, 1.4_
  - [ ]* 4.3 Write property test for empty prompt rejection
    - **Property 2: Empty prompt without image is rejected**
    - **Validates: Requirements 1.4**

  - [x] 4.4 Implement GET /api/tasks handler
    - Return all tasks from database
    - _Requirements: 3.4, 4.1_
  - [x] 4.5 Implement GET /api/tasks/:id handler

    - Return single task status and details
    - _Requirements: 2.3_

  - [x] 4.6 Implement DELETE /api/tasks/:id handler
    - Delete video file from output directory
    - Remove task record from database
    - _Requirements: 5.1, 5.2, 5.4_

  - [x] 4.7 Implement static file server for videos
    - Serve files from output directory at /api/videos/:filename
    - _Requirements: 4.3_

- [x] 5. Implement background task processing






  - [x] 5.1 Create task processor with polling logic

    - Poll VectorEngine API every 3 seconds for pending tasks
    - Update task status and progress in database
    - Download video when task completes
    - _Requirements: 2.1, 2.2, 3.1, 3.3_


- [x] 6. Checkpoint - Backend tests




  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. Set up frontend API client and types





  - [x] 7.1 Create TypeScript interfaces


    - Create `types.ts` with Task and CreateTaskRequest interfaces
    - _Requirements: 6.3_
  - [x] 7.2 Create API client module


    - Create `api.ts` with functions for all backend endpoints
    - createTask, getTasks, getTask, deleteTask
    - _Requirements: 1.1, 4.1, 5.1_

- [x] 8. Update frontend UI components





  - [x] 8.1 Refactor App.tsx to use real API


    - Replace mock data with API calls
    - Implement task creation flow
    - Add polling for task status updates
    - _Requirements: 1.1, 2.4, 4.1, 4.2_
  - [x] 8.2 Add delete functionality to video cards


    - Add delete button to video card hover state
    - Implement delete confirmation and API call
    - _Requirements: 5.1, 5.3_
  - [x] 8.3 Fix TypeScript errors in template


    - Add proper type definitions
    - Install missing dependencies (lucide-react)
    - _Requirements: 4.2_


- [x] 9. Create build and startup scripts




  - [x] 9.1 Create build.bat for Go executable


    - Compile Go code to videogen.exe
    - _Requirements: 7.1, 7.2_


  - [ ] 9.2 Create start.bat for one-click startup
    - Check for videogen.exe existence
    - Start backend server
    - Start frontend dev server
    - Open browser
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

- [x] 10. Final Checkpoint - Full system test





  - Ensure all tests pass, ask the user if questions arise.

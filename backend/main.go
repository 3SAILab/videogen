package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

//go:embed dist/*
var frontendFS embed.FS

const (
	// DatabasePath is the path to the SQLite database file
	DatabasePath = "videogen.db"
)

// Global task processor instance
var taskProcessor *TaskProcessor

// Global config
var appConfig *Config

func main() {
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	appConfig = config

	// Check if API key is configured
	if config.DyuAPIKey == "" {
		log.Println("WARNING: 未配置API密钥。请编辑config.json添加dyu_api_key。")
		log.Println("应用将启动，但视频生成功能需要有效的API密钥。")
	}

	// Initialize database
	if err := InitDB(DatabasePath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer CloseDB()

	// Ensure output directory exists
	if err := EnsureOutputDirectory(); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Start background task processor
	taskProcessor = NewTaskProcessor(config.DyuAPIKey)
	taskProcessor.Start()
	defer taskProcessor.Stop()

	// Set up HTTP routes
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/tasks", corsMiddleware(handleTasks))
	mux.HandleFunc("/api/tasks/", corsMiddleware(handleTaskByID))
	mux.HandleFunc("/api/tasks-failed", corsMiddleware(handleDeleteFailedTasks))
	mux.HandleFunc("/api/tasks-by-date", corsMiddleware(handleDeleteTasksByDateRange))
	mux.HandleFunc("/api/tasks-retry-alt", corsMiddleware(handleRetryWithAlt))
	mux.HandleFunc("/api/videos/", corsMiddleware(handleVideos))
	mux.HandleFunc("/api/character-pictures/", corsMiddleware(handleCharacterPictures))

	// Character API routes (Requirements 5.1)
	mux.HandleFunc("/api/characters", corsMiddleware(handleCharacters))
	mux.HandleFunc("/api/characters/", corsMiddleware(handleCharacterByID))

	// Serve embedded frontend files
	frontendContent, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		log.Fatalf("Failed to get frontend files: %v", err)
	}
	fileServer := http.FileServer(http.FS(frontendContent))

	// Handle all other routes with frontend
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS preflight
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Try to serve static file
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists in embedded FS
		if _, err := fs.Stat(frontendContent, strings.TrimPrefix(path, "/")); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// For SPA routing, serve index.html for non-API routes
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})

	serverAddr := fmt.Sprintf(":%d", config.Port)
	url := fmt.Sprintf("http://localhost:%d", config.Port)

	log.Printf("Starting server on %s", serverAddr)
	log.Printf("Open your browser at: %s", url)

	// Open browser automatically
	go openBrowser(url)

	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// openBrowser opens the default browser to the given URL
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Run()
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

// handleTasks handles GET and POST requests to /api/tasks
func handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetAllTasks(w, r)
	case http.MethodPost:
		handleCreateTask(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleTaskByID handles GET and DELETE requests to /api/tasks/:id
func handleTaskByID(w http.ResponseWriter, r *http.Request) {
	// Extract task ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	if path == "" {
		writeError(w, http.StatusBadRequest, "Task ID required")
		return
	}

	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetTask(w, r, id)
	case http.MethodDelete:
		handleDeleteTask(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleVideos serves video files from the output directory
func handleVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract filename from URL path
	filename := strings.TrimPrefix(r.URL.Path, "/api/videos/")
	if filename == "" {
		writeError(w, http.StatusBadRequest, "Filename required")
		return
	}

	// Prevent directory traversal
	filename = filepath.Base(filename)
	filePath := filepath.Join(OutputDirectory, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "Video not found")
		return
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}

// handleCharacterPictures serves character profile pictures from the output/characters directory
func handleCharacterPictures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract filename from URL path
	filename := strings.TrimPrefix(r.URL.Path, "/api/character-pictures/")
	if filename == "" {
		writeError(w, http.StatusBadRequest, "Filename required")
		return
	}

	// Prevent directory traversal
	filename = filepath.Base(filename)
	filePath := filepath.Join("output/characters", filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "Picture not found")
		return
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}

// handleCreateTask handles POST /api/tasks
func handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate: prompt or image is required
	promptEmpty := strings.TrimSpace(req.Prompt) == ""
	imageEmpty := strings.TrimSpace(req.ImageURL) == ""
	if promptEmpty && imageEmpty {
		writeError(w, http.StatusBadRequest, "Prompt or image is required")
		return
	}

	// Convert character references in prompt (Requirements 4.3)
	// Only completed characters are used for conversion
	if req.Prompt != "" {
		characters, err := GetAllCharacters()
		if err != nil {
			log.Printf("Warning: Failed to get characters for reference conversion: %v", err)
			// Continue without conversion if we can't get characters
		} else {
			req.Prompt = ConvertCharacterReferences(req.Prompt, characters)
		}
	}

	// Set defaults if not provided
	if req.Duration == "" {
		req.Duration = Duration10s
	}
	if req.Orientation == "" {
		req.Orientation = OrientationLandscape
	}
	if req.Model == "" {
		req.Model = ModelSora2
	}

	// Validate and set count (default to 1, allowed values: 1, 2, 4)
	count := req.Count
	if count <= 0 {
		count = 1
	} else if count != 1 && count != 2 && count != 4 {
		count = 1 // Default to 1 if invalid value
	}

	// Create multiple tasks based on count
	var createdTasks []CreateTaskResponse
	for i := 0; i < count; i++ {
		task, err := CreateTask(&req)
		if err != nil {
			log.Printf("Failed to create task: %v", err)
			writeError(w, http.StatusInternalServerError, "Failed to create task")
			return
		}

		createdTasks = append(createdTasks, CreateTaskResponse{
			ID:          task.ID,
			Prompt:      task.Prompt,
			ImageURL:    task.ImageURL,
			Duration:    task.Duration,
			Orientation: task.Orientation,
			Model:       task.Model,
			Status:      task.Status,
			Progress:    task.Progress,
			CreatedAt:   task.CreatedAt,
		})
	}

	// Return response (array of created tasks)
	writeJSON(w, http.StatusCreated, createdTasks)
}

// handleGetAllTasks handles GET /api/tasks with optional pagination, status filter, or ID filter
func handleGetAllTasks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Check for IDs filter (for polling specific tasks by ID)
	idsFilter := query.Get("ids")
	if idsFilter != "" {
		idStrs := strings.Split(idsFilter, ",")
		var ids []int64
		for _, idStr := range idStrs {
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err == nil {
				ids = append(ids, id)
			}
		}
		tasks, err := GetTasksByIds(ids)
		if err != nil {
			log.Printf("Failed to get tasks by IDs: %v", err)
			writeError(w, http.StatusInternalServerError, "Failed to get tasks")
			return
		}
		if tasks == nil {
			tasks = []Task{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"tasks": tasks})
		return
	}

	// Check for status filter (for polling pending tasks only)
	statusFilter := query.Get("status")
	if statusFilter != "" {
		statuses := strings.Split(statusFilter, ",")
		tasks, err := GetTasksByStatus(statuses)
		if err != nil {
			log.Printf("Failed to get tasks by status: %v", err)
			writeError(w, http.StatusInternalServerError, "Failed to get tasks")
			return
		}
		if tasks == nil {
			tasks = []Task{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"tasks": tasks})
		return
	}

	// Check for pagination
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 20
		}
		offset := 0
		if offsetStr != "" {
			offset, _ = strconv.Atoi(offsetStr)
			if offset < 0 {
				offset = 0
			}
		}

		tasks, total, err := GetTasksPaginated(limit, offset)
		if err != nil {
			log.Printf("Failed to get paginated tasks: %v", err)
			writeError(w, http.StatusInternalServerError, "Failed to get tasks")
			return
		}
		if tasks == nil {
			tasks = []Task{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"tasks":  tasks,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
		return
	}

	// Default: return all tasks
	tasks, err := GetAllTasks()
	if err != nil {
		log.Printf("Failed to get tasks: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to get tasks")
		return
	}

	if tasks == nil {
		tasks = []Task{}
	}

	writeJSON(w, http.StatusOK, TaskListResponse{Tasks: tasks})
}

// handleGetTask handles GET /api/tasks/:id
func handleGetTask(w http.ResponseWriter, r *http.Request, id int64) {
	task, err := GetTask(id)
	if err != nil {
		log.Printf("Failed to get task: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to get task")
		return
	}

	if task == nil {
		writeError(w, http.StatusNotFound, "Task not found")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// handleDeleteTask handles DELETE /api/tasks/:id
func handleDeleteTask(w http.ResponseWriter, r *http.Request, id int64) {
	// Get task to find local file path
	task, err := GetTask(id)
	if err != nil {
		log.Printf("Failed to get task for deletion: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to delete task")
		return
	}

	// Delete video file if it exists (per requirement 5.4, continue even if file doesn't exist)
	if task != nil && task.LocalPath != "" {
		if err := DeleteVideoFile(task.LocalPath); err != nil {
			log.Printf("Warning: failed to delete video file: %v", err)
			// Continue with database deletion per requirement 5.4
		}
	}

	// Delete from database
	if err := DeleteTask(id); err != nil {
		log.Printf("Failed to delete task: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to delete task")
		return
	}

	writeJSON(w, http.StatusOK, DeleteTaskResponse{
		Success: true,
		Message: "Task deleted successfully",
	})
}

// handleDeleteFailedTasks handles DELETE /api/tasks-failed - delete all failed tasks
func handleDeleteFailedTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get all failed tasks
	failedTasks, err := GetTasksByStatus([]string{StatusFailed})
	if err != nil {
		log.Printf("Failed to get failed tasks: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to get failed tasks")
		return
	}

	deletedCount := 0
	for _, task := range failedTasks {
		// Delete video file if exists
		if task.LocalPath != "" {
			DeleteVideoFile(task.LocalPath)
		}
		// Delete from database
		if err := DeleteTask(task.ID); err != nil {
			log.Printf("Failed to delete task %d: %v", task.ID, err)
			continue
		}
		deletedCount++
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"deleted": deletedCount,
		"message": fmt.Sprintf("Deleted %d failed tasks", deletedCount),
	})
}

// handleRetryWithAlt handles POST /api/tasks-retry-alt - retry failed sora-2 tasks with sora-2-alt
func handleRetryWithAlt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	count, err := ResetFailedTasks()
	if err != nil {
		log.Printf("Failed to retry tasks with alt: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to retry tasks")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"updated": count,
		"message": fmt.Sprintf("已将 %d 个失败/进行中的任务重置为待处理", count),
	})
}

// handleDeleteTasksByDateRange handles DELETE /api/tasks-by-date - delete tasks within date range
func handleDeleteTasksByDateRange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get date range from query params
	query := r.URL.Query()
	startDate := query.Get("start")
	endDate := query.Get("end")

	if startDate == "" || endDate == "" {
		writeError(w, http.StatusBadRequest, "start and end date are required (format: YYYY-MM-DD)")
		return
	}

	// Get tasks in date range
	tasks, err := GetTasksByDateRange(startDate, endDate)
	if err != nil {
		log.Printf("Failed to get tasks by date range: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to get tasks")
		return
	}

	deletedCount := 0
	for _, task := range tasks {
		// Delete video file if exists
		if task.LocalPath != "" {
			DeleteVideoFile(task.LocalPath)
		}
		// Delete from database
		if err := DeleteTask(task.ID); err != nil {
			log.Printf("Failed to delete task %d: %v", task.ID, err)
			continue
		}
		deletedCount++
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"deleted": deletedCount,
		"message": fmt.Sprintf("Deleted %d tasks from %s to %s", deletedCount, startDate, endDate),
	})
}

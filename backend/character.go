package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ConvertCharacterReferences converts custom character names in a prompt to @{api_character_id} format
// For each character, if the custom_name appears in the prompt, it is replaced with @{api_character_id}
// Only completed characters are used for conversion
// Returns the converted prompt string
// **Feature: character-creation, Property 5: Custom name to API character ID conversion**
// **Validates: Requirements 4.3**
func ConvertCharacterReferences(prompt string, characters []Character) string {
	result := prompt
	for _, char := range characters {
		// Only use completed characters for reference conversion
		if char.CustomName != "" && char.ApiCharacterID != "" && char.Status == StatusCompleted {
			// Replace custom name with @{api_character_id} format
			result = strings.ReplaceAll(result, char.CustomName, "@{"+char.ApiCharacterID+"}")
		}
	}
	return result
}

// ValidateCustomName validates that the custom name is between 1 and 10 characters
// Returns nil if valid, error otherwise
func ValidateCustomName(name string) error {
	length := len([]rune(name)) // Use rune count for proper Unicode support
	if length < 1 || length > 10 {
		return fmt.Errorf("custom name must be 1-10 characters, got %d", length)
	}
	return nil
}

// ValidateDescription validates that the description is between 1 and 500 characters
// Returns nil if valid, error otherwise
func ValidateDescription(description string) error {
	length := len([]rune(description)) // Use rune count for proper Unicode support
	if length < 1 || length > 500 {
		return fmt.Errorf("description must be 1-500 characters, got %d", length)
	}
	return nil
}

// ValidateTimestamps validates the timestamp range string
// Format: "start,end" where end-start is between 1 and 3 seconds, and start >= 0
// Returns nil if valid, error otherwise
func ValidateTimestamps(timestamps string) error {
	parts := strings.Split(timestamps, ",")
	if len(parts) != 2 {
		return fmt.Errorf("timestamps must be in format 'start,end', got '%s'", timestamps)
	}

	start, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return fmt.Errorf("invalid start timestamp: %s", parts[0])
	}

	end, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return fmt.Errorf("invalid end timestamp: %s", parts[1])
	}

	if start < 0 {
		return fmt.Errorf("start timestamp must be >= 0, got %v", start)
	}

	diff := end - start
	if diff < 1 || diff > 3 {
		return fmt.Errorf("timestamp range must be 1-3 seconds, got %v", diff)
	}

	return nil
}

// ValidateSourceType validates that the source type is either 'task' or 'url'
// Returns nil if valid, error otherwise
func ValidateSourceType(sourceType string) error {
	if sourceType != "task" && sourceType != "url" {
		return fmt.Errorf("source type must be 'task' or 'url', got '%s'", sourceType)
	}
	return nil
}

// ValidateSourceValue validates the source value based on source type
// For 'task': must be non-empty
// For 'url': must be non-empty and valid URL format
// Returns nil if valid, error otherwise
func ValidateSourceValue(sourceType, sourceValue string) error {
	if sourceValue == "" {
		return fmt.Errorf("source value cannot be empty")
	}

	if sourceType == "url" {
		// Validate URL format
		parsedURL, err := url.Parse(sourceValue)
		if err != nil {
			return fmt.Errorf("invalid URL format: %v", err)
		}
		if parsedURL.Scheme == "" || parsedURL.Host == "" {
			return fmt.Errorf("invalid URL: must include scheme and host")
		}
	}

	return nil
}

// GetTaskByTaskID retrieves a task by its VectorEngine task_id
func GetTaskByTaskID(taskID string) (*Task, error) {
	task := &Task{}
	var imageURL, videoURL, localPath, dbTaskID sql.NullString

	err := DB.QueryRow(`
		SELECT id, task_id, prompt, image_url, duration, orientation, status, progress, video_url, local_path, created_at, updated_at
		FROM tasks WHERE task_id = ?`, taskID).Scan(
		&task.ID, &dbTaskID, &task.Prompt, &imageURL, &task.Duration, &task.Orientation,
		&task.Status, &task.Progress, &videoURL, &localPath, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get task by task_id: %w", err)
	}

	task.TaskID = dbTaskID.String
	task.ImageURL = imageURL.String
	task.VideoURL = videoURL.String
	task.LocalPath = localPath.String

	return task, nil
}

// handleCreateCharacter handles POST /api/characters
// Validates request fields, calls Sora2 Character Training API,
// and stores character in database with status='pending'
// Requirements: 1.1, 1.5, 2.1, 3.1
func handleCreateCharacter(w http.ResponseWriter, r *http.Request) {
	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var req CreateCharacterRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate custom name (Requirements 1.2)
	if err := ValidateCustomName(req.CustomName); err != nil {
		writeError(w, http.StatusBadRequest, "Custom name must be 1-10 characters")
		return
	}

	// Validate description (Requirements 1.3)
	if err := ValidateDescription(req.Description); err != nil {
		writeError(w, http.StatusBadRequest, "Description must be 1-500 characters")
		return
	}

	// Validate timestamps (Requirements 1.4)
	if err := ValidateTimestamps(req.Timestamps); err != nil {
		writeError(w, http.StatusBadRequest, "Timestamp range must be 1-3 seconds")
		return
	}

	// Validate source type (Requirements 2.1)
	if err := ValidateSourceType(req.SourceType); err != nil {
		writeError(w, http.StatusBadRequest, "Source type must be 'task' or 'url'")
		return
	}

	// Validate source value (Requirements 2.1)
	if err := ValidateSourceValue(req.SourceType, req.SourceValue); err != nil {
		if req.SourceType == "url" {
			writeError(w, http.StatusBadRequest, "Invalid video URL")
		} else {
			writeError(w, http.StatusBadRequest, "Source value cannot be empty")
		}
		return
	}

	// For task source type, verify task exists and is completed (Requirements 1.5)
	if req.SourceType == "task" {
		task, err := GetTaskByTaskID(req.SourceValue)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to verify task")
			return
		}
		if task == nil {
			writeError(w, http.StatusNotFound, "Task not found")
			return
		}
		if task.Status != StatusCompleted {
			writeError(w, http.StatusBadRequest, "Task must be completed to create character")
			return
		}
	}

	// Call Sora2 Character Training API (Requirements 1.5, 2.1)
	client := NewVectorEngineClient(appConfig.DyuAPIKey)
	sora2Resp, err := client.CreateCharacterSora2(req.SourceType, req.SourceValue, req.Timestamps)
	if err != nil {
		log.Printf("[Character] API错误: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "源视频不存在") {
			writeError(w, http.StatusBadRequest, "源视频不存在，请检查任务ID或URL是否正确")
		} else {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("创建角色失败: %v", err))
		}
		return
	}

	// Store character in database with status='pending' (Requirements 3.1)
	char := &Character{
		ApiCharacterID: sora2Resp.ID,
		CustomName:     req.CustomName,
		Description:    req.Description,
		SourceType:     req.SourceType,
		SourceValue:    req.SourceValue,
		Timestamps:     req.Timestamps,
		Status:         StatusPending,
		Progress:       0,
	}

	savedChar, err := CreateCharacter(char)
	if err != nil {
		log.Printf("[Character] 保存失败: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to save character")
		return
	}

	log.Printf("[Character] 创建成功: %s (ID: %d)", savedChar.CustomName, savedChar.ID)
	writeJSON(w, http.StatusCreated, savedChar)
}

// handleGetAllCharacters handles GET /api/characters
// Returns all characters from database with new fields (Requirements 5.1, 5.2)
func handleGetAllCharacters(w http.ResponseWriter, r *http.Request) {
	characters, err := GetAllCharacters()
	if err != nil {
		log.Printf("Failed to get characters: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to get characters")
		return
	}

	if characters == nil {
		characters = []Character{}
	}

	writeJSON(w, http.StatusOK, CharacterListResponse{Characters: characters})
}

// CharacterStatusResponse represents the response for character status query
type CharacterStatusResponse struct {
	ID             int64  `json:"id"`
	ApiCharacterID string `json:"api_character_id,omitempty"`
	Username       string `json:"username,omitempty"`   // 用于引用角色 @username
	AvatarURL      string `json:"avatar_url,omitempty"` // 角色头像URL
	Status         string `json:"status"`
	Progress       int    `json:"progress"`
	FailReason     string `json:"fail_reason,omitempty"`
}

// handleGetCharacterStatus handles GET /api/characters/:id/status
// Queries Sora2 API for current status, updates local database, and returns status
// Requirements: 3.2, 3.3, 3.4
func handleGetCharacterStatus(w http.ResponseWriter, r *http.Request, id int64) {
	// Get character from database
	char, err := GetCharacter(id)
	if err != nil {
		log.Printf("Failed to get character: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to get character")
		return
	}
	if char == nil {
		writeError(w, http.StatusNotFound, "Character not found")
		return
	}

	// If character is already completed or failed, return current status without querying API
	if char.Status == StatusCompleted || char.Status == StatusFailed {
		writeJSON(w, http.StatusOK, CharacterStatusResponse{
			ID:             char.ID,
			ApiCharacterID: char.ApiCharacterID,
			Status:         char.Status,
			Progress:       char.Progress,
			FailReason:     char.FailReason,
		})
		return
	}

	// Query Sora2 API for current status (Requirements 3.2)
	if char.ApiCharacterID == "" {
		writeError(w, http.StatusBadRequest, "Character has no API ID")
		return
	}

	client := NewVectorEngineClient(appConfig.DyuAPIKey)
	sora2Resp, err := client.QueryCharacterStatus(char.ApiCharacterID)
	if err != nil {
		log.Printf("[Character] 查询状态失败: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query character status: %v", err))
		return
	}

	// Map Sora2 status to our status
	newStatus := char.Status
	newProgress := sora2Resp.Progress
	newFailReason := sora2Resp.FailReason
	newUsername := sora2Resp.Username
	newAvatarURL := sora2Resp.AvatarURL

	switch strings.ToLower(sora2Resp.Status) {
	case "pending", "queued":
		newStatus = StatusPending
	case "processing", "in_progress":
		newStatus = StatusProcessing
	case "completed", "success":
		newStatus = StatusCompleted
		newProgress = 100
		log.Printf("[Character] 训练完成: %s (@%s)", char.CustomName, newUsername)
	case "failed", "failure", "error":
		newStatus = StatusFailed
		log.Printf("[Character] 训练失败: %s - %s", char.CustomName, newFailReason)
	}

	// Update local database with new status/progress (Requirements 3.3, 3.4)
	if newStatus != char.Status || newProgress != char.Progress || newFailReason != char.FailReason || newUsername != char.Username || newAvatarURL != char.AvatarURL {
		err = UpdateCharacterStatus(char.ID, newStatus, newProgress, char.ApiCharacterID, newUsername, newAvatarURL, newFailReason)
		if err != nil {
			log.Printf("[Character] 更新状态失败: %v", err)
			// Continue to return the status even if update fails
		}
	}

	// Return current status to frontend
	writeJSON(w, http.StatusOK, CharacterStatusResponse{
		ID:             char.ID,
		ApiCharacterID: char.ApiCharacterID,
		Username:       newUsername,
		AvatarURL:      newAvatarURL,
		Status:         newStatus,
		Progress:       newProgress,
		FailReason:     newFailReason,
	})
}

// handleDeleteCharacter handles DELETE /api/characters/:id
// Removes character from database (Requirements 5.3)
// Note: No longer needs to clean up character pictures (removed in new schema)
func handleDeleteCharacter(w http.ResponseWriter, r *http.Request, id int64) {
	if err := DeleteCharacter(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "Character not found")
			return
		}
		log.Printf("Failed to delete character: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to delete character")
		return
	}

	writeJSON(w, http.StatusOK, DeleteCharacterResponse{
		Success: true,
		Message: "Character deleted successfully",
	})
}

// handleCharacters handles GET and POST requests to /api/characters
func handleCharacters(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetAllCharacters(w, r)
	case http.MethodPost:
		handleCreateCharacter(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleCharacterByID handles requests to /api/characters/:id and /api/characters/:id/status
func handleCharacterByID(w http.ResponseWriter, r *http.Request) {
	// Extract path after /api/characters/
	path := strings.TrimPrefix(r.URL.Path, "/api/characters/")
	if path == "" {
		writeError(w, http.StatusBadRequest, "Character ID required")
		return
	}

	// Check if this is a status request: /api/characters/:id/status
	parts := strings.Split(path, "/")
	idStr := parts[0]
	isStatusRequest := len(parts) > 1 && parts[1] == "status"

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid character ID")
		return
	}

	if isStatusRequest {
		// Handle GET /api/characters/:id/status
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		handleGetCharacterStatus(w, r, id)
		return
	}

	// Handle DELETE /api/characters/:id
	switch r.Method {
	case http.MethodDelete:
		handleDeleteCharacter(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

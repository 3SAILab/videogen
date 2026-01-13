package main

import (
	"time"
)

// Task represents a video generation task stored in the database
type Task struct {
	ID          int64     `json:"id"`
	TaskID      string    `json:"task_id"`
	Prompt      string    `json:"prompt"`
	ImageURL    string    `json:"image_url,omitempty"`
	ImageURL2   string    `json:"image_url2,omitempty"` // Second image for Veo3
	Duration    string    `json:"duration"`
	Orientation string    `json:"orientation"`
	Model       string    `json:"model"`
	Status      string    `json:"status"`
	Progress    int       `json:"progress"`
	VideoURL    string    `json:"video_url,omitempty"`
	LocalPath   string    `json:"local_path,omitempty"`
	FailReason  string    `json:"fail_reason,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateTaskRequest represents the request body for creating a new task
type CreateTaskRequest struct {
	Prompt      string `json:"prompt"`
	ImageURL    string `json:"image_url,omitempty"`
	ImageURL2   string `json:"image_url2,omitempty"` // Second image for Veo3 (last frame)
	Duration    string `json:"duration"`
	Orientation string `json:"orientation"`
	Model       string `json:"model"`
	Count       int    `json:"count,omitempty"` // Number of videos to generate: 1, 2, or 4
}

// CreateTaskResponse represents the response after creating a task
type CreateTaskResponse struct {
	ID          int64     `json:"id"`
	Prompt      string    `json:"prompt"`
	ImageURL    string    `json:"image_url,omitempty"`
	Duration    string    `json:"duration"`
	Orientation string    `json:"orientation"`
	Model       string    `json:"model"`
	Status      string    `json:"status"`
	Progress    int       `json:"progress"`
	CreatedAt   time.Time `json:"created_at"`
}

// TaskListResponse represents the response for listing all tasks
type TaskListResponse struct {
	Tasks []Task `json:"tasks"`
}

// DeleteTaskResponse represents the response after deleting a task
type DeleteTaskResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// VectorEngineCreateResponse represents the response from VectorEngine API when creating a task
type VectorEngineCreateResponse struct {
	ID string `json:"id"`
}

// VectorEngineError represents an error from VectorEngine API
type VectorEngineError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

// VectorEngineQueryResponse represents the response from VectorEngine API when querying task status
// Supports both formats:
// - "限时特价" token_group: video_url in data.video_url
// - "逆向" token_group: video_url in data.detail.url
type VectorEngineQueryResponse struct {
	Status     string                 `json:"status"`
	Progress   int                    `json:"progress"`
	VideoURL   string                 `json:"video_url,omitempty"`
	ID         string                 `json:"id,omitempty"`
	Error      *VectorEngineError     `json:"error,omitempty"`
	Data       *VectorEngineQueryData `json:"data,omitempty"`
	TokenGroup string                 `json:"token_group,omitempty"`
	FailReason string                 `json:"fail_reason,omitempty"`
}

// VectorEngineQueryData represents the nested data object in API response
type VectorEngineQueryData struct {
	Status   string                   `json:"status"`
	Progress int                      `json:"progress"`
	VideoURL string                   `json:"video_url,omitempty"`
	Detail   *VectorEngineQueryDetail `json:"detail,omitempty"`
}

// VectorEngineQueryDetail represents the detail object (used in "逆向" format)
type VectorEngineQueryDetail struct {
	URL    string `json:"url,omitempty"`
	Status string `json:"status,omitempty"`
}

// Task status constants
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// Duration constants
const (
	Duration10s = "10s"
	Duration15s = "15s"
)

// Orientation constants
const (
	OrientationPortrait  = "portrait"
	OrientationLandscape = "landscape"
)

// Count constants
const (
	Count1 = 1
	Count2 = 2
	Count4 = 4
)

// Model constants
const (
	ModelSora2 = "sora-2"
)

// Character represents a character stored in the database
type Character struct {
	ID             int64     `json:"id"`
	ApiCharacterID string    `json:"api_character_id,omitempty"` // char_xxx 格式的 ID
	Username       string    `json:"username,omitempty"`         // 用于引用角色 @username
	AvatarURL      string    `json:"avatar_url,omitempty"`       // 角色头像URL
	CustomName     string    `json:"custom_name"`
	Description    string    `json:"description,omitempty"`
	SourceType     string    `json:"source_type"`  // "task" or "url"
	SourceValue    string    `json:"source_value"` // task_id or video URL
	Timestamps     string    `json:"timestamps"`
	Status         string    `json:"status"` // pending, processing, completed, failed
	Progress       int       `json:"progress"`
	FailReason     string    `json:"fail_reason,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateCharacterRequest represents the request body for creating a character
type CreateCharacterRequest struct {
	CustomName  string `json:"custom_name"`
	Description string `json:"description"`
	SourceType  string `json:"source_type"`  // "task" or "url"
	SourceValue string `json:"source_value"` // task_id or video URL
	Timestamps  string `json:"timestamps"`
}

// Sora2CharacterRequest represents the request body for Sora2 Character Training API
type Sora2CharacterRequest struct {
	Character  string `json:"character,omitempty"` // task_id when source_type is "task"
	URL        string `json:"url,omitempty"`       // video URL when source_type is "url"
	Prompt     string `json:"prompt"`
	Model      string `json:"model"` // always "character-training"
	Timestamps string `json:"timestamps"`
}

// Sora2CharacterResponse represents the response from Sora2 Character Training API
type Sora2CharacterResponse struct {
	CreatedAt  int64  `json:"created_at"`
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	Model      string `json:"model"`
	Object     string `json:"object"`
	Progress   int    `json:"progress"`
	Size       string `json:"size"`
	Status     string `json:"status"`
	CameoID    string `json:"cameo_id,omitempty"`
	Username   string `json:"username,omitempty"` // 用于引用角色 @username
	VideoURL   string `json:"video_url,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"` // 角色头像URL
	FailReason string `json:"fail_reason,omitempty"`
}

// CharacterListResponse represents the response for listing all characters
type CharacterListResponse struct {
	Characters []Character `json:"characters"`
}

// DeleteCharacterResponse represents the response after deleting a character
type DeleteCharacterResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

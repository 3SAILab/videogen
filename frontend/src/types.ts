/**
 * TypeScript interfaces for the Video Generation System
 * These interfaces match the Go backend models defined in backend/models.go
 */

// Task status constants
export type TaskStatus = 'pending' | 'processing' | 'completed' | 'failed';

// Duration options
export type Duration = '10s' | '15s';

// Orientation options
export type Orientation = 'portrait' | 'landscape';

// Model options (UI display names)
export type Model = 'sora-2';

/**
 * Task represents a video generation task
 * Matches the Go Task struct in backend/models.go
 */
export interface Task {
  id: number;
  task_id: string;
  prompt: string;
  image_url?: string;
  duration: Duration;
  orientation: Orientation;
  model: Model;
  status: TaskStatus;
  progress: number;
  video_url?: string;
  local_path?: string;
  fail_reason?: string;
  created_at: string;
  updated_at: string;
}

// Count options for number of videos to generate
export type Count = 1 | 2 | 4;

/**
 * Request body for creating a new video generation task
 * Matches the Go CreateTaskRequest struct
 */
export interface CreateTaskRequest {
  prompt: string;
  image_url?: string;
  duration: Duration;
  orientation: Orientation;
  model: Model;
  count?: Count;
}

/**
 * Response after creating a task
 * Matches the Go CreateTaskResponse struct
 */
export interface CreateTaskResponse {
  id: number;
  prompt: string;
  image_url?: string;
  duration: Duration;
  orientation: Orientation;
  model: Model;
  status: TaskStatus;
  progress: number;
  created_at: string;
}

/**
 * Response for listing all tasks
 * Matches the Go TaskListResponse struct
 */
export interface TaskListResponse {
  tasks: Task[];
}

/**
 * Response after deleting a task
 * Matches the Go DeleteTaskResponse struct
 */
export interface DeleteTaskResponse {
  success: boolean;
  message: string;
}

/**
 * Error response from the API
 * Matches the Go ErrorResponse struct
 */
export interface ErrorResponse {
  error: string;
}

// ============================================
// Character Types (for character-creation feature)
// ============================================

// Character source type options
export type CharacterSourceType = 'task' | 'url';

// Character training status options
export type CharacterStatus = 'pending' | 'processing' | 'completed' | 'failed';

/**
 * Character represents a character created from a video
 * Matches the Go Character struct in backend/models.go
 * 
 * Requirements: 6.3
 */
export interface Character {
  id: number;
  api_character_id?: string;
  username?: string; // 用于引用角色 @username
  avatar_url?: string; // 角色头像URL
  custom_name: string;
  description?: string;
  source_type: CharacterSourceType;
  source_value: string;
  timestamps: string;
  status: CharacterStatus;
  progress: number;
  fail_reason?: string;
  created_at: string;
}

/**
 * Request body for creating a new character
 * Matches the Go CreateCharacterRequest struct
 * 
 * Requirements: 1.5, 2.1
 */
export interface CreateCharacterRequest {
  custom_name: string;
  description: string;
  source_type: CharacterSourceType;
  source_value: string;
  timestamps: string;
}

/**
 * Response for listing all characters
 */
export interface CharacterListResponse {
  characters: Character[];
}

/**
 * Response after deleting a character
 */
export interface DeleteCharacterResponse {
  success: boolean;
  message: string;
}

/**
 * Response for character training status query
 * 
 * Requirements: 3.2
 */
export interface CharacterStatusResponse {
  id: number;
  api_character_id?: string;
  username?: string; // 用于引用角色 @username
  avatar_url?: string; // 角色头像URL
  status: CharacterStatus;
  progress: number;
  fail_reason?: string;
}

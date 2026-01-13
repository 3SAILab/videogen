/**
 * API client for the Video Generation System backend
 * Provides functions for all backend endpoints
 */

import type {
  Task,
  CreateTaskRequest,
  CreateTaskResponse,
  DeleteTaskResponse,
  ErrorResponse,
  Character,
  CreateCharacterRequest,
  CharacterListResponse,
  DeleteCharacterResponse,
  CharacterStatusResponse,
} from './types';

// Backend API base URL - use relative path since frontend is served by the same server
const API_BASE_URL = '/api';

/**
 * Custom error class for API errors
 */
export class ApiError extends Error {
  status: number;
  
  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

/**
 * Helper function to handle API responses
 */
async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let errorMessage = 'An error occurred';
    try {
      const errorData: ErrorResponse = await response.json();
      errorMessage = errorData.error || errorMessage;
    } catch {
      // If we can't parse the error response, use status text
      errorMessage = response.statusText || errorMessage;
    }
    throw new ApiError(response.status, errorMessage);
  }
  return response.json();
}

/**
 * Create a new video generation task
 * POST /api/tasks
 * 
 * @param request - The task creation request containing prompt, image_url, duration, and orientation
 * @returns The created task response
 * @throws ApiError if the request fails
 * 
 * Requirements: 1.1 - Create new generation task and submit to backend
 */
export async function createTask(request: CreateTaskRequest): Promise<CreateTaskResponse> {
  const response = await fetch(`${API_BASE_URL}/tasks`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });
  return handleResponse<CreateTaskResponse>(response);
}

/**
 * Get video generation tasks with optional pagination
 * GET /api/tasks
 * 
 * @param limit - Optional limit for pagination
 * @param offset - Optional offset for pagination
 * @returns List of tasks and pagination info
 * @throws ApiError if the request fails
 * 
 * Requirements: 4.1 - Fetch and display all historical generation records
 */
export async function getTasks(limit?: number, offset?: number): Promise<{ tasks: Task[]; total?: number }> {
  let url = `${API_BASE_URL}/tasks`;
  const params = new URLSearchParams();
  if (limit !== undefined) params.set('limit', limit.toString());
  if (offset !== undefined) params.set('offset', offset.toString());
  if (params.toString()) url += `?${params.toString()}`;
  
  const response = await fetch(url, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });
  const data = await handleResponse<{ tasks: Task[]; total?: number }>(response);
  return data;
}

/**
 * Get tasks by status (for polling pending/processing tasks only)
 * GET /api/tasks?status=pending,processing
 * 
 * @param statuses - Array of statuses to filter by
 * @returns List of tasks with matching statuses
 * @throws ApiError if the request fails
 */
export async function getTasksByStatus(statuses: string[]): Promise<Task[]> {
  const url = `${API_BASE_URL}/tasks?status=${statuses.join(',')}`;
  const response = await fetch(url, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });
  const data = await handleResponse<{ tasks: Task[] }>(response);
  return data.tasks;
}

/**
 * Get tasks by IDs (for polling specific tasks)
 * GET /api/tasks?ids=1,2,3
 * 
 * @param ids - Array of task IDs to fetch
 * @returns List of tasks with matching IDs
 * @throws ApiError if the request fails
 */
export async function getTasksByIds(ids: number[]): Promise<Task[]> {
  if (ids.length === 0) return [];
  const url = `${API_BASE_URL}/tasks?ids=${ids.join(',')}`;
  const response = await fetch(url, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });
  const data = await handleResponse<{ tasks: Task[] }>(response);
  return data.tasks;
}

/**
 * Get a single video generation task by ID
 * GET /api/tasks/:id
 * 
 * @param id - The task ID
 * @returns The task details
 * @throws ApiError if the request fails or task not found
 * 
 * Requirements: 2.3 - Return current status and progress percentage
 */
export async function getTask(id: number): Promise<Task> {
  const response = await fetch(`${API_BASE_URL}/tasks/${id}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });
  return handleResponse<Task>(response);
}

/**
 * Delete a video generation task
 * DELETE /api/tasks/:id
 * 
 * @param id - The task ID to delete
 * @returns The delete response
 * @throws ApiError if the request fails
 * 
 * Requirements: 5.1 - Remove video file and database record
 */
export async function deleteTask(id: number): Promise<DeleteTaskResponse> {
  const response = await fetch(`${API_BASE_URL}/tasks/${id}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
  });
  return handleResponse<DeleteTaskResponse>(response);
}

/**
 * Get the URL for a video file
 * 
 * @param filename - The local filename of the video
 * @returns The full URL to access the video
 * 
 * Requirements: 4.3 - Allow video playback from local file
 */
export function getVideoUrl(filename: string): string {
  return `${API_BASE_URL}/videos/${encodeURIComponent(filename)}`;
}

/**
 * Get the URL for a character profile picture
 * 
 * @param filename - The local filename of the picture
 * @returns The full URL to access the picture
 */
export function getCharacterPictureUrl(filename: string): string {
  return `${API_BASE_URL}/character-pictures/${encodeURIComponent(filename)}`;
}


/**
 * Delete all failed tasks
 * DELETE /api/tasks-failed
 * 
 * @returns Number of deleted tasks
 * @throws ApiError if the request fails
 */
export async function deleteFailedTasks(): Promise<{ deleted: number }> {
  const response = await fetch(`${API_BASE_URL}/tasks-failed`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
  });
  return handleResponse<{ deleted: number }>(response);
}


/**
 * Delete tasks by date range
 * DELETE /api/tasks-by-date?start=YYYY-MM-DD&end=YYYY-MM-DD
 * 
 * @param startDate - Start date (YYYY-MM-DD)
 * @param endDate - End date (YYYY-MM-DD)
 * @returns Number of deleted tasks
 * @throws ApiError if the request fails
 */
export async function deleteTasksByDateRange(startDate: string, endDate: string): Promise<{ deleted: number; message: string }> {
  const response = await fetch(`${API_BASE_URL}/tasks-by-date?start=${startDate}&end=${endDate}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
  });
  return handleResponse<{ deleted: number; message: string }>(response);
}


// ============================================
// Character API Functions (for character-creation feature)
// ============================================

/**
 * Create a new character from a video task
 * POST /api/characters
 * 
 * @param request - The character creation request containing custom_name, description, from_task_id, and timestamps
 * @returns The created character
 * @throws ApiError if the request fails
 * 
 * Requirements: 1.5 - Call VectorEngine Characters API with from_task ID and timestamps
 */
export async function createCharacter(request: CreateCharacterRequest): Promise<Character> {
  const response = await fetch(`${API_BASE_URL}/characters`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });
  return handleResponse<Character>(response);
}

/**
 * Get all characters
 * GET /api/characters
 * 
 * @returns List of all characters
 * @throws ApiError if the request fails
 * 
 * Requirements: 3.1 - Fetch and display all saved characters
 */
export async function getCharacters(): Promise<Character[]> {
  const response = await fetch(`${API_BASE_URL}/characters`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });
  const data = await handleResponse<CharacterListResponse>(response);
  return data.characters;
}

/**
 * Delete a character by ID
 * DELETE /api/characters/:id
 * 
 * @param id - The character ID to delete
 * @returns The delete response
 * @throws ApiError if the request fails
 * 
 * Requirements: 3.3 - Remove character record from database
 */
export async function deleteCharacter(id: number): Promise<DeleteCharacterResponse> {
  const response = await fetch(`${API_BASE_URL}/characters/${id}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
  });
  return handleResponse<DeleteCharacterResponse>(response);
}

/**
 * Get character training status
 * GET /api/characters/:id/status
 * 
 * @param id - The character ID to query status for
 * @returns The character status response with progress and status
 * @throws ApiError if the request fails
 * 
 * Requirements: 3.2 - Poll the API and update the progress percentage
 */
export async function getCharacterStatus(id: number): Promise<CharacterStatusResponse> {
  const response = await fetch(`${API_BASE_URL}/characters/${id}/status`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });
  return handleResponse<CharacterStatusResponse>(response);
}

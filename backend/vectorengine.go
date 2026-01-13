package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// VectorEngineBaseURL is the base URL for the VectorEngine API
	VectorEngineBaseURL = "https://api.vectorengine.ai"
	// DyuAPIBaseURL is the base URL for the Dyu API (sora2-alt)
	DyuAPIBaseURL = "https://api.dyuapi.com"
	// OutputDirectory is the directory where downloaded videos are saved
	OutputDirectory = "output"
)

// VectorEngineClient handles communication with the VectorEngine API
type VectorEngineClient struct {
	httpClient *http.Client
	baseURL    string
	dyuAPIKey  string
}

// NewVectorEngineClient creates a new VectorEngine API client
func NewVectorEngineClient(dyuAPIKey string) *VectorEngineClient {
	return &VectorEngineClient{
		httpClient: &http.Client{
			// No timeout - let requests complete naturally
			// Errors will be displayed to the user
		},
		baseURL:   VectorEngineBaseURL,
		dyuAPIKey: dyuAPIKey,
	}
}

// VectorEngineCreateRequest represents the request body for creating a video task (sora-2)
type VectorEngineCreateRequest struct {
	Images      []string `json:"images,omitempty"`
	Model       string   `json:"model"`
	Orientation string   `json:"orientation"`
	Prompt      string   `json:"prompt"`
	Size        string   `json:"size"`
	Duration    int      `json:"duration"`
	Watermark   bool     `json:"watermark"`
}

// CreateVideoTaskDyuAPI submits a video generation task to Dyu API
// - Text-to-video (no image): uses application/json format
// - Image-to-video (with image): uses multipart/form-data format
func (c *VectorEngineClient) CreateVideoTaskDyuAPI(prompt, imageURL, duration, orientation string) (*VectorEngineCreateResponse, error) {
	// Map duration and orientation to model name
	// sora2-portrait-test, sora2-landscape-test, sora2-portrait-15s-test, sora2-landscape-15s-test
	var modelName string
	if duration == "15s" {
		if orientation == "landscape" {
			modelName = "sora2-landscape-15s-test"
		} else {
			modelName = "sora2-portrait-15s-test"
		}
	} else {
		// 10s (default)
		if orientation == "landscape" {
			modelName = "sora2-landscape-test"
		} else {
			modelName = "sora2-portrait-test"
		}
	}

	log.Printf("[VideoGen] 使用模型: %s, 有图片: %v", modelName, imageURL != "")

	// If no image, use JSON format (text-to-video)
	if imageURL == "" {
		result, err := c.createVideoTaskJSON(prompt, modelName)
		// If -test model shows "暂无渠道", fallback to non-test model
		if err != nil {
			errStr := err.Error()
			log.Printf("[VideoGen] 创建任务失败: %s", errStr)
			if strings.Contains(errStr, "暂无渠道") && strings.HasSuffix(modelName, "-test") {
				fallbackModel := strings.TrimSuffix(modelName, "-test")
				log.Printf("[VideoGen] -test 模型暂无渠道，回退到: %s", fallbackModel)
				return c.createVideoTaskJSON(prompt, fallbackModel)
			}
		}
		return result, err
	}

	// If has image, use multipart/form-data format (image-to-video)
	result, err := c.createVideoTaskMultipart(prompt, imageURL, modelName)
	// If -test model shows "暂无渠道", fallback to non-test model
	if err != nil {
		errStr := err.Error()
		log.Printf("[VideoGen] 创建任务失败: %s", errStr)
		if strings.Contains(errStr, "暂无渠道") && strings.HasSuffix(modelName, "-test") {
			fallbackModel := strings.TrimSuffix(modelName, "-test")
			log.Printf("[VideoGen] -test 模型暂无渠道，回退到: %s", fallbackModel)
			return c.createVideoTaskMultipart(prompt, imageURL, fallbackModel)
		}
	}
	return result, err
}

// createVideoTaskJSON creates a video task using JSON format (for text-to-video)
func (c *VectorEngineClient) createVideoTaskJSON(prompt, modelName string) (*VectorEngineCreateResponse, error) {
	reqBody := map[string]interface{}{
		"prompt": prompt,
		"model":  modelName,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", DyuAPIBaseURL+"/v1/videos", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.dyuAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.dyuAPIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result VectorEngineCreateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// createVideoTaskMultipart creates a video task using multipart/form-data format (for image-to-video)
func (c *VectorEngineClient) createVideoTaskMultipart(prompt, imageURL, modelName string) (*VectorEngineCreateResponse, error) {
	boundary := "wL36Yn8afVp8Ag7AmP8qZ0SA4n1v9T"
	var body bytes.Buffer

	// Helper function to add a text field
	addField := func(name, value string) {
		body.WriteString("--" + boundary + "\r\n")
		body.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"%s\"\r\n", name))
		body.WriteString("Content-Type: text/plain\r\n")
		body.WriteString("\r\n")
		body.WriteString(value + "\r\n")
	}

	// Add model field
	addField("model", modelName)

	// Add prompt field
	addField("prompt", prompt)

	// Add input_reference (image)
	// Check if it's a base64 data URL
	if strings.HasPrefix(imageURL, "data:image/") {
		// Parse data URL: data:image/png;base64,xxxxx
		parts := strings.SplitN(imageURL, ",", 2)
		if len(parts) == 2 {
			// Get mime type from the first part
			mimeType := "image/png"
			if strings.Contains(parts[0], "image/jpeg") {
				mimeType = "image/jpeg"
			} else if strings.Contains(parts[0], "image/gif") {
				mimeType = "image/gif"
			} else if strings.Contains(parts[0], "image/webp") {
				mimeType = "image/webp"
			}

			// Decode base64
			imageData, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 image: %w", err)
			}

			// Determine file extension
			ext := ".png"
			if mimeType == "image/jpeg" {
				ext = ".jpg"
			} else if mimeType == "image/gif" {
				ext = ".gif"
			} else if mimeType == "image/webp" {
				ext = ".webp"
			}

			// Add image as file field
			body.WriteString("--" + boundary + "\r\n")
			body.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"input_reference\"; filename=\"image%s\"\r\n", ext))
			body.WriteString(fmt.Sprintf("Content-Type: %s\r\n", mimeType))
			body.WriteString("\r\n")
			body.Write(imageData)
			body.WriteString("\r\n")
		}
	}

	// End boundary
	body.WriteString("--" + boundary + "--\r\n")

	req, err := http.NewRequest("POST", DyuAPIBaseURL+"/v1/videos", &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	if c.dyuAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.dyuAPIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result VectorEngineCreateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// CreateVideoTask submits a new video generation task to Dyu API
func (c *VectorEngineClient) CreateVideoTask(prompt, imageURL, imageURL2, duration, orientation, model string) (*VectorEngineCreateResponse, error) {
	if c.dyuAPIKey == "" {
		return nil, fmt.Errorf("未配置API密钥，请在config.json中配置dyu_api_key")
	}
	return c.CreateVideoTaskDyuAPI(prompt, imageURL, duration, orientation)
}

// QueryTaskStatus queries the status of a video generation task from Dyu API
func (c *VectorEngineClient) QueryTaskStatus(taskID string) (*VectorEngineQueryResponse, error) {
	// Use Dyu API: /v1/videos/{task_id}
	req, err := http.NewRequest("GET", DyuAPIBaseURL+"/v1/videos/"+taskID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.dyuAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.dyuAPIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result VectorEngineQueryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse Dyu API response format
	// Dyu returns: {"id": "video_xxx", "status": "completed/FAILURE", "progress": 100, "video_url": "...", "fail_reason": "..."}
	var dyuResp struct {
		ID         string `json:"id"`
		Status     string `json:"status"`
		Progress   int    `json:"progress"`
		VideoURL   string `json:"video_url,omitempty"`
		FailReason string `json:"fail_reason,omitempty"`
	}
	if err := json.Unmarshal(body, &dyuResp); err == nil {
		result.Status = dyuResp.Status
		result.Progress = dyuResp.Progress
		if dyuResp.VideoURL != "" {
			result.VideoURL = dyuResp.VideoURL
		}
		if dyuResp.FailReason != "" {
			result.FailReason = dyuResp.FailReason
		}
	}

	return &result, nil
}

// isNumericString checks if a string contains only digits
func isNumericString(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// GenerateVideoFilename generates a unique filename for a downloaded video
// using the task ID and current timestamp
// Replaces invalid characters (like ':') for Windows compatibility
func GenerateVideoFilename(taskID string) string {
	timestamp := time.Now().UnixNano()
	// Replace ':' with '_' for Windows compatibility (e.g., "sora-2:xxx" -> "sora-2_xxx")
	safeTaskID := strings.ReplaceAll(taskID, ":", "_")
	return fmt.Sprintf("%s_%d.mp4", safeTaskID, timestamp)
}

// EnsureOutputDirectory creates the output directory if it doesn't exist
func EnsureOutputDirectory() error {
	if err := os.MkdirAll(OutputDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}

// DownloadVideo downloads a video from the given URL and saves it to the output directory
// Uses multi-threaded download for faster speeds
// Returns the local filename (not full path) of the saved video
func (c *VectorEngineClient) DownloadVideo(videoURL, taskID string) (string, error) {
	// Ensure output directory exists
	if err := EnsureOutputDirectory(); err != nil {
		return "", err
	}

	// Generate unique filename
	filename := GenerateVideoFilename(taskID)
	localPath := filepath.Join(OutputDirectory, filename)

	// First, get the file size with a HEAD request
	headResp, err := c.httpClient.Head(videoURL)
	if err != nil {
		// Fallback to simple download if HEAD fails
		return c.downloadVideoSimple(videoURL, localPath, filename)
	}
	headResp.Body.Close()

	contentLength := headResp.ContentLength
	acceptRanges := headResp.Header.Get("Accept-Ranges")

	// If server doesn't support range requests or file is small, use simple download
	if acceptRanges != "bytes" || contentLength <= 0 || contentLength < 1024*1024 {
		return c.downloadVideoSimple(videoURL, localPath, filename)
	}

	log.Printf("[Download] 使用多线程下载, 文件大小: %.2f MB", float64(contentLength)/1024/1024)

	// Use multi-threaded download
	numThreads := 8
	if contentLength < 10*1024*1024 { // Less than 10MB
		numThreads = 4
	}

	return c.downloadVideoMultiThread(videoURL, localPath, filename, contentLength, numThreads)
}

// downloadVideoSimple downloads video using simple single-thread method
func (c *VectorEngineClient) downloadVideoSimple(videoURL, localPath, filename string) (string, error) {
	resp, err := c.httpClient.Get(videoURL)
	if err != nil {
		return "", fmt.Errorf("failed to download video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download video: status %d", resp.StatusCode)
	}

	// Create the output file
	outFile, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy the response body to the file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		os.Remove(localPath)
		return "", fmt.Errorf("failed to save video: %w", err)
	}

	return filename, nil
}

// downloadVideoMultiThread downloads video using multiple threads
func (c *VectorEngineClient) downloadVideoMultiThread(videoURL, localPath, filename string, contentLength int64, numThreads int) (string, error) {
	// Create the output file
	outFile, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}

	// Pre-allocate file size
	if err := outFile.Truncate(contentLength); err != nil {
		outFile.Close()
		os.Remove(localPath)
		return "", fmt.Errorf("failed to allocate file: %w", err)
	}
	outFile.Close()

	// Calculate chunk size
	chunkSize := contentLength / int64(numThreads)

	var wg sync.WaitGroup
	errChan := make(chan error, numThreads)

	for i := 0; i < numThreads; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1
		if i == numThreads-1 {
			end = contentLength - 1 // Last chunk gets the remainder
		}

		wg.Add(1)
		go func(threadID int, start, end int64) {
			defer wg.Done()
			err := c.downloadChunk(videoURL, localPath, start, end)
			if err != nil {
				errChan <- fmt.Errorf("thread %d failed: %w", threadID, err)
			}
		}(i, start, end)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		os.Remove(localPath)
		return "", err
	}

	log.Printf("[Download] 多线程下载完成: %s", filename)
	return filename, nil
}

// downloadChunk downloads a specific byte range of the file
func (c *VectorEngineClient) downloadChunk(videoURL, localPath string, start, end int64) error {
	req, err := http.NewRequest("GET", videoURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Open file for writing at specific position
	file, err := os.OpenFile(localPath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Seek to the correct position
	if _, err := file.Seek(start, 0); err != nil {
		return err
	}

	// Copy data
	_, err = io.Copy(file, resp.Body)
	return err
}

// DeleteVideoFile removes a video file from the output directory
func DeleteVideoFile(filename string) error {
	if filename == "" {
		return nil
	}
	localPath := filepath.Join(OutputDirectory, filename)
	err := os.Remove(localPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete video file: %w", err)
	}
	return nil
}

// CreateCharacterSora2 creates a new character using Sora2 Character Training API
// API: POST https://api.dyuapi.com/v1/videos
// Supports both task ID (character param) and URL (url param)
// Sets model="character-training", prompt="角色创建"
func (c *VectorEngineClient) CreateCharacterSora2(sourceType, sourceValue, timestamps string) (*Sora2CharacterResponse, error) {
	reqBody := Sora2CharacterRequest{
		Prompt:     "角色创建",
		Model:      "character-training",
		Timestamps: timestamps,
	}

	// Set character or url based on source type
	if sourceType == "task" {
		reqBody.Character = sourceValue
	} else if sourceType == "url" {
		reqBody.URL = sourceValue
	} else {
		return nil, fmt.Errorf("invalid source type: %s, must be 'task' or 'url'", sourceType)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", DyuAPIBaseURL+"/v1/videos", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.dyuAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.dyuAPIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result Sora2CharacterResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// QueryCharacterStatus queries the training status of a character from Sora2 API
// API: GET https://api.dyuapi.com/v1/videos/{id}
// Returns status, progress, and fail_reason
func (c *VectorEngineClient) QueryCharacterStatus(characterID string) (*Sora2CharacterResponse, error) {
	req, err := http.NewRequest("GET", DyuAPIBaseURL+"/v1/videos/"+characterID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.dyuAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.dyuAPIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result Sora2CharacterResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// CharacterPictureDirectory is the directory where character profile pictures are saved
const CharacterPictureDirectory = "output/characters"

// EnsureCharacterPictureDirectory creates the character picture directory if it doesn't exist
func EnsureCharacterPictureDirectory() error {
	if err := os.MkdirAll(CharacterPictureDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create character picture directory: %w", err)
	}
	return nil
}

// DownloadCharacterPicture downloads a character profile picture and saves it locally
// Returns the local filename (not full path) of the saved picture
func (c *VectorEngineClient) DownloadCharacterPicture(pictureURL, characterID string) (string, error) {
	if pictureURL == "" {
		return "", nil
	}

	// Ensure directory exists
	if err := EnsureCharacterPictureDirectory(); err != nil {
		return "", err
	}

	// Generate unique filename
	safeCharID := strings.ReplaceAll(characterID, ":", "_")
	safeCharID = strings.ReplaceAll(safeCharID, "/", "_")
	filename := fmt.Sprintf("%s_%d.jpg", safeCharID, time.Now().UnixNano())
	localPath := filepath.Join(CharacterPictureDirectory, filename)

	// Download the picture
	resp, err := c.httpClient.Get(pictureURL)
	if err != nil {
		return "", fmt.Errorf("failed to download picture: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download picture: status %d", resp.StatusCode)
	}

	// Create the output file
	outFile, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy the response body to the file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		os.Remove(localPath)
		return "", fmt.Errorf("failed to save picture: %w", err)
	}

	return filename, nil
}

// DeleteCharacterPicture removes a character picture file
func DeleteCharacterPicture(filename string) error {
	if filename == "" {
		return nil
	}
	localPath := filepath.Join(CharacterPictureDirectory, filename)
	err := os.Remove(localPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete character picture: %w", err)
	}
	return nil
}

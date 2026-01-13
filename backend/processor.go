package main

import (
	"log"
	"sync"
	"time"
)

const (
	// PollInterval is the interval between polling for task status updates
	PollInterval = 3 * time.Second
)

// TaskProcessor handles background processing of video generation tasks
type TaskProcessor struct {
	client   *VectorEngineClient
	stopChan chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
}

// NewTaskProcessor creates a new task processor with the given API key
func NewTaskProcessor(dyuAPIKey string) *TaskProcessor {
	return &TaskProcessor{
		client:   NewVectorEngineClient(dyuAPIKey),
		stopChan: make(chan struct{}),
	}
}

// Start begins the background task processing loop
func (p *TaskProcessor) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	p.wg.Add(1)
	go p.processLoop()
	log.Println("Task processor started")
}

// Stop gracefully stops the task processor
func (p *TaskProcessor) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	close(p.stopChan)
	p.wg.Wait()
	log.Println("Task processor stopped")
}

// processLoop is the main processing loop that polls for pending tasks
func (p *TaskProcessor) processLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	// Process immediately on start
	p.processPendingTasks()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.processPendingTasks()
		}
	}
}

// processPendingTasks processes all pending and processing tasks
func (p *TaskProcessor) processPendingTasks() {
	tasks, err := GetPendingTasks()
	if err != nil {
		log.Printf("Error getting pending tasks: %v", err)
		return
	}

	for _, task := range tasks {
		select {
		case <-p.stopChan:
			return
		default:
			p.processTask(&task)
		}
	}
}

// processTask handles a single task based on its current status
func (p *TaskProcessor) processTask(task *Task) {
	switch task.Status {
	case StatusPending:
		p.submitTask(task)
	case StatusProcessing:
		p.pollTaskStatus(task)
	}
}

// submitTask submits a pending task to the API
func (p *TaskProcessor) submitTask(task *Task) {
	log.Printf("提交视频任务 %d", task.ID)

	model := task.Model
	if model == "" {
		model = ModelSora2
	}

	resp, err := p.client.CreateVideoTask(task.Prompt, task.ImageURL, task.ImageURL2, task.Duration, task.Orientation, model)
	if err != nil {
		log.Printf("任务 %d 提交失败: %v", task.ID, err)
		task.Status = StatusFailed
		task.FailReason = err.Error()
		if err := UpdateTask(task); err != nil {
			log.Printf("更新任务 %d 状态失败: %v", task.ID, err)
		}
		return
	}

	// Update task with task ID and set status to processing
	task.TaskID = resp.ID
	task.Status = StatusProcessing
	if err := UpdateTask(task); err != nil {
		log.Printf("更新任务 %d 失败: %v", task.ID, err)
	}
	log.Printf("视频任务 %d 提交成功，任务ID: %s", task.ID, resp.ID)
}

// pollTaskStatus polls the API for task status updates
func (p *TaskProcessor) pollTaskStatus(task *Task) {
	if task.TaskID == "" {
		log.Printf("任务 %d 没有任务ID，标记为失败", task.ID)
		task.Status = StatusFailed
		task.FailReason = "任务ID为空"
		if err := UpdateTask(task); err != nil {
			log.Printf("更新任务 %d 状态失败: %v", task.ID, err)
		}
		return
	}

	resp, err := p.client.QueryTaskStatus(task.TaskID)
	if err != nil {
		log.Printf("查询任务 %d 状态失败: %v (将重试)", task.ID, err)
		// Don't mark as failed immediately, just log and retry on next poll
		return
	}

	// Check if API returned an error
	if resp.Error != nil {
		log.Printf("任务 %d API错误: %s", task.ID, resp.Error.Message)
		task.Status = StatusFailed
		task.FailReason = resp.Error.Message
		if err := UpdateTask(task); err != nil {
			log.Printf("更新任务 %d 状态失败: %v", task.ID, err)
		}
		return
	}

	// Check for fail_reason in response
	if resp.FailReason != "" {
		log.Printf("任务 %d 失败: %s", task.ID, resp.FailReason)
		task.Status = StatusFailed
		task.FailReason = resp.FailReason
		if err := UpdateTask(task); err != nil {
			log.Printf("更新任务 %d 状态失败: %v", task.ID, err)
		}
		return
	}

	// Update progress
	task.Progress = resp.Progress

	// Handle status changes
	switch resp.Status {
	case "completed", "success":
		p.handleTaskCompletion(task, resp)
	case "failed", "error", "FAILURE":
		task.Status = StatusFailed
		if resp.FailReason != "" {
			task.FailReason = resp.FailReason
		}
		if err := UpdateTask(task); err != nil {
			log.Printf("更新任务 %d 状态失败: %v", task.ID, err)
		}
		log.Printf("任务 %d 失败", task.ID)
	default:
		// Still processing, just update progress
		if err := UpdateTask(task); err != nil {
			log.Printf("更新任务 %d 进度失败: %v", task.ID, err)
		}
	}
}

// handleTaskCompletion handles a completed task by downloading the video
func (p *TaskProcessor) handleTaskCompletion(task *Task, resp *VectorEngineQueryResponse) {
	log.Printf("Task %d completed, downloading video", task.ID)

	task.VideoURL = resp.VideoURL
	task.Progress = 100

	if resp.VideoURL != "" {
		// Download the video with retry until success
		maxRetries := 10
		retryDelay := 5 * time.Second

		for attempt := 1; attempt <= maxRetries; attempt++ {
			filename, err := p.client.DownloadVideo(resp.VideoURL, task.TaskID)
			if err == nil {
				task.LocalPath = filename
				log.Printf("Video downloaded for task %d: %s", task.ID, filename)
				break
			}

			log.Printf("Failed to download video for task %d (attempt %d/%d): %v", task.ID, attempt, maxRetries, err)

			if attempt < maxRetries {
				log.Printf("Retrying download for task %d in %v...", task.ID, retryDelay)
				time.Sleep(retryDelay)
			}
		}

		// If still no local path after all retries, keep task in processing state to retry later
		if task.LocalPath == "" {
			log.Printf("Task %d: video download failed after %d attempts, will retry on next poll", task.ID, maxRetries)
			// Don't mark as completed, keep processing so it will be retried
			if err := UpdateTask(task); err != nil {
				log.Printf("Failed to update task %d: %v", task.ID, err)
			}
			return
		}
	}

	task.Status = StatusCompleted
	if err := UpdateTask(task); err != nil {
		log.Printf("Failed to update task %d to completed: %v", task.ID, err)
	}
	log.Printf("Task %d completed successfully", task.ID)
}

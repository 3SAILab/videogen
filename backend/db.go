package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DB is the global database connection
var DB *sql.DB

// InitDB initializes the SQLite database and creates required tables
func InitDB(dbPath string) error {
	var err error
	// Add connection parameters for better concurrency
	// _busy_timeout: wait up to 5 seconds when database is locked
	// _journal_mode=WAL: use Write-Ahead Logging for better concurrency
	// _synchronous=NORMAL: balance between safety and performance
	connStr := dbPath + "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL"
	DB, err = sql.Open("sqlite", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for better concurrency
	DB.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	DB.SetMaxIdleConns(1)
	DB.SetConnMaxLifetime(0) // Don't close idle connections

	// Test the connection
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tasks table if not exists
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT,
		prompt TEXT NOT NULL,
		image_url TEXT,
		duration TEXT NOT NULL,
		orientation TEXT NOT NULL,
		model TEXT DEFAULT 'sora-2',
		status TEXT DEFAULT 'pending',
		progress INTEGER DEFAULT 0,
		video_url TEXT,
		local_path TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = DB.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}

	// Remove UNIQUE constraint from task_id if it exists (for existing databases)
	// SQLite doesn't support DROP CONSTRAINT, so we need to recreate the table
	// For now, just try to drop the index if it exists
	_, _ = DB.Exec("DROP INDEX IF EXISTS sqlite_autoindex_tasks_1")

	// Add model column if it doesn't exist (for existing databases)
	_, _ = DB.Exec("ALTER TABLE tasks ADD COLUMN model TEXT DEFAULT 'sora-2'")

	// Add image_url2 column if it doesn't exist (for Veo3 second image)
	_, _ = DB.Exec("ALTER TABLE tasks ADD COLUMN image_url2 TEXT")

	// Add fail_reason column if it doesn't exist
	_, _ = DB.Exec("ALTER TABLE tasks ADD COLUMN fail_reason TEXT")

	// Create characters table if not exists (new schema for Sora2 Character Training API)
	createCharactersTableSQL := `
	CREATE TABLE IF NOT EXISTS characters (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		api_character_id TEXT,
		username TEXT,
		custom_name TEXT NOT NULL,
		description TEXT,
		source_type TEXT NOT NULL,
		source_value TEXT NOT NULL,
		timestamps TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		progress INTEGER DEFAULT 0,
		fail_reason TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = DB.Exec(createCharactersTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create characters table: %w", err)
	}

	// Migrate old characters table schema to new schema if needed
	migrateCharactersTable()

	// Add username column if not exists
	addUsernameColumn()

	// Migration: Remove UNIQUE constraint from task_id
	migrateTasksTable()

	// Create indexes for better query performance (especially for large datasets)
	// Index on created_at for faster sorting (ORDER BY created_at DESC)
	_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at DESC)")
	// Index on status for faster filtering (WHERE status IN ...)
	_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)")
	// Composite index for common query pattern (status + created_at)
	_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_status_created ON tasks(status, created_at DESC)")

	return nil
}

// migrateTasksTable removes UNIQUE constraint from task_id column
// SQLite doesn't support ALTER TABLE DROP CONSTRAINT, so we need to recreate the table
func migrateTasksTable() {
	// Try to insert two rows with empty task_id to check if UNIQUE constraint exists
	// If it fails, we need to migrate
	_, err := DB.Exec("INSERT INTO tasks (task_id, prompt, duration, orientation) VALUES ('', 'test', '10s', 'landscape')")
	if err == nil {
		// First insert succeeded, try second
		_, err2 := DB.Exec("INSERT INTO tasks (task_id, prompt, duration, orientation) VALUES ('', 'test2', '10s', 'landscape')")
		// Clean up test rows
		DB.Exec("DELETE FROM tasks WHERE prompt = 'test' OR prompt = 'test2'")
		if err2 == nil {
			// No UNIQUE constraint, no migration needed
			return
		}
	} else {
		// Clean up if first insert somehow succeeded
		DB.Exec("DELETE FROM tasks WHERE prompt = 'test'")
	}

	// Need to migrate - recreate table without UNIQUE constraint
	log.Println("Migrating tasks table to remove UNIQUE constraint on task_id...")

	tx, err := DB.Begin()
	if err != nil {
		log.Printf("Migration failed to start transaction: %v", err)
		return
	}
	defer tx.Rollback()

	// Create new table without UNIQUE constraint
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS tasks_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT,
			prompt TEXT NOT NULL,
			image_url TEXT,
			duration TEXT NOT NULL,
			orientation TEXT NOT NULL,
			model TEXT DEFAULT 'sora-2',
			status TEXT DEFAULT 'pending',
			progress INTEGER DEFAULT 0,
			video_url TEXT,
			local_path TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			image_url2 TEXT
		)`)
	if err != nil {
		log.Printf("Migration failed to create new table: %v", err)
		return
	}

	// Copy data - explicitly specify columns to handle column order differences
	_, err = tx.Exec(`INSERT INTO tasks_new (id, task_id, prompt, image_url, duration, orientation, model, status, progress, video_url, local_path, created_at, updated_at, image_url2)
		SELECT id, task_id, prompt, image_url, duration, orientation, model, status, progress, video_url, local_path, created_at, updated_at, image_url2 FROM tasks`)
	if err != nil {
		log.Printf("Migration failed to copy data: %v", err)
		return
	}

	// Drop old table
	_, err = tx.Exec(`DROP TABLE tasks`)
	if err != nil {
		log.Printf("Migration failed to drop old table: %v", err)
		return
	}

	// Rename new table
	_, err = tx.Exec(`ALTER TABLE tasks_new RENAME TO tasks`)
	if err != nil {
		log.Printf("Migration failed to rename table: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Migration failed to commit: %v", err)
		return
	}
	log.Println("Migration completed successfully")
}

// addUsernameColumn adds the username column to characters table if it doesn't exist
func addUsernameColumn() {
	// Check if username column exists
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('characters') WHERE name='username'").Scan(&count)
	if err != nil || count > 0 {
		// Column already exists or error checking
	} else {
		log.Println("Adding username column to characters table...")
		_, err = DB.Exec("ALTER TABLE characters ADD COLUMN username TEXT")
		if err != nil {
			log.Printf("Warning: Failed to add username column: %v", err)
		} else {
			log.Println("Username column added successfully")
		}
	}

	// Check if avatar_url column exists
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('characters') WHERE name='avatar_url'").Scan(&count)
	if err != nil || count > 0 {
		// Column already exists or error checking
		return
	}

	log.Println("Adding avatar_url column to characters table...")
	_, err = DB.Exec("ALTER TABLE characters ADD COLUMN avatar_url TEXT")
	if err != nil {
		log.Printf("Warning: Failed to add avatar_url column: %v", err)
		return
	}
	log.Println("Avatar_url column added successfully")
}

// migrateCharactersTable migrates the characters table from old schema to new schema
// Old schema: api_id, api_username, profile_picture_url, permalink, from_task_id, local_picture_path
// New schema: api_character_id, source_type, source_value, status, progress, fail_reason
func migrateCharactersTable() {
	// Check if old schema exists by looking for api_id column
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('characters') WHERE name='api_id'").Scan(&count)
	if err != nil || count == 0 {
		// No old schema, no migration needed
		return
	}

	log.Println("Migrating characters table to new schema...")

	tx, err := DB.Begin()
	if err != nil {
		log.Printf("Characters migration failed to start transaction: %v", err)
		return
	}
	defer tx.Rollback()

	// Create new table with new schema
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS characters_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_character_id TEXT,
			custom_name TEXT NOT NULL,
			description TEXT,
			source_type TEXT NOT NULL,
			source_value TEXT NOT NULL,
			timestamps TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			progress INTEGER DEFAULT 0,
			fail_reason TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		log.Printf("Characters migration failed to create new table: %v", err)
		return
	}

	// Copy data from old table to new table
	// Map old fields to new fields:
	// - api_id -> api_character_id
	// - from_task_id -> source_value (with source_type='task')
	// - Set status='completed' for existing characters (they were already created)
	// - Set progress=100 for existing characters
	_, err = tx.Exec(`
		INSERT INTO characters_new (id, api_character_id, custom_name, description, source_type, source_value, timestamps, status, progress, created_at)
		SELECT id, api_id, custom_name, description, 'task', from_task_id, timestamps, 'completed', 100, created_at
		FROM characters`)
	if err != nil {
		log.Printf("Characters migration failed to copy data: %v", err)
		return
	}

	// Drop old table
	_, err = tx.Exec(`DROP TABLE characters`)
	if err != nil {
		log.Printf("Characters migration failed to drop old table: %v", err)
		return
	}

	// Rename new table
	_, err = tx.Exec(`ALTER TABLE characters_new RENAME TO characters`)
	if err != nil {
		log.Printf("Characters migration failed to rename table: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Characters migration failed to commit: %v", err)
		return
	}
	log.Println("Characters migration completed successfully")
}

// CloseDB closes the database connection
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// CreateTask inserts a new task into the database
func CreateTask(req *CreateTaskRequest) (*Task, error) {
	now := time.Now()
	model := req.Model
	if model == "" {
		model = ModelSora2
	}
	result, err := DB.Exec(`
		INSERT INTO tasks (prompt, image_url, image_url2, duration, orientation, model, status, progress, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Prompt, req.ImageURL, req.ImageURL2, req.Duration, req.Orientation, model, StatusPending, 0, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &Task{
		ID:          id,
		Prompt:      req.Prompt,
		ImageURL:    req.ImageURL,
		ImageURL2:   req.ImageURL2,
		Duration:    req.Duration,
		Orientation: req.Orientation,
		Model:       model,
		Status:      StatusPending,
		Progress:    0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// GetTask retrieves a single task by ID
func GetTask(id int64) (*Task, error) {
	task := &Task{}
	var imageURL, imageURL2, videoURL, localPath, taskID, model, failReason sql.NullString

	err := DB.QueryRow(`
		SELECT id, task_id, prompt, image_url, COALESCE(image_url2, '') as image_url2, duration, orientation, COALESCE(model, 'sora-2') as model, status, progress, video_url, local_path, COALESCE(fail_reason, '') as fail_reason, created_at, updated_at
		FROM tasks WHERE id = ?`, id).Scan(
		&task.ID, &taskID, &task.Prompt, &imageURL, &imageURL2, &task.Duration, &task.Orientation, &model,
		&task.Status, &task.Progress, &videoURL, &localPath, &failReason, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	task.TaskID = taskID.String
	task.ImageURL = imageURL.String
	task.ImageURL2 = imageURL2.String
	task.VideoURL = videoURL.String
	task.LocalPath = localPath.String
	task.Model = model.String
	task.FailReason = failReason.String

	return task, nil
}

// GetAllTasks retrieves all tasks from the database (without image_url for performance)
func GetAllTasks() ([]Task, error) {
	rows, err := DB.Query(`
		SELECT id, task_id, prompt, duration, orientation, COALESCE(model, 'sora-2') as model, status, progress, video_url, local_path, COALESCE(fail_reason, '') as fail_reason, created_at, updated_at
		FROM tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var videoURL, localPath, taskID, model, failReason sql.NullString

		err := rows.Scan(
			&task.ID, &taskID, &task.Prompt, &task.Duration, &task.Orientation, &model,
			&task.Status, &task.Progress, &videoURL, &localPath, &failReason, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		task.TaskID = taskID.String
		task.VideoURL = videoURL.String
		task.LocalPath = localPath.String
		task.Model = model.String
		task.FailReason = failReason.String
		// image_url is intentionally not loaded for performance (base64 images are large)

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// GetTasksPaginated retrieves tasks with pagination (without image_url for performance)
func GetTasksPaginated(limit, offset int) ([]Task, int, error) {
	// Get total count
	var total int
	err := DB.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	rows, err := DB.Query(`
		SELECT id, task_id, prompt, duration, orientation, COALESCE(model, 'sora-2') as model, status, progress, video_url, local_path, COALESCE(fail_reason, '') as fail_reason, created_at, updated_at
		FROM tasks ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var videoURL, localPath, taskID, model, failReason sql.NullString

		err := rows.Scan(
			&task.ID, &taskID, &task.Prompt, &task.Duration, &task.Orientation, &model,
			&task.Status, &task.Progress, &videoURL, &localPath, &failReason, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan task: %w", err)
		}

		task.TaskID = taskID.String
		task.VideoURL = videoURL.String
		task.LocalPath = localPath.String
		task.Model = model.String
		task.FailReason = failReason.String

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, total, nil
}

// GetTasksByStatus retrieves tasks with specific statuses (for polling pending tasks)
func GetTasksByStatus(statuses []string) ([]Task, error) {
	if len(statuses) == 0 {
		return []Task{}, nil
	}

	// Build query with placeholders
	placeholders := make([]string, len(statuses))
	args := make([]interface{}, len(statuses))
	for i, s := range statuses {
		placeholders[i] = "?"
		args[i] = s
	}

	query := fmt.Sprintf(`
		SELECT id, task_id, prompt, duration, orientation, COALESCE(model, 'sora-2') as model, status, progress, video_url, local_path, COALESCE(fail_reason, '') as fail_reason, created_at, updated_at
		FROM tasks WHERE status IN (%s) ORDER BY created_at DESC`,
		strings.Join(placeholders, ","))

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by status: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var videoURL, localPath, taskID, model, failReason sql.NullString

		err := rows.Scan(
			&task.ID, &taskID, &task.Prompt, &task.Duration, &task.Orientation, &model,
			&task.Status, &task.Progress, &videoURL, &localPath, &failReason, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		task.TaskID = taskID.String
		task.VideoURL = videoURL.String
		task.LocalPath = localPath.String
		task.Model = model.String
		task.FailReason = failReason.String

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// GetTasksByIds retrieves tasks by their IDs (for polling specific tasks)
func GetTasksByIds(ids []int64) ([]Task, error) {
	if len(ids) == 0 {
		return []Task{}, nil
	}

	// Build query with placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, task_id, prompt, duration, orientation, COALESCE(model, 'sora-2') as model, status, progress, video_url, local_path, COALESCE(fail_reason, '') as fail_reason, created_at, updated_at
		FROM tasks WHERE id IN (%s) ORDER BY created_at DESC`,
		strings.Join(placeholders, ","))

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by IDs: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var videoURL, localPath, taskID, model, failReason sql.NullString

		err := rows.Scan(
			&task.ID, &taskID, &task.Prompt, &task.Duration, &task.Orientation, &model,
			&task.Status, &task.Progress, &videoURL, &localPath, &failReason, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		task.TaskID = taskID.String
		task.VideoURL = videoURL.String
		task.LocalPath = localPath.String
		task.Model = model.String
		task.FailReason = failReason.String

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// UpdateTask updates an existing task in the database
func UpdateTask(task *Task) error {
	task.UpdatedAt = time.Now()
	_, err := DB.Exec(`
		UPDATE tasks SET
			task_id = ?,
			prompt = ?,
			image_url = ?,
			duration = ?,
			orientation = ?,
			model = ?,
			status = ?,
			progress = ?,
			video_url = ?,
			local_path = ?,
			fail_reason = ?,
			updated_at = ?
		WHERE id = ?`,
		task.TaskID, task.Prompt, task.ImageURL, task.Duration, task.Orientation, task.Model,
		task.Status, task.Progress, task.VideoURL, task.LocalPath, task.FailReason, task.UpdatedAt, task.ID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	return nil
}

// DeleteTask removes a task from the database by ID
func DeleteTask(id int64) error {
	result, err := DB.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil // Task not found, but that's okay per requirement 5.4
	}

	return nil
}

// GetPendingTasks retrieves all tasks that need processing (pending or processing status)
func GetPendingTasks() ([]Task, error) {
	rows, err := DB.Query(`
		SELECT id, task_id, prompt, image_url, COALESCE(image_url2, '') as image_url2, duration, orientation, COALESCE(model, 'sora-2') as model, status, progress, video_url, local_path, COALESCE(fail_reason, '') as fail_reason, created_at, updated_at
		FROM tasks 
		WHERE status IN (?, ?)
		ORDER BY created_at ASC`,
		StatusPending, StatusProcessing)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var imageURL, imageURL2, videoURL, localPath, taskID, model, failReason sql.NullString

		err := rows.Scan(
			&task.ID, &taskID, &task.Prompt, &imageURL, &imageURL2, &task.Duration, &task.Orientation, &model,
			&task.Status, &task.Progress, &videoURL, &localPath, &failReason, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		task.TaskID = taskID.String
		task.ImageURL = imageURL.String
		task.ImageURL2 = imageURL2.String
		task.VideoURL = videoURL.String
		task.LocalPath = localPath.String
		task.Model = model.String
		task.FailReason = failReason.String

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending tasks: %w", err)
	}

	return tasks, nil
}

// GetTasksByDateRange retrieves tasks within a date range
func GetTasksByDateRange(startDate, endDate string) ([]Task, error) {
	rows, err := DB.Query(`
		SELECT id, task_id, prompt, duration, orientation, COALESCE(model, 'sora-2') as model, status, progress, video_url, local_path, COALESCE(fail_reason, '') as fail_reason, created_at, updated_at
		FROM tasks 
		WHERE date(created_at) >= date(?) AND date(created_at) <= date(?)
		ORDER BY created_at DESC`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by date range: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var videoURL, localPath, taskID, model, failReason sql.NullString

		err := rows.Scan(
			&task.ID, &taskID, &task.Prompt, &task.Duration, &task.Orientation, &model,
			&task.Status, &task.Progress, &videoURL, &localPath, &failReason, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		task.TaskID = taskID.String
		task.VideoURL = videoURL.String
		task.LocalPath = localPath.String
		task.Model = model.String
		task.FailReason = failReason.String

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// CreateCharacter inserts a new character into the database
func CreateCharacter(char *Character) (*Character, error) {
	now := time.Now()
	// Set initial status to pending and progress to 0 for new characters
	status := char.Status
	if status == "" {
		status = StatusPending
	}
	progress := char.Progress
	if status == StatusPending {
		progress = 0
	}

	result, err := DB.Exec(`
		INSERT INTO characters (api_character_id, username, custom_name, description, source_type, source_value, timestamps, status, progress, fail_reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		char.ApiCharacterID, char.Username, char.CustomName, char.Description,
		char.SourceType, char.SourceValue, char.Timestamps, status, progress, char.FailReason, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert character: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	char.ID = id
	char.Status = status
	char.Progress = progress
	char.CreatedAt = now
	return char, nil
}

// GetAllCharacters retrieves all characters from the database ordered by created_at DESC
func GetAllCharacters() ([]Character, error) {
	rows, err := DB.Query(`
		SELECT id, COALESCE(api_character_id, '') as api_character_id, COALESCE(username, '') as username, 
		       COALESCE(avatar_url, '') as avatar_url, custom_name, COALESCE(description, '') as description, 
		       source_type, source_value, timestamps, status, progress, COALESCE(fail_reason, '') as fail_reason, created_at
		FROM characters ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query characters: %w", err)
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var char Character
		var apiCharacterID, username, avatarURL, description, failReason sql.NullString

		err := rows.Scan(
			&char.ID, &apiCharacterID, &username, &avatarURL, &char.CustomName, &description,
			&char.SourceType, &char.SourceValue, &char.Timestamps,
			&char.Status, &char.Progress, &failReason, &char.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan character: %w", err)
		}

		char.ApiCharacterID = apiCharacterID.String
		char.Username = username.String
		char.AvatarURL = avatarURL.String
		char.Description = description.String
		char.FailReason = failReason.String

		characters = append(characters, char)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating characters: %w", err)
	}

	return characters, nil
}

// GetCharacter retrieves a single character by ID
func GetCharacter(id int64) (*Character, error) {
	char := &Character{}
	var apiCharacterID, username, avatarURL, description, failReason sql.NullString

	err := DB.QueryRow(`
		SELECT id, COALESCE(api_character_id, '') as api_character_id, COALESCE(username, '') as username,
		       COALESCE(avatar_url, '') as avatar_url, custom_name, COALESCE(description, '') as description,
		       source_type, source_value, timestamps, status, progress, COALESCE(fail_reason, '') as fail_reason, created_at
		FROM characters WHERE id = ?`, id).Scan(
		&char.ID, &apiCharacterID, &username, &avatarURL, &char.CustomName, &description,
		&char.SourceType, &char.SourceValue, &char.Timestamps,
		&char.Status, &char.Progress, &failReason, &char.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get character: %w", err)
	}

	char.ApiCharacterID = apiCharacterID.String
	char.Username = username.String
	char.AvatarURL = avatarURL.String
	char.Description = description.String
	char.FailReason = failReason.String

	return char, nil
}

// UpdateCharacterStatus updates the status, progress, api_character_id, username, avatar_url, and fail_reason of a character
func UpdateCharacterStatus(id int64, status string, progress int, apiCharacterID string, username string, avatarURL string, failReason string) error {
	result, err := DB.Exec(`
		UPDATE characters SET
			status = ?,
			progress = ?,
			api_character_id = ?,
			username = ?,
			avatar_url = ?,
			fail_reason = ?
		WHERE id = ?`,
		status, progress, apiCharacterID, username, avatarURL, failReason, id)
	if err != nil {
		return fmt.Errorf("failed to update character status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("character not found")
	}

	return nil
}

// DeleteCharacter removes a character from the database by ID
func DeleteCharacter(id int64) error {
	result, err := DB.Exec("DELETE FROM characters WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete character: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("character not found")
	}

	return nil
}

// ResetFailedTasks resets failed and processing tasks to pending for retry
// Returns the number of tasks updated
func ResetFailedTasks() (int64, error) {
	result, err := DB.Exec(`
		UPDATE tasks SET
			status = ?,
			task_id = '',
			progress = 0,
			video_url = '',
			updated_at = ?
		WHERE status IN (?, ?)`,
		StatusPending, time.Now(), StatusFailed, StatusProcessing)
	if err != nil {
		return 0, fmt.Errorf("failed to reset tasks: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

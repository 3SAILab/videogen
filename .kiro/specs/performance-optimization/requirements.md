# Requirements Document

## Introduction

优化视频生成系统在大量数据（1000+视频记录）场景下的使用体验和流畅度。当前系统在视频数量增多后存在性能问题，需要从前端渲染、数据加载、轮询策略等多方面进行优化。

## Glossary

- **Virtual_Scrolling**: 虚拟滚动技术，只渲染可视区域内的元素，大幅减少DOM节点数量
- **Lazy_Loading**: 延迟加载，只在需要时才加载资源
- **Debounce**: 防抖，限制函数执行频率
- **Throttle**: 节流，限制函数执行频率
- **Intersection_Observer**: 浏览器API，用于检测元素是否进入可视区域
- **Polling**: 轮询，定期查询服务器获取更新
- **Pagination**: 分页，将大量数据分批加载

## Requirements

### Requirement 1: 虚拟滚动优化

**User Story:** As a user, I want the video gallery to remain smooth when scrolling through 1000+ videos, so that I can browse my video history without lag.

#### Acceptance Criteria

1. WHEN the video gallery contains more than 50 videos, THE System SHALL use virtual scrolling to render only visible items plus a buffer zone
2. WHEN scrolling through the gallery, THE System SHALL maintain 60fps frame rate without visible stuttering
3. WHEN virtual scrolling is active, THE System SHALL render at most 30 video cards at any time regardless of total video count
4. WHEN a video card scrolls out of the visible area, THE System SHALL remove it from DOM to free memory

### Requirement 2: 视频预览优化

**User Story:** As a user, I want video previews to load efficiently, so that the page doesn't freeze when many videos are visible.

#### Acceptance Criteria

1. WHEN a video card enters the viewport, THE System SHALL delay loading the video element by 200ms to avoid loading during fast scrolling
2. WHEN a video is not in the viewport, THE System SHALL pause playback and release video resources
3. WHEN more than 6 videos are simultaneously in viewport, THE System SHALL only auto-play the first 4 visible videos
4. THE System SHALL use video poster images instead of loading full video for initial display

### Requirement 3: 轮询策略优化

**User Story:** As a user, I want the system to efficiently check for task updates without impacting performance, so that I get timely updates without browser slowdown.

#### Acceptance Criteria

1. WHEN there are no pending/processing tasks, THE System SHALL stop polling entirely
2. WHEN polling for task updates, THE System SHALL only request updates for pending/processing tasks by ID, not reload all tasks
3. WHEN the browser tab is not visible, THE System SHALL pause polling to save resources
4. WHEN polling resumes after tab becomes visible, THE System SHALL immediately fetch updates once

### Requirement 4: 数据加载优化

**User Story:** As a user, I want the initial page load to be fast, so that I can start using the app quickly.

#### Acceptance Criteria

1. WHEN the page loads, THE System SHALL load only the first 20 tasks initially
2. WHEN scrolling near the bottom, THE System SHALL load the next batch of 20 tasks
3. WHEN loading more tasks, THE System SHALL show a loading indicator without blocking interaction
4. THE Backend SHALL return task data without image_url field in list queries to reduce payload size

### Requirement 5: 内存管理优化

**User Story:** As a user, I want the app to use memory efficiently, so that it doesn't slow down my browser over time.

#### Acceptance Criteria

1. WHEN a video element is removed from DOM, THE System SHALL properly dispose of video resources
2. WHEN the total loaded tasks exceed 200, THE System SHALL consider removing oldest tasks from memory
3. THE System SHALL avoid storing base64 image data in component state for displayed tasks
4. WHEN switching away from the app tab, THE System SHALL release non-essential resources

### Requirement 6: 渲染性能优化

**User Story:** As a user, I want UI interactions to be responsive, so that clicking buttons and scrolling feels instant.

#### Acceptance Criteria

1. WHEN updating task status, THE System SHALL only re-render affected task cards, not the entire list
2. WHEN the settings panel opens/closes, THE System SHALL not trigger re-render of video cards
3. THE System SHALL use React.memo or similar techniques to prevent unnecessary re-renders
4. WHEN deleting a task, THE System SHALL use optimistic UI update without waiting for server response

### Requirement 7: 后端查询优化

**User Story:** As a developer, I want the backend to handle large datasets efficiently, so that API responses remain fast.

#### Acceptance Criteria

1. WHEN querying tasks with pagination, THE Backend SHALL use database indexes for efficient sorting
2. WHEN returning task list, THE Backend SHALL exclude large fields (image_url, image_url2) unless specifically requested
3. THE Backend SHALL add database index on created_at column for faster sorting
4. WHEN querying tasks by status, THE Backend SHALL use indexed status column

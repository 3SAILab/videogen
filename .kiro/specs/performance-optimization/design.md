# Design Document: Performance Optimization

## Overview

本设计文档描述了视频生成系统在大数据量（1000+视频记录）场景下的性能优化方案。优化涵盖前端虚拟滚动、视频资源管理、轮询策略、数据加载和后端查询等多个方面。

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend (React)                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │ Virtual Scroll  │  │ Video Resource  │  │   Polling    │ │
│  │    Manager      │  │    Manager      │  │   Manager    │ │
│  └────────┬────────┘  └────────┬────────┘  └──────┬───────┘ │
│           │                    │                   │         │
│  ┌────────▼────────────────────▼───────────────────▼───────┐ │
│  │                    App State (React)                     │ │
│  │  - visibleRange: {start, end}                           │ │
│  │  - tasks: Task[] (paginated)                            │ │
│  │  - pendingTaskIds: number[]                             │ │
│  └─────────────────────────┬───────────────────────────────┘ │
└─────────────────────────────┼───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Backend (Go)                            │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │   API Handler   │  │   Database      │                   │
│  │  (Pagination)   │  │   (Indexed)     │                   │
│  └─────────────────┘  └─────────────────┘                   │
└─────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. VirtualScrollManager

负责管理虚拟滚动，只渲染可视区域内的视频卡片。

```typescript
interface VirtualScrollConfig {
  itemHeight: number;        // 每个视频卡片的高度
  containerHeight: number;   // 容器高度
  overscan: number;          // 缓冲区大小（上下各多渲染几个）
  columns: number;           // 列数
}

interface VirtualScrollState {
  scrollTop: number;
  visibleStartIndex: number;
  visibleEndIndex: number;
  totalHeight: number;
}

// 计算可见范围
function calculateVisibleRange(
  scrollTop: number,
  containerHeight: number,
  itemHeight: number,
  totalItems: number,
  columns: number,
  overscan: number
): { start: number; end: number } {
  const rowHeight = itemHeight;
  const startRow = Math.floor(scrollTop / rowHeight);
  const visibleRows = Math.ceil(containerHeight / rowHeight);
  
  const start = Math.max(0, (startRow - overscan) * columns);
  const end = Math.min(totalItems, (startRow + visibleRows + overscan) * columns);
  
  return { start, end };
}
```

### 2. VideoResourceManager

管理视频资源的加载和释放。

```typescript
interface VideoResourceManager {
  // 当前正在播放的视频ID集合
  playingVideos: Set<number>;
  
  // 最大同时播放数量
  maxConcurrentPlaying: number;
  
  // 注册视频进入视口
  onVideoEnterViewport(taskId: number): void;
  
  // 注册视频离开视口
  onVideoLeaveViewport(taskId: number): void;
  
  // 检查是否应该自动播放
  shouldAutoPlay(taskId: number): boolean;
}

// 实现逻辑
class VideoResourceManagerImpl implements VideoResourceManager {
  playingVideos = new Set<number>();
  maxConcurrentPlaying = 4;
  
  shouldAutoPlay(taskId: number): boolean {
    return this.playingVideos.size < this.maxConcurrentPlaying;
  }
  
  onVideoEnterViewport(taskId: number): void {
    if (this.shouldAutoPlay(taskId)) {
      this.playingVideos.add(taskId);
    }
  }
  
  onVideoLeaveViewport(taskId: number): void {
    this.playingVideos.delete(taskId);
  }
}
```

### 3. PollingManager

管理任务状态轮询。

```typescript
interface PollingManager {
  // 开始轮询指定任务
  startPolling(taskIds: number[]): void;
  
  // 停止轮询
  stopPolling(): void;
  
  // 暂停轮询（标签页不可见时）
  pausePolling(): void;
  
  // 恢复轮询
  resumePolling(): void;
}

// 使用 Page Visibility API
function useVisibilityPolling(
  pendingTaskIds: number[],
  onUpdate: (tasks: Task[]) => void
) {
  const [isVisible, setIsVisible] = useState(!document.hidden);
  
  useEffect(() => {
    const handleVisibilityChange = () => {
      setIsVisible(!document.hidden);
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, []);
  
  useEffect(() => {
    if (!isVisible || pendingTaskIds.length === 0) return;
    
    // 立即获取一次
    fetchTasksByIds(pendingTaskIds).then(onUpdate);
    
    // 设置轮询
    const interval = setInterval(() => {
      fetchTasksByIds(pendingTaskIds).then(onUpdate);
    }, 3000);
    
    return () => clearInterval(interval);
  }, [isVisible, pendingTaskIds]);
}
```

### 4. Backend API Optimization

优化后端API响应。

```go
// 添加数据库索引
const createIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
`

// 优化的任务列表查询（不包含大字段）
func GetTasksPaginatedOptimized(limit, offset int) ([]TaskSummary, int, error) {
    // 只查询必要字段，排除 image_url, image_url2
    query := `
        SELECT id, task_id, prompt, duration, orientation, model, 
               status, progress, video_url, local_path, created_at, updated_at
        FROM tasks 
        ORDER BY created_at DESC 
        LIMIT ? OFFSET ?`
    // ...
}
```

## Data Models

### TaskSummary (优化后的任务摘要)

```typescript
// 列表显示用的精简任务数据
interface TaskSummary {
  id: number;
  task_id: string;
  prompt: string;
  duration: string;
  orientation: string;
  model: string;
  status: string;
  progress: number;
  video_url: string;
  local_path: string;
  created_at: string;
  updated_at: string;
  // 注意：不包含 image_url 和 image_url2
}

// 完整任务数据（编辑时获取）
interface TaskFull extends TaskSummary {
  image_url?: string;
  image_url2?: string;
}
```

### VirtualScrollState

```typescript
interface VirtualScrollState {
  // 当前滚动位置
  scrollTop: number;
  
  // 可见任务的索引范围
  visibleRange: {
    start: number;
    end: number;
  };
  
  // 已加载的任务总数
  loadedCount: number;
  
  // 服务器上的任务总数
  totalCount: number;
  
  // 是否正在加载更多
  isLoadingMore: boolean;
  
  // 是否还有更多数据
  hasMore: boolean;
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Virtual Scroll DOM Limit

*For any* scroll position and any total task count greater than 50, the number of rendered video card DOM elements SHALL be at most 30.

**Validates: Requirements 1.1, 1.3**

### Property 2: Video Playback Limit

*For any* viewport state with more than 4 videos visible, the number of simultaneously auto-playing videos SHALL be at most 4.

**Validates: Requirements 2.3**

### Property 3: Video Pause on Exit Viewport

*For any* video that was playing and then scrolls out of the viewport, the video element SHALL be paused.

**Validates: Requirements 2.2**

### Property 4: Polling Stops When No Pending Tasks

*For any* state where all tasks have status "completed" or "failed", the system SHALL not make any polling API requests.

**Validates: Requirements 3.1**

### Property 5: Polling Uses Task IDs

*For any* polling request when there are pending tasks, the API request SHALL include only the IDs of pending/processing tasks, not request all tasks.

**Validates: Requirements 3.2**

### Property 6: API Response Excludes Large Fields

*For any* task list API response, the returned task objects SHALL NOT contain image_url or image_url2 fields.

**Validates: Requirements 4.4, 7.2**

### Property 7: Memory Limit on Loaded Tasks

*For any* state where more than 200 tasks have been loaded, the system SHALL maintain at most 200 tasks in memory by removing oldest tasks.

**Validates: Requirements 5.2**

### Property 8: Isolated Re-renders

*For any* task status update, only the affected task card component SHALL re-render, not sibling task cards.

**Validates: Requirements 6.1, 6.2**

## Error Handling

1. **虚拟滚动计算错误**: 如果计算出的可见范围无效，回退到渲染前50个任务
2. **视频加载失败**: 显示错误占位图，不影响其他视频
3. **轮询失败**: 静默重试，不显示错误提示
4. **分页加载失败**: 显示重试按钮，允许用户手动重试

## Testing Strategy

### Unit Tests

1. **VirtualScrollManager**: 测试可见范围计算逻辑
2. **VideoResourceManager**: 测试播放限制逻辑
3. **PollingManager**: 测试轮询启停逻辑

### Property-Based Tests

使用 fast-check 库进行属性测试：

1. **Property 1**: 生成随机任务数量和滚动位置，验证DOM元素数量限制
2. **Property 4**: 生成随机任务状态组合，验证轮询行为
3. **Property 6**: 生成随机API响应，验证字段排除

### Integration Tests

1. 测试完整的滚动加载流程
2. 测试标签页切换时的轮询行为
3. 测试大数据量下的性能指标

### Performance Tests

1. 使用 Lighthouse 测量首次加载性能
2. 使用 Chrome DevTools 测量滚动帧率
3. 使用内存分析工具检测内存泄漏

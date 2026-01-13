import React, { useState, useRef, useEffect, useCallback, memo } from 'react';
import { 
  Settings, 
  Plus, 
  Image as ImageIcon, 
  Smartphone, 
  Monitor, 
  X,
  Loader2,
  Play,
  Trash2,
  Download,
  RefreshCw,
  Edit3,
  ArrowUp,
  Check,
  Calendar,
  User
} from 'lucide-react';
import { createTask, getTasks, getTask, getTasksByIds, deleteTask, deleteFailedTasks, deleteTasksByDateRange, getVideoUrl } from './api';
import type { Task, Duration, Orientation, Count, Model, CreateTaskRequest, Character } from './types';
import CharacterCreationDialog from './CharacterCreationDialog';
import CharacterList from './CharacterList';

// Maximum number of videos that can auto-play simultaneously
const MAX_PLAYING_VIDEOS = 4;

// Global set to track currently playing videos
const playingVideosSet = new Set<number>();

// Video card props interface
interface VideoCardProps {
  task: Task;
  hasVideoError: boolean;
  isGenerating: boolean;
  deleteConfirmId: number | null;
  onPlay: (task: Task) => void;
  onEdit: (task: Task) => void;
  onRegenerate: (task: Task) => void;
  onDelete: (taskId: number) => void;
  onDeleteConfirm: (taskId: number | null) => void;
  onVideoError: (taskId: number) => void;
  onCreateCharacter: (task: Task) => void;
}

// Memoized video card component - prevents re-render when parent state changes
const VideoCard = memo(function VideoCard({
  task,
  hasVideoError,
  isGenerating,
  deleteConfirmId,
  onPlay,
  onEdit,
  onRegenerate,
  onDelete,
  onDeleteConfirm,
  onVideoError,
  onCreateCharacter,
}: VideoCardProps) {
  const cardRef = useRef<HTMLDivElement>(null);
  const videoRef = useRef<HTMLVideoElement>(null);
  const [shouldLoad, setShouldLoad] = useState(false);
  const loadTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  
  // Observer for lazy loading with delay
  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          loadTimeoutRef.current = setTimeout(() => {
            setShouldLoad(true);
          }, 200);
        } else {
          setShouldLoad(false);
          if (loadTimeoutRef.current) {
            clearTimeout(loadTimeoutRef.current);
            loadTimeoutRef.current = null;
          }
        }
      },
      { rootMargin: '200px' }
    );
    
    if (cardRef.current) {
      observer.observe(cardRef.current);
    }
    
    return () => {
      observer.disconnect();
      if (loadTimeoutRef.current) {
        clearTimeout(loadTimeoutRef.current);
      }
    };
  }, []);
  
  // Observer for auto-play with global limit
  useEffect(() => {
    if (!shouldLoad) return;
    
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (videoRef.current) {
          if (entry.isIntersecting) {
            if (playingVideosSet.size < MAX_PLAYING_VIDEOS) {
              playingVideosSet.add(task.id);
              videoRef.current.play().catch(() => {
                playingVideosSet.delete(task.id);
              });
            }
          } else {
            videoRef.current.pause();
            playingVideosSet.delete(task.id);
          }
        }
      },
      { threshold: 0.5 }
    );
    
    if (cardRef.current) {
      observer.observe(cardRef.current);
    }
    
    return () => {
      observer.disconnect();
      playingVideosSet.delete(task.id);
    };
  }, [shouldLoad, task.id]);
  
  const isProcessing = task.status === 'pending' || task.status === 'processing';
  const isCompleted = task.status === 'completed';
  const isFailed = task.status === 'failed';
  const videoSrc = task.local_path ? getVideoUrl(task.local_path) : null;
  const aspectLabel = task.orientation === 'portrait' ? '9:16' : '16:9';
  
  return (
    <div 
      ref={cardRef}
      onClick={() => isCompleted && !hasVideoError && onPlay(task)}
      className={`relative group overflow-hidden bg-black cursor-pointer transition-all duration-300 hover:opacity-90 ${isProcessing ? 'animate-pulse' : ''}`}
    >
      {/* 视频容器 - 固定 9:16 竖屏比例 */}
      <div className="w-full aspect-[9/16] flex items-center justify-center bg-neutral-900">
        {isCompleted && videoSrc && !hasVideoError && shouldLoad ? (
          <video 
            ref={videoRef}
            src={videoSrc}
            className="w-full h-auto transition-transform duration-700 group-hover:scale-105"
            muted
            loop
            preload="metadata"
            playsInline
            onError={() => onVideoError(task.id)}
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center">
            {isProcessing && (
              <div className="text-center">
                <Loader2 size={28} className="animate-spin text-white/40 mx-auto mb-2" />
                <p className="text-white/50 text-xs">{task.progress}%</p>
              </div>
            )}
            {isFailed && (
              <div className="text-center text-white/40 px-2">
                <X size={28} className="mx-auto mb-2" />
                <p className="text-xs">失败</p>
                {task.fail_reason && (
                  <p className="text-[10px] text-red-400/80 mt-1 line-clamp-3">{task.fail_reason}</p>
                )}
              </div>
            )}
            {isCompleted && hasVideoError && (
              <div className="text-center text-white/40">
                <X size={28} className="mx-auto mb-2" />
                <p className="text-xs">加载失败</p>
              </div>
            )}
            {isCompleted && !hasVideoError && !videoSrc && (
              <div className="text-center text-white/30">
                <Play size={28} className="mx-auto mb-2" />
                <p className="text-xs">文件缺失</p>
              </div>
            )}
          </div>
        )}
      </div>
      
      {/* 悬停遮罩 - 绝对定位撑满 */}
      <div className="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity duration-200 flex flex-col justify-between p-3">
        <div className="flex justify-end gap-1">
          <button onClick={(e) => { e.stopPropagation(); onEdit(task); }} className="w-7 h-7 rounded bg-black/60 hover:bg-white/20 text-white/70 hover:text-white flex items-center justify-center transition-all" title="编辑">
            <Edit3 size={12} />
          </button>
          <button onClick={(e) => { e.stopPropagation(); onRegenerate(task); }} disabled={isGenerating} className="w-7 h-7 rounded bg-black/60 hover:bg-white/20 text-white/70 hover:text-white flex items-center justify-center transition-all disabled:opacity-50" title="重新生成">
            <RefreshCw size={12} className={isGenerating ? 'animate-spin' : ''} />
          </button>
          {isCompleted && task.local_path && (
            <a href={getVideoUrl(task.local_path)} download={`video_${task.id}.mp4`} onClick={(e) => e.stopPropagation()} className="w-7 h-7 rounded bg-black/60 hover:bg-white/20 text-white/70 hover:text-white flex items-center justify-center transition-all" title="下载">
              <Download size={12} />
            </a>
          )}
          {isCompleted && task.model === 'sora-2' && (
            <button onClick={(e) => { e.stopPropagation(); onCreateCharacter(task); }} className="w-7 h-7 rounded bg-black/60 hover:bg-purple-500/80 text-white/70 hover:text-white flex items-center justify-center transition-all" title="创建角色">
              <User size={12} />
            </button>
          )}
          <button onClick={(e) => { e.stopPropagation(); onDeleteConfirm(task.id); }} className="w-7 h-7 rounded bg-black/60 hover:bg-red-500/80 text-white/70 hover:text-white flex items-center justify-center transition-all" title="删除">
            <Trash2 size={12} />
          </button>
        </div>
        
        {isCompleted && (
          <div className="flex-1 flex items-center justify-center">
            <div className="w-12 h-12 rounded-full bg-white/10 backdrop-blur flex items-center justify-center text-white">
              <Play size={20} fill="currentColor" />
            </div>
          </div>
        )}
        
        <div>
          <p className="text-xs text-white/90 line-clamp-2 mb-1">{task.prompt || '图生视频'}</p>
          <div className="flex items-center gap-1 text-[10px] text-white/60">
            <span>{aspectLabel}</span>
            <span>•</span>
            <span>{task.duration}</span>
          </div>
        </div>
      </div>
      
      {/* 删除确认 - 绝对定位撑满 */}
      {deleteConfirmId === task.id && (
        <div className="absolute inset-0 bg-black/90 flex flex-col items-center justify-center p-3 z-10">
          <p className="text-white text-xs mb-3 text-center">删除此视频？</p>
          <div className="flex gap-2">
            <button onClick={(e) => { e.stopPropagation(); onDelete(task.id); }} className="px-3 py-1.5 bg-white text-black text-xs font-medium transition-colors hover:bg-white/90">删除</button>
            <button onClick={(e) => { e.stopPropagation(); onDeleteConfirm(null); }} className="px-3 py-1.5 bg-white/10 text-white text-xs transition-colors hover:bg-white/20">取消</button>
          </div>
        </div>
      )}
    </div>
  );
}, (prevProps, nextProps) => {
  // Custom comparison - only re-render if these specific props change
  return (
    prevProps.task.id === nextProps.task.id &&
    prevProps.task.status === nextProps.task.status &&
    prevProps.task.progress === nextProps.task.progress &&
    prevProps.task.local_path === nextProps.task.local_path &&
    prevProps.task.fail_reason === nextProps.task.fail_reason &&
    prevProps.hasVideoError === nextProps.hasVideoError &&
    prevProps.deleteConfirmId === nextProps.deleteConfirmId &&
    prevProps.isGenerating === nextProps.isGenerating
  );
});

// Polling interval for task status updates (3 seconds as per requirements)
const POLLING_INTERVAL = 3000;
// Page size for pagination
const PAGE_SIZE = 20;

export default function App() {
  const [input, setInput] = useState('');
  const [tasks, setTasks] = useState<Task[]>([]);
  const [isGenerating, setIsGenerating] = useState(false);
  
  // Toast state
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);
  
  // Pagination state
  const [, setTotalTasks] = useState(0);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);

  // Show toast with auto-dismiss after 5 seconds
  const showToast = useCallback((message: string, type: 'success' | 'error') => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 5000);
  }, []);
  
  // Settings State
  const [showSettings, setShowSettings] = useState(false);
  const [duration, setDuration] = useState<Duration>('10s');
  const [orientation, setOrientation] = useState<Orientation>('landscape');
  const [model, setModel] = useState<Model>('sora-2');
  const [count, setCount] = useState<Count>(1);
  
  // Upload State
  const [uploadedImage, setUploadedImage] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  
  // Delete confirmation state
  const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);
  
  // Track video load errors
  const [videoErrors, setVideoErrors] = useState<Set<number>>(new Set());
  
  // Video player modal state
  const [playingTask, setPlayingTask] = useState<Task | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const playerRef = useRef<HTMLVideoElement>(null);
  
  // Character creation dialog state
  const [characterDialogOpen, setCharacterDialogOpen] = useState(false);
  const [characterDialogTaskId, setCharacterDialogTaskId] = useState<string>('');
  const [characterListRefresh, setCharacterListRefresh] = useState(0);
  
  // Scroll refs
  const topRef = useRef<HTMLDivElement>(null);
  const loadMoreRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // Auto-resize textarea based on content
  useEffect(() => {
    const textarea = textareaRef.current;
    if (textarea) {
      textarea.style.height = 'auto';
      const maxHeight = 600;
      const newHeight = Math.min(textarea.scrollHeight, maxHeight);
      textarea.style.height = newHeight + 'px';
      // Show scrollbar when content exceeds max height
      textarea.style.overflowY = textarea.scrollHeight > maxHeight ? 'auto' : 'hidden';
    }
  }, [input]);

  // Fetch initial tasks on mount
  useEffect(() => {
    fetchInitialTasks();
  }, []);

  // Track page visibility for smart polling
  const [isPageVisible, setIsPageVisible] = useState(true);
  
  // Page Visibility API - pause polling when tab is hidden
  useEffect(() => {
    const handleVisibilityChange = () => {
      setIsPageVisible(!document.hidden);
    };
    
    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, []);

  // Poll for pending/processing task status updates by ID
  // Smart polling: only poll when there are pending tasks AND page is visible
  useEffect(() => {
    const pendingTaskIds = tasks
      .filter(t => t.status === 'pending' || t.status === 'processing')
      .map(t => t.id);
    
    // Stop polling if no pending tasks or page is hidden
    if (pendingTaskIds.length === 0 || !isPageVisible) return;

    // Immediate fetch when page becomes visible again
    const fetchUpdates = async () => {
      try {
        const updatedTasks = await getTasksByIds(pendingTaskIds);
        setTasks(prev => prev.map(task => {
          const updated = updatedTasks.find(t => t.id === task.id);
          return updated || task;
        }));
      } catch (err) {
        console.error('Failed to poll task status:', err);
      }
    };

    // Fetch immediately when visibility changes to visible
    fetchUpdates();

    const interval = setInterval(fetchUpdates, POLLING_INTERVAL);

    return () => clearInterval(interval);
  }, [tasks, isPageVisible]);

  // Infinite scroll - load more when reaching bottom
  useEffect(() => {
    if (!loadMoreRef.current || !hasMore) return;
    
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && !isLoadingMore && hasMore) {
          loadMoreTasks();
        }
      },
      { rootMargin: '200px' }
    );
    
    observer.observe(loadMoreRef.current);
    return () => observer.disconnect();
  }, [hasMore, isLoadingMore, tasks.length]);

  // Track if we just created a new task (to scroll to top since newest is first)
  const [shouldScrollToTop, setShouldScrollToTop] = useState(false);
  
  // Scroll to top when a new task is created (newest tasks are at top)
  useEffect(() => {
    if (shouldScrollToTop) {
      topRef.current?.scrollIntoView({ behavior: 'smooth' });
      setShouldScrollToTop(false);
    }
  }, [shouldScrollToTop]);

  const fetchInitialTasks = async () => {
    try {
      const result = await getTasks(PAGE_SIZE, 0);
      setTasks(result.tasks);
      setTotalTasks(result.total || result.tasks.length);
      setHasMore(result.tasks.length >= PAGE_SIZE);
    } catch (err) {
      console.error('Failed to fetch tasks:', err);
      showToast('无法加载任务列表', 'error');
    }
  };

  const loadMoreTasks = async () => {
    if (isLoadingMore || !hasMore) return;
    
    setIsLoadingMore(true);
    try {
      const result = await getTasks(PAGE_SIZE, tasks.length);
      if (result.tasks.length > 0) {
        setTasks(prev => [...prev, ...result.tasks]);
        setHasMore(result.tasks.length >= PAGE_SIZE);
      } else {
        setHasMore(false);
      }
    } catch (err) {
      console.error('Failed to load more tasks:', err);
    } finally {
      setIsLoadingMore(false);
    }
  };


  const handleSendMessage = async () => {
    if ((!input.trim() && !uploadedImage) || isGenerating) return;

    setIsGenerating(true);

    try {
      const request: CreateTaskRequest = {
        prompt: input.trim(),
        image_url: uploadedImage || undefined,
        duration,
        orientation,
        model,
        count
      };

      await createTask(request);
      setInput('');
      setUploadedImage(null);
      showToast('任务提交成功', 'success');
      
      // Refresh tasks to show the new one
      await fetchInitialTasks();
      setShouldScrollToTop(true);
    } catch (err: unknown) {
      console.error('Failed to create task:', err);
      const errorMessage = err instanceof Error ? err.message : '创建任务失败，请重试';
      showToast(errorMessage, 'error');
    } finally {
      setIsGenerating(false);
    }
  };

  const handleDeleteTask = useCallback(async (taskId: number) => {
    // Optimistic update - remove from UI immediately
    const taskToDelete = tasks.find(t => t.id === taskId);
    setTasks(prev => prev.filter(t => t.id !== taskId));
    setDeleteConfirmId(null);
    
    try {
      await deleteTask(taskId);
      showToast('删除成功', 'success');
    } catch (err: unknown) {
      console.error('Failed to delete task:', err);
      // Rollback on error - restore the deleted task
      if (taskToDelete) {
        setTasks(prev => {
          // Insert back in the correct position (sorted by id desc)
          const newTasks = [...prev, taskToDelete];
          return newTasks.sort((a, b) => b.id - a.id);
        });
      }
      const errorMessage = err instanceof Error ? err.message : '删除任务失败';
      showToast(errorMessage, 'error');
    }
  }, [tasks, showToast]);

  // Delete all failed tasks
  const handleDeleteFailedTasks = useCallback(async () => {
    const failedCount = tasks.filter(t => t.status === 'failed').length;
    if (failedCount === 0) {
      showToast('没有失败的任务', 'success');
      return;
    }
    
    try {
      const result = await deleteFailedTasks();
      setTasks(prev => prev.filter(t => t.status !== 'failed'));
      showToast(`已删除 ${result.deleted} 个失败任务`, 'success');
    } catch (err: unknown) {
      console.error('Failed to delete failed tasks:', err);
      const errorMessage = err instanceof Error ? err.message : '删除失败任务出错';
      showToast(errorMessage, 'error');
    }
  }, [tasks, showToast]);

  // Date range delete state
  const [showDateRangeModal, setShowDateRangeModal] = useState(false);
  const [dateRangeStart, setDateRangeStart] = useState('');
  const [dateRangeEnd, setDateRangeEnd] = useState('');
  const [isDeletingByDate, setIsDeletingByDate] = useState(false);

  // Delete tasks by date range
  const handleDeleteByDateRange = useCallback(async () => {
    if (!dateRangeStart || !dateRangeEnd) {
      showToast('请选择开始和结束日期', 'error');
      return;
    }
    
    setIsDeletingByDate(true);
    try {
      const result = await deleteTasksByDateRange(dateRangeStart, dateRangeEnd);
      showToast(result.message, 'success');
      setShowDateRangeModal(false);
      setDateRangeStart('');
      setDateRangeEnd('');
      // Refresh tasks
      await fetchInitialTasks();
    } catch (err: unknown) {
      console.error('Failed to delete tasks by date range:', err);
      const errorMessage = err instanceof Error ? err.message : '按日期删除失败';
      showToast(errorMessage, 'error');
    } finally {
      setIsDeletingByDate(false);
    }
  }, [dateRangeStart, dateRangeEnd, showToast]);

  // Regenerate task with same prompt and image (直接发送)
  const handleRegenerateTask = useCallback(async (task: Task) => {
    if (isGenerating) return;
    
    setIsGenerating(true);

    try {
      const request: CreateTaskRequest = {
        prompt: task.prompt,
        image_url: task.image_url || undefined,
        duration: task.duration,
        orientation: task.orientation,
        model: task.model || 'sora-2',
        count: 1
      };

      await createTask(request);
      showToast('重新生成任务已提交', 'success');
      await fetchInitialTasks();
      setShouldScrollToTop(true);
    } catch (err: unknown) {
      console.error('Failed to regenerate task:', err);
      const errorMessage = err instanceof Error ? err.message : '重新生成失败，请重试';
      showToast(errorMessage, 'error');
    } finally {
      setIsGenerating(false);
    }
  }, [isGenerating, showToast]);

  // Edit task - fill input with previous prompt and image (用户可修改后发送)
  // Need to fetch full task data because list API doesn't include image_url for performance
  const handleEditTask = useCallback(async (task: Task) => {
    setInput(task.prompt || '');
    setDuration(task.duration);
    setOrientation(task.orientation);
    setModel(task.model || 'sora-2');
    // Scroll to show input area at bottom of screen
    window.scrollTo({ top: document.body.scrollHeight, behavior: 'smooth' });
    
    // Fetch full task data to get image_url (not included in list for performance)
    try {
      const fullTask = await getTask(task.id);
      if (fullTask.image_url) {
        setUploadedImage(fullTask.image_url);
      }
    } catch (err) {
      console.error('Failed to fetch task image:', err);
      // Still allow editing without image
    }
  }, []);

  const handleImageUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      processImageFile(file);
    }
  };

  // Process image file (used by upload, drag, and paste)
  const processImageFile = useCallback((file: File) => {
    if (!file.type.startsWith('image/')) {
      showToast('请上传图片文件', 'error');
      return;
    }
    const reader = new FileReader();
    reader.onload = (event) => {
      const result = event.target?.result;
      if (typeof result === 'string') {
        setUploadedImage(result);
      }
    };
    reader.readAsDataURL(file);
  }, [showToast]);

  // Handle drag and drop - use document-level events to prevent flickering
  const [isDragging, setIsDragging] = useState(false);
  const dragTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    const handleDragEnter = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
    };

    const handleDragOver = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      // Clear any pending timeout to hide the overlay
      if (dragTimeoutRef.current) {
        clearTimeout(dragTimeoutRef.current);
        dragTimeoutRef.current = null;
      }
      setIsDragging(true);
    };

    const handleDragLeave = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      // Use timeout to debounce - only hide if no new dragover within 50ms
      if (dragTimeoutRef.current) {
        clearTimeout(dragTimeoutRef.current);
      }
      dragTimeoutRef.current = setTimeout(() => {
        setIsDragging(false);
      }, 50);
    };

    const handleDrop = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (dragTimeoutRef.current) {
        clearTimeout(dragTimeoutRef.current);
        dragTimeoutRef.current = null;
      }
      setIsDragging(false);

      const files = e.dataTransfer?.files;
      if (files && files.length > 0) {
        processImageFile(files[0]);
      }
    };

    // Must prevent default on all drag events to enable drop
    document.addEventListener('dragenter', handleDragEnter);
    document.addEventListener('dragover', handleDragOver);
    document.addEventListener('dragleave', handleDragLeave);
    document.addEventListener('drop', handleDrop);

    return () => {
      document.removeEventListener('dragenter', handleDragEnter);
      document.removeEventListener('dragover', handleDragOver);
      document.removeEventListener('dragleave', handleDragLeave);
      document.removeEventListener('drop', handleDrop);
      if (dragTimeoutRef.current) {
        clearTimeout(dragTimeoutRef.current);
      }
    };
  }, [processImageFile]);

  // Handle paste from clipboard
  useEffect(() => {
    const handlePaste = (e: ClipboardEvent) => {
      const items = e.clipboardData?.items;
      if (!items) return;

      for (let i = 0; i < items.length; i++) {
        if (items[i].type.startsWith('image/')) {
          const file = items[i].getAsFile();
          if (file) {
            processImageFile(file);
            break;
          }
        }
      }
    };

    document.addEventListener('paste', handlePaste);
    return () => document.removeEventListener('paste', handlePaste);
  }, []);

  // Close settings when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      if (showSettings && !target.closest('.settings-container')) {
        setShowSettings(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [showSettings]);

  const getAspectLabel = (task: Task) => {
    return task.orientation === 'portrait' ? '9:16' : '16:9';
  };

  // Video player controls
  const openVideoPlayer = (task: Task) => {
    if (task.status === 'completed' && task.local_path) {
      setPlayingTask(task);
      setIsPlaying(true);
    }
  };

  const closeVideoPlayer = () => {
    // Stop video playback before closing
    if (playerRef.current) {
      playerRef.current.pause();
    }
    setPlayingTask(null);
    setIsPlaying(false);
  };

  const togglePlay = () => {
    if (playerRef.current) {
      if (isPlaying) {
        playerRef.current.pause();
      } else {
        playerRef.current.play();
      }
      setIsPlaying(!isPlaying);
    }
  };

  // Close video player on Escape key
  useEffect(() => {
    if (!playingTask) return;
    
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        closeVideoPlayer();
      }
      if (e.key === ' ') {
        e.preventDefault();
        togglePlay();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [playingTask, isPlaying]);


  // Callbacks for video card
  const handleVideoError = useCallback((taskId: number) => {
    setVideoErrors(prev => new Set(prev).add(taskId));
  }, []);

  // Open character creation dialog
  const handleCreateCharacter = useCallback((task: Task) => {
    if (!task.task_id) {
      showToast('此视频没有有效的任务ID，无法创建角色', 'error');
      return;
    }
    setCharacterDialogTaskId(task.task_id);
    setCharacterDialogOpen(true);
  }, [showToast]);

  // Handle character creation success
  const handleCharacterCreated = useCallback((character: Character) => {
    showToast(`角色 "${character.custom_name}" 创建成功`, 'success');
    setCharacterListRefresh(prev => prev + 1);
  }, [showToast]);

  // Handle character deletion
  const handleCharacterDeleted = useCallback((character: Character) => {
    showToast(`角色 "${character.custom_name}" 已删除`, 'success');
  }, [showToast]);

  const renderVideoCard = (task: Task) => {
    return (
      <VideoCard
        key={task.id}
        task={task}
        hasVideoError={videoErrors.has(task.id)}
        isGenerating={isGenerating}
        deleteConfirmId={deleteConfirmId}
        onPlay={openVideoPlayer}
        onEdit={handleEditTask}
        onRegenerate={handleRegenerateTask}
        onDelete={handleDeleteTask}
        onDeleteConfirm={setDeleteConfirmId}
        onVideoError={handleVideoError}
        onCreateCharacter={handleCreateCharacter}
      />
    );
  };


  return (
    <div 
      className="flex h-screen bg-black text-white font-sans overflow-hidden"
    >
      {/* Drag overlay */}
      {isDragging && (
        <div className="absolute inset-0 bg-purple-500/20 backdrop-blur-sm z-50 flex items-center justify-center border-4 border-dashed border-purple-500 pointer-events-none">
          <div className="text-center">
            <ImageIcon size={64} className="mx-auto mb-4 text-purple-400" />
            <p className="text-xl text-white">松开鼠标上传图片</p>
          </div>
        </div>
      )}

      {/* Video Player Modal */}
      {playingTask && (
        <div 
          className="fixed inset-0 bg-black/90 backdrop-blur-md z-[100] flex items-center justify-center p-4"
          onClick={closeVideoPlayer}
        >
          <div 
            className="relative max-w-4xl w-full max-h-[95vh] flex flex-col items-center"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Close button */}
            <button
              onClick={closeVideoPlayer}
              className="absolute -top-2 -right-2 w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-white transition-colors z-10"
            >
              <X size={24} />
            </button>

            {/* Video container */}
            <div className="relative rounded-2xl overflow-hidden bg-black flex-shrink-0 w-full flex items-center justify-center">
              <video
                ref={playerRef}
                src={getVideoUrl(playingTask.local_path!)}
                className="max-w-full max-h-[75vh] object-contain"
                autoPlay
                controls
                onEnded={() => setIsPlaying(false)}
                onClick={togglePlay}
              />
            </div>

            {/* Video info - scrollable if needed */}
            <div className="mt-3 text-center flex-shrink-0 max-h-[15vh] overflow-y-auto px-2">
              <p className="text-white/90 text-base leading-relaxed">{playingTask.prompt || '图生视频'}</p>
              <p className="text-white/50 text-sm mt-1">
                {getAspectLabel(playingTask)} • {playingTask.duration}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Date Range Delete Modal */}
      {showDateRangeModal && (
        <div 
          className="fixed inset-0 bg-black/80 backdrop-blur-sm z-[100] flex items-center justify-center"
          onClick={() => setShowDateRangeModal(false)}
        >
          <div 
            className="bg-[#1a1a1a] rounded-2xl p-6 w-full max-w-sm mx-4 border border-red-500/30 shadow-xl shadow-red-500/10"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center gap-3 mb-5">
              <div className="w-12 h-12 rounded-full bg-red-500/20 flex items-center justify-center">
                <Calendar size={24} className="text-red-400" />
              </div>
              <div>
                <h3 className="text-white text-lg font-medium">按日期删除</h3>
                <p className="text-white/40 text-xs">批量删除指定时间段的视频</p>
              </div>
            </div>
            
            <div className="space-y-4">
              <div>
                <label className="text-white/80 text-sm mb-2 block font-medium">开始日期</label>
                <input
                  type="date"
                  value={dateRangeStart}
                  max={dateRangeEnd || undefined}
                  onChange={(e) => setDateRangeStart(e.target.value)}
                  className="w-full bg-white/10 border-2 border-white/20 rounded-xl px-4 py-3.5 text-white text-base focus:outline-none focus:border-red-500/50 [color-scheme:dark] cursor-pointer hover:bg-white/15 transition-all"
                />
              </div>
              
              <div>
                <label className="text-white/80 text-sm mb-2 block font-medium">结束日期</label>
                <input
                  type="date"
                  value={dateRangeEnd}
                  min={dateRangeStart || undefined}
                  onChange={(e) => setDateRangeEnd(e.target.value)}
                  className="w-full bg-white/10 border-2 border-white/20 rounded-xl px-4 py-3.5 text-white text-base focus:outline-none focus:border-red-500/50 [color-scheme:dark] cursor-pointer hover:bg-white/15 transition-all"
                />
              </div>
              
              {/* Date validation warning */}
              {dateRangeStart && dateRangeEnd && dateRangeStart > dateRangeEnd && (
                <p className="text-red-400 text-sm bg-red-500/10 px-3 py-2 rounded-lg">⚠️ 开始日期不能大于结束日期</p>
              )}
            </div>
            
            <div className="flex gap-3 mt-6">
              <button
                onClick={() => {
                  setShowDateRangeModal(false);
                  setDateRangeStart('');
                  setDateRangeEnd('');
                }}
                className="flex-1 px-4 py-3 text-sm text-white/70 hover:text-white bg-white/10 hover:bg-white/15 rounded-xl transition-all font-medium"
              >
                取消
              </button>
              <button
                onClick={handleDeleteByDateRange}
                disabled={isDeletingByDate || !dateRangeStart || !dateRangeEnd || dateRangeStart > dateRangeEnd}
                className="flex-1 px-4 py-3 text-sm text-white bg-red-500 hover:bg-red-600 rounded-xl transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2 font-medium"
              >
                {isDeletingByDate ? <Loader2 size={16} className="animate-spin" /> : <Trash2 size={16} />}
                确认删除
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Character Creation Dialog */}
      <CharacterCreationDialog
        isOpen={characterDialogOpen}
        onClose={() => setCharacterDialogOpen(false)}
        taskId={characterDialogTaskId}
        onSuccess={handleCharacterCreated}
        onError={(message) => showToast(message, 'error')}
      />

      {/* Sidebar - Character List */}
      <div className="w-64 flex-shrink-0 border-r border-white/5 overflow-y-auto p-3" style={{ scrollbarWidth: 'none' }}>
        <CharacterList
          onCharacterDeleted={handleCharacterDeleted}
          onError={(message) => showToast(message, 'error')}
          onSuccess={(message) => showToast(message, 'success')}
          refreshTrigger={characterListRefresh}
        />
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col relative h-full z-10">
        {/* Toast notification */}
        {toast && (
          <div className={`absolute top-6 left-1/2 -translate-x-1/2 px-4 py-2 rounded-lg shadow-lg z-50 flex items-center gap-2 ${
            toast.type === 'success' ? 'bg-green-500/90' : 'bg-red-500/90'
          } text-white`}>
            {toast.type === 'success' ? <Check size={16} /> : <X size={16} />}
            {toast.message}
            <button onClick={() => setToast(null)} className="ml-2 hover:text-white/80">
              <X size={14} />
            </button>
          </div>
        )}

        {/* Content - Video Gallery */}
        <main 
          ref={scrollContainerRef}
          className="flex-1 overflow-y-auto pb-32" 
          style={{ scrollbarWidth: 'none' }}
        >
          <div ref={topRef} />
          
          {/* Toolbar */}
          {tasks.length > 0 && (
            <div className="sticky top-0 z-20 bg-black/90 backdrop-blur-md px-4 py-2.5 flex justify-end gap-3 border-b border-white/5">
              {/* Delete by date range button */}
              <button
                onClick={() => setShowDateRangeModal(true)}
                className="flex items-center gap-2 px-4 py-2 text-sm text-orange-400 hover:text-orange-300 bg-orange-500/10 hover:bg-orange-500/20 rounded-lg transition-all border border-orange-500/20"
              >
                <Calendar size={16} />
                按日期删除
              </button>
              
              {/* Delete failed tasks button */}
              {tasks.some(t => t.status === 'failed') && (
                <button
                  onClick={handleDeleteFailedTasks}
                  className="flex items-center gap-2 px-4 py-2 text-sm text-red-400 hover:text-red-300 bg-red-500/10 hover:bg-red-500/20 rounded-lg transition-all border border-red-500/20"
                >
                  <Trash2 size={16} />
                  删除失败 ({tasks.filter(t => t.status === 'failed').length})
                </button>
              )}
            </div>
          )}
          
          {tasks.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-white/30">
              <Play size={48} className="mb-4" />
              <p>开始创作你的第一个视频</p>
            </div>
          ) : (
            <>
              {/* Video grid - Fixed 4 columns */}
              <div className="grid grid-cols-4 gap-[2px] bg-black">
                {tasks.map(task => (
                  <div key={task.id}>
                    {renderVideoCard(task)}
                  </div>
                ))}
              </div>
              {/* Load more trigger */}
              <div ref={loadMoreRef} className="h-20 flex items-center justify-center">
                {isLoadingMore && (
                  <Loader2 size={24} className="animate-spin text-white/30" />
                )}
              </div>
            </>
          )}
        </main>


        {/* Input Area - Absolute bottom, centered with limited width */}
        <div className="absolute bottom-0 left-0 right-0 z-30 flex justify-center pb-6 px-4 pointer-events-none">
          <div className="w-full max-w-2xl relative pointer-events-auto">

            {/* Uploaded Image Preview - above the input box */}
            {uploadedImage && (
              <div className="mb-3 flex justify-start gap-3">
                <div className="relative inline-block">
                  <img src={uploadedImage} alt="参考图" className="h-20 w-auto rounded-lg object-cover" />
                  <button 
                    onClick={() => setUploadedImage(null)}
                    className="absolute -top-2 -right-2 bg-black/80 hover:bg-red-500 text-white rounded-full p-1 transition-colors"
                  >
                    <X size={12} />
                  </button>
                </div>
              </div>
            )}

            {/* Main input container - glass effect rounded box */}
            <div className={`backdrop-blur-xl bg-[#2a2a2a]/80 rounded-2xl border border-white/5 ${isGenerating ? 'opacity-50 pointer-events-none' : ''}`}>
              {/* Text input */}
              <div className="px-4 pt-3 pb-2">
                <textarea
                  ref={textareaRef}
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault();
                      handleSendMessage();
                    }
                  }}
                  placeholder="描述你想要生成的视频..."
                  className="w-full bg-transparent border-none outline-none text-sm text-white placeholder-white/40 resize-none"
                  style={{ minHeight: '24px', maxHeight: '600px' }}
                  rows={1}
                />
              </div>
              
              {/* Bottom toolbar - inside the same container */}
              <div className="flex items-center justify-between px-3 pb-3">
                <div className="flex items-center gap-1">
                  <input 
                    type="file" 
                    ref={fileInputRef} 
                    className="hidden" 
                    accept="image/*"
                    onChange={handleImageUpload}
                  />
                  
                  {/* Add image button - both sora-2 and sora-2-alt support reference images */}
                  <button 
                    onClick={() => fileInputRef.current?.click()}
                    className={`h-8 px-3 flex items-center gap-2 rounded-full text-xs transition-all ${uploadedImage ? 'bg-white/10 text-white' : 'bg-white/5 text-white/60 hover:text-white hover:bg-white/10'}`}
                    title="上传参考图"
                  >
                    <Plus size={14} />
                    <span>图片</span>
                  </button>
                </div>
                
                <div className="flex items-center gap-2">
                  {/* Current settings display */}
                  <div className="flex items-center gap-1.5 text-[11px] text-white/50">
                    <span>{orientation === 'portrait' ? '9:16' : '16:9'}</span>
                    <span>•</span>
                    <span>{duration}</span>
                    <span>•</span>
                    <span>×{count}</span>
                  </div>
                  
                  {/* Settings button with popover */}
                  <div className="settings-container relative">
                    <button 
                      type="button"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        setShowSettings(!showSettings);
                      }}
                      className={`w-8 h-8 flex items-center justify-center rounded-full transition-all ${showSettings ? 'bg-white/10 text-white' : 'text-white/50 hover:text-white hover:bg-white/5'}`}
                    >
                      <Settings size={16} />
                    </button>
                    
                    {/* Settings Popover - positioned relative to settings button */}
                    {showSettings && (
                      <div className="absolute bottom-full right-0 mb-2 w-72 backdrop-blur-xl bg-[#2a2a2a] rounded-xl border border-white/5 shadow-2xl overflow-hidden z-50">
                        <div className="p-3 space-y-3">
                          <div>
                            <div className="text-[10px] text-white/40 font-medium mb-2 uppercase tracking-wider">画面比例</div>
                            <div className="grid grid-cols-2 gap-1.5">
                              {(['portrait', 'landscape'] as Orientation[]).map(o => (
                                <button
                                  key={o}
                                  onClick={() => setOrientation(o)}
                                  className={`flex items-center justify-center gap-1.5 py-2.5 text-xs rounded-lg transition-all ${orientation === o ? 'bg-white text-black font-medium' : 'bg-white/5 text-white/60 hover:bg-white/10'}`}
                                >
                                  {o === 'portrait' ? <Smartphone size={12} /> : <Monitor size={12} />}
                                  {o === 'portrait' ? '9:16' : '16:9'}
                                </button>
                              ))}
                            </div>
                          </div>
                          <div>
                            <div className="text-[10px] text-white/40 font-medium mb-2 uppercase tracking-wider">时长</div>
                            <div className="grid grid-cols-2 gap-1.5">
                              {(['10s', '15s'] as Duration[]).map(d => (
                                <button
                                  key={d}
                                  onClick={() => setDuration(d)}
                                  className={`py-2.5 text-xs rounded-lg transition-all ${duration === d ? 'bg-white text-black font-medium' : 'bg-white/5 text-white/60 hover:bg-white/10'}`}
                                >
                                  {d}
                                </button>
                              ))}
                            </div>
                          </div>
                          <div>
                            <div className="text-[10px] text-white/40 font-medium mb-2 uppercase tracking-wider">数量</div>
                            <div className="grid grid-cols-3 gap-1.5">
                              {([1, 2, 4] as Count[]).map(c => (
                                <button
                                  key={c}
                                  onClick={() => setCount(c)}
                                  className={`py-2.5 text-xs rounded-lg transition-all ${count === c ? 'bg-white text-black font-medium' : 'bg-white/5 text-white/60 hover:bg-white/10'}`}
                                >
                                  {c}
                                </button>
                              ))}
                            </div>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>

                  {/* Send button */}
                  <button 
                    onClick={handleSendMessage}
                    disabled={(!input.trim() && !uploadedImage) || isGenerating}
                    className={`w-8 h-8 flex items-center justify-center rounded-full transition-all
                      ${(input.trim() || uploadedImage) && !isGenerating
                        ? 'bg-white text-black hover:bg-white/90' 
                        : 'bg-white/10 text-white/30 cursor-not-allowed'}`}
                  >
                    {isGenerating ? <Loader2 size={16} className="animate-spin" /> : <ArrowUp size={16} />}
                  </button>
                </div>
              </div>
            </div>
            
          </div>
        </div>
      </div>
    </div>
  );
}

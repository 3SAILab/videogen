import React, { useState, useRef, useEffect } from 'react';
import { X, Loader2, User, Link, Video } from 'lucide-react';
import { createCharacter } from './api';
import type { Character, CharacterSourceType } from './types';

interface CharacterCreationDialogProps {
  isOpen: boolean;
  onClose: () => void;
  taskId?: string; // Optional - pre-fill when creating from a task
  onSuccess: (character: Character) => void;
  onError: (message: string) => void;
}

interface FormErrors {
  customName?: string;
  description?: string;
  timestamps?: string;
  sourceValue?: string;
}

export default function CharacterCreationDialog({
  isOpen,
  onClose,
  taskId,
  onSuccess,
  onError,
}: CharacterCreationDialogProps) {
  const [customName, setCustomName] = useState('');
  const [description, setDescription] = useState('');
  const [startSeconds, setStartSeconds] = useState('0');
  const [endSeconds, setEndSeconds] = useState('2');
  const [sourceType, setSourceType] = useState<CharacterSourceType>('task');
  const [sourceValue, setSourceValue] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errors, setErrors] = useState<FormErrors>({});
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Pre-fill source value when taskId is provided
  useEffect(() => {
    if (taskId) {
      setSourceType('task');
      setSourceValue(taskId);
    }
  }, [taskId]);

  // Auto-resize textarea based on content
  const adjustTextareaHeight = () => {
    const textarea = textareaRef.current;
    if (textarea) {
      textarea.style.height = 'auto';
      const maxHeight = 200;
      textarea.style.height = `${Math.min(textarea.scrollHeight, maxHeight)}px`;
      textarea.style.overflowY = textarea.scrollHeight > maxHeight ? 'auto' : 'hidden';
    }
  };

  useEffect(() => {
    adjustTextareaHeight();
  }, [description]);


  // Validation functions
  const validateCustomName = (name: string): string | undefined => {
    if (name.length < 1) return '名称不能为空';
    if (name.length > 10) return '名称不能超过10个字符';
    return undefined;
  };

  const validateDescription = (desc: string): string | undefined => {
    if (desc.length < 1) return '描述不能为空';
    if (desc.length > 500) return '描述不能超过500个字符';
    return undefined;
  };

  const validateTimestamps = (start: string, end: string): string | undefined => {
    const startNum = parseFloat(start);
    const endNum = parseFloat(end);
    
    if (isNaN(startNum) || isNaN(endNum)) return '请输入有效的数字';
    if (startNum < 0) return '起始时间不能小于0';
    
    const range = endNum - startNum;
    if (range < 1) return '时间范围至少1秒';
    if (range > 3) return '时间范围不能超过3秒';
    
    return undefined;
  };

  const validateSourceValue = (type: CharacterSourceType, value: string): string | undefined => {
    if (!value || value.trim().length === 0) {
      return type === 'task' ? '请输入任务ID' : '请输入视频URL';
    }
    
    if (type === 'url') {
      // Basic URL validation
      try {
        new URL(value);
      } catch {
        return '请输入有效的URL地址';
      }
    }
    
    return undefined;
  };

  // Validate all fields
  const validateForm = (): boolean => {
    const newErrors: FormErrors = {};
    
    const nameError = validateCustomName(customName);
    if (nameError) newErrors.customName = nameError;
    
    const descError = validateDescription(description);
    if (descError) newErrors.description = descError;
    
    const timestampError = validateTimestamps(startSeconds, endSeconds);
    if (timestampError) newErrors.timestamps = timestampError;
    
    const sourceError = validateSourceValue(sourceType, sourceValue);
    if (sourceError) newErrors.sourceValue = sourceError;
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) return;
    
    setIsSubmitting(true);
    
    try {
      const timestamps = `${startSeconds},${endSeconds}`;
      const character = await createCharacter({
        custom_name: customName,
        description,
        source_type: sourceType,
        source_value: sourceValue.trim(),
        timestamps,
      });
      
      onSuccess(character);
      handleClose();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : '创建角色失败';
      onError(errorMessage);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Reset form and close
  const handleClose = () => {
    setCustomName('');
    setDescription('');
    setStartSeconds('0');
    setEndSeconds('2');
    setSourceType('task');
    setSourceValue(taskId || '');
    setErrors({});
    onClose();
  };

  // Handle source type change
  const handleSourceTypeChange = (type: CharacterSourceType) => {
    setSourceType(type);
    // Clear source value when switching types, unless it's task and we have a taskId
    if (type === 'task' && taskId) {
      setSourceValue(taskId);
    } else {
      setSourceValue('');
    }
    // Clear any source value errors
    if (errors.sourceValue) {
      setErrors(prev => ({ ...prev, sourceValue: undefined }));
    }
  };

  if (!isOpen) return null;


  return (
    <div 
      className="fixed inset-0 bg-black/80 backdrop-blur-sm z-[100] flex items-center justify-center"
      onClick={handleClose}
    >
      <div 
        className="bg-[#2a2a2a] rounded-2xl p-6 w-full max-w-md mx-4 border border-white/10"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-purple-500/20 flex items-center justify-center">
              <User size={20} className="text-purple-400" />
            </div>
            <h3 className="text-white text-lg font-medium">创建角色</h3>
          </div>
          <button
            onClick={handleClose}
            className="w-8 h-8 rounded-full hover:bg-white/10 flex items-center justify-center text-white/60 hover:text-white transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Custom Name Input */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-white/60 text-xs">角色名称</label>
              <span className={`text-xs ${customName.length > 10 ? 'text-red-400' : 'text-white/40'}`}>
                {customName.length}/10
              </span>
            </div>
            <input
              type="text"
              value={customName}
              onChange={(e) => {
                setCustomName(e.target.value);
                if (errors.customName) {
                  setErrors(prev => ({ ...prev, customName: undefined }));
                }
              }}
              maxLength={10}
              placeholder="输入角色名称"
              className={`w-full bg-black/30 border rounded-lg px-4 py-3 text-white text-sm focus:outline-none transition-colors ${
                errors.customName ? 'border-red-500/50 focus:border-red-500' : 'border-white/10 focus:border-white/30'
              }`}
            />
            {errors.customName && (
              <p className="text-red-400 text-xs mt-1">{errors.customName}</p>
            )}
          </div>

          {/* Description Textarea */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-white/60 text-xs">人物描述</label>
              <span className={`text-xs ${description.length > 500 ? 'text-red-400' : 'text-white/40'}`}>
                {description.length}/500
              </span>
            </div>
            <textarea
              ref={textareaRef}
              value={description}
              onChange={(e) => {
                setDescription(e.target.value);
                if (errors.description) {
                  setErrors(prev => ({ ...prev, description: undefined }));
                }
              }}
              maxLength={500}
              placeholder="描述角色的外观特征..."
              rows={2}
              className={`w-full bg-black/30 border rounded-lg px-4 py-3 text-white text-sm focus:outline-none resize-none transition-colors overflow-hidden ${
                errors.description ? 'border-red-500/50 focus:border-red-500' : 'border-white/10 focus:border-white/30'
              }`}
              style={{ minHeight: '60px', maxHeight: '200px' }}
            />
            {errors.description && (
              <p className="text-red-400 text-xs mt-1">{errors.description}</p>
            )}
          </div>


          {/* Source Type Selector */}
          <div>
            <label className="text-white/60 text-xs mb-2 block">视频来源</label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => handleSourceTypeChange('task')}
                className={`flex-1 flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg text-sm transition-all ${
                  sourceType === 'task'
                    ? 'bg-purple-500/20 text-purple-400 border border-purple-500/50'
                    : 'bg-black/30 text-white/60 border border-white/10 hover:border-white/20'
                }`}
              >
                <Video size={16} />
                任务ID
              </button>
              <button
                type="button"
                onClick={() => handleSourceTypeChange('url')}
                className={`flex-1 flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg text-sm transition-all ${
                  sourceType === 'url'
                    ? 'bg-purple-500/20 text-purple-400 border border-purple-500/50'
                    : 'bg-black/30 text-white/60 border border-white/10 hover:border-white/20'
                }`}
              >
                <Link size={16} />
                视频URL
              </button>
            </div>
          </div>

          {/* Source Value Input */}
          <div>
            <label className="text-white/60 text-xs mb-2 block">
              {sourceType === 'task' ? '任务ID' : '视频URL'}
            </label>
            <input
              type={sourceType === 'url' ? 'url' : 'text'}
              value={sourceValue}
              onChange={(e) => {
                setSourceValue(e.target.value);
                if (errors.sourceValue) {
                  setErrors(prev => ({ ...prev, sourceValue: undefined }));
                }
              }}
              placeholder={sourceType === 'task' ? '输入已生成视频的任务ID' : '输入视频URL地址'}
              className={`w-full bg-black/30 border rounded-lg px-4 py-3 text-white text-sm focus:outline-none transition-colors ${
                errors.sourceValue ? 'border-red-500/50 focus:border-red-500' : 'border-white/10 focus:border-white/30'
              }`}
            />
            {errors.sourceValue && (
              <p className="text-red-400 text-xs mt-1">{errors.sourceValue}</p>
            )}
            <p className="text-white/40 text-xs mt-1">
              {sourceType === 'task' 
                ? '从已生成的视频任务中提取角色' 
                : '从外部视频URL中提取角色'}
            </p>
          </div>

          {/* Timestamps */}
          <div>
            <label className="text-white/60 text-xs mb-2 block">视频时间范围（秒）</label>
            <p className="text-white/40 text-xs mb-2">选择角色出现的时间段，范围1-3秒</p>
            <div className="flex items-center gap-3">
              <div className="flex-1">
                <input
                  type="number"
                  value={startSeconds}
                  onChange={(e) => {
                    setStartSeconds(e.target.value);
                    if (errors.timestamps) {
                      setErrors(prev => ({ ...prev, timestamps: undefined }));
                    }
                  }}
                  min="0"
                  step="0.1"
                  placeholder="起始"
                  className={`w-full bg-black/30 border rounded-lg px-4 py-3 text-white text-sm focus:outline-none transition-colors ${
                    errors.timestamps ? 'border-red-500/50 focus:border-red-500' : 'border-white/10 focus:border-white/30'
                  }`}
                />
              </div>
              <span className="text-white/40">至</span>
              <div className="flex-1">
                <input
                  type="number"
                  value={endSeconds}
                  onChange={(e) => {
                    setEndSeconds(e.target.value);
                    if (errors.timestamps) {
                      setErrors(prev => ({ ...prev, timestamps: undefined }));
                    }
                  }}
                  min="0"
                  step="0.1"
                  placeholder="结束"
                  className={`w-full bg-black/30 border rounded-lg px-4 py-3 text-white text-sm focus:outline-none transition-colors ${
                    errors.timestamps ? 'border-red-500/50 focus:border-red-500' : 'border-white/10 focus:border-white/30'
                  }`}
                />
              </div>
            </div>
            {errors.timestamps && (
              <p className="text-red-400 text-xs mt-1">{errors.timestamps}</p>
            )}
          </div>


          {/* Buttons */}
          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={handleClose}
              disabled={isSubmitting}
              className="flex-1 px-4 py-2.5 text-sm text-white/60 hover:text-white bg-white/5 hover:bg-white/10 rounded-lg transition-all disabled:opacity-50"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="flex-1 px-4 py-2.5 text-sm text-white bg-purple-500 hover:bg-purple-600 rounded-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            >
              {isSubmitting ? (
                <>
                  <Loader2 size={14} className="animate-spin" />
                  创建中...
                </>
              ) : (
                '创建角色'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

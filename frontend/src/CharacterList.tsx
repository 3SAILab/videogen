import React, { useState, useEffect, useCallback, useRef } from 'react';
import { User, Trash2, Loader2, ChevronDown, ChevronUp, Search, X, Copy, Clock, CheckCircle, XCircle, RefreshCw } from 'lucide-react';
import { getCharacters, deleteCharacter, getCharacterStatus } from './api';
import type { Character, CharacterStatus } from './types';

interface CharacterListProps {
  onCharacterDeleted?: (character: Character) => void;
  onError?: (message: string) => void;
  onSuccess?: (message: string) => void;
  refreshTrigger?: number;
}

// Status badge component
function StatusBadge({ status, progress }: { status: CharacterStatus; progress: number }) {
  const statusConfig = {
    pending: { icon: Clock, color: 'text-yellow-400', bg: 'bg-yellow-400/10', label: '等待中' },
    processing: { icon: RefreshCw, color: 'text-blue-400', bg: 'bg-blue-400/10', label: '训练中' },
    completed: { icon: CheckCircle, color: 'text-green-400', bg: 'bg-green-400/10', label: '已完成' },
    failed: { icon: XCircle, color: 'text-red-400', bg: 'bg-red-400/10', label: '失败' },
  };

  const config = statusConfig[status] || statusConfig.pending;
  const Icon = config.icon;

  return (
    <div className={`flex items-center gap-1.5 px-2 py-0.5 rounded-full ${config.bg}`}>
      <Icon size={12} className={`${config.color} ${status === 'processing' ? 'animate-spin' : ''}`} />
      <span className={`text-xs ${config.color}`}>
        {status === 'processing' ? `${progress}%` : config.label}
      </span>
    </div>
  );
}

// Progress bar component
function ProgressBar({ progress }: { progress: number }) {
  return (
    <div className="w-full h-1 bg-white/10 rounded-full overflow-hidden">
      <div 
        className="h-full bg-blue-400 transition-all duration-300"
        style={{ width: `${progress}%` }}
      />
    </div>
  );
}

export default function CharacterList({
  onCharacterDeleted,
  onError,
  onSuccess,
  refreshTrigger = 0,
}: CharacterListProps) {
  const [characters, setCharacters] = useState<Character[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isExpanded, setIsExpanded] = useState(true);
  const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);
  const [isDeletingId, setIsDeletingId] = useState<number | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCharacter, setSelectedCharacter] = useState<Character | null>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const pollingIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Fetch characters on mount and when refreshTrigger changes
  const fetchCharacters = useCallback(async () => {
    try {
      setIsLoading(true);
      const data = await getCharacters();
      setCharacters(data);
    } catch (err) {
      console.error('Failed to fetch characters:', err);
      onError?.('无法加载角色列表');
    } finally {
      setIsLoading(false);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    fetchCharacters();
  }, [fetchCharacters, refreshTrigger]);

  // Poll status for non-completed characters
  const pollCharacterStatuses = useCallback(async () => {
    const nonCompletedCharacters = characters.filter(
      c => c.status === 'pending' || c.status === 'processing'
    );

    if (nonCompletedCharacters.length === 0) return;

    for (const character of nonCompletedCharacters) {
      try {
        const statusResponse = await getCharacterStatus(character.id);
        setCharacters(prev => prev.map(c => 
          c.id === character.id 
            ? { ...c, status: statusResponse.status, progress: statusResponse.progress, api_character_id: statusResponse.api_character_id, username: statusResponse.username, avatar_url: statusResponse.avatar_url, fail_reason: statusResponse.fail_reason }
            : c
        ));
        // Also update selected character if it's the one being polled
        setSelectedCharacter(prev => 
          prev?.id === character.id 
            ? { ...prev, status: statusResponse.status, progress: statusResponse.progress, api_character_id: statusResponse.api_character_id, username: statusResponse.username, avatar_url: statusResponse.avatar_url, fail_reason: statusResponse.fail_reason }
            : prev
        );
      } catch (err) {
        console.error(`Failed to poll status for character ${character.id}:`, err);
      }
    }
  }, [characters]);

  // Set up polling interval
  useEffect(() => {
    const hasNonCompleted = characters.some(
      c => c.status === 'pending' || c.status === 'processing'
    );

    if (hasNonCompleted) {
      pollingIntervalRef.current = setInterval(pollCharacterStatuses, 5000);
    }

    return () => {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
        pollingIntervalRef.current = null;
      }
    };
  }, [characters, pollCharacterStatuses]);

  // Filter characters based on search query
  const filteredCharacters = characters.filter(char => {
    if (!searchQuery.trim()) return true;
    const query = searchQuery.toLowerCase();
    return (
      char.custom_name.toLowerCase().includes(query) ||
      char.api_character_id?.toLowerCase().includes(query) ||
      char.description?.toLowerCase().includes(query)
    );
  });

  // Handle delete confirmation
  const handleDeleteClick = (e: React.MouseEvent, characterId: number) => {
    e.stopPropagation();
    setDeleteConfirmId(characterId);
  };

  // Handle actual delete
  const handleConfirmDelete = async (e: React.MouseEvent, character: Character) => {
    e.stopPropagation();
    setIsDeletingId(character.id);
    
    try {
      await deleteCharacter(character.id);
      setCharacters(prev => prev.filter(c => c.id !== character.id));
      setSelectedCharacter(null);
      onCharacterDeleted?.(character);
    } catch (err) {
      console.error('Failed to delete character:', err);
      const errorMessage = err instanceof Error ? err.message : '删除角色失败';
      onError?.(errorMessage);
    } finally {
      setIsDeletingId(null);
      setDeleteConfirmId(null);
    }
  };

  // Cancel delete
  const handleCancelDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    setDeleteConfirmId(null);
  };

  // Copy to clipboard
  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text).then(() => {
      onSuccess?.(`${label} 已复制`);
    }).catch(() => {
      onError?.('复制失败');
    });
  };

  if (isLoading) {
    return (
      <div className="bg-[#1a1a1a] rounded-xl border border-white/5 p-4">
        <div className="flex items-center justify-center py-4">
          <Loader2 size={20} className="animate-spin text-white/40" />
        </div>
      </div>
    );
  }

  return (
    <>
      <div className="bg-[#1a1a1a] rounded-xl border border-white/5 overflow-hidden">
        {/* Header */}
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="w-full flex items-center justify-between px-4 py-3 hover:bg-white/5 transition-colors"
        >
          <div className="flex items-center gap-2">
            <User size={16} className="text-purple-400" />
            <span className="text-sm text-white/80">角色</span>
            <span className="text-xs text-white/40">({characters.length})</span>
          </div>
          {isExpanded ? (
            <ChevronUp size={16} className="text-white/40" />
          ) : (
            <ChevronDown size={16} className="text-white/40" />
          )}
        </button>

        {/* Search and Character list */}
        {isExpanded && (
          <div className="px-2 pb-2">
            {/* Search input */}
            {characters.length > 0 && (
              <div className="relative mb-2">
                <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-white/30" />
                <input
                  ref={searchInputRef}
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="搜索角色..."
                  className="w-full bg-black/30 border border-white/10 rounded-lg pl-9 pr-8 py-2 text-xs text-white placeholder-white/30 focus:outline-none focus:border-white/20"
                />
                {searchQuery && (
                  <button
                    onClick={() => setSearchQuery('')}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-white/30 hover:text-white/60"
                  >
                    <X size={14} />
                  </button>
                )}
              </div>
            )}

            {characters.length === 0 ? (
              <div className="text-center py-6 text-white/30 text-xs">
                暂无角色
              </div>
            ) : filteredCharacters.length === 0 ? (
              <div className="text-center py-6 text-white/30 text-xs">
                未找到匹配的角色
              </div>
            ) : (
              <div className="space-y-1 max-h-[400px] overflow-y-auto" style={{ scrollbarWidth: 'thin' }}>
                {filteredCharacters.map(character => (
                  <div
                    key={character.id}
                    onClick={() => setSelectedCharacter(character)}
                    className="relative group flex items-center gap-3 p-2 rounded-lg hover:bg-white/5 transition-colors cursor-pointer"
                  >
                    {/* Character avatar */}
                    <div className="w-10 h-10 rounded-full overflow-hidden bg-white/10 flex-shrink-0 flex items-center justify-center">
                      {character.avatar_url ? (
                        <img src={character.avatar_url} alt={character.custom_name} className="w-full h-full object-cover" />
                      ) : (
                        <User size={16} className="text-white/40" />
                      )}
                    </div>

                    {/* Names and status */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <p className="text-sm text-white truncate">{character.custom_name}</p>
                        <StatusBadge status={character.status} progress={character.progress} />
                      </div>
                      {character.status === 'processing' && (
                        <div className="mt-1">
                          <ProgressBar progress={character.progress} />
                        </div>
                      )}
                      {character.status === 'completed' && (character.username || character.api_character_id) && (
                        <p className="text-xs text-white/40 truncate">@{character.username || character.api_character_id}</p>
                      )}
                      {character.status === 'failed' && character.fail_reason && (
                        <p className="text-xs text-red-400/80 truncate">{character.fail_reason}</p>
                      )}
                    </div>

                    {/* Delete button - visible on hover */}
                    <button
                      onClick={(e) => handleDeleteClick(e, character.id)}
                      className="opacity-0 group-hover:opacity-100 w-7 h-7 rounded-full hover:bg-red-500/20 flex items-center justify-center text-white/40 hover:text-red-400 transition-all"
                      title="删除角色"
                    >
                      <Trash2 size={14} />
                    </button>

                    {/* Delete confirmation overlay */}
                    {deleteConfirmId === character.id && (
                      <div className="absolute inset-0 bg-[#1a1a1a]/95 rounded-lg flex items-center justify-center gap-2 z-10">
                        <span className="text-xs text-white/60 mr-2">删除?</span>
                        <button
                          onClick={(e) => handleConfirmDelete(e, character)}
                          disabled={isDeletingId === character.id}
                          className="px-2 py-1 text-xs bg-red-500 hover:bg-red-600 text-white rounded transition-colors disabled:opacity-50 flex items-center gap-1"
                        >
                          {isDeletingId === character.id ? (
                            <Loader2 size={12} className="animate-spin" />
                          ) : (
                            '确认'
                          )}
                        </button>
                        <button
                          onClick={handleCancelDelete}
                          className="px-2 py-1 text-xs bg-white/10 hover:bg-white/20 text-white/60 rounded transition-colors"
                        >
                          取消
                        </button>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Character Detail Modal */}
      {selectedCharacter && (
        <div 
          className="fixed inset-0 bg-black/80 backdrop-blur-sm z-[100] flex items-center justify-center"
          onClick={() => setSelectedCharacter(null)}
        >
          <div 
            className="bg-[#2a2a2a] rounded-2xl p-6 w-full max-w-sm mx-4 border border-white/10"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Header */}
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center gap-4">
                <div className="w-16 h-16 rounded-full overflow-hidden bg-white/10 flex-shrink-0 flex items-center justify-center">
                  {selectedCharacter.avatar_url ? (
                    <img src={selectedCharacter.avatar_url} alt={selectedCharacter.custom_name} className="w-full h-full object-cover" />
                  ) : (
                    <User size={24} className="text-white/40" />
                  )}
                </div>
                <div>
                  <h3 className="text-white text-lg font-medium">{selectedCharacter.custom_name}</h3>
                  <StatusBadge status={selectedCharacter.status} progress={selectedCharacter.progress} />
                </div>
              </div>
              <button
                onClick={() => setSelectedCharacter(null)}
                className="w-8 h-8 rounded-full hover:bg-white/10 flex items-center justify-center text-white/60 hover:text-white transition-colors"
              >
                <X size={18} />
              </button>
            </div>

            {/* Progress bar for processing */}
            {selectedCharacter.status === 'processing' && (
              <div className="mb-4">
                <div className="flex justify-between text-xs text-white/60 mb-1">
                  <span>训练进度</span>
                  <span>{selectedCharacter.progress}%</span>
                </div>
                <div className="w-full h-2 bg-white/10 rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-blue-400 transition-all duration-300"
                    style={{ width: `${selectedCharacter.progress}%` }}
                  />
                </div>
              </div>
            )}

            {/* Fail reason */}
            {selectedCharacter.status === 'failed' && selectedCharacter.fail_reason && (
              <div className="mb-4 p-3 bg-red-500/10 border border-red-500/20 rounded-lg">
                <label className="text-red-400 text-xs mb-1 block">失败原因</label>
                <p className="text-white/80 text-sm">{selectedCharacter.fail_reason}</p>
              </div>
            )}

            {/* Description */}
            {selectedCharacter.description && (
              <div className="mb-4">
                <label className="text-white/40 text-xs mb-1 block">描述</label>
                <p className="text-white/80 text-sm bg-black/20 rounded-lg p-3">
                  {selectedCharacter.description}
                </p>
              </div>
            )}

            {/* Details */}
            <div className="space-y-3">
              {/* Username - copyable (only show when completed) */}
              {selectedCharacter.status === 'completed' && (selectedCharacter.username || selectedCharacter.api_character_id) && (
                <div className="flex items-center justify-between bg-black/20 rounded-lg p-3">
                  <div className="flex-1 min-w-0">
                    <label className="text-white/40 text-xs block">角色引用</label>
                    <p className="text-white/80 text-sm truncate">@{selectedCharacter.username || selectedCharacter.api_character_id}</p>
                  </div>
                  <button
                    onClick={() => copyToClipboard(`@${selectedCharacter.username || selectedCharacter.api_character_id}`, '角色引用')}
                    className="w-8 h-8 rounded hover:bg-white/10 flex items-center justify-center text-white/40 hover:text-white transition-colors flex-shrink-0"
                    title="复制引用格式"
                  >
                    <Copy size={14} />
                  </button>
                </div>
              )}

              {/* Source type and value */}
              <div className="bg-black/20 rounded-lg p-3">
                <label className="text-white/40 text-xs block">来源</label>
                <p className="text-white/80 text-sm">
                  {selectedCharacter.source_type === 'task' ? '任务 ID: ' : 'URL: '}
                  <span className="text-white/60 break-all">{selectedCharacter.source_value}</span>
                </p>
              </div>

              {/* Timestamps */}
              {selectedCharacter.timestamps && (
                <div className="bg-black/20 rounded-lg p-3">
                  <label className="text-white/40 text-xs block">时间戳</label>
                  <p className="text-white/80 text-sm">{selectedCharacter.timestamps} 秒</p>
                </div>
              )}

              {/* Created time */}
              {selectedCharacter.created_at && (
                <div className="bg-black/20 rounded-lg p-3">
                  <label className="text-white/40 text-xs block">创建时间</label>
                  <p className="text-white/80 text-sm">
                    {new Date(selectedCharacter.created_at).toLocaleString('zh-CN')}
                  </p>
                </div>
              )}
            </div>

            {/* Actions */}
            <div className="flex gap-3 mt-5">
              {selectedCharacter.status === 'completed' && (selectedCharacter.username || selectedCharacter.api_character_id) && (
                <button
                  onClick={() => copyToClipboard(`@${selectedCharacter.username || selectedCharacter.api_character_id}`, '角色引用')}
                  className="flex-1 px-4 py-2.5 text-sm text-white/60 hover:text-white bg-white/5 hover:bg-white/10 rounded-lg transition-all flex items-center justify-center gap-2"
                >
                  <Copy size={14} />
                  复制引用
                </button>
              )}
              <button
                onClick={() => {
                  setDeleteConfirmId(selectedCharacter.id);
                }}
                className={`${selectedCharacter.status === 'completed' && (selectedCharacter.username || selectedCharacter.api_character_id) ? 'flex-1' : 'w-full'} px-4 py-2.5 text-sm text-red-400 hover:text-white bg-red-500/10 hover:bg-red-500 rounded-lg transition-all flex items-center justify-center gap-2`}
              >
                <Trash2 size={14} />
                删除角色
              </button>
            </div>

            {/* Delete confirmation in modal */}
            {deleteConfirmId === selectedCharacter.id && (
              <div className="mt-4 p-3 bg-red-500/10 border border-red-500/20 rounded-lg">
                <p className="text-white/80 text-sm mb-3">确定要删除角色 "{selectedCharacter.custom_name}" 吗？</p>
                <div className="flex gap-2">
                  <button
                    onClick={(e) => handleConfirmDelete(e, selectedCharacter)}
                    disabled={isDeletingId === selectedCharacter.id}
                    className="flex-1 px-3 py-2 text-sm bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                  >
                    {isDeletingId === selectedCharacter.id ? (
                      <Loader2 size={14} className="animate-spin" />
                    ) : (
                      '确认删除'
                    )}
                  </button>
                  <button
                    onClick={handleCancelDelete}
                    className="flex-1 px-3 py-2 text-sm bg-white/10 hover:bg-white/20 text-white/60 rounded-lg transition-colors"
                  >
                    取消
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
}

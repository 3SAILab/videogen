# Implementation Plan: Sora2 Character Creation

## Overview

将角色创建功能从 VectorEngine API 迁移到 Sora2 Character Training API。支持从任务 ID 或外部视频 URL 创建角色，并实现异步训练状态跟踪。

## Tasks

- [x] 1. Update character data models and database schema
  - [x] 1.1 Update Character struct in models.go
    - Replace api_id, api_username with api_character_id
    - Add source_type, source_value, status, progress, fail_reason fields
    - Remove profile_picture_url, permalink, from_task_id fields
    - _Requirements: 6.2, 6.3_
  - [x] 1.2 Update CreateCharacterRequest struct
    - Add source_type and source_value fields
    - Remove from_task_id field
    - _Requirements: 1.5, 2.1_
  - [x] 1.3 Add Sora2CharacterRequest and Sora2CharacterResponse structs
    - Sora2CharacterRequest with character, url, prompt, model, timestamps
    - Sora2CharacterResponse with created_at, id, model, object, progress, size, status
    - _Requirements: 1.5, 2.1_
  - [ ]* 1.4 Write property test for Character JSON serialization round-trip
    - **Property 7: Character JSON serialization round-trip**
    - **Validates: Requirements 6.3, 6.4**

- [x] 2. Update database schema and operations
  - [x] 2.1 Update characters table schema in db.go
    - Add columns: source_type, source_value, status, progress, fail_reason
    - Remove columns: api_id, api_username, profile_picture_url, permalink, from_task_id
    - Handle migration for existing data
    - _Requirements: 6.1, 6.2_
  - [x] 2.2 Update CreateCharacter function
    - Store new fields: source_type, source_value, status='pending', progress=0
    - _Requirements: 3.1, 4.1, 6.2_
  - [x] 2.3 Add UpdateCharacterStatus function
    - Update status, progress, api_character_id, fail_reason
    - _Requirements: 3.2, 3.3, 3.4_
  - [ ]* 2.4 Write property test for character storage completeness
    - **Property 4: Character storage completeness**
    - **Validates: Requirements 4.1, 6.2**
  - [ ]* 2.5 Write property test for initial status is pending
    - **Property 8: Initial status is pending**
    - **Validates: Requirements 3.1**
  - [ ]* 2.6 Write property test for character deletion
    - **Property 6: Character deletion removes record**
    - **Validates: Requirements 5.3**

- [x] 3. Implement Sora2 Character Training API client
  - [x] 3.1 Update CreateCharacter method in vectorengine.go
    - Call POST /v1/videos with new request format
    - Support both task ID (character param) and URL (url param)
    - Set model="character-training", prompt="角色创建"
    - Parse Sora2CharacterResponse
    - _Requirements: 1.5, 2.1_
  - [x] 3.2 Add QueryCharacterStatus method
    - Call GET /v1/videos/{id} to get training status
    - Parse status, progress, fail_reason
    - _Requirements: 3.2, 3.3, 3.4_

- [ ] 4. Update validation functions
  - [ ] 4.1 Update validation functions for new fields
    - ValidateSourceType: must be 'task' or 'url'
    - ValidateSourceValue: non-empty, valid URL format if source_type is 'url'
    - Keep existing: ValidateCustomName, ValidateDescription, ValidateTimestamps
    - _Requirements: 1.2, 1.3, 1.4, 2.1_
  - [ ]* 4.2 Write property test for custom name validation
    - **Property 1: Custom name validation**
    - **Validates: Requirements 1.2**
  - [ ]* 4.3 Write property test for description validation
    - **Property 2: Description validation**
    - **Validates: Requirements 1.3**
  - [ ]* 4.4 Write property test for timestamp range validation
    - **Property 3: Timestamp range validation**
    - **Validates: Requirements 1.4**

- [x] 5. Update character HTTP API handlers
  - [x] 5.1 Update POST /api/characters handler
    - Accept source_type and source_value instead of from_task_id
    - Call Sora2 API with appropriate parameters
    - Store character with status='pending'
    - _Requirements: 1.1, 1.5, 2.1, 3.1_
  - [x] 5.2 Add GET /api/characters/:id/status handler
    - Query Sora2 API for current status
    - Update local database with new status/progress
    - Return current status to frontend
    - _Requirements: 3.2, 3.3, 3.4_
  - [x] 5.3 Update GET /api/characters handler
    - Return all characters with new fields
    - _Requirements: 5.1, 5.2_
  - [x] 5.4 Update DELETE /api/characters/:id handler
    - Remove character picture cleanup (no longer needed)
    - _Requirements: 5.3_

- [x] 6. Checkpoint - Backend implementation
  - Ensure all backend changes compile and basic tests pass
  - Ask the user if questions arise

- [x] 7. Update frontend TypeScript types
  - [x] 7.1 Update Character interface in types.ts
    - Add source_type, source_value, status, progress, fail_reason
    - Remove api_id, api_username, profile_picture_url, permalink, from_task_id
    - _Requirements: 6.3_
  - [x] 7.2 Update CreateCharacterRequest interface
    - Add source_type and source_value
    - Remove from_task_id
    - _Requirements: 1.5, 2.1_
  - [x] 7.3 Add character status API function to api.ts
    - getCharacterStatus(id): fetch training status
    - _Requirements: 3.2_

- [x] 8. Update character creation dialog UI
  - [x] 8.1 Update CharacterCreationDialog component
    - Add source type selector (task/url)
    - Show task ID input when source_type='task'
    - Show URL input when source_type='url'
    - _Requirements: 1.1, 2.1_
  - [x] 8.2 Update form submission logic
    - Send source_type and source_value
    - Handle async response (status='pending')
    - _Requirements: 1.5, 2.1, 3.1_

- [x] 9. Update character list UI
  - [x] 9.1 Update CharacterList component
    - Display training status (pending/processing/completed/failed)
    - Show progress percentage for processing characters
    - Poll status for non-completed characters
    - _Requirements: 3.2, 5.1, 5.2_
  - [x] 9.2 Update CharacterCard component
    - Remove profile picture display
    - Show status badge and progress bar
    - _Requirements: 5.2_

- [x] 10. Update character reference in prompts
  - [x] 10.1 Update character name conversion utility
    - Convert custom_name to api_character_id format
    - Only allow referencing completed characters
    - _Requirements: 4.3_
  - [ ]* 10.2 Write property test for name conversion
    - **Property 5: Custom name to API character ID conversion**
    - **Validates: Requirements 4.3**

- [x] 11. Final Checkpoint - Full system test
  - Ensure all tests pass
  - Test character creation from task ID
  - Test character creation from URL
  - Ask the user if questions arise

## Notes

- Tasks marked with `*` are optional property-based tests
- The migration from old schema to new schema should handle existing data gracefully
- Status polling should be implemented with reasonable intervals to avoid API rate limits

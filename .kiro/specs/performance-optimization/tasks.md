# Implementation Plan: Performance Optimization

## Overview

实现视频生成系统的性能优化，包括虚拟滚动、视频资源管理、轮询优化和后端查询优化。

## Tasks

- [x] 1. 后端数据库索引优化
  - [x] 1.1 添加 created_at 和 status 列的数据库索引
    - 在 db.go 的 InitDB 函数中添加索引创建语句
    - _Requirements: 7.1, 7.3, 7.4_

- [x] 2. 后端API响应优化
  - [x] 2.1 确保任务列表API不返回 image_url 和 image_url2 字段
    - 验证 GetTasksPaginated 和 GetAllTasks 查询不包含这些字段
    - _Requirements: 4.4, 7.2_

- [x] 3. 前端虚拟滚动实现
  - [x] 3.1 安装 @tanstack/react-virtual 虚拟滚动库
    - 运行 npm install @tanstack/react-virtual
    - _Requirements: 1.1_
  - [x] 3.2 重构视频画廊使用虚拟滚动
    - 使用 useVirtualizer hook 替换当前的 map 渲染
    - 配置 overscan 为 5 行，确保滚动流畅
    - _Requirements: 1.1, 1.3, 1.4_

- [x] 4. 视频资源管理优化
  - [x] 4.1 实现视频播放数量限制
    - 创建 useVideoPlaybackManager hook
    - 限制最多 4 个视频同时自动播放
    - _Requirements: 2.3_
  - [x] 4.2 优化视频懒加载逻辑
    - 视频进入视口后延迟 200ms 再加载
    - 视频离开视口时暂停并释放资源
    - _Requirements: 2.1, 2.2_

- [x] 5. 轮询策略优化
  - [x] 5.1 实现基于任务状态的智能轮询
    - 无 pending/processing 任务时停止轮询
    - 只轮询 pending/processing 任务的 ID
    - _Requirements: 3.1, 3.2_
  - [x] 5.2 实现标签页可见性感知
    - 使用 Page Visibility API 检测标签页状态
    - 标签页不可见时暂停轮询
    - 标签页恢复可见时立即获取一次更新
    - _Requirements: 3.3, 3.4_

- [x] 6. 渲染性能优化
  - [x] 6.1 优化 VideoCard 组件的 memo 比较函数
    - 确保只有相关 props 变化时才重新渲染
    - _Requirements: 6.1, 6.2, 6.3_
  - [x] 6.2 实现删除任务的乐观更新
    - 点击删除后立即从 UI 移除，不等待服务器响应
    - _Requirements: 6.4_

- [x] 7. Checkpoint - 验证优化效果
  - 确保所有优化正常工作
  - 测试 1000+ 视频记录的滚动流畅度
  - 检查内存使用情况

## Notes

- 虚拟滚动是最关键的优化，可以大幅减少 DOM 节点数量
- 视频播放限制可以减少 CPU 和内存占用
- 轮询优化可以减少不必要的网络请求
- 所有优化都应该保持现有功能不变

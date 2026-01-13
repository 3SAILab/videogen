"""
Sora-2 视频生成完整流程测试脚本
包含：创建任务 -> 轮询状态 -> 下载视频
"""

import http.client
import mimetypes
import json
import time
import os
from codecs import encode

# ==================== 配置 ====================
API_KEY = "Bearer sk-YIRSJjrtRablveDqg9NrJUtQe6q67g7JvRNUAsifrp6neD0h"  # 替换为你的 API Key
BASE_URL = "api.vectorengine.ai"
OUTPUT_DIR = "test_output"  # 视频保存目录

# ==================== 1. 创建视频任务 ====================
def create_video_task(prompt, image_path=None, seconds=10, size="1280x720"):
    """
    创建视频生成任务
    
    Request 参数:
    - model: 模型名称 (sora-2)
    - prompt: 提示词
    - seconds: 视频时长 (10 或 15)
    - size: 视频尺寸 (1280x720 或 720x1280)
    - input_reference: 参考图片 (可选)
    
    Response 格式:
    {
        "id": "video_5c6a605a-30c0-4a6a-9dbd-d1d6cfdd9980",
        "object": "video",
        "model": "sora-2",
        "status": "queued",
        "progress": 0,
        "created_at": 1761622232,
        "seconds": "10",
        "size": "1280x720"
    }
    """
    print(f"\n{'='*60}")
    print("步骤 1: 创建视频任务")
    print(f"{'='*60}")
    print(f"提示词: {prompt}")
    print(f"时长: {seconds}秒")
    print(f"尺寸: {size}")
    if image_path:
        print(f"参考图片: {image_path}")
    
    conn = http.client.HTTPSConnection(BASE_URL)
    dataList = []
    boundary = 'wL36Yn8afVp8Ag7AmP8qZ0SA4n1v9T'
    
    # model
    dataList.append(encode('--' + boundary))
    dataList.append(encode('Content-Disposition: form-data; name=model;'))
    dataList.append(encode('Content-Type: {}'.format('text/plain')))
    dataList.append(encode(''))
    dataList.append(encode("sora-2"))
    
    # prompt
    dataList.append(encode('--' + boundary))
    dataList.append(encode('Content-Disposition: form-data; name=prompt;'))
    dataList.append(encode('Content-Type: {}'.format('text/plain')))
    dataList.append(encode(''))
    dataList.append(encode(prompt))
    
    # seconds
    dataList.append(encode('--' + boundary))
    dataList.append(encode('Content-Disposition: form-data; name=seconds;'))
    dataList.append(encode('Content-Type: {}'.format('text/plain')))
    dataList.append(encode(''))
    dataList.append(encode(str(seconds)))
    
    # input_reference (可选)
    if image_path and os.path.exists(image_path):
        dataList.append(encode('--' + boundary))
        dataList.append(encode('Content-Disposition: form-data; name=input_reference; filename={0}'.format(os.path.basename(image_path))))
        fileType = mimetypes.guess_type(image_path)[0] or 'application/octet-stream'
        dataList.append(encode('Content-Type: {}'.format(fileType)))
        dataList.append(encode(''))
        with open(image_path, 'rb') as f:
            dataList.append(f.read())
    
    # size
    dataList.append(encode('--' + boundary))
    dataList.append(encode('Content-Disposition: form-data; name=size;'))
    dataList.append(encode('Content-Type: {}'.format('text/plain')))
    dataList.append(encode(''))
    dataList.append(encode(size))
    
    dataList.append(encode('--'+boundary+'--'))
    dataList.append(encode(''))
    
    body = b'\r\n'.join(dataList)
    headers = {
        'Authorization': API_KEY,
        'Content-type': 'multipart/form-data; boundary={}'.format(boundary)
    }
    
    try:
        conn.request("POST", "/v1/videos", body, headers)
        res = conn.getresponse()
        data = res.read()
        response = json.loads(data.decode("utf-8"))
        
        print(f"\n✓ 任务创建成功!")
        print(f"任务ID: {response.get('id')}")
        print(f"状态: {response.get('status')}")
        print(f"完整响应: {json.dumps(response, indent=2, ensure_ascii=False)}")
        
        return response.get('id')
    except Exception as e:
        print(f"\n✗ 创建任务失败: {e}")
        return None
    finally:
        conn.close()


# ==================== 2. 查询任务状态 ====================
def query_task_status(task_id):
    """
    查询视频生成任务状态
    
    Request 参数:
    - task_id: 任务ID (例如: sora-2:task_01k81e7r1mf0qtvp3ett3mr4jm)
    
    Response 格式 (pending):
    {
        "id": "sora-2:task_01k6x15vhrff09dkkqjrzwhm60",
        "detail": {
            "id": "task_01k6x15vhrff09dkkqjrzwhm60",
            "status": "pending",
            "pending_info": { ... }
        },
        "status": "pending",
        "status_update_time": 1759763621142
    }
    
    Response 格式 (completed):
    {
        "id": "sora-2:task_xxx",
        "status": "completed",
        "video_url": "https://...",
        "enhanced_prompt": "...",
        "status_update_time": 1759763621142
    }
    """
    conn = http.client.HTTPSConnection(BASE_URL)
    headers = {
        'Accept': 'application/json',
        'Authorization': API_KEY
    }
    
    try:
        conn.request("GET", f"/v1/videos/{task_id}", "", headers)
        res = conn.getresponse()
        data = res.read()
        response = json.loads(data.decode("utf-8"))
        return response
    except Exception as e:
        print(f"✗ 查询状态失败: {e}")
        return None
    finally:
        conn.close()


# ==================== 3. 轮询直到完成 ====================
def wait_for_completion(task_id, check_interval=10, max_wait_time=600):
    """
    轮询任务状态直到完成
    
    参数:
    - task_id: 任务ID
    - check_interval: 检查间隔(秒)
    - max_wait_time: 最大等待时间(秒)
    """
    print(f"\n{'='*60}")
    print("步骤 2: 等待视频生成完成")
    print(f"{'='*60}")
    print(f"任务ID: {task_id}")
    print(f"检查间隔: {check_interval}秒")
    
    start_time = time.time()
    
    while True:
        elapsed = time.time() - start_time
        if elapsed > max_wait_time:
            print(f"\n✗ 超时: 等待时间超过 {max_wait_time} 秒")
            return None
        
        response = query_task_status(task_id)
        if not response:
            time.sleep(check_interval)
            continue
        
        status = response.get('status', '')
        
        # 提取进度信息
        progress = 0
        if 'detail' in response and 'pending_info' in response['detail']:
            progress_pct = response['detail']['pending_info'].get('progress_pct', 0)
            progress = int(progress_pct * 100)
        
        print(f"[{int(elapsed)}s] 状态: {status} | 进度: {progress}%")
        
        if status == 'completed':
            print(f"\n✓ 视频生成完成!")
            video_url = response.get('video_url')
            if not video_url and 'detail' in response:
                # 尝试从 detail.url 获取
                video_url = response.get('detail', {}).get('url')
            
            print(f"视频URL: {video_url}")
            return video_url
        
        elif status == 'failed':
            print(f"\n✗ 任务失败")
            print(f"完整响应: {json.dumps(response, indent=2, ensure_ascii=False)}")
            return None
        
        time.sleep(check_interval)


# ==================== 4. 下载视频 ====================
def download_video(video_url, task_id):
    """
    下载视频到本地
    
    参数:
    - video_url: 视频URL
    - task_id: 任务ID (用于生成文件名)
    """
    print(f"\n{'='*60}")
    print("步骤 3: 下载视频")
    print(f"{'='*60}")
    print(f"视频URL: {video_url}")
    
    # 确保输出目录存在
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    
    # 生成文件名
    safe_task_id = task_id.replace(':', '_').replace('/', '_')
    filename = f"{safe_task_id}_{int(time.time())}.mp4"
    filepath = os.path.join(OUTPUT_DIR, filename)
    
    try:
        # 解析URL
        from urllib.parse import urlparse
        parsed = urlparse(video_url)
        
        # 建立连接
        if parsed.scheme == 'https':
            conn = http.client.HTTPSConnection(parsed.netloc)
        else:
            conn = http.client.HTTPConnection(parsed.netloc)
        
        # 下载
        path = parsed.path
        if parsed.query:
            path += '?' + parsed.query
        
        conn.request("GET", path)
        res = conn.getresponse()
        
        if res.status == 200:
            with open(filepath, 'wb') as f:
                f.write(res.read())
            
            file_size = os.path.getsize(filepath)
            print(f"\n✓ 下载成功!")
            print(f"保存路径: {filepath}")
            print(f"文件大小: {file_size / 1024 / 1024:.2f} MB")
            return filepath
        else:
            print(f"\n✗ 下载失败: HTTP {res.status}")
            return None
    except Exception as e:
        print(f"\n✗ 下载失败: {e}")
        return None
    finally:
        conn.close()


# ==================== 主流程 ====================
def main():
    """
    完整流程: 创建任务 -> 等待完成 -> 下载视频
    """
    print("\n" + "="*60)
    print("Sora-2 视频生成完整流程测试")
    print("="*60)
    
    # 配置参数
    prompt = "一只可爱的熊猫在竹林里吃竹子"
    image_path = None  # 如果有参考图片，填写路径
    seconds = 10
    size = "1280x720"  # 横屏: 1280x720, 竖屏: 720x1280
    
    # 步骤1: 创建任务
    task_id = create_video_task(prompt, image_path, seconds, size)
    if not task_id:
        print("\n流程终止: 创建任务失败")
        return
    
    # 步骤2: 等待完成
    video_url = wait_for_completion(task_id, check_interval=10, max_wait_time=600)
    if not video_url:
        print("\n流程终止: 视频生成失败或超时")
        return
    
    # 步骤3: 下载视频
    filepath = download_video(video_url, task_id)
    if not filepath:
        print("\n流程终止: 下载视频失败")
        return
    
    print(f"\n{'='*60}")
    print("✓ 完整流程执行成功!")
    print(f"{'='*60}")
    print(f"任务ID: {task_id}")
    print(f"视频文件: {filepath}")


if __name__ == "__main__":
    main()

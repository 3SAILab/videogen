import http.client
import json
import time
import concurrent.futures
from threading import Lock

# --- 配置部分 ---
API_HOST = "api.vectorengine.ai"
# 注意：生产环境中请勿将 Key 硬编码在代码里，建议使用环境变量
API_KEY = "Bearer sk-YIRSJjrtRablveDqg9NrJUtQe6q67g7JvRNUAsifrp6neD0h" 

headers = {
    'Accept': 'application/json',
    'Authorization': API_KEY,
    'Content-Type': 'application/json'
}

print_lock = Lock()

def safe_print(*args, **kwargs):
    """线程安全的打印函数"""
    with print_lock:
        print(*args, **kwargs)

def create_video_task(task_index):
    """第一步：创建视频生成任务"""
    conn = http.client.HTTPSConnection(API_HOST)
    
    payload = json.dumps({
        "images": [
            "https://filesystem.site/cdn/20250612/998IGmUiM2koBGZM3UnZeImbPBNIUL.png"
        ],
        "model": "sora-2",
        "orientation": "portrait",
        "prompt": "make animate",
        "size": "large",
        "duration": 15,
        "watermark": False
    })
    
    safe_print(f"[任务 {task_index}] >>> 正在提交任务...")
    conn.request("POST", "/v1/video/create", payload, headers)
    
    res = conn.getresponse()
    data = res.read().decode("utf-8")
    conn.close()
    
    try:
        response_json = json.loads(data)
        if "id" in response_json:
            safe_print(f"[任务 {task_index}] ✅ 任务提交成功! Task ID: {response_json['id']}")
            return response_json["id"]
        else:
            safe_print(f"[任务 {task_index}] ❌ 提交失败，未获取到ID: {data}")
            return None
    except json.JSONDecodeError:
        safe_print(f"[任务 {task_index}] ❌ 解析响应失败: {data}")
        return None

def poll_task_status(task_id, task_index):
    """第二步：循环查询任务状态直到完成"""
    safe_print(f"[任务 {task_index}] >>> 开始轮询任务状态 (ID: {task_id})...")
    
    while True:
        conn = http.client.HTTPSConnection(API_HOST)
        conn.request("GET", f"/v1/video/query?id={task_id}", headers=headers)
        
        res = conn.getresponse()
        data = res.read().decode("utf-8")
        conn.close()

        try:
            task_info = json.loads(data)
            status = task_info.get("status")
            progress = task_info.get("progress", 0)
            
            safe_print(f"[任务 {task_index}] Status: {status} | Progress: {progress}%")
            
            if status == "completed":
                safe_print(f"\n[任务 {task_index}] 🎉 任务完成！")
                return task_info
            
            elif status == "failed":
                safe_print(f"\n[任务 {task_index}] ❌ 任务失败。")
                safe_print(task_info)
                return task_info
            
            else:
                time.sleep(3)
                
        except json.JSONDecodeError:
            safe_print(f"[任务 {task_index}] ⚠️ 解析查询响应失败，稍后重试... Raw: {data}")
            time.sleep(3)

def run_single_task(task_index):
    """运行单个完整任务流程"""
    task_id = create_video_task(task_index)
    
    if task_id:
        final_result = poll_task_status(task_id, task_index)
        
        if final_result:
            video_url = final_result.get('video_url', 'N/A')
            safe_print(f"\n[任务 {task_index}] >>> 最终视频链接: {video_url}")
            
            # 保存结果到单独文件
            filename = f"final_response_task_{task_index}.json"
            with open(filename, "w", encoding="utf-8") as f:
                json.dump(final_result, f, indent=4, ensure_ascii=False)
            safe_print(f"[任务 {task_index}] >>> 结果已保存至 {filename}")
            
            return {"task_index": task_index, "result": final_result}
    
    return {"task_index": task_index, "result": None}

# --- 主程序流程 ---

if __name__ == "__main__":
    NUM_TASKS = 4  # 同时启动的任务数量
    
    print(f"🚀 开始同时启动 {NUM_TASKS} 个视频生成任务...\n")
    
    # 使用线程池并行执行任务
    with concurrent.futures.ThreadPoolExecutor(max_workers=NUM_TASKS) as executor:
        # 提交所有任务
        futures = {executor.submit(run_single_task, i+1): i+1 for i in range(NUM_TASKS)}
        
        # 收集所有结果
        all_results = []
        for future in concurrent.futures.as_completed(futures):
            task_index = futures[future]
            try:
                result = future.result()
                all_results.append(result)
            except Exception as e:
                safe_print(f"[任务 {task_index}] ❌ 执行异常: {e}")
    
    # 汇总所有结果
    print("\n" + "="*50)
    print("📊 所有任务执行完毕，汇总结果：")
    print("="*50)
    
    for r in sorted(all_results, key=lambda x: x["task_index"]):
        idx = r["task_index"]
        res = r["result"]
        if res and res.get("status") == "completed":
            print(f"  ✅ 任务 {idx}: 成功 - {res.get('video_url', 'N/A')}")
        elif res and res.get("status") == "failed":
            print(f"  ❌ 任务 {idx}: 失败")
        else:
            print(f"  ⚠️ 任务 {idx}: 未完成或无结果")
    
    # 保存汇总结果
    with open("all_results.json", "w", encoding="utf-8") as f:
        json.dump(all_results, f, indent=4, ensure_ascii=False)
    print("\n>>> 汇总结果已保存至 all_results.json")

import requests
import time

API = "http://localhost:8080/api"

print("=== 第一步：降低所有服务器功耗到 200W，让温度降下来 ===")
ids = []
for rack in ['A', 'B', 'C', 'D']:
    for i in range(1, 11):
        ids.append(f"{rack}{i:02d}")

r = requests.post(f"{API}/servers/batch-power-limit", json={
    "server_ids": ids,
    "power_limit": 200
})
print(f"设置功耗限制: {r.json()['success']}")

print("\n等待 6 秒让温度下降...")
for i in range(6):
    time.sleep(1)
    r = requests.get(f"{API}/servers")
    servers = r.json()["data"]
    avg_temp = sum(s["cpu_temp"] for s in servers) / len(servers)
    cold_count = sum(1 for s in servers if s["cpu_temp"] < 50)
    print(f"  第 {i+1}s: 平均温度 {avg_temp:.1f}°C, <50°C 的有 {cold_count} 台")

print("\n=== 第二步：开启静音模式 ===")
r = requests.post(f"{API}/silent-mode", json={"enabled": True})
data = r.json()["data"]
print(f"静音模式: {data['enabled']}")
print(f"被限制服务器: {data['limited_servers']}/{data['total_servers']}")

print("\n等待 4 秒观察效果...")
for i in range(4):
    time.sleep(1)
    r = requests.get(f"{API}/servers")
    servers = r.json()["data"]
    cold = [s for s in servers if s["cpu_temp"] < 50]
    limited = [s for s in servers if s["fan_silent_limited"]]
    
    avg_fan_cold = sum(s["fan_speed"] for s in cold) / len(cold) if cold else 0
    avg_fan_hot = sum(s["fan_speed"] for s in servers if s["cpu_temp"] >= 50) / max(1, sum(1 for s in servers if s["cpu_temp"] >= 50))
    
    print(f"  第 {i+1}s: 低温 {len(cold)} 台(被限 {len(limited)} 台), 低温均速 {avg_fan_cold:.0f} RPM, 高温均速 {avg_fan_hot:.0f} RPM")

print("\n=== 第三步：恢复功耗到 500W，看温度升高后自动解除 ===")
r = requests.post(f"{API}/servers/batch-power-limit", json={
    "server_ids": ids,
    "power_limit": 500
})

print("等待 8 秒观察温度回升...")
for i in range(8):
    time.sleep(1)
    r = requests.get(f"{API}/servers")
    servers = r.json()["data"]
    cold = [s for s in servers if s["cpu_temp"] < 50]
    limited = [s for s in servers if s["fan_silent_limited"]]
    avg_temp = sum(s["cpu_temp"] for s in servers) / len(servers)
    
    print(f"  第 {i+1}s: 平均 {avg_temp:.1f}°C, 低温 {len(cold)} 台, 被静音限制 {len(limited)} 台")

print("\n=== 第四步：关闭静音模式 ===")
r = requests.post(f"{API}/silent-mode", json={"enabled": False})
data = r.json()["data"]
print(f"静音模式: {data['enabled']}")
print(f"被限制服务器: {data['limited_servers']}/{data['total_servers']}")

print("\n✅ 静音模式完整测试完成")

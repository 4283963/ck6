import requests
import time

API = "http://localhost:8080/api"

print("=== 1. 初始状态 ===")
r = requests.get(f"{API}/silent-mode")
data = r.json()["data"]
print(f"静音模式: {data['enabled']}")
print(f"被限制服务器: {data['limited_servers']}/{data['total_servers']}")

print("\n=== 2. 开启静音模式 ===")
r = requests.post(f"{API}/silent-mode", json={"enabled": True})
data = r.json()["data"]
print(f"静音模式: {data['enabled']}")
print(f"被限制服务器: {data['limited_servers']}/{data['total_servers']}")

time.sleep(3)

print("\n=== 3. 3秒后状态 ===")
r = requests.get(f"{API}/servers")
servers = r.json()["data"]

cold = [s for s in servers if s["cpu_temp"] < 50]
hot = [s for s in servers if s["cpu_temp"] >= 50]
limited_cold = [s for s in cold if s["fan_silent_limited"]]
limited_hot = [s for s in hot if s["fan_silent_limited"]]

print(f"温度<50°C 服务器: {len(cold)} 台")
print(f"  其中被静音限制: {len(limited_cold)} 台")
if limited_cold:
    avg = sum(s["fan_speed"] for s in limited_cold) / len(limited_cold)
    print(f"  平均风扇转速: {avg:.0f} RPM")
    print(f"  最大风扇转速: {max(s['fan_speed'] for s in limited_cold)} RPM")
    print(f"  30% 上限: {int(8000 * 0.3)} RPM")

print(f"\n温度>=50°C 服务器: {len(hot)} 台")
print(f"  其中被静音限制: {len(limited_hot)} 台 (应为0)")
if hot:
    avg = sum(s["fan_speed"] for s in hot) / len(hot)
    print(f"  平均风扇转速: {avg:.0f} RPM")

print("\n=== 4. 关闭静音模式 ===")
r = requests.post(f"{API}/silent-mode", json={"enabled": False})
data = r.json()["data"]
print(f"静音模式: {data['enabled']}")
print(f"被限制服务器: {data['limited_servers']}/{data['total_servers']}")

print("\n✅ 静音模式 API 测试完成")

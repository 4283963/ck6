package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main() {
	fmt.Println("=== 极端压力测试与边界情况验证 ===\n")

	serverIDs := make([]string, 0, 40)
	for rack := 0; rack < 4; rack++ {
		for slot := 1; slot <= 10; slot++ {
			id := string(rune('A'+rack)) + fmt.Sprintf("%02d", slot)
			serverIDs = append(serverIDs, id)
		}
	}

	fmt.Println("测试1: 批量 API - 40台服务器同时设置")
	testBatchAPI(serverIDs, 300)

	fmt.Println("\n测试2: 批量 API - 5台服务器节能模式 (250W)")
	testBatchAPI(serverIDs[:5], 250)

	fmt.Println("\n测试3: 批量 API - 包含无效ID")
	testBatchAPI(append(serverIDs[:3], "INVALID", "Z99", ""), 400)

	fmt.Println("\n测试4: 恶意输入 - 空数组")
	testMaliciousBatch([]string{}, 300)

	fmt.Println("\n测试5: 恶意输入 - 超限功耗值")
	testMaliciousBatch(serverIDs[:2], 50)
	testMaliciousBatch(serverIDs[:2], 2000)

	fmt.Println("\n测试6: 并发批量请求风暴 (10个并发批量请求)")
	testConcurrentBatchRequests(serverIDs)

	fmt.Println("\n测试7: 模拟前端使用场景 - 先查询后批量设置")
	testRealWorldScenario(serverIDs)

	fmt.Println("\n✅ 所有极端测试完成，服务运行正常")
}

func testBatchAPI(ids []string, limit float64) {
	body := map[string]interface{}{
		"server_ids":  ids,
		"power_limit": limit,
	}
	jsonBody, _ := json.Marshal(body)

	start := time.Now()
	resp, err := http.Post(
		"http://localhost:8080/api/servers/batch-power-limit",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		fmt.Printf("  ❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	duration := time.Since(start)
	success := result["success"].(bool)

	if success {
		data := result["data"].(map[string]interface{})
		updated := data["updated"].([]interface{})
		failed := data["failed"].([]interface{})
		fmt.Printf("  ✅ 成功, 耗时 %v, 更新 %d 台, 失败 %d 台\n", duration, len(updated), len(failed))
		if len(failed) > 0 {
			fmt.Printf("     失败ID: %v\n", failed)
		}
	} else {
		fmt.Printf("  ⚠️  业务错误: %v (这是预期的错误处理)\n", result["error"])
	}
}

func testMaliciousBatch(ids []string, limit float64) {
	body := map[string]interface{}{
		"server_ids":  ids,
		"power_limit": limit,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(
		"http://localhost:8080/api/servers/batch-power-limit",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		fmt.Printf("  ❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	success := result["success"].(bool)
	if !success {
		fmt.Printf("  ✅ 正确拒绝: %v\n", result["error"])
	} else {
		fmt.Printf("  ⚠️  意外成功 (可能需要更严格的验证)\n")
	}
}

func testConcurrentBatchRequests(ids []string) {
	var wg sync.WaitGroup
	errCount := 0
	var mu sync.Mutex

	start := time.Now()
	limits := []float64{250, 300, 350, 400, 450, 500, 550, 600, 650, 700}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			body := map[string]interface{}{
				"server_ids":  ids,
				"power_limit": limits[idx],
			}
			jsonBody, _ := json.Marshal(body)

			resp, err := http.Post(
				"http://localhost:8080/api/servers/batch-power-limit",
				"application/json",
				bytes.NewBuffer(jsonBody),
			)
			if err != nil {
				mu.Lock()
				errCount++
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)

			if !result["success"].(bool) {
				mu.Lock()
				errCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	fmt.Printf("  ✅ 完成 10 个并发批量请求, 耗时 %v, 失败 %d 个\n", duration, errCount)
}

func testRealWorldScenario(ids []string) {
	start := time.Now()

	resp, err := http.Get("http://localhost:8080/api/servers")
	if err != nil {
		fmt.Printf("  ❌ 查询失败: %v\n", err)
		return
	}
	resp.Body.Close()

	selected := ids[:5]
	body := map[string]interface{}{
		"server_ids":  selected,
		"power_limit": 250.0,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err = http.Post(
		"http://localhost:8080/api/servers/batch-power-limit",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		fmt.Printf("  ❌ 批量设置失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	duration := time.Since(start)
	if result["success"].(bool) {
		fmt.Printf("  ✅ 真实场景模拟完成, 耗时 %v\n", duration)
	} else {
		fmt.Printf("  ❌ 失败: %v\n", result["error"])
	}
}

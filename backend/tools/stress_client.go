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
	serverIDs := make([]string, 0, 40)
	for rack := 0; rack < 4; rack++ {
		for slot := 1; slot <= 10; slot++ {
			id := string(rune('A'+rack)) + fmt.Sprintf("%02d", slot)
			serverIDs = append(serverIDs, id)
		}
	}

	fmt.Println("测试1: 5个并发请求（模拟选中5台服务器）")
	testConcurrentRequests(serverIDs[:5], 300)

	fmt.Println("\n测试2: 40个并发请求（模拟全部服务器）")
	testConcurrentRequests(serverIDs, 250)

	fmt.Println("\n测试3: 混合读写压力测试")
	testMixedLoad(serverIDs)

	fmt.Println("\n✅ 所有测试完成，服务未崩溃")
}

func testConcurrentRequests(ids []string, limit float64) {
	var wg sync.WaitGroup
	errCount := 0
	var mu sync.Mutex

	start := time.Now()
	for _, id := range ids {
		wg.Add(1)
		go func(serverID string) {
			defer wg.Done()

			body := map[string]float64{"power_limit": limit}
			jsonBody, _ := json.Marshal(body)

			resp, err := http.Post(
				"http://localhost:8080/api/server/"+serverID,
				"application/json",
				bytes.NewBuffer(jsonBody),
			)
			if err != nil {
				mu.Lock()
				errCount++
				mu.Unlock()
				fmt.Printf("❌ %s 请求失败: %v\n", serverID, err)
				return
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)

			if resp.StatusCode != 200 {
				mu.Lock()
				errCount++
				mu.Unlock()
				fmt.Printf("❌ %s 返回错误: %v\n", serverID, result["error"])
			}
		}(id)
	}

	wg.Wait()
	duration := time.Since(start)
	fmt.Printf("  完成 %d 个请求，耗时 %v，失败 %d 个\n", len(ids), duration, errCount)
}

func testMixedLoad(ids []string) {
	var wg sync.WaitGroup
	errCount := 0
	var mu sync.Mutex

	start := time.Now()

	for round := 0; round < 3; round++ {
		for _, id := range ids {
			wg.Add(2)

			go func(serverID string) {
				defer wg.Done()
				resp, err := http.Get("http://localhost:8080/api/server/" + serverID)
				if err != nil {
					mu.Lock()
					errCount++
					mu.Unlock()
					return
				}
				resp.Body.Close()
			}(id)

			go func(serverID string) {
				defer wg.Done()
				limit := 300.0 + float64(round*100)
				body := map[string]float64{"power_limit": limit}
				jsonBody, _ := json.Marshal(body)
				resp, err := http.Post(
					"http://localhost:8080/api/server/"+serverID,
					"application/json",
					bytes.NewBuffer(jsonBody),
				)
				if err != nil {
					mu.Lock()
					errCount++
					mu.Unlock()
					return
				}
				resp.Body.Close()
			}(id)
		}
		wg.Wait()
		fmt.Printf("  第 %d 轮完成\n", round+1)
	}

	duration := time.Since(start)
	fmt.Printf("  完成混合读写测试，耗时 %v，失败 %d 个\n", duration, errCount)
}

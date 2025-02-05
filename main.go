package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

func getOnlineWorkers(token string) (int, error) {
	// URL 생성
	url := "https://relay.kuzco.xyz/api/trpc/instance.countOnline?batch=1&input={\"0\":{\"json\":null,\"meta\":{\"values\":[\"undefined\"]}}}"

	// HTTP GET 요청 준비
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating request: %v", err)
	}

	// 헤더 설정
	req.Header.Set("Authorization", "Bearer "+token)

	// 요청 보내기
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response: %v", err)
	}

	// 응답 파싱
	var countResp []CountOnlineResponse
	err = json.Unmarshal(body, &countResp)
	if err != nil {
		return 0, fmt.Errorf("error parsing response: %v", err)
	}

	if len(countResp) > 0 {
		return countResp[0].Result.Data.JSON.Count, nil
	}

	return 0, fmt.Errorf("no response received")
}

func getServerRPM(token string) (int, error) {
	// URL 생성
	url := "https://relay.kuzco.xyz/api/trpc/metrics.rpm?batch=1&input={\"0\":{\"json\":null,\"meta\":{\"values\":[\"undefined\"]}}}"

	// HTTP GET 요청 준비
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating request: %v", err)
	}

	// 헤더 설정
	req.Header.Set("Authorization", "Bearer "+token)

	// 요청 보내기
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response: %v", err)
	}

	// 응답 파싱
	var rpmResp []RPMResponse
	err = json.Unmarshal(body, &rpmResp)
	if err != nil {
		return 0, fmt.Errorf("error parsing response: %v", err)
	}

	if len(rpmResp) > 0 {
		return rpmResp[0].Result.Data.JSON, nil
	}

	return 0, fmt.Errorf("no response received")
}

func getTokensLast24Hours(token string) (int64, error) {
	// URL 생성
	url := "https://relay.kuzco.xyz/api/trpc/metrics.globalWorker?batch=1&input={\"0\":{\"json\":null,\"meta\":{\"values\":[\"undefined\"]}}}"

	// HTTP GET 요청 준비
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating request: %v", err)
	}

	// 헤더 설정
	req.Header.Set("Authorization", "Bearer "+token)

	// 요청 보내기
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response: %v", err)
	}

	// 응답 파싱
	var workerResp []GlobalWorkerResponse
	err = json.Unmarshal(body, &workerResp)
	if err != nil {
		return 0, fmt.Errorf("error parsing response: %v", err)
	}

	if len(workerResp) > 0 {
		return workerResp[0].Result.Data.JSON.TokensLast24Hours, nil
	}

	return 0, fmt.Errorf("no response received")
}

func getUserMetrics(token string, userId string) (*UserMetricsResponse, error) {
	// URL 생성
	url := fmt.Sprintf("https://relay.kuzco.xyz/api/trpc/metrics.user?batch=1&input={\"0\":{\"json\":{\"userId\":\"%s\"}}}", userId)

	// HTTP GET 요청 준비
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// 헤더 설정
	req.Header.Set("Authorization", "Bearer "+token)

	// 요청 보내기
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// 응답 파싱
	var metricsResp []UserMetricsResponse
	err = json.Unmarshal(body, &metricsResp)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if len(metricsResp) > 0 {
		return &metricsResp[0], nil
	}

	return nil, fmt.Errorf("no response received")
}

func getActiveWorkers(token string) ([]Worker, error) {
	// URL 생성
	url := "https://relay.kuzco.xyz/api/trpc/worker.list?batch=1&input={\"0\":{\"json\":null,\"meta\":{\"values\":[\"undefined\"]}}}"

	// HTTP GET 요청 준비
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// 헤더 설정
	req.Header.Set("Authorization", "Bearer "+token)

	// 요청 보내기
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// 응답 파싱
	var workerResp []WorkerListResponse
	err = json.Unmarshal(body, &workerResp)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if len(workerResp) > 0 {
		// isArchived가 false이고 실행 중인 인스턴스가 있는 워커만 필터링
		var activeWorkers []Worker
		for _, worker := range workerResp[0].Result.Data.JSON.Workers {
			if !worker.IsArchived && len(worker.Instances) > 0 {
				// 실행 중인 인스턴스가 하나라도 있는지 확인
				hasRunningInstance := false
				for _, instance := range worker.Instances {
					if instance.Status == "Running" {
						hasRunningInstance = true
						break
					}
				}
				if hasRunningInstance {
					activeWorkers = append(activeWorkers, worker)
				}
			}
		}
		return activeWorkers, nil
	}

	return nil, fmt.Errorf("no response received")
}

func getWorkerMetrics(token string, workerId string) (*WorkerMetricsResponse, error) {
	// URL 생성
	url := fmt.Sprintf("https://relay.kuzco.xyz/api/trpc/metrics.worker?batch=1&input={\"0\":{\"json\":{\"workerId\":\"%s\"}}}", workerId)

	// HTTP GET 요청 준비
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// 헤더 설정
	req.Header.Set("Authorization", "Bearer "+token)

	// 요청 보내기
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// 응답 파싱
	var metricsResp []WorkerMetricsResponse
	err = json.Unmarshal(body, &metricsResp)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if len(metricsResp) > 0 {
		return &metricsResp[0], nil
	}

	return nil, fmt.Errorf("no response received")
}

func loadConfig(filename string) (*Config, error) {
	configFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &config, nil
}

func printWorkerInfo(worker Worker, metrics *WorkerMetricsResponse) {
	fmt.Printf("\nWorker: %s\n", worker.Name)
	fmt.Printf("  ID: %s\n", worker.ID)
	fmt.Printf("  Metrics:\n")
	fmt.Printf("    Generations Last 24h: %d\n", metrics.Result.Data.JSON.GenerationsLast24Hours)
	fmt.Printf("    Tokens Last 24h: %d\n", metrics.Result.Data.JSON.TokensLast24Hours)
	fmt.Printf("    Tokens All Time: %d\n", metrics.Result.Data.JSON.TokensAllTime)
	fmt.Printf("  Instances:\n")

	for _, instance := range worker.Instances {

		fmt.Printf("    - %s (%s)\n", instance.Name, instance.Status)
		fmt.Printf("      Runtime: %s\n", instance.Info.Runtime)
		fmt.Printf("      Location: %s, %s (%s)\n",
			instance.Info.City,
			instance.Info.Country,
			instance.Info.IPAddress)

	}
}

func main() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	// 로거 함수 정의
	logger := func(message string) {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("[%s] %s\n", timestamp, message)
	}

	// 로거 함수를 NewMonitor에 전달
	monitor := NewMonitor(config, logger)
	monitor.Start()

	// 프로그램이 종료되지 않도록 대기
	select {}
}

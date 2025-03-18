package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

var (
	globalCurrentMetrics *MinuteMetrics
	globalMetricsLock    sync.Mutex
)

// MetricsServer는 메트릭스 데이터를 제공하는 HTTP 서버입니다
type MetricsServer struct {
	port int
}

// NewMetricsServer는 새로운 MetricsServer 인스턴스를 생성합니다
func NewMetricsServer(port int) *MetricsServer {
	return &MetricsServer{
		port: port,
	}
}

// UpdateMetrics는 글로벌 메트릭스 변수를 업데이트합니다
func UpdateMetrics(metrics MinuteMetrics) {
	globalMetricsLock.Lock()
	defer globalMetricsLock.Unlock()
	globalCurrentMetrics = &metrics
	log.Printf("Global metrics updated in API server")
}

// Start는 메트릭스 서버를 시작합니다
func (s *MetricsServer) Start() {
	http.HandleFunc("/", s.handleRoot)
	http.HandleFunc("/api/metrics", s.handleMetrics)
	http.HandleFunc("/api/user", s.handleUserMetrics)
	http.HandleFunc("/api/general", s.handleGeneralMetrics)
	http.HandleFunc("/api/hourly", s.handleHourlyStats)
	http.HandleFunc("/api/workers", s.handleWorkers)
	http.HandleFunc("/api/calculations", s.handleCalculations)

	log.Printf("Starting metrics server on port %d... (개발 모드 전용)", s.port)
	addr := fmt.Sprintf(":%d", s.port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start metrics server: %v", err)
	}
}

// 루트 경로에서 HTML 페이지를 제공합니다
func (s *MetricsServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>쿠즈코 모니터링 데이터</title>
		<meta charset="UTF-8">
		<style>
			body {
				font-family: Arial, sans-serif;
				margin: 20px;
				background-color: #f5f5f5;
			}
			h1 {
				color: #333;
				border-bottom: 1px solid #ddd;
				padding-bottom: 10px;
			}
			.section {
				background-color: white;
				border-radius: 5px;
				padding: 15px;
				margin-bottom: 20px;
				box-shadow: 0 2px 4px rgba(0,0,0,0.1);
			}
			.endpoints {
				background-color: #f0f8ff;
				padding: 15px;
				border-radius: 5px;
				margin-top: 20px;
			}
			a {
				color: #0066cc;
				text-decoration: none;
				display: block;
				margin: 5px 0;
			}
			a:hover {
				text-decoration: underline;
			}
			pre {
				background-color: #f9f9f9;
				padding: 10px;
				border-radius: 4px;
				overflow-x: auto;
			}
			.refresh-btn {
				padding: 8px 16px;
				background-color: #4CAF50;
				color: white;
				border: none;
				border-radius: 4px;
				cursor: pointer;
				margin-bottom: 20px;
			}
			.refresh-btn:hover {
				background-color: #45a049;
			}
			#jsonDisplay {
				white-space: pre-wrap;
				background-color: #f8f8f8;
				border: 1px solid #ddd;
				padding: 10px;
				border-radius: 5px;
				max-height: 400px;
				overflow-y: auto;
			}
		</style>
	</head>
	<body>
		<h1>쿠즈코 모니터링 데이터</h1>
		
		<button class="refresh-btn" onclick="fetchData('/api/metrics')">모든 데이터 조회</button>
		
		<div class="section">
			<h2>데이터 미리보기</h2>
			<pre id="jsonDisplay">데이터를 조회하려면 위의 버튼을 클릭하거나 아래 링크 중 하나를 선택하세요.</pre>
		</div>
		
		<div class="endpoints">
			<h2>사용 가능한 API 엔드포인트</h2>
			<a href="#" onclick="fetchData('/api/metrics'); return false;">/api/metrics - 모든 메트릭스 데이터</a>
			<a href="#" onclick="fetchData('/api/user'); return false;">/api/user - 사용자 메트릭스 데이터</a>
			<a href="#" onclick="fetchData('/api/general'); return false;">/api/general - 일반 메트릭스 데이터</a>
			<a href="#" onclick="fetchData('/api/hourly'); return false;">/api/hourly - 시간별 통계 데이터</a>
			<a href="#" onclick="fetchData('/api/workers'); return false;">/api/workers - 워커 리스트 및 상세 정보</a>
			<a href="#" onclick="fetchData('/api/calculations'); return false;">/api/calculations - 포인트 및 효율성 계산</a>
		</div>
		
		<script>
			function fetchData(url) {
				fetch(url)
					.then(response => response.json())
					.then(data => {
						document.getElementById('jsonDisplay').textContent = JSON.stringify(data, null, 2);
					})
					.catch(err => {
						document.getElementById('jsonDisplay').textContent = '데이터를 가져오는 중 오류가 발생했습니다: ' + err.message;
					});
			}
		</script>
	</body>
	</html>
	`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// handleMetrics는 전체 메트릭스 데이터를 JSON으로 반환합니다
func (s *MetricsServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	globalMetricsLock.Lock()
	metrics := globalCurrentMetrics
	globalMetricsLock.Unlock()

	if metrics == nil {
		http.Error(w, "메트릭스 데이터가 아직 수집되지 않았습니다", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(metrics)
}

// handleUserMetrics는 사용자 메트릭스 데이터를 JSON으로 반환합니다
func (s *MetricsServer) handleUserMetrics(w http.ResponseWriter, r *http.Request) {
	globalMetricsLock.Lock()
	metrics := globalCurrentMetrics
	globalMetricsLock.Unlock()

	if metrics == nil {
		http.Error(w, "메트릭스 데이터가 아직 수집되지 않았습니다", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(metrics.User)
}

// handleGeneralMetrics는 일반 메트릭스 데이터를 JSON으로 반환합니다
func (s *MetricsServer) handleGeneralMetrics(w http.ResponseWriter, r *http.Request) {
	globalMetricsLock.Lock()
	metrics := globalCurrentMetrics
	globalMetricsLock.Unlock()

	if metrics == nil {
		http.Error(w, "메트릭스 데이터가 아직 수집되지 않았습니다", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(metrics.General)
}

// handleHourlyStats는 시간별 통계 데이터를 JSON으로 반환합니다
func (s *MetricsServer) handleHourlyStats(w http.ResponseWriter, r *http.Request) {
	stats := GlobalHourlyStats.GetStats()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(stats)
}

// handleWorkers는 워커 데이터를 JSON으로 반환합니다
func (s *MetricsServer) handleWorkers(w http.ResponseWriter, r *http.Request) {
	globalMetricsLock.Lock()
	metrics := globalCurrentMetrics
	globalMetricsLock.Unlock()

	if metrics == nil {
		http.Error(w, "메트릭스 데이터가 아직 수집되지 않았습니다", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(metrics.User.Workers)
}

// handleCalculations는 포인트 계산 및 효율성 계산 데이터를 JSON으로 반환합니다
func (s *MetricsServer) handleCalculations(w http.ResponseWriter, r *http.Request) {
	globalMetricsLock.Lock()
	metrics := globalCurrentMetrics
	globalMetricsLock.Unlock()

	if metrics == nil {
		http.Error(w, "메트릭스 데이터가 아직 수집되지 않았습니다", http.StatusNotFound)
		return
	}

	// 포인트 계산 (나눗셈 수치별)
	// 실제 사용되는 값: myPoints := float64(metrics.User.TokensLast24Hours) / 10000
	userTokens := metrics.User.TokensLast24Hours
	generalTokens := metrics.General.TokensLast24Hours

	// 효율성 계산
	vastaiEfficiency := 0.0
	kuzcoEfficiency := 0.0
	if metrics.User.Share > 0 {
		vastaiEfficiency = metrics.User.VastaiDailyCost / (metrics.User.Share * 100)
		kuzcoEfficiency = metrics.User.KuzcoDailyCost / (metrics.User.Share * 100)
	}

	// 포맷팅 샘플
	myPointsAt1000 := float64(userTokens) / 1000
	myPointsAt10000 := float64(userTokens) / 10000
	myPointsAt100000 := float64(userTokens) / 100000

	totalPointsAt1000 := float64(generalTokens) / 1000
	totalPointsAt10000 := float64(generalTokens) / 10000
	totalPointsAt100000 := float64(generalTokens) / 100000

	// 모든 가능한 나눗셈 값에 대한 포맷 결과
	result := map[string]interface{}{
		"raw_values": map[string]interface{}{
			"user_tokens_last_24h":    userTokens,
			"general_tokens_last_24h": generalTokens,
			"vastai_daily_cost":       metrics.User.VastaiDailyCost,
			"kuzco_daily_cost":        metrics.User.KuzcoDailyCost,
			"share_percentage":        metrics.User.Share * 100,
		},
		"points_calculation": map[string]interface{}{
			"at_1000_division": map[string]interface{}{
				"my_points_raw":          myPointsAt1000,
				"my_points_formatted":    formatNumber(myPointsAt1000),
				"total_points_raw":       totalPointsAt1000,
				"total_points_formatted": formatNumber(totalPointsAt1000),
			},
			"at_10000_division": map[string]interface{}{
				"my_points_raw":          myPointsAt10000,
				"my_points_formatted":    formatNumber(myPointsAt10000),
				"total_points_raw":       totalPointsAt10000,
				"total_points_formatted": formatNumber(totalPointsAt10000),
			},
			"at_100000_division": map[string]interface{}{
				"my_points_raw":          myPointsAt100000,
				"my_points_formatted":    formatNumber(myPointsAt100000),
				"total_points_raw":       totalPointsAt100000,
				"total_points_formatted": formatNumber(totalPointsAt100000),
			},
		},
		"efficiency_calculation": map[string]interface{}{
			"vastai_efficiency":         vastaiEfficiency,
			"kuzco_efficiency":          kuzcoEfficiency,
			"vastai_efficiency_integer": int(vastaiEfficiency),
			"kuzco_efficiency_integer":  int(kuzcoEfficiency),
		},
		"sample_messages": map[string]interface{}{
			"at_1000_division": fmt.Sprintf("포인트 : %s | %s\n비중 : %.1f%%\n비용(vast,kuzco) : $%.2f | $%.2f\n1%% 효율(vast,kuzco) : $%d | $%d",
				formatNumber(myPointsAt1000),
				formatNumber(totalPointsAt1000),
				metrics.User.Share*100,
				metrics.User.VastaiDailyCost,
				metrics.User.KuzcoDailyCost,
				int(vastaiEfficiency),
				int(kuzcoEfficiency)),
			"at_10000_division": fmt.Sprintf("포인트 : %s | %s\n비중 : %.1f%%\n비용(vast,kuzco) : $%.2f | $%.2f\n1%% 효율(vast,kuzco) : $%d | $%d",
				formatNumber(myPointsAt10000),
				formatNumber(totalPointsAt10000),
				metrics.User.Share*100,
				metrics.User.VastaiDailyCost,
				metrics.User.KuzcoDailyCost,
				int(vastaiEfficiency),
				int(kuzcoEfficiency)),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(result)
}

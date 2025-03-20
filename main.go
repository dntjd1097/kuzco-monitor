package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"test/api"
	"test/config"
	"test/telegram"
	"time"

	"github.com/joho/godotenv"
)

var (
	currentMetrics *api.MinuteMetrics
	metricsLock    sync.Mutex
)

// updateCurrentMetrics safely updates the current metrics
func updateCurrentMetrics(mm api.MinuteMetrics) {
	log.Printf("Updating current metrics")
	metricsLock.Lock()
	defer metricsLock.Unlock()
	currentMetrics = &mm
	log.Printf("Current metrics updated")

	// 개발 모드일 때만 API 서버로 메트릭스 데이터 전달
	if os.Getenv("ENV") == "dev" {
		api.UpdateMetrics(mm)
	}
}

// getCurrentMetrics safely retrieves the current metrics
func getCurrentMetrics() *api.MinuteMetrics {
	log.Printf("Getting current metrics")
	metricsLock.Lock()
	defer metricsLock.Unlock()
	if currentMetrics == nil {
		log.Printf("No metrics available")
		return nil
	}
	log.Printf("Retrieved current metrics")
	return currentMetrics
}

// formatHourlyStats formats hourly statistics into a message string
func formatHourlyStats(stats api.HourlyStats) string {
	return fmt.Sprintf("시간별 통계 (%s ~ %s)\n\n"+
		"RPM:\n"+
		"  최소: %d\n"+
		"  최대: %d\n"+
		"  평균: %.0f\n"+
		"  현재: %d\n\n"+
		"인스턴스 수:\n"+
		"  최소: %d\n"+
		"  최대: %d\n"+
		"  평균: %.0f\n"+
		"  현재: %d\n\n"+
		"생성량:\n"+
		"  전체: %d\n"+
		"  사용자: %d\n"+
		"  비율: %.2f%%",
		stats.StartTime.Format("15:04:05"),
		stats.EndTime.Format("15:04:05"),
		stats.RPM.Min,
		stats.RPM.Max,
		stats.RPM.Avg,
		stats.RPM.Current,
		stats.TotalInstances.Min,
		stats.TotalInstances.Max,
		stats.TotalInstances.Avg,
		stats.TotalInstances.Current,
		stats.GenerationLastHour.General,
		stats.GenerationLastHour.User,
		stats.GenerationLastHour.Ratio)
}

// formatReport formats the report message
func formatReport(metrics *api.MinuteMetrics) string {
	vastaiEfficiency := 0.0
	kuzcoEfficiency := 0.0
	if metrics.User.Share > 0 {
		vastaiEfficiency = metrics.User.VastaiDailyCost / (metrics.User.Share * 100)
		kuzcoEfficiency = metrics.User.KuzcoDailyCost / (metrics.User.Share * 100)
	}

	// 포인트 값 먼저 1000으로 나누기 (소수점 조정)
	myPoints := float64(metrics.User.TokensLast24Hours) / 10000
	totalPoints := float64(metrics.General.TokensLast24Hours) / 10000

	// 적절한 단위 결정 (K, M, B)
	myPointsFormatted := formatNumber(myPoints)
	totalPointsFormatted := formatNumber(totalPoints)

	message := fmt.Sprintf("포인트 : %s | %s\n비중 : %.3f%%\n비용(vast,kuzco) : $%.2f | $%.2f\n1%% 효율(vast,kuzco) : $%d | $%d",
		myPointsFormatted,
		totalPointsFormatted,
		metrics.User.Share*100,
		metrics.User.VastaiDailyCost,
		metrics.User.KuzcoDailyCost,
		int(vastaiEfficiency),
		int(kuzcoEfficiency))

	if metrics.User.VastaiCredit != nil {
		message += fmt.Sprintf("\n잔액 : $%.2f", metrics.User.VastaiCredit.Credit)
	}

	return message
}

// formatNumber 함수 추가: 숫자를 K, M, B 단위로 자동 변환
func formatNumber(num float64) string {
	if num >= 1000000000 {
		return fmt.Sprintf("%.2fB", num/1000000000)
	} else if num >= 1000000 {
		return fmt.Sprintf("%.2fM", num/1000000)
	} else if num >= 1000 {
		return fmt.Sprintf("%.2fK", num/1000)
	}
	return fmt.Sprintf("%.2f", num)
}

// handleTelegramCommand processes telegram bot commands
func handleTelegramCommand(update telegram.Update, telegramClient *telegram.Client, cfg *config.Config) error {
	command := strings.TrimSpace(update.Message.Text)
	log.Printf("Processing command: %s", command)

	// /report 명령어는 최신 데이터를 가져옵니다
	if command == "/report" {
		log.Printf("Generating fresh report")

		// 계정 정보 가져오기 (첫 번째 계정 사용)
		if len(cfg.Accounts) == 0 {
			return telegramClient.SendMessage(update.Message.MessageThreadID, "계정 정보가 없습니다.")
		}

		account := cfg.Accounts[0]
		client := api.NewClient()

		// 로그인
		token, userID, err := client.Login(account.Kuzco.Email, account.Kuzco.Password)
		if err != nil {
			log.Printf("Login failed: %v", err)
			return telegramClient.SendMessage(update.Message.MessageThreadID, "로그인 실패: "+err.Error())
		}

		client.SetToken(token)

		// 최신 메트릭스 수집
		kuzcoClient := api.NewKuzcoClient(client)
		metrics, err := kuzcoClient.GetAllMetrics(userID)
		if err != nil {
			log.Printf("Failed to get metrics: %v", err)
			return telegramClient.SendMessage(update.Message.MessageThreadID, "메트릭스 수집 실패: "+err.Error())
		}

		// Vastai 정보 가져오기 (활성화된 경우)
		var vastaiCredit *api.VastaiCredit
		var vastaiCost float64

		if account.Vastai.Enabled {
			vastaiClient := api.NewVastaiClient(account.Vastai.Token)

			// 크레딧 정보 가져오기
			credit, err := vastaiClient.GetCredit()
			if err != nil {
				log.Printf("Failed to get vastai credit: %v", err)
			} else {
				vastaiCredit = credit
			}

			// 비용 정보 가져오기 (포함하도록 설정된 경우)
			if account.Vastai.IncludeVastaiCost {
				cost, err := vastaiClient.GetDailyCost()
				if err != nil {
					log.Printf("Failed to get vastai cost: %v", err)
				} else {
					vastaiCost = cost
				}
			}
		}

		// 효율성 계산
		vastaiEfficiency := 0.0
		kuzcoEfficiency := 0.0
		if metrics.User.Share > 0 {
			vastaiEfficiency = vastaiCost / (metrics.User.Share * 100)
			kuzcoEfficiency = metrics.User.TotalDailyCost / (metrics.User.Share * 100)
		}

		// 포인트 값 먼저 1000으로 나누기 (소수점 조정)
		myPoints := float64(metrics.User.TokensLast24Hours) / 10000
		totalPoints := float64(metrics.General.TokensLast24Hours) / 10000

		// 적절한 단위 결정 (K, M, B)
		myPointsFormatted := formatNumber(myPoints)
		totalPointsFormatted := formatNumber(totalPoints)

		// 응답 메시지 생성
		response := fmt.Sprintf("포인트 : %s | %s\n비중 : %.3f%%\n비용(vast,kuzco) : $%.2f | $%.2f\n1%% 효율(vast,kuzco) : $%d | $%d",
			myPointsFormatted,
			totalPointsFormatted,
			metrics.User.Share*100,
			vastaiCost,
			metrics.User.TotalDailyCost,
			int(vastaiEfficiency),
			int(kuzcoEfficiency))

		// Vastai 크레딧 정보 추가
		if vastaiCredit != nil {
			response += fmt.Sprintf("\n잔액 : $%.2f", vastaiCredit.Credit)
		}

		return telegramClient.SendMessage(update.Message.MessageThreadID, response)
	}

	// 다른 명령어는 캐시된 메트릭스 사용
	metrics := getCurrentMetrics()
	if metrics == nil {
		log.Printf("[ERROR] No metrics available for command: %s", command)
		response := "No metrics available. \nPlease wait a moment."
		return telegramClient.SendMessage(update.Message.MessageThreadID, response)
	}

	var response string

	switch command {
	case "/help":
		log.Printf("Generating help message")
		response = "사용 가능한 명령어:\n\n" +
			"`/help` - 이 도움말을 표시합니다\n" +
			"`/balance` - Vast.ai 잔액을 표시합니다\n" +
			"`/status` - 인스턴스 상태를 표시합니다\n" +
			"`/report` - 상세 리포트를 표시합니다\n" +
			"`/cost` - Vast.ai와 Kuzco의 일일 비용과 잔액을 표시합니다\n" +
			"`/hourly` - 지난 1시간 동안의 통계를 표시합니다\n" +
			"`/workers` - 워커별 시간당 생성량을 표시합니다"

	case "/balance":
		log.Printf("Checking balance")
		if metrics.User.VastaiCredit != nil {
			response = fmt.Sprintf("Balance : `$%.2f`", metrics.User.VastaiCredit.Credit)
			log.Printf("Balance: $%.2f", metrics.User.VastaiCredit.Credit)
		} else {
			response = "Balance information not available"
			log.Printf("Balance information not available")
		}

	case "/status":
		log.Printf("Checking status")
		response = fmt.Sprintf("Vast.Ai  : %d\nActual Instances : %d",
			metrics.User.TotalInstances,
			metrics.User.ActualTotalInstances)
		log.Printf("Status - Vast.Ai: %d, Actual Instances: %d",
			metrics.User.TotalInstances,
			metrics.User.ActualTotalInstances)

	case "/cost":
		log.Printf("Calculating costs")
		response = fmt.Sprintf("Kuzco 일일 비용: `$%.2f`", metrics.User.KuzcoDailyCost)
		log.Printf("Kuzco daily cost: $%.2f", metrics.User.KuzcoDailyCost)

		if metrics.User.VastaiCredit != nil {
			response += fmt.Sprintf("\nVast.ai 일일 비용: `$%.2f`", metrics.User.VastaiDailyCost)
			response += fmt.Sprintf("\n잔액: `$%.2f`", metrics.User.VastaiCredit.Credit)
			log.Printf("Vast.ai daily cost: $%.2f, Credit: $%.2f",
				metrics.User.VastaiDailyCost,
				metrics.User.VastaiCredit.Credit)

			if metrics.User.VastaiCredit.Credit <= metrics.User.VastaiDailyCost {
				response += fmt.Sprintf("\n⚠️ 잔액이 일일 비용보다 적습니다!")
				log.Printf("WARNING: Credit is less than daily cost")
			}

			if metrics.User.VastaiDailyCost > 0 {
				daysLeft := metrics.User.VastaiCredit.Credit / metrics.User.VastaiDailyCost
				response += fmt.Sprintf("\n예상 가능 사용일: %.1f일", daysLeft)
				log.Printf("Estimated days left: %.1f", daysLeft)
			}
		} else {
			log.Printf("Vast.ai credit information not available")
		}

	case "/hourly":
		log.Printf("Getting hourly stats")
		stats := api.GlobalHourlyStats.GetStats()
		response = formatHourlyStats(stats)
		log.Printf("Hourly stats generated")

	case "/workers":
		log.Printf("Getting worker stats")
		response = formatWorkerStats(metrics)

		// 추가 페이지가 있는지 확인
		if strings.Contains(response, "$$$") {
			parts := strings.Split(response, "$$$")
			response = parts[0] // 첫 번째 페이지 내용

			// 첫 번째 페이지 전송
			if err := telegramClient.SendMessage(update.Message.MessageThreadID, response); err != nil {
				log.Printf("Error sending first worker page: %v", err)
				return err
			}

			// 추가 페이지가 있으면 JSON에서 파싱
			if len(parts) > 1 {
				var workerPages struct {
					Pages []string `json:"pages"`
				}

				if err := json.Unmarshal([]byte(parts[1]), &workerPages); err != nil {
					log.Printf("Error parsing worker pages: %v", err)
				} else {
					// 각 추가 페이지를 순차적으로 전송 (0.5초 딜레이)
					for i, page := range workerPages.Pages {
						time.Sleep(500 * time.Millisecond) // 0.5초 딜레이로 순서 보장
						if err := telegramClient.SendMessage(update.Message.MessageThreadID, page); err != nil {
							log.Printf("Error sending worker page %d: %v", i+2, err)
						}
					}
				}
			}

			// 이미 메시지를 보냈으므로 빈 문자열로 설정
			response = ""
		}

		log.Printf("Worker stats generated")

	default:
		log.Printf("Unknown command: %s", command)
		return nil
	}

	log.Printf("Sending response to thread %d", update.Message.MessageThreadID)
	return telegramClient.SendMessage(update.Message.MessageThreadID, response)
}

// startTelegramBot starts the telegram bot and listens for updates
func startTelegramBot(telegramClient *telegram.Client, cfg *config.Config) {
	log.Printf("Starting Telegram bot...")
	offset := 0
	for {
		updates, err := telegramClient.GetUpdates(offset)
		if err != nil {
			log.Printf("[ERROR] Failed to get updates: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			log.Printf("Received command: %s in thread %d", update.Message.Text, update.Message.MessageThreadID)
			if err := handleTelegramCommand(update, telegramClient, cfg); err != nil {
				log.Printf("[ERROR] Failed to handle command '%s': %v", update.Message.Text, err)
			} else {
				log.Printf("Successfully handled command: %s", update.Message.Text)
			}
			offset = update.UpdateID + 1
		}

		time.Sleep(1 * time.Second)
	}
}

// startHourlyReporter starts the automatic hourly report sender
func startHourlyReporter(telegramClient *telegram.Client, cfg *config.Config) {
	log.Printf("Starting hourly reporter...")

	// 개발 모드 체크
	isDev := os.Getenv("ENV") == "dev"

	// 타이머 간격 설정
	var reportInterval time.Duration
	var initialDelay time.Duration

	if isDev {
		// 개발 모드에서는 2분 간격으로 보고서 전송
		reportInterval = 2 * time.Minute
		// 다음 짝수 분(0, 2, 4...)에 맞춰 시작
		now := time.Now()
		nextEvenMinute := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), ((now.Minute()/2)+1)*2, 0, 0, now.Location())
		initialDelay = nextEvenMinute.Sub(now)
		log.Printf("개발 모드: 첫 시간별 보고서 %s 후 전송, 이후 %s 간격으로 전송", initialDelay, reportInterval)
	} else {
		// 프로덕션 모드에서는 1시간 간격으로 전송
		reportInterval = time.Hour
		// 다음 정시(00분)에 맞춰 시작
		now := time.Now()
		nextHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
		initialDelay = nextHour.Sub(now)
		log.Printf("프로덕션 모드: 첫 시간별 보고서 %s 후 전송(정시), 이후 1시간 간격으로 전송", initialDelay)
	}

	// 초기 지연 후 첫 보고서 전송
	time.Sleep(initialDelay)

	// 첫 보고서 전송
	log.Printf("시간별 통계 조회 중...")
	stats := api.GlobalHourlyStats.GetStats()
	message := formatHourlyStats(stats)

	log.Printf("시간별 보고서 스레드 %d로 전송 중...", cfg.Telegram.Threads.Hourly)
	if err := telegramClient.SendMessage(cfg.Telegram.Threads.Hourly, message); err != nil {
		log.Printf("[ERROR] 시간별 보고서 전송 실패: %v", err)
	} else {
		log.Printf("시간별 보고서 전송 완료")
	}

	// 워커 보고서도 함께 전송
	sendWorkerReport(telegramClient, cfg)

	// 이후 정기적으로 보고서 전송
	ticker := time.NewTicker(reportInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		log.Printf("시간별 통계 조회 중...")
		stats := api.GlobalHourlyStats.GetStats()
		message := formatHourlyStats(stats)

		log.Printf("시간별 보고서 스레드 %d로 전송 중...", cfg.Telegram.Threads.Hourly)
		if err := telegramClient.SendMessage(cfg.Telegram.Threads.Hourly, message); err != nil {
			log.Printf("[ERROR] 시간별 보고서 전송 실패: %v", err)
		} else {
			log.Printf("시간별 보고서 전송 완료")
		}

		// 워커 보고서도 함께 전송
		sendWorkerReport(telegramClient, cfg)
	}
}

// sendWorkerReport 함수는 워커 보고서를 생성하고 전송합니다
func sendWorkerReport(telegramClient *telegram.Client, cfg *config.Config) {
	log.Printf("시간별 워커 보고서 생성 중...")
	metrics := getCurrentMetrics()
	if metrics != nil {
		workerReport := formatWorkerStats(metrics)

		// 추가 페이지 처리
		if strings.Contains(workerReport, "$$$") {
			parts := strings.Split(workerReport, "$$$")
			firstPage := parts[0]

			// 첫 번째 페이지 전송
			if err := telegramClient.SendMessage(cfg.Telegram.Threads.Workers, firstPage); err != nil {
				log.Printf("[ERROR] 시간별 워커 보고서(첫 페이지) 전송 실패: %v", err)
			} else {
				log.Printf("시간별 워커 보고서(첫 페이지) 전송 완료")
			}

			// 추가 페이지 처리
			if len(parts) > 1 {
				var workerPages struct {
					Pages []string `json:"pages"`
				}

				if err := json.Unmarshal([]byte(parts[1]), &workerPages); err != nil {
					log.Printf("워커 페이지 파싱 오류: %v", err)
				} else {
					// 각 추가 페이지를 순차적으로 전송 (0.5초 딜레이)
					for i, page := range workerPages.Pages {
						time.Sleep(500 * time.Millisecond) // 0.5초 딜레이로 순서 보장
						if err := telegramClient.SendMessage(cfg.Telegram.Threads.Workers, page); err != nil {
							log.Printf("워커 페이지 %d 전송 오류: %v", i+2, err)
						} else {
							log.Printf("시간별 워커 보고서(페이지 %d) 전송 완료", i+2)
						}
					}
				}
			}
		} else {
			// 단일 페이지 전송
			if err := telegramClient.SendMessage(cfg.Telegram.Threads.Workers, workerReport); err != nil {
				log.Printf("[ERROR] 시간별 워커 보고서 전송 실패: %v", err)
			} else {
				log.Printf("시간별 워커 보고서 전송 완료")
			}
		}
	} else {
		log.Printf("[ERROR] 시간별 워커 보고서용 메트릭스가 없습니다")
	}
}

// startInstanceMonitoring starts the instance monitoring loop
func startInstanceMonitoring(vastaiClient *api.VastaiClient, sendAlert func(string, string) error) {
	log.Printf("Starting instance monitoring service...")
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	monitoringCount := 0
	lastSuccessTime := time.Now()

	for {
		monitoringCount++
		currentTime := time.Now()
		log.Printf("=== Instance Monitoring Cycle #%d ===", monitoringCount)
		log.Printf("Last successful check: %s (%.1f minutes ago)",
			lastSuccessTime.Format("15:04:05"),
			currentTime.Sub(lastSuccessTime).Minutes())

		log.Printf("Starting instance status check...")
		if err := vastaiClient.MonitorAndRebootInstances(sendAlert); err != nil {
			log.Printf("[ERROR] Failed to monitor instances: %v", err)
			if sendAlert != nil {
				message := fmt.Sprintf("⚠️ Instance Monitoring Error\n시간: %s\n오류: %s",
					currentTime.Format("15:04:05"),
					err.Error())
				log.Printf("Sending error alert: %s", message)
				if err := sendAlert(message, "error"); err != nil {
					log.Printf("[ERROR] Failed to send monitoring error alert: %v", err)
				} else {
					log.Printf("Successfully sent error alert")
				}
			}
		} else {
			lastSuccessTime = currentTime
			log.Printf("Successfully completed instance monitoring at %s", currentTime.Format("15:04:05"))
		}

		nextCheckTime := currentTime.Add(5 * time.Minute)
		log.Printf("Next monitoring check scheduled for: %s", nextCheckTime.Format("15:04:05"))
		log.Printf("Waiting for next monitoring interval (5 minutes)...")
		log.Printf("=== End of Monitoring Cycle #%d ===\n", monitoringCount)
		<-ticker.C
	}
}

// formatWorkerStats 함수는 워커별 토큰당 수익을 포맷합니다
func formatWorkerStats(metrics *api.MinuteMetrics) string {
	// 워커 정보를 저장할 슬라이스
	type WorkerInfo struct {
		Name               string
		ModelType          []string // 모델 타입을
		GPU                []string // GPU 유형들
		Lane               []string // Lane 정보
		TokensPerInstance  int64
		GenerationsLast24H int
		GenerationLastHour int
		InstanceCount      int   // 인스턴스 개수 추가
		AvgTokens          int64 // 인스턴스당 평균 토큰
		AvgGenLastHour     int   // 인스턴스당 평균 시간당 생성량
		AvgGenLast24H      int   // 인스턴스당 평균 24시간 생성량
	}

	// 유효한 워커(토큰당 수익이 0이 아닌) 정보를 저장할 슬라이스
	workers := make([]WorkerInfo, 0, len(metrics.User.Workers))

	// 워커 정보 수집 - TokensPerInstance가 0인 워커는 제외
	for _, worker := range metrics.User.Workers {
		// 토큰당 수익이 0인 워커는 건너뜀
		if worker.TokensPerInstance <= 0 {
			continue
		}

		// 인스턴스 개수 확인
		instanceCount := worker.InstanceCount
		if instanceCount < 1 {
			instanceCount = 1 // 0으로 나누기 방지
		}

		// 인스턴스당 평균값 계산
		avgTokens := worker.TokensPerInstance
		avgGenLastHour := worker.GenerationLastHour / instanceCount
		avgGenLast24H := worker.GenerationsLast24H / instanceCount

		// 각 인스턴스의 고유 모델, GPU, Lane 유형 수집
		uniqueModels := make(map[string]bool)
		uniqueGPUs := make(map[string]bool)
		uniqueLanes := make(map[string]bool)

		// 인스턴스별 정보 추출
		for _, inst := range worker.Instances {
			// 모델 정보 수집
			if inst.Model != "" {
				uniqueModels[inst.Model] = true
			}

			// GPU 정보 수집 및 정리
			if inst.GPUModel != "" {
				// GPU 모델명 정리 (예: RTX 3060)
				gpuName := inst.GPUModel

				uniqueGPUs[gpuName] = true
			}

			// Lane 정보 수집
			if inst.Lane != "" {
				uniqueLanes[inst.Lane] = true
			}
		}

		// 유일한 모델 목록 생성
		var modelList []string
		for model := range uniqueModels {
			modelList = append(modelList, model)
		}

		// 유일한 GPU 목록 생성
		var gpuList []string
		for gpu := range uniqueGPUs {
			gpuList = append(gpuList, gpu)
		}

		// 유일한 Lane 목록 생성
		var laneList []string
		for lane := range uniqueLanes {
			laneList = append(laneList, lane)
		}

		info := WorkerInfo{
			Name:               worker.Name,
			ModelType:          modelList,
			GPU:                gpuList,
			Lane:               laneList,
			TokensPerInstance:  worker.TokensPerInstance,
			GenerationsLast24H: worker.GenerationsLast24H,
			GenerationLastHour: worker.GenerationLastHour,
			InstanceCount:      instanceCount,
			AvgTokens:          avgTokens,
			AvgGenLastHour:     avgGenLastHour,
			AvgGenLast24H:      avgGenLast24H,
		}

		workers = append(workers, info)
	}

	// 토큰당 수익 기준으로 내림차순 정렬
	sort.Slice(workers, func(i, j int) bool {
		return workers[i].TokensPerInstance > workers[j].TokensPerInstance
	})

	// 총 워커 수와 전체 생성량 계산
	totalWorkers := len(workers)
	if totalWorkers == 0 {
		return "🖥️ 토큰당 수익이 있는 워커가 없습니다."
	}

	totalGenerations := 0
	totalGenerationsLast24H := 0
	totalInstances := 0
	for _, w := range workers {
		totalGenerations += w.GenerationLastHour
		totalGenerationsLast24H += w.GenerationsLast24H
		totalInstances += w.InstanceCount
	}

	// 전체 인스턴스당 평균 생성량 계산
	var avgGenerationPerInstance, avgGeneration24HPerInstance int
	if totalInstances > 0 {
		avgGenerationPerInstance = totalGenerations / totalInstances
		avgGeneration24HPerInstance = totalGenerationsLast24H / totalInstances
	}

	// 결과 메시지 생성
	var messageBuilder strings.Builder

	// 헤더 메시지 생성
	messageBuilder.WriteString(fmt.Sprintf("📊 워커 현황 요약 (%d개 워커/%d개 인스턴스)\n", totalWorkers, totalInstances))
	messageBuilder.WriteString(fmt.Sprintf("• 총 생성량: %d/시간 | %d/24시간\n", totalGenerations, totalGenerationsLast24H))
	messageBuilder.WriteString(fmt.Sprintf("• 인스턴스당 평균: %d/시간 | %d/24시간\n\n", avgGenerationPerInstance, avgGeneration24HPerInstance))

	// 헤더 구분선
	messageBuilder.WriteString("-----------------------------------------------------------------------\n")
	messageBuilder.WriteString("  R  | 워커 | I |  토큰/I    | 1hG/I | 모델 | GPU | Lane\n")
	messageBuilder.WriteString("-----------------------------------------------------------------------\n")

	// 모든 워커 정보를 한꺼번에 표시
	for i, w := range workers {
		// 모델 타입에 따라 아이콘 선택
		modelType := "일반"
		if len(w.ModelType) > 0 {
			// 모델 타입 단순화
			simplifiedModels := make([]string, 0, len(w.ModelType))
			for _, model := range w.ModelType {
				simpleModel := "기타"
				if strings.Contains(strings.ToLower(model), "vllm") {
					simpleModel = "VL"
				} else if strings.Contains(strings.ToLower(model), "ollama") {
					simpleModel = "Ol"
				} else if strings.Contains(strings.ToLower(model), "sglang") {
					simpleModel = "SG"
				}
				simplifiedModels = append(simplifiedModels, simpleModel)
			}

			// 중복 제거
			uniqueModels := make(map[string]bool)
			for _, m := range simplifiedModels {
				uniqueModels[m] = true
			}

			var modelsList []string
			for m := range uniqueModels {
				modelsList = append(modelsList, m)
			}

			modelType = strings.Join(modelsList, ",")
		}

		// GPU 목록 처리
		gpuInfo := "N/A"
		if len(w.GPU) > 0 {
			gpuInfo = strings.Join(w.GPU, ",")
		}

		// Lane 정보 처리
		laneInfo := "N/A"
		if len(w.Lane) > 0 {
			laneInfo = strings.Join(w.Lane, ",")
		}

		// 토큰당 수익 포맷팅
		tokensFormatted := formatNumber(float64(w.TokensPerInstance))

		// 1시간 생성량/인스턴스 사용
		genPerInstance := w.AvgGenLastHour

		// 워커 이름 추출 (v숫자만 남기기)
		workerName := w.Name

		// GPU 모델 추출 - 3060 등의 숫자만
		// gpuModel := w.GPU

		// 순위에 따라 들여쓰기 수준 조정
		rankStr := fmt.Sprintf("%3d", i+1)

		// 표시할 행 생성 (요청된 형식으로)
		messageBuilder.WriteString(fmt.Sprintf(" %-4s | %-5s | %1d | %-11s | %5d | %-5s | %-8s | %s\n",
			rankStr,
			workerName,
			w.InstanceCount,
			tokensFormatted,
			genPerInstance,
			modelType,
			gpuInfo,
			laneInfo))
	}

	return messageBuilder.String()
}

// startDailyWorkerReporter는 매일 워커 현황을 전송합니다
func startDailyWorkerReporter(telegramClient *telegram.Client, cfg *config.Config) {
	log.Printf("Starting daily worker reporter...")

	// 개발 모드 체크
	isDev := os.Getenv("ENV") == "dev"

	// 타이머 간격 설정
	var initialDelay time.Duration

	if isDev {
		// 개발 모드에서는 20초 후에 첫 보고서 전송, 그 후 1분 간격으로 전송
		initialDelay = 20 * time.Second
		log.Printf("개발 모드: %s 후 첫 워커 보고서 전송, 이후 1분 간격으로 전송", initialDelay)
	} else {
		// 프로덕션 모드에서는 매일 오전 9시에 전송
		now := time.Now()
		nextReport := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.Local)
		if now.After(nextReport) {
			nextReport = nextReport.Add(24 * time.Hour)
		}
		initialDelay = nextReport.Sub(now)
		log.Printf("다음 워커 보고서 예정 시간: %s", nextReport.Format("2006-01-02 15:04:05"))
	}

	timer := time.NewTimer(initialDelay)
	defer timer.Stop()

	for {
		<-timer.C
		log.Printf("워커 보고서 생성 중...")

		// 현재 메트릭스 가져오기
		metrics := getCurrentMetrics()
		if metrics == nil {
			log.Printf("[ERROR] 워커 보고서용 메트릭스가 없습니다")
			// 메트릭스가 없는 경우 1시간 후 다시 시도 (개발 모드에서는 30초 후)
			if isDev {
				log.Printf("개발 모드: 30초 후 다시 시도")
				timer.Reset(30 * time.Second)
			} else {
				timer.Reset(time.Hour)
			}
			continue
		}

		// 워커 보고서 생성
		workerReport := formatWorkerStats(metrics)

		// 추가 페이지 처리
		if strings.Contains(workerReport, "$$$") {
			parts := strings.Split(workerReport, "$$$")
			firstPage := parts[0]

			// 첫 번째 페이지 전송
			if err := telegramClient.SendMessage(cfg.Telegram.Threads.Workers, firstPage); err != nil {
				log.Printf("[ERROR] 워커 보고서(첫 페이지) 전송 실패: %v", err)
			} else {
				log.Printf("워커 보고서(첫 페이지) 전송 완료")
			}

			// 추가 페이지 처리
			if len(parts) > 1 {
				var workerPages struct {
					Pages []string `json:"pages"`
				}

				if err := json.Unmarshal([]byte(parts[1]), &workerPages); err != nil {
					log.Printf("워커 페이지 파싱 오류: %v", err)
				} else {
					// 각 추가 페이지를 순차적으로 전송 (0.5초 딜레이)
					for i, page := range workerPages.Pages {
						time.Sleep(500 * time.Millisecond) // 0.5초 딜레이로 순서 보장
						if err := telegramClient.SendMessage(cfg.Telegram.Threads.Workers, page); err != nil {
							log.Printf("워커 페이지 %d 전송 오류: %v", i+2, err)
						} else {
							log.Printf("워커 보고서(페이지 %d) 전송 완료", i+2)
						}
					}
				}
			}
		} else {
			// 단일 페이지 전송
			if err := telegramClient.SendMessage(cfg.Telegram.Threads.Workers, workerReport); err != nil {
				log.Printf("[ERROR] 워커 보고서 전송 실패: %v", err)
			} else {
				log.Printf("워커 보고서 전송 완료")
			}
		}

		// 다음 전송 시간 설정
		if isDev {
			// 개발 모드에서는 1분 후 다시 전송
			timer.Reset(1 * time.Minute)
			log.Printf("개발 모드: 다음 워커 보고서 %s 후 전송", 1*time.Minute)
		} else {
			// 프로덕션 모드에서는 다음 날 같은 시간
			timer.Reset(24 * time.Hour)
			log.Printf("다음 워커 보고서 예정 시간: %s", time.Now().Add(24*time.Hour).Format("2006-01-02 15:04:05"))
		}
	}
}

func main() {
	// Configure logging with timestamp, source file, and line number
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Printf("Starting Kuzco Monitor...")

	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	client := api.NewClient()
	telegramClient := telegram.NewClient(cfg.Telegram.Token, cfg.Telegram.ChatID)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 개발 모드일 때만 API 서버 시작
	isDev := os.Getenv("ENV") == "dev"
	if isDev {
		// API 서버 시작 (포트 8080)
		metricsServer := api.NewMetricsServer(8080)
		go metricsServer.Start()
		log.Printf("Metrics API server started on port 8080 (개발 모드)")
	}

	// Start telegram bot
	go startTelegramBot(telegramClient, cfg)

	// Start hourly reporter
	go startHourlyReporter(telegramClient, cfg)

	// Start daily worker reporter
	go startDailyWorkerReporter(telegramClient, cfg)

	for _, account := range cfg.Accounts {
		fmt.Printf("Starting metrics collection for account: %s\n", account.Name)

		token, userID, err := client.Login(account.Kuzco.Email, account.Kuzco.Password)
		if err != nil {
			log.Printf("Login failed for %s: %v", account.Name, err)
			continue
		}

		client.SetToken(token)

		dailyChan := make(chan api.DailyMetrics, 1)
		minuteChan := make(chan api.MinuteMetrics, 1)
		stopChan := make(chan struct{})

		sendAlert := func(message, alertType string) error {
			var threadID int
			switch alertType {
			case "daily":
				threadID = cfg.Telegram.Threads.Daily
			case "hourly":
				threadID = cfg.Telegram.Threads.Hourly
			case "error":
				threadID = cfg.Telegram.Threads.Error
			case "status":
				threadID = cfg.Telegram.Threads.Status
			case "worker":
				threadID = cfg.Telegram.Threads.Workers
			}
			return telegramClient.SendMessage(threadID, message)
		}

		var vastaiToken string
		var vastaiClient *api.VastaiClient
		if account.Vastai.Enabled {
			vastaiToken = account.Vastai.Token
			vastaiClient = api.NewVastaiClient(vastaiToken)
			// Start instance monitoring if Vast.ai is enabled
			go startInstanceMonitoring(vastaiClient, sendAlert)
		}

		go client.CollectMetrics(
			userID,
			vastaiToken,
			account.Vastai.IncludeVastaiCost,
			account.Alerts,
			sendAlert,
			dailyChan,
			minuteChan,
			stopChan,
		)

		go func(email string) {
			for {
				select {
				case <-dailyChan:
					fmt.Printf("Daily Metrics for %s:\n", email)
				case mm := <-minuteChan:
					fmt.Printf("Minute Metrics for %s:\n", email)
					updateCurrentMetrics(mm)
				}
			}
		}(account.Name)
	}

	<-sigChan
	fmt.Println("\nShutting down...")
}

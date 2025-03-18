package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
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

	message := fmt.Sprintf("포인트 : %s | %s\n비중 : %.1f%%\n비용(vast,kuzco) : $%.2f | $%.2f\n1%% 효율(vast,kuzco) : $%d | $%d",
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
		return fmt.Sprintf("%.1fB", num/1000000000)
	} else if num >= 1000000 {
		return fmt.Sprintf("%.1fM", num/1000000)
	} else if num >= 1000 {
		return fmt.Sprintf("%.1fK", num/1000)
	}
	return fmt.Sprintf("%.1f", num)
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
		response := fmt.Sprintf("포인트 : %s | %s\n비중 : %.1f%%\n비용(vast,kuzco) : $%.2f | $%.2f\n1%% 효율(vast,kuzco) : $%d | $%d",
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
			"`/hourly` - 지난 1시간 동안의 통계를 표시합니다"

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
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		log.Printf("Waiting for next hour to send report...")
		<-ticker.C
		log.Printf("Getting hourly stats...")
		stats := api.GlobalHourlyStats.GetStats()
		message := formatHourlyStats(stats)

		log.Printf("Sending hourly report to thread %d...", cfg.Telegram.Threads.Hourly)
		if err := telegramClient.SendMessage(cfg.Telegram.Threads.Hourly, message); err != nil {
			log.Printf("[ERROR] Failed to send hourly report: %v", err)
		} else {
			log.Printf("Successfully sent hourly report")
		}
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

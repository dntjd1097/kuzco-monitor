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
	metricsLock.Lock()
	defer metricsLock.Unlock()
	currentMetrics = &mm
}

// getCurrentMetrics safely retrieves the current metrics
func getCurrentMetrics() *api.MinuteMetrics {
	metricsLock.Lock()
	defer metricsLock.Unlock()
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
	efficiency := 0.0
	if metrics.User.Share > 0 {
		efficiency = metrics.User.TotalDailyCost / (metrics.User.Share * 100)
	}

	message := fmt.Sprintf("포인트 : %d\n비중 : %.2f%%\n비용 : $%.2f\n효율(1%%) : $%d",
		metrics.User.TokensLast24Hours,
		metrics.User.Share*100,
		metrics.User.TotalDailyCost,
		int(efficiency))

	if metrics.User.VastaiCredit != nil {
		message += fmt.Sprintf("\n잔액 : $%.2f", metrics.User.VastaiCredit.Credit)
	}

	return message
}

// handleTelegramCommand processes telegram bot commands
func handleTelegramCommand(update telegram.Update, telegramClient *telegram.Client, cfg *config.Config) error {
	metrics := getCurrentMetrics()
	if metrics == nil {
		response := "No metrics available. \n Please wait a moment."
		return telegramClient.SendMessage(update.Message.MessageThreadID, response)
	}

	command := strings.TrimSpace(update.Message.Text)
	var response string

	switch command {
	case "/help":
		response = "사용 가능한 명령어:\n\n" +
			"`/help` - 이 도움말을 표시합니다\n" +
			"`/balance` - Vast.ai 잔액을 표시합니다\n" +
			"`/status` - 인스턴스 상태를 표시합니다\n" +
			"`/report` - 상세 리포트를 표시합니다\n" +
			"`/hourly` - 지난 1시간 동안의 통계를 표시합니다\n" +
			"  • RPM 최소/최대/평균\n" +
			"  • 인스턴스 수 최소/최대/평균\n" +
			"  • 생성량 전체/사용자/비율"

	case "/balance":
		if metrics.User.VastaiCredit != nil {
			response = fmt.Sprintf("Balance : `$%.2f`", metrics.User.VastaiCredit.Credit)
		} else {
			response = "Balance information not available"
		}

	case "/status":
		response = fmt.Sprintf("Vast.Ai  : %d\nActual Instances : %d",
			metrics.User.TotalInstances,
			metrics.User.ActualTotalInstances)

	case "/report":
		response = formatReport(metrics)

	case "/hourly":
		stats := api.GlobalHourlyStats.GetStats()
		response = formatHourlyStats(stats)

	default:
		return nil
	}

	return telegramClient.SendMessage(update.Message.MessageThreadID, response)
}

// startTelegramBot starts the telegram bot and listens for updates
func startTelegramBot(telegramClient *telegram.Client, cfg *config.Config) {
	offset := 0
	for {
		updates, err := telegramClient.GetUpdates(offset)
		if err != nil {
			log.Printf("Error getting updates: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			if err := handleTelegramCommand(update, telegramClient, cfg); err != nil {
				log.Printf("Error handling command: %v", err)
			}
			offset = update.UpdateID + 1
		}

		time.Sleep(1 * time.Second)
	}
}

// startHourlyReporter starts the automatic hourly report sender
func startHourlyReporter(telegramClient *telegram.Client, cfg *config.Config) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		stats := api.GlobalHourlyStats.GetStats()
		message := formatHourlyStats(stats)

		if err := telegramClient.SendMessage(cfg.Telegram.Threads.Hourly, message); err != nil {
			log.Printf("Error sending hourly report: %v", err)
		}

		<-ticker.C
	}
}

func main() {
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

	// Start telegram bot
	go startTelegramBot(telegramClient, cfg)

	// Start hourly reporter
	go startHourlyReporter(telegramClient, cfg)

	for _, account := range cfg.Accounts {
		fmt.Printf("\nStarting metrics collection for account: %s\n", account.Name)

		token, userID, err := client.Login(account.Kuzco.Email, account.Kuzco.Password)
		if err != nil {
			log.Printf("Login failed for %s: %v", account.Name, err)
			continue
		}

		client.SetToken(token)

		dailyChan := make(chan api.DailyMetrics, 1)
		minuteChan := make(chan api.MinuteMetrics, 1)
		stopChan := make(chan struct{})

		var vastaiToken string
		if account.Vastai.Enabled {
			vastaiToken = account.Vastai.Token
		}

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
					fmt.Printf("\nDaily Metrics for %s:\n", email)
				case mm := <-minuteChan:
					fmt.Printf("\nMinute Metrics for %s:\n", email)
					updateCurrentMetrics(mm)
				}
			}
		}(account.Name)
	}

	<-sigChan
	fmt.Println("\nShutting down...")
}

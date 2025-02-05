package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Monitor struct {
	service    *Service
	config     *Config
	state      *MonitorState
	token      string
	lastAuth   time.Time
	logMessage func(string)
}

// 텔레그램 업데이트 구조체 추가
type TelegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

type TelegramUpdates struct {
	Ok     bool             `json:"ok"`
	Result []TelegramUpdate `json:"result"`
}

func NewMonitor(config *Config, logMessage func(string)) *Monitor {
	m := &Monitor{
		config:     config,
		state:      &MonitorState{Workers: make(map[string]*WorkerStatus)},
		logMessage: logMessage,
	}
	m.service = NewService("", m.logMessage)
	return m
}

func (m *Monitor) refreshAuthIfNeeded() error {
	if time.Since(m.lastAuth) >= 1*time.Hour || m.token == "" {
		m.logMessage("🔄 Refreshing authentication token...")
		service := NewService("", m.logMessage)
		token, err := service.Login(m.config.Kuzco.ID, m.config.Kuzco.Password)
		if err != nil {
			m.logMessage(fmt.Sprintf("❌ Authentication failed: %v", err))
			return err
		}
		m.token = token
		m.service = NewService(token, m.logMessage)
		m.lastAuth = time.Now()
		m.logMessage("✅ Authentication refreshed successfully")
	}
	return nil
}

func (m *Monitor) sendTelegramMessage(message string, threadType string) error {
	// ChatID를 int64로 변환
	chatID, err := strconv.ParseInt(m.config.Telegram.ChatID, 10, 64)
	if err != nil {
		m.logMessage(fmt.Sprintf("❌ Error parsing chat ID: %v", err))
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", m.config.Telegram.BotToken)
	body := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	// MessageThread가 0이 아닌 경우에만 추가
	if m.config.Telegram.MessageThread != 0 {
		body["message_thread_id"] = m.config.Telegram.MessageThread
	}

	respBody, err := m.service.client.doRequest("POST", url, body)
	if err != nil {
		m.logMessage(fmt.Sprintf("❌ Failed to send telegram message: %v", err))
		return err
	}

	// 응답 확인
	var response struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		m.logMessage(fmt.Sprintf("❌ Error parsing telegram response: %v", err))
		return err
	}

	if !response.Ok {
		m.logMessage(fmt.Sprintf("❌ Telegram API error: %s", response.Description))
		return fmt.Errorf("telegram API error: %s", response.Description)
	}

	m.logMessage("✅ Message sent successfully")
	return nil
}

func (m *Monitor) checkWorkerChanges() error {
	workers, err := getActiveWorkers(m.token)
	if err != nil {
		return err
	}

	changes := []string{}

	for _, worker := range workers {
		currentStatus := &WorkerStatus{
			InstanceCount: len(worker.Instances),
			Instances:     make(map[string]string),
		}

		for _, instance := range worker.Instances {
			currentStatus.Instances[instance.ID] = instance.Status
		}

		prevStatus, exists := m.state.Workers[worker.ID]
		if !exists {
			// 새로운 워커 발견
			changes = append(changes, fmt.Sprintf("New worker detected: %s", worker.Name))
		} else {
			// 인스턴스 수 변경 확인
			if currentStatus.InstanceCount != prevStatus.InstanceCount {
				changes = append(changes, fmt.Sprintf("Worker %s instance count changed: %d -> %d",
					worker.Name, prevStatus.InstanceCount, currentStatus.InstanceCount))
			}

			// 상태 변경 확인
			for instanceID, currentState := range currentStatus.Instances {
				if prevState, ok := prevStatus.Instances[instanceID]; ok {
					if currentState != prevState {
						changes = append(changes, fmt.Sprintf("Instance %s status changed: %s -> %s",
							instanceID, prevState, currentState))
					}
				}
			}
		}

		m.state.Workers[worker.ID] = currentStatus
	}

	if len(changes) > 0 {
		message := "🔔 Worker Status Changes:\n" + strings.Join(changes, "\n")
		return m.sendTelegramMessage(message, "status")
	}

	return nil
}

func (m *Monitor) sendDailyReport() error {
	if err := m.refreshAuthIfNeeded(); err != nil {
		return err
	}

	// 서버 정보 수집
	count, err := getOnlineWorkers(m.token)
	if err != nil {
		return err
	}

	rpm, err := getServerRPM(m.token)
	if err != nil {
		return err
	}

	tokens24h, err := getTokensLast24Hours(m.token)
	if err != nil {
		return err
	}

	// 사용자 메트릭 수집
	userMetrics, err := getUserMetrics(m.token, "c9XhxeKcWRChV875-H7u3")
	if err != nil {
		return err
	}

	// 워커 정보 수집
	workers, err := getActiveWorkers(m.token)
	if err != nil {
		return err
	}

	// 리포트 생성
	report := strings.Builder{}
	report.WriteString("📊 Daily Report\n\n")
	report.WriteString(fmt.Sprintf("Server Status:\n"))
	report.WriteString(fmt.Sprintf("- Online Workers: %d\n", count))
	report.WriteString(fmt.Sprintf("- Server RPM: %d\n", rpm))
	report.WriteString(fmt.Sprintf("- Global Tokens (24h): %d\n\n", tokens24h))

	report.WriteString("User Metrics:\n")
	report.WriteString(fmt.Sprintf("- Generations (24h): %d\n", userMetrics.Result.Data.JSON.GenerationsLast24Hours))
	report.WriteString(fmt.Sprintf("- Tokens (24h): %d\n", userMetrics.Result.Data.JSON.TokensLast24Hours))
	report.WriteString(fmt.Sprintf("- Total Tokens: %d\n\n", userMetrics.Result.Data.JSON.TokensAllTime))

	report.WriteString("Active Workers:\n")
	for _, worker := range workers {
		if !worker.IsArchived {
			metrics, err := getWorkerMetrics(m.token, worker.ID)
			if err != nil {
				continue
			}

			report.WriteString(fmt.Sprintf("\n%s:\n", worker.Name))
			report.WriteString(fmt.Sprintf("- Generations (24h): %d\n", metrics.Result.Data.JSON.GenerationsLast24Hours))
			report.WriteString(fmt.Sprintf("- Tokens (24h): %d\n", metrics.Result.Data.JSON.TokensLast24Hours))
			report.WriteString(fmt.Sprintf("- Total Tokens: %d\n", metrics.Result.Data.JSON.TokensAllTime))
			report.WriteString("- Instances:\n")

			for _, instance := range worker.Instances {
				report.WriteString(fmt.Sprintf("  • %s (%s)\n", instance.Name, instance.Status))
				report.WriteString(fmt.Sprintf("    Location: %s, %s\n", instance.Info.City, instance.Info.Country))
			}
		}
	}

	// 리포트 생성 후 전송
	return m.sendTelegramMessage(report.String(), "daily")
}

// 에러 알림을 위한 새로운 메서드
func (m *Monitor) sendErrorAlert(err error) {
	message := fmt.Sprintf("⚠️ Error Alert:\n%v", err)
	if sendErr := m.sendTelegramMessage(message, "error"); sendErr != nil {
		fmt.Printf("Failed to send error alert: %v\n", sendErr)
	}
}

func (m *Monitor) handleCommands() {
	var lastUpdateID int
	for {
		updates, err := m.getUpdates(lastUpdateID + 1)
		if err != nil {
			fmt.Printf("Error getting updates: %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		for _, update := range updates.Result {
			if update.UpdateID >= lastUpdateID {
				lastUpdateID = update.UpdateID
			}

			// 커맨드 처리
			switch update.Message.Text {
			case "/status":
				m.handleStatusCommand()
			case "/report":
				m.handleReportCommand()
			case "/help":
				m.handleHelpCommand()
			}
		}

		time.Sleep(1 * time.Second)
	}
}

func (m *Monitor) getUpdates(offset int) (*TelegramUpdates, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", m.config.Telegram.BotToken)
	body := map[string]interface{}{
		"offset":  offset,
		"timeout": 30,
	}

	respBody, err := m.service.client.doRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	var updates TelegramUpdates
	if err := json.Unmarshal(respBody, &updates); err != nil {
		return nil, err
	}

	return &updates, nil
}

func (m *Monitor) handleStatusCommand() {
	if err := m.refreshAuthIfNeeded(); err != nil {
		m.sendErrorAlert(err)
		return
	}

	count, err := getOnlineWorkers(m.token)
	if err != nil {
		m.sendErrorAlert(err)
		return
	}

	workers, err := getActiveWorkers(m.token)
	if err != nil {
		m.sendErrorAlert(err)
		return
	}

	// 상태 메시지 생성
	status := strings.Builder{}
	status.WriteString("📊 Current Status\n\n")
	status.WriteString(fmt.Sprintf("Online Workers: %d\n\n", count))
	status.WriteString("Active Workers:\n")

	for _, worker := range workers {
		if !worker.IsArchived {
			status.WriteString(fmt.Sprintf("\n%s:\n", worker.Name))
			for _, instance := range worker.Instances {
				status.WriteString(fmt.Sprintf("  • %s (%s)\n", instance.Name, instance.Status))
				status.WriteString(fmt.Sprintf("    Location: %s, %s\n", instance.Info.City, instance.Info.Country))
			}
		}
	}

	m.sendTelegramMessage(status.String(), "status")
}

func (m *Monitor) handleReportCommand() {
	if err := m.sendDailyReport(); err != nil {
		m.sendErrorAlert(err)
	}
}

func (m *Monitor) handleHelpCommand() {
	help := `Available commands:
/status - Show current worker status
/report - Generate a full report
/help - Show this help message`

	m.sendTelegramMessage(help, "status")
}

func (m *Monitor) getChatID() error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", m.config.Telegram.BotToken)
	respBody, err := m.service.client.doRequest("GET", url, nil)
	if err != nil {
		m.logMessage(fmt.Sprintf("❌ Failed to get updates: %v", err))
		return err
	}

	var response struct {
		Ok     bool `json:"ok"`
		Result []struct {
			Message struct {
				Chat struct {
					ID int64 `json:"id"`
				} `json:"chat"`
			} `json:"message"`
		} `json:"result"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		m.logMessage(fmt.Sprintf("❌ Error parsing response: %v", err))
		return err
	}

	if response.Ok && len(response.Result) > 0 {
		chatID := response.Result[0].Message.Chat.ID
		m.logMessage(fmt.Sprintf("✅ Found chat ID: %d", chatID))
		// config.yaml 파일 업데이트
		m.config.Telegram.ChatID = strconv.FormatInt(chatID, 10)
		return nil
	}

	return fmt.Errorf("no chat found. Please send /start to the bot first")
}

func (m *Monitor) Start() {
	m.logMessage("Monitor starting...")

	// Chat ID 확인 및 업데이트
	if err := m.getChatID(); err != nil {
		m.logMessage(fmt.Sprintf("❌ Failed to get chat ID: %v", err))
		m.logMessage("Please follow these steps:")
		m.logMessage("1. Find your bot in Telegram: @" + strings.Split(m.config.Telegram.BotToken, ":")[0])
		m.logMessage("2. Send /start command to the bot")
		m.logMessage("3. Restart this application")
		return
	}

	// 초기 텔레그램 연결 테스트
	testMessage := "🚀 Monitor service started"
	if err := m.sendTelegramMessage(testMessage, "status"); err != nil {
		m.logMessage(fmt.Sprintf("❌ Initial telegram test failed: %v", err))
		// 텔레그램 전송 실패해도 계속 진행
	} else {
		m.logMessage("✅ Initial telegram test successful")
	}

	// 초기 인증
	if err := m.refreshAuthIfNeeded(); err != nil {
		m.logMessage(fmt.Sprintf("❌ Initial authentication failed: %v", err))
		m.sendErrorAlert(err)
		return
	}
	m.logMessage("✅ Initial authentication successful")

	// 커맨드 처리 고루틴 추가
	go func() {
		m.logMessage("Starting command handler")
		m.handleCommands()
	}()

	// 1분마다 워커 상태 체크
	go func() {
		m.logMessage("Starting worker status checker")
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			if err := m.refreshAuthIfNeeded(); err != nil {
				m.logMessage(fmt.Sprintf("❌ Auth refresh failed: %v", err))
				m.sendErrorAlert(err)
				continue
			}
			if err := m.checkWorkerChanges(); err != nil {
				m.logMessage(fmt.Sprintf("❌ Worker check failed: %v", err))
				m.sendErrorAlert(err)
			}
		}
	}()

	// 24시간마다 리포트 전송
	go func() {
		m.logMessage("Starting daily report scheduler")
		// 즉시 첫 리포트 전송
		if err := m.sendDailyReport(); err != nil {
			m.logMessage(fmt.Sprintf("❌ Initial daily report failed: %v", err))
			m.sendErrorAlert(err)
		}

		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			if err := m.sendDailyReport(); err != nil {
				m.logMessage(fmt.Sprintf("❌ Daily report failed: %v", err))
				m.sendErrorAlert(err)
			}
		}
	}()

	m.logMessage("✅ Monitor started successfully")
}

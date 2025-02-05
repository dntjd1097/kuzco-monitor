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
	currentWorkers := make(map[string]bool)

	for _, worker := range workers {
		currentWorkers[worker.ID] = true
		currentStatus := &WorkerStatus{
			Name:          worker.Name,
			InstanceCount: len(worker.Instances),
			Instances:     make(map[string]InstanceStatus),
		}

		for _, instance := range worker.Instances {
			gpuInfo := InstanceStatus{
				Name:      instance.Name,
				Status:    instance.Status,
				IPAddress: instance.Info.IPAddress,
				Runtime:   instance.Info.Runtime,
			}

			if instance.Info.NvidiaSmi != nil && len(instance.Info.NvidiaSmi.GPU) > 0 {
				gpu := instance.Info.NvidiaSmi.GPU[0]
				if len(gpu.ProductName) > 0 {
					gpuInfo.GPU.Name = gpu.ProductName[0]
				}
				if len(gpu.Temperature) > 0 && len(gpu.Temperature[0].GPUTemp) > 0 {
					gpuInfo.GPU.Temp = gpu.Temperature[0].GPUTemp[0]
				}
				if len(gpu.Utilization) > 0 && len(gpu.Utilization[0].GPUUtil) > 0 {
					gpuInfo.GPU.Utilization = gpu.Utilization[0].GPUUtil[0]
				}
				if len(gpu.FBMemoryUsage) > 0 {
					if len(gpu.FBMemoryUsage[0].Used) > 0 {
						gpuInfo.GPU.Memory.Used = gpu.FBMemoryUsage[0].Used[0]
					}
					if len(gpu.FBMemoryUsage[0].Total) > 0 {
						gpuInfo.GPU.Memory.Total = gpu.FBMemoryUsage[0].Total[0]
					}
				}
				if len(gpu.PowerReadings) > 0 {
					if len(gpu.PowerReadings[0].PowerDraw) > 0 {
						gpuInfo.GPU.Power.Draw = gpu.PowerReadings[0].PowerDraw[0]
					}
					if len(gpu.PowerReadings[0].PowerState) > 0 {
						gpuInfo.GPU.Power.State = gpu.PowerReadings[0].PowerState[0]
					}
				}
			}

			currentStatus.Instances[instance.ID] = gpuInfo
		}

		prevStatus, exists := m.state.Workers[worker.ID]
		if !exists {
			// 새로운 워커 발견
			changes = append(changes, fmt.Sprintf("🆕 New Worker Detected:\n"+
				"  Worker: %s\n"+
				"  Instances: %d", worker.Name, len(worker.Instances)))

			// 각 인스턴스 정보 추가
			for _, instance := range worker.Instances {
				changes = append(changes, fmt.Sprintf("    • Instance: %s\n"+
					"      Status: %s\n"+
					"      IP: %s\n"+
					"      GPU: %s",
					instance.Name,
					instance.Status,
					instance.Info.IPAddress,
					instance.Info.NvidiaSmi.GPU[0].ProductName))
			}
		} else {
			// 인스턴스 수 변경 확인
			if currentStatus.InstanceCount != prevStatus.InstanceCount {
				changes = append(changes, fmt.Sprintf("📊 Worker '%s' Instance Count Changed: %d → %d",
					worker.Name, prevStatus.InstanceCount, currentStatus.InstanceCount))
			}

			// 현재 인스턴스 상태 확인
			currentInstanceIDs := make(map[string]bool)
			for instanceID, currentInstance := range currentStatus.Instances {
				currentInstanceIDs[instanceID] = true

				if prevInstance, ok := prevStatus.Instances[instanceID]; ok {
					// 상태 변경 확인
					if currentInstance.Status != prevInstance.Status {
						changes = append(changes, fmt.Sprintf("🔄 Instance Status Changed:\n"+
							"  Worker: %s\n"+
							"  Instance: %s\n"+
							"  Status: %s → %s\n"+
							"  IP: %s\n"+
							"  GPU: %s",
							worker.Name,
							currentInstance.Name,
							prevInstance.Status,
							currentInstance.Status,
							currentInstance.IPAddress,
							currentInstance.GPU.Name))
					}
					// IP 변경 확인
					if currentInstance.IPAddress != prevInstance.IPAddress {
						changes = append(changes, fmt.Sprintf("🌐 Instance IP Changed:\n"+
							"  Worker: %s\n"+
							"  Instance: %s\n"+
							"  IP: %s → %s",
							worker.Name,
							currentInstance.Name,
							prevInstance.IPAddress,
							currentInstance.IPAddress))
					}
				} else {
					// 새로운 인스턴스 발견
					changes = append(changes, fmt.Sprintf("➕ New Instance Added:\n"+
						"  Worker: %s\n"+
						"  Instance: %s\n"+
						"  Status: %s\n"+
						"  IP: %s\n"+
						"  GPU: %s",
						worker.Name,
						currentInstance.Name,
						currentInstance.Status,
						currentInstance.IPAddress,
						currentInstance.GPU.Name))
				}
			}

			// 삭제된 인스턴스 확인
			for instanceID, prevInstance := range prevStatus.Instances {
				if !currentInstanceIDs[instanceID] {
					changes = append(changes, fmt.Sprintf("➖ Instance Removed:\n"+
						"  Worker: %s\n"+
						"  Instance: %s\n"+
						"  Last Status: %s\n"+
						"  Last IP: %s\n"+
						"  GPU: %s",
						worker.Name,
						prevInstance.Name,
						prevInstance.Status,
						prevInstance.IPAddress,
						prevInstance.GPU.Name))
				}
			}
		}

		m.state.Workers[worker.ID] = currentStatus
	}

	// 삭제된 워커 확인
	for workerID, prevWorker := range m.state.Workers {
		if !currentWorkers[workerID] {
			changes = append(changes, fmt.Sprintf("❌ Worker Removed:\n"+
				"  Worker: %s\n"+
				"  Last Instance Count: %d",
				prevWorker.Name,
				prevWorker.InstanceCount))
			delete(m.state.Workers, workerID)
		}
	}

	if len(changes) > 0 {
		message := "🔔 Worker Status Changes:\n\n" + strings.Join(changes, "\n\n")
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

	// 전체 Online Workers 수, Active Workers 수, 총 인스턴스 수 표시
	activeWorkerCount := 0
	totalInstanceCount := 0
	runningInstanceCount := 0
	initializingInstanceCount := 0

	for _, worker := range workers {
		if !worker.IsArchived && len(worker.Instances) > 0 {
			activeWorkerCount++
			totalInstanceCount += len(worker.Instances)
			for _, instance := range worker.Instances {
				switch instance.Status {
				case "Running":
					runningInstanceCount++
				case "Initializing":
					initializingInstanceCount++
				}
			}
		}
	}

	status.WriteString(fmt.Sprintf("Total Online Workers: %d\n", count))
	status.WriteString(fmt.Sprintf("My Active Workers: %d\n", activeWorkerCount))
	status.WriteString(fmt.Sprintf("Total Instances: %d (Running: %d, Initializing: %d)\n\n",
		totalInstanceCount, runningInstanceCount, initializingInstanceCount))

	status.WriteString("Active Workers:\n")

	for _, worker := range workers {
		if !worker.IsArchived && len(worker.Instances) > 0 {
			// 워커 이름과 인스턴스 개수 표시
			runningCount := 0
			initializingCount := 0
			for _, instance := range worker.Instances {
				switch instance.Status {
				case "Running":
					runningCount++
				case "Initializing":
					initializingCount++
				}
			}

			status.WriteString(fmt.Sprintf("\n%s:\n", worker.Name))
			status.WriteString(fmt.Sprintf("Total Instances: %d (Running: %d, Initializing: %d)\n",
				len(worker.Instances), runningCount, initializingCount))

			// 각 인스턴스 정보 표시
			for _, instance := range worker.Instances {
				// Runtime 정보 가져오기
				runtime := instance.Info.Runtime
				if len(instance.PoolAssignments) > 0 {
					model := instance.PoolAssignments[0].Model
					lane := instance.PoolAssignments[0].Lane
					status.WriteString(fmt.Sprintf("  • %s (%s)\n", instance.Name, instance.Status))
					status.WriteString(fmt.Sprintf("    Location: %s, %s\n", instance.Info.City, instance.Info.Country))
					status.WriteString(fmt.Sprintf("    Runtime: %s\n", runtime))
					status.WriteString(fmt.Sprintf("    Model: %s\n", model))
					status.WriteString(fmt.Sprintf("    Lane: %s\n", lane))

					// GPU 정보가 있는 경우 표시
					if instance.Info.NvidiaSmi != nil && len(instance.Info.NvidiaSmi.GPU) > 0 {
						gpu := instance.Info.NvidiaSmi.GPU[0]
						if len(gpu.ProductName) > 0 {
							status.WriteString(fmt.Sprintf("    GPU: %s\n", gpu.ProductName[0]))
						}
					}
				}
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
	// 이미 config에 chat_id가 설정되어 있는 경우 검증만 수행
	if m.config.Telegram.ChatID != "" {
		// chat_id가 유효한 숫자인지 확인
		_, err := strconv.ParseInt(m.config.Telegram.ChatID, 10, 64)
		if err == nil {
			m.logMessage(fmt.Sprintf("✅ Using existing chat ID: %s", m.config.Telegram.ChatID))
			return nil
		}
		// 유효하지 않은 경우 로그 출력
		m.logMessage(fmt.Sprintf("❌ Invalid chat ID in config: %v", err))
	}

	// chat_id가 없거나 유효하지 않은 경우 새로 가져오기 시도
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

	// 초기 상태 설정 (알림 없이)
	if err := m.initializeState(); err != nil {
		m.logMessage(fmt.Sprintf("❌ Initial state setup failed: %v", err))
		m.sendErrorAlert(err)
		return
	}
	m.logMessage("✅ Initial state setup successful")

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
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			if err := m.sendDailyReport(); err != nil {
				m.logMessage(fmt.Sprintf("❌ Daily report failed: %v", err))
				m.sendErrorAlert(err)
			}
		}
	}()

	// 1시간마다 토큰 리포트 전송
	go func() {
		m.logMessage("Starting hourly token report scheduler")
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			if err := m.sendHourlyReport(); err != nil {
				m.logMessage(fmt.Sprintf("❌ Hourly report failed: %v", err))
				m.sendErrorAlert(err)
			}
		}
	}()

	m.logMessage("✅ Monitor started successfully")
}

// 초기 상태를 설정하는 새로운 메서드
func (m *Monitor) initializeState() error {
	// 초기 워커 상태 설정
	workers, err := getActiveWorkers(m.token)
	if err != nil {
		return err
	}

	for _, worker := range workers {
		currentStatus := &WorkerStatus{
			Name:          worker.Name,
			InstanceCount: len(worker.Instances),
			Instances:     make(map[string]InstanceStatus),
		}

		for _, instance := range worker.Instances {
			gpuInfo := InstanceStatus{
				Name:      instance.Name,
				Status:    instance.Status,
				IPAddress: instance.Info.IPAddress,
				Runtime:   instance.Info.Runtime,
			}

			if instance.Info.NvidiaSmi != nil && len(instance.Info.NvidiaSmi.GPU) > 0 {
				gpu := instance.Info.NvidiaSmi.GPU[0]
				if len(gpu.ProductName) > 0 {
					gpuInfo.GPU.Name = gpu.ProductName[0]
				}
				if len(gpu.Temperature) > 0 && len(gpu.Temperature[0].GPUTemp) > 0 {
					gpuInfo.GPU.Temp = gpu.Temperature[0].GPUTemp[0]
				}
				if len(gpu.Utilization) > 0 && len(gpu.Utilization[0].GPUUtil) > 0 {
					gpuInfo.GPU.Utilization = gpu.Utilization[0].GPUUtil[0]
				}
				if len(gpu.FBMemoryUsage) > 0 {
					if len(gpu.FBMemoryUsage[0].Used) > 0 {
						gpuInfo.GPU.Memory.Used = gpu.FBMemoryUsage[0].Used[0]
					}
					if len(gpu.FBMemoryUsage[0].Total) > 0 {
						gpuInfo.GPU.Memory.Total = gpu.FBMemoryUsage[0].Total[0]
					}
				}
				if len(gpu.PowerReadings) > 0 {
					if len(gpu.PowerReadings[0].PowerDraw) > 0 {
						gpuInfo.GPU.Power.Draw = gpu.PowerReadings[0].PowerDraw[0]
					}
					if len(gpu.PowerReadings[0].PowerState) > 0 {
						gpuInfo.GPU.Power.State = gpu.PowerReadings[0].PowerState[0]
					}
				}
			}

			currentStatus.Instances[instance.ID] = gpuInfo
		}

		m.state.Workers[worker.ID] = currentStatus
	}

	// 초기 토큰 캐시 설정
	tokens24h, err := getTokensLast24Hours(m.token)
	if err != nil {
		return err
	}

	userMetrics, err := getUserMetrics(m.token, "c9XhxeKcWRChV875-H7u3")
	if err != nil {
		return err
	}

	m.state.TokenCache.GlobalTokens = TokenMetrics{
		TokensCount: tokens24h,
		LastUpdated: time.Now(),
	}

	m.state.TokenCache.UserTokens = TokenMetrics{
		GenerationsCount: int64(userMetrics.Result.Data.JSON.GenerationsLast24Hours),
		TokensCount:      userMetrics.Result.Data.JSON.TokensLast24Hours,
		LastUpdated:      time.Now(),
	}

	m.state.TokenCache.WorkerTokens = make(map[string]TokenMetrics)
	for _, worker := range workers {
		if worker.IsArchived {
			continue
		}

		metrics, err := getWorkerMetrics(m.token, worker.ID)
		if err != nil {
			continue
		}

		m.state.TokenCache.WorkerTokens[worker.ID] = TokenMetrics{
			GenerationsCount: int64(metrics.Result.Data.JSON.GenerationsLast24Hours),
			TokensCount:      metrics.Result.Data.JSON.TokensLast24Hours,
			LastUpdated:      time.Now(),
		}
	}

	return nil
}

func (m *Monitor) calculateTokenChanges(current, previous TokenMetrics) (int64, int64) {
	if previous.LastUpdated.IsZero() {
		return 0, 0
	}
	tokenChange := current.TokensCount - previous.TokensCount
	genChange := current.GenerationsCount - previous.GenerationsCount
	return tokenChange, genChange
}

// 시간별 리포트 함수
func (m *Monitor) sendHourlyReport() error {
	if err := m.refreshAuthIfNeeded(); err != nil {
		return err
	}

	// 글로벌 토큰 정보 수집
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

	// 현재 시간의 메트릭 생성
	currentGlobal := TokenMetrics{
		TokensCount: tokens24h,
		LastUpdated: time.Now(),
	}

	currentUser := TokenMetrics{
		GenerationsCount: int64(userMetrics.Result.Data.JSON.GenerationsLast24Hours),
		TokensCount:      userMetrics.Result.Data.JSON.TokensLast24Hours,
		LastUpdated:      time.Now(),
	}

	// 변화량 계산
	globalTokenChange, _ := m.calculateTokenChanges(currentGlobal, m.state.TokenCache.GlobalTokens)
	userTokenChange, userGenChange := m.calculateTokenChanges(currentUser, m.state.TokenCache.UserTokens)

	// 리포트 생성
	report := strings.Builder{}
	report.WriteString("⏰ Hourly Token Report\n\n")

	report.WriteString(fmt.Sprintf("Global Changes (Last Hour):\n"))
	report.WriteString(fmt.Sprintf("- Tokens: %d → %d (Δ%d)\n\n",
		m.state.TokenCache.GlobalTokens.TokensCount,
		currentGlobal.TokensCount,
		globalTokenChange))

	report.WriteString("User Changes (Last Hour):\n")
	report.WriteString(fmt.Sprintf("- Generations: %d → %d (Δ%d)\n",
		m.state.TokenCache.UserTokens.GenerationsCount,
		currentUser.GenerationsCount,
		userGenChange))
	report.WriteString(fmt.Sprintf("- Tokens: %d → %d (Δ%d)\n\n",
		m.state.TokenCache.UserTokens.TokensCount,
		currentUser.TokensCount,
		userTokenChange))

	report.WriteString("Worker Changes (Last Hour):\n")
	for _, worker := range workers {
		if worker.IsArchived {
			continue
		}

		metrics, err := getWorkerMetrics(m.token, worker.ID)
		if err != nil {
			continue
		}

		current := TokenMetrics{
			GenerationsCount: int64(metrics.Result.Data.JSON.GenerationsLast24Hours),
			TokensCount:      metrics.Result.Data.JSON.TokensLast24Hours,
			LastUpdated:      time.Now(),
		}

		previous, exists := m.state.TokenCache.WorkerTokens[worker.ID]
		tokenChange, genChange := int64(0), int64(0)
		if exists {
			tokenChange, genChange = m.calculateTokenChanges(current, previous)
		}

		if tokenChange != 0 || genChange != 0 {
			report.WriteString(fmt.Sprintf("\n%s:\n", worker.Name))
			report.WriteString(fmt.Sprintf("- Generations: %d → %d (Δ%d)\n",
				previous.GenerationsCount,
				current.GenerationsCount,
				genChange))
			report.WriteString(fmt.Sprintf("- Tokens: %d → %d (Δ%d)\n",
				previous.TokensCount,
				current.TokensCount,
				tokenChange))
		}

		// 캐시 업데이트
		if m.state.TokenCache.WorkerTokens == nil {
			m.state.TokenCache.WorkerTokens = make(map[string]TokenMetrics)
		}
		m.state.TokenCache.WorkerTokens[worker.ID] = current
	}

	// 캐시 업데이트
	m.state.TokenCache.GlobalTokens = currentGlobal
	m.state.TokenCache.UserTokens = currentUser

	// 리포트 전송
	return m.sendTelegramMessage(report.String(), "hourly")
}

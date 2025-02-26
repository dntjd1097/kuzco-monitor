package api

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// HourlyStats는 시간별 통계를 저장하는 구조체입니다
type HourlyStats struct {
	RPM struct {
		Min     int     `json:"min"`
		Max     int     `json:"max"`
		Avg     float64 `json:"avg"`
		Current int     `json:"current"`
		Count   int     `json:"count"`
		Sum     int     `json:"sum"`
	} `json:"rpm"`
	TotalInstances struct {
		Min     int     `json:"min"`
		Max     int     `json:"max"`
		Current int     `json:"current"`
		Avg     float64 `json:"avg"`
		Count   int     `json:"count"`
		Sum     int     `json:"sum"`
	} `json:"totalInstances"`
	GenerationLastHour struct {
		General int     `json:"general"`
		User    int     `json:"user"`
		Ratio   float64 `json:"ratio"`
	} `json:"generationLastHour"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// AlertStateManager는 알림 상태를 관리합니다
type AlertStateManager struct {
	state AlertState
	mu    sync.Mutex
}

var globalAlertState = &AlertStateManager{}

func (m *AlertStateManager) getState() AlertState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *AlertStateManager) setState(state AlertState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = state
}

type MetricsResponse struct {
	Result struct {
		Data struct {
			JSON interface{} `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

type MetricsQuery struct {
	Endpoint string
	Payload  interface{}
}

type GenerationHistory struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
	Label string `json:"label"`
}

type GenerationHistoryResponse struct {
	Result struct {
		Data struct {
			JSON []GenerationHistory `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

type VersionResponse struct {
	Result struct {
		Data struct {
			JSON struct {
				TauriVersion string `json:"tauriVersion"`
				CLIVersion   string `json:"cliVersion"`
			} `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

type GeneralMetrics struct {
	RunningInstanceCount   int                 `json:"runningInstanceCount"`
	RPM                    int                 `json:"rpm"`
	TokensLast24Hours      int64               `json:"tokensLast24Hours"`
	TokensAllTime          int64               `json:"tokensAllTime"`
	GenerationsLast24Hours int                 `json:"generationsLast24Hours"`
	CLIVersion             string              `json:"cliVersion"`
	GenerationsHistory     []GenerationHistory `json:"generationsHistory"`
}

type UserMetrics struct {
	TokensLast24Hours      int64               `json:"tokensLast24Hours"`
	TokensAllTime          int64               `json:"tokensAllTime"`
	GenerationsLast24Hours int                 `json:"generationsLast24Hours"`
	TotalInstances         int                 `json:"totalInstances"`
	TotalDailyCost         float64             `json:"totalDailyCost"`
	TokensPerInstance      int64               `json:"tokensPerInstance"`
	Share                  float64             `json:"share"`
	Efficiency             float64             `json:"efficiency"`
	GenerationsHistory     []GenerationHistory `json:"generationsHistory"`
	Workers                []Worker            `json:"workers"`
}

type DailyMetrics struct {
	Share           float64 `json:"share"`
	Efficiency      float64 `json:"efficiency"`
	KuzcoTotalCost  float64 `json:"kuzcoTotalCost"`  // Kuzco 일일 비용
	VastaiTotalCost float64 `json:"vastaiTotalCost"` // Vastai 일일 비용
	TotalDailyCost  float64 `json:"totalDailyCost"`  // 전체 일일 비용
	Timestamp       string  `json:"timestamp"`
}

type InstanceMetrics struct {
	Status          string `json:"status"`
	Model           string `json:"model"`
	Lane            string `json:"lane"`
	IP              string `json:"ip"`
	GPUModel        string `json:"gpuModel"`
	Version         string `json:"version"`
	VersionMismatch bool   `json:"versionMismatch"`
}

type WorkerMinuteMetrics struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	InstanceCount      int               `json:"instanceCount"`
	DailyCost          float64           `json:"dailyCost"`
	TokensPerInstance  int64             `json:"tokensPerInstance"`
	TokensLast24H      int64             `json:"tokensLast24h"`
	TotalTokens        int64             `json:"totalTokens"`
	GenerationsLast24H int               `json:"generationsLast24h"`
	GenerationLastHour int               `json:"generationLastHour"`
	Instances          []InstanceMetrics `json:"instances"`
}

type MinuteMetrics struct {
	General struct {
		TotalInstances         int    `json:"totalInstances"`
		RPM                    int    `json:"rpm"`
		TokensLast24Hours      int64  `json:"tokensLast24Hours"`
		GenerationsLast24Hours int    `json:"generationsLast24Hours"`
		GenerationLastHour     int    `json:"generationLastHour"`
		CLIVersion             string `json:"cliVersion"`
	} `json:"general"`
	User struct {
		TokensLast24Hours      int64                 `json:"tokensLast24Hours"`
		TokensAllTime          int64                 `json:"tokensAllTime"`
		GenerationsLast24Hours int                   `json:"generationsLast24Hours"`
		TotalInstances         int                   `json:"totalInstances"`       // Vast.ai API의 instances_found
		ActualTotalInstances   int                   `json:"actualTotalInstances"` // 기존 Kuzco의 totalInstances
		InstancesMismatch      bool                  `json:"instancesMismatch"`    // 두 값이 다른 경우 true
		KuzcoDailyCost         float64               `json:"kuzcoDailyCost"`       // instance.json 기반 예상 비용
		VastaiDailyCost        float64               `json:"vastaiDailyCost"`      // Vast.ai API에서 가져온 실제 비용
		TotalDailyCost         float64               `json:"totalDailyCost"`       // 최종 사용될 비용
		TokensPerInstance      int64                 `json:"tokensPerInstance"`
		Share                  float64               `json:"share"`
		GenerationLastHour     int                   `json:"generationLastHour"`
		VastaiCredit           *VastaiCredit         `json:"vastaiCredit,omitempty"` // Vast.ai credit 정보
		Workers                []WorkerMinuteMetrics `json:"workers"`
	} `json:"user"`
	Timestamp  string     `json:"timestamp"`
	AlertState AlertState `json:"alertState"` // 알림 상태 추가
}

type Metrics struct {
	General GeneralMetrics `json:"general"`
	User    UserMetrics    `json:"user"`
}

type TokenHistory struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
	Label string `json:"label"`
}

type TokenHistoryResponse struct {
	Result struct {
		Data struct {
			JSON []TokenHistory `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

// AlertState는 각 알림의 상태를 관리하는 구조체입니다
type AlertState struct {
	VersionMismatchAlerted bool      `json:"versionMismatchAlerted"` // 버전 불일치 알림 여부
	InstanceCountAlerted   bool      `json:"instanceCountAlerted"`   // 인스턴스 수 알림 여부
	CreditAlerted          bool      `json:"creditAlerted"`          // credit 알림 여부
	LastAlertTime          time.Time `json:"lastAlertTime"`          // 마지막 알림 시간
	InstanceMismatchStart  time.Time `json:"instanceMismatchStart"`  // 인스턴스 불일치 시작 시간
}

// AlertConfig는 알림 설정을 관리하는 구조체입니다
type AlertConfig struct {
	MinInstanceCount int     `json:"minInstanceCount" yaml:"minInstanceCount"` // 최소 인스턴스 수
	MinCredit        float64 `json:"minCredit" yaml:"minCredit"`               // 최소 credit 잔액
	Enabled          bool    `json:"enabled" yaml:"enabled"`                   // 알림 활성화 여부
}

// HourlyStatsManager는 시간별 통계를 관리합니다
type HourlyStatsManager struct {
	stats []MinuteStats
	mutex sync.Mutex
}

// MinuteStats는 1분 단위의 통계를 저장하는 구조체입니다
type MinuteStats struct {
	RPM            int       `json:"rpm"`
	TotalInstances int       `json:"totalInstances"`
	GeneralGen     int       `json:"generalGen"`
	UserGen        int       `json:"userGen"`
	Timestamp      time.Time `json:"timestamp"`
}

var GlobalHourlyStats = &HourlyStatsManager{
	stats: make([]MinuteStats, 0, 60), // 60분 동안의 데이터를 저장
}

// GetStats는 지난 60분 동안의 통계를 반환합니다
func (m *HourlyStatsManager) GetStats() HourlyStats {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	result := HourlyStats{}
	if len(m.stats) == 0 {
		result.StartTime = time.Now()
		result.EndTime = time.Now()
		return result
	}

	// 시작 시간과 종료 시간 설정
	result.StartTime = m.stats[0].Timestamp
	result.EndTime = m.stats[len(m.stats)-1].Timestamp

	// RPM 통계 계산
	for i, stat := range m.stats {
		if i == 0 {
			result.RPM.Min = stat.RPM
			result.RPM.Max = stat.RPM

		} else {
			if stat.RPM < result.RPM.Min {
				result.RPM.Min = stat.RPM
			}
			if stat.RPM > result.RPM.Max {
				result.RPM.Max = stat.RPM
			}
		}
		result.RPM.Current = stat.RPM
		result.RPM.Sum += stat.RPM
		result.RPM.Count++
	}
	if result.RPM.Count > 0 {
		result.RPM.Avg = float64(result.RPM.Sum) / float64(result.RPM.Count)
	}

	// TotalInstances 통계 계산
	for i, stat := range m.stats {
		result.TotalInstances.Current = stat.TotalInstances
		if i == 0 {
			result.TotalInstances.Min = stat.TotalInstances
			result.TotalInstances.Max = stat.TotalInstances

		} else {
			if stat.TotalInstances < result.TotalInstances.Min {
				result.TotalInstances.Min = stat.TotalInstances
			}
			if stat.TotalInstances > result.TotalInstances.Max {
				result.TotalInstances.Max = stat.TotalInstances
			}
		}
		result.TotalInstances.Sum += stat.TotalInstances
		result.TotalInstances.Count++
	}
	if result.TotalInstances.Count > 0 {
		result.TotalInstances.Avg = float64(result.TotalInstances.Sum) / float64(result.TotalInstances.Count)
	}

	// Generation 통계 계산
	for _, stat := range m.stats {
		result.GenerationLastHour.General += stat.GeneralGen
		result.GenerationLastHour.User += stat.UserGen
	}
	if result.GenerationLastHour.General > 0 {
		result.GenerationLastHour.Ratio = float64(result.GenerationLastHour.User) / float64(result.GenerationLastHour.General) * 100
	}

	return result
}

// UpdateStats는 새로운 통계를 추가하고 60분이 지난 데이터는 제거합니다
func (m *HourlyStatsManager) UpdateStats(metrics MinuteMetrics) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-60 * time.Minute)

	// 60분이 지난 데이터 제거
	validStats := make([]MinuteStats, 0, 60)
	for _, stat := range m.stats {
		if stat.Timestamp.After(cutoff) {
			validStats = append(validStats, stat)
		}
	}

	// 새로운 데이터 추가
	newStat := MinuteStats{
		RPM:            metrics.General.RPM,
		TotalInstances: metrics.General.TotalInstances,
		GeneralGen:     metrics.General.GenerationLastHour,
		UserGen:        metrics.User.GenerationLastHour,
		Timestamp:      now,
	}
	validStats = append(validStats, newStat)

	m.stats = validStats
}

// GetAndResetStats는 더 이상 사용하지 않습니다
func (m *HourlyStatsManager) GetAndResetStats() HourlyStats {
	return m.GetStats()
}

func (m *Client) collectDailyMetrics(userID string, vastaiToken string, includeVastaiCost bool, sendAlert func(string, string) error, ch chan<- DailyMetrics) error {
	kuzcoClient := NewKuzcoClient(m)
	metrics, err := kuzcoClient.GetAllMetrics(userID)
	if err != nil {
		return err
	}

	// Vastai 일일 비용 가져오기
	var vastaiCost float64
	var vastaiCredit *VastaiCredit
	// vastaiToken이 비어있지 않으면 enabled=true라는 의미
	isVastaiEnabled := vastaiToken != ""

	if isVastaiEnabled && includeVastaiCost {
		vastaiClient := NewVastaiClient(vastaiToken)
		vastaiCost, err = vastaiClient.GetDailyCost()
		if err != nil {
			log.Printf("Failed to get vastai cost: %v", err)
			vastaiCost = 0
		}

		// Get credit information
		vastaiCredit, err = vastaiClient.GetCredit()
		if err != nil {
			log.Printf("Failed to get vastai credit: %v", err)
		}
	}

	// 가격 책정 로직
	var totalDailyCost float64
	if isVastaiEnabled && includeVastaiCost {
		// Vastai가 활성화되어 있고 Vastai 비용을 포함하는 경우
		totalDailyCost = vastaiCost
	} else {
		// Vastai가 비활성화되어 있거나 Vastai 비용을 포함하지 않는 경우
		totalDailyCost = metrics.User.TotalDailyCost
	}

	// efficiency 계산
	efficiency := metrics.User.Efficiency
	if metrics.User.Share > 0 {
		efficiency = totalDailyCost / (metrics.User.Share * 100)
	}

	// 현재 KST 시간 가져오기
	kst := time.Now().In(time.FixedZone("KST", 9*60*60))
	dateStr := kst.Format("2006-01-02")

	// 텔레그램 메시지 작성
	message := fmt.Sprintf("%s\n\n포인트 : %d\n비중 : %.2f%%\n비용 : $%.2f\n효율(1%%) : $%d",
		dateStr,
		metrics.User.TokensLast24Hours,
		metrics.User.Share*100,
		totalDailyCost,
		int(efficiency))

	// Vastai credit 정보가 있는 경우 추가
	if vastaiCredit != nil {
		message += fmt.Sprintf("\n잔액 : $%.2f", vastaiCredit.Credit)
	}

	// 텔레그램으로 메시지 전송
	if err := sendAlert(message, "daily"); err != nil {
		log.Printf("Failed to send daily metrics alert: %v", err)
	}

	ch <- DailyMetrics{
		Share:           metrics.User.Share,
		Efficiency:      efficiency,
		KuzcoTotalCost:  metrics.User.TotalDailyCost,
		VastaiTotalCost: vastaiCost,
		TotalDailyCost:  totalDailyCost,
		Timestamp:       time.Now().Format(time.RFC3339),
	}
	return nil
}

// CollectMetrics collects metrics periodically
func (c *Client) CollectMetrics(
	userID string,
	vastaiToken string,
	includeVastaiCost bool,
	alertConfig AlertConfig,
	sendAlert func(string, string) error,
	dailyChan chan<- DailyMetrics,
	minuteChan chan<- MinuteMetrics,
	stop <-chan struct{},
) {
	// 개발 모드 체크
	isDev := os.Getenv("ENV") == "dev"

	// 타이머 간격 설정
	var (
		dailyInterval  time.Duration
		minuteInterval time.Duration
	)

	if isDev {
		dailyInterval = time.Minute  // 개발 환경: 1분
		minuteInterval = time.Minute // 개발 환경: 1분
	} else {
		// 프로덕션 환경: 기존 로직
		now := time.Now().UTC()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
		dailyInterval = nextMidnight.Sub(now)
		minuteInterval = time.Minute
	}

	// 타이머 설정
	dailyTimer := time.NewTimer(dailyInterval)
	minuteTicker := time.NewTicker(minuteInterval)
	defer dailyTimer.Stop()
	defer minuteTicker.Stop()

	// 초기 메트릭스 수집
	if err := c.collectMinuteMetrics(userID, vastaiToken, includeVastaiCost, alertConfig, sendAlert, minuteChan); err != nil {
		log.Printf("Failed to collect minute metrics: %v", err)
	}
	if isDev {
		// 개발 환경에서는 즉시 일일 메트릭스도 수집
		if err := c.collectDailyMetrics(userID, vastaiToken, includeVastaiCost, sendAlert, dailyChan); err != nil {
			log.Printf("Failed to collect daily metrics: %v", err)
		}
	}

	for {
		select {
		case <-dailyTimer.C:
			if err := c.collectDailyMetrics(userID, vastaiToken, includeVastaiCost, sendAlert, dailyChan); err != nil {
				log.Printf("Failed to collect daily metrics: %v", err)
			}
			// 타이머 재설정
			if isDev {
				dailyTimer.Reset(10 * time.Second)
			} else {
				dailyTimer.Reset(24 * time.Hour)
			}

		case <-minuteTicker.C:
			if err := c.collectMinuteMetrics(userID, vastaiToken, includeVastaiCost, alertConfig, sendAlert, minuteChan); err != nil {
				log.Printf("Failed to collect minute metrics: %v", err)
			}

		case <-stop:
			return
		}
	}
}

func (m *Client) collectMinuteMetrics(userID string, vastaiToken string, includeVastaiCost bool, alertConfig AlertConfig, sendAlert func(string, string) error, ch chan<- MinuteMetrics) error {
	kuzcoClient := NewKuzcoClient(m)
	metrics, err := kuzcoClient.GetAllMetrics(userID)
	if err != nil {
		return err
	}

	mm := MinuteMetrics{
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// General metrics
	mm.General.TotalInstances = metrics.General.RunningInstanceCount
	mm.General.RPM = metrics.General.RPM
	mm.General.TokensLast24Hours = metrics.General.TokensLast24Hours
	mm.General.GenerationsLast24Hours = metrics.General.GenerationsLast24Hours
	mm.General.CLIVersion = metrics.General.CLIVersion
	if len(metrics.General.GenerationsHistory) > 0 {
		mm.General.GenerationLastHour = metrics.General.GenerationsHistory[0].Value
	}

	// User metrics
	mm.User.TokensLast24Hours = metrics.User.TokensLast24Hours
	mm.User.TokensAllTime = metrics.User.TokensAllTime
	mm.User.GenerationsLast24Hours = metrics.User.GenerationsLast24Hours
	mm.User.ActualTotalInstances = metrics.User.TotalInstances // 기존 Kuzco의 totalInstances 저장

	// Vast.ai API에서 인스턴스 수와 credit 정보 가져오기
	if vastaiToken != "" {
		vastaiClient := NewVastaiClient(vastaiToken)

		// Get instance count
		vastaiInstances, err := vastaiClient.GetInstanceCount()
		if err != nil {
			log.Printf("Failed to get vastai instance count: %v", err)
			mm.User.TotalInstances = mm.User.ActualTotalInstances
		} else {
			mm.User.TotalInstances = vastaiInstances
			mm.User.InstancesMismatch = mm.User.TotalInstances != mm.User.ActualTotalInstances
		}

		// Get credit information
		if includeVastaiCost {
			credit, err := vastaiClient.GetCredit()
			if err != nil {
				log.Printf("Failed to get vastai credit: %v", err)
			} else {
				mm.User.VastaiCredit = credit
			}
		}
	} else {
		mm.User.TotalInstances = mm.User.ActualTotalInstances
	}

	// Calculate Kuzco daily cost from instance.json
	mm.User.KuzcoDailyCost = metrics.User.TotalDailyCost

	// Get Vast.ai cost if enabled
	if vastaiToken != "" && includeVastaiCost {
		vastaiClient := NewVastaiClient(vastaiToken)
		vastaiCost, err := vastaiClient.GetDailyCost()
		if err != nil {
			log.Printf("Failed to get vastai cost: %v", err)
			mm.User.VastaiDailyCost = 0
		} else {
			mm.User.VastaiDailyCost = vastaiCost
		}

		// Use Vast.ai cost as total cost
		mm.User.TotalDailyCost = mm.User.VastaiDailyCost
	} else {
		// Use Kuzco cost as total cost
		mm.User.TotalDailyCost = mm.User.KuzcoDailyCost
	}

	mm.User.TokensPerInstance = metrics.User.TokensPerInstance
	mm.User.Share = metrics.User.Share
	if len(metrics.User.GenerationsHistory) > 0 {
		mm.User.GenerationLastHour = metrics.User.GenerationsHistory[0].Value
	}

	// Worker metrics
	mm.User.Workers = make([]WorkerMinuteMetrics, 0, len(metrics.User.Workers))
	for _, w := range metrics.User.Workers {
		worker := WorkerMinuteMetrics{
			ID:                 w.ID,
			Name:               w.Name,
			InstanceCount:      w.InstanceCount,
			DailyCost:          w.DailyCost,
			TokensPerInstance:  w.TokensPerInstance,
			TokensLast24H:      w.TokensLast24H,
			TotalTokens:        w.TotalTokens,
			GenerationsLast24H: w.GenerationsLast24H,
			Instances:          make([]InstanceMetrics, 0, len(w.Instances)),
		}

		// 인스턴스 정보 추가
		for _, inst := range w.Instances {
			worker.Instances = append(worker.Instances, InstanceMetrics(inst))
		}

		if len(w.GenerationsHistory) > 0 {
			worker.GenerationLastHour = w.GenerationsHistory[0].Value
		}
		mm.User.Workers = append(mm.User.Workers, worker)
	}

	// 알림 상태 가져오기
	mm.AlertState = globalAlertState.getState()

	// Check alerts with provided configuration
	if err := m.checkAlerts(&mm, alertConfig, sendAlert); err != nil {
		log.Printf("Failed to check alerts: %v", err)
	}

	// 알림 상태 업데이트
	globalAlertState.setState(mm.AlertState)

	// 시간별 통계 업데이트
	GlobalHourlyStats.UpdateStats(mm)

	ch <- mm
	return nil
}

// checkAlerts는 모든 알림을 체크하고 관리합니다
func (m *Client) checkAlerts(mm *MinuteMetrics, config AlertConfig, sendAlert func(string, string) error) error {
	if err := m.checkVersionMismatch(mm, config, sendAlert); err != nil {
		return fmt.Errorf("version mismatch check failed: %w", err)
	}

	if err := m.checkInstanceCount(mm, config, sendAlert); err != nil {
		return fmt.Errorf("instance count check failed: %w", err)
	}

	if err := m.checkCredit(mm, config, sendAlert); err != nil {
		return fmt.Errorf("credit check failed: %w", err)
	}

	return nil
}

// checkVersionMismatch는 버전 불일치를 체크하고 알림을 보냅니다
func (m *Client) checkVersionMismatch(mm *MinuteMetrics, config AlertConfig, sendAlert func(string, string) error) error {
	hasVersionMismatch := false
	var mismatchedWorkers []string

	for _, worker := range mm.User.Workers {
		for _, instance := range worker.Instances {
			if instance.VersionMismatch {
				hasVersionMismatch = true
				mismatchedWorkers = append(mismatchedWorkers, fmt.Sprintf("%s (IP: %s)", worker.Name, instance.IP))
			}
		}
	}

	// 문제가 발생했고, 아직 알림을 보내지 않은 경우에만 알림 전송
	if hasVersionMismatch && !mm.AlertState.VersionMismatchAlerted {
		title := "⚠️ Version Mismatch Alert"
		msg := fmt.Sprintf("The following workers have version mismatches:\n%s", strings.Join(mismatchedWorkers, "\n"))
		message := fmt.Sprintf("%s\n%s", title, CodeBlock(msg))
		if err := sendAlert(message, "error"); err != nil {
			return err
		}
		mm.AlertState.VersionMismatchAlerted = true
	} else if !hasVersionMismatch && mm.AlertState.VersionMismatchAlerted {
		// 문제가 해결되었고, 이전에 알림을 보냈던 경우에만 복구 알림 전송
		title := "✅ Version Mismatch Resolved"
		msg := fmt.Sprintf("All workers are now running the correct version.")
		message := fmt.Sprintf("%s\n%s", title, CodeBlock(msg))
		if err := sendAlert(message, "error"); err != nil {
			return err
		}
		mm.AlertState.VersionMismatchAlerted = false
	}

	return nil
}

// checkInstanceCount는 인스턴스 수가 최소 기준보다 낮은지 체크합니다
func (m *Client) checkInstanceCount(mm *MinuteMetrics, config AlertConfig, sendAlert func(string, string) error) error {
	if !config.Enabled {
		return nil
	}

	// Vast.ai가 활성화된 경우에만 체크 (VastaiCredit이 nil이 아닌 경우)
	if mm.User.VastaiCredit == nil {
		return nil
	}

	// 인스턴스 수 불일치 체크
	if mm.User.InstancesMismatch {
		// 불일치가 처음 발생한 경우
		if mm.AlertState.InstanceMismatchStart.IsZero() {
			mm.AlertState.InstanceMismatchStart = time.Now()
		}

		// 5분 이상 불일치가 지속되었고, 아직 알림을 보내지 않은 경우
		if time.Since(mm.AlertState.InstanceMismatchStart) >= 5*time.Minute && !mm.AlertState.InstanceCountAlerted {
			title := "⚠️ Instance Count Mismatch Alert"
			msg := fmt.Sprintf("Vast.ai instances: %d\nActual instances: %d", mm.User.TotalInstances, mm.User.ActualTotalInstances)
			message := fmt.Sprintf("%s\n%s", title, CodeBlock(msg))

			if err := sendAlert(message, "status"); err != nil {
				return fmt.Errorf("failed to send instance count mismatch alert: %w", err)
			}
			mm.AlertState.InstanceCountAlerted = true
		}
	} else {
		// 불일치가 해결된 경우
		if mm.AlertState.InstanceCountAlerted {
			title := "✅ Instance Count Mismatch Resolved"
			msg := fmt.Sprintf("Vast.ai instances: %d\nActual instances: %d", mm.User.TotalInstances, mm.User.ActualTotalInstances)
			message := fmt.Sprintf("%s\n%s", title, CodeBlock(msg))

			if err := sendAlert(message, "status"); err != nil {
				return fmt.Errorf("failed to send instance count mismatch recovery alert: %w", err)
			}
			mm.AlertState.InstanceCountAlerted = false
		}
		// 불일치 시작 시간 초기화
		mm.AlertState.InstanceMismatchStart = time.Time{}
	}

	return nil
}

// checkCredit는 credit이 최소 기준보다 낮은지 체크합니다
func (m *Client) checkCredit(mm *MinuteMetrics, config AlertConfig, sendAlert func(string, string) error) error {
	if !config.Enabled {
		return nil
	}

	// Vast.ai가 활성화된 경우에만 체크 (VastaiCredit이 nil이 아닌 경우)
	if mm.User.VastaiCredit == nil {
		return nil
	}

	// credit 잔액 체크
	if mm.User.VastaiCredit.Credit <= mm.User.TotalDailyCost && !mm.AlertState.CreditAlerted {
		title := "⚠️ Credit Alert"
		msg := fmt.Sprintf("Vast.ai balance: $%.2f\nDaily cost: $%.2f", mm.User.VastaiCredit.Credit, mm.User.TotalDailyCost)
		message := fmt.Sprintf("%s\n%s", title, CodeBlock(msg))

		if err := sendAlert(message, "status"); err != nil {
			return fmt.Errorf("failed to send credit alert: %w", err)
		}

		mm.AlertState.CreditAlerted = true
	} else if mm.User.VastaiCredit.Credit > mm.User.TotalDailyCost && mm.AlertState.CreditAlerted {
		title := "✅ Credit Balance Recovered"
		msg := fmt.Sprintf("Vast.ai balance: $%.2f\nDaily cost: $%.2f", mm.User.VastaiCredit.Credit, mm.User.TotalDailyCost)
		message := fmt.Sprintf("%s\n%s", title, CodeBlock(msg))

		if err := sendAlert(message, "status"); err != nil {
			return fmt.Errorf("failed to send credit recovery alert: %w", err)
		}

		mm.AlertState.CreditAlerted = false
	}

	return nil
}

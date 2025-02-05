package main

// Config 관련 구조체
type TelegramConfig struct {
	BotToken      string `yaml:"token"`
	ChatID        string `yaml:"chat_id"`
	MessageThread int    `yaml:"message_thread"` // thread_ids 대신 단일 thread 사용
}

type Config struct {
	Kuzco struct {
		ID       string `yaml:"id"`
		Password string `yaml:"password"`
	} `yaml:"kuzco"`
	Telegram TelegramConfig `yaml:"telegram"`
}

// API 요청/응답 구조체
type LoginRequest struct {
	JSON struct {
		Email          string      `json:"email"`
		Password       string      `json:"password"`
		TwoFactorToken interface{} `json:"twoFactorToken"`
	} `json:"json"`
	Meta struct {
		Values struct {
			TwoFactorToken []string `json:"twoFactorToken"`
		} `json:"values"`
	} `json:"meta"`
}

type LoginResponse struct {
	Result struct {
		Data struct {
			JSON struct {
				Status string `json:"status"`
				Token  string `json:"token"`
			} `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

type CountOnlineRequest struct {
	JSON interface{} `json:"json"`
	Meta struct {
		Values []string `json:"values"`
	} `json:"meta"`
}
type CountOnlineResponse struct {
	Result struct {
		Data struct {
			JSON struct {
				Status string `json:"status"`
				Count  int    `json:"count"`
			} `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

type RPMResponse struct {
	Result struct {
		Data struct {
			JSON int `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

type GlobalWorkerResponse struct {
	Result struct {
		Data struct {
			JSON struct {
				TokensLast24Hours int64 `json:"tokensLast24Hours"`
			} `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

type MetricsData struct {
	GenerationsLast24Hours int   `json:"generationsLast24Hours"`
	TokensLast24Hours      int64 `json:"tokensLast24Hours"`
	TokensAllTime          int64 `json:"tokensAllTime"`
}

type UserMetricsResponse struct {
	Result struct {
		Data struct {
			JSON MetricsData `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

type WorkerMetricsResponse struct {
	Result struct {
		Data struct {
			JSON MetricsData `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

type WorkerInstance struct {
	ID     string `json:"_id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Info   struct {
		Runtime   string `json:"runtime"`
		IPAddress string `json:"ipAddress"`
		Country   string `json:"country"`
		City      string `json:"city"`
	} `json:"info"`
}

type Worker struct {
	ID         string           `json:"_id"`
	Name       string           `json:"name"`
	IsArchived bool             `json:"isArchived"`
	Instances  []WorkerInstance `json:"instances"`
}

type WorkerListResponse struct {
	Result struct {
		Data struct {
			JSON struct {
				Status  string   `json:"status"`
				Workers []Worker `json:"workers"`
			} `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

// 워커 상태 추적을 위한 구조체
type WorkerStatus struct {
	InstanceCount int
	Instances     map[string]string // instanceID -> status
}

type MonitorState struct {
	Workers map[string]*WorkerStatus // workerID -> WorkerStatus
}

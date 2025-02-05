package main

import "time"

// Config 관련 구조체
type TelegramConfig struct {
	BotToken string `yaml:"token"`
	ChatID   string `yaml:"chat_id"`
	Threads  struct {
		Daily   int `yaml:"daily"`
		Hourly  int `yaml:"hourly"`
		Error   int `yaml:"error"`
		Status  int `yaml:"status"`
	} `yaml:"threads"`
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

type NvidiaSmiGPU struct {
	PCI []struct {
		PCIBus      []string `json:"pci_bus"`
		PCIBusID    []string `json:"pci_bus_id"`
		PCIDevice   []string `json:"pci_device"`
		PCIDomain   []string `json:"pci_domain"`
		PCIDeviceID []string `json:"pci_device_id"`
		RXUtil      []string `json:"rx_util"`
		TXUtil      []string `json:"tx_util"`
	} `json:"pci"`
	UUID         []string `json:"uuid"`
	ProductName  []string `json:"product_name"`
	ProductBrand []string `json:"product_brand"`
	Temperature  []struct {
		GPUTemp []string `json:"gpu_temp"`
	} `json:"temperature"`
	Utilization []struct {
		GPUUtil    []string `json:"gpu_util"`
		MemoryUtil []string `json:"memory_util"`
	} `json:"utilization"`
	FBMemoryUsage []struct {
		Free     []string `json:"free"`
		Used     []string `json:"used"`
		Total    []string `json:"total"`
		Reserved []string `json:"reserved"`
	} `json:"fb_memory_usage"`
	PowerReadings []struct {
		PowerDraw         []string `json:"power_draw"`
		PowerState        []string `json:"power_state"`
		MaxPowerLimit     []string `json:"max_power_limit"`
		CurrentPowerLimit []string `json:"current_power_limit"`
	} `json:"gpu_power_readings"`
}

type NvidiaSmi struct {
	GPU           []NvidiaSmiGPU `json:"gpu"`
	Timestamp     []string       `json:"timestamp"`
	CUDAVersion   []string       `json:"cuda_version"`
	AttachedGPUs  []string       `json:"attached_gpus"`
	DriverVersion []string       `json:"driver_version"`
}

type WorkerInstance struct {
	ID              string           `json:"_id"`
	Name            string           `json:"name"`
	Status          string           `json:"status"`
	PoolAssignments []PoolAssignment `json:"poolAssignments"`
	Info            struct {
		InstanceID       string     `json:"instanceId"`
		Runtime          string     `json:"runtime"`
		IPAddress        string     `json:"ipAddress"`
		Country          string     `json:"country"`
		City             string     `json:"city"`
		Continent        string     `json:"continent"`
		Region           string     `json:"region"`
		RegionCode       string     `json:"regionCode"`
		Timezone         string     `json:"timezone"`
		Arch             string     `json:"arch"`
		Platform         string     `json:"platform"`
		TotalMemoryBytes int64      `json:"totalMemoryBytes"`
		TotalSwapBytes   int64      `json:"totalSwapBytes"`
		KernelVersion    string     `json:"kernelVersion"`
		OSVersion        string     `json:"osVersion"`
		HostName         string     `json:"hostName"`
		CPUs             int        `json:"cpus"`
		Version          string     `json:"version"`
		NvidiaSmi        *NvidiaSmi `json:"nvidiaSmi"`
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
type InstanceStatus struct {
	Name      string
	Status    string
	IPAddress string
	Runtime   string
	GPU       struct {
		Name        string
		Temp        string
		Utilization string
		Memory      struct {
			Used  string
			Total string
		}
		Power struct {
			Draw  string
			State string
		}
	}
}

type TokenMetrics struct {
	GenerationsCount int64
	TokensCount      int64
	LastUpdated      time.Time
}

type TokenCache struct {
	GlobalTokens TokenMetrics
	UserTokens   TokenMetrics
	WorkerTokens map[string]TokenMetrics // workerID -> TokenMetrics

}
type WorkerStatus struct {
	Name          string
	InstanceCount int
	Instances     map[string]InstanceStatus // instanceID -> InstanceStatus
}

type MonitorState struct {
	Workers    map[string]*WorkerStatus // workerID -> WorkerStatus
	TokenCache TokenCache
}

type PoolAssignment struct {
	Model    string `json:"model"`
	Lane     string `json:"lane"`
	LaneInfo struct {
		Runtime string `json:"runtime"`
	} `json:"lane-info"`
}

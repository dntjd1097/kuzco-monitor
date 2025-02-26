package api

// Metrics endpoints
const (
	EndpointMetricsRunningInstanceCount   = "metrics.runningInstanceCount"   // General
	EndpointMetricsRPM                    = "metrics.rpm"                    // General
	EndpointMetricsTokensLast24Hours      = "metrics.tokensLast24Hours"      // General, User
	EndpointMetricsGenerationsLast24Hours = "metrics.generationsLast24Hours" // General, User
	EndpointMetricsTokensAllTime          = "metrics.tokensAllTime"          // User
	EndpointMetricsGenerationsHistory     = "metrics.generationsHistory"     // General, User
	EndpointMetricsTokensHistory          = "metrics.tokenEarningsHistory"   // General, User
)

// Auth endpoints
const (
	EndpointUserLogin = "user.login"
)

// System endpoints
const (
	EndpointSystemBucketVersions = "system.bucketVersions"
)

const (
	KuzcoAPI  = "https://relay.inference.supply/api/trpc/"
	VastaiAPI = "https://console.vast.ai/api/v0/"
)

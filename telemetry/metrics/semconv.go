package metrics

const (
	// MetricHTTPServerRequestDuration is the OTel semantic HTTP server duration metric.
	MetricHTTPServerRequestDuration = "http.server.request.duration"
	// MetricHTTPServerActiveRequests is the OTel semantic HTTP active requests metric.
	MetricHTTPServerActiveRequests = "http.server.active_requests"
	// MetricGoGoroutineCount is the OTel semantic Go goroutine count metric.
	MetricGoGoroutineCount = "go.goroutine.count"
	// MetricGoMemoryAllocated is the OTel semantic Go heap allocation bytes metric.
	MetricGoMemoryAllocated = "go.memory.allocated"
	// MetricGoMemoryAllocations is the OTel semantic Go heap allocation count metric.
	MetricGoMemoryAllocations = "go.memory.allocations"
	// MetricGoMemoryGCGoal is the OTel semantic Go GC heap goal metric.
	MetricGoMemoryGCGoal = "go.memory.gc.goal"
	// MetricGoMemoryLimit is the OTel semantic Go memory limit metric.
	MetricGoMemoryLimit = "go.memory.limit"
	// MetricGoMemoryUsed is the OTel semantic Go runtime memory usage metric.
	MetricGoMemoryUsed = "go.memory.used"
	// MetricGoProcessorLimit is the OTel semantic Go processor limit metric.
	MetricGoProcessorLimit = "go.processor.limit"
	// MetricBusinessPrefix is the recommended prefix for business-domain metrics.
	MetricBusinessPrefix = "business."
)

const (
	AttrHTTPRequestMethod       = "http.request.method"
	AttrHTTPRoute               = "http.route"
	AttrHTTPResponseStatusCode  = "http.response.status_code"
	AttrURLScheme               = "url.scheme"
	AttrErrorType               = "error.type"
	AttrBusinessOperation       = "business.operation"
	AttrBusinessOperationStatus = "business.operation.status"
)

package metrics

const (
	// MetricHTTPServerRequestDuration is the OTel semantic HTTP server duration metric.
	MetricHTTPServerRequestDuration = "http.server.request.duration"
	// MetricHTTPServerActiveRequests is the OTel semantic HTTP active requests metric.
	MetricHTTPServerActiveRequests = "http.server.active_requests"
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

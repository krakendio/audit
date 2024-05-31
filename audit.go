package audit

import (
	"github.com/luraproject/lura/v2/config"
)

// Audit audits the received configuration and generates an AuditResult with all the Recommendations
func Audit(cfg *config.ServiceConfig, ignore, severities []string) (AuditResult, error) {
	service := Parse(cfg)

	res := AuditResult{Recommendations: []Recommendation{}}
	keysToIgnore := map[string]struct{}{}
	for _, k := range ignore {
		keysToIgnore[k] = struct{}{}
	}
	severitiesToCatch := map[string]struct{}{}
	for _, k := range severities {
		severitiesToCatch[k] = struct{}{}
	}

	for i := range ruleSet {
		if _, ok := keysToIgnore[ruleSet[i].Recommendation.Rule]; ok {
			continue
		}

		if _, ok := severitiesToCatch[ruleSet[i].Recommendation.Severity]; !ok {
			continue
		}

		if ruleSet[i].Evaluate(&service) {
			res.Recommendations = append(res.Recommendations, ruleSet[i].Recommendation)
		}
	}

	return res, nil
}

const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityMedium   = "MEDIUM"
	SeverityLow      = "LOW"
)

// Rule encapsulates a recommendation and an evaluation function that determines if the recommendation
// applies for a given service definition
type Rule struct {
	Recommendation Recommendation
	Evaluate       func(*Service) bool
}

// NewRule creates a Rule with the given arguments
func NewRule(id, severity, msg string, ef func(*Service) bool) Rule {
	return Rule{
		Recommendation: Recommendation{
			Rule:     id,
			Severity: severity,
			Message:  msg,
		},
		Evaluate: ef,
	}
}

// AuditResult contains all the recommendations and stats generated by the audit process
type AuditResult struct {
	Recommendations []Recommendation `json:"recommendations"`
	Stats           Stats            `json:"stats"`
}

// Recommendation maps a rule id with a severity and a message
type Recommendation struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// Stats is an empty struct that will be completed in the future
type Stats struct{}

var ruleSet = []Rule{
	/*
	   Section 1: Security
	*/
	NewRule("1.1.1", SeverityHigh, "Implement more secure alternatives than Basic Auth to protect your data.", hasBasicAuth),
	NewRule("1.1.2", SeverityMedium, "Implement stateless authorization methods such as JWT to secure your endpoints as opposed to using API keys.", hasApiKeys),
	NewRule("1.2.1", SeverityHigh, "Prioritize using JWT for endpoint authorization to ensure security.", hasNoJWT),

	/*
	   Section 2: Service level recommendations
	*/
	NewRule("2.1.1", SeverityHigh, "Only allow secure connections (avoid insecure_connections).", hasInsecureConnections),
	NewRule("2.1.2", SeverityHigh, "Enable TLS or use a terminator in front of KrakenD.", hasNoTLS),
	NewRule("2.1.3", SeverityCritical, "TLS is configured but its disable flag prevents from using it.", hasTLSDisabled),
	NewRule("2.1.7", SeverityHigh, "Enable HTTP security header checks (security/http).", hasNoHTTPSecure),
	NewRule("2.1.8", SeverityHigh, "Avoid clear text communication (h2c).", hasH2C),
	NewRule("2.1.9", SeverityLow, "Establish secure connections in internal traffic (avoid insecure_connections internally)", hasBackendInsecureConnections),
	NewRule("2.2.1", SeverityMedium, "Hide the version banner in runtime.", hasNoObfuscatedVersionHeader),
	NewRule("2.2.2", SeverityHigh, "Enable CORS.", hasNoCORS),
	NewRule("2.2.3", SeverityHigh, "Avoid passing all input headers to the backend.", hasHeadersWildcard),
	NewRule("2.2.4", SeverityHigh, "Avoid passing all input query strings to the backend.", hasQueryStringWildcard),
	NewRule("2.2.5", SeverityLow, "Avoid exposing gRPC server without services declared.", hasEmptyGRPCServer),

	/*
	   Section 3: Traffic management / rate limits
	*/
	NewRule("3.1.1", SeverityLow, "Enable a bot detector.", hasBotdetectorDisabled),
	NewRule("3.1.2", SeverityHigh, "Implement a rate-limiting strategy and avoid having an All-You-Can-Eat API.", hasNoRatelimit),
	NewRule("3.1.3", SeverityHigh, "Protect your backends with a circuit breaker.", hasNoCB),
	NewRule("3.3.1", SeverityLow, "Set timeouts to below 3 seconds for improved performance.", hasTimeoutBiggerThan(3000)),
	NewRule("3.3.2", SeverityMedium, "Set timeouts to below 5 seconds for improved performance.", hasTimeoutBiggerThan(5000)),
	NewRule("3.3.3", SeverityHigh, "Set timeouts to below 30 seconds for improved performance.", hasTimeoutBiggerThan(30000)),
	NewRule("3.3.4", SeverityCritical, "Set timeouts to below 1 minute for improved performance.", hasTimeoutBiggerThan(60000)),

	/*
	   Section 4 : Telemetry
	*/
	NewRule("4.1.1", SeverityMedium, "Implement a telemetry system for collecting metrics for monitoring and troubleshooting.", hasNoMetrics),
	NewRule("4.1.2", SeverityMedium, "Give your configuration a name for easy identification in metric tracking.", hasTelemetryMissingName),
	NewRule("4.1.3", SeverityHigh, "Avoid duplicating telemetry options to prevent system overload.", hasSeveralTelemetryComponents),
	NewRule("4.2.1", SeverityMedium, "Implement a telemetry system for tracing for monitoring and troubleshooting.", hasNoTracing),
	NewRule("4.3.1", SeverityMedium, "Use the improved logging component for better log parsing.", hasNoLogging),
	/*
	   Section 5: Endpoint level audit
	*/
	NewRule("5.1.1", SeverityLow, "Follow a RESTful endpoint structure for improved readability and maintainability.", hasRestfulDisabled),
	NewRule("5.1.2", SeverityLow, "Disable the /__debug/ endpoint for added security.", hasDebugEnabled),
	NewRule("5.1.3", SeverityLow, "Disable the /__echo/ endpoint for added security.", hasEchoEnabled),
	NewRule("5.1.4", SeverityLow, "Declare explicit endpoints instead of using wildcards.", hasEndpointWildcard),
	NewRule("5.1.5", SeverityMedium, "Declare explicit endpoints instead of using /__catchall.", hasEndpointCatchAll),
	NewRule("5.1.6", SeverityMedium, "Avoid using multiple write methods in endpoint definitions.", hasMultipleUnsafeMethods),
	NewRule("5.1.7", SeverityMedium, "Avoid using sequential proxy.", hasSequentialProxy),
	NewRule("5.2.1", SeverityCritical, "Ensure all endpoints have at least one backend for proper functionality.", hasEndpointWithoutBackends),
	NewRule("5.2.2", SeverityLow, "Benefit from the backend for frontend pattern capabilities.", hasASingleBackendPerEndpoint),
	NewRule("5.2.3", SeverityLow, "Avoid coupling clients by overusing no-op encoding.", hasAllEndpointsAsNoop),

	/*
	   Section 6: Async agents.
	*/
	NewRule("6.1.1", SeverityLow, "Ensure Async Agents do not start sequentially to avoid overloading the system (+10 agents).", hasSequentialStart),

	/*
	   Section 7: Deprecations
	*/
	// 7.1 Plugin Deprecations:
	NewRule("7.1.1", SeverityHigh, "Do not use deprecated plugin virtualhost. Please visit https://www.krakend.io/docs/enterprise/service-settings/virtual-hosts/#upgrading-from-the-old-plugin-before-v24 to upgrade to the new virtualhost.", hasDeprecatedServerPlugin("virtualhost")),
	NewRule("7.1.2", SeverityHigh, "Do not use deprecated plugin static-filesystem. Please visit https://www.krakend.io/docs/enterprise/endpoints/serve-static-content/#upgrading-from-the-old-plugin-before-v24 to upgrade to the new static-filesystem.", hasDeprecatedServerPlugin("static-filesystem")),
	NewRule("7.1.3", SeverityHigh, "Do not use deprecated plugin basic-auth. Please move your configuration to the namespace auth/basic to use the new component. See: https://www.krakend.io/docs/enterprise/authentication/basic-authentication/ .", hasDeprecatedServerPlugin("basic-auth")),
	NewRule("7.1.4", SeverityHigh, "Do not use deprecated plugin wildcard. Please visit https://www.krakend.io/docs/enterprise/endpoints/wildcard/#upgrading-from-the-old-wildcard-plugin-before-v23 to upgrade to the new Wildcard.", hasDeprecatedServerPlugin("wildcard")),

	NewRule("7.1.5", SeverityHigh, "Do not use deprecated plugin http-proxy. Please visit https://www.krakend.io/docs/enterprise/backends/http-proxy/#migration-from-old-plugin to upgrade to the new options.", hasDeprecatedClientPlugin("http-proxy")),
	NewRule("7.1.6", SeverityHigh, "Do not use deprecated plugin static-filesystem. Please visit https://www.krakend.io/docs/enterprise/endpoints/serve-static-content/#upgrading-from-the-old-plugin-before-v24 to upgrade to the new static-filesystem.", hasDeprecatedClientPlugin("static-filesystem")),
	NewRule("7.1.7", SeverityHigh, "Do not use deprecated plugin no-redirect. Please visit https://www.krakend.io/docs/enterprise/backends/client-redirect/#migration-from-old-plugin to upgrade to the new options.", hasDeprecatedClientPlugin("no-redirect")),

	// 7.2 Component Deprecations
	NewRule("7.2.1", SeverityHigh, "Do not use deprecated component telemetry/ganalytics.", hasDeprecatedGanalytics),
	NewRule("7.2.2", SeverityHigh, "Do not use deprecated component telemetry/instana.", hasDeprecatedInstana),
	NewRule("7.2.3", SeverityHigh, "Do not use deprecated component telemetry/instana.", hasDeprecatedOpenCensus),
}

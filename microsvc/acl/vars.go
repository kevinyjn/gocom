package acl

// Constants
var (
	BaseRouteUsers  = "/users"
	RouteUserLogin  = "/login"
	RouteUserSignup = "/signup"

	FullLoginURLPath = BaseRouteUsers + RouteUserLogin

	HeaderAuthorization = "Authorization"
	HeaderSignature     = "Signature"

	DefaultSignatureAppKey = ""

	ValidateFailureDurationSecondsThreshold = 60
	ValidateFailureCountThreshold           = 15
	BlockingIPDurationSeconds               = 900

	// DefaultUsers users<name: password> were used as localhost access
	DefaultUsers = map[string]string{"guest": "guest"}
)

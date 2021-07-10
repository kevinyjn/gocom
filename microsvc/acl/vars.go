package acl

// Constants
var (
	BaseRouteUsers   = "/authenticates"
	RouteUserSignin  = "/signin"
	RouteUserSignup  = "/signup"
	RouteUserSignout = "/signout"

	FullLoginURLPath = BaseRouteUsers + RouteUserSignin

	HeaderAuthorization = "Authorization"
	HeaderSignature     = "Signature"

	DefaultSignatureAppKey = ""

	ValidateFailureDurationSecondsThreshold = 60
	ValidateFailureCountThreshold           = 15
	BlockingIPDurationSeconds               = 900

	SessionExpireHours = 24

	// DefaultUsers users<name: password> were used as localhost access
	DefaultUsers = map[string]string{"guest": "guest"}
)

// LoginParam form data
type LoginParam struct {
	Name     string `json:"name" form:"name" validate:"required" label:"名称"`
	Password string `json:"password" form:"password" validate:"required" label:"密码"`
}

package gcompliance

const (
	PCWeb          = "pcweb"
	H5             = "h5"
	AndroidAPP     = "android_app"
	AndroidWebview = "android_webview"
	IOSAPP         = "ios_app"
	IOSWebview     = "ios_webview"
	Internal       = "internal"
	CRMAPP         = "crm_app"
	OpenAPI        = "openapi"
	FIX            = "fix"
	Unknown        = ""
)

var sources = map[string]string{
	SourceWeb:      SourceWeb,
	SourceApp:      SourceApp,
	SourceOpenapi:  SourceOpenapi,
	PCWeb:          SourceWeb,
	H5:             SourceWeb,
	AndroidWebview: SourceWeb,
	IOSWebview:     SourceWeb,
	AndroidAPP:     SourceApp,
	IOSAPP:         SourceApp,
	OpenAPI:        SourceOpenapi,
}

// 其他以空值处理
func getSource(s string) string {
	return sources[s]
}

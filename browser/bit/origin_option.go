package bit

type FingerPrint struct {
	CoreProduct                string `json:"coreProduct" default:"chrome"`
	CoreVersion                string `json:"coreVersion" default:"130"`
	Ostype                     string `json:"ostype" validate:"oneof=Windows macOS Linux Android IOS" default:"Windows"`
	Os                         string `json:"os" validate:"oneof=Win32 MacIntel 'Linux x86_64' iPhone 'Linux armv81'" default:"Win32"`
	OsVersion                  string `json:"osVersion"`
	Version                    string `json:"version"`
	UserAgent                  string `json:"userAgent"`
	IsIpCreateTimeZone         bool   `json:"isIpCreateTimeZone" default:"true"`
	TimeZone                   string `json:"timeZone"`
	TimeZoneOffset             int    `json:"timeZoneOffset"`
	WebRTC                     string `json:"webRTC" default:"3"`
	IgnoreHttpsErrors          bool   `json:"ignoreHttpsErrors"`
	Position                   string `json:"position" default:"1"`
	IsIpCreatePosition         bool   `json:"isIpCreatePosition" default:"true"`
	Lat                        string `json:"lat"`
	Lng                        string `json:"lng"`
	PrecisionData              string `json:"precisionData"`
	IsIpCreateLanguage         bool   `json:"isIpCreateLanguage" default:"true"`
	Languages                  string `json:"languages"`
	IsIpCreateDisplayLanguage  bool   `json:"isIpCreateDisplayLanguage"`
	DisplayLanguages           string `json:"displayLanguages"`
	OpenWidth                  int    `json:"openWidth" default:"1280"`
	OpenHeight                 int    `json:"openHeight" default:"720"`
	ResolutionType             string `json:"resolutionType" default:"0"`
	Resolution                 string `json:"resolution" default:"1920 x 1080"`
	WindowSizeLimit            bool   `json:"windowSizeLimit" default:"true"`
	DevicePixelRatio           int    `json:"devicePixelRatio" default:"1"`
	FontType                   string `json:"fontType" default:"2"`
	Canvas                     string `json:"canvas" default:"0"`
	WebGL                      string `json:"webGL" default:"0"`
	WebGLMeta                  string `json:"webGLMeta" default:"0"`
	WebGLManufacturer          string `json:"webGLManufacturer"`
	WebGLRender                string `json:"webGLRender"`
	AudioContext               string `json:"audioContext" default:"0"`
	MediaDevice                string `json:"mediaDevice" default:"0"`
	SpeechVoices               string `json:"speechVoices" default:"0"`
	HardwareConcurrency        string `json:"hardwareConcurrency" default:"4"`
	DeviceMemory               string `json:"deviceMemory" default:"8"`
	DoNotTrack                 string `json:"doNotTrack" default:"1"`
	ClientRectNoiseEnabled     bool   `json:"clientRectNoiseEnabled" default:"true"`
	PortScanProtect            string `json:"portScanProtect" default:"0"`
	PortWhiteList              string `json:"portWhiteList"`
	DeviceInfoEnabled          bool   `json:"deviceInfoEnabled" default:"true"`
	ComputerName               string `json:"computerName"`
	MacAddr                    string `json:"macAddr"`
	DisableSslCipherSuitesFlag bool   `json:"disableSslCipherSuitesFlag"`
	DisableSslCipherSuites     string `json:"disableSslCipherSuites"`
	EnablePlugins              bool   `json:"enablePlugins"`
	Plugins                    string `json:"plugins"`
	LaunchArgs                 string `json:"launchArgs"`
}
type OriginOption struct {
	Platform                    string       `json:"platform"`
	Url                         string       `json:"url"`
	Remark                      string       `json:"remark"`
	UserName                    string       `json:"userName"`
	Password                    string       `json:"password"`
	IsSynOpen                   int          `json:"isSynOpen"`
	FaSecretKey                 string       `json:"faSecretKey"`
	Cookie                      string       `json:"cookie"`
	ProxyMethod                 int          `json:"proxyMethod"`
	ProxyType                   string       `json:"proxyType"`
	Host                        string       `json:"host"`
	Port                        int          `json:"port"`
	IpCheckService              string       `json:"ipCheckService"`
	IsIpv6                      string       `json:"isIpv6"`
	ProxyUserName               string       `json:"proxyUserName"`
	ProxyPassword               string       `json:"proxyPassword"`
	RefreshProxyUrl             string       `json:"refreshProxyUrl"`
	Country                     string       `json:"country"`
	Province                    string       `json:"province"`
	City                        string       `json:"city"`
	Workbench                   string       `json:"workbench"`
	AbortImage                  bool         `json:"abortImage"`
	AbortImageMaxSize           int          `json:"abortImageMaxSize"`
	AbortMedia                  bool         `json:"abortMedia"`
	MuteAudio                   bool         `json:"muteAudio"`
	StopWhileNetError           bool         `json:"stopWhileNetError"`
	StopWhileCountryChange      bool         `json:"stopWhileCountryChange"`
	DynamicIpUrl                string       `json:"dynamicIpUrl"`
	DynamicIpChannel            string       `json:"dynamicIpChannel"`
	IsDynamicIpChangeIp         bool         `json:"isDynamicIpChangeIp"`
	DuplicateCheck              int          `json:"duplicateCheck"`
	IsGlobalProxyInfo           bool         `json:"isGlobalProxyInfo"`
	SyncTabs                    bool         `json:"syncTabs"`
	SyncCookies                 bool         `json:"syncCookies"`
	SyncIndexedDb               bool         `json:"syncIndexedDb"`
	SyncLocalStorage            bool         `json:"syncLocalStorage"`
	SyncBookmarks               bool         `json:"syncBookmarks"`
	SyncAuthorization           bool         `json:"syncAuthorization"`
	CredentialsEnableService    bool         `json:"credentialsEnableService"`
	SyncHistory                 bool         `json:"syncHistory"`
	SyncExtensions              bool         `json:"syncExtensions"`
	IsValidUsername             bool         `json:"isValidUsername"`
	AllowedSignin               bool         `json:"allowedSignin"`
	ClearCacheFilesBeforeLaunch bool         `json:"clearCacheFilesBeforeLaunch"`
	ClearCacheWithoutExtensions bool         `json:"clearCacheWithoutExtensions"`
	ClearCookiesBeforeLaunch    bool         `json:"clearCookiesBeforeLaunch"`
	ClearHistoriesBeforeLaunch  bool         `json:"clearHistoriesBeforeLaunch"`
	DisableGpu                  bool         `json:"disableGpu"`
	DisableTranslatePopup       bool         `json:"disableTranslatePopup"`
	DisableNotifications        bool         `json:"disableNotifications"`
	DisableClipboard            bool         `json:"disableClipboard"`
	MemorySaver                 bool         `json:"memorySaver"`
	RandomFingerprint           bool         `json:"randomFingerprint"`
	FingerPrint                 *FingerPrint `json:"browserFingerPrint" validate:"required" default:""`
}

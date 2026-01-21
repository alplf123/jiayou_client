package model

type BindLoginUser struct {
	Name      string `json:"name" binding:"required"`
	Pwd       string `json:"pwd" binding:"required"`
	Captcha   string `json:"captcha"`
	CaptchaId string `json:"captcha_id"`
}
type BindUser struct {
	Name   string     `form:"name" binding:"required,min=3,max=20"`
	Pwd    string     `form:"pwd" binding:"omitempty,len=32"`
	Role   Role       `form:"role"`
	Desc   string     `form:"desc" binding:"max=255"`
	Phone  string     `form:"phone"`
	Avatar string     `form:"avatar"`
	Group  string     `form:"group"`
	Status UserStatus `form:"status" binding:"oneof=baned actived"`
}
type BindUserGroup struct {
	Name string `json:"name" binding:"required,min=2,max=20"`
	Desc string `json:"desc" binding:"max=255"`
}
type BindTiktokDeviceGroup struct {
	BindId
	Name string `json:"name" binding:"required,min=3,max=20"`
	Desc string `json:"desc" binding:"max=255"`
}

type BindTiktokDeviceCreate struct {
	Name     string   `form:"name" binding:"required,min=3,max=20"`
	Pwd      string   `form:"pwd" binding:"required,gt=0"`
	Os       Os       `form:"os" binding:"required,oneof=windows android"`
	Platform Platform `form:"platform" binding:"required,oneof=pc mobile"`
	Baned    bool     `form:"baned"`
	Priority int      `json:"priority" binding:"required,min=0,max=100"`
	Region   string   `form:"region"`
	Group    string   `form:"group"`
}
type BindTiktokDeviceUpdate struct {
	BindId
	Name     string   `form:"name" binding:"required,min=3,max=20"`
	Pwd      string   `form:"pwd" binding:"required,gt=0"`
	Os       Os       `form:"os" binding:"required,oneof=windows android"`
	Platform Platform `form:"platform" binding:"required,oneof=pc mobile"`
	Baned    bool     `form:"baned"`
	Priority int      `json:"priority" binding:"required,min=0,max=100"`
	Region   string   `form:"region"`
	Group    string   `form:"group"`
}

type BindPage struct {
	Page     int `json:"page,default=20"`
	PageSize int `json:"pageSize,default=1"`
}
type BindGroupPage struct {
	BindPage
	Name string `json:"name"`
}
type BindUserPage struct {
	BindPage
	Group string `json:"group"`
	Name  string `json:"name"`
}
type BindId struct {
	Id string `json:"id" binding:"required"`
}
type BindGroupUpdate struct {
	BindId
	Name string `json:"name"`
	Desc string `json:"desc"`
}
type BindDevicePage struct {
	BindPage
	Name  string `json:"name"`
	Group string `json:"group"`
}
type BindDeviceGroupPage struct {
	BindPage
	Name string `json:"name"`
}
type BindTaskGroupPage struct {
	BindPage
	Name string `json:"name"`
}
type BindTaskPage struct {
	BindPage
	Group string `json:"group"`
}

type BinSyncTiktokDevice struct {
	Name    string `json:"name" binding:"required"`
	Avatar  string `json:"avatar"`
	Headers string `json:"headers" binding:"required"`
	Cookie  string `json:"cookie" binding:"required"`
	Info    string `json:"info" binding:"required"`
}

type BindTaskGroup struct {
	Name         string       `json:"name" binding:"required,max=255" `
	TaskType     TaskType     `json:"task_type" binding:"oneof=1 2"`
	TaskArgs     string       `json:"task_args" binding:"required"`
	Devices      string       `json:"devices"`
	DeviceGroup  string       `json:"device_group"`
	PostInterval int          `json:"post_interval" binding:"gte=0"`
	StartAt      *DateTime    `json:"start_at"`
	SortBy       DeviceSortBy `json:"sort_by" binding:"oneof=0 1"`
}

type BindUploadVideoItems struct {
	Name string `json:"name" binding:"required"`
	Text string `json:"text" binding:"required"`
}
type BindUploadVideoParams struct {
	Items          []BindUploadVideoItems `json:"items" binding:"required"`
	AllowComment   bool                   `json:"allow_comment"`
	VisibilityType int                    `json:"visibility_type" binding:"oneof=0 1 2"`
}
type BindCommentParams struct {
	Url        string `json:"url" binding:"url"`
	Level      int    `json:"level" binding:"oneof=0 1"`
	ReplyUser  string `json:"reply_user" binding:"required_if=Level 1"`
	ReplyFile  string `json:"reply_file" binding:"required_if=Level 1"`
	ReplyLines int    `json:"reply_lines" binding:"gt=0"`
}

type BindSystemSetting struct {
	AdsKey       string `json:"ads_key" binding:"required"`
	ProxyType    string `json:"proxy_type" binding:"required"`
	ProxyAddress string `json:"proxy_address" binding:"required"`
	ProxyAccount string `json:"proxy_account"`
	ProxyPwd     string `json:"proxy_pwd"`
}

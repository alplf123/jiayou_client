package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"jiayou_backend_spider/request"
	"net/url"
	"strings"
	"time"

	"gorm.io/plugin/soft_delete"
)

type DeviceStatus string

const (
	DeviceInActived DeviceStatus = "inactived"
	DeviceActived   DeviceStatus = "actived"
	DeviceExpired   DeviceStatus = "expired"
	DeviceBaned     DeviceStatus = "baned"
	DeviceInvalid   DeviceStatus = "invalid"
)

type DeviceSortBy int

const (
	Priority DeviceSortBy = iota
	LastUseAt
)

type Platform string

const (
	Mobile Platform = "mobile"
	Pc     Platform = "pc"
)

type VerifyType string

const (
	Email VerifyType = "email"
	TwoFA VerifyType = "2fa"
)

type Os string

const (
	Windows Platform = "windows"
	Android          = "android"
)

type UserStatus string

const (
	UserActived UserStatus = "actived"
	UserBanded  UserStatus = "baned"
)

type TaskStatus string

const (
	Failed   = "failed"
	Running  = "running"
	Finished = "finished"
	Standby  = "standby"
)

type TaskType int

const (
	VideoComment TaskType = iota
	VideoPublish
	VideoAddComment
)

type DateTime time.Time

func (datetime *DateTime) MarshalJSON() ([]byte, error) {
	return json.Marshal((time.Time)(*datetime).Format(time.DateTime))
}
func (datetime *DateTime) UnmarshalJSON(data []byte) error {
	var dt string
	if err := json.Unmarshal(data, &dt); err != nil {
		return err
	}
	t, err := time.Parse(time.DateTime, dt)
	if err != nil {
		return err
	}

	*datetime = DateTime(t)
	return nil
}
func (datetime *DateTime) Value() (driver.Value, error) {
	if datetime == nil {
		return nil, nil
	}
	return time.Time(*datetime), nil
}

type Base struct {
	ID        uint                  `gorm:"primarykey" json:"-"`
	CreatedAt *DateTime             `json:"created_at"`
	UpdatedAt *DateTime             `json:"-"`
	DeletedAt soft_delete.DeletedAt `gorm:"uniqueIndex:unique_idx" json:"-"`
}

type TiktokDevice struct {
	Base
	TiktokDeviceGroupID uint              `json:"-"`
	TiktokDeviceGroup   TiktokDeviceGroup `gorm:"foreignKey:TiktokDeviceGroupID" json:"group"`
	BindName            string            `gorm:"not null;uniqueIndex:unique_idx;size:255" json:"bind_name"`
	BindPwd             string            `gorm:"not null;size:255" json:"bind_pwd"`
	Avatar              string            `json:"avatar"`
	DeviceId            string            `gorm:"index;size:100" json:"device_id"`
	UniqueId            string            `gorm:"not null;uniqueIndex;size:32" json:"unique"`
	Os                  Os                `json:"os"`
	Platform            Platform          `json:"platform"`
	Region              string            `json:"region"`
	Cookie              string            `json:"cookie"`
	Headers             string            `json:"headers"`
	Info                string            `json:"info"`
	Status              DeviceStatus      `gorm:"default:'unsynced'" json:"status"`
	SyncAt              *DateTime         `json:"sync_at"`
	LastUseAt           *DateTime         `json:"last_use_at"`
	Priority            int               `json:"priority"`
	Used                int               `json:"used"`
	Creator             string            `json:"creator"`
}

type TiktokDeviceGroup struct {
	Base
	Name          string         `gorm:"not null;uniqueIndex:unique_idx;size:255" json:"name"`
	Desc          string         `gorm:"not null;" json:"desc"`
	UniqueId      string         `gorm:"not null;uniqueIndex;size:32" json:"unique"`
	Creator       string         `json:"creator"`
	TiktokDevices []TiktokDevice `gorm:"foreignKey:TiktokDeviceGroupID" json:"devices,omitempty"`
}
type Role int

var Roles = []string{"admin", "manager", "member"}

const (
	Admin Role = iota + 1
	Manager
	Member
)

func (role Role) String() string {
	if role < Admin || role > Manager {
		return ""
	}
	return Roles[role-1]
}

type UserGroup struct {
	Base
	Name     string `gorm:"not null;uniqueIndex:unique_idx;size:255" json:"name"`
	Desc     string `json:"desc" gorm:"size:255"`
	UniqueId string `gorm:"uniqueIndex;size:32" json:"unique"`
	Creator  string `json:"creator"`
	Users    []User `gorm:"foreignKey:UserGroupID" json:"users,omitempty"`
}
type User struct {
	Base
	UserGroupID   uint          `json:"-"`
	Name          string        `gorm:"not null;uniqueIndex:unique_idx;size:255" json:"name"`
	Pwd           string        `gorm:"not null;size:100" json:"-"`
	UniqueId      string        `gorm:"not null;uniqueIndex;size:32" json:"unique"`
	Role          Role          `json:"role"`
	Desc          string        `json:"desc"`
	Phone         string        `json:"phone"`
	Avatar        string        `json:"avatar"`
	Creator       string        `json:"creator"`
	UserGroup     UserGroup     `gorm:"foreignKey:UserGroupID;" json:"group"`
	Status        UserStatus    `gorm:"default:'active'" json:"status"`
	SystemSetting SystemSetting `gorm:"foreignKey:UserID" json:"system_setting"`
}
type TaskGroup struct {
	Base
	Name           string       `gorm:"not null;size:255" json:"name"`
	UniqueId       string       `gorm:"not null;uniqueIndex:unique_idx;size:32" json:"unique"`
	TaskType       TaskType     `gorm:"not null;" json:"task_type"`
	TaskArgs       string       `json:"task_args"`
	Devices        string       `json:"devices"`
	DeviceGroup    string       `json:"device_group"`
	DeviceSize     int          `json:"size"`
	DeviceQueued   int          `json:"queued"`
	DeviceFailed   int          `json:"failed"`
	DeviceFinished int          `json:"finished"`
	PostInterval   int          `json:"post_interval"`
	StartAt        *DateTime    `json:"start_at"`
	FinishAt       *DateTime    `json:"finish_at"`
	Tasks          []Task       `gorm:"foreignKey:TaskGroupID" json:"tasks,omitempty"`
	Status         TaskStatus   `gorm:"default:'standby'" json:"status"`
	SortBy         DeviceSortBy `gorm:"default:1" json:"sort_by"`
	Error          string       `json:"error"`
	Creator        string       `json:"creator"`
}
type Task struct {
	Base
	TiktokDeviceID uint         `json:"-"`
	TaskGroupID    uint         `json:"-"`
	TaskID         string       `json:"task_id"`
	UniqueId       string       `gorm:"not null;uniqueIndex:unique_idx;size:32" json:"unique"`
	TiktokDevice   TiktokDevice `gorm:"foreignKey:TiktokDeviceID" json:"-"`
	TaskGroup      TaskGroup    `gorm:"foreignKey:TaskGroupID" json:"-"`
	TaskType       TaskType     `gorm:"not null;" json:"task_type"`
	TaskArgs       string       `gorm:"not null;" json:"task_args"`
	Error          string       `json:"error"`
	Status         TaskStatus   `gorm:"default:'standby'" json:"status"`
	FinishAt       *DateTime    `json:"finish_at" `
}
type SystemSetting struct {
	Base
	UserID   uint   `json:"-"`
	UniqueId string `gorm:"not null;uniqueIndex:unique_idx;size:32" json:"unique"`
	AdsKey   string `json:"ads_key" binding:"required"`
	Proxy    string `json:"proxy" binding:"required"`
}

type TaskArg struct {
	UniqueId      string     `json:"unique_id"`
	BindName      string     `json:"bind_name"`
	Headers       string     `json:"headers"`
	Cookie        string     `json:"cookie"`
	Info          string     `json:"info"`
	ProxyName     string     `json:"proxy_name"`
	ProxyValue    string     `json:"proxy_value"`
	VerifyType    VerifyType `json:"verify_type"`
	VerifyToken   string     `json:"verify_token"`
	VerifyAddress string     `json:"verify_address"`
}

func (arg *TaskArg) Query() map[string]string {
	var query map[string]string
	json.Unmarshal([]byte(arg.Info), &query)
	return query
}
func (arg *TaskArg) ReqHeaders() *request.ReqHeader {
	var headers map[string]string
	json.Unmarshal([]byte(arg.Headers), &headers)
	var reqHeader = request.Map2Headers(headers)
	reqHeader.SetCookieText(arg.Cookie)
	return reqHeader
}

type TiktokWebTaskArg struct {
	TaskArg
}

func (arg *TiktokWebTaskArg) Query() string {
	var queryMap = arg.TaskArg.Query()
	var requiredKeys = []string{
		"WebIdLastTime", "app_language", "app_name", "browser_language", "browser_name", "browser_online",
		"browser_platform", "browser_version", "channel", "cookie_enabled", "data_collection_enabled", "device_id",
		"device_platform", "focus_state", "from_page", "is_fullscreen", "is_page_visible", "odinId", "os", "priority_region",
		"referer", "region", "root_referer", "screen_height", "screen_width", "tz_name", "user_is_login", "webcast_language",
	}
	var items []string
	for _, key := range requiredKeys {
		var v = queryMap[key]
		items = append(items, fmt.Sprintf("%s=%s", key, url.QueryEscape(v)))
	}
	return strings.Join(items, "&")
}

type WebPublicVideoTaskArg struct {
	TiktokWebTaskArg
	FileName       string `json:"file_name"`
	Text           string `json:"text"`
	AllowComment   bool   `json:"allow_comment"`
	VisibilityType int    `json:"visibility_type"`
}
type WebAddCommentTaskArg struct {
	TiktokWebTaskArg
	VideoId   string `json:"video_id"`
	File      string `json:"file"`
	Level     int    `json:"level"`
	ReplyUser string `json:"reply_user"`
	ReplyId   string `json:"reply_id"`
}
type WebDiggLikeTaskArg struct {
	TiktokWebTaskArg
	VideoUrl  string `json:"video_url"`
	VideoId   string `json:"video_id"`
	ReplyUser string `json:"reply_user"`
	ReplyId   string `json:"reply_id"`
}
type WebCommentTaskArg struct {
	TiktokWebTaskArg
	Url       string `json:"url"`
	Level     int    `json:"level"`
	ReplyUser string `json:"reply_user"`
}
type WebCommentTaskArgResult struct {
	VideoId   string `json:"video_id"`
	ReplyId   string `json:"reply_id"`
	ReplyUser string `json:"reply_user"`
}
type WebAddCommentTaskResult struct {
	VideoId   string `json:"video_id"`
	ReplyText string `json:"reply_text"`
}
type WebUpdateAvatar struct {
	Avatar string `json:"avatar"`
	Proxy  string `json:"proxy"`
}

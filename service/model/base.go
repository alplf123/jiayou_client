package model

var DefaultTiktokDeviceGroup = TiktokDeviceGroup{
	Base: Base{ID: 1},
	Name: "默认分组",
	Desc: "默认分组",
}
var DefaultUserGroup = UserGroup{
	Base: Base{ID: 1},
	Name: "默认分组",
	Desc: "默认分组",
}
var DefaultSystemSetting = SystemSetting{
	Base:   Base{ID: 1},
	UserID: 1,
	Proxy:  "http://127.0.0.1:7890",
	AdsKey: "17bc218cf4d561e7355f4d0b65fa6c40",
}
var DefaultUsers = []User{
	{
		UserGroupID: 1,
		Name:        "jiayou",
		Pwd:         "123456",
		Desc:        "默认账号",
		Role:        Admin,
	},
}

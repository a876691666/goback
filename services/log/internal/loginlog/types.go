package loginlog

// ListRequest 登录日志列表请求
type ListRequest struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
	Username  string `form:"username"`
	IP        string `form:"ip"`
	Status    *int8  `form:"status"`
	StartTime string `form:"startTime"`
	EndTime   string `form:"endTime"`
}

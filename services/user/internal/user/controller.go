package user

import (
	"strconv"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/auth"
	"github.com/goback/pkg/dal"
	"github.com/goback/services/user/internal/model"
)

// Create 创建用户
func Create(e *core.RequestEvent) error {
	var req CreateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	exists, err := model.Users.ExistsByUsername(req.Username)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if exists {
		return apis.Error(e, 409, "用户名已存在")
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return apis.Error(e, 500, "密码加密失败")
	}

	user := &model.User{
		Username: req.Username,
		Password: hashedPassword,
		Nickname: req.Nickname,
		Email:    req.Email,
		Phone:    req.Phone,
		Avatar:   req.Avatar,
		RoleID:   req.RoleID,
		Status:   req.Status,
	}
	if user.Status == 0 {
		user.Status = 1
	}

	if err := model.Users.Create(user); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, user)
}

// Update 更新用户
func Update(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的用户ID")
	}
	var req UpdateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	user, err := model.Users.GetOne(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if user == nil {
		return apis.Error(e, 404, "用户不存在")
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.RoleID > 0 {
		user.RoleID = req.RoleID
	}
	if req.Status > 0 {
		user.Status = req.Status
	}

	if err := model.Users.Save(user); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, user)
}

// Delete 删除用户
func Delete(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的用户ID")
	}
	if err := model.Users.DeleteByID(id); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, nil)
}

// Get 获取用户详情
func Get(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的用户ID")
	}
	user, err := model.Users.GetByIDWithPreload(id, "Role")
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if user == nil {
		return apis.Error(e, 404, "用户不存在")
	}
	return apis.Success(e, user)
}

// List 用户列表
func List(e *core.RequestEvent) error {
	params, err := dal.BindQueryFromRequest(e.Request)
	if err != nil {
		return apis.Error(e, 400, err.Error())
	}
	if params.Expand == "" {
		params.Expand = "Role"
	}
	result, err := model.Users.GetList(params)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Paged(e, result.Items, result.TotalItems, result.Page, result.PerPage)
}

// GetProfile 获取个人信息
func GetProfile(e *core.RequestEvent) error {
	userID := apis.GetUserID(e)
	if userID == 0 {
		return apis.Error(e, 401, "未授权")
	}
	user, err := model.Users.GetByIDWithPreload(userID, "Role")
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, user)
}

// UpdateProfile 更新个人信息
func UpdateProfile(e *core.RequestEvent) error {
	userID := apis.GetUserID(e)
	if userID == 0 {
		return apis.Error(e, 401, "未授权")
	}
	var req UpdateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	user, err := model.Users.GetOne(userID)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if user == nil {
		return apis.Error(e, 404, "用户不存在")
	}

	// 普通用户不能修改角色和状态
	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}

	if err := model.Users.Save(user); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, user)
}

// ChangePassword 修改密码
func ChangePassword(e *core.RequestEvent) error {
	userID := apis.GetUserID(e)
	if userID == 0 {
		return apis.Error(e, 401, "未授权")
	}
	var req ChangePasswordRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	user, err := model.Users.GetOne(userID)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if user == nil {
		return apis.Error(e, 404, "用户不存在")
	}
	if !auth.CheckPassword(req.OldPassword, user.Password) {
		return apis.Error(e, 400, "旧密码错误")
	}

	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return apis.Error(e, 500, "密码加密失败")
	}
	if err := model.Users.UpdateByID(userID, map[string]any{"password": hashedPassword}); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, nil)
}

// ResetPassword 重置密码
func ResetPassword(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的用户ID")
	}
	hashedPassword, err := auth.HashPassword("123456")
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if err := model.Users.UpdateByID(id, map[string]any{"password": hashedPassword}); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, nil)
}

// ================== 导出函数（供其他服务调用） ==================

// GetByUsername 根据用户名获取用户
func GetByUsername(username string) (*model.User, error) {
	return model.Users.GetByUsername(username)
}

// GetByID 根据ID获取用户
func GetByID(id int64) (*model.User, error) {
	return model.Users.GetByIDWithPreload(id, "Role")
}

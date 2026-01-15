package dept

import (
	"strconv"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/dal"
	"github.com/goback/services/user/internal/model"
)

// FindAllEnabled 查找所有启用的部门
func FindAllEnabled() ([]model.Dept, error) {
	return model.Depts.GetFullList(&dal.ListParams{
		Filter: "status=1",
	})
}

// Create 创建部门
func Create(e *core.RequestEvent) error {
	var req CreateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	dept := &model.Dept{
		ParentID: req.ParentID,
		Name:     req.Name,
		Sort:     req.Sort,
		Leader:   req.Leader,
		Phone:    req.Phone,
		Email:    req.Email,
		Status:   req.Status,
	}
	if dept.Status == 0 {
		dept.Status = 1
	}
	if err := model.Depts.Create(dept); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, dept)
}

// Update 更新部门
func Update(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的部门ID")
	}
	var req UpdateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	dept, err := model.Depts.GetOne(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if dept == nil {
		return apis.Error(e, 404, "部门不存在")
	}

	if req.Name != "" {
		dept.Name = req.Name
	}
	if req.ParentID > 0 {
		dept.ParentID = req.ParentID
	}
	if req.Sort > 0 {
		dept.Sort = req.Sort
	}
	if req.Leader != "" {
		dept.Leader = req.Leader
	}
	if req.Phone != "" {
		dept.Phone = req.Phone
	}
	if req.Email != "" {
		dept.Email = req.Email
	}
	if req.Status > 0 {
		dept.Status = req.Status
	}

	if err := model.Depts.Save(dept); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, dept)
}

// Delete 删除部门
func Delete(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的部门ID")
	}
	if err := model.Depts.DeleteByID(id); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, nil)
}

// Get 获取部门详情
func Get(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的部门ID")
	}
	dept, err := model.Depts.GetOne(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if dept == nil {
		return apis.Error(e, 404, "部门不存在")
	}
	return apis.Success(e, dept)
}

// List 部门列表
func List(e *core.RequestEvent) error {
	params, err := dal.BindQueryFromRequest(e.Request)
	if err != nil {
		return apis.Error(e, 400, err.Error())
	}
	depts, err := model.Depts.GetFullList(params)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, depts)
}

// GetTree 获取部门树
func GetTree(e *core.RequestEvent) error {
	depts, err := FindAllEnabled()
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	tree := buildTree(depts, 0)
	return apis.Success(e, tree)
}

func buildTree(depts []model.Dept, parentID int64) []*model.Dept {
	var tree []*model.Dept
	for i := range depts {
		if depts[i].ParentID == parentID {
			dept := &depts[i]
			dept.Children = buildTree(depts, dept.ID)
			tree = append(tree, dept)
		}
	}
	return tree
}


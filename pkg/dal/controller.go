package dal

import (
	"context"
	"strconv"
	"strings"

	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// IDType 支持的 ID 类型约束
type IDType interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~string
}

// ParseID 解析单个 ID，支持 int64 和 string 类型
func ParseID[T IDType](s string) (T, error) {
	var zero T
	s = strings.TrimSpace(s)
	if s == "" {
		return zero, errors.New(400, "ID不能为空")
	}

	// 根据目标类型进行解析
	switch any(zero).(type) {
	case int:
		v, err := strconv.ParseInt(s, 10, 64)
		return any(int(v)).(T), err
	case int8:
		v, err := strconv.ParseInt(s, 10, 8)
		return any(int8(v)).(T), err
	case int16:
		v, err := strconv.ParseInt(s, 10, 16)
		return any(int16(v)).(T), err
	case int32:
		v, err := strconv.ParseInt(s, 10, 32)
		return any(int32(v)).(T), err
	case int64:
		v, err := strconv.ParseInt(s, 10, 64)
		return any(v).(T), err
	case uint:
		v, err := strconv.ParseUint(s, 10, 64)
		return any(uint(v)).(T), err
	case uint8:
		v, err := strconv.ParseUint(s, 10, 8)
		return any(uint8(v)).(T), err
	case uint16:
		v, err := strconv.ParseUint(s, 10, 16)
		return any(uint16(v)).(T), err
	case uint32:
		v, err := strconv.ParseUint(s, 10, 32)
		return any(uint32(v)).(T), err
	case uint64:
		v, err := strconv.ParseUint(s, 10, 64)
		return any(v).(T), err
	case string:
		return any(s).(T), nil
	default:
		return zero, errors.New(400, "不支持的ID类型")
	}
}

// ParseIDs 解析逗号分隔的 ID 列表，支持泛型 ID 类型
func ParseIDs[T IDType](s string) ([]T, error) {
	parts := strings.Split(s, ",")
	ids := make([]T, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := ParseID[T](p)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, errors.New(400, "没有有效的ID")
	}
	return ids, nil
}

// ParseInt64ID 便捷方法：解析 int64 类型的 ID
func ParseInt64ID(s string) (int64, error) {
	return ParseID[int64](s)
}

// ParseInt64IDs 便捷方法：解析 int64 类型的 ID 列表
func ParseInt64IDs(s string) ([]int64, error) {
	return ParseIDs[int64](s)
}

// ParseStringID 便捷方法：解析 string 类型的 ID
func ParseStringID(s string) (string, error) {
	return ParseID[string](s)
}

// ParseStringIDs 便捷方法：解析 string 类型的 ID 列表
func ParseStringIDs(s string) ([]string, error) {
	return ParseIDs[string](s)
}

// BaseControllerConfig 基础控制器配置
type BaseControllerConfig[T any] struct {
	Collection      *Collection[T]
	Repository      Repository[T]
	ResourceName    string            // 资源名称，用于错误消息
	EnableList      bool              // 启用分页列表
	EnableGet       bool              // 启用获取单条
	EnableCreate    bool              // 启用创建
	EnableUpdate    bool              // 启用更新
	EnableDelete    bool              // 启用删除
	EnableBatchDel  bool              // 启用批量删除
	EnableClear     bool              // 启用清空表
	EnableGetAll    bool              // 启用获取全部
	CreateValidator func(*fiber.Ctx, *T) error
	UpdateValidator func(*fiber.Ctx, int64, map[string]interface{}) error
}

// BaseController 基础控制器，提供通用的 CRUD 操作
type BaseController[T any] struct {
	collection   *Collection[T]
	repo         Repository[T]
	resourceName string
	config       BaseControllerConfig[T]
}

// NewBaseController 创建基础控制器
func NewBaseController[T any](cfg BaseControllerConfig[T]) *BaseController[T] {
	if cfg.ResourceName == "" {
		cfg.ResourceName = "resource"
	}
	return &BaseController[T]{
		collection:   cfg.Collection,
		repo:         cfg.Repository,
		resourceName: cfg.ResourceName,
		config:       cfg,
	}
}

// Collection 获取集合查询器
func (c *BaseController[T]) Collection() *Collection[T] {
	return c.collection
}

// Repository 获取仓储
func (c *BaseController[T]) Repository() Repository[T] {
	return c.repo
}

// DB 获取数据库实例
func (c *BaseController[T]) DB() *gorm.DB {
	return c.repo.DB()
}

// RegisterCRUDRoutes 注册基础 CRUD 路由
func (c *BaseController[T]) RegisterCRUDRoutes(g fiber.Router) {
	if c.config.EnableList {
		g.Get("", c.list)
	}
	if c.config.EnableGetAll {
		g.Get("/all", c.getAll)
	}
	if c.config.EnableGet {
		g.Get("/:id", c.get)
	}
	if c.config.EnableCreate {
		g.Post("", c.create)
	}
	if c.config.EnableUpdate {
		g.Put("/:id", c.update)
	}
	if c.config.EnableDelete {
		g.Delete("/:id", c.delete)
	}
	if c.config.EnableBatchDel {
		g.Delete("/batch/:ids", c.batchDelete)
	}
	if c.config.EnableClear {
		g.Delete("/clear", c.clear)
	}
}

// list 分页列表
func (c *BaseController[T]) list(ctx *fiber.Ctx) error {
	params, err := BindQuery(ctx)
	if err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	result, err := c.collection.GetList(ctx.UserContext(), params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.SuccessPage(ctx, result.Items, result.TotalItems, result.Page, result.PerPage)
}

// getAll 获取全部列表（无分页）
func (c *BaseController[T]) getAll(ctx *fiber.Ctx) error {
	params, err := BindQuery(ctx)
	if err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	items, err := c.collection.GetFullList(ctx.UserContext(), params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, items)
}

// get 获取单条记录
func (c *BaseController[T]) get(ctx *fiber.Ctx) error {
	id, err := ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的"+c.resourceName+"ID")
	}
	params, _ := BindQuery(ctx)
	entity, err := c.collection.GetOne(ctx.UserContext(), id, params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if entity == nil {
		return response.NotFound(ctx, c.resourceName+"不存在")
	}
	return response.Success(ctx, entity)
}

// create 创建记录
func (c *BaseController[T]) create(ctx *fiber.Ctx) error {
	var entity T
	if err := ctx.BodyParser(&entity); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	if c.config.CreateValidator != nil {
		if err := c.config.CreateValidator(ctx, &entity); err != nil {
			return response.ValidateError(ctx, err.Error())
		}
	}
	if err := c.repo.Create(ctx.UserContext(), &entity); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, entity)
}

// update 更新记录
func (c *BaseController[T]) update(ctx *fiber.Ctx) error {
	id, err := ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的"+c.resourceName+"ID")
	}
	var fields map[string]interface{}
	if err := ctx.BodyParser(&fields); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	if c.config.UpdateValidator != nil {
		if err := c.config.UpdateValidator(ctx, id, fields); err != nil {
			return response.ValidateError(ctx, err.Error())
		}
	}
	// 移除不应更新的字段
	delete(fields, "id")
	delete(fields, "createdAt")
	delete(fields, "created_at")
	delete(fields, "deletedAt")
	delete(fields, "deleted_at")

	if err := c.repo.UpdateFields(ctx.UserContext(), id, fields); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

// delete 删除单条记录
func (c *BaseController[T]) delete(ctx *fiber.Ctx) error {
	id, err := ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的"+c.resourceName+"ID")
	}
	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

// batchDelete 批量删除
func (c *BaseController[T]) batchDelete(ctx *fiber.Ctx) error {
	ids, err := ParseInt64IDs(ctx.Params("ids"))
	if err != nil {
		return response.BadRequest(ctx, "无效的ID格式")
	}
	if err := c.repo.DeleteBatch(ctx.UserContext(), ids); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

// clear 清空表数据
func (c *BaseController[T]) clear(ctx *fiber.Ctx) error {
	if err := c.collection.Truncate(ctx.UserContext()); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

// CreateEntity 创建实体（供子类调用）
func (c *BaseController[T]) CreateEntity(ctx context.Context, entity *T) error {
	return c.repo.Create(ctx, entity)
}

// UpdateEntity 更新实体（供子类调用）
func (c *BaseController[T]) UpdateEntity(ctx context.Context, entity *T) error {
	return c.repo.Update(ctx, entity)
}

// FindByID 根据 ID 查找（供子类调用）
func (c *BaseController[T]) FindByID(ctx context.Context, id int64) (*T, error) {
	return c.repo.FindByID(ctx, id)
}

// GetIDParam 从路由参数获取 ID（支持泛型）
func GetIDParam[T IDType](ctx *fiber.Ctx, paramName string) (T, error) {
	return ParseID[T](ctx.Params(paramName))
}

// GetIDsParam 从路由参数获取 ID 列表（支持泛型）
func GetIDsParam[T IDType](ctx *fiber.Ctx, paramName string) ([]T, error) {
	return ParseIDs[T](ctx.Params(paramName))
}

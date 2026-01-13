package model

import "github.com/goback/pkg/dal"

// User 用户
type User struct {
	dal.Model
	*dal.Collection[User] `gorm:"-" json:"-"`
	Username              string `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Password              string `gorm:"size:255;not null" json:"-"`
	Nickname              string `gorm:"size:50" json:"nickname"`
	Email                 string `gorm:"size:100" json:"email"`
	Phone                 string `gorm:"size:20" json:"phone"`
	Avatar                string `gorm:"size:255" json:"avatar"`
	Status                int8   `gorm:"default:1" json:"status"` // 1:正常 0:禁用
	RoleID                int64  `gorm:"index" json:"roleId"`
	Role                  *Role  `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

func (User) TableName() string { return "sys_user" }

// Users 用户 Collection 实例
var Users = &User{
	Collection: &dal.Collection[User]{
		DefaultSort: "-id",
		MaxPerPage:  100,
		FieldAlias: map[string]string{
			"createdAt": "created_at",
			"updatedAt": "updated_at",
			"roleId":    "role_id",
		},
	},
}

// Save 保存用户
func (c *User) Save(data *User) error {
	return c.DB().Save(data).Error
}

// ExistsByUsername 检查用户名是否存在
func (c *User) ExistsByUsername(username string, excludeID ...int64) (bool, error) {
	var count int64
	db := c.DB().Model(&User{}).Where("username = ?", username)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		db = db.Where("id != ?", excludeID[0])
	}
	err := db.Count(&count).Error
	return count > 0, err
}

// GetByUsername 根据用户名获取用户
func (c *User) GetByUsername(username string) (*User, error) {
	var user User
	err := c.DB().Where("username = ?", username).Preload("Role").First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByIDWithPreload 根据ID获取用户并预加载关联
func (c *User) GetByIDWithPreload(id int64, preloads ...string) (*User, error) {
	var user User
	db := c.DB().Where("id = ?", id)
	for _, p := range preloads {
		db = db.Preload(p)
	}
	err := db.First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Role 角色（用于关联查询）
type Role struct {
	dal.Model
	Name        string `gorm:"size:50;not null" json:"name"`
	Code        string `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Status      int8   `gorm:"default:1" json:"status"`
	Sort        int    `gorm:"default:0" json:"sort"`
	Description string `gorm:"size:255" json:"description"`
}

func (Role) TableName() string { return "sys_role" }

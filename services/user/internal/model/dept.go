package model

import "github.com/goback/pkg/dal"

// Dept 部门
type Dept struct {
	dal.Model
	*dal.Collection[Dept] `gorm:"-" json:"-"`
	ParentID              int64   `gorm:"default:0;index" json:"parentId"`
	Name                  string  `gorm:"size:50;not null" json:"name"`
	Sort                  int     `gorm:"default:0" json:"sort"`
	Leader                string  `gorm:"size:50" json:"leader"`
	Phone                 string  `gorm:"size:20" json:"phone"`
	Email                 string  `gorm:"size:100" json:"email"`
	Status                int8    `gorm:"default:1" json:"status"`
	Children              []*Dept `gorm:"-" json:"children,omitempty"`
}

func (Dept) TableName() string { return "sys_dept" }

// Depts 部门 Collection 实例
var Depts = &Dept{
	Collection: &dal.Collection[Dept]{
		DefaultSort: "sort,id",
		MaxPerPage:  500,
		FieldAlias: map[string]string{
			"createdAt": "created_at",
			"updatedAt": "updated_at",
			"parentId":  "parent_id",
		},
	},
}

// Save 保存部门
func (c *Dept) Save(data *Dept) error {
	return c.DB().Save(data).Error
}

package model

import (
	"github.com/goback/pkg/dal"
)

// Dept 部门模型
type Dept struct {
	dal.Model
	ParentID int64   `gorm:"default:0;index" json:"parentId"`
	Name     string  `gorm:"size:50;not null" json:"name"`
	Sort     int     `gorm:"default:0" json:"sort"`
	Leader   string  `gorm:"size:50" json:"leader"`
	Phone    string  `gorm:"size:20" json:"phone"`
	Email    string  `gorm:"size:100" json:"email"`
	Status   int8    `gorm:"default:1" json:"status"`
	Children []*Dept `gorm:"-" json:"children,omitempty"`
}

// TableName 表名
func (Dept) TableName() string {
	return "sys_dept"
}

package model

import (
	"github.com/goback/pkg/dal"
)

// DictType 字典类型模型
type DictType struct {
	dal.Model
	Name        string `gorm:"size:50;not null" json:"name"`
	Code        string `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Status      int8   `gorm:"default:1" json:"status"`
	Description string `gorm:"size:255" json:"description"`
}

// TableName 表名
func (DictType) TableName() string {
	return "sys_dict_type"
}

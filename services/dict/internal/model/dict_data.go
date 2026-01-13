package model

import (
	"github.com/goback/pkg/dal"
)

// DictData 字典数据模型
type DictData struct {
	dal.Model
	DictTypeID int64  `gorm:"index;not null" json:"dictTypeId"`
	Label      string `gorm:"size:100;not null" json:"label"`
	Value      string `gorm:"size:100;not null" json:"value"`
	Sort       int    `gorm:"default:0" json:"sort"`
	Status     int8   `gorm:"default:1" json:"status"`
	CSSClass   string `gorm:"size:50" json:"cssClass"`
	ListClass  string `gorm:"size:50" json:"listClass"`
	Remark     string `gorm:"size:255" json:"remark"`
}

// TableName 表名
func (DictData) TableName() string {
	return "sys_dict_data"
}

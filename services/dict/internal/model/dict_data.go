package model

import (
	"github.com/goback/pkg/dal"
)

// DictData 字典数据模型
type DictData struct {
	dal.Model
	*dal.Collection[DictData] `gorm:"-" json:"-"`
	DictTypeID                int64  `gorm:"index;not null" json:"dictTypeId"`
	Label                     string `gorm:"size:100;not null" json:"label"`
	Value                     string `gorm:"size:100;not null" json:"value"`
	Sort                      int    `gorm:"default:0" json:"sort"`
	Status                    int8   `gorm:"default:1" json:"status"`
	CSSClass                  string `gorm:"size:50" json:"cssClass"`
	ListClass                 string `gorm:"size:50" json:"listClass"`
	Remark                    string `gorm:"size:255" json:"remark"`
}

func (DictData) TableName() string {
	return "sys_dict_data"
}

var DictDatas = &DictData{
	Collection: &dal.Collection[DictData]{
		DefaultSort: "sort,-id",
		MaxPerPage:  100,
		FieldAlias: map[string]string{
			"dictTypeId": "dict_type_id",
			"createdAt":  "created_at",
			"updatedAt":  "updated_at",
			"cssClass":   "css_class",
			"listClass":  "list_class",
		},
	},
}

// Save 保存字典数据
func (c *DictData) Save(data *DictData) error {
	return c.DB().Save(data).Error
}

// GetByTypeCode 根据类型编码获取字典数据列表
func (c *DictData) GetByTypeCode(code string) ([]DictData, error) {
	var list []DictData
	err := c.DB().
		Joins("JOIN sys_dict_type ON sys_dict_type.id = sys_dict_data.dict_type_id").
		Where("sys_dict_type.code = ? AND sys_dict_data.status = 1", code).
		Order("sys_dict_data.sort ASC").
		Find(&list).Error
	return list, err
}

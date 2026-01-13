package model

import (
	"github.com/goback/pkg/dal"
)

// DictType 字典类型模型
type DictType struct {
	dal.Model
	*dal.Collection[DictType] `gorm:"-" json:"-"`
	Name                      string `gorm:"size:50;not null" json:"name"`
	Code                      string `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Status                    int8   `gorm:"default:1" json:"status"`
	Description               string `gorm:"size:255" json:"description"`
}

func (DictType) TableName() string { return "sys_dict_type" }

var DictTypes = &DictType{
	Collection: &dal.Collection[DictType]{
		DefaultSort: "-id",
		MaxPerPage:  100,
		FieldAlias: map[string]string{
			"createdAt": "created_at",
			"updatedAt": "updated_at",
		},
	},
}

// Save 保存字典类型
func (c *DictType) Save(data *DictType) error {
	return c.DB().Save(data).Error
}

// ExistsByCode 检查编码是否存在
func (c *DictType) ExistsByCode(code string, excludeID ...int64) (bool, error) {
	var count int64
	db := c.DB().Model(&DictType{}).Where("code = ?", code)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		db = db.Where("id != ?", excludeID[0])
	}
	err := db.Count(&count).Error
	return count > 0, err
}

// GetByCode 根据编码获取字典类型
func (c *DictType) GetByCode(code string) (*DictType, error) {
	var dictType DictType
	err := c.DB().Where("code = ?", code).First(&dictType).Error
	if err != nil {
		return nil, err
	}
	return &dictType, nil
}

package model

import (
	"github.com/goback/pkg/dal"
)

// SysConfig 系统参数配置模型
type SysConfig struct {
	dal.Model
	*dal.Collection[SysConfig] `gorm:"-" json:"-"`
	ConfigName                 string `gorm:"column:config_name;size:100;not null" json:"configName"`
	ConfigKey                  string `gorm:"column:config_key;size:100;not null;uniqueIndex" json:"configKey"`
	ConfigValue                string `gorm:"column:config_value;type:text;not null" json:"configValue"`
	ConfigType                 string `gorm:"column:config_type;size:1;default:N" json:"configType"`
	CreateBy                   int64  `gorm:"column:create_by;default:0" json:"createBy"`
	Remark                     string `gorm:"column:remark;size:500" json:"remark"`
}

// TableName 返回表名
func (SysConfig) TableName() string {
	return "sys_config"
}

// SysConfigs 系统参数配置集合（全局单例）
var SysConfigs = &SysConfig{
	Collection: &dal.Collection[SysConfig]{
		DefaultSort: "-id",
		MaxPerPage:  100,
		FieldAlias: map[string]string{
			"createdAt":   "created_at",
			"updatedAt":   "updated_at",
			"configName":  "config_name",
			"configKey":   "config_key",
			"configValue": "config_value",
			"configType":  "config_type",
			"createBy":    "create_by",
		},
	},
}

// Save 保存系统参数配置
func (c *SysConfig) Save(data *SysConfig) error {
	return c.DB().Save(data).Error
}

// GetByKey 根据键名获取系统参数配置
func (c *SysConfig) GetByKey(configKey string) (*SysConfig, error) {
	var config SysConfig
	err := c.DB().Where("config_key = ?", configKey).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ExistsByKey 检查键名是否存在
func (c *SysConfig) ExistsByKey(configKey string, excludeID ...int64) (bool, error) {
	var count int64
	db := c.DB().Model(&SysConfig{}).Where("config_key = ?", configKey)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		db = db.Where("id != ?", excludeID[0])
	}
	err := db.Count(&count).Error
	return count > 0, err
}

// DeleteByIds 批量删除系统参数配置
func (c *SysConfig) DeleteByIds(ids []int64) (int64, error) {
	result := c.DB().Where("id IN ?", ids).Delete(&SysConfig{})
	return result.RowsAffected, result.Error
}

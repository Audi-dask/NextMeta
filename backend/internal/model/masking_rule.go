package model

import "gorm.io/gorm"

/*
DataSourceMaskingRule 是数据源字段脱敏规则模型。
Pattern 用于匹配返回列名或字段血缘，RuleType 决定具体脱敏方式。
*/
type DataSourceMaskingRule struct {
	gorm.Model
	DataSourceID uint   `gorm:"column:data_source_id;not null;index;comment:关联数据源ID" json:"dataSourceId"`
	Pattern      string `gorm:"column:pattern;type:varchar(100);not null;comment:匹配模式(正则或通配符)" json:"pattern"`
	RuleType     string `gorm:"column:rule_type;type:varchar(50);not null;comment:脱敏类型(mask_middle, mask_all, mask_left, mask_right)" json:"ruleType"`
	Description  string `gorm:"column:description;type:varchar(255);comment:规则描述" json:"description"`
}

/*
TableName 指定数据源脱敏规则模型对应 data_source_masking_rules 表。
*/
func (DataSourceMaskingRule) TableName() string {
	return "data_source_masking_rules"
}

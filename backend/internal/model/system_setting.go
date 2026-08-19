package model

import "time"

/*
SystemSetting 是系统设置模型。
配置项按 key-value 存储，Key 作为主键，Value 保存具体配置内容。
*/
type SystemSetting struct {
	Key         string    `gorm:"primaryKey;type:varchar(100);not null" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Description string    `gorm:"type:varchar(255)" json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

/*
SeedSettings 返回系统默认配置项。
服务初始化时会写入全局 SQL 限制和通知模板等基础配置。
*/
func SeedSettings() []SystemSetting {
	return []SystemSetting{
		{
			Key:         "global_sql_limit",
			Value:       "1000",
			Description: "全局 SQL 查询行数限制 (Global SQL Query Limit)",
		},
		{
			Key:         "notification_enabled",
			Value:       "false",
			Description: "是否启用系统通知 (Enable System Notification)",
		},
		{
			Key:         "notification_webhook_url",
			Value:       "",
			Description: "系统通知 Webhook 地址 (System Notification Webhook URL)",
		},
		{
			Key:         "notification_event_ticket_created",
			Value:       "true",
			Description: "工单创建通知开关 (Ticket Created Notification Switch)",
		},
		{
			Key:         "notification_event_ticket_rejected",
			Value:       "true",
			Description: "工单驳回通知开关 (Ticket Rejected Notification Switch)",
		},
		{
			Key:         "notification_event_ticket_executed",
			Value:       "true",
			Description: "工单执行成功通知开关 (Ticket Executed Notification Switch)",
		},
		{
			Key:         "notification_event_ticket_failed",
			Value:       "true",
			Description: "工单执行失败通知开关 (Ticket Failed Notification Switch)",
		},
		{
			Key: "notification_template_ticket",
			Value: `{STATUS}｜{TYPE}｜{DATABASE}

工单：{TICKET_NO} - {TITLE}
数据源：{DATASOURCE}
操作人：{OPERATOR}
时间：{OPERATION_TIME}

处理说明：
{REMARK}

执行结果：
{EXECUTE_RESULT}`,
			Description: "工单通用通知模板 (Ticket Notification Template)",
		},
	}
}

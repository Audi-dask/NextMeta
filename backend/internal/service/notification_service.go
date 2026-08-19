package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
	"nextmeta-backend/pkg/logger"
	"strings"
	"time"

	"go.uber.org/zap"
)

// defaultTicketNotificationTemplate 是工单通知的默认模板。
// 当系统设置中没有配置 notification_template_ticket 时使用该模板兜底。
const defaultTicketNotificationTemplate = `{STATUS}｜{TYPE}｜{DATABASE}

工单：{TICKET_NO} - {TITLE}
数据源：{DATASOURCE}
操作人：{OPERATOR}
时间：{OPERATION_TIME}

处理说明：
{REMARK}

执行结果：
{EXECUTE_RESULT}`

// notificationProvider 表示 webhook 对应的通知平台。
// 当前通过 webhook URL 特征自动识别平台类型。
type notificationProvider string

const (
	notificationProviderFeishu     notificationProvider = "feishu"
	notificationProviderWechatWork notificationProvider = "wechat_work"
	notificationProviderDingTalk   notificationProvider = "dingtalk"
	notificationProviderUnknown    notificationProvider = "unknown"
)

// ticketNotificationEvent 表示工单生命周期中的通知事件。
// 每类事件都可通过系统设置单独开启或关闭。
type ticketNotificationEvent string

const (
	ticketNotificationEventCreated  ticketNotificationEvent = "ticket_created"
	ticketNotificationEventRejected ticketNotificationEvent = "ticket_rejected"
	ticketNotificationEventExecuted ticketNotificationEvent = "ticket_executed"
	ticketNotificationEventFailed   ticketNotificationEvent = "ticket_failed"
)

// notificationMessage 是平台无关的结构化通知内容。
// 飞书、企业微信、钉钉会基于该对象渲染各自的卡片消息体。
type notificationMessage struct {
	Title         string
	StatusLine    string
	TicketNo      string
	TicketTitle   string
	DataSource    string
	Operator      string
	OperationTime string
	Remark        string
	ExecuteResult string
	Content       string
}

/*
NotificationService 定义通知发送能力。
当前用于系统设置测试通知和工单生命周期通知。
*/
type NotificationService interface {
	SendNotification(content string) error
	SendDirectNotification(webhook, content string) error
	NotifyPending(ticket *model.SQLTicket)
	NotifyResult(ticket *model.SQLTicket, operator string, remark string)
	NotifyTicketCreated(ticket *model.SQLTicket)
	NotifyTicketRejected(ticket *model.SQLTicket, operator string, remark string)
	NotifyTicketExecuted(ticket *model.SQLTicket, operator string, remark string)
	NotifyTicketFailed(ticket *model.SQLTicket, operator string, remark string)
}

/*
notificationService 是 NotificationService 的默认实现。
通知 webhook、开关和模板内容都从系统设置表读取。
*/
type notificationService struct {
	settingRepo repository.SystemSettingRepository
	httpClient  *http.Client
}

/*
NewNotificationService 创建通知服务。
settingRepo 由 main.go 注入，用于读取 webhook、开关和通知模板配置。
*/
func NewNotificationService(settingRepo repository.SystemSettingRepository) NotificationService {
	return &notificationService{
		settingRepo: settingRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *notificationService) getSetting(key string) string {
	val, err := s.settingRepo.Get(key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(val)
}

func (s *notificationService) getBoolSetting(key string, defaultValue bool) bool {
	val := strings.ToLower(s.getSetting(key))
	if val == "" {
		return defaultValue
	}
	return val == "true" || val == "1" || val == "yes" || val == "on"
}

/*
getWebhook 读取系统通知 Webhook 地址。
配置缺失时返回空字符串，调用方会跳过发送。
*/
func (s *notificationService) getWebhook() string {
	return s.getSetting("notification_webhook_url")
}

// getTicketTemplate 读取工单通知模板。
// 模板为空时使用代码内默认模板，避免通知内容为空。
func (s *notificationService) getTicketTemplate() string {
	if val := s.getSetting("notification_template_ticket"); val != "" {
		return val
	}
	return defaultTicketNotificationTemplate
}

// detectNotificationProvider 根据 webhook URL 特征识别通知平台。
// 未识别的平台不会发送，避免把错误格式的消息投递到未知 webhook。
func detectNotificationProvider(webhook string) notificationProvider {
	switch {
	case strings.Contains(webhook, "open.feishu.cn/open-apis/bot/v2/hook"):
		return notificationProviderFeishu
	case strings.Contains(webhook, "qyapi.weixin.qq.com/cgi-bin/webhook/send"):
		return notificationProviderWechatWork
	case strings.Contains(webhook, "oapi.dingtalk.com/robot/send"):
		return notificationProviderDingTalk
	default:
		return notificationProviderUnknown
	}
}

/*
SendNotification 使用系统配置的 webhook 发送通知。
未启用通知或未配置 webhook 时直接返回 nil，避免影响主业务流程。
*/
func (s *notificationService) SendNotification(content string) error {
	return s.sendNotificationMessage(notificationMessage{
		Title:   "NextMeta 工单通知",
		Content: content,
	})
}

/*
SendDirectNotification 按 webhook 平台构造对应机器人消息体。
飞书使用 interactive 卡片，企业微信使用 template_card，钉钉使用 actionCard。
*/
func (s *notificationService) SendDirectNotification(webhook, content string) error {
	return s.postNotificationMessage(webhook, notificationMessage{
		Title:   "NextMeta 工单通知",
		Content: content,
	})
}

// sendNotificationMessage 使用系统配置发送结构化通知。
// 开关关闭或 webhook 为空时直接跳过，不阻断工单主流程。
func (s *notificationService) sendNotificationMessage(message notificationMessage) error {
	if !s.getBoolSetting("notification_enabled", false) {
		return nil
	}
	webhook := s.getWebhook()
	if webhook == "" {
		return nil
	}
	return s.postNotificationMessage(webhook, message)
}

// postNotificationMessage 将结构化通知转换为对应平台 payload 并提交到 webhook。
// 该方法只负责 HTTP 投递，不读取系统设置。
func (s *notificationService) postNotificationMessage(webhook string, message notificationMessage) error {
	payload, err := buildNotificationPayload(detectNotificationProvider(webhook), message)
	if err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("notification failed with status: %d", resp.StatusCode)
	}
	return nil
}

// buildNotificationPayload 按平台生成机器人消息体。
// 飞书、企业微信、钉钉字段结构不同，因此这里作为统一分发入口。
func buildNotificationPayload(provider notificationProvider, message notificationMessage) (map[string]interface{}, error) {
	switch provider {
	case notificationProviderFeishu:
		return buildFeishuNotificationPayload(message), nil
	case notificationProviderWechatWork:
		return buildWechatWorkNotificationPayload(message), nil
	case notificationProviderDingTalk:
		return buildDingTalkNotificationPayload(message), nil
	default:
		return nil, errorsUnsupportedNotificationProvider()
	}
}

// buildFeishuNotificationPayload 生成飞书 interactive 卡片消息体。
// 飞书卡片使用 markdown 元素承载用户自定义模板渲染后的内容。
func buildFeishuNotificationPayload(message notificationMessage) map[string]interface{} {
	return map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"schema": "2.0",
			"header": map[string]interface{}{
				"title": map[string]string{
					"tag":     "plain_text",
					"content": message.titleOrDefault(),
				},
				"template": feishuCardTemplate(message.statusOrFirstLine()),
			},
			"body": map[string]interface{}{
				"elements": []map[string]string{
					{
						"tag":     "markdown",
						"content": formatFeishuCardContent(message.markdownContent()),
					},
				},
			},
		},
	}
}

// buildWechatWorkNotificationPayload 生成企业微信 template_card 消息体。
// 企业微信卡片优先使用结构化字段展示关键工单信息。
func buildWechatWorkNotificationPayload(message notificationMessage) map[string]interface{} {
	card := map[string]interface{}{
		"card_type": "text_notice",
		"source": map[string]interface{}{
			"desc":       "NextMeta",
			"desc_color": wechatWorkDescColor(message.statusOrFirstLine()),
		},
		"main_title": map[string]string{
			"title": message.titleOrDefault(),
			"desc":  truncateForNotification(message.statusOrFirstLine(), 64),
		},
		"sub_title_text": truncateForNotification("处理说明："+message.remarkOrDefault(), 96),
		"horizontal_content_list": []map[string]string{
			{"keyname": "执行结果", "value": truncateForNotification(message.executeResultOrDefault(), 128)},
		},
		"card_action": map[string]interface{}{
			"type": 1,
			"url":  "https://work.weixin.qq.com",
		},
	}

	if message.TicketNo != "" {
		card["emphasis_content"] = map[string]string{
			"title": truncateForNotification(message.TicketNo, 26),
			"desc":  "工单编号",
		}
	}
	if quote := message.wechatQuoteText(); quote != "" {
		card["quote_area"] = map[string]interface{}{
			"type":       0,
			"quote_text": truncateForNotification(quote, 256),
		}
	}

	return map[string]interface{}{
		"msgtype":       "template_card",
		"template_card": card,
	}
}

// buildDingTalkNotificationPayload 生成钉钉 actionCard 消息体。
// 当前暂不接入业务详情 URL，singleURL 使用钉钉默认占位地址。
func buildDingTalkNotificationPayload(message notificationMessage) map[string]interface{} {
	return map[string]interface{}{
		"msgtype": "actionCard",
		"actionCard": map[string]interface{}{
			"title":          message.titleOrDefault(),
			"text":           formatDingTalkActionCardText(message),
			"singleTitle":    "查看详情",
			"singleURL":      "https://www.dingtalk.com",
			"btnOrientation": "0",
		},
	}
}

// formatDingTalkActionCardText 生成钉钉 actionCard 的 markdown 文本。
// 钉钉 actionCard 的正文仍是 markdown，但外层消息形态是卡片。
func formatDingTalkActionCardText(message notificationMessage) string {
	lines := []string{
		fmt.Sprintf("### %s", message.titleOrDefault()),
		"",
		fmt.Sprintf("> %s", message.statusOrFirstLine()),
	}

	if message.TicketNo != "" || message.TicketTitle != "" {
		lines = append(lines, "", fmt.Sprintf("**工单：** %s", formatTicketTitle(message)))
	}
	if message.DataSource != "" {
		lines = append(lines, fmt.Sprintf("**数据源：** %s", message.DataSource))
	}
	if message.Operator != "" {
		lines = append(lines, fmt.Sprintf("**操作人：** %s", message.Operator))
	}
	if message.OperationTime != "" {
		lines = append(lines, fmt.Sprintf("**时间：** %s", message.OperationTime))
	}

	lines = append(lines,
		"",
		"**处理说明：**",
		message.remarkOrDefault(),
		"",
		"**执行结果：**",
		message.executeResultOrDefault(),
	)
	return strings.Join(lines, "\n\n")
}

// formatTicketTitle 组合工单编号和标题。
// 非工单测试通知没有编号和标题时返回“无”。
func formatTicketTitle(message notificationMessage) string {
	switch {
	case message.TicketNo != "" && message.TicketTitle != "":
		return fmt.Sprintf("%s - %s", message.TicketNo, message.TicketTitle)
	case message.TicketNo != "":
		return message.TicketNo
	case message.TicketTitle != "":
		return message.TicketTitle
	default:
		return "无"
	}
}

// feishuCardTemplate 根据工单状态选择飞书卡片头部颜色。
// 颜色只影响展示，不参与通知事件判断。
func feishuCardTemplate(statusLine string) string {
	switch {
	case strings.Contains(statusLine, "执行失败"), strings.Contains(statusLine, "已驳回"):
		return "red"
	case strings.Contains(statusLine, "已执行"):
		return "green"
	case strings.Contains(statusLine, "部分成功"):
		return "orange"
	case strings.Contains(statusLine, "待审批"), strings.Contains(statusLine, "执行中"):
		return "blue"
	default:
		return "blue"
	}
}

// wechatWorkDescColor 根据工单状态选择企业微信卡片来源颜色。
// 取值遵循企业微信 template_card desc_color 约定。
func wechatWorkDescColor(statusLine string) int {
	switch {
	case strings.Contains(statusLine, "执行失败"), strings.Contains(statusLine, "已驳回"):
		return 1
	case strings.Contains(statusLine, "已执行"):
		return 3
	case strings.Contains(statusLine, "部分成功"):
		return 2
	default:
		return 0
	}
}

// formatFeishuCardContent 修正飞书 markdown 卡片的展示细节。
// 单独的“-”在飞书中容易和标题样式混淆，这里统一替换为“无”。
func formatFeishuCardContent(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	firstLine := strings.TrimSpace(lines[0])
	bodyLines := make([]string, 0, len(lines))
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "处理说明：", "执行结果：":
			bodyLines = append(bodyLines, "", fmt.Sprintf("**%s**", trimmed))
		case "-":
			bodyLines = append(bodyLines, "无")
		default:
			bodyLines = append(bodyLines, line)
		}
	}
	body := strings.TrimSpace(strings.Join(bodyLines, "\n"))
	if firstLine == "" {
		return body
	}
	if body == "" {
		return fmt.Sprintf("**%s**", firstLine)
	}
	return fmt.Sprintf("**%s**\n\n%s", firstLine, body)
}

func errorsUnsupportedNotificationProvider() error {
	return fmt.Errorf("unsupported notification webhook provider")
}

func userDisplayName(user model.User, fallbackID uint) string {
	if strings.TrimSpace(user.RealName) != "" {
		return user.RealName
	}
	if strings.TrimSpace(user.Username) != "" {
		return user.Username
	}
	if fallbackID > 0 {
		return fmt.Sprintf("%d", fallbackID)
	}
	return "-"
}

func ticketStatusLabel(status string) string {
	switch status {
	case "pending":
		return "待审批"
	case "executing":
		return "执行中"
	case "executed":
		return "已执行"
	case "partial_success":
		return "部分成功"
	case "failed":
		return "执行失败"
	case "rejected":
		return "已驳回"
	case "withdrawn":
		return "已撤回"
	default:
		if status == "" {
			return "-"
		}
		return status
	}
}

func formatNotificationTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

func formatNotificationTimePtr(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return formatNotificationTime(*t)
}

// buildTicketNotificationMessage 将工单模型转换为平台无关的通知对象。
// 用户自定义模板只负责生成 Content，平台卡片优先读取结构化字段。
func buildTicketNotificationMessage(template string, ticket *model.SQLTicket, operator, remark string) notificationMessage {
	operationTime := time.Now().Format("2006-01-02 15:04:05")
	if strings.TrimSpace(operator) == "" {
		operator = userDisplayName(ticket.Creator, ticket.CreatorID)
	}
	if strings.TrimSpace(remark) == "" {
		remark = "-"
	}
	executeResult := strings.TrimSpace(ticket.ExecuteResult)
	if executeResult == "" {
		executeResult = "-"
	}
	executor := strings.TrimSpace(ticket.ExecutorName)
	if executor == "" {
		executor = userDisplayName(ticket.Executor, ticket.ExecutorID)
	}

	message := notificationMessage{
		Title:         "NextMeta 工单通知",
		StatusLine:    fmt.Sprintf("%s｜%s｜%s", ticketStatusLabel(ticket.Status), ticket.TicketType, ticket.Database),
		TicketNo:      fmt.Sprintf("TCK-%06d", ticket.ID),
		TicketTitle:   ticket.Title,
		DataSource:    ticket.DataSource.Name,
		Operator:      operator,
		OperationTime: operationTime,
		Remark:        remark,
		ExecuteResult: executeResult,
	}

	r := strings.NewReplacer(
		"{TICKET_ID}", fmt.Sprintf("%d", ticket.ID),
		"{TICKET_NO}", message.TicketNo,
		"{TITLE}", ticket.Title,
		"{TYPE}", ticket.TicketType,
		"{STATUS}", ticketStatusLabel(ticket.Status),
		"{CREATOR}", userDisplayName(ticket.Creator, ticket.CreatorID),
		"{APPROVER}", userDisplayName(ticket.Approver, ticket.ApproverID),
		"{EXECUTOR}", executor,
		"{OPERATOR}", operator,
		"{DATASOURCE}", ticket.DataSource.Name,
		"{DATABASE}", ticket.Database,
		"{CREATED_AT}", formatNotificationTime(ticket.CreatedAt),
		"{UPDATED_AT}", formatNotificationTime(ticket.UpdatedAt),
		"{EXECUTED_AT}", formatNotificationTimePtr(ticket.ExecutedAt),
		"{OPERATION_TIME}", operationTime,
		"{REMARK}", remark,
		"{EXECUTE_RESULT}", executeResult,
		"{AFFECTED_ROWS}", fmt.Sprintf("%d", ticket.AffectedRows),
		"{EXECUTION_DURATION_MS}", fmt.Sprintf("%d", ticket.ExecutionDurationMS),
	)
	message.Content = r.Replace(template)
	return message
}

func (m notificationMessage) titleOrDefault() string {
	if strings.TrimSpace(m.Title) != "" {
		return m.Title
	}
	return "NextMeta 工单通知"
}

func (m notificationMessage) markdownContent() string {
	if strings.TrimSpace(m.Content) != "" {
		return m.Content
	}
	return "【NextMeta】系统通知测试"
}

func (m notificationMessage) statusOrFirstLine() string {
	if strings.TrimSpace(m.StatusLine) != "" {
		return m.StatusLine
	}
	content := m.markdownContent()
	if idx := strings.Index(content, "\n"); idx >= 0 {
		return strings.TrimSpace(content[:idx])
	}
	return strings.TrimSpace(content)
}

func (m notificationMessage) remarkOrDefault() string {
	if strings.TrimSpace(m.Remark) == "" || strings.TrimSpace(m.Remark) == "-" {
		return "无"
	}
	return strings.TrimSpace(m.Remark)
}

func (m notificationMessage) executeResultOrDefault() string {
	if strings.TrimSpace(m.ExecuteResult) == "" || strings.TrimSpace(m.ExecuteResult) == "-" {
		return "无"
	}
	return strings.TrimSpace(m.ExecuteResult)
}

// wechatQuoteText 生成企业微信卡片的引用区域文本。
// 该区域用于聚合工单、数据源、操作人和时间等辅助信息。
func (m notificationMessage) wechatQuoteText() string {
	parts := make([]string, 0, 4)
	if m.TicketTitle != "" {
		if m.TicketNo != "" {
			parts = append(parts, fmt.Sprintf("工单：%s - %s", m.TicketNo, m.TicketTitle))
		} else {
			parts = append(parts, fmt.Sprintf("工单：%s", m.TicketTitle))
		}
	}
	if m.DataSource != "" {
		parts = append(parts, fmt.Sprintf("数据源：%s", m.DataSource))
	}
	if m.Operator != "" {
		parts = append(parts, fmt.Sprintf("操作人：%s", m.Operator))
	}
	if m.OperationTime != "" {
		parts = append(parts, fmt.Sprintf("时间：%s", m.OperationTime))
	}
	return strings.Join(parts, "\n")
}

// truncateForNotification 限制机器人卡片字段长度。
// 不同平台对卡片字段长度有限制，过长内容会影响展示或投递。
func truncateForNotification(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" {
		return "无"
	}
	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// eventEnabled 判断指定工单通知事件是否启用。
// 默认开启各类工单事件，保证老配置升级后不丢通知。
func (s *notificationService) eventEnabled(event ticketNotificationEvent) bool {
	switch event {
	case ticketNotificationEventCreated:
		return s.getBoolSetting("notification_event_ticket_created", true)
	case ticketNotificationEventRejected:
		return s.getBoolSetting("notification_event_ticket_rejected", true)
	case ticketNotificationEventExecuted:
		return s.getBoolSetting("notification_event_ticket_executed", true)
	case ticketNotificationEventFailed:
		return s.getBoolSetting("notification_event_ticket_failed", true)
	default:
		return false
	}
}

// sendTicketNotification 异步发送工单事件通知。
// 通知失败只记录日志，不影响工单创建、审批或执行流程。
func (s *notificationService) sendTicketNotification(ticket *model.SQLTicket, event ticketNotificationEvent, operator, remark string) {
	if ticket == nil || !s.eventEnabled(event) {
		return
	}
	message := buildTicketNotificationMessage(s.getTicketTemplate(), ticket, operator, remark)
	go func() {
		if err := s.sendNotificationMessage(message); err != nil {
			logger.Log.Error("Failed to send ticket notification",
				zap.Uint("ticket_id", ticket.ID),
				zap.String("notification_event", string(event)),
				zap.String("operator", operator),
				zap.Error(err),
			)
		}
	}()
}

/*
NotifyPending 兼容旧调用，语义等同于工单创建待审批通知。
*/
func (s *notificationService) NotifyPending(ticket *model.SQLTicket) {
	s.NotifyTicketCreated(ticket)
}

/*
NotifyResult 兼容旧调用，并根据工单最终状态归类到驳回、成功或失败事件。
*/
func (s *notificationService) NotifyResult(ticket *model.SQLTicket, operator string, remark string) {
	if ticket == nil {
		return
	}
	switch ticket.Status {
	case "rejected":
		s.NotifyTicketRejected(ticket, operator, remark)
	case "failed", "partial_success":
		s.NotifyTicketFailed(ticket, operator, remark)
	default:
		s.NotifyTicketExecuted(ticket, operator, remark)
	}
}

func (s *notificationService) NotifyTicketCreated(ticket *model.SQLTicket) {
	s.sendTicketNotification(ticket, ticketNotificationEventCreated, userDisplayName(ticket.Creator, ticket.CreatorID), "工单已提交，等待审批")
}

func (s *notificationService) NotifyTicketRejected(ticket *model.SQLTicket, operator string, remark string) {
	s.sendTicketNotification(ticket, ticketNotificationEventRejected, operator, remark)
}

func (s *notificationService) NotifyTicketExecuted(ticket *model.SQLTicket, operator string, remark string) {
	s.sendTicketNotification(ticket, ticketNotificationEventExecuted, operator, remark)
}

func (s *notificationService) NotifyTicketFailed(ticket *model.SQLTicket, operator string, remark string) {
	s.sendTicketNotification(ticket, ticketNotificationEventFailed, operator, remark)
}

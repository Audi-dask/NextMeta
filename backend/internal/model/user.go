package model

import (
	"time"

	"gorm.io/gorm"
)

/*
User 是本地或 LDAP 用户模型。
IsApprover 是运行时展示字段，不落库，用于前端判断是否展示审批入口。
*/
type User struct {
	gorm.Model
	Username    string     `gorm:"uniqueIndex;type:varchar(50);not null;comment:用户名"`
	Password    string     `gorm:"type:varchar(255);not null;comment:加密密码" json:"-"`
	RealName    string     `gorm:"type:varchar(50);not null;comment:真实姓名"`
	Email       string     `gorm:"type:varchar(100);comment:邮箱"`
	AvatarURL   string     `gorm:"type:varchar(500);comment:头像地址" json:"avatar_url"`
	Role        string     `gorm:"type:varchar(20);default:'developer';comment:角色(super_admin/admin/developer/readonly)" json:"role"`
	Status      string     `gorm:"type:varchar(20);default:'enabled';comment:状态(enabled/disabled)" json:"status"`
	Source      string     `gorm:"type:varchar(20);default:'local';comment:来源(local/ldap/feishu)"`
	DN          string     `gorm:"type:varchar(255);comment:LDAP DN" json:"-"`
	LastLoginAt *time.Time `gorm:"comment:最后登录时间" json:"lastLoginAt"`
	IsApprover  bool       `gorm:"-" json:"is_approver"`
}

/*
TableName 指定用户模型对应 users 表。
*/
func (User) TableName() string {
	return "users"
}

/*
Group 是用户组模型。
用户组承载成员、数据源授权和审批人配置，是当前权限模型的核心实体。
*/
type Group struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex;type:varchar(100);not null;comment:组名" json:"name"`
	Code        string `gorm:"column:code;type:varchar(100);comment:组编码" json:"code"`
	Status      string `gorm:"column:status;type:varchar(20);default:'enabled';comment:状态(enabled/disabled)" json:"status"`
	Description string `gorm:"type:varchar(255);comment:描述" json:"description"`
	Source      string `gorm:"type:varchar(20);default:'local';comment:来源(local/ldap)" json:"source"`
	DN          string `gorm:"type:varchar(255);comment:LDAP DN" json:"-"`
}

/*
TableName 指定用户组模型对应 groups 表。
*/
func (Group) TableName() string {
	return "groups"
}

/*
UserGroup 是用户和用户组的成员关系模型。
用户通过所属用户组获得数据源访问权和工单审批范围。
*/
type UserGroup struct {
	gorm.Model
	UserID  uint `gorm:"not null;comment:用户ID"`
	GroupID uint `gorm:"not null;comment:组ID"`
}

/*
TableName 指定用户组成员关系模型对应 user_groups 表。
*/
func (UserGroup) TableName() string {
	return "user_groups"
}

/*
GroupDataSource 是用户组和数据源的授权关系模型。
组内成员通过该关系获得对应数据源访问权限。
*/
type GroupDataSource struct {
	gorm.Model
	GroupID      uint `gorm:"not null;comment:组ID"`
	DataSourceID uint `gorm:"not null;comment:数据源ID"`
}

/*
TableName 指定用户组数据源授权模型对应 group_datasources 表。
*/
func (GroupDataSource) TableName() string {
	return "group_datasources"
}

/*
GroupApprover 是用户组审批人关系模型。
工单提交时会根据目标数据源所属用户组选择可用审批人。
*/
type GroupApprover struct {
	gorm.Model
	GroupID uint `gorm:"not null;comment:组ID"`
	UserID  uint `gorm:"not null;comment:审批人用户ID"`
}

/*
TableName 指定用户组审批人模型对应 group_approvers 表。
*/
func (GroupApprover) TableName() string {
	return "group_approvers"
}

/*
SQLTicket 是 SQL 工单模型。
它记录提交人、审批人、目标数据源、SQL 内容、状态流转和执行结果摘要。
*/
type SQLTicket struct {
	gorm.Model
	CreatorID           uint             `gorm:"not null;comment:创建人ID"`
	Creator             User             `gorm:"foreignKey:CreatorID"`
	ApproverID          uint             `gorm:"not null;default:0;comment:指定审核人ID"`
	Approver            User             `gorm:"foreignKey:ApproverID"`
	GroupID             uint             `gorm:"not null;comment:所属组ID"`
	DataSourceID        uint             `gorm:"not null;comment:目标数据源ID"`
	DataSource          DataSource       `gorm:"foreignKey:DataSourceID"`
	Database            string           `gorm:"type:varchar(64);comment:目标数据库名"`
	Title               string           `gorm:"type:varchar(255);not null;comment:工单标题"`
	SQLContent          string           `gorm:"type:text;not null;comment:SQL内容"`
	TicketType          string           `gorm:"type:varchar(20);not null;comment:工单类型:query/change"`
	IsForce             bool             `gorm:"default:false;comment:是否强制提交"`
	Status              string           `gorm:"type:varchar(20);not null;default:'pending';comment:状态:pending/approved/rejected/executing/executed/partial_success/failed/withdrawn"`
	ExecuteResult       string           `gorm:"type:text;comment:执行结果"`
	StatementResults    string           `gorm:"type:longtext;comment:逐语句执行结果JSON"`
	ExecutorID          uint             `gorm:"comment:执行人ID"`
	Executor            User             `gorm:"foreignKey:ExecutorID"`
	ExecutorName        string           `gorm:"type:varchar(50);comment:执行人名称"`
	ExecutedAt          *time.Time       `gorm:"comment:执行时间"`
	AffectedRows        int64            `gorm:"comment:影响行数"`
	ExecutionDurationMS int64            `gorm:"comment:执行耗时毫秒"`
	Approvals           []TicketApproval `gorm:"foreignKey:TicketID"`
}

/*
TableName 指定 SQL 工单模型对应 sql_tickets 表。
*/
func (SQLTicket) TableName() string {
	return "sql_tickets"
}

/*
TicketApproval 是工单审批记录模型。
每次审批动作都会写入一条记录，用于审批历史和重复审批判断。
*/
type TicketApproval struct {
	gorm.Model
	TicketID   uint            `gorm:"not null;comment:工单ID"`
	ApproverID uint            `gorm:"not null;comment:审批人ID"`
	Approver   User            `gorm:"foreignKey:ApproverID"`
	Action     string          `gorm:"type:varchar(20);not null;comment:操作:approve/reject"`
	Comment    string          `gorm:"type:text;comment:审批意见"`
	ApprovedAt *gorm.DeletedAt `gorm:"comment:审批时间"`
}

/*
TableName 指定工单审批记录模型对应 ticket_approvals 表。
*/
func (TicketApproval) TableName() string {
	return "ticket_approvals"
}

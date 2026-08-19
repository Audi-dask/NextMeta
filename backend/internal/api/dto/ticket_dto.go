package dto

import "time"

type TicketListUserResponse struct {
	ID       uint   `json:"ID"`
	Username string `json:"Username"`
	RealName string `json:"RealName"`
}

type TicketListDataSourceResponse struct {
	ID          uint   `json:"ID"`
	Name        string `json:"name"`
	Environment string `json:"environment"`
}

type TicketListResponse struct {
	ID           uint                         `json:"ID"`
	CreatedAt    time.Time                    `json:"CreatedAt"`
	UpdatedAt    time.Time                    `json:"UpdatedAt,omitempty"`
	Title        string                       `json:"Title"`
	TicketType   string                       `json:"TicketType"`
	Status       string                       `json:"Status"`
	Database     string                       `json:"Database"`
	IsForce      bool                         `json:"IsForce"`
	Creator      TicketListUserResponse       `json:"Creator"`
	Approver     TicketListUserResponse       `json:"Approver"`
	DataSource   TicketListDataSourceResponse `json:"DataSource"`
	Executor     TicketListUserResponse       `json:"Executor"`
	ExecutorName string                       `json:"ExecutorName"`
	ExecutedAt   *time.Time                   `json:"ExecutedAt"`
}

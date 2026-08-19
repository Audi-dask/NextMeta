package service

import (
	"errors"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
)

/*
SnippetService 定义 SQL 片段业务能力。
所有操作都按用户 ID 隔离，用户只能管理自己的 SQL 片段。
*/
type SnippetService interface {
	CreateSnippet(userID uint, title, content string) error
	GetMySnippets(userID uint) ([]model.SQLSnippet, error)
	UpdateSnippet(userID, id uint, title, content string) error
	DeleteSnippet(userID, id uint) error
}

/*
snippetService 是 SnippetService 的默认实现。
实际创建、查询、更新和删除由 SnippetRepository 负责。
*/
type snippetService struct {
	repo repository.SnippetRepository
}

/*
NewSnippetService 创建 SQL 片段业务服务。
repo 由 main.go 注入，用于访问 sql_snippets 表。
*/
func NewSnippetService(repo repository.SnippetRepository) SnippetService {
	return &snippetService{repo: repo}
}

/*
CreateSnippet 创建当前用户的 SQL 片段。
创建前会限制每个用户最多保存 10 条片段，避免无限增长。
*/
func (s *snippetService) CreateSnippet(userID uint, title, content string) error {
	count, err := s.repo.CountByUserID(userID)
	if err != nil {
		return err
	}
	if count >= 10 {
		return errors.New("最多只能保存 10 条 SQL 片段")
	}

	snippet := &model.SQLSnippet{
		UserID:  userID,
		Title:   title,
		Content: content,
	}
	return s.repo.Create(snippet)
}

/*
GetMySnippets 返回当前用户的 SQL 片段列表。
repository 层会按 userID 过滤，避免跨用户读取。
*/
func (s *snippetService) GetMySnippets(userID uint) ([]model.SQLSnippet, error) {
	return s.repo.FindByUserID(userID)
}

/*
UpdateSnippet 更新当前用户的指定 SQL 片段。
userID 会随模型一起传给 repository，用于限制只能更新自己的片段。
*/
func (s *snippetService) UpdateSnippet(userID, id uint, title, content string) error {
	snippet := &model.SQLSnippet{
		ID:      id,
		UserID:  userID,
		Title:   title,
		Content: content,
	}
	return s.repo.Update(snippet)
}

/*
DeleteSnippet 删除当前用户的指定 SQL 片段。
删除条件同时包含片段 ID 和用户 ID，防止删除其他用户的数据。
*/
func (s *snippetService) DeleteSnippet(userID, id uint) error {
	return s.repo.Delete(id, userID)
}

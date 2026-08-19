package repository

import (
	"nextmeta-backend/internal/model"

	"gorm.io/gorm"
)

/*
SnippetRepository 定义 SQL 片段表的数据访问能力。
所有查询、更新和删除都按 userID 限定，保证用户只能操作自己的片段。
*/
type SnippetRepository interface {
	Create(snippet *model.SQLSnippet) error
	FindByUserID(userID uint) ([]model.SQLSnippet, error)
	Update(snippet *model.SQLSnippet) error
	Delete(id uint, userID uint) error
	CountByUserID(userID uint) (int64, error)
}

/*
snippetRepository 是 SnippetRepository 的 GORM 实现。
所有 SQL 片段数据访问都通过注入的 *gorm.DB 执行。
*/
type snippetRepository struct {
	db *gorm.DB
}

/*
NewSnippetRepository 创建 SQL 片段仓储。
db 由 main.go 初始化并注入。
*/
func NewSnippetRepository(db *gorm.DB) SnippetRepository {
	return &snippetRepository{db: db}
}

/*
Create 创建 SQL 片段记录。
调用方需要提前写入当前用户 ID、标题和 SQL 内容。
*/
func (r *snippetRepository) Create(snippet *model.SQLSnippet) error {
	return r.db.Create(snippet).Error
}

/*
FindByUserID 返回指定用户的 SQL 片段列表。
结果按创建时间倒序排列，便于前端优先展示最新片段。
*/
func (r *snippetRepository) FindByUserID(userID uint) ([]model.SQLSnippet, error) {
	var snippets []model.SQLSnippet
	err := r.db.Where("user_id = ?", userID).Order("created_at desc").Find(&snippets).Error
	return snippets, err
}

/*
Update 更新指定用户拥有的 SQL 片段。
更新条件同时包含 id 和 user_id，防止跨用户修改。
*/
func (r *snippetRepository) Update(snippet *model.SQLSnippet) error {
	return r.db.Model(snippet).Where("id = ? AND user_id = ?", snippet.ID, snippet.UserID).Updates(map[string]interface{}{
		"title":   snippet.Title,
		"content": snippet.Content,
	}).Error
}

/*
Delete 删除指定用户拥有的 SQL 片段。
删除条件同时包含 id 和 user_id，防止跨用户删除。
*/
func (r *snippetRepository) Delete(id uint, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.SQLSnippet{}).Error
}

/*
CountByUserID 统计指定用户已有的 SQL 片段数量。
service 层会用该数量限制单个用户最多保存 10 条片段。
*/
func (r *snippetRepository) CountByUserID(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.SQLSnippet{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

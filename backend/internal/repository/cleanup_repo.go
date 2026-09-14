package repository

import (
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

/*
CleanupRepository 定义历史数据物理清理的数据访问能力。
按截止时间分批硬删除流水表数据，避免一次性删除造成长事务和锁表。
*/
type CleanupRepository interface {
	PurgeBefore(table string, cutoff time.Time, batchSize int) (int64, error)
}

/*
cleanupRepository 是 CleanupRepository 的 GORM 实现。
直接使用原生 DELETE 语句绕过软删除，确保历史数据被物理移除。
*/
type cleanupRepository struct {
	db *gorm.DB
}

/*
NewCleanupRepository 创建清理仓储。
db 由 main.go 初始化并注入。
*/
func NewCleanupRepository(db *gorm.DB) CleanupRepository {
	return &cleanupRepository{db: db}
}

/*
cleanupAllowedTables 是允许清理的表白名单。
表名由 service 层硬编码传入，这里做二次校验避免误删或注入。
*/
var cleanupAllowedTables = map[string]bool{
	"ticket_approvals": true,
	"sql_tickets":      true,
	"audit_logs":       true,
}

/*
PurgeBefore 分批物理删除指定表中 created_at 早于 cutoff 的记录。
每批删除 batchSize 行，批间短暂休眠以降低锁和主从压力。
*/
func (r *cleanupRepository) PurgeBefore(table string, cutoff time.Time, batchSize int) (int64, error) {
	if !cleanupAllowedTables[table] {
		return 0, fmt.Errorf("unsupported cleanup table: %s", table)
	}

	var total int64
	for {
		result := r.db.Exec("DELETE FROM `"+table+"` WHERE created_at < ? LIMIT "+strconv.Itoa(batchSize), cutoff)
		if result.Error != nil {
			return total, result.Error
		}
		if result.RowsAffected == 0 {
			break
		}
		total += result.RowsAffected
		time.Sleep(100 * time.Millisecond)
	}
	return total, nil
}

package gormx

import (
	"context"
	"time"
)

type DataBaseSoftDelete int8

const (
	Normal DataBaseSoftDelete = iota + 1
	Deleted
)

type ModelBaseInfo struct {
	CreateAt time.Time          `gorm:"column:create_at;default:CURRENT_TIMESTAMP;NOT NULL" json:"create_at"`                             // 创建时间
	UpdateAt time.Time          `gorm:"column:update_at;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;NOT NULL" json:"update_at"` // 最后修改时间
	Deleted  DataBaseSoftDelete `gorm:"column:deleted;default:1;NOT NULL" json:"deleted"`                                                 // 是否删除 1-正常 2-删除
}

type Transaction interface {
	InTx(context.Context, func(ctx context.Context) error) error
}

type contextTxKey struct{}

package gormx

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/registry"

	"e.coding.net/healthmate/fftp_golang/fftp_infrastructure/conf"
	"e.coding.net/healthmate/fftp_golang/fftp_infrastructure/nacos"
	"e.coding.net/healthmate/fftp_golang/fftp_infrastructure/util"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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

// Data .
type Data struct {
	db  *gorm.DB
	h   *log.Helper
	rdb *redis.Client
	rs  *redsync.Redsync
	dis registry.Discovery
}

type contextTxKey struct{}

// DB 和事务相关的db操作，在取db连接时均采用此方法
func (d *Data) DB(ctx context.Context) *gorm.DB {
	tx, ok := ctx.Value(contextTxKey{}).(*gorm.DB)
	if ok {
		return tx
	}
	return d.db.WithContext(ctx)
}

func (d *Data) RDB() *redis.Client {
	return d.rdb
}

// InTx fn是包含了事务操作的方法，只要fn里面有异常，里面的db操作都会回滚
func (d *Data) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, contextTxKey{}, tx)
		return fn(ctx)
	})
}

// NewTransaction .
func NewTransaction(d *Data) Transaction {
	return d
}

func InitMysql(c *conf.Ops, systemLogger log.Logger, zapLogger *util.ZapLogger) *gorm.DB {
	h := log.NewHelper(systemLogger)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Mysql.Username,
		c.Mysql.Password,
		c.Mysql.Host,
		c.Mysql.Port,
		c.Mysql.DB,
	)

	gConfig := &gorm.Config{
		Logger: zapLogger,
	}
	db, err := gorm.Open(mysql.Open(dsn), gConfig)
	if err != nil {
		h.Errorf("init mysql error: %v", err)
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		h.Errorf("init mysql error: %v", err)
		return nil
	}
	sqlDB.SetMaxIdleConns(100)
	sqlDB.SetMaxOpenConns(150)
	sqlDB.SetConnMaxLifetime(time.Second * 25)
	if sqlDB, err := db.DB(); err != nil {
		h.Info("closing gorm mysql connection")
		defer sqlDB.Close()
	}

	return db
}

// NewData .
func NewData(c *conf.Ops, systemLogger log.Logger, zapLogger *util.ZapLogger) (*Data, func(), error) {
	h := log.NewHelper(systemLogger)

	dataBaseConnections := InitMysql(c, systemLogger, zapLogger)

	addr := fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     c.Redis.Password,
		DB:           c.Redis.DB,
		MinIdleConns: 100,
		PoolSize:     150,
	})

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		// 连接失败，返回错误
		return nil, nil, err
	}

	// todo set nacos nameSpace
	rg, err := nacos.NewRegistry(nacos.WithRegisterNamespace(util.GetEnv("", "")))
	if err != nil {
		return nil, nil, err
	}

	pool := goredis.NewPool(rdb)
	rs := redsync.New(pool)
	d := &Data{db: dataBaseConnections, rdb: rdb, rs: rs, h: h, dis: rg}

	cleanup := func() {
		h.Info("closing redis connection")
		d.rdb.Close()
		h.Info("closing the data resources")
	}
	return d, cleanup, nil
}

func (d *Data) WithRegistry(dis registry.Discovery) {
	d.dis = dis
}

func (d *Data) GetRegistry() registry.Discovery {
	return d.dis
}

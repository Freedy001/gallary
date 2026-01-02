package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gallary/server/config"
	"gallary/server/internal/model"
)

var db *gorm.DB

// InitDatabase 初始化数据库连接
func InitDatabase(cfg *config.DatabaseConfig) error {
	var err error

	// 配置 GORM logger
	gormLogger := logger.Default
	switch cfg.LogLevel {
	case "silent":
		gormLogger = logger.Default.LogMode(logger.Silent)
	case "error":
		gormLogger = logger.Default.LogMode(logger.Error)
	case "warn":
		gormLogger = logger.Default.LogMode(logger.Warn)
	case "info":
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	// 连接数据库
	db, err = gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: false,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层的 sql.db 对象
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Println("数据库连接成功")
	return nil
}

// AutoMigrate 自动迁移数据库表
func AutoMigrate() error {
	err := db.AutoMigrate(
		&model.Image{},
		&model.Tag{},
		&model.ImageTag{},
		&model.ImageMetadata{},
		&model.Share{},
		&model.ShareImage{},
		&model.Setting{},
		&model.MigrationTask{},
		&model.AIQueue{},
		&model.AITaskItem{},
		&model.ImageEmbedding{},
		&model.TagEmbedding{},
	)
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	log.Println("数据库表迁移成功")
	return nil
}

// Close 关闭数据库连接
func Close() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

type contextKey string

const txKey contextKey = "gallary_db_gorm_tx"

// Transaction0 封装事务逻辑，将 tx 注入 context
func Transaction0(ctx context.Context, fn func(ctx context.Context) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将带有事务的 db 放入 context
		txCtx := context.WithValue(ctx, txKey, tx)
		return fn(txCtx)
	})
}

func Transaction1[T any](ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	var r any
	return *r.(*T), db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将带有事务的 db 放入 context
		txCtx := context.WithValue(ctx, txKey, tx)
		t, err := fn(txCtx)
		r = &t
		return err
	})
}

// GetDB 智能获取 DB：优先取事务 DB，否则取全局 DB
func GetDB(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return db
	}

	tx, ok := ctx.Value(txKey).(*gorm.DB)
	if ok {
		return tx
	}
	return db.WithContext(ctx)
}

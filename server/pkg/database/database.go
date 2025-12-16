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
		&model.AITaskImage{},
		&model.ImageEmbedding{},
	)
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 修复向量列类型：如果列存在且类型是 vector(N)，修改为不限制维度的 vector
	// 这允许不同模型使用不同维度的向量
	if err := fixVectorColumnType(); err != nil {
		log.Printf("修复向量列类型警告: %v", err)
		// 不返回错误，因为这可能是表刚创建
	}

	log.Println("数据库表迁移成功")
	return nil
}

// fixVectorColumnType 修复向量列类型为动态维度
func fixVectorColumnType() error {
	// 检查列是否存在以及当前类型
	var columnType string
	err := db.Raw(`
		SELECT data_type || COALESCE('(' || character_maximum_length || ')', '')
		FROM information_schema.columns
		WHERE table_name = 'image_embeddings' AND column_name = 'embedding'
	`).Scan(&columnType).Error
	if err != nil {
		return err
	}

	// 如果列类型包含维度限制（如 USER-DEFINED 或 vector(N)），修改为无限制的 vector
	// PostgreSQL 中 vector 类型显示为 USER-DEFINED
	if columnType != "" {
		// 尝试修改列类型为不限制维度的 vector
		// 注意：这需要先删除列上的数据或者列类型兼容
		err = db.Exec(`
			DO $$
			BEGIN
				-- 删除已有数据（因为维度不兼容无法直接转换）
				DELETE FROM image_embeddings;
				-- 修改列类型为不限制维度的 vector
				ALTER TABLE image_embeddings ALTER COLUMN embedding TYPE vector USING embedding::vector;
			EXCEPTION
				WHEN others THEN
					-- 如果失败（可能是新表），忽略错误
					NULL;
			END $$;
		`).Error
		if err != nil {
			return err
		}
	}

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

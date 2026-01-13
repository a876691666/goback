package database

import (
	"fmt"
	"sync"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/goback/pkg/config"
	"github.com/goback/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var (
	once sync.Once
	db   *gorm.DB
)

// Init 初始化数据库连接
func Init(cfg *config.DatabaseConfig) error {
	var err error
	once.Do(func() {
		db, err = connect(cfg)
	})
	return err
}

// connect 连接数据库
func connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "mysql":
		dialector = mysql.Open(cfg.DSN())
	case "postgres":
		dialector = postgres.Open(cfg.DSN())
	case "sqlite":
		dialector = sqlite.Open(cfg.DSN())
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数表名
		},
		Logger:                                   logger.NewGormLogger(cfg.LogLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	conn, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 获取底层sql.DB并配置连接池
	sqlDB, err := conn.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return conn, nil
}

// Get 获取数据库实例
func Get() *gorm.DB {
	if db == nil {
		panic("database not initialized, call Init first")
	}
	return db
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

// Transaction 执行事务
func Transaction(fn func(tx *gorm.DB) error) error {
	return Get().Transaction(fn)
}

// AutoMigrate 自动迁移
func AutoMigrate(models ...interface{}) error {
	return Get().AutoMigrate(models...)
}

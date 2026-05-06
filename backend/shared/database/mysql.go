package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func NewMySQL(cfg Config) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("数据库连接失败:%v", err)
	}

	//获取底层的SQL连接池
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取底层SQL连接池失败: %v", err)
	}
	//设置连接池参数
	sqlDB.SetMaxOpenConns(100)  //最大打开连接数
	sqlDB.SetMaxIdleConns(10)   //最大空闲连接数
	sqlDB.SetConnMaxLifetime(0) //连接最大生命周期，0表示无限制

	log.Printf("数据库连接成功:%s,%s,%s", cfg.Host, cfg.Port, cfg.DBName)
	return db
}

// 从环境变量中加载配置

// 支持默认值
func GetEnv(key, value string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return value
}

func LoadConfigFromEnv() Config {
	return Config{
		Host:     GetEnv("MYSQL_HOST", "127.0.0.1"),
		Port:     GetEnv("MYSQL_PORT", "3306"),
		User:     GetEnv("MYSQL_USER", "root"),
		Password: GetEnv("MYSQL_PASSWORD", "root"),
		DBName:   GetEnv("MYSQL_DATABASE", "blog"),
	}
}

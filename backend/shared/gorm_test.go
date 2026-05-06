package shared

import (
	"blog-community/shared/database"
	"blog-community/shared/models"
	"testing"
)

func TestCreateTable(t *testing.T) {
	cfg := database.Config{
		Host:     "127.0.0.1",
		Port:     "3306",
		User:     "root",
		Password: "123456",
		DBName:   "blog",
	}
	db := database.NewMySQL(cfg)
	if db == nil {
		t.Fatal("Failed to connect to MySQL")
	}
	//自动迁移，创建表
	err := db.AutoMigrate(
		&models.User{},
		&models.Follow{},
		&models.Article{},
		&models.Comment{},
		&models.Like{},
		&models.Collection{},
	)
	if err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}
}

func TestUser(t *testing.T) {
	cfg := database.Config{
		Host:     "127.0.0.1",
		Port:     "3306",
		User:     "root",
		Password: "123456",
		DBName:   "blog",
	}
	db := database.NewMySQL(cfg)
	if db == nil {
		t.Fatal("Failed to connect to MySQL")
	}
	//创建用户
	user := models.User{
		Username:     "testuser",
		Email:        "1111",
		PasswordHash: "hashedpassword",
		Avatar:       "http://example.com/avatar.jpg",
	}
	err := db.Create(&user).Error
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	//查询用户
	var queriedUser models.User
	err = db.First(&queriedUser, "username = ?", "testuser").Error
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}

	//更新用户
	err = db.Model(&queriedUser).Update("bio", "This is a test user").Error
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	//删除用户
	err = db.Delete(&queriedUser).Error
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}
}

module github.com/blog-community/user-service

go 1.25.0

require gorm.io/gorm v1.31.1

require github.com/google/uuid v1.6.0 // indirect

require (
	blog-community/shared v0.0.0
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/text v0.37.0 // indirect
)

replace blog-community/shared => ../../shared

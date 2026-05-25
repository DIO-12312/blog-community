module blog-community/interaction-service

go 1.26.2

require (
	blog-community/shared v0.0.0-00010101000000-000000000000
	gorm.io/gorm v1.31.1
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/text v0.36.0 // indirect
)

replace blog-community/shared => ../../shared

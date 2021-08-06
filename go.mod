module wxbot-lostandfound

go 1.15

require (
	github.com/spf13/viper v1.8.1
	github.com/yanyiwu/gojieba v1.1.2
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.21.12
)

replace github.com/yanyiwu/gojieba v1.1.2 => github.com/ttys3/gojieba v1.1.3

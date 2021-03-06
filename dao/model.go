package dao

import "time"

// 失物
type ItemRecord struct {
	Id           int64 `gorm:"column:id;primary_key"`
	Type         int64 // 1 丢失物品 2 捡到物品
	ItemName     string
	User         string // 创建记录的用户名
	CompleteUser string // 完成记录(取走失物或者是捡到失物)的用户
	Tags         string
	City         string
	Description  string
	ImgName      string // 在本地文件夹中的图片名称
	Status       string //完成与否
	CreatedAt    time.Time
}

// 标签
type Tag struct {
	Id      int64   `gorm:"column:id;primary_key"`
	TagName string `gorm:"unique"`
}

// 标签关联
type TagItem struct {
	Id     int64 `gorm:"column:id;primary_key"`
	TagId  int64
	ItemId int64
	Type   int64 // 捡到物品的记录和丢失物品的记录的标签分开处理

}
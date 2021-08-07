package dao

import (
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"strings"
	"sync"
	"wxbot-lostandfound/conversation"
	"wxbot-lostandfound/utils"
)

var (
	db                                   *gorm.DB
	itemRecordPool, tagPool, tagItemPool sync.Pool
)

func init() {
	var err error
	db, err = gorm.Open(sqlite.Open("database.db"), &gorm.Config{})
	utils.CheckError(err, "数据库连接")
	err = db.AutoMigrate(&ItemRecord{}, &Tag{}, &TagItem{})
	utils.CheckError(err, "数据库初始化")
	itemRecordPool = sync.Pool{
		New: func() interface{} {
			return new(ItemRecord)
		},
	}
	tagPool = sync.Pool{
		New: func() interface{} {
			return new(Tag)
		},
	}
	tagItemPool = sync.Pool{
		New: func() interface{} {
			return new(TagItem)
		},
	}
}

func GetDB() *gorm.DB {
	return db
}

func AddRecord(ctx conversation.ConversationContext) (err error) {
	// 读取出标签进行储存
	itemRecord := itemRecordPool.Get().(*ItemRecord)
	defer itemRecordPool.Put(itemRecord)
	itemRecord.User = ctx.ReceiveContent.FromUsername
	itemRecord.ItemName = ctx.Conversation.Form.ItemName
	itemRecord.Type = ctx.Conversation.Type
	itemRecord.City = ctx.Conversation.Form.City
	itemRecord.Status = "未完成"
	itemRecord.Description = ctx.Conversation.Form.Description
	itemRecord.ImgName = ctx.Conversation.Form.ItemImgName
	// 记录TAG关系
	itemRecord.Tags = strings.Join(ctx.Conversation.Form.ItemTags, ",")
	db.Omit("Id").Create(itemRecord)
	// 解析所有tag
	// BUG sync.Pool和gorm共用出现异常
	for _, tagName := range ctx.Conversation.Form.ItemTags {
		tag := tagPool.Get().(*Tag)
		tag.TagName = tagName
		// 添加标签，如果存在就改为获取
		err = db.Where(tag).First(tag).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 新增标签
			db.Omit("Id").Create(tag)
		} else if err == nil {
			// 已有标签
		} else {
			log.Println("创建标签出错", err.Error())
			return
		}
		// 添加 物品——标签关联关系
		//tagItemRecord := tagItemPool.Get().(*TagItem)
		tagItemRecord := &TagItem{
			TagId:  tag.Id,
			ItemId: itemRecord.Id,
			Type:   ctx.Conversation.Type,
		}
		//tagItemRecord.ItemId = itemRecord.Id
		//tagItemRecord.TagId = tag.Id
		//tagItemRecord.Type = ctx.Conversation.Type
		db.Omit("Id").Create(tagItemRecord)
		tagItemRecord.Id = 0
		tagItemPool.Put(tagItemRecord)
		// 必须要手动清空 Pool 没有做任何“清空”的处理
		tag.Id = 0
		tagPool.Put(tag)
	}
	return
}

// 直接返回markdown列表
func GetRecord(recordType int64, status string, tags []string) (records []ItemRecord) {
	queryDB := db.Where(&ItemRecord{Type: recordType, Status: status})
	if tags != nil {
		for _, tag := range tags {
			// 不考虑性能的实现...
			log.Printf("模糊查找标签%s 类型:%d\n", tag, recordType)
			queryDB = queryDB.Where("tags like ?", "%"+tag+"%")
		}
	}
	if err := queryDB.Find(&records).Error; err != nil {
		log.Println("查询记录出错", err.Error())
	}
	return
}

// TODO 给tag加一个TYPE字段
func GetAllTag() (tags []Tag) {
	if err := db.Find(&tags).Error; err != nil {
		log.Println("查询记录出错", err.Error())
	}
	return
}

// 通过标签搜索markdown列表

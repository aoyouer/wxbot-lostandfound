package dao

import (
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"strings"
	"sync"
	"wxbot-lostandfound/bot"
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
			return new(ItemRecord)
		},
	}
	tagItemPool = sync.Pool{
		New: func() interface{} {
			return new(ItemRecord)
		},
	}
}

func GetDB() *gorm.DB {
	return db
}

func AddRecord(ctx bot.ConversationContext) (err error) {
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
	db.Create(itemRecord)
	// 解析所有tag
	for _, tagName := range ctx.Conversation.Form.ItemTags {
		tag := tagPool.Get().(*Tag)
		tag.TagName = tagName
		// 添加标签，如果存在就改为获取
		err = db.First(tag).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 新增标签
			db.Create(tag)
		} else if err == nil {
			// 已有标签
		} else {
			log.Println("创建标签出错", err.Error())
			return
		}
		// 添加 物品——标签关联关系
		tagItemRecord := tagItemPool.Get().(*TagItem)
		tagItemRecord.ItemId = itemRecord.ItemId
		tagItemRecord.TagId = tag.TagId
		tagItemRecord.Type = ctx.Conversation.Type
		db.Create(tagItemRecord)
		tagItemPool.Put(tagItemRecord)
		tagPool.Put(tag)
	}
	return
}

package dao

import (
	"errors"
	"fmt"
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
		//tag := &Tag{
		//	TagName: tagName,
		//}
		// 添加标签，如果存在就改为获取
		err = db.First(tag).Error
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

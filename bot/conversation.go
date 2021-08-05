package bot

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"time"
	"wxbot-lostandfound/utils"
)

// 针对每个用户维护一个会话map,长时间不活跃则清理

type Conversation struct {
	ConversationId uint64
	UserName       string
	LastActive     time.Time
	Stage          int    // 表单阶段
	Type           int    //捡到东西或者是丢失东西   丢失物品-1 捡到东西-2
	Operation      string // 采取的操作 如添加记录或者查看列表
	PickForm       PickForm
	LoseForm       LoseForm
}

// 捡到东西的表单

type PickForm struct {
	Who         string
	City        string   // 城市
	Location    string   // 详细地点
	ItemName    string   // 物品名词
	ItemTags    []string // 标签，用于查找
	ItemImg     string   // 物品图片链接
	Description string   // 完整描述
}

// 遗失物品的表单

type LoseForm struct {
	Who         string
	City        string   // 城市
	Location    string   // 详细地点
	ItemName    string   // 物品名词
	ItemTags    []string // 标签，用于查找
	ItemImg     string   // 物品图片链接
	Description string   // 完整描述
}

type ConversationContext struct {
	receiveContent MsgContent
	w              http.ResponseWriter
	timestamp      string
	nonce          string
}

// 回复消息的方法绑定在了对话上下文上
func (c ConversationContext) replyText(text string) (err error) {
	replyTextMsg := replyTextMsgPool.Get().(*ReplyTextMsg)
	replyTextMsg.ToUsername = c.receiveContent.FromUsername
	replyTextMsg.FromUsername = c.receiveContent.ToUsername
	replyTextMsg.CreateTime = c.receiveContent.CreateTime
	replyTextMsg.MsgType = "text"
	replyTextMsg.Content = text
	replyMsg, _ := xml.Marshal(replyTextMsg)
	replyTextMsgPool.Put(replyTextMsg)

	encryptMsg, cryptErr := wxcrypt.EncryptMsg(string(replyMsg), c.timestamp, c.nonce)
	if cryptErr != nil {
		utils.CheckError(errors.New(cryptErr.ErrMsg), "加密被动回复消息")
	}
	_, err = c.w.Write(encryptMsg)
	return
}

// 开始会话
func startConversation(receiveContent MsgContent, w http.ResponseWriter, timestamp string, nonce string) (err error) {
	ctx := ConversationContext{
		receiveContent: receiveContent,
		w:              w,
		timestamp:      timestamp,
		nonce:          nonce,
	}

	if conversation, exist := conversationMap[receiveContent.FromUsername]; exist {
		// 后续会话
		switch conversation.Stage {
		case 0:
			fmt.Println("阶段0")
			err = stage0Conversation(ctx)
		case 1:
			fmt.Println("阶段1")
			err = stage1Conversation(ctx)
		case 2:
			// 阶段二会有分叉,因为可能是添加记录或者只是查看记录
			// 添加记录的
			if conversation.Operation == "add" {
				err = stage2AddConversation(ctx)
			} else {

			}
		}
	} else {
		err = initConversationo(ctx)
	}
	return
}

func initConversationo(ctx ConversationContext) (err error) {
	// 还未记录map
	err = ctx.replyText("欢迎使用失物小助手，请问您遇到了什么问题呢?\n1.我丢失了物品\n2.我捡到了物品\n3.我是管理员!")
	//err = replyText(receiveContent, w, timestamp, nonce, "欢迎使用失物小助手，请问您遇到了什么问题呢?\n1.我丢失了物品\n2.我捡到了物品\n3.我是管理员!")
	if err == nil {
		conversationMap[ctx.receiveContent.FromUsername] = &Conversation{
			UserName:   ctx.receiveContent.FromUsername,
			LastActive: time.Now(),
			Stage:      0,
		}
	}
	return
}

// 0阶段会话 设置类型
// 当前不使用阶段嵌套,避免过多层的嵌套调用
func stage0Conversation(ctx ConversationContext) (err error) {
	conversation := conversationMap[ctx.receiveContent.FromUsername]
	conversation.LastActive = time.Now()
	switch ctx.receiveContent.Content {
	case "1", "我丢失了物品", "丢失物品":
		conversation.Stage = 1
		conversation.Type = 1
		// 虽然可以直接调用 stage1,但是为了避免过多层的嵌套,还是只进行回复
		err = ctx.replyText("1.添加丢失物品的记录\n2.查看捡到的物品列表")
	case "2", "我捡到了物品", "捡到物品":
		conversation.Stage = 1
		conversation.Type = 2
		err = ctx.replyText("1.添加捡到物品的记录\n2.查看失物记录列表")
	case "3", "我是管理员":
		// TODO 允许企业中指定身份的人进行一些管理操作
		conversation.Type = 3
	default:
		// 无效输入
		err = ctx.replyText("无效输入,请重新选择")
	}
	return
}

// 1 阶段会话 设置操作 添加或者是查看
func stage1Conversation(ctx ConversationContext) (err error) {
	conversation := conversationMap[ctx.receiveContent.FromUsername]
	conversation.LastActive = time.Now()
	switch ctx.receiveContent.Content {
	case "1", "添加丢失物品的记录", "添加捡到物品的记录":
		conversation.Operation = "add"
		if conversation.Type == 1 {
			conversation.Stage = 2
			err = ctx.replyText("请问你在哪里丢失了物品呢?")
		} else {
			conversation.Stage = 2
			err = ctx.replyText("请问你在哪里捡到了物品呢?")
		}
	case "2", "查看捡到的物品列表", "查看失物记录列表":
		conversation.Operation = "list"
		// TODO 展示已经记录的列表
	default:
		err = ctx.replyText("无效输入,请重新选择")
	}
	return
}

// 2 阶段会话 询问地点
func stage2AddConversation(ctx ConversationContext) (err error) {
	// TODO 对城市进行检查
	conversation := conversationMap[ctx.receiveContent.FromUsername]
	city := ctx.receiveContent.Content
	if !utils.IfWordInSlice(city,utils.CitySlice) {
		err = ctx.replyText("无效城市名,请重新输入")
	} else {
		if conversation.Type == 1 {
			conversation.LoseForm.City = city
		} else {
			conversation.PickForm.City = city
		}
	}
	return
}

// 被动回复文字消息
func replyText(receiveContent MsgContent, w http.ResponseWriter, timestamp string, nonce string, text string) (err error) {
	replyTextMsg := replyTextMsgPool.Get().(*ReplyTextMsg)
	replyTextMsg.ToUsername = receiveContent.FromUsername
	replyTextMsg.FromUsername = receiveContent.ToUsername
	replyTextMsg.CreateTime = receiveContent.CreateTime
	replyTextMsg.MsgType = "text"
	replyTextMsg.Content = text
	replyMsg, _ := xml.Marshal(replyTextMsg)
	replyTextMsgPool.Put(replyTextMsg)

	encryptMsg, cryptErr := wxcrypt.EncryptMsg(string(replyMsg), timestamp, nonce)
	if cryptErr != nil {
		utils.CheckError(errors.New(cryptErr.ErrMsg), "加密被动回复消息")
	}
	_, err = w.Write(encryptMsg)
	return
}

package bot

import (
	"encoding/xml"
	"errors"
	"net/http"
	"time"
	"wxbot-lostandfound/utils"
)

// 针对每个用户维护一个会话map,长时间不活跃则清理

type Conversation struct {
	ConversationId uint64
	UserName       string
	LastActive     time.Time
	Stage          int // 表单阶段
	Type           int //捡到东西或者是丢失东西   丢失物品-1 捡到东西-2
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

// 开始会话
func startConversation(receiveContent MsgContent, w http.ResponseWriter, timestamp string, nonce string) (err error) {
	if conversation, exist := conversationMap[receiveContent.FromUsername]; exist {
		// 后续会话
		switch conversation.Stage {
		case 0:
			err = stage0Conversation(receiveContent, w, timestamp, nonce)
		}
	} else {
		// 还未记录map
		err = replyText(receiveContent, w, timestamp, nonce, "欢迎使用失物小助手，请问您遇到了什么问题呢?\n1.我丢失了物品\n2.我捡到了物品\n3.我是管理员!")
		if err == nil {
			conversationMap[receiveContent.FromUsername] = &Conversation{
				UserName:   receiveContent.FromUsername,
				LastActive: time.Now(),
				Stage:      0,
			}
		}
	}
	return
}

// 0阶段会话 设置类型

func stage0Conversation(receiveContent MsgContent, w http.ResponseWriter, timestamp string, nonce string) (err error){
	conversation := conversationMap[receiveContent.FromUsername]
	conversation.LastActive = time.Now()
	switch receiveContent.Content {
	case "1", "我捡到了物品", "捡到物品":
		conversation.Stage = 1
		conversation.Type = 1
	case "2", "我丢失了物品", "丢失物品":
		conversation.Stage = 1
		conversation.Type = 2
	case "3", "我是管理员":
		// TODO 允许企业中指定身份的人进行一些管理操作
		conversation.Type = 3
	default:
		// 无效输入
		err = replyText(receiveContent, w, timestamp, nonce, "无效输入,请重新选择")
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

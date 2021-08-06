package conversation

import (
	"net/http"
	"sync"
	"time"
)

type Conversation struct {
	ConversationId int64
	UserName       string
	LastActive     time.Time
	Stage          int64    // 表单阶段
	Type           int64   //捡到东西或者是丢失东西   丢失物品-1 捡到东西-2
	Operation      string // 采取的操作 如添加记录或者查看列表
	Form           Form
	Status         string
	Edited         bool //编辑状态 在最终确认时可以选择编辑某一阶段,编辑该阶段后直接跳转到最终确认，而不是下一阶段
}

var (
	MsgContentPool, ImgMsgContentPool, ReplyTextMsgPool, InitiativeTextMsgPool sync.Pool
)

func init()  {
	MsgContentPool = sync.Pool{
		New: func() interface{} {
			return new(MsgContent)
		},
	}
	ImgMsgContentPool = sync.Pool{
		New: func() interface{} {
			return new(ImgContent)
		},
	}
	ReplyTextMsgPool = sync.Pool{
		New: func() interface{} {
			return new(ReplyTextMsg)
		},
	}
	InitiativeTextMsgPool = sync.Pool{
		New: func() interface{} {
			return new(InitiativeTextMsg)
		},
	}
}



type Form struct {
	Who         string
	City        string   // 城市
	ItemName    string   // 物品名词
	ItemTags    []string // 标签，用于查找
	ItemImg     string   // 物品图片链接
	ItemImgName string
	Description string // 完整描述
}

type ConversationContext struct {
	ReceiveContent *MsgContent
	ImgContent     *ImgContent
	Conversation   *Conversation
	W              http.ResponseWriter
	Timestamp      string
	Nonce          string
}

// 各类消息定义
//ToUserName	成员UserID
//FromUserName	企业微信CorpID
//CreateTime	消息创建时间（整型）
//MsgType	消息类型，此时固定为：text
//Content	文本消息内容,最长不超过2048个字节，超过将截断

type ReplyTextMsg struct {
	ToUsername   string `xml:"ToUserName"`
	FromUsername string `xml:"FromUserName"`
	CreateTime   uint32 `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
}

type MarkDownMsg struct {
	Touser   string `json:"touser"`
	Toparty  string `json:"toparty"`
	Totag    string `json:"totag"`
	Msgtype  string `json:"msgtype"`
	Agentid  int    `json:"agentid"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown"`
	EnableDuplicateCheck   int `json:"enable_duplicate_check"`
	DuplicateCheckInterval int `json:"duplicate_check_interval"`
}

type NewsMsg struct {
	Touser  string `json:"touser"`
	Toparty string `json:"toparty"`
	Totag   string `json:"totag"`
	Msgtype string `json:"msgtype"`
	Agentid int    `json:"agentid"`
	News    struct {
		Articles []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Url         string `json:"url"`
			Picurl      string `json:"picurl"`
		} `json:"articles"`
	} `json:"news"`
	EnableIdTrans          int `json:"enable_id_trans"`
	EnableDuplicateCheck   int `json:"enable_duplicate_check"`
	DuplicateCheckInterval int `json:"duplicate_check_interval"`
}

// 主动发送消息

type InitiativeTextMsg struct {
	Touser  string `json:"touser"`
	Toparty string `json:"toparty"`
	Totag   string `json:"totag"`
	Msgtype string `json:"msgtype"`
	Agentid int    `json:"agentid"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
	Safe                   int `json:"safe"`
	EnableIdTrans          int `json:"enable_id_trans"`
	EnableDuplicateCheck   int `json:"enable_duplicate_check"`
	DuplicateCheckInterval int `json:"duplicate_check_interval"`
}

type InitiativeMsgResponse struct {
	Errcode      int    `json:"errcode"`
	Errmsg       string `json:"errmsg"`
	Invaliduser  string `json:"invaliduser"`
	Invalidparty string `json:"invalidparty"`
	Invalidtag   string `json:"invalidtag"`
}

type MsgContent struct {
	ToUsername   string `xml:"ToUserName"`
	FromUsername string `xml:"FromUserName"`
	CreateTime   uint32 `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	Msgid        string `xml:"MsgId"`
	Agentid      uint32 `xml:"AgentId"`
}

type ImgContent struct {
	ToUsername   string `xml:"ToUserName"`
	FromUsername string `xml:"FromUserName"`
	CreateTime   uint32 `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	PicUrl       string `xml:"PicUrl"`
	MediaId      string `xml:"MediaId"`
	Msgid        string `xml:"MsgId"`
	Agentid      uint32 `xml:"AgentId"`
}

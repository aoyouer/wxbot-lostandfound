package bot

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"wxbot-lostandfound/utils"
)

var (
	initPrompt               = "欢迎使用失物小助手，请问您遇到了什么问题呢?\n1.我丢失了物品\n2.我捡到了物品\n3.我是管理员!"
	askLostItemPrompt        = "你丢失的东西是什么呢?"
	askFoundItemPrompt       = "你捡到的东西是什么呢?"
	askLostOperationPrompt   = "1.添加丢失物品的记录\n2.查看捡到的物品列表"
	askFoundOperationPrompt  = "1.添加捡到物品的记录\n2.查看失物记录列表"
	askLostPlacePrompt       = "请问你在哪里(城市)丢失了物品呢?"
	askFoundPlacePrompt      = "请问你在哪里(城市)捡到了物品呢?"
	askLostDescriptionPrompt = "请对丢失的物品进行详细一些的描述(如颜色、品牌等)。"
	askPickDescriptionPrompt = "请对捡到的物品进行详细一些的描述(如颜色、品牌等)。"
	askImgPrompt             = "请上传一张物品的图片,没有图片则输入任何文字即可。"
	generalInvalidPrompt     = "无效输入,请重新选择。"
	cityInvalidPrompt        = "无效城市名,请重新输入。"
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
	Confirm        bool
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
	receiveContent *MsgContent
	imgContent     *ImgContent
	conversation   *Conversation
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
func startConversation(receiveContent *MsgContent, imgContent *ImgContent, w http.ResponseWriter, timestamp string, nonce string) (err error) {
	ctx := ConversationContext{
		receiveContent: receiveContent,
		imgContent:     imgContent,
		w:              w,
		timestamp:      timestamp,
		nonce:          nonce,
		conversation:   conversationMap[receiveContent.FromUsername],
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
		case 3:
			fmt.Println("阶段3")
			err = stage3ItemConversation(ctx)
		case 4:
			fmt.Println("阶段4")
			err = stage4DescriptionConversation(ctx)
		case 5:
			fmt.Println("阶段5")
			err = stage5ImgConversation(ctx)
		}
	} else {
		err = initConversationo(ctx)
	}
	return
}

func initConversationo(ctx ConversationContext) (err error) {
	// 还未记录map
	err = ctx.replyText(initPrompt)
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

// 阶段0 设置类型
// 当前不使用阶段嵌套,避免过多层的嵌套调用
func stage0Conversation(ctx ConversationContext) (err error) {
	conversation := conversationMap[ctx.receiveContent.FromUsername]
	conversation.LastActive = time.Now()
	switch ctx.receiveContent.Content {
	case "1", "我丢失了物品", "丢失物品":
		conversation.Stage = 1
		conversation.Type = 1
		// 虽然可以直接调用 stage1,但是为了避免过多层的嵌套,还是只进行回复
		err = ctx.replyText(askLostOperationPrompt)
	case "2", "我捡到了物品", "捡到物品":
		conversation.Stage = 1
		conversation.Type = 2
		err = ctx.replyText(askFoundOperationPrompt)
	case "3", "我是管理员":
		// TODO 允许企业中指定身份的人进行一些管理操作
		conversation.Type = 3
	default:
		// 无效输入
		err = ctx.replyText(generalInvalidPrompt)
	}
	return
}

// 阶段1 设置操作 添加或者是查看
func stage1Conversation(ctx ConversationContext) (err error) {
	conversation := conversationMap[ctx.receiveContent.FromUsername]
	conversation.LastActive = time.Now()
	switch ctx.receiveContent.Content {
	case "1", "添加丢失物品的记录", "添加捡到物品的记录":
		conversation.Operation = "add"
		if conversation.Type == 1 {
			conversation.Stage = 2
			err = ctx.replyText(askLostPlacePrompt)
		} else {
			conversation.Stage = 2
			err = ctx.replyText(askFoundPlacePrompt)
		}
	case "2", "查看捡到的物品列表", "查看失物记录列表":
		conversation.Operation = "list"
		// TODO 展示已经记录的列表
	default:
		err = ctx.replyText(generalInvalidPrompt)
	}
	return
}

// 阶段2 添加地点
func stage2AddConversation(ctx ConversationContext) (err error) {
	// TODO 对城市进行检查
	conversation := conversationMap[ctx.receiveContent.FromUsername]
	city := ctx.receiveContent.Content
	if !utils.IfWordInSlice(city, utils.CitySlice) {
		err = ctx.replyText(cityInvalidPrompt)
	} else {
		if conversation.Type == 1 {
			conversation.LoseForm.City = city
			err = ctx.replyText(askLostItemPrompt)
		} else {
			conversation.PickForm.City = city
			err = ctx.replyText(askFoundItemPrompt)
		}
		conversation.Stage = 3
	}
	return
}

// 阶段3 添加物品名称
func stage3ItemConversation(ctx ConversationContext) (err error) {
	content := ctx.receiveContent.Content
	conversation := conversationMap[ctx.receiveContent.FromUsername]
	if conversation.Type == 1 {
		conversation.LoseForm.ItemName = content
		err = ctx.replyText(askLostDescriptionPrompt)
	} else {
		conversation.PickForm.ItemName = content
		err = ctx.replyText(askPickDescriptionPrompt)
	}
	conversation.Stage = 4
	// 根据之前的输入生成标签
	return
}

// 阶段4 添加描述 并生成标签,询问用户是否还要加上新的标签
func stage4DescriptionConversation(ctx ConversationContext) (err error) {
	desc := ctx.receiveContent.Content
	conversation := conversationMap[ctx.receiveContent.FromUsername]
	allTextBuilder := strings.Builder{}
	if conversation.Type == 1 {
		conversation.LoseForm.Description = desc
		allTextBuilder.WriteString(conversation.LoseForm.City)
		allTextBuilder.WriteString(conversation.LoseForm.ItemName)
		allTextBuilder.WriteString(conversation.LoseForm.Description)
		tags := GenerateTags(allTextBuilder.String())
		log.Println("物品ID:", tags)
		conversation.LoseForm.ItemTags = tags
		err = sendTextToUser("解析出来的标签:\n"+strings.Join(tags, ","), ctx.receiveContent.FromUsername)
		if err != nil {
			log.Println("主动消息", err.Error())
		}
	} else {
		conversation.PickForm.Description = desc
		allTextBuilder.WriteString(conversation.PickForm.City)
		allTextBuilder.WriteString(conversation.PickForm.ItemName)
		allTextBuilder.WriteString(conversation.PickForm.Description)
		tags := GenerateTags(allTextBuilder.String())
		log.Println("物品ID:", tags)
		conversation.PickForm.ItemTags = tags
	}
	conversation.Stage = 5
	err = ctx.replyText(askImgPrompt)
	return
}

// 阶段5 添加图片
func stage5ImgConversation(ctx ConversationContext) (err error) {
	if ctx.conversation.Stage != 5 {
		err = ctx.replyText("当前会话阶段无法处理图片")
	} else {
		if imgContent := ctx.imgContent; imgContent != nil {
			log.Println("读取到图片消息", ctx.imgContent)
			utils.DownloadFile(ctx.imgContent.PicUrl)
			err = ctx.replyText("图片已下载")
		} else {
			// 没有图片需要上传的情况
			err = ctx.replyText("没有图片需要上传")
		}
	}
	return
}

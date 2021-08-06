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
	Type           uint   //捡到东西或者是丢失东西   丢失物品-1 捡到东西-2
	Operation      string // 采取的操作 如添加记录或者查看列表
	Form           Form
	Status         string
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
	w              http.ResponseWriter
	timestamp      string
	nonce          string
}

// 回复消息的方法绑定在了对话上下文上
func (c ConversationContext) replyText(text string) (err error) {
	replyTextMsg := replyTextMsgPool.Get().(*ReplyTextMsg)
	replyTextMsg.ToUsername = c.ReceiveContent.FromUsername
	replyTextMsg.FromUsername = c.ReceiveContent.ToUsername
	replyTextMsg.CreateTime = c.ReceiveContent.CreateTime
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
func startConversation(ReceiveContent *MsgContent, imgContent *ImgContent, w http.ResponseWriter, timestamp string, nonce string) (err error) {
	ctx := ConversationContext{
		ReceiveContent: ReceiveContent,
		ImgContent:     imgContent,
		w:              w,
		timestamp:      timestamp,
		nonce:          nonce,
		Conversation:   conversationMap[ReceiveContent.FromUsername],
	}

	if conversation, exist := conversationMap[ReceiveContent.FromUsername]; exist {
		// 后续会话
		conversation.LastActive = time.Now()
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
		err = initConversation(ctx)
	}
	return
}

func initConversation(ctx ConversationContext) (err error) {
	// 还未记录map
	err = ctx.replyText(initPrompt)
	// 保存当前会话
	if err == nil {
		conversationMap[ctx.ReceiveContent.FromUsername] = &Conversation{
			UserName:   ctx.ReceiveContent.FromUsername,
			LastActive: time.Now(),
			Stage:      0,
		}
	}
	return
}

// 进行询问并切换阶段
func switchStage(ctx ConversationContext, targetStage int) (err error) {
	return
}

// 阶段0 设置类型
// 当前不使用阶段嵌套,避免过多层的嵌套调用
func stage0Conversation(ctx ConversationContext) (err error) {
	conversation := conversationMap[ctx.ReceiveContent.FromUsername]
	conversation.LastActive = time.Now()
	switch ctx.ReceiveContent.Content {
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
	conversation := conversationMap[ctx.ReceiveContent.FromUsername]
	conversation.LastActive = time.Now()
	switch ctx.ReceiveContent.Content {
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

// 阶段2 添加地点 需要确认
func stage2AddConversation(ctx ConversationContext) (err error) {
	// TODO 对城市进行检查
	conversation := conversationMap[ctx.ReceiveContent.FromUsername]
	content := ctx.ReceiveContent.Content
	switch ctx.Conversation.Status {
	case "":
		if !utils.IfWordInSlice(content, utils.CitySlice) {
			err = ctx.replyText(cityInvalidPrompt)
		} else {
			conversation.Status = "waitconfirm"
			conversation.Form.City = content
			// 询问,要求确认城市名称无误
			err = ctx.replyText(fmt.Sprintf("您所在的城市是:%s\n1.yes\n2.no", content))
		}
	case "waitconfirm":
		// 要求进行确认
		conversation.Status = ""
		switch content {
		case "1", "yes":
			conversation.Stage = 3
			if conversation.Type == 1 {
				err = ctx.replyText(askLostItemPrompt)
			} else {
				err = ctx.replyText(askFoundItemPrompt)
			}
		case "2", "no":
			fallthrough
		default:
			// 重新进行输入
			err = ctx.replyText("请重新输入城市名")
		}
	}

	return
}

// 阶段3 添加物品名称 需要进行确认
func stage3ItemConversation(ctx ConversationContext) (err error) {
	content := ctx.ReceiveContent.Content
	conversation := conversationMap[ctx.ReceiveContent.FromUsername]
	switch conversation.Status {
	case "":
		conversation.Form.ItemName = content
		err = ctx.replyText(fmt.Sprintf("物品名称为:%s\n1.yes\n2.no", content))
		conversation.Status = "waitconfirm"
	case "waitconfirm":
		conversation.Status = ""
		switch content {
		case "1", "yes":
			conversation.Stage = 4
			if conversation.Type == 1 {
				err = ctx.replyText(askLostDescriptionPrompt)
			} else {
				err = ctx.replyText(askPickDescriptionPrompt)
			}
		case "2", "no":
			fallthrough
		default:
			err = ctx.replyText("请重新输入物品名称")
		}
	}
	// 根据之前的输入生成标签
	return
}

// 阶段4 添加描述 需要确认 	并生成标签,~~询问用户是否还要加上新的标签~~（暂时不做...）
func stage4DescriptionConversation(ctx ConversationContext) (err error) {
	content := ctx.ReceiveContent.Content
	conversation := conversationMap[ctx.ReceiveContent.FromUsername]
	switch conversation.Status {
	case "":
		conversation.Status = "waitconfirm"
		conversation.Form.Description = content
		err = ctx.replyText(fmt.Sprintf("您的描述是:\n%s\n1.yes\n2.no", content))
	case "waitconfirm":
		conversation.Status = ""
		switch content {
		case "1", "yes":
			conversation.Stage = 5
			allTextBuilder := strings.Builder{}
			allTextBuilder.WriteString(conversation.Form.City)
			allTextBuilder.WriteString(conversation.Form.ItemName)
			allTextBuilder.WriteString(conversation.Form.Description)
			tags := GenerateTags(allTextBuilder.String())
			log.Println("物品TAGS:", tags)
			conversation.Form.ItemTags = tags
			err = ctx.replyText(askImgPrompt)
		case "2", "no":
			fallthrough
		default:
			err = ctx.replyText("请重新输入描述")
		}
	}
	return
}

// 阶段5 添加图片 需要确认
func stage5ImgConversation(ctx ConversationContext) (err error) {
	if ctx.Conversation.Stage != 5 {
		err = ctx.replyText("当前会话阶段无法处理图片")
	} else {
		switch ctx.Conversation.Status {
		case "":
			ctx.Conversation.Status = "waitconfirm"
			if imgContent := ctx.ImgContent; imgContent != nil {
				log.Println("读取到图片消息", ctx.ImgContent)
				ctx.Conversation.Form.ItemImg = imgContent.PicUrl
				err = ctx.replyText("确认是这张图片吗?\n1.yes\n2.no")
			} else {
				// 没有图片需要上传的情况
				ctx.Conversation.Form.ItemImg = ""
				err = ctx.replyText("确认没有图片需要上传吗?\n1.yes\n2.no")
			}
		case "waitconfirm":
			ctx.Conversation.Status = ""
			switch ctx.ReceiveContent.Content {
			case "1", "yes":
				// 生成最终的表格要求确认以添加到数据库  （或者将图片下载放到这里?）
				picUrl := ctx.Conversation.Form.ItemImg
				if picUrl != "" {
					ctx.Conversation.Form.ItemImgName = utils.DownloadFile(picUrl)
					log.Println("图片已下载")
				}
				// TODO stage change
				// TODO 写入数据库与推送到群消息前确认
			case "2", "no":
				fallthrough
			default:
				// 另外删除本地的图片
				err = ctx.replyText("请重新决定吧")
			}
		}
	}
	return
}

// TODO 展示当前表单已填项目
func showForm(ctx ConversationContext) (form string) {
	return
}

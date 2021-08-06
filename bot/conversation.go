package bot

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"wxbot-lostandfound/conversation"
	"wxbot-lostandfound/dao"
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
// 开始会话
func startConversation(ReceiveContent *conversation.MsgContent, imgContent *conversation.ImgContent, w http.ResponseWriter, timestamp string, nonce string) (err error) {
	ctx := conversation.ConversationContext{
		ReceiveContent: ReceiveContent,
		ImgContent:     imgContent,
		W:              w,
		Timestamp:      timestamp,
		Nonce:          nonce,
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
		case 6:
			fmt.Println("阶段6")
			err = askForConfirm(ctx)
		}
	} else {
		err = initConversation(ctx)
	}
	return
}

func initConversation(ctx conversation.ConversationContext) (err error) {
	// 还未记录map
	err = replyTextWithCtx(ctx, initPrompt)
	// 保存当前会话
	if err == nil {
		conversationMap[ctx.ReceiveContent.FromUsername] = &conversation.Conversation{
			UserName:   ctx.ReceiveContent.FromUsername,
			LastActive: time.Now(),
			Stage:      0,
		}
	}
	return
}

// 进行询问并切换阶段
func switchStage(ctx conversation.ConversationContext, targetStage int) (err error) {
	return
}

// 阶段0 设置类型
// 当前不使用阶段嵌套,避免过多层的嵌套调用
func stage0Conversation(ctx conversation.ConversationContext) (err error) {
	conversation := conversationMap[ctx.ReceiveContent.FromUsername]
	conversation.LastActive = time.Now()
	switch ctx.ReceiveContent.Content {
	case "1", "我丢失了物品", "丢失物品":
		conversation.Stage = 1
		conversation.Type = 1
		// 虽然可以直接调用 stage1,但是为了避免过多层的嵌套,还是只进行回复
		err = replyTextWithCtx(ctx, askLostOperationPrompt)
	case "2", "我捡到了物品", "捡到物品":
		conversation.Stage = 1
		conversation.Type = 2
		err = replyTextWithCtx(ctx, askFoundOperationPrompt)
	case "3", "我是管理员":
		// TODO 允许企业中指定身份的人进行一些管理操作
		conversation.Type = 3
	default:
		// 无效输入
		err = replyTextWithCtx(ctx, generalInvalidPrompt)
	}
	return
}

// 阶段1 设置操作 添加或者是查看
func stage1Conversation(ctx conversation.ConversationContext) (err error) {
	conversation := conversationMap[ctx.ReceiveContent.FromUsername]
	conversation.LastActive = time.Now()
	switch ctx.ReceiveContent.Content {
	case "1", "添加丢失物品的记录", "添加捡到物品的记录":
		conversation.Operation = "add"
		if conversation.Type == 1 {
			conversation.Stage = 2
			err = replyTextWithCtx(ctx, askLostPlacePrompt)
		} else {
			conversation.Stage = 2
			err = replyTextWithCtx(ctx, askFoundPlacePrompt)
		}
	case "2", "查看捡到的物品列表", "查看失物记录列表":
		conversation.Operation = "list"
		// TODO 展示已经记录的列表
	default:
		err = replyTextWithCtx(ctx, generalInvalidPrompt)
	}
	return
}

// 阶段2 添加地点 需要确认
func stage2AddConversation(ctx conversation.ConversationContext) (err error) {
	// TODO 对城市进行检查
	conversation := conversationMap[ctx.ReceiveContent.FromUsername]
	content := ctx.ReceiveContent.Content
	switch ctx.Conversation.Status {
	case "":
		if !utils.IfWordInSlice(content, utils.CitySlice) {
			err = replyTextWithCtx(ctx, cityInvalidPrompt)
		} else {
			conversation.Status = "waitconfirm"
			conversation.Form.City = content
			// 询问,要求确认城市名称无误
			err = replyTextWithCtx(ctx, fmt.Sprintf("您所在的城市是:%s\n1.yes\n2.no", content))
		}
	case "waitconfirm":
		// 要求进行确认
		conversation.Status = ""
		switch content {
		case "1", "yes":
			conversation.Stage = 3
			if conversation.Type == 1 {
				err = replyTextWithCtx(ctx, askLostItemPrompt)
			} else {
				err = replyTextWithCtx(ctx, askFoundItemPrompt)
			}
		case "2", "no":
			fallthrough
		default:
			// 重新进行输入
			err = replyTextWithCtx(ctx, "请重新输入城市名")
		}
	}

	return
}

// 阶段3 添加物品名称 需要进行确认
func stage3ItemConversation(ctx conversation.ConversationContext) (err error) {
	content := ctx.ReceiveContent.Content
	conversation := conversationMap[ctx.ReceiveContent.FromUsername]
	switch conversation.Status {
	case "":
		conversation.Form.ItemName = content
		err = replyTextWithCtx(ctx, fmt.Sprintf("物品名称为:%s\n1.yes\n2.no", content))
		conversation.Status = "waitconfirm"
	case "waitconfirm":
		conversation.Status = ""
		switch content {
		case "1", "yes":
			conversation.Stage = 4
			if conversation.Type == 1 {
				err = replyTextWithCtx(ctx, askLostDescriptionPrompt)
			} else {
				err = replyTextWithCtx(ctx, askPickDescriptionPrompt)
			}
		case "2", "no":
			fallthrough
		default:
			err = replyTextWithCtx(ctx, "请重新输入物品名称")
		}
	}
	// 根据之前的输入生成标签
	return
}

// 阶段4 添加描述 需要确认 	并生成标签,~~询问用户是否还要加上新的标签~~（暂时不做...）
func stage4DescriptionConversation(ctx conversation.ConversationContext) (err error) {
	content := ctx.ReceiveContent.Content
	conversation := conversationMap[ctx.ReceiveContent.FromUsername]
	switch conversation.Status {
	case "":
		conversation.Status = "waitconfirm"
		conversation.Form.Description = content
		err = replyTextWithCtx(ctx, fmt.Sprintf("您的描述是:\n%s\n1.yes\n2.no", content))
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
			err = replyTextWithCtx(ctx, askImgPrompt)
		case "2", "no":
			fallthrough
		default:
			err = replyTextWithCtx(ctx, "请重新输入描述")
		}
	}
	return
}

// 阶段5 添加图片 需要确认
func stage5ImgConversation(ctx conversation.ConversationContext) (err error) {
	if ctx.Conversation.Stage != 5 {
		err = replyTextWithCtx(ctx, "当前会话阶段无法处理图片")
	} else {
		switch ctx.Conversation.Status {
		case "":
			ctx.Conversation.Status = "waitconfirm"
			if imgContent := ctx.ImgContent; imgContent != nil {
				log.Println("读取到图片消息", ctx.ImgContent)
				ctx.Conversation.Form.ItemImg = imgContent.PicUrl
				err = replyTextWithCtx(ctx, "确认是这张图片吗?\n1.yes\n2.no")
			} else {
				// 没有图片需要上传的情况
				ctx.Conversation.Form.ItemImg = ""
				err = replyTextWithCtx(ctx, "确认没有图片需要上传吗?\n1.yes\n2.no")
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
				ctx.Conversation.Stage = 6
				err = replyTextWithCtx(ctx, "请确认下面的信息是否正确:")
				err = askForConfirm(ctx)
			case "2", "no":
				fallthrough
			default:
				// 另外删除本地的图片
				err = replyTextWithCtx(ctx, "请重新决定吧")
			}
		}
	}
	return
}

// 提交数据库前的确认 stage6

func askForConfirm(ctx conversation.ConversationContext) (err error) {
	switch ctx.Conversation.Status {
	case "":
		if !ctx.Conversation.Edited {
			// 主动发送消息，显示当前填的所有项目
			ctx.Conversation.Status = "waitconfirm"
			msg := fmt.Sprintf("在提交前进行确认:\n城市:%s\n物品:%s\n描述:%s\n标签:%s\n1.yes\n2.no", ctx.Conversation.Form.City,
				ctx.Conversation.Form.ItemName, ctx.Conversation.Form.Description, strings.Join(ctx.Conversation.Form.ItemTags, ","))
			err = sendTextToUser(msg, ctx.ReceiveContent.FromUsername)
		} else {
			// 选择切换stage处理
			switch ctx.ReceiveContent.Content {
			case "1":
				ctx.Conversation.Stage = 1
				err = replyTextWithCtx(ctx, "请重新选择要进行的操作\n1.添加记录\n2.列出记录")
			case "2":
				ctx.Conversation.Stage = 2
				err = replyTextWithCtx(ctx, "请重新输入您所在的城市")
			case "3":
				ctx.Conversation.Stage = 3
				err = replyTextWithCtx(ctx, "请重新输入物品的名称")
			case "4":
				ctx.Conversation.Stage = 4
				err = replyTextWithCtx(ctx, "请重新输入详细描述")
			case "5":
				ctx.Conversation.Stage = 5
				err = replyTextWithCtx(ctx, "请重新上传图片(输入文字则不上传图片)")
			case "6":
				ctx.Conversation.Edited = false
				ctx.Conversation.Stage = 6
				err = replyTextWithCtx(ctx, "取消选择，输入任意文本继续操作")
			case "7":
				err = replyTextWithCtx(ctx, "已删除会话")
				delete(conversationMap, ctx.ReceiveContent.FromUsername)
			}
		}
	case "waitconfirm":
		ctx.Conversation.Status = ""
		switch ctx.ReceiveContent.Content {
		case "1", "yes":
			// 提交至数据库
			err = dao.AddRecord(ctx)
			replyTextWithCtx(ctx, "已添加记录")
		case "2", "no":
			fallthrough
		default:
			// 要求选择需要修改哪一阶段
			ctx.Conversation.Edited = true
			err = replyTextWithCtx(ctx, "请输入您想要修改哪一阶段\n1.操作选择(添加记录或者是列出已有记录)\n2.城市修改\n3.物品名称修改\n4.修改描述\n5.重新上传图片\n6.取消\n7.退出会话")
		}
	}
	return
}

// TODO 展示当前表单已填项目
func showForm(ctx conversation.ConversationContext) (form string) {
	return
}

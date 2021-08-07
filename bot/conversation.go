package bot

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"wxbot-lostandfound/conversation"
	"wxbot-lostandfound/dao"
	"wxbot-lostandfound/handler"
	"wxbot-lostandfound/utils"
)

var (
	initPrompt               = "欢迎使用失物小助手，请问您遇到了什么问题呢?\n1.我丢失了物品\n2.我捡到了物品\n3.我是管理员!\n4.结束会话"
	askLostItemPrompt        = "你丢失的东西是什么呢?"
	askFoundItemPrompt       = "你捡到的东西是什么呢?"
	askLostOperationPrompt   = "1.添加丢失物品的记录\n2.查看捡到的物品列表\n3.返回上一步"
	askFoundOperationPrompt  = "1.添加捡到物品的记录\n2.查看失物记录列表\n3.返回上一步"
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
	// 收到消息后马上进行回复,避免微信服务器多次推送,之后改用异步方法向企业微信发送消息
	// TODO 有时候可能会丢包造成微信服务器没收到确认消息进而发生重传
	replyTextWithCtx(ctx, "")

	if c, exist := conversationMap[ReceiveContent.FromUsername]; exist {
		// 后续会话
		c.LastActive = time.Now()
		switch c.Stage {
		case 0:
			log.Println("阶段0")
			err = stage0Conversation(ctx)
		case 1:
			log.Println("阶段1")
			err = stage1Conversation(ctx)
		case 2:
			// 阶段二会有分叉,因为可能是添加记录或者只是查看记录
			// 添加记录的
			if c.Operation == "add" {
				err = stage2AddConversation(ctx)
			} else if c.Operation == "list" {
				stage2ListConversation(ctx)
			}
		case 3:
			log.Println("阶段3")
			err = stage3ItemConversation(ctx)
		case 4:
			log.Println("阶段4")
			err = stage4DescriptionConversation(ctx)
		case 5:
			log.Println("阶段5")
			err = stage5ImgConversation(ctx)
		case 6:
			log.Println("阶段6")
			err = askForConfirm(ctx)
		}
	} else {
		err = initConversation(ctx)
	}
	return
}

func initConversation(ctx conversation.ConversationContext) (err error) {
	// 还未记录map
	err = sendTextWithCtx(ctx, initPrompt)
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
	switch ctx.ReceiveContent.Content {
	case "1", "我丢失了物品", "丢失物品":
		ctx.Conversation.Stage = 1
		ctx.Conversation.Type = 1
		// 虽然可以直接调用 stage1,但是为了避免过多层的嵌套,还是只进行回复
		err = sendTextWithCtx(ctx, askLostOperationPrompt)
	case "2", "我捡到了物品", "捡到物品":
		ctx.Conversation.Stage = 1
		ctx.Conversation.Type = 2
		err = sendTextWithCtx(ctx, askFoundOperationPrompt)
	case "3", "我是管理员":
		// TODO 允许企业中指定身份的人进行一些管理操作
		ctx.Conversation.Type = 3
	case "4", "结束会话":
		err = sendTextWithCtx(ctx, "再见，当前会话已结束")
		delete(conversationMap, ctx.ReceiveContent.FromUsername)
	default:
		// 无效输入
		err = sendTextWithCtx(ctx, generalInvalidPrompt)
	}
	return
}

// 阶段1 设置操作 添加或者是查看
func stage1Conversation(ctx conversation.ConversationContext) (err error) {
	ctx.Conversation.Stage = 2
	switch ctx.ReceiveContent.Content {
	case "1", "添加丢失物品的记录", "添加捡到物品的记录":
		ctx.Conversation.Operation = "add"
		if ctx.Conversation.Type == 1 {
			err = sendTextWithCtx(ctx, askLostPlacePrompt)
		} else {
			err = sendTextWithCtx(ctx, askFoundPlacePrompt)
		}
	case "2", "查看捡到的物品列表", "查看失物记录列表":
		ctx.Conversation.Operation = "list"
		err = sendTextWithCtx(ctx, "1.查看所有记录\n2.查看未完成记录\n3.查看已完成记录\n4.根据描述搜索记录\n5.返回上一步")
	case "3", "返回上一步":
		ctx.Conversation.Stage = 0
		err = sendTextWithCtx(ctx, initPrompt)
	default:
		err = sendTextWithCtx(ctx, generalInvalidPrompt)
	}
	return
}

// 阶段2 添加地点 需要确认
func stage2AddConversation(ctx conversation.ConversationContext) (err error) {
	content := ctx.ReceiveContent.Content
	switch ctx.Conversation.Status {
	case "":
		if !utils.IfWordInSlice(content, utils.CitySlice) {
			err = sendTextWithCtx(ctx, cityInvalidPrompt)
		} else {
			ctx.Conversation.Status = "waitconfirm"
			ctx.Conversation.Form.City = content
			// 询问,要求确认城市名称无误
			err = sendTextWithCtx(ctx, fmt.Sprintf("您所在的城市是:%s\n1.yes\n2.no", content))
		}
	case "waitconfirm":
		// 要求进行确认
		ctx.Conversation.Status = ""
		switch content {
		case "1", "yes":
			if ctx.Conversation.Edited {
				ctx.Conversation.Stage = 6
				ctx.Conversation.Edited = false
			} else {
				ctx.Conversation.Stage = 3
			}
			if ctx.Conversation.Type == 1 {
				err = sendTextWithCtx(ctx, askLostItemPrompt)
			} else {
				err = sendTextWithCtx(ctx, askFoundItemPrompt)
			}
		case "2", "no":
			fallthrough
		default:
			// 重新进行输入
			err = sendTextWithCtx(ctx, "请重新输入城市名")
		}
	}

	return
}

// 阶段2 查看记录 以多个Markdown返回 暂时未做分页和时间等筛选
func stage2ListConversation(ctx conversation.ConversationContext) {
	var searchType int64
	if ctx.Conversation.Type == 1 {
		// 要进行相反的查找, 丢失东西应该查找的是捡到东西的记录
		searchType = 2
	} else if ctx.Conversation.Type == 2 {
		searchType = 1
	}
	switch ctx.Conversation.Status {
	case "":
		ctx.Conversation.Status = "waitchoose"
		var mds, tags []string
		switch ctx.ReceiveContent.Content {
		case "2", "查看未完成记录":
			sendTextWithCtx(ctx, "正在进行查询")
			mds = handler.GetRecordMarkdown(searchType, "未完成", nil)
		case "3", "查看已完成记录":
			sendTextWithCtx(ctx, "正在进行查询")
			mds = handler.GetRecordMarkdown(searchType, "已完成", nil)
		case "4", "根据标签搜索记录":
			sendTextWithCtx(ctx, "正在进行查询")
			tags = handler.GetAllTag()
			if len(tags) == 0 {
				sendTextWithCtx(ctx, "当前还没有任何标签")
			}
		case "1", "查看所有记录":
			mds = handler.GetRecordMarkdown(searchType, "", nil)
		case "5", "返回上一步":
			fallthrough
		default:
			ctx.Conversation.Stage = 1
			if ctx.Conversation.Type == 1 {
				sendTextWithCtx(ctx, askLostOperationPrompt)
			} else {
				sendTextWithCtx(ctx, askFoundOperationPrompt)
			}
		}
		// 返回多个markdown
		if len(mds) > 0 {
			sendTextWithCtx(ctx, "共找到"+strconv.Itoa(len(mds))+"条记录")
			for _, md := range mds {
				if err := sendMDtoUserWithCtx(ctx, md); err != nil {
					log.Println("返回markdown出错", err.Error())
				}
			}
			sendTextWithCtx(ctx, "1.返回上一步\n2.结束会话")
		}
		if len(tags) > 0 {
			ctx.Conversation.Status = "waittags"
			sendTextWithCtx(ctx, fmt.Sprintf("当前共有如下标签\n%s\n输入标签进行查询(多个标签用空格分隔)", strings.Join(tags, ",")))
		}

	case "waitchoose":
		ctx.Conversation.Status = ""
		switch ctx.ReceiveContent.Content {
		case "1", "返回上一步":
			ctx.Conversation.Stage = 2
			sendTextWithCtx(ctx, "1.查看所有记录\n2.查看未完成记录\n3.查看已完成记录\n4.根据描述搜索记录\n5.返回上一步")
		case "2", "结束会话":
			sendTextWithCtx(ctx, "当前会话已结束")
			delete(conversationMap, ctx.ReceiveContent.FromUsername)
		}
	case "waittags":
		// 对输入的文本进行提取，提取出标签
		ctx.Conversation.Status = "waitchoose"
		content := ctx.ReceiveContent.Content
		tags := strings.Fields(content)
		mds := handler.GetRecordMarkdown(searchType, "", tags)
		sendTextWithCtx(ctx, "共找到"+strconv.Itoa(len(mds))+"条记录")
		if len(mds) > 0 {
			for _, md := range mds {
				if err := sendMDtoUserWithCtx(ctx, md); err != nil {
					log.Println("返回markdown出错", err.Error())
				}
			}
		}
		sendTextWithCtx(ctx, "1.返回上一步\n2.结束会话")
	}
}

// 阶段3 添加物品名称 需要进行确认
func stage3ItemConversation(ctx conversation.ConversationContext) (err error) {
	content := ctx.ReceiveContent.Content
	switch ctx.Conversation.Status {
	case "":
		ctx.Conversation.Form.ItemName = content
		err = sendTextWithCtx(ctx, fmt.Sprintf("物品名称为:%s\n1.yes\n2.no", content))
		ctx.Conversation.Status = "waitconfirm"
	case "waitconfirm":
		ctx.Conversation.Status = ""
		switch content {
		case "1", "yes":
			if ctx.Conversation.Edited {
				ctx.Conversation.Stage = 6
				ctx.Conversation.Edited = false
			} else {
				ctx.Conversation.Stage = 4
			}
			if ctx.Conversation.Type == 1 {
				err = sendTextWithCtx(ctx, askLostDescriptionPrompt)
			} else {
				err = sendTextWithCtx(ctx, askPickDescriptionPrompt)
			}
		case "2", "no":
			fallthrough
		default:
			err = sendTextWithCtx(ctx, "请重新输入物品名称")
		}
	}
	// 根据之前的输入生成标签
	return
}

// 阶段4 添加描述 需要确认 	并生成标签,~~询问用户是否还要加上新的标签~~（暂时不做...）
func stage4DescriptionConversation(ctx conversation.ConversationContext) (err error) {
	content := ctx.ReceiveContent.Content
	switch ctx.Conversation.Status {
	case "":
		ctx.Conversation.Status = "waitconfirm"
		ctx.Conversation.Form.Description = content
		err = sendTextWithCtx(ctx, fmt.Sprintf("您的描述是:\n%s\n1.yes\n2.no", content))
	case "waitconfirm":
		ctx.Conversation.Status = ""
		switch content {
		case "1", "yes":
			if ctx.Conversation.Edited {
				ctx.Conversation.Stage = 6
				ctx.Conversation.Edited = false
			} else {
				ctx.Conversation.Stage = 5
			}
			allTextBuilder := strings.Builder{}
			allTextBuilder.WriteString(ctx.Conversation.Form.City)
			allTextBuilder.WriteString(ctx.Conversation.Form.ItemName)
			allTextBuilder.WriteString(ctx.Conversation.Form.Description)
			tags := GenerateTags(allTextBuilder.String())
			log.Println("物品TAGS:", tags)
			ctx.Conversation.Form.ItemTags = tags
			err = sendTextWithCtx(ctx, askImgPrompt)
		case "2", "no":
			fallthrough
		default:
			err = sendTextWithCtx(ctx, "请重新输入描述")
		}
	}
	return
}

// 阶段5 添加图片 需要确认
func stage5ImgConversation(ctx conversation.ConversationContext) (err error) {
	if ctx.Conversation.Stage != 5 {
		err = sendTextWithCtx(ctx, "当前会话阶段无法处理图片")
	} else {
		switch ctx.Conversation.Status {
		case "":
			ctx.Conversation.Status = "waitconfirm"
			if imgContent := ctx.ImgContent; imgContent != nil {
				log.Println("读取到图片消息", ctx.ImgContent)
				ctx.Conversation.Form.ItemImg = imgContent.PicUrl
				ctx.ImgContent = nil
				err = sendTextWithCtx(ctx, "确认是这张图片吗?\n1.yes\n2.no")
			} else {
				// 没有图片需要上传的情况
				ctx.Conversation.Form.ItemImg = ""
				err = sendTextWithCtx(ctx, "确认没有图片需要上传吗?\n1.yes\n2.no")
			}
		case "waitconfirm":
			ctx.Conversation.Status = ""
			switch ctx.ReceiveContent.Content {
			case "1", "yes":
				// 生成最终的表格要求确认以添加到数据库  （或者将图片下载放到这里?）
				picUrl := ctx.Conversation.Form.ItemImg
				if picUrl != "" {
					// 下载完成后的消息采取主动推送，否则的超过5秒连接会被断开，无法回复
					log.Println("开始下载图片")
					ctx.Conversation.Form.ItemImgName = utils.DownloadFile(picUrl)
					log.Println("图片已下载")
					err = sendTextWithCtx(ctx, "图片已下载")
				} else {
					err = sendTextWithCtx(ctx, "选择不上传图片")
				}
				ctx.Conversation.Stage = 6
				err = askForConfirm(ctx)
			case "2", "no":
				fallthrough
			default:
				// 另外删除本地的图片
				err = sendTextWithCtx(ctx, "请重新决定吧")
			}
		}
	}
	return
}

// 提交数据库前的确认 stage6
func askForConfirm(ctx conversation.ConversationContext) (err error) {
	switch ctx.Conversation.Status {
	case "":
		log.Println("向用户展示确认信息")
		if !ctx.Conversation.Edited {
			// 主动发送消息，显示当前填的所有项目
			ctx.Conversation.Status = "waitconfirm"
			msg := fmt.Sprintf("在提交前进行确认:\n城市:%s\n物品:%s\n描述:%s\n标签:%s\n1.yes\n2.no", ctx.Conversation.Form.City,
				ctx.Conversation.Form.ItemName, ctx.Conversation.Form.Description, strings.Join(ctx.Conversation.Form.ItemTags, ","))
			//err = sendTextToUser(msg, ctx.ReceiveContent.FromUsername)
			err = sendTextWithCtx(ctx, msg)
		} else {
			// 选择切换stage处理
			switch ctx.ReceiveContent.Content {
			case "1":
				ctx.Conversation.Stage = 1
				err = sendTextWithCtx(ctx, "请重新选择要进行的操作\n1.添加记录\n2.列出记录")
			case "2":
				ctx.Conversation.Stage = 2
				err = sendTextWithCtx(ctx, "请重新输入您所在的城市")
			case "3":
				ctx.Conversation.Stage = 3
				err = sendTextWithCtx(ctx, "请重新输入物品的名称")
			case "4":
				ctx.Conversation.Stage = 4
				err = sendTextWithCtx(ctx, "请重新输入详细描述")
			case "5":
				ctx.Conversation.Stage = 5
				err = sendTextWithCtx(ctx, "请重新上传图片(输入文字则不上传图片)")
			case "6":
				ctx.Conversation.Edited = false
				ctx.Conversation.Stage = 6
				err = sendTextWithCtx(ctx, "取消选择，输入任意文本继续操作")
			case "7":
				err = sendTextWithCtx(ctx, "已取消该次会话")
				delete(conversationMap, ctx.ReceiveContent.FromUsername)
			}
		}
	case "waitconfirm":
		log.Println("等待用户输入确认信息")

		ctx.Conversation.Status = ""
		switch ctx.ReceiveContent.Content {
		case "1", "yes":
			// 提交至数据库
			err = dao.AddRecord(ctx)
			err = sendTextWithCtx(ctx, "已添加记录,当前会话已结束") //TODO 可扩展提交记录后进行查找
			delete(conversationMap, ctx.ReceiveContent.FromUsername)
		case "2", "no":
			fallthrough
		default:
			// 要求选择需要修改哪一阶段
			ctx.Conversation.Edited = true
			err = sendTextWithCtx(ctx, "请输入您想要修改哪一阶段\n1.操作选择(添加记录或者是列出已有记录)\n2.城市修改\n3.物品名称修改\n4.修改描述\n5.重新上传图片\n6.取消\n7.退出会话")
		}
	}
	return
}

// TODO 展示当前表单已填项目
func showForm(ctx conversation.ConversationContext) (form string) {
	return
}

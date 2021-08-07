package bot

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"wxbot-lostandfound/conversation"
	"wxbot-lostandfound/utils"
)

// 被动回复文字消息
func replyText(receiveContent conversation.MsgContent, w http.ResponseWriter, timestamp string, nonce string, text string) (err error) {
	replyTextMsg := conversation.ReplyTextMsgPool.Get().(*conversation.ReplyTextMsg)
	replyTextMsg.ToUsername = receiveContent.FromUsername
	replyTextMsg.FromUsername = receiveContent.ToUsername
	replyTextMsg.CreateTime = receiveContent.CreateTime
	replyTextMsg.MsgType = "text"
	replyTextMsg.Content = text
	replyMsg, _ := xml.Marshal(replyTextMsg)
	conversation.ReplyTextMsgPool.Put(replyTextMsg)

	encryptMsg, cryptErr := wxcrypt.EncryptMsg(string(replyMsg), timestamp, nonce)
	if cryptErr != nil {
		utils.CheckError(errors.New(cryptErr.ErrMsg), "加密被动回复消息")
	}
	_, err = w.Write(encryptMsg)
	return
}

// 被动回复消息
func replyTextWithCtx(ctx conversation.ConversationContext, text string) (err error) {
	replyTextMsg := conversation.ReplyTextMsgPool.Get().(*conversation.ReplyTextMsg)
	replyTextMsg.ToUsername = ctx.ReceiveContent.FromUsername
	replyTextMsg.FromUsername = ctx.ReceiveContent.ToUsername
	replyTextMsg.CreateTime = ctx.ReceiveContent.CreateTime
	replyTextMsg.MsgType = "text"
	replyTextMsg.Content = text
	replyMsg, _ := xml.Marshal(replyTextMsg)
	conversation.ReplyTextMsgPool.Put(replyTextMsg)

	encryptMsg, cryptErr := wxcrypt.EncryptMsg(string(replyMsg), ctx.Timestamp, ctx.Nonce)
	if cryptErr != nil {
		utils.CheckError(errors.New(cryptErr.ErrMsg), "加密被动回复消息")
	}
	_, err = ctx.W.Write(encryptMsg)
	return
}

// 主动发送消息
func sendTextWithCtx(ctx conversation.ConversationContext, text string) error {
	return sendTextToUser(text, ctx.ReceiveContent.FromUsername)
}

//主动发送消息

func sendTextToUser(text string, userName string) error {
	initiativeMsgResponse := &conversation.InitiativeMsgResponse{}
	initiativeTextMsg := conversation.InitiativeTextMsgPool.Get().(*conversation.InitiativeTextMsg)
	initiativeTextMsg.Agentid = botConfig.AgentId
	initiativeTextMsg.Msgtype = "text"
	initiativeTextMsg.Touser = userName
	initiativeTextMsg.Text.Content = text
	client := &http.Client{}
	jsonMsg, err := json.Marshal(initiativeTextMsg)
	if err != nil {
		return err
	}
	conversation.InitiativeTextMsgPool.Put(initiativeTextMsg)
	// 最多重试3次
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest("POST", "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token="+botConfig.AccessToken, bytes.NewReader(jsonMsg))
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// 需要检查token是否有效，无效需要重新获取，重新发送
		json.Unmarshal(body, initiativeMsgResponse)
		log.Println(initiativeMsgResponse)
		if initiativeMsgResponse.Errcode != 0 {
			log.Println("主动发送消息出现错误", string(body))
			getAccessToken()
		} else {
			log.Println("成功发送主动消息。")
			break
		}
	}
	return err
}

// 主动发送markdown
// TODO 使用type switch 和上一个方法进行合并 减少重复代码
func sendMDtoUserWithCtx(ctx conversation.ConversationContext, md string) error {
	initiativeMsgResponse := &conversation.InitiativeMsgResponse{}
	markdownMsg := conversation.MarkDownMsg{
		Touser:  ctx.ReceiveContent.FromUsername,
		Msgtype: "markdown",
		Agentid: botConfig.AgentId,
		EnableDuplicateCheck:   0,
		DuplicateCheckInterval: 0,
	}
	markdownMsg.Markdown.Content = md
	client := &http.Client{}
	jsonMsg, err := json.Marshal(markdownMsg)
	if err != nil {
		return err
	}
	// 最多重试3次
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest("POST", "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token="+botConfig.AccessToken, bytes.NewReader(jsonMsg))
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// 需要检查token是否有效，无效需要重新获取，重新发送
		json.Unmarshal(body, initiativeMsgResponse)
		log.Println(initiativeMsgResponse)
		if initiativeMsgResponse.Errcode != 0 {
			log.Println("主动发送Markdown消息出现错误", string(body))
			getAccessToken()
		} else {
			log.Println("成功发送主动Markdown消息。")
			break
		}
	}
	return err
}

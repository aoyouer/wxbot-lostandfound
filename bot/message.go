package bot

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"wxbot-lostandfound/utils"
)

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

//主动发送消息

func sendTextToUser(text string, userName string) error {
	initiativeMsgResponse := &InitiativeMsgResponse{}
	initiativeTextMsg := initiativeTextMsgPool.Get().(*InitiativeTextMsg)
	initiativeTextMsg.Agentid = botConfig.AgentId
	initiativeTextMsg.Msgtype = "text"
	initiativeTextMsg.Touser = userName
	initiativeTextMsg.Text.Content = text
	client := &http.Client{}
	jsonMsg, err := json.Marshal(initiativeTextMsg)
	if err != nil {
		return err
	}
	initiativeTextMsgPool.Put(initiativeTextMsg)
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

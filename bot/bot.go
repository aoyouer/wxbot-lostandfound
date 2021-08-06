package bot

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"wxbot-lostandfound/conversation"
	"wxbot-lostandfound/utils"
	"wxbot-lostandfound/wxbizmsgcrypt"
)

type BotConfig struct {
	CorpId         string
	CorpSecret     string
	AgentId        int
	Token          string
	AccessToken    string
	EncodingAesKey string
}
type TokenResponse struct {
	Errcode     int    `json:"errcode"`
	Errmsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type HandleFunc func(http.ResponseWriter, *http.Request)

var (
	botConfig                                                                  *BotConfig
	wxcrypt                                                                    *wxbizmsgcrypt.WXBizMsgCrypt
	// 会话map,定时清理
	conversationMap map[string]*conversation.Conversation
)

func init() {
	botConfig = new(BotConfig)
	conversationMap = make(map[string]*conversation.Conversation)
}

func GetBotConfig() *BotConfig {
	return botConfig
}

func Start() {
	// receive_id 企业应用的回调，表示corpid
	log.Println("Starting bot...")
	token, err := getAccessToken()
	botConfig.AccessToken = token
	utils.CheckError(err, "初始化获取access token")
	wxcrypt = wxbizmsgcrypt.NewWXBizMsgCrypt(botConfig.Token, botConfig.EncodingAesKey, botConfig.CorpId, wxbizmsgcrypt.XmlType)
	// 静态文件服务器，用于展示图片
	fs := http.FileServer(http.Dir("imgs/"))
	http.Handle("/api/bot/imgs/", http.StripPrefix("/api/bot/imgs/", fs))
	// 开启一个http服务器，接收来自企业微信的消息
	http.HandleFunc("/api/bot/message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			log.Println("接收到回调验证请求")
			protect(handleVerify)(w, r)
		} else if r.Method == "POST" {
			log.Println("接收到消息")
			protect(handleMessage)(w, r)
		}
	})
	log.Fatalln(http.ListenAndServe("127.0.0.1:8888", nil))
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
	msgSignature := r.URL.Query().Get("msg_signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")
	echoStr := r.URL.Query().Get("echostr")
	// 合法性验证
	echoStrBytes, err := wxcrypt.VerifyURL(msgSignature, timestamp, nonce, echoStr)
	if err != nil {
		log.Println("验证失败", err.ErrMsg)
	} else {
		log.Println("验证成功", string(echoStrBytes))
		// 需要返回才能通过验证
		_, err := w.Write(echoStrBytes)
		utils.CheckError(err, "回调url验证写回")
	}
}

func handleMessage(w http.ResponseWriter, r *http.Request) {
	msgSignature := r.URL.Query().Get("msg_signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")
	body, err := ioutil.ReadAll(r.Body)
	utils.CheckError(err, "读取消息body")
	msg, cryptErr := wxcrypt.DecryptMsg(msgSignature, timestamp, nonce, body)
	if cryptErr != nil {
		utils.CheckError(errors.New(cryptErr.ErrMsg), "解密消息")
	}
	// 因为只是需要临时用于反序列化 所以使用了结构池
	msgContent := conversation.MsgContentPool.Get().(*conversation.MsgContent)
	utils.CheckError(xml.Unmarshal(msg, &msgContent), "消息反序列化")
	log.Println("读取到消息", msgContent)
	// 读取当前的会话map
	switch msgContent.MsgType {
	/*
		stage定义 用户在对话中可以查看当前表单状态，并切换对话阶段
			0: 选择一步步对话的模式还是“智能”对话模式 选做
			1: 收到任意消息后发现map中没有该用户的记录，询问用户是 捡到东西还是丢失了东西
			1.1: 用户可以选择添加遗失、捡到东西的记录或者只是查看现在已有的记录 (比如丢失了东西，则查看现在已经被捡到的东西的记录
			2: 询问用户捡到东西、丢失东西的地点
			3: 询问捡到、丢失的东西是什么
			4: 询问用户对物品的描述
			5: 拼接前面所有的内容，分词器提取关键词，生成标签，询问用户补充标签
			6: 询问用户上传图片
			7: 针对捡到东西的用户，询问东西应该去哪里取回

			如果用户捡到了东西，并查看丢失物品列表(或者负责人进行更新), 如果捡到的东西和丢失列表匹配了则可选择告知登记丢失了物品的失主，
			失主或者管理员可以标记丢失物品的记录为已完成，完成时会通知丢失物品的人 进行感谢

			如果用户丢失了东西，也可以只是查看捡到的失物列表

			可选:
			通过标签进行模糊查找
			添加失物(丢失或捡到)后，根据标签列出当前记录的信息，让用户查看是否捡到的东西、丢失的东西已经有记录了

			添加记录后(丢失或者捡到)向指定群聊推送消息
			定时推送消息
	*/
	case "text":
		err = startConversation(msgContent, nil, w, timestamp, nonce)
		utils.CheckError(err, "被动回复消息")
	case "image":
		imgMsgContent := conversation.ImgMsgContentPool.Get().(*conversation.ImgContent)
		_ = xml.Unmarshal(msg, imgMsgContent)
		err = startConversation(msgContent, imgMsgContent, w, timestamp, nonce)
		conversation.ImgMsgContentPool.Put(imgMsgContent)
	default:
		// 无法处理的消息
		err = replyText(*msgContent, w, timestamp, nonce, "抱歉,机器人无法处理当前类型消息。")
		utils.CheckError(err, "被动回复消息(消息无法处理)")
	}
	conversation.MsgContentPool.Put(msgContent)
}

func protect(function HandleFunc) HandleFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			r := recover()
			if r != nil {
				if err, ok := r.(error); ok {
					log.Println(err.Error())
				} else {
					log.Printf("%v", r)
				}
			}
		}()
		function(writer, request)
	}
}

func getAccessToken() (accessToken string, err error) {
	resp, err := http.Get(fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", botConfig.CorpId, botConfig.CorpSecret))
	accessTokenResponse := new(TokenResponse)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, accessTokenResponse)
	if err != nil {
		return
	}
	accessToken = accessTokenResponse.AccessToken
	botConfig.AccessToken = accessTokenResponse.AccessToken
	return
}

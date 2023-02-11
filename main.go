package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
	"wechat-chatGPT/convert"
	"wechat-chatGPT/gtp"
	"wechat-chatGPT/service"
	"wechat-chatGPT/util"
)

const wxToken = "jindingwen" // 这里填微信开发平台里设置的 Token

var reqGroup singleflight.Group

var UserService = service.NewUserService()

// UserQuestion 用户询问内容
var UserQuestion = make(map[string]string, 0)
var UserAnswer = make(map[string]string, 0)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(util.DefaultLogFormatter())
}

func main() {
	r := chi.NewRouter()

	r.Use(middleware.RequestLogger(
		&middleware.DefaultLogFormatter{
			Logger:  log.StandardLogger(),
			NoColor: runtime.GOOS == "windows",
		}))
	r.Use(middleware.Recoverer)

	// 微信接入校验
	r.Get("/botGPT", wechatCheck)
	// 微信消息处理
	r.Post("/botGPT", wechatMsgReceive)

	l, err := net.Listen("tcp", ":7458")
	if err != nil {
		log.Fatalln(err)
	}
	log.Infof("Server listening at %s", l.Addr())
	if err = http.Serve(l, r); err != nil {
		log.Fatalln(err)
	}
}

// 微信接入校验
func wechatCheck(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	signature := query.Get("signature")
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")
	echostr := query.Get("echostr")

	// 校验
	if util.CheckSignature(signature, timestamp, nonce, wxToken) {
		render.PlainText(w, r, echostr)
		return
	}

	log.Errorln("微信接入校验失败")
}

// 微信消息处理
func wechatMsgReceive(w http.ResponseWriter, r *http.Request) {
	// 解析消息
	body, _ := io.ReadAll(r.Body)
	xmlMsg := convert.ToTextMsg(body)

	log.Infof("[消息接收] Type: %s, From: %s, MsgId: %d, Content: %s", xmlMsg.MsgType, xmlMsg.FromUserName, xmlMsg.MsgId, xmlMsg.Content)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// 回复消息
	replyMsg := ""

	// 关注公众号事件
	if xmlMsg.MsgType == "event" {
		if xmlMsg.Event == "unsubscribe" {
			//chatGPT.DefaultGPT.DeleteUser(xmlMsg.FromUserName)
			log.Infof("[取消订阅] From: %s", xmlMsg.FromUserName)
			return
		}
		if xmlMsg.Event != "subscribe" {
			util.TodoEvent(w)
			log.Infof("[订阅] From: %s", xmlMsg.FromUserName)
			return
		}
		replyMsg = ":) 感谢你发现了这里，灵境魔盒的AiGPT很高兴为您服务"
	} else if xmlMsg.MsgType == "text" {
		val, ok := UserQuestion[xmlMsg.FromUserName]
		if !ok {
			UserQuestion[xmlMsg.FromUserName] = xmlMsg.Content
			// 空回复，保证返回时间
			_, err := w.Write([]byte(""))
			if err != nil {
				log.Errorln(err)
			}
		} else {
			log.Infof("找到了  值为%v", val)
			if val == xmlMsg.Content {
				log.Infof("*** 值相同 ***")
				//return //相同的提问直接跳过，返回空字符串
				answer, ok := UserAnswer[xmlMsg.FromUserName]
				if ok {
					replyMsg = answer
					log.Infof("replyMsg: %s", replyMsg)
				} else {
					return
				}
			}
		}
		replyMsg = ReplyText(xmlMsg.FromUserName, xmlMsg.FromUserName, xmlMsg.Content)
		UserAnswer[xmlMsg.FromUserName] = replyMsg
		if len(replyMsg) > 0 {
			textRes := &convert.TextRes{
				ToUserName:   xmlMsg.FromUserName,
				FromUserName: xmlMsg.ToUserName,
				CreateTime:   time.Now().Unix(),
				MsgType:      "text",
				Content:      replyMsg,
			}
			_, err := w.Write(textRes.ToXml())
			if err != nil {
				log.Errorln(err)
			}
		}
	} else {
		util.TodoEvent(w)
		return
	}
}

// ReplyText 发送文本消息到群
func ReplyText(SenderName string, UserID string, Content string) string {
	// 替换掉@文本，设置会话上下文，然后向GPT发起请求。
	requestText := buildRequestText(SenderName, UserID, Content)
	if requestText == "" {
		return ""
	}
	requestText = UserService.GetUserSessionContext(UserID) + requestText
	log.Printf("gtp requestText: %v \n", requestText)
	reply, err := gtp.Completions(requestText)
	if err != nil {
		log.Printf("gtp request error: %v \n", err)
		return "我脑子有些乱了，等我捋一捋思路。"
	}
	if reply == "" {
		return "为啥我脑子空空如也？我傻了吗？"
	}

	// 回复@我的用户
	reply = strings.TrimSpace(reply)
	reply = strings.Trim(reply, "\n")
	// 设置上下文
	UserService.SetUserSessionContext(UserID, Content, reply)
	//reply = "本消息由灵境魔盒ChatGPT回复：\n" + reply
	return reply
}

// buildRequestText 构建请求GPT的文本，替换掉机器人名称，然后检查是否有上下文，如果有拼接上
func buildRequestText(NickName string, SenderID string, Content string) string {
	replaceText := "@" + NickName
	requestText := strings.TrimSpace(strings.ReplaceAll(Content, replaceText, ""))
	if requestText == "" {
		return ""
	}
	requestText = UserService.GetUserSessionContext(SenderID) + requestText
	return requestText
}

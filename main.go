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
	"time"
	"wx-ChatGPT/chatGPT"
	"wx-ChatGPT/convert"
	"wx-ChatGPT/util"
)

const wxToken = "" // 这里填微信开发平台里设置的 Token

var reqGroup singleflight.Group

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		DisableColors:   runtime.GOOS == "windows",
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
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
	r.Get("/weChatGPT", wechatCheck)
	// 微信消息处理
	r.Post("/weChatGPT", wechatMsgReceive)

	l, err := net.Listen("tcp", ":7458")
	if err != nil {
		log.Errorln(err)
	}
	log.Infof("Server listening at %s", l.Addr())
	if err = http.Serve(l, r); err != nil {
		log.Errorln(err)
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

	log.Infof("[消息接收] Type: %s, From: %s, Content: %s", xmlMsg.MsgType, xmlMsg.FromUserName, xmlMsg.Content)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// 回复消息
	replyMsg := ""

	if xmlMsg.MsgType != "text" {
		replyMsg = "不支持的消息类型"
	} else {
		msg, _, _ := reqGroup.Do(xmlMsg.FromUserName, func() (interface{}, error) {
			return chatGPT.DefaultGPT.SendMsg(xmlMsg.Content, xmlMsg.FromUserName), nil
		})
		replyMsg = msg.(string)
	}

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

package main

import (
	log "github.com/sirupsen/logrus"
	"strings"
	"wechat-chatGPT/gtp"
	"wechat-chatGPT/service"
)

var TestUserService = service.NewUserService()

func main() {
	rep := TestReplyText("Name", "ID", "Hello!")
	log.Debug("rep: %v", rep)
}

// ReplyText 发送文本消息到群
func TestReplyText(SenderName string, UserID string, Content string) string {
	// 替换掉@文本，设置会话上下文，然后向GPT发起请求。
	requestText := TestbuildRequestText(SenderName, UserID, Content)
	if requestText == "" {
		return ""
	}
	requestText = TestUserService.GetUserSessionContext(UserID) + requestText
	log.Printf("gtp requestText: %s \n", requestText)
	reply, err := gtp.Completions(requestText)
	if err != nil {
		log.Printf("gtp request error: %v \n", err)
		return "稍等一下，正在清理库存中。"
	}
	if reply == "" {
		return "为啥我脑子空空如也？我傻了吗？"
	}

	// 回复@我的用户
	reply = strings.TrimSpace(reply)
	reply = strings.Trim(reply, "\n")
	// 设置上下文
	TestUserService.SetUserSessionContext(UserID, Content, reply)
	return reply
}

// buildRequestText 构建请求GPT的文本，替换掉机器人名称，然后检查是否有上下文，如果有拼接上
func TestbuildRequestText(NickName string, SenderID string, Content string) string {
	replaceText := "@" + NickName
	requestText := strings.TrimSpace(strings.ReplaceAll(Content, replaceText, ""))
	if requestText == "" {
		return ""
	}
	requestText = TestUserService.GetUserSessionContext(SenderID) + requestText
	return requestText
}

package main

import (
	"bytes"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"wechat-chatGPT/config"
)

// RequestBody 响应体
type RequestBody struct {
	Model            string  `json:"model"`
	Prompt           string  `json:"prompt"`
	MaxTokens        int     `json:"max_tokens"`
	Temperature      float32 `json:"temperature"`
	TopP             int     `json:"top_p"`
	FrequencyPenalty int     `json:"frequency_penalty"`
	PresencePenalty  int     `json:"presence_penalty"`
}

/*
curl https://api.openai.com/v1/completions \
-H "Content-Type: application/json" \
-H "Authorization: Bearer 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" \
-d '{"model": "text-davinci-003", "prompt": "give me good song", "temperature": 0, "max_tokens": 7}'
*/
func test(text string) {
	requestBody := RequestBody{
		Model:            "text-davinci-003",
		Prompt:           text,
		MaxTokens:        1024,
		Temperature:      0.7,
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	}
	requestData, err := json.Marshal(requestBody)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("request gtp json string : %v", string(requestData))
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/completions", bytes.NewBuffer(requestData))
	if err != nil {
		log.Println(err)
		return
	}

	apiKey := config.LoadConfig().ApiKey
	Authorization := "Bearer " + apiKey
	log.Printf("Authorization : %v", string(Authorization))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer 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")
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Printf("error : %v", err)
		return
	}

	defer response.Body.Close()
	log.Println("resp: %v", response)
	if response.StatusCode != 200 {
		log.Println("response.StatusCode: %v", response.StatusCode)
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("error : %v", err)
	}

	log.Println(string(body))
}
func main() {
	test("测试")
}

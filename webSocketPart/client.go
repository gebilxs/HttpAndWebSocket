package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type ResultStruct struct {
	Header Header `json:"header"`
}
type Header struct {
	Score string `json:"score"`
}

func main() {
	// Way 1
	//message := "hello world!"
	//ws, err := websocket.Dial(url, "", origin)
	//if err != nil {
	//	println("Dial error: ", err)
	//}
	//defer ws.Close()
	//
	//var msg = make([]byte, 512)
	//m, err := ws.Write([]byte(message))
	//fmt.Println(msg[:m])
	//if err != nil {
	//	log.Println(err)
	//}
	mes := `{"header":{"name":"xck","message_id":"7d7314f291c1440483f4a3bc19a2d2fb"},"payload":{}}"`
	var header http.Header
	header = make(map[string][]string)
	// 这里不清楚header放置什么具体的参数
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws/v1"}
	ws, res, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		println(err)
		body, _ := ioutil.ReadAll(res.Body)
		fmt.Println(body)
		return
	}
	defer res.Body.Close()
	defer ws.Close()
	err = ws.WriteMessage(1, []byte(mes))
	if err != nil {
		println(err)
	}
	messageTypeRead, messageRead, err := ws.ReadMessage()
	if err != nil {
		panic(err)
	}
	result := &ResultStruct{}
	err = json.Unmarshal(messageRead, &result)
	if err != nil {
		println(err)
	}
	if messageTypeRead != websocket.TextMessage {
		println("1")
	}
	go func() {
		for {
			messageType, messageData, err := ws.ReadMessage()
			if err != nil {
				log.Println(err)
				break
			}
			if messageType == websocket.BinaryMessage {
				print("binary")
			}
			if messageType == websocket.TextMessage {
				fmt.Println(string(messageData))
				break
			}
		}
	}()
}

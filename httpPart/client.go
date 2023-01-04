package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Student struct {
	id   string
	name string
}

type StudentReq struct {
	id   string
	name string
}
type ResultStruct struct {
	Score int `json:"score"`
}

func main() {
	// Create Student
	stu1 := Student{
		id:   "11223344556677889900",
		name: "xck",
	}
	// 进行序列化
	//相对于解码，json.NewEncoder进行大JSON的编码比json.marshal性能高，因为内部使用pool
	//json.NewDecoder用于http连接与socket连接的读取与写入，或者文件读取；
	//json.Unmarshal用于直接是byte的输入。
	//body := bytes.NewReader()

	// 创建缓存器
	body := bytes.NewBuffer([]byte{}) // bytes切片
	stu := json.NewEncoder(body)
	err := stu.Encode(stu1)
	if err != nil {
		println(err)
	}

	//发送请求
	println("body is" + (body).String())
	client := http.Client{Timeout: time.Duration(10) * time.Second}
	addr := "http://localhost:8080"
	resp, err := client.Post(addr, "application/json", body)
	if err != nil {
		println(err)
	}

	//读取返回的数据
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		println(err)
	}

	result := ResultStruct{}
	//此处能否使用
	if err = json.Unmarshal(respBody, &result); err != nil {
		println(err)
		return
	}
	fmt.Println(result.Score)
}

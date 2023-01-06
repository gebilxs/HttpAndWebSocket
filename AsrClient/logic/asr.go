package logic

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"io"
	"io/fs"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var m = sync.Map{}

type AsrLogic struct {
	TaskId, filename        string
	Url                     url.URL
	Conn                    *websocket.Conn
	AudioLen                int64
	StartTime, SendStopTime time.Time
	C                       chan time.Time
}

func DoAsr(cmd *cobra.Command, args []string) {

	err := viper.BindPFlags(cmd.Flags())
	cobra.CheckErr(err)

	filename := viper.GetString("path")
	stat, err := os.Stat(filename)
	//handle error
	cobra.CheckErr(err)
	if stat.IsDir() {
		doAsrDir(filename)
	} else {
		w := new(sync.WaitGroup)
		w.Add(1)
		doAsrFile(w, filename)
		w.Wait()
	}
	//计算并发情况数据
	Calculate()
}
func doAsrFile(w *sync.WaitGroup, filename string) {
	defer w.Done()
	//新建处理文档的线程
	wg := sync.WaitGroup{}
	wg.Add(viper.GetInt("thread"))
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < viper.GetInt("thread"); i++ {
		// define random to avoid requesting at same time
		time.Sleep(time.Duration(r.Intn(240)) * time.Millisecond)
		go func() {
			defer wg.Done()
			l, err := NewAsrLogic(filename)
			//handle errors
			cobra.CheckErr(err)
			err = l.Start()
			cobra.CheckErr(err)
			//sendErrorChan := l.Send()
			//receiveErrorChan := l.Receive()
			select {
			case err = <-l.Send():
			case err = <-l.Receive():
			}
			cobra.CheckErr(err)
			l.Count()

		}()
	}
	wg.Wait()
}
func NewAsrLogic(filename string) (*AsrLogic, error) {
	l := new(AsrLogic)
	l.filename = filename
	l.Url = url.URL{
		Scheme: viper.GetString("scheme"),
		Host:   viper.GetString("addr"), Path: "/ws/v1",
	}
	fmt.Println("connecting to", l.Url.String())
	// create websocket
	conn, resp, err := websocket.DefaultDialer.Dial(l.Url.String(), nil)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			err = errors.Wrapf(err, "Resp body:%s", body)
		}
		return nil, err
	}
	l.Conn = conn
	l.C = make(chan time.Time)
	return l, nil
}
func (l *AsrLogic) Start() error {
	header := map[string]interface{}{
		"namespace": "SpeechTranscriber",
		"name":      "StartTranscription",
	}
	if viper.GetInt("server_type") == 2 {
		header["namespace"] = "SpeechRecognizer"
		header["name"] = "StartRecognition"
	}
	payload := map[string]interface{}{
		"lang_type":                         viper.GetString("lang_type"),
		"enable_intermediate_result":        viper.GetBool("enable_intermediate_result"),
		"sample_rate":                       viper.GetInt("sample_rate"),
		"format":                            viper.GetString("format"),
		"max_sentence_silence":              viper.GetInt("max_sentence_silence"),
		"enable_punctuation_prediction":     viper.GetBool("enable_punctuation_prediction"),
		"enable_inverse_text_normalization": viper.GetBool("enable_inverse_text_normalization"),
		"enable_words":                      viper.GetBool("enable_words"),
		"language_model_id":                 "general",
		"hotwords_id":                       viper.GetString("hotwords_id"),
		"hotwords_weight":                   viper.GetFloat64("hotwords_weight"),
		"correction_words_id":               viper.GetString("correction_words_id"),
		"forbidden_words_id":                viper.GetString("forbidden_words_id"),
	}
	index := strings.LastIndex(l.filename, ".")
	if index != -1 && index+1 < len(l.filename) {
		payload["format"] = l.filename[index+1:]
	}
	// param = header + payload
	param := map[string]interface{}{
		"header":  header,
		"payload": payload,
	}

	err := l.Conn.WriteJSON(param)
	if err != nil {
		return err
	}

	_, message, err := l.Conn.ReadMessage()
	if err != nil {
		return errors.Errorf("Faild read started message,err:%v", err)
	}
	//获取相关数据
	name := gjson.GetBytes(message, "header.name").String()
	if name != "TranscriptionStarted" && name != "RecognitionStarted" {
		return errors.Errorf("TaskFailed,message:%s", message)
	}
	l.TaskId = gjson.GetBytes(message, "header.task_id").String()

	return nil
}
func (l *AsrLogic) Receive() <-chan error {
	var (
		f       *os.File
		message []byte
		errChan = make(chan error)
		err     error
	)

	go func() {
		defer close(errChan)
		if viper.GetBool("save_output") {
			f, err = os.Create(l.TaskId)
			if err != nil {
				errChan <- errors.Errorf("Failed create ouput file,err:%v", err)
				return
			}
			defer f.Close()
		}

		for {
			_, message, err = l.Conn.ReadMessage()
			if err != nil {
				errChan <- errors.Errorf("Failed read asr server message:err:%v", err)
				return
			}

			fmt.Printf("taskId:%s\toutput:%s\n", l.TaskId, message)
			switch gjson.GetBytes(message, "header.name").String() {
			case "SentenceEnd":
				if viper.GetBool("save_output") {
					f.WriteString(fmt.Sprintf("taskId:%s\toutput:%s\n", l.TaskId, message))
				} else {
					f.WriteString(fmt.Sprintf("success!"))
				}
			case "RecognitionCompleted", "TranscriptionCompleted":
				l.C <- time.Now()
				f.WriteString(fmt.Sprintf("%s\n", l.C))
				break
			case "TaskFailed":
				errChan <- errors.Errorf("TaskFailed,message:%s", message)
			}
		}
	}()
	return errChan
}

func (l *AsrLogic) Send() <-chan error {
	var (
		f       *os.File
		buf     []byte
		n       int
		errChan = make(chan error)
		err     error
	)

	go func() {
		defer close(errChan)

		f, err = os.Open(l.filename)
		if err != nil {
			errChan <- errors.Errorf("Failed open input audio,err:%v", err)
			return
		}
		defer f.Close()

		stat, _ := f.Stat()
		l.AudioLen = stat.Size()
		l.StartTime = time.Now()

		switch viper.GetInt("sample_rate") {
		case 8000:
			buf = make([]byte, 3840)
		default:
			buf = make([]byte, 7680)
		}

		for {
			n, err = f.Read(buf)
			// judge whether err == io.EOF
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				errChan <- errors.Errorf("Failed read input audio,err:%v", err)
				return
			}

			if err = l.Conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				errChan <- errors.Errorf("Failed write audio to asr server,err:%v", err)
				return
			}

			if viper.GetBool("sleep") {
				time.Sleep(240 * time.Millisecond)
			}
		}
		l.SendStopTime = time.Now()

		header := map[string]interface{}{
			"namespace": "SpeechTranscriber",
			"name":      "StopTranscription",
		}
		if viper.GetInt("server_type") == 2 {
			header["namespace"] = "SpeechRecognizer"
			header["name"] = "StopRecognition"
		}

		param := map[string]interface{}{"header": header}
		//链接中放入param
		if err = l.Conn.WriteJSON(param); err != nil {
			errChan <- errors.Errorf("Failed write stop,err:%v", err)
			return
		}
	}()
	return errChan
}
func doAsrDir(filename string) {
	w := new(sync.WaitGroup)
	// ioutil.ReadDir(filename) 有更好的速度支持
	err := filepath.WalkDir(filename, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		w.Add(1)
		go doAsrFile(w, path)
		return nil
	})
	cobra.CheckErr(err)
	w.Wait()
	//w := new(sync.WaitGroup)
	//rd, err := ioutil.ReadDir(filename)
	//if err != nil {
	//	errors.Errorf("Read file error:%v", err)
	//}
	//for _, fi := range rd {
	//	if fi.IsDir() {
	//		return nil
	//	}
	//	w.Add(1)
	//	go doAsrFile(w, path)
	//	return nil
	//}
	//w.Wait()
}

func Calculate() {
	if viper.GetInt("thread") <= 1 {
		return
	}

	if viper.GetBool("sleep") {
		calculateDelay()
	} else {
		calculateRate()
	}
	calculateDuration()
}

func calculateDelay() {
	var (
		res []int64
		sum int64
	)
	m.Range(func(key, value any) bool {
		if !strings.HasPrefix(key.(string), "delay:") {
			return true
		}

		res = append(res, value.(int64))
		sum += value.(int64)
		return true
	})
	sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
	fmt.Printf("\n平均尾包延迟:%dms\t最大尾包延迟:%dms\t最小尾包延迟:%dms\n", sum/int64(len(res)), res[len(res)-1], res[0])
}

func calculateRate() {
	var (
		res []float64
		sum float64
	)
	m.Range(func(key, value any) bool {
		if !strings.HasPrefix(key.(string), "rate:") {
			return true
		}

		res = append(res, value.(float64))
		sum += value.(float64)
		return true
	})
	sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
	fmt.Printf("\n平均实时率:%f\t最大实时率:%f\t最小实时率:%f\n", sum/float64(len(res)), res[len(res)-1], res[0])
}

func calculateDuration() {
	var (
		res []int64
		sum int64
	)
	m.Range(func(key, value any) bool {
		if !strings.HasPrefix(key.(string), "duration:") {
			return true
		}

		res = append(res, value.(int64))
		sum += value.(int64)
		return true
	})
	sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
	fmt.Printf("平均识别耗时:%dms\t最大识别耗时:%dms\t最小识别耗时:%dms\n", sum/int64(len(res)), res[len(res)-1], res[0])
}
func (l *AsrLogic) Count() {
	endTime := <-l.C
	asrDuration := float64(endTime.Sub(l.StartTime).Milliseconds()) // ms
	// stat 得到的AudioLen来计算时间的长度
	audioDuration := float64(l.AudioLen) * 1000 / float64(viper.GetInt("sample_rate")) / 16 * 8

	if viper.GetBool("sleep") {
		fmt.Printf("taskId:%s\t识别耗时:%fms\t音频时长:%fms\t尾包延迟:%dms\n", l.TaskId, asrDuration, audioDuration, endTime.Sub(l.SendStopTime).Milliseconds())
		m.Store("delay:"+l.TaskId, endTime.Sub(l.SendStopTime).Milliseconds())
	} else {
		fmt.Printf("taskId:%s\t识别耗时:%fms\t音频时长:%fms\t实时率:%f\n", l.TaskId, asrDuration, audioDuration, asrDuration/audioDuration)
		m.Store("rate:"+l.TaskId, asrDuration/audioDuration)
	}
	m.Store("duration:"+l.TaskId, endTime.Sub(l.StartTime).Milliseconds())
}

package cmd

import (
	"HttpAndWebSocket/AsrClient/logic"
	"github.com/spf13/cobra"
)

var asrCmd = &cobra.Command{
	Use:   "asr",
	Short: "流式音频识别（websocket）",
	Args:  cobra.NoArgs,
	Run:   logic.DoAsr,
}

func init() {
	rootCmd.AddCommand(asrCmd)

	asrCmd.Flags().StringP("scheme", "", "ws", "scheme")
	asrCmd.Flags().StringP("addr", "a", "localhost:7100", "识别服务地址")
	asrCmd.Flags().StringP("path", "p", "./105.wav", "输入音频地址（暂时不支持多路并发的功能）")
	asrCmd.Flags().StringP("lang_type", "l", "zh-cmn-Hans-CN", "识别语种")
	asrCmd.Flags().StringP("format", "", "pcm", "识别采样率")
	asrCmd.Flags().StringP("hotwords_id", "", "default", "热词ID")
	asrCmd.Flags().Float64P("hotwords_weight", "", 0.7, "热词权重")
	asrCmd.Flags().StringP("correction_words", "c", "", "强制替换词ID")
	asrCmd.Flags().StringP("forbidden_words", "f", "", "敏感词ID")
	asrCmd.Flags().IntP("sample_rate", "s", 16000, "采样率")
	asrCmd.Flags().IntP("thread", "", 1, "并发数量")
	asrCmd.Flags().IntP("max_sentence_silence", "", 450, "最大静音片段")
	asrCmd.Flags().IntP("server_type", "", 1, "服务类型 1:transcriber 2: recognizer")
	asrCmd.Flags().BoolP("enable_intermediate_result", "i", true, "是否返回中间结果")
	asrCmd.Flags().BoolP("punctuation_prediction", "", true, "是否加标点")
	asrCmd.Flags().BoolP("enable_inverse_text_normalization", "t", true, "是否打开ITN")
	asrCmd.Flags().BoolP("enable_words", "", false, "是否返回词信息")
	asrCmd.Flags().BoolP("save_output", "", false, "是否保存识别结果")
	asrCmd.Flags().BoolP("sleep", "", false, "发送完每个音频包是否sleep")
}

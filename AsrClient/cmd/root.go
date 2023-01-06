package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// 集成cmd
var rootCmd = &cobra.Command{
	Short:              "识别客户端",
	Long:               "流式识别和非流式识别的测试客户端",
	Use:                "asr-client",
	DisableSuggestions: false,
	Version:            "1.0",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

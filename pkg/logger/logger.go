package logger

import (
	"gin-chat-room/config"
	"log"
	"os"
)

// InitLogger 初始化日志
func InitLogger() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 如果是生产环境，可以设置日志文件
	if config.AppConfig.Server.Mode == "release" {
		// 在生产环境中，可以将日志输出到文件
		// file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		// if err != nil {
		//     log.Fatalln("Failed to open log file:", err)
		// }
		// log.SetOutput(file)
	} else {
		// 开发环境输出到控制台
		log.SetOutput(os.Stdout)
	}

	log.Println("Logger initialized")
}

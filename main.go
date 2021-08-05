package main

import (
	"github.com/spf13/viper"
	"log"
	"wxbot-lostandfound/bot"
	"wxbot-lostandfound/utils"
)

func init() {
	viper.SetConfigName("config.yaml")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
}

func main()  {
	defer func() {
		r := recover()
		if r != nil {
			if err, ok := r.(error); ok {
				log.Println(err.Error())
			} else {
				log.Printf("%v", r)
			}
		}
	}()
	log.Println("Loading config...")
	utils.CheckError(viper.ReadInConfig(),"读取配置文件")
	utils.CheckError(viper.Unmarshal(bot.GetBotConfig()),"反序列化配置文件")
	bot.Start()
}
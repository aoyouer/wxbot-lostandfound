package bot

import (
	"github.com/yanyiwu/gojieba"
	"strings"
	"wxbot-lostandfound/utils"
)

// 简单的关键词提取 返回多个切片 如地点切片 名词切片 以及对于部分会议室的正则匹配

func ParseMsg(msg string) (allWords []string, placeWords []string, nameWords []string) {
	defer utils.MetricTimeCost("分词解析")()
	jieba := gojieba.NewJieba()
	defer jieba.Free()
	// 带标注分词
	allWords = jieba.Tag(msg)
	for _, word := range allWords {
		words := strings.Split(word, "/")
		switch words[1] {
		case "ns":
			placeWords = append(placeWords, words[0])
		case "n":
			nameWords = append(nameWords, words[0])
		}
	}
	return
}

// 根据已经输入的内容 自动生成标签

func GenerateTags(input string) (tags []string) {
	defer utils.MetricTimeCost("标签生成")()
	jieba := gojieba.NewJieba()
	defer jieba.Free()
	allWords := jieba.Tag(input)
	tagSet := make(map[string]struct{})
	for _, word := range allWords {
		words := strings.Split(word, "/")
		switch words[1] {
		// 当前标签提取 名词 地点
		case "ns","n","eng":
			tagSet[words[0]] = struct{}{}
		}
	}
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	return
}

package handler

import (
	"fmt"
	"log"
	"strings"
	"wxbot-lostandfound/dao"
)

func GetRecordMarkdown(recordType int64, status string,tags []string) (mds []string) {
	records := dao.GetRecord(recordType, status,tags)
	log.Printf("类型:%d查找了%d条记录\n",recordType,len(records))
	builder := strings.Builder{}
	for _, record := range records {
		builder.Reset()
		switch record.Type {
		case 1:
			builder.WriteString(fmt.Sprintf("丢失物品记录 ID:%d\n",record.Id))
		case 2:
			builder.WriteString(fmt.Sprintf("捡到物品记录 ID:%d\n",record.Id))
		}
		builder.WriteString(fmt.Sprintf("所在城市:%s\n", record.City))
		builder.WriteString(fmt.Sprintf("物品名称:%s\n", record.ItemName))
		if record.ImgName != "" {
			// TODO 当前链接为硬编码需要修改
			builder.WriteString(fmt.Sprintf("[图片链接](https://thk.ifine.eu/api/bot/imgs/%s)\n", record.ImgName))
		}
		builder.WriteString(fmt.Sprintf("描述:%s\n", record.Description))
		builder.WriteString(fmt.Sprintf(">标签:%s\n", record.Tags))
		if record.Status == "未完成" {
			builder.WriteString("状态:<font color=\"warning\">未完成</font>\n")
		} else {
			builder.WriteString("状态:<font color=\"info\">已完成</font>\n")
		}
		mds = append(mds, builder.String())
	}
	return
}

func GetAllTag() (tags []string) {
	tagRecords := dao.GetAllTag()
	for _,tagRecord := range tagRecords {
		tags = append(tags,tagRecord.TagName)
	}
	return
}

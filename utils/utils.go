package utils

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var CitySlice = []string{"杭州", "上海", "成都", "广州", "北京"}

func CheckError(err error, where string) {
	if err != nil {
		panic("[" + where + "]" + err.Error())
	}
}

func DownloadFile(url string) (downloadFileName string) {
	defer func() {
		r := recover()
		if r != nil {
			err := r.(error)
			log.Println(err.Error())
		}
	}()
	resp, err := http.Get(url)
	CheckError(err, "图片下载")
	fileName := filepath.Base(url)
	log.Println("下载文件", fileName)
	defer resp.Body.Close()
	out, err := os.Create(filepath.Join("imgs", fileName) + ".png")
	downloadFileName = fileName + ".png"
	CheckError(err, "图片下载")
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	CheckError(err, "图片写入")
	return
}

func MetricTimeCost(funcName string) func() {
	start := time.Now()
	return func() {
		cost := time.Since(start)
		fmt.Printf("[耗时统计]: %s %v\n", funcName, cost)
	}
}

func IfWordInSlice(word string, words []string) (found bool) {
	for _, w := range words {
		if word == w {
			found = true
		}
	}
	return
}

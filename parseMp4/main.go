package main

import (
	"flag"
	"log"
)

const (
	vfp = "/Users/yuankang/Movies/testData/videoH265a.flv"
	lfu = "http://172.20.25.20:8082/SP63nBbfmlbW/GSP63nBbfmlbW-brFd0oAgbR.flv"
)

var (
	fp string
	fu string
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile) // 前台打印

	flag.StringVar(&fp, "f", vfp, "vod flv path")
	flag.StringVar(&fu, "u", lfu, "live flv url")
	flag.Parse()
	log.Println(fp, fu)
}

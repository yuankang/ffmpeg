// go run main.go serialize.go &> aaa.log
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"utils"
)

const (
	vfp = "/Users/yuankang/Movies/testData/videoH265a.flv"
	lfu = "http://172.20.25.20:8082/SP63nBbfmlbW/GSP63nBbfmlbW-brFd0oAgbR.flv"
)

var (
	fp  string
	fu  string
	flv Flv
)

type Flv struct {
	FlvHeader
	FlvHeaderSize uint32
	FlvBody       []Tag
}

// 3 1 1 4 = 9
type FlvHeader struct {
	Signature  string // 3byte
	Version    uint8  // 1byte
	Flags      uint8  // 1byte
	HeaderSize uint32 // 4byte
	HaveAudio  bool   //
	HaveVideo  bool   //
}

type Tag struct {
	TagHeader        // 11byte
	TagData   []byte // nbyte
	TagSize   uint32 // 4byte
}

// 1 3 3 1 3 = 11
type TagHeader struct {
	Type        uint8  // 1byte
	DataSize    uint32 // 3byte
	Timestamp   uint32 // 3byte
	TimestampEx uint8  // 1byte
	StreamId    uint32 // 3byte
}

type AudioData struct {
}

type VideoData struct {
	FrameType  uint8  // 4bit
	CodecId    uint8  // 4bit
	PacketType uint8  // 1byte
	CompTime   uint32 // 3byte
	DataLen    uint32
	Data       []byte
}

func GetFlvHeader(f *os.File) {
	flv.FlvHeader.Signature, _ = ReadString(f, 3)
	flv.FlvHeader.Version, _ = ReadUint8(f)
	flv.FlvHeader.Flags, _ = ReadUint8(f)
	flv.FlvHeader.HeaderSize, _ = ReadUint32(f, 4, BE)
	flv.FlvHeaderSize, _ = ReadUint32(f, 4, BE)

	if flv.FlvHeader.Flags&0x4 == 0x4 {
		flv.FlvHeader.HaveAudio = true
	}
	if flv.FlvHeader.Flags&0x1 == 0x1 {
		flv.FlvHeader.HaveVideo = true
	}
}

func GetTag(f *os.File) Tag {
	var tag Tag
	tag.TagHeader.Type, _ = ReadUint8(f)
	tag.TagHeader.DataSize, _ = ReadUint32(f, 3, BE)
	tag.TagHeader.Timestamp, _ = ReadUint32(f, 3, BE)
	tag.TagHeader.TimestampEx, _ = ReadUint8(f)
	tag.TagHeader.StreamId, _ = ReadUint32(f, 3, BE)

	tag.TagData, _ = ReadByte(f, tag.TagHeader.DataSize)
	tag.TagSize, _ = ReadUint32(f, 4, BE)
	return tag
}

func ParseFlv(f *os.File) {
	GetFlvHeader(f)
	log.Printf("%#v", flv)

	var tag Tag
	var data []byte
	var idx int
	for {
		log.Println("------ net tag ------")
		tag = GetTag(f)
		data = tag.TagData
		tag.TagData = nil

		log.Printf("%#v", tag)
		switch tag.Type {
		case 0x8:
			log.Println("Audio", idx, tag.TagHeader.DataSize, tag.TagSize)
		case 0x9:
			log.Println("Video", idx, tag.TagHeader.DataSize, tag.TagSize)
		case 0x12:
			log.Println("Script", idx, tag.TagHeader.DataSize, tag.TagSize)
		default:
			log.Println("Undefined", idx, tag.TagHeader.DataSize, tag.TagSize)
		}

		if tag.TagHeader.DataSize == 0 && tag.TagSize == 0 {
			break
		}

		tag.TagData = data
		flv.FlvBody = append(flv.FlvBody, tag)
		//if tag.Type == 0x9 && idx < 500 {
		if tag.Type == 0x9 {
			GetVideoData(tag, strconv.Itoa(idx))
		}
		idx++
	}

	log.Printf("%#v", flv.FlvHeader)
	//log.Printf("%#v", flv.FlvBody[3])
}

type Rsps struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func GetRsps(code int, msg string) []byte {
	r := Rsps{code, msg}
	d, err := json.Marshal(r)
	if err != nil {
		log.Println(err)
		return d
	}
	log.Println(string(d))
	return d
}

func GetUrlArg(r *http.Request, key string) (string, error) {
	var v string

	s, err := url.PathUnescape(r.URL.String())
	if err != nil {
		log.Println(err)
		return v, err
	}
	u, err := url.Parse(s)
	if err != nil {
		log.Println(err)
		return v, err
	}
	p, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		log.Println(err)
		return v, err
	}

	if p[key] != nil {
		v = p[key][0]
		return v, nil
	}
	return "", fmt.Errorf("have not")
}

func GetVideoData(tag Tag, idx string) {
	var vd VideoData
	vd.FrameType = tag.TagData[0] & 0xf0 >> 4
	vd.CodecId = tag.TagData[0] & 0xf
	vd.PacketType = tag.TagData[1] // 0:conf, 1:nalu
	vd.CompTime = ByteToUint32(tag.TagData[2:5], BE)
	vd.DataLen = tag.TagHeader.DataSize - 5

	log.Printf("%#v, DataSize=%d", tag.TagHeader, tag.TagHeader.DataSize)
	log.Printf("%#v", vd)
	vd.Data = tag.TagData[5:]
	//log.Printf("%x", vd.Data)

	ft := "pFrame"
	if vd.FrameType == 0x1 {
		ft = "keyFrame"
	}
	ct := "h264" // 0x7为h264, 0xc为h265
	if vd.CodecId == 0xc {
		ct = "h265"
	}
	fn := fmt.Sprintf("video_%s_%s_%s_%d.data", idx, ft, ct, vd.DataLen)
	log.Printf("save data to %s", fn)

	n, err := utils.SaveToDisk(fn, vd.Data)
	if err != nil {
		log.Println(n, err)
		return
	}

	f, err := os.Open(fn)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()

	ParseH265(f, fn)
}

func ParseH265(f *os.File, fn string) {
	var i, dataLen uint32
	var naluLen uint32
	var naluData []byte

	for {
		naluLen, _ = ReadUint32(f, 4, BE)
		if naluLen == 0 {
			break
		}
		naluData, _ = ReadByte(f, naluLen)

		log.Printf("%s nalu %d, len=%d, dataStart=%x, dataEnd=%x", fn, i, naluLen, naluData[0:6], naluData[naluLen-5:])
		dataLen += 4 + naluLen
		i++
	}
	log.Printf("have %d nalu, dataLen=%d", i, dataLen)
}

func GetFrameData(idx string) ([]byte, error) {
	fn, _ := strconv.Atoi(idx)
	tag := flv.FlvBody[fn]

	var ts string
	switch tag.Type {
	case 0x8:
		log.Println("Audio", fn, tag.TagHeader.DataSize, tag.TagSize)
		ts = "Audio"
	case 0x9:
		log.Println("Video", fn, tag.TagHeader.DataSize, tag.TagSize)
		ts = "Video"
		GetVideoData(tag, idx)
	case 0x12:
		log.Println("Script", fn, tag.TagHeader.DataSize, tag.TagSize)
		ts = "Script"
	default:
		log.Println("Undefined", fn, tag.TagHeader.DataSize, tag.TagSize)
		ts = "Undefined"
	}

	return GetRsps(200, ts), nil
}

func HttpServer(w http.ResponseWriter, r *http.Request) {
	log.Println("====== new http request ======")
	log.Println(r.Proto, r.Method, r.URL, r.RemoteAddr, r.Host)

	var rsps []byte
	idx, err := GetUrlArg(r, "idx")
	if err != nil {
		log.Println(err)
		goto ERR
	}

	if r.Method == "GET" {
		// http://localhost:8088/framedata?idx=2
		rsps, err = GetFrameData(idx)
	} else if r.Method == "POST" {
		err = fmt.Errorf("undefined POST request")
		goto ERR
	} else {
		err = fmt.Errorf("undefined %s request", r.Method)
		goto ERR
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Content-length", strconv.Itoa(len(rsps)))
	w.Write(rsps)
	return
ERR:
	rsps = GetRsps(500, err.Error())
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-length", strconv.Itoa(len(rsps)))
	w.Write(rsps)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile) // 前台打印

	flag.StringVar(&fp, "f", vfp, "vod flv path")
	flag.StringVar(&fu, "u", lfu, "live flv url")
	flag.Parse()
	log.Println(fp, fu)

	f, err := os.Open(fp)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()

	ParseFlv(f)
	log.Println("====== 数据分析完毕 ======")

	http.HandleFunc("/", HttpServer)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

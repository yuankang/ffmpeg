package main

import (
	"bytes"
	"io"
	"log"
	"os"
)

type Flv struct {
	FlvHeader *FlvHeader
	FlvBodys  []*FlvBody
	Metadata
}

type FlvHeader struct {
	Signature  string // 3byte, FLV
	Version    uint8  // 1byte, 1
	Flags      uint8  // 1byte, 5
	HeaderSize uint32 // 4byte, 9
	FlagsAudio uint8  // 1bit
	FlagsVideo uint8  // 1bit
}

type FlvBody struct {
	PreTagSize   uint32 // 4byte
	TagType      uint8  // 1byte
	TagDataSize  uint32 // 3byte
	TagTimestamp uint32 // 3byte
	TagTimeExtd  uint8  // 1byte
	TagStreamId  uint32 // 3byte
	TagData      []byte // TagDataSize
}

type Metadata struct {
}

var FlvGv Flv

func FlvHeaderParse(f *os.File) (*FlvHeader, error) {
	var fh FlvHeader
	var err error

	fh.Signature, err = ReadString(f, 3)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	fh.Version, err = ReadUint8(f)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	ui8, err := ReadUint8(f)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	fh.Flags = ui8
	fh.FlagsAudio = (ui8 & 0x4) >> 2
	fh.FlagsVideo = ui8 & 0x1

	fh.HeaderSize, err = ReadUint32(f, 4, BE)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &fh, nil
}

func FlvBodyParse(f *os.File) (*FlvBody, error) {
	var fb FlvBody
	var err error

	fb.PreTagSize, err = ReadUint32(f, 4, BE)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	fb.TagType, err = ReadUint8(f)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	fb.TagDataSize, err = ReadUint32(f, 3, BE)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	fb.TagTimestamp, err = ReadUint32(f, 3, BE)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	fb.TagTimeExtd, err = ReadUint8(f)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	fb.TagStreamId, err = ReadUint32(f, 3, BE)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	fb.TagData, err = ReadByte(f, fb.TagDataSize)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &fb, nil
}

func FlvMetadataParse(d []byte) (*Metadata, error) {
	r := bytes.NewReader(d)
	vs, err := AmfUnmarshal(r) // 序列化转结构化
	if err != nil && err != io.EOF {
		log.Println(err)
		return nil, err
	}
	log.Printf("%#v", vs)
	return nil, nil
}

func main() {
	var err error
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	f, err := os.Open("big_buck_bunny0.flv")
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()

	FlvGv.FlvHeader, err = FlvHeaderParse(f)
	log.Printf("%#v", FlvGv.FlvHeader)

	fb, err := FlvBodyParse(f)
	log.Printf("%#v", fb)

	md, err := FlvMetadataParse(fb.TagData)
	log.Printf("%#v", md)

	//fb, err = FlvBodyParse(f)
	//log.Printf("%#v", fb)
}

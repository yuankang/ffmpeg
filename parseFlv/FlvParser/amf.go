package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
)

const (
	// https://developer.aliyun.com/article/344904
	Amf0MarkerNumber        = 0x00 // 1byte类型，8byte数据(double类型)
	Amf0MarkerBoolen        = 0x01 // 1byte类型, 1byte数据
	Amf0MarkerString        = 0x02 // 1byte类型，2byte长度，Nbyte数据
	Amf0MarkerObject        = 0x03 // 1byte类型，然后是N个kv键值对，最后00 00 09; kv键值对: key为字符串(不需要类型标识了) 2byte长度 Nbyte数据, value可以是任意amf数据类型 包括object类型
	Amf0MarkerMovieClip     = 0x04 // reserved, not supported
	Amf0MarkerNull          = 0x05 // 1byte类型，没有数据
	Amf0MarkerUndefined     = 0x06
	Amf0MarkerReference     = 0x07
	Amf0MarkerEcmaArray     = 0x08 // MixedArray, 1byte类型后是4byte的kv个数, 其他和Object差不多
	Amf0MarkerObjectEnd     = 0x09
	Amf0MarkerStrictArray   = 0x0a // StrictArray, 1byte类型后是4byte的数组个数, 1byte类型 + 数据
	Amf0MarkerDate          = 0x0b // 1byte类型, 8byte数据(double类型) 1970.1.1 毫秒, 2byte表示时区
	Amf0MarkerLongString    = 0x0c
	Amf0MarkerUnSupported   = 0x0d
	Amf0MarkerRecordSet     = 0x0e // reserved, not supported
	Amf0MarkerXmlDocument   = 0x0f
	Amf0MarkerTypedObject   = 0x10
	Amf0MarkerAcmPlusObject = 0x11 // AMF3 data, Sent by Flash player 9+
)

type AmfInfo struct {
	CmdName        string
	TransactionId  float64
	App            string `amf:"app" json:"app"`
	FlashVer       string `amf:"flashVer" json:"flashVer"`
	SwfUrl         string `amf:"swfUrl" json:"swfUrl"`
	TcUrl          string `amf:"tcUrl" json:"tcUrl"`
	Fpad           bool   `amf:"fpad" json:"fpad"`
	AudioCodecs    int    `amf:"audioCodecs" json:"audioCodecs"`
	VideoCodecs    int    `amf:"videoCodecs" json:"videoCodecs"`
	VideoFunction  int    `amf:"videoFunction" json:"videoFunction"`
	PageUrl        string `amf:"pageUrl" json:"pageUrl"`
	ObjectEncoding int    // 0 is AMF0, 3 is AMF3
	Type           string
	PublishName    string
	PublishType    string  // live/ record/ append
	StreamName     string  // play cmd use
	Start          float64 // play cmd use
	Duration       float64 // play cmd use, live is -1
	Reset          bool    // play cmd use
}

type Object map[string]interface{}

/////////////////////////////////////////////////////////////////
// amf decode
/////////////////////////////////////////////////////////////////
func AmfUnmarshal(r io.Reader) (vs []interface{}, err error) {
	var v interface{}
	for {
		log.Println("------")
		v, err = AmfDecode(r)
		if err != nil {
			log.Println(err)
			break
		}
		vs = append(vs, v)
	}
	return vs, err
}

func AmfDecode(r io.Reader) (interface{}, error) {
	t, err := ReadUint8(r)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println("AmfType", t)

	switch t {
	case Amf0MarkerNumber:
		return Amf0DecodeNumber(r)
	case Amf0MarkerBoolen:
		return Amf0DecodeBoolean(r)
	case Amf0MarkerString:
		return Amf0DecodeString(r)
	case Amf0MarkerObject:
		return Amf0DecodeObject(r)
	case Amf0MarkerNull:
		return Amf0DecodeNull(r)
	case Amf0MarkerEcmaArray:
		return Amf0DecodeEcmaArray(r)
	case Amf0MarkerStrictArray:
		return Amf0DecodeStrictArray(r)
	case Amf0MarkerDate:
		return Amf0DecodeData(r)
	}
	err = fmt.Errorf("Untreated AmfType %d", t)
	log.Println(err)
	return nil, err
}

func Amf0DecodeNumber(r io.Reader) (float64, error) {
	var ret float64
	err := binary.Read(r, binary.BigEndian, &ret)
	if err != nil {
		if err != io.EOF {
			log.Println(err)
		}
		return 0, err
	}
	log.Println(ret)
	return ret, nil
}

func Amf0DecodeBoolean(r io.Reader) (bool, error) {
	var ret bool
	err := binary.Read(r, binary.BigEndian, &ret)
	if err != nil {
		if err != io.EOF {
			log.Println(err)
		}
		return false, err
	}
	log.Println(ret)
	return ret, nil
}

func Amf0DecodeString(r io.Reader) (string, error) {
	len, err := ReadUint32(r, 2, BE)
	if err != nil {
		if err != io.EOF {
			log.Println(err)
		}
		return "", err
	}

	ret, _ := ReadString(r, len)
	log.Println(len, ret)
	return ret, nil
}

func Amf0DecodeObject(r io.Reader) (Object, error) {
	ret := make(Object)
	for {
		// 00 00 09
		len, _ := ReadUint32(r, 2, BE)
		if len == 0 {
			ReadUint8(r)
			break
		}

		key, _ := ReadString(r, len)
		log.Println(key)

		value, err := AmfDecode(r)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		ret[key] = value
		log.Println(key, value)
	}
	log.Printf("%#v", ret)
	return ret, nil
}

func Amf0DecodeNull(r io.Reader) (interface{}, error) {
	return nil, nil
}

func Amf0DecodeEcmaArray(r io.Reader) (Object, error) {
	len, err := ReadUint32(r, 4, BE)
	if err != nil {
		if err != io.EOF {
			log.Println(err)
		}
		return nil, err
	}
	log.Println("Amf0EcmaArray len", len)

	ret, err := Amf0DecodeObject(r)
	if err != nil {
		log.Println(err)
		if err != io.EOF {
			log.Println(err)
		}
		return nil, err
	}
	log.Printf("%#v", ret)
	return ret, nil
}

func Amf0DecodeStrictArray(r io.Reader) (Object, error) {
	var o Object
	len, err := ReadUint32(r, 4, BE)
	if err != nil {
		if err != io.EOF {
			log.Println(err)
		}
		return o, err
	}
	log.Println("Amf0StrictArray len", len)

	var vs []interface{}
	for i := uint32(0); i < len; i++ {
		v, err := AmfDecode(r)
		if err != nil {
			log.Println(err)
			break
		}
		vs = append(vs, v)
	}
	log.Printf("%#v", vs)
	return o, nil
}

func Amf0DecodeData(r io.Reader) (float64, error) {
	var ret float64
	err := binary.Read(r, binary.BigEndian, &ret)
	if err != nil {
		if err != io.EOF {
			log.Println(err)
		}
		return 0, err
	}

	ui16, err := ReadUint16(r, 2, BE)
	if err != nil {
		log.Println(err)
		return 0, err
	}

	log.Println(ret, ui16)
	return ret, nil
}

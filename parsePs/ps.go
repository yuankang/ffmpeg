package main

import (
	"fmt"
	"log"
	"os"
	"utils"
)

//系统时钟参考(SCR)是一个分两部分编码的42位字段。
//第一部分system_clock_reference_base是一个长度为33位
//第二部分system_clock_reference_extenstion是一个长度为9位
//SCR字段指出了基本流中包含ESCR_base最后一位的字节到达节目目标解码器输入端的期望时间。
// 4 + 1 + 5(40b) + 3(24b) + 2 = 15byte
// 4(32bit) + 10(80bit) + 3(24bit) + 1(8bit) = 18(144bit)
type PsHeader struct {
	PackStartCode      uint32 // 32bit, 固定值 0x000001BA 表示包的开始
	Reversed0          uint8  // 2bit, 0x01
	ScrBase32_30       uint8  // 3bit, SystemClockReferenceBase32_30
	MarkerBit0         uint8  // 1bit, 标记位 0x1
	ScrBase29_15       uint16 // 15bit, 系统时钟参考基准
	MarkerBit1         uint8  // 1bit 标记位 0x1
	ScrBase14_0        uint16 // 15bit
	MarkerBit2         uint8  // 1bit 标记位 0x1
	ScrExtension       uint16 // 9bit, 系统时钟参考扩展, SystemClockReferenceExtension
	MarkerBit3         uint8  // 1bit 标记位 0x1
	ProgramMuxRate     uint32 // 22bit, 节目复合速率
	MarkerBit4         uint8  // 1bit 标记位 0x1
	MarkerBit5         uint8  // 1bit 标记位 0x1
	Reserved1          uint8  // 5bit
	PackStuffingLength uint8  // 3bit, 该字段后填充字节的个数
	StuffingByte       []byte // 8bit, 填充字节 0xff
	//SystemHeader
}

type SysStream struct {
	StreamId uint8 // 8bit, 流标识, 指示其后的P-STD_buffer_bound_scale和P-STD_buffer_size_bound字段所涉及的流的编码和基本流号码。
	//若取值'1011 1000'，则其后的P-STD_buffer_bound_scale和P-STD_buffer_size_bound字段指节目流中所有的音频流。
	//若取值'1011 1001'，则其后的P-STD_buffer_bound_scale和P-STD_buffer_size_bound字段指节目流中所有的视频流。
	//若stream_id取其它值，则应该是大于或等于'1011 1100'的一字节值且应根据表2-18解释为流的编码和基本流号码。
	Reversed             uint8  // 2bit, 0x11
	PStdBufferBoundScale uint8  // 1bit, 缓冲区界限比例, 表示用于解释后续P-STD_buffer_size_bound字段的比例系数。若前面的stream_id表示一个音频流，则该字段值为'0'。若表示一个视频流，则该字段值为'1'。对于所有其它的流类型，该字段值可以为'0'也可以为'1'。
	PStdBufferSizeBound  uint16 // 13bit, 缓冲区大小界限, 若P-STD_buffer_bound_scale的值为'0'，则该字段以128字节为单位来度量缓冲区大小的边界。若P-STD_buffer_bound_scale的值为'1'，则该字段以1024字节为单位来度量缓冲区大小的边界。
}

// 4 + 2 + 3 + 1 + 1 + 2 + 2 = 15byte
type PsSystemHeader struct {
	SystemHeaderStartCode     uint32 // 32bit, 固定值 0x000001BB
	HeaderLength              uint16 // 16bit, 表示后面还有多少字节
	MarkerBit0                uint8  // 1bit
	RateBound                 uint32 // 22bit, 速率界限, 取值不小于编码在节目流的任何包中的program_mux_rate字段的最大值。该字段可被解码器用于估计是否有能力对整个流解码。
	MarkerBit1                uint8  // 1bit
	AudioBound                uint8  // 6bit, 音频界限, 取值是在从0到32的闭区间中的整数
	FixedFlag                 uint8  // 1bit, 固定标志, 1表示比特率恒定, 0表示比特率可变
	CspsFlag                  uint8  // 1bit, 1表示节目流符合2.7.9中定义的限制
	SystemAudioLockFlag       uint8  // 1bit, 系统音频锁定标志, 1表示在系统目标解码器的音频采样率和system_clock_frequency之间存在规定的比率
	SystemVideoLockFlag       uint8  // 1bit, 系统视频锁定标志, 1表示在系统目标解码器的视频帧速率和system_clock_frequency之间存在规定的比率
	MarkerBit2                uint8  // 1bit
	VideoBound                uint8  // 5bit, 视频界限, 取值是在从0到16的闭区间中的整数
	PacketRateRestrictionFlag uint8  // 1bit, 分组速率限制, 若CSPS标识为'1'，则该字段表示2.7.9中规定的哪个限制适用于分组速率。若CSPS标识为'0'，则该字段的含义未定义
	ReservedBits              uint8  // 7bit, 保留位字段 0x7f
	SysStreams                []SysStream
}

//StreamType  uint8  // 8bit, 表示PES分组中的基本流且取值不能为0x05
//0x10	MPEG-4 视频流
//0x1B	H.264 视频流
//0x24  H.265 视频流, ISO/IEC 13818-1:2018 增加了这个
//0x80	SVAC 视频流
//0x90	G.711 音频流
//0x92	G.722.1 音频流
//0x93	G.723.1 音频流
//0x99	G.729 音频流
//0x9B	SVAC音频流
//ElementaryStreamId uint8  // 8bit 指出PES分组中stream_id字段的值
//0x(C0~DF)指音频
//0x(E0~EF)为视频
// StreamType && ElementaryStreamId 可以判断是 h264 还是 h265
// StreamType == 0x1b && ElementaryStreamId == 0xe0 这个是h264
// StreamType == 0x24 && ElementaryStreamId == 0xe0 这个是h265
// 1 + 1 + 2 = 4byte
type PsMapStream struct {
	StreamType                 uint8  // 8bit, 表示PES分组中的基本流且取值不能为0x05
	ElementaryStreamId         uint8  // 8bit, 指出PES分组中stream_id字段的值, 其中0x(C0~DF)指音频, 0x(E0~EF)为视频
	ElementaryStreamInfoLength uint16 // 16bit, 指出紧跟在该字段后的描述的字节长度
}

// 节目流映射
// 4 + 2 + 2 + 4 + 4*n + 4 = 20byte
type PsMap struct {
	PacketStartCodePrefix     uint32        // 24bit, 固定值 0x000001
	MapStreamId               uint8         // 8bit, 映射流标识 值为0xBC
	ProgramStreamMapLength    uint16        // 16bit, 表示后面还有多少字节, 该字段的最大值为0x3FA(1018)
	CurrentNextIndicator      uint8         // 1bit, 1表示当前可用, 0表示下个可用
	Reserved0                 uint8         // 2bit
	ProgramStreamMapVersion   uint8         // 5bit, 表示整个节目流映射的版本号, 节目流映射的定义发生变化，该字段将递增1，并对32取模
	Reserved1                 uint8         // 7bit
	MarkerBit                 uint8         // 1bit
	ProgramStreamInfoLength   uint16        // 16bit, 紧跟在该字段后的描述符的总长度
	ElementaryStreamMapLength uint16        // 16bit, 基本流映射长度, PsMapStream的长度
	PsMapStreams              []PsMapStream // 32bit, 基本流信息
	CRC32                     uint32        // 32bit
}

// 针对H264 做如下PS 封装：每个IDR NALU 前一般都会包含SPS、PPS 等NALU，因此将SPS、PPS、IDR 的NALU 封装为一个PS 包，包括ps 头，然后加上PS system header，PS system map，PES header+h264 raw data。
// 所以一个IDR NALU PS 包由外到内顺序是：PSheader| PS system header | PS system Map | PES header | h264 raw data。
// 对于其它非关键帧的PS 包，就简单多了，直接加上PS头和PES 头就可以了。顺序为：PS header | PES header | h264raw data。
// 以上是对只有视频video 的情况，如果要把音频Audio也打包进PS 封装，也可以。
// 当有音频数据时，将数据加上PES header 放到视频PES 后就可以了。
// 顺序如下：PS 包=PS头|PES(video)|PES(audio)，再用RTP 封装发送就可以了。

func Parse(f *os.File) {
	var code uint32
	var psIdx int
	for i := 0; i < 150; i++ {
		//log.Printf("------ idx %d ------", i)
		code, _ = ReadUint32(f, 4, BE)
		//log.Printf("start code: %#v", code)
		switch code {
		case 0x1b9:
			log.Println("psEnd")
		case 0x1ba:
			//log.Println("psStart")
			if i != 0 {
				psIdx += 1
			}
			ParsePs(f, code, psIdx, i)
		case 0x1bb:
			//log.Println("sysHeader")
			ParseSysHeader(f, code, psIdx, i)
		case 0x1bc:
			//log.Println("psMap")
			ParsePsMap(f, code, psIdx, i)
		case 0x1c0:
			//log.Println("audio data")
			ParsePes(f, 0xc0, psIdx, i)
		case 0x1e0:
			//log.Println("video data")
			ParsePes(f, 0xe0, psIdx, i)
		default:
			log.Printf("undefined start code: %#v", code)
		}
	}
}

func ParsePs(f *os.File, sc uint32, psIdx, i int) {
	var psh PsHeader
	psh.PackStartCode = sc
	b1, _ := ReadUint8(f)                                            // 0100 0111, 47
	psh.Reversed0 = b1 & 0xc0 >> 6                                   // 1100 0000, 1
	psh.ScrBase32_30 = b1 & 0x38 >> 3                                // 0011 1000, 0
	psh.MarkerBit0 = b1 & 0x4 >> 2                                   // 0000 0100, 1
	b2, _ := ReadUint16(f, 2, BE)                                    // 1010 0110 1110 1111, a6ef
	psh.ScrBase29_15 = (uint16(b1) & 0x3 << 13) | (b2 & 0xfff8 >> 3) // 1111 1111 1111 1000
	psh.MarkerBit1 = uint8(b2 & 0x4 >> 2)                            // 0000 0000 0000 0100
	b3, _ := ReadUint16(f, 2, BE)                                    // 1100 1100 0101 0100, cc54
	psh.ScrBase14_0 = (b2 & 0x3 << 13) | (b3 & 0xfff8 >> 3)          // 1111 1111 1111 1000
	psh.MarkerBit2 = uint8(b3 & 0x4 >> 2)                            // 0000 0000 0000 0100
	b1, _ = ReadUint8(f)                                             // 0000 0001, 01
	psh.ScrExtension = (b3 & 0x3 << 7) | (uint16(b1) & 0xf8 >> 1)    // 1111 1110
	psh.MarkerBit3 = (b1 & 0x1)                                      // 0000 0001
	b4, _ := ReadUint32(f, 4, BE)                                    // 0000 0000 0101 1111 0110 1011 1111 1000, 005F 6BF8
	psh.ProgramMuxRate = b4 >> 10                                    // 1111 1111 1111 1111 1111 1100 0000 0000
	psh.MarkerBit4 = uint8(b4 >> 9 & 0x1)                            // 0000 0000 0000 0000 0000 0010 0000 0000
	psh.MarkerBit5 = uint8(b4 >> 8 & 0x1)                            // 0000 0000 0000 0000 0000 0001 0000 0000
	psh.Reserved1 = uint8(b4 >> 3 & 0x1f)                            // 0000 0000 0000 0000 0000 0000 1111 1000
	psh.PackStuffingLength = uint8(b4 & 0x7)                         // 0000 0000 0000 0000 0000 0000 0000 0111
	if psh.PackStuffingLength != 0 {
		l := uint32(psh.PackStuffingLength)
		psh.StuffingByte, _ = ReadByte(f, l)
	}

	info := fmt.Sprintf("%#v", psh)
	//log.Println(info)

	// 0000000_ps0_psHeader_scr.data
	fn := fmt.Sprintf("%07d_ps%d_psHeader_scr.data", i, psIdx)
	log.Printf("save %s", fn)

	_, err := utils.SaveToDisk(fn, []byte(info))
	if err != nil {
		log.Println(err)
		return
	}
}

func ParseSysHeader(f *os.File, sc uint32, psIdx, i int) {
	var sysSubLen uint16
	var sysh PsSystemHeader
	sysh.SystemHeaderStartCode = sc
	sysh.HeaderLength, _ = ReadUint16(f, 2, BE)
	b1, _ := ReadUint8(f)
	sysSubLen += 1
	sysh.MarkerBit0 = b1 >> 7     // 0111 1111
	b2, _ := ReadUint16(f, 2, BE) // 1111 1111 1111 1111
	sysSubLen += 2
	// 00000000 00000000 00000000 00000000
	//            xxxxxx x1111111 11111111
	sysh.RateBound = (uint32(b1) & 0x7f << 15) | (uint32(b2) >> 1)
	sysh.MarkerBit1 = uint8(b2 & 0x1)
	b1, _ = ReadUint8(f)
	sysSubLen += 1
	sysh.AudioBound = b1 >> 2
	sysh.FixedFlag = b1 & 0x2 >> 1
	sysh.CspsFlag = b1 & 0x1
	b1, _ = ReadUint8(f)
	sysSubLen += 1
	sysh.SystemAudioLockFlag = b1 & 0x80 >> 7
	sysh.SystemVideoLockFlag = b1 & 0x40 >> 6
	sysh.MarkerBit2 = b1 & 0x20 >> 5
	sysh.VideoBound = b1 & 0x1f
	b1, _ = ReadUint8(f)
	sysSubLen += 1
	sysh.PacketRateRestrictionFlag = b1 & 0x80 >> 7
	sysh.ReservedBits = b1 & 0x7f

	ssLen := int(sysh.HeaderLength - sysSubLen)
	ssNum := ssLen / 3

	var ss SysStream
	for i := 0; i < ssNum; i++ {
		ss.StreamId, _ = ReadUint8(f)
		sysSubLen += 1
		b1, _ = ReadUint8(f)
		sysSubLen += 1
		ss.Reversed = b1 & 0xc0 >> 6
		ss.PStdBufferBoundScale = b1 & 0x20 >> 5
		b3, _ := ReadUint8(f)
		sysSubLen += 1
		ss.PStdBufferSizeBound = (uint16(b1) & 0x1f << 8) | uint16(b3)
		//log.Printf("%#v", ss)

		sysh.SysStreams = append(sysh.SysStreams, ss)
	}

	//log.Printf("%#v, HeaderLength=%d, sysSubLen=%d", sysh, sysh.HeaderLength, sysSubLen)
	info := fmt.Sprintf("%#v, HeaderLength=%d, sysSubLen=%d", sysh, sysh.HeaderLength, sysSubLen)
	//log.Println(info)

	// 0000133_ps44_sysHeader_stream2.data
	fn := fmt.Sprintf("%07d_ps%d_sysHeader_stream%d.data", i, psIdx, ssNum)
	log.Printf("save %s", fn)

	_, err := utils.SaveToDisk(fn, []byte(info))
	if err != nil {
		log.Println(err)
		return
	}
}

func ParsePsMap(f *os.File, sc uint32, psIdx, i int) {
	var pmSubLen uint16
	var pm PsMap
	pm.PacketStartCodePrefix = sc >> 8
	pm.MapStreamId = uint8(sc & 0xff)
	pm.ProgramStreamMapLength, _ = ReadUint16(f, 2, BE)
	b1, _ := ReadUint8(f)
	pmSubLen += 1
	pm.CurrentNextIndicator = b1 & 0x80 >> 7
	pm.Reserved0 = b1 & 0x60 >> 5
	pm.ProgramStreamMapVersion = b1 & 0x1f
	b1, _ = ReadUint8(f)
	pmSubLen += 1
	pm.Reserved1 = b1 & 0xfe >> 1
	pm.MarkerBit = b1 & 0x1
	pm.ProgramStreamInfoLength, _ = ReadUint16(f, 2, BE)
	pmSubLen += 2
	pm.ElementaryStreamMapLength, _ = ReadUint16(f, 2, BE)
	pmSubLen += 2

	//log.Printf("ProgramStreamInfoLength=%d, ElementaryStreamMapLength=%d", pm.ProgramStreamInfoLength, pm.ElementaryStreamMapLength)

	pmsNum := int(pm.ElementaryStreamMapLength / 4)
	var pms PsMapStream
	for i := 0; i < pmsNum; i++ {
		pms.StreamType, _ = ReadUint8(f)
		pmSubLen += 1
		pms.ElementaryStreamId, _ = ReadUint8(f)
		pmSubLen += 1
		pms.ElementaryStreamInfoLength, _ = ReadUint16(f, 2, BE)
		pmSubLen += 2
		//log.Printf("%#v", pms)

		if pms.ElementaryStreamId == 0xe0 {
			switch pms.StreamType {
			case 0x1b: // 27
				videoCodec = "h264"
			case 0x24: // 36
				videoCodec = "h265"
			default:
				log.Printf("unknow videoCodec %x", pms.ElementaryStreamId)
			}
		}

		if pms.ElementaryStreamId == 0xc0 {
			switch pms.StreamType {
			case 0x90: // 144
				audioCodec = "G711" // AAC ???
			default:
				log.Printf("unknow videoCodec %x", pms.ElementaryStreamId)
			}
		}

		pm.PsMapStreams = append(pm.PsMapStreams, pms)
	}
	pm.CRC32, _ = ReadUint32(f, 4, BE)
	pmSubLen += 4

	//log.Printf("%#v, ProgramStreamMapLength=%d, pmSubLen=%d", pm, pm.ProgramStreamMapLength, pmSubLen)
	info := fmt.Sprintf("%#v, ProgramStreamMapLength=%d, pmSubLen=%d", pm, pm.ProgramStreamMapLength, pmSubLen)
	//log.Println(info)

	// 0000134_ps44_psMap_mapStream5.data
	fn := fmt.Sprintf("%07d_ps%d_psMap_mapStream%d.data", i, psIdx, pmsNum)
	log.Printf("save %s", fn)

	_, err := utils.SaveToDisk(fn, []byte(info))
	if err != nil {
		log.Println(err)
		return
	}
}

func ParsePes(f *os.File, sid uint8, psIdx, i int) {
	var pesh PesHeader
	pesh.PacketStartCodePrefix = 0x1
	pesh.StreamId = sid
	pesh.PesPacketLength, _ = ReadUint16(f, 2, BE)

	var pesSubLen uint16
	var oph OptionalPesHeader
	b1, _ := ReadUint8(f)
	pesSubLen += 1
	oph.FixedValue0 = b1 & 0xc0 >> 6
	oph.PesScramblingControl = b1 & 0x30 >> 4
	oph.PesPriority = b1 & 0x8 >> 3
	oph.DataAlignmentIndicator = b1 & 0x4 >> 2
	oph.Copyright = b1 & 0x2 >> 1
	oph.OriginalOrCopy = b1 & 0x1

	b1, _ = ReadUint8(f)
	pesSubLen += 1
	oph.PtsDtsFlags = b1 & 0xc0 >> 6
	oph.EscrFlag = b1 & 0x20 >> 5
	oph.EsRateFlag = b1 & 0x10 >> 4
	oph.DsmTrickModeFlag = b1 & 0x8 >> 3
	oph.AdditionalCopyInfoFlag = b1 & 0x4 >> 2
	oph.PesCrcFlag = b1 & 0x2 >> 1
	oph.PesExtensionFlag = b1 & 0x1
	oph.PesHeaderDataLength, _ = ReadUint8(f)
	pesSubLen += 1

	pesh.OptionalPesHeader = oph

	switch oph.PtsDtsFlags {
	case 0x0:
		//log.Println("no pts, no dts")
	case 0x1:
		//log.Println("forbidden")
	case 0x2:
		//log.Println("have pts, no dts")
		pesh.Pts = GetTimestamp(f)
		pesh.PtsValue = GetTimestampValue(pesh.Pts)
		pesSubLen += uint16(oph.PesHeaderDataLength)
	case 0x3:
		log.Println("have pts, have dts")
		//oph.Pts = GetTimestamp(f)
		//oph.Dts = GetTimestamp(f)
		//pesSubLen += uint16(oph.PesHeaderDataLength)
	}
	//log.Printf("%#v, PesHeaderDataLength=%d", oph, oph.PesHeaderDataLength)

	switch oph.EscrFlag {
	case 0x0:
		//log.Println("no Escr")
	case 0x1:
		log.Println("have Escr")
	}

	switch oph.EsRateFlag {
	case 0x0:
		//log.Println("no EsRate")
	case 0x1:
		log.Println("have EsRate")
	}

	switch oph.DsmTrickModeFlag {
	case 0x0:
		//log.Println("no DsmTrickMode")
	case 0x1:
		log.Println("have DsmTrickMode")
	}

	switch oph.AdditionalCopyInfoFlag {
	case 0x0:
		//log.Println("no AdditionalCopyInfo")
	case 0x1:
		log.Println("have AdditionalCopyInfo")
	}

	switch oph.PesCrcFlag {
	case 0x0:
		//log.Println("no PesCrc")
	case 0x1:
		log.Println("have PesCrc")
	}

	switch oph.PesExtensionFlag {
	case 0x0:
		//log.Println("no PesExtension")
	case 0x1:
		log.Println("have PesExtension")
	}

	var pes Pes
	pes.PesHeader = pesh
	n := uint32(pesh.PesPacketLength - pesSubLen)
	//log.Printf("PesDataLen %d - %d = %d", pesh.PesPacketLength, pesSubLen, n)
	pes.PesData, _ = ReadByte(f, n)

	//log.Printf("%#v, PesPacketLength=%d", pesh, pesh.PesPacketLength)

	var dt, codec string
	if sid == 0xc0 {
		dt = "audio"
		codec = audioCodec
	}
	if sid == 0xe0 {
		dt = "video"
		codec = videoCodec
	}
	// 0000001_ps0_video_unknow_5222_1546193975940.data
	// 0000135_ps44_video_h265_32724_1546190537220.data
	// 0000144_ps44_audio_G711_320_1546190537220.data
	fn := fmt.Sprintf("%07d_ps%d_%s_%s_%d_%d.data", i, psIdx, dt, codec, n, pesh.PtsValue)
	log.Printf("save %s", fn)

	_, err := utils.SaveToDisk(fn, pes.PesData)
	if err != nil {
		log.Println(err)
		return
	}
}

func GetTimestamp(f *os.File) Timestamp {
	var ts Timestamp
	b1, _ := ReadUint8(f)
	ts.FixValue = b1 & 0xf0 >> 4 // 1111 0000
	ts.Ts32_30 = b1 & 0xe >> 1   // 0000 1110
	ts.MarkerBit0 = b1 & 0x1     // 0000 0001
	b2, _ := ReadUint16(f, 2, BE)
	ts.Ts29_15 = b2 & 0xfffe >> 1   // 1111 1111 1111 1110
	ts.MarkerBit1 = uint8(b2 & 0x1) // 0000 0000 0000 0001
	b2, _ = ReadUint16(f, 2, BE)
	ts.Ts14_0 = b2 & 0xfffe >> 1    // 1111 1111 1111 1110
	ts.MarkerBit2 = uint8(b2 & 0x1) // 0000 0000 0000 0001
	return ts
}

func GetTimestampValue(ts Timestamp) int64 {
	var tsv int64
	// 00000000 00000000 00000000 00000000 00000000
	//        c ccbbbbbb bbbbbbbb baaaaaaa aaaaaaaa
	tsv = (int64(ts.Ts32_30) << 30) | (int64(ts.Ts29_15) << 15) | int64(ts.Ts14_0)
	// tsv个格子 * 每个格子的时间 (1000/90000 毫秒)
	// tsv = tsv * 1000 / 90000, 90kHz
	tsv = tsv / 90 // 单位是毫秒
	return tsv
}

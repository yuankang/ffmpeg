package main

const (
	H264ClockFrequency = 90 // ISO/IEC13818-1中指定, 时钟频率为90kHz
	TsPacketLen        = 188
	PatPid             = 0x0
	PmtPid             = 0x1001
	VideoPid           = 0x100
	AudioPid           = 0x101
	VideoStreamId      = 0xe0
	AudioStreamId      = 0xc0
)

/**********************************************************/
/* tsFile里 tsPacket的顺序和结构
/**********************************************************/
// rtmp流如何生成ts? 详见 notes/tsFormat.md (必看)
// 第1个tsPacket内容为: tsHeader + 0x00 + pat
// 第2个tsPacket内容为: tsHeader + 0x00 + pmt
// *** 每个关键帧都要有sps和pps
// *** 关键帧的 PesPacketLength == 0x0000
// 第3个tsPacket内容为: tsHeader + adaptation(pcr) + pesHeader + 0x00000001 + 0x09 + 0xf0 + 0x00000001 + 0x67 + sps + 0x00000001 + 0x68 + pps + 0x00000001 + 0x65 + keyFrame
// 第4个tsPacket内容为: tsHeader + keyFrame
// ...
// 第388个tsPacket内容为: tsHeader + pesHeader + 0x00000001 + 0x09 + 0xf0 + 0x00000001 + 0x61 + interFrame
// 第389个tsPacket内容为: tsHeader + interFrame
// ...
// 第481个tsPacket内容为: tsHeader + pesHeader + adts + aacFrame
// 第482个tsPacket内容为: tsHeader + aacFrame
// ...

// 0x00000001 或 0x000001 是NALU单元的开始码
//NalRefIdc        uint8 // 2bit, 简写为NRI
//似乎指示NALU的重要性, 如00的NALU解码器可以丢弃它而不影响图像的回放,取值越大, 表示当前NAL越重要, 需要优先受到保护.
//NalUnitType      uint8 // 5bit, 简写为Type
// nal_unit_type	1-23	表示单一Nal单元模式
// nal_unit_type	24-27	表示聚合Nal单元模式, 本类型用于聚合多个NAL单元到单个RTP荷载中
// nal_unit_type	28-29	表示分片Nal单元模式, 将NALU 单元拆分到多个RTP包中发送
// 0, Reserved
// 1, 非关键帧
// 2, 非IDR图像中A类数据划分片段
// 3, 非IDR图像中B类数据划分片段
// 4, 非IDR图像中C类数据划分片段
// 5, 关键帧
// 6, SEI 补充增强信息
// 7, SPS 序列参数集
// 8, PPS 图像参数集
// 9, 分隔符, 后跟1字节 0xf0
// 10, 序列结束
// 11, 码流结束
// 12, 填充
// 13...23, 保留
// 24, STAP-A 单时间聚合包类型A
// 25, STAP-B 单时间聚合包类型B
// 26, MTAP16 多时间聚合包类型(MTAP)16位位移
// 27, MTAP24 多时间聚合包类型(MTAP)24位位移
// 28, FU-A 单个NALU size 大于 MTU 时候就要拆分 使用FU-A
// 29, FU-B 不常用
// 30-31 Reserved
// +---------------+
// |0|1|2|3|4|5|6|7|
// +-+-+-+-+-+-+-+-+
// |F|NRI|  Type   |
// +---------------+
// 1 + 2 + 5 = 1byte
type NaluHeader struct {
	ForbiddenZeroBit uint8 // 1bit, 简写为F
	NalRefIdc        uint8 // 2bit, 简写为NRI, NalUnitType = 5 或者 7 8 的时候 NRI必须是11
	NalUnitType      uint8 // 5bit, 简写为Type
}

/**********************************************************/
/* prepare SpsPpsData and AdtsData
/**********************************************************/

// FF F9 50 80 2E 7F FC
// 11111111 11111001 01010000 10000000 00101110 01111111 11111100
// fff 1 00 1 01 0100 0 010 0 0 0 0 0000101110011 11111111111 00
// 366-2=364, 371-264=7, 7字节adts  0x173 = 371
//ProfileObjectType            uint8  // 2bit
// 0	Main profile
// 1	Low Complexity profile(LC)
// 2	Scalable Sampling Rate profile(SSR)
// 3	(reserved)
//SamplingFrequencyIndex       uint8  // 4bit, 使用的采样率下标
// 0: 96000 Hz
// 1: 88200 Hz
// 2: 64000 Hz
// 3: 48000 Hz
// 4: 44100 Hz
// 5: 32000 Hz
// 6: 24000 Hz
// 7: 22050 Hz
// 8: 16000 Hz
// 9: 12000 Hz
// 10: 11025 Hz
// 11: 8000 Hz
// 12: 7350 Hz
// 13: Reserved
// 14: Reserved
// 15: frequency is written explictly
// ADTS 定义在 ISO 14496-3, P122
// 固定头信息 + 可变头信息(home之后，不包括home)
//28bit固定头 + 28bit可变头 = 56bit, 7byte
type Adts struct {
	Syncword                     uint16 // 12bit, 固定值0xfff
	Id                           uint8  // 1bit, 固定值0x1, MPEG Version: 0 is MPEG-4, 1 is MPEG-2
	Layer                        uint8  // 2bit, 固定值00
	ProtectionAbsent             uint8  // 1bit, 0表示有CRC校验, 1表示没有CRC校验
	ProfileObjectType            uint8  // 2bit, 表示使用哪个级别的AAC，有些芯片只支持AAC LC
	SamplingFrequencyIndex       uint8  // 4bit, 使用的采样率下标
	PrivateBit                   uint8  // 1bit, 0x0
	ChannelConfiguration         uint8  // 3bit, 表示声道数
	OriginalCopy                 uint8  // 1bit
	Home                         uint8  // 1bit
	CopyrightIdentificationBit   uint8  // 1bit
	CopyrightIdentificationStart uint8  // 1bit
	AacFrameLength               uint16 // 13bit, adts头长度 + aac数据长度
	AdtsBufferFullness           uint16 // 11bit, 固定值0x7ff, 表示码率可变
	NumberOfRawDataBlocksInFrame uint8  // 2bit
}

// ffmpeg-4.4.1/libavcodec/adts_header.c
// ff_adts_header_parse() ffmpeg中解析adts的代码

// size = adts头(7字节) + aac数据长度
// 函数A 把[]byte 传给函数B, B修改后 A里的值也会变
func SetAdtsLength(d []byte, size uint16) {
	d[3] = (d[3] & 0xfc) | uint8((size>>11)&0x2) // 最右2bit
	d[4] = (d[4] & 0x00) | uint8((size>>3)&0xff) // 8bit
	d[5] = (d[5] & 0x1f) | uint8((size&0x7)<<5)  // 最左3bit
}

/**********************************************************/
/* pes
/**********************************************************/
// PTS or DTS
// 4 + 3 + 1 + 15 + 1 + 15 + 1 = 5byte
type OptionalTs struct {
	FixedValue1 uint8  // 4bit, PTS:0x0010 or 0x0011, DTS:0x0001
	Ts32_30     uint8  // 3bit, 33bit
	MarkerBit0  uint8  // 1bit
	Ts29_15     uint16 // 15bit
	MarkerBit1  uint8  // 1bit
	Ts14_0      uint16 // 15bit
	MarkerBit2  uint8  // 1bit
}

// 1 + 2 + 2 = 5byte
type Timestamp struct {
	FixValue   uint8  // 4bit
	Ts32_30    uint8  // 3bit
	MarkerBit0 uint8  // 1bit
	Ts29_15    uint16 // 15bit
	MarkerBit1 uint8  // 1bit
	Ts14_0     uint16 // 15bit
	MarkerBit2 uint8  // 1bit
}

// PtsDtsFlags            uint8  // 2bit
// 0x0 00, 没有PTS和DTS
// 0x1 01, 禁止使用
// 0x2 10, 只有PTS
// 0x3 11, 有PTS 有DTS
// 1 + 1 + 1 = 3byte
type OptionalPesHeader struct {
	FixedValue0            uint8 // 2bit, 固定值0x2
	PesScramblingControl   uint8 // 2bit, 加扰控制
	PesPriority            uint8 // 1bit, 优先级
	DataAlignmentIndicator uint8 // 1bit,
	Copyright              uint8 // 1bit
	OriginalOrCopy         uint8 // 1bit, 原始或复制
	PtsDtsFlags            uint8 // 2bit, 时间戳标志位, 00表示没有对应的信息; 01是被禁用的; 10表示只有PTS; 11表示有PTS和DTS
	EscrFlag               uint8 // 1bit
	EsRateFlag             uint8 // 1bit, 对于PES流而言，它指出了系统目标解码器接收PES分组的速率。该字段在它所属的PES分组以及同一个PES流的后续PES分组中一直有效，直到遇到一个新的ES_rate字段。该字段的值以50字节/秒为单位，且不能为0。
	DsmTrickModeFlag       uint8 // 1bit, 表示作用于相关视频流的特技方式(快进/快退/冻结帧)
	AdditionalCopyInfoFlag uint8 // 1bit
	PesCrcFlag             uint8 // 1bit
	PesExtensionFlag       uint8 // 1bit
	PesHeaderDataLength    uint8 // 8bit, 表示后面还有x个字节, 之后就是负载数据
}

type Pes struct {
	PesHeader
	PesData []byte
}

// 6 + 3 + 5 = 14byte
// 6 + 3 + 5 + 5 = 19byte
// pes 最长是 6 + 65536 = 65542字节
type PesHeader struct {
	PacketStartCodePrefix uint32    // 24bit, 固定值 0x000001
	StreamId              uint8     // 8bit, 0xe0视频 0xc0音频
	PesPacketLength       uint16    // 16bit, 包长度, 表示后面还有x个字节的数据，包括剩余的pes头数据和负载数据, 最大值65536
	OptionalPesHeader               // 24bit
	Pts                   Timestamp // 40bit
	Dts                   Timestamp // 40bit
	PtsValue              int64     // 不是包结构成员, 只为打印
	DtsValue              int64     // 不是包结构成员, 只为打印
}

// rtmp里面的数据 是ES(h264/aac), tsFile里是PES
// rtmp的message(chunk)转换为pes
// rtmp里的Timestamp应该是dts
// 一个pes就是一帧数据(关键帧/非关键帧/音频帧)
// PTS 和 DTS
// GOP分为开放式和闭合式, 最后一帧不是P帧为开放式, 最后一帧是P帧为闭合式; GOP中 不能没有I帧，不能没有P帧，可以没有B帧(如监控视频);
// 音频的pts等于dts; 视频I帧(关键帧)的pts等于dts;
// 视频P帧(没有B帧)的pts等于dts; 视频P帧(有B帧)的pts不等于dts;
// 视频B帧(没有P帧)的pts等于dts; 视频B帧(有P帧)的pts不等于dts;

func SetPesPakcetLength(d []byte, size uint16) {
	// 16bit, 最大值65536, 如果放不下就不放了
	if size > 0xffff {
		return
	}
	Uint16ToByte(size, d[4:6], BE)
}

//0x00000001 + 0x09 + 0xf0, ffmpeg转出的ts有这6个字节, 没有也可以
//0x00000001 + 0x67 + sps + 0x00000001 + 0x68 + pps + 0x00000001 + 0x65 + iFrame
// 返回值: pesHeader + pesBody

/**********************************************************/
/* pat
/**********************************************************/
type PatProgram struct {
	ProgramNumber uint16 // 16bit, arr 4byte,  0 is NetworkPid
	Reserved2     uint8  // 3bit, arr
	PID           uint16 // 13bit, arr, NetworkPid or ProgramMapPid
}

// 3 + 5 + 4 + 4 = 16byte
type Pat struct {
	TableId                uint8  // 8bit, 固定值0x00, 表示是PAT
	SectionSyntaxIndicator uint8  // 1bit, 固定值0x1
	Zero                   uint8  // 1bit, 0x0
	Reserved0              uint8  // 2bit, 0x3
	SectionLength          uint16 // 12bit, 表示后面还有多少字节 包括CRC32
	TransportStreamId      uint16 // 16bit, 传输流id, 区别与其他路流id
	Reserved1              uint8  // 2bit, 保留位
	VersionNumber          uint8  // 5bit, 范围0-31，表示PAT的版本号
	CurrentNextIndicator   uint8  // 1bit, 是当前有效还是下一个有效
	SectionNumber          uint8  // 8bit, PAT可能分为多段传输，第一段为00，以后每个分段加1，最多可能有256个分段
	LastSectionNumber      uint8  // 8bit, 最后一个分段的号码
	ProgramNumber          uint16 // 16bit, arr 4byte,  0 is NetworkPid
	Reserved2              uint8  // 3bit, arr
	PID                    uint16 // 13bit, arr, NetworkPid or ProgramMapPid
	CRC32                  uint32 // 32bit
}

func PatCreate() (*Pat, []byte) {
	var pat Pat
	pat.TableId = 0x00
	pat.SectionSyntaxIndicator = 0x1
	pat.Zero = 0x0
	pat.Reserved0 = 0x3
	pat.SectionLength = 0xd // 13 = 5 + 4 + 4
	pat.TransportStreamId = 0x1
	pat.Reserved1 = 0x3
	pat.VersionNumber = 0x0
	pat.CurrentNextIndicator = 0x1
	pat.SectionNumber = 0x0
	pat.LastSectionNumber = 0x0
	pat.ProgramNumber = 0x1
	pat.Reserved2 = 0x7
	pat.PID = PmtPid
	pat.CRC32 = 0

	patData := make([]byte, 16)
	patData[0] = pat.TableId
	patData[1] = (pat.SectionSyntaxIndicator&0x1)<<7 | (pat.Zero&0x1)<<6 | (pat.Reserved0&0x3)<<4 | uint8((pat.SectionLength&0xf00)>>8)
	patData[2] = uint8(pat.SectionLength & 0xff)
	Uint16ToByte(pat.TransportStreamId, patData[3:5], BE)
	patData[5] = (pat.Reserved1&0x3)<<6 | (pat.VersionNumber&0x1f)<<1 | (pat.CurrentNextIndicator & 0x1)
	patData[6] = pat.SectionNumber
	patData[7] = pat.LastSectionNumber
	Uint16ToByte(pat.ProgramNumber, patData[8:10], BE)
	patData[10] = (pat.Reserved2&0x7)<<5 | uint8((pat.PID&0x1f00)>>8)
	patData[11] = uint8(pat.PID & 0xff)

	pat.CRC32 = Crc32Create(patData[:12])
	Uint32ToByte(pat.CRC32, patData[12:16], BE)
	return &pat, patData
}

/**********************************************************/
/* pmt
/**********************************************************/
// StreamType             uint8  // 8bit, arr 5byte
// 0x0f		Audio with ADTS transport syntax
// 0x1b		H.264
// 40bit = 5byte
type PmtStream struct {
	StreamType    uint8  // 8bit, 节目数据类型
	Reserved4     uint8  // 3bit,
	ElementaryPID uint16 // 13bit, 节目数据类型对应的pid
	Reserved5     uint8  // 4bit,
	EsInfoLength  uint16 // 12bit, 私有数据长度
}

// 3 + 9 + 5*2 + 4 = 26byte
type Pmt struct {
	TableId                uint8       // 8bit, 固定值0x02, 表示是PMT
	SectionSyntaxIndicator uint8       // 1bit, 固定值0x1
	Zero                   uint8       // 1bit, 固定值0x0
	Reserved0              uint8       // 2bit, 0x3
	SectionLength          uint16      // 12bit, 表示后面还有多少字节 包括CRC32
	ProgramNumber          uint16      // 16bit, 不同节目此值不同 依次递增
	Reserved1              uint8       // 2bit, 0x3
	VersionNumber          uint8       // 5bit, 指示当前TS流中program_map_secton 的版本号
	CurrentNextIndicator   uint8       // 1bit, 当该字段为1时表示当前传送的program_map_section可用，当该字段为0时，表示当前传送的program_map_section不可用，下一个TS的program_map_section有效。
	SectionNumber          uint8       // 8bit, 0x0
	LastSectionNumber      uint8       // 8bit, 0x0
	Reserved2              uint8       // 3bit, 0x7
	PcrPID                 uint16      // 13bit, pcr会在哪个pid包里出现，一般是视频包里，PcrPID设置为 0x1fff 表示没有pcr
	Reserved3              uint8       // 4bit, 0xf
	ProgramInfoLength      uint16      // 12bit, 节目信息描述的字节数, 通常为 0x0
	PmtStream              []PmtStream // 40bit, 节目信息
	CRC32                  uint32      // 32bit
}

func PmtCreate() (*Pmt, []byte) {
	var pmt Pmt
	pmt.TableId = 0x2
	pmt.SectionSyntaxIndicator = 0x1
	pmt.Zero = 0x0
	pmt.Reserved0 = 0x3
	pmt.SectionLength = 0x17
	pmt.ProgramNumber = 0x1
	pmt.Reserved1 = 0x3
	pmt.VersionNumber = 0x0
	pmt.CurrentNextIndicator = 0x1
	pmt.SectionNumber = 0x0
	pmt.LastSectionNumber = 0x0
	pmt.Reserved2 = 0x7
	pmt.PcrPID = VideoPid
	pmt.Reserved3 = 0xf
	pmt.ProgramInfoLength = 0x0
	pmt.PmtStream = make([]PmtStream, 2)
	pmt.PmtStream[1].StreamType = 0x1b // AVC video stream as defined in ITU-T Rec. H.264 | ISO/IEC 14496-10 Video
	pmt.PmtStream[1].Reserved4 = 0x7
	pmt.PmtStream[1].ElementaryPID = VideoPid
	pmt.PmtStream[1].Reserved5 = 0xf
	pmt.PmtStream[1].EsInfoLength = 0x0
	pmt.PmtStream[0].StreamType = 0xf // ISO/IEC 13818-7 Audio with ADTS transport syntax
	pmt.PmtStream[0].Reserved4 = 0x7
	pmt.PmtStream[0].ElementaryPID = AudioPid
	pmt.PmtStream[0].Reserved5 = 0xf
	pmt.PmtStream[0].EsInfoLength = 0x0
	pmt.CRC32 = 0

	pmtData := make([]byte, 26)
	pmtData[0] = pmt.TableId
	pmtData[1] = (pmt.SectionSyntaxIndicator&0x1)<<7 | (pmt.Zero&0x1)<<6 | (pmt.Reserved0&0x3)<<4 | uint8((pmt.SectionLength&0xf00)>>8)
	pmtData[2] = uint8(pmt.SectionLength & 0xff)
	Uint16ToByte(pmt.ProgramNumber, pmtData[3:5], BE)
	pmtData[5] = (pmt.Reserved1&0x3)<<6 | (pmt.VersionNumber&0x1f)<<1 | (pmt.CurrentNextIndicator & 0x1)
	pmtData[6] = pmt.SectionNumber
	pmtData[7] = pmt.LastSectionNumber
	pmtData[8] = (pmt.Reserved2&0x7)<<5 | uint8((pmt.PcrPID&0x1f00)>>8)
	pmtData[9] = uint8(pmt.PcrPID & 0xff)
	pmtData[10] = (pmt.Reserved3&0xf)<<4 | uint8((pmt.ProgramInfoLength&0xf00)>>8)
	pmtData[11] = uint8(pmt.ProgramInfoLength & 0xff)
	ps0 := pmt.PmtStream[0]
	ps1 := pmt.PmtStream[1]
	pmtData[12] = ps0.StreamType
	pmtData[13] = (ps0.Reserved4&0x7)<<5 | uint8((ps0.ElementaryPID&0x1f00)>>8)
	pmtData[14] = uint8(ps0.ElementaryPID & 0xff)
	pmtData[15] = (ps0.Reserved5|0xf)<<4 | uint8((ps0.EsInfoLength&0xf00)>>8)
	pmtData[16] = uint8(ps0.EsInfoLength & 0xff)
	pmtData[17] = ps1.StreamType
	pmtData[18] = (ps1.Reserved4&0x7)<<5 | uint8((ps1.ElementaryPID&0x1f00)>>8)
	pmtData[19] = uint8(ps1.ElementaryPID & 0xff)
	pmtData[20] = (ps1.Reserved5|0xf)<<4 | uint8((ps1.EsInfoLength&0xf00)>>8)
	pmtData[21] = uint8(ps1.EsInfoLength & 0xff)

	pmt.CRC32 = Crc32Create(pmtData[:22])
	Uint32ToByte(pmt.CRC32, pmtData[22:26], BE)
	return &pmt, pmtData
}

/**********************************************************/
/* crc
/**********************************************************/
var crcTable = []uint32{
	0x00000000, 0x04c11db7, 0x09823b6e, 0x0d4326d9,
	0x130476dc, 0x17c56b6b, 0x1a864db2, 0x1e475005,
	0x2608edb8, 0x22c9f00f, 0x2f8ad6d6, 0x2b4bcb61,
	0x350c9b64, 0x31cd86d3, 0x3c8ea00a, 0x384fbdbd,
	0x4c11db70, 0x48d0c6c7, 0x4593e01e, 0x4152fda9,
	0x5f15adac, 0x5bd4b01b, 0x569796c2, 0x52568b75,
	0x6a1936c8, 0x6ed82b7f, 0x639b0da6, 0x675a1011,
	0x791d4014, 0x7ddc5da3, 0x709f7b7a, 0x745e66cd,
	0x9823b6e0, 0x9ce2ab57, 0x91a18d8e, 0x95609039,
	0x8b27c03c, 0x8fe6dd8b, 0x82a5fb52, 0x8664e6e5,
	0xbe2b5b58, 0xbaea46ef, 0xb7a96036, 0xb3687d81,
	0xad2f2d84, 0xa9ee3033, 0xa4ad16ea, 0xa06c0b5d,
	0xd4326d90, 0xd0f37027, 0xddb056fe, 0xd9714b49,
	0xc7361b4c, 0xc3f706fb, 0xceb42022, 0xca753d95,
	0xf23a8028, 0xf6fb9d9f, 0xfbb8bb46, 0xff79a6f1,
	0xe13ef6f4, 0xe5ffeb43, 0xe8bccd9a, 0xec7dd02d,
	0x34867077, 0x30476dc0, 0x3d044b19, 0x39c556ae,
	0x278206ab, 0x23431b1c, 0x2e003dc5, 0x2ac12072,
	0x128e9dcf, 0x164f8078, 0x1b0ca6a1, 0x1fcdbb16,
	0x018aeb13, 0x054bf6a4, 0x0808d07d, 0x0cc9cdca,
	0x7897ab07, 0x7c56b6b0, 0x71159069, 0x75d48dde,
	0x6b93dddb, 0x6f52c06c, 0x6211e6b5, 0x66d0fb02,
	0x5e9f46bf, 0x5a5e5b08, 0x571d7dd1, 0x53dc6066,
	0x4d9b3063, 0x495a2dd4, 0x44190b0d, 0x40d816ba,
	0xaca5c697, 0xa864db20, 0xa527fdf9, 0xa1e6e04e,
	0xbfa1b04b, 0xbb60adfc, 0xb6238b25, 0xb2e29692,
	0x8aad2b2f, 0x8e6c3698, 0x832f1041, 0x87ee0df6,
	0x99a95df3, 0x9d684044, 0x902b669d, 0x94ea7b2a,
	0xe0b41de7, 0xe4750050, 0xe9362689, 0xedf73b3e,
	0xf3b06b3b, 0xf771768c, 0xfa325055, 0xfef34de2,
	0xc6bcf05f, 0xc27dede8, 0xcf3ecb31, 0xcbffd686,
	0xd5b88683, 0xd1799b34, 0xdc3abded, 0xd8fba05a,
	0x690ce0ee, 0x6dcdfd59, 0x608edb80, 0x644fc637,
	0x7a089632, 0x7ec98b85, 0x738aad5c, 0x774bb0eb,
	0x4f040d56, 0x4bc510e1, 0x46863638, 0x42472b8f,
	0x5c007b8a, 0x58c1663d, 0x558240e4, 0x51435d53,
	0x251d3b9e, 0x21dc2629, 0x2c9f00f0, 0x285e1d47,
	0x36194d42, 0x32d850f5, 0x3f9b762c, 0x3b5a6b9b,
	0x0315d626, 0x07d4cb91, 0x0a97ed48, 0x0e56f0ff,
	0x1011a0fa, 0x14d0bd4d, 0x19939b94, 0x1d528623,
	0xf12f560e, 0xf5ee4bb9, 0xf8ad6d60, 0xfc6c70d7,
	0xe22b20d2, 0xe6ea3d65, 0xeba91bbc, 0xef68060b,
	0xd727bbb6, 0xd3e6a601, 0xdea580d8, 0xda649d6f,
	0xc423cd6a, 0xc0e2d0dd, 0xcda1f604, 0xc960ebb3,
	0xbd3e8d7e, 0xb9ff90c9, 0xb4bcb610, 0xb07daba7,
	0xae3afba2, 0xaafbe615, 0xa7b8c0cc, 0xa379dd7b,
	0x9b3660c6, 0x9ff77d71, 0x92b45ba8, 0x9675461f,
	0x8832161a, 0x8cf30bad, 0x81b02d74, 0x857130c3,
	0x5d8a9099, 0x594b8d2e, 0x5408abf7, 0x50c9b640,
	0x4e8ee645, 0x4a4ffbf2, 0x470cdd2b, 0x43cdc09c,
	0x7b827d21, 0x7f436096, 0x7200464f, 0x76c15bf8,
	0x68860bfd, 0x6c47164a, 0x61043093, 0x65c52d24,
	0x119b4be9, 0x155a565e, 0x18197087, 0x1cd86d30,
	0x029f3d35, 0x065e2082, 0x0b1d065b, 0x0fdc1bec,
	0x3793a651, 0x3352bbe6, 0x3e119d3f, 0x3ad08088,
	0x2497d08d, 0x2056cd3a, 0x2d15ebe3, 0x29d4f654,
	0xc5a92679, 0xc1683bce, 0xcc2b1d17, 0xc8ea00a0,
	0xd6ad50a5, 0xd26c4d12, 0xdf2f6bcb, 0xdbee767c,
	0xe3a1cbc1, 0xe760d676, 0xea23f0af, 0xeee2ed18,
	0xf0a5bd1d, 0xf464a0aa, 0xf9278673, 0xfde69bc4,
	0x89b8fd09, 0x8d79e0be, 0x803ac667, 0x84fbdbd0,
	0x9abc8bd5, 0x9e7d9662, 0x933eb0bb, 0x97ffad0c,
	0xafb010b1, 0xab710d06, 0xa6322bdf, 0xa2f33668,
	0xbcb4666d, 0xb8757bda, 0xb5365d03, 0xb1f740b4,
}

func Crc32Create(src []byte) uint32 {
	crc32 := uint32(0xFFFFFFFF)
	j := byte(0)
	for i := 0; i < len(src); i++ {
		j = (byte(crc32>>24) ^ src[i]) & 0xff
		crc32 = uint32(uint32(crc32<<8) ^ uint32(crcTable[j]))
	}
	return crc32
}

// 四舍五入取整
// 1.4 + 0.5 = 1.9 向下取整为 1
// 1.5 + 0.5 = 2.0 向下取整为 2
// s := uint32(math.Floor((c.Timestamp / 1000) + 0.5))
// 向上取整 math.Ceil(x) 传入和返回值都是float64
// 向下取整 math.Floor(x) 传入和返回值都是float64
// s.TsExtInfo = math.Floor(float64(c.Timestamp-s.TsFirstTs) / 1000)

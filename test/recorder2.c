#include <stdio.h>

extern "C"
{
	#include <libavcodec/avcodec.h>
	#include<libavformat/avformat.h>
	#include <libswscale/swscale.h>
	#include <libavdevice/avdevice.h>
	#include <libavutil/opt.h>
}
#pragma comment(lib,"avcodec.lib")
#pragma comment(lib,"avformat.lib")
#pragma comment(lib,"swscale.lib")
#pragma comment(lib,"avutil.lib")
#pragma comment(lib,"avdevice.lib")
//AVFormatContext* pFormat = NULL;

//AVDictionary* opt = NULL;
AVPacket* packet = NULL;
const char* path = "11.mp4";
SwsContext * swsCtx = NULL;
AVFrame*  frame = NULL;
AVFrame*  frameYUV = NULL;

int main()
{
	AVFormatContext* pFormat = NULL;
	//const char* path = "11.mp4";
	AVDictionary* opt = NULL;
	AVPacket* packet = NULL;
	SwsContext * swsCtx = NULL;
	AVFrame*  frame = NULL;
	AVFrame*  frameYUV = NULL;
	//寻找流
	int VideoStream = -1;
	int AudioStream = -1;
	//读帧
	int go = 0;
	int FrameCount = 0;
	int ret = 0;
	int width = 0;
	int height = 0;
	int fmt = 0;

	const int win_width = 720;
	const int win_height = 480;
	printf("%s\n",avcodec_configuration());

	//测试DLL
	printf("%s\n", avcodec_configuration());
	//注册DLL
	av_register_all();
	//网络
	avformat_network_init();
	
	avdevice_register_all();


	AVInputFormat* ifmt = av_find_input_format("dshow");

	ret = avformat_open_input(&pFormat, "/dev/video0", ifmt, &opt);
	if (ret)
	{
		printf(" avformat_open_input failed\n");
	}
	printf(" avformat_open_input success\n");

	//寻找流信息 =》 H264  width  height
/*
	ret = avformat_find_stream_info(pFormat, NULL);
	if (ret)
	{
		printf(" avformat_find_stream_info failed\n");
		return -1;
	}
	printf(" avformat_find_stream_info success\n");
	int time = pFormat->duration;
	int mbittime = (time / 1000000) / 60;
	int mmintime = (time / 1000000) % 60;
	printf("%d min :%d second\n", mbittime, mmintime);
	//av_dump_format(pFormat, NULL, path, 1);
*/


	VideoStream = av_find_best_stream(pFormat, AVMEDIA_TYPE_VIDEO, -1, -1, NULL, NULL);
	AudioStream = av_find_best_stream(pFormat, AVMEDIA_TYPE_AUDIO, -1, -1, NULL, NULL); //难道不能得到吗？
	printf("VideoStream  is %d,AudioStream is %d\n",VideoStream,AudioStream);
	AVCodec* vCodec = avcodec_find_decoder(pFormat->streams[VideoStream]->codec->codec_id);
	if (!vCodec)
	{
		printf(" avcodec_find_decoder failed\n");
		return -1;
	}
	//avcodec_find_decoder success, codec name: rawvideo, codec long name :raw video (实际打印)
	printf(" avcodec_find_decoder success, codec name: %s, codec long name :%s  \n",vCodec->name,vCodec->long_name);

	ret = avcodec_open2(pFormat->streams[VideoStream]->codec,
		vCodec, NULL);
	if (ret)
	{
		printf(" avcodec_open2 failed\n");
		return -1;
	}
	printf(" avcodec_open2 success\n");

	//开始解码视频
	//申请原始空间  =》创建帧空间
	frame = av_frame_alloc();
	frameYUV = av_frame_alloc();
	width = pFormat->streams[VideoStream]->codec->width;
	height = pFormat->streams[VideoStream]->codec->height;
	printf("width is %d,height is %d\n",width,height);
	fmt = pFormat->streams[VideoStream]->codec->pix_fmt;
	//分配空间  进行图像转换
	int nSize = avpicture_get_size(AV_PIX_FMT_YUV420P,
		width, height);
	uint8_t* buff = NULL;
	buff = (uint8_t*)av_malloc(nSize);

	//一帧图像
	avpicture_fill((AVPicture*)frameYUV, buff, AV_PIX_FMT_YUV420P, width, height);

	//av_malloc 等价于 malloc()
	packet = (AVPacket*)av_malloc(sizeof(AVPacket));

	//转换上下文
	//swsCtx = sws_getCachedContext(swsCtx,XXXX)

	swsCtx = sws_getContext(width, height, (AVPixelFormat)fmt,
		width, height, AV_PIX_FMT_YUV420P, SWS_BICUBIC, NULL, NULL, NULL);
	FILE* yuv = fopen("1.yuv","wb+");

	while (av_read_frame(pFormat, packet) >= 0)
	{
		//判断stream_index
		if (packet->stream_index == AVMEDIA_TYPE_VIDEO)
		{
			//vCodec  pFormat->streams[VideoStream]->codec
			ret = avcodec_decode_video2(pFormat->streams[VideoStream]->codec, frame, &go, packet);
			if (ret<0)
			{
				printf(" avcodec_decode_video2 failed\n");
				return -1;
			}
			if (go)
			{
				sws_scale(swsCtx,
					(const uint8_t**)frame->data,
					frame->linesize,
					0,
					height,
					frameYUV->data,
					frameYUV->linesize
				);
				int size = width* height;
				fwrite(frameYUV->data[0], 1,  size,yuv);
				fwrite(frameYUV->data[1], 1,  size/4, yuv);
				fwrite(frameYUV->data[2], 1,  size/4, yuv);
				FrameCount++;
				printf("frame index:%d \n", FrameCount++);
			}
		}
		av_free_packet(packet);
	}
	fclose(yuv);
	sws_freeContext(swsCtx);
	av_frame_free(&frame);
	av_frame_free(&frameYUV);
	avformat_close_input(&pFormat);

	return 0;
}

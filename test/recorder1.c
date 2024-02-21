#include <iostream>
 
extern "C" 
{
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/avutil.h>
#include <libswscale/swscale.h>
#include <libavdevice/avdevice.h>
}
 
int main()
{
	av_register_all();
	avdevice_register_all();
	//获取设备，初始换上下文
	AVFormatContext* pFormat = NULL;
	AVInputFormat* ifmt = av_find_input_format("dshow");
	AVDictionary *dict = NULL;
	//设置参数（适合设备的处理能力）
	av_dict_set(&dict, "video_size", "320x180", 0);
	int ret = avformat_open_input(&pFormat, "video=Integrated Camera", ifmt, &dict);
	if (ret < 0)
	{
		cout << "avformat_open_input error" << endl;
	}
	//查找流信息-获取音+视频的基本信息，宽高，duration，rate等
	ret = avformat_find_stream_info(pFormat, NULL);
	if (ret < 0)
	{
		cout << "avformat_find_stream_info error:" << ret;
		return -1;
	}
	//获取流
	int avVideoStream = -1;
	int avAudioStream = -1;
	avVideoStream = av_find_best_stream(pFormat, AVMEDIA_TYPE_VIDEO, -1, -1, NULL, NULL);
	//avAudio = av_find_best_stream(pFormat, AVMEDIA_TYPE_AUDIO, -1, -1, NULL, NULL);
	AVCodecContext* videoCodecCtx = pFormat->streams[avVideoStream]->codec;
	//获取解码器
	AVCodec* videoCodec = avcodec_find_decoder(videoCodecCtx->codec_id);
	if (!videoCodec)
	{
		cout << "avcodec_find_decoder error" << endl;
		return -1;
	}
	//打开编码器
	ret = avcodec_open2(videoCodecCtx, videoCodec, NULL);
	if (ret < 0)
	{
		cout << "avcodec_open2" << endl;
		return -1;
	}
	//创建帧空间
	AVFrame* frame = av_frame_alloc();
	AVFrame* frameYUV = av_frame_alloc();
	int width = videoCodecCtx->width;
	int height = videoCodecCtx->height;
	cout << "input frame w=" << width << ", h=" << height << endl;
	AVPixelFormat fmt = (AVPixelFormat)videoCodecCtx->pix_fmt;
	int nSize = avpicture_get_size(AV_PIX_FMT_YUV420P, width, height);
	uint8_t* buff = (uint8_t*)av_malloc(nSize);
	avpicture_fill((AVPicture*)frameYUV, buff, AV_PIX_FMT_YUV420P, width, height);
	AVPacket* packet = (AVPacket*)av_malloc(sizeof(AVPacket));
	SwsContext* swsCtx = sws_getContext(width, height, fmt, width, height, AV_PIX_FMT_YUV420P,
		SWS_BICUBIC, NULL, NULL, NULL);
	int go = 0;
	int frameIndex = 0;
 
	FILE* f1 = fopen("11.yuv", "wb+");
	while(av_read_frame(pFormat, packet) >= 0)
	{
		frameIndex++;
		if (packet->stream_index == AVMEDIA_TYPE_VIDEO)
		{
			ret = avcodec_decode_video2(videoCodecCtx, frame, &go, packet);
			if (ret < 0)
			{
				cout << "avcodec_decode_video2 error" << endl;
				return -1;
			}
			if (go)
			{
				sws_scale(swsCtx, (const uint8_t**)frame->data, frame->linesize, 0,
					height, frameYUV->data, frameYUV->linesize);
				int size = width * height;
				//写文件
				fwrite(frameYUV->data[0], 1, size, f1);
				fwrite(frameYUV->data[1], 1, size/4, f1);
				fwrite(frameYUV->data[2], 1, size/4, f1);
				cout << "frame index=" << frameIndex << ", size=" << size << endl;
			}
		}
		av_free_packet(packet);
	}
	fclose(f1);
	sws_freeContext(swsCtx);
	av_frame_free(&frame);
	av_frame_free(&frameYUV);
	avformat_close_input(&pFormat);
 
	return 0;
}

//gcc recorder.c -lavcodec -lavdevice -lavfilter -lavformat -lavutil -lpostproc -lswresample -lswscale
#include <stdio.h>
#include <libavcodec/avcodec.h>
#include <libavdevice/avdevice.h>
#include <libavformat/avformat.h>
#include <libavutil/avutil.h>
#include <libswscale/swscale.h>

int main(int argc, char const *argv[])
{
	AVFormatContext *fmt_ctx = avformat_alloc_context();
	const AVInputFormat *ifmt = NULL;
	AVDictionary *dict = NULL;
    int ret = 0;
    printf("%s\n", avcodec_configuration());

    /*
    数值越大, 打印信息越多, -8不打印, 56全打印, 默认32
    #define AV_LOG_QUIET    -8
    #define AV_LOG_PANIC     0
    #define AV_LOG_FATAL     8
    #define AV_LOG_ERROR    16
    #define AV_LOG_WARNING  24
    #define AV_LOG_INFO     32
    #define AV_LOG_VERBOSE  40
    #define AV_LOG_DEBUG    48
    #define AV_LOG_TRACE    56
    */
    ret = av_log_get_level();
    printf("logLevel:%d\n", ret);
    av_log_set_level(AV_LOG_TRACE);
    ret = av_log_get_level();
    printf("logLevel:%d\n", ret);

    //av_register_all();
    avdevice_register_all();

    ifmt = av_find_input_format("avfoundation");

    char buf[128];
    //libavformat/demux.c:220
	//ret = avformat_open_input(&fmt_ctx, "test.mp4", NULL, NULL);
	ret = avformat_open_input(&fmt_ctx, "1:", ifmt, NULL);
	if (ret == 0) {
	    printf("avformat_open_input ok, ret=%d\n", ret);
    } else {
		printf("avformat_open_input fail, ret=%d\n", ret);
        av_strerror(ret, buf, sizeof(buf));
        printf("%d, %s\n", ret, buf);
        return -1;
    }

    ret = avformat_find_stream_info(fmt_ctx, NULL);
	if (ret == 0) {
	    printf("avformat_find_stream_info ok, ret=%d\n", ret);
    } else {
		printf("avformat_find_stream_info fail, ret=%d\n", ret);
    }

    int vi = -1;
    vi = av_find_best_stream(fmt_ctx, AVMEDIA_TYPE_VIDEO, -1, -1, NULL, 0);
	printf("video stream is %d\n", ret);


    AVStream *stream = fmt_ctx->streams[vi];
    const AVCodec *dec = avcodec_find_decoder(stream->codecpar->codec_id);
    if (!dec) {
	    printf("avcodec_find_decoder() fail\n");
        return -1;
    }

    AVCodecContext *codec_ctx = avcodec_alloc_context3(dec);
    if (!codec_ctx) {
	    printf("avcodec_alloc_context3() fail\n");
        return -1;
    }

    ret = avcodec_parameters_to_context(codec_ctx, stream->codecpar);
    if (ret < 0) {
	    printf("avcodec_parameters_to_context() fail, ret=%d\n", ret);
        return -1;
    }

    ret = avcodec_open2(codec_ctx, dec, NULL);
    if (ret < 0) {
	    printf("avcodec_open2() fail, ret=%d\n", ret);
        return -1;
    }
	printf("avcodec_open2() ok, ret=%d\n", ret);

    return 0;
}

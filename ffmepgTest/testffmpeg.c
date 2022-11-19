/*
 * Protocol:  FFmpeg类库支持的输入输出协议
 * AVFormat:  FFmpeg类库支持的封装格式
 * AVCodec:   FFmpeg类库支持的编解码器
 * AVFilter:  FFmpeg类库支持的滤镜
 * Configure: FFmpeg类库的配置信息
 */
#include <stdio.h>
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavfilter/avfilter.h>

int main()
{
    av_register_all();

    // Configuration Information
    printf("====== %s ======\n", "Configuration Information");
    const char *s = avcodec_configuration();
    printf("%s\n", s);

    // Protocol Support Information
    printf("====== %s ======\n", "Protocol Support Information");
    struct URLProtocol* p = NULL;
    struct URLProtocol** pp = &p;
    avio_enum_protocols((void**)pp, 0);
    while ((*pp) != NULL) {
        printf("[In ][%s]\n", avio_enum_protocols((void**)pp, 0));
    }
    avio_enum_protocols((void**)pp, 1);
    while ((*pp) != NULL) {
        printf("[Out ][%s]\n", avio_enum_protocols((void**)pp, 0));
    }

    // AVFormat Support Information
    printf("====== %s ======\n", "AVFormat Support Information");
    AVInputFormat* avif = av_iformat_next(NULL);
    AVOutputFormat* avof = av_oformat_next(NULL);
    while (avif != NULL) {
        printf("[In ][%s]\n", avif->name);
        avif = avif->next;
    }
    while (avof != NULL) {
        printf("[Out ][%s]\n", avof->name);
        avof = avof->next;
    }

    // AVCodec Support Information
    printf("====== %s ======\n", "AVCodec Support Information");
    AVCodec* avc = av_codec_next(NULL);
    while (avc != NULL) {
        if (avc->decode != NULL) {
            printf("[Dec]");
        } else {
            printf("[Enc]");
        }
        switch (avc->type) {
            case AVMEDIA_TYPE_VIDEO:
                printf("[Video]");
                break;
            case AVMEDIA_TYPE_AUDIO:
                printf("[Audio]");
                break;
            default:
                printf("[Other]");
                break;
        }
        printf("%20s\n", avc->name);
        avc = avc->next;
    }

    // AVFilter Support Information
    printf("====== %s ======\n", "AVFilter Support Information");
    AVFilter *avf = (AVFilter *)avfilter_next(NULL);
    while (avf != NULL){
        printf("[%s]\n", avf->name);
        avf = avf->next;
    }

    return 0;
}

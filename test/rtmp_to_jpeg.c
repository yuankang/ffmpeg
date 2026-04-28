#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libswscale/swscale.h>
#include <pthread.h>
#include <stdio.h>

// 全局变量，用于存储当前解码的帧
AVFrame *current_frame = NULL;
AVCodecContext *jpegCtx = NULL;
pthread_rwlock_t frame_lock; // 读写锁

void rtmp_pull(const char *rtmp_url) {
    AVFormatContext *fmtCtx = NULL;
    AVCodecContext *videoDecCtx = NULL;
    AVStream *videoStream = NULL;
    struct SwsContext *swsCtx = NULL;
    int videoStreamIdx = -1;
    int ret;

    // 打开RTMP流
    if (avformat_open_input(&fmtCtx, rtmp_url, NULL, NULL) < 0) {
        fprintf(stderr, "无法打开RTMP流 %s\n", rtmp_url);
        return;
    }

    // 获取流信息
    if (avformat_find_stream_info(fmtCtx, NULL) < 0) {
        fprintf(stderr, "无法找到流信息\n");
        return;
    }

    // 查找视频流
    for (int i = 0; i < fmtCtx->nb_streams; i++) {
        if (fmtCtx->streams[i]->codecpar->codec_type == AVMEDIA_TYPE_VIDEO) {
            videoStreamIdx = i;
            break;
        }
    }
    if (videoStreamIdx == -1) {
        fprintf(stderr, "无法找到视频流\n");
        return;
    }

    videoStream = fmtCtx->streams[videoStreamIdx];
    const AVCodec *videoDec = avcodec_find_decoder(videoStream->codecpar->codec_id);
    if (!videoDec) {
        fprintf(stderr, "无法找到视频解码器\n");
        return;
    }

    videoDecCtx = avcodec_alloc_context3(videoDec);
    if (!videoDecCtx) {
        fprintf(stderr, "无法分配视频解码器上下文\n");
        return;
    }

    if (avcodec_parameters_to_context(videoDecCtx, videoStream->codecpar) < 0) {
        fprintf(stderr, "无法将解码器参数复制到上下文\n");
        return;
    }

    if (avcodec_open2(videoDecCtx, videoDec, NULL) < 0) {
        fprintf(stderr, "无法打开解码器\n");
        return;
    }

    // 初始化SWS转换上下文
    swsCtx = sws_getContext(videoDecCtx->width, videoDecCtx->height, videoDecCtx->pix_fmt,
                            videoDecCtx->width, videoDecCtx->height, AV_PIX_FMT_YUVJ420P,
                            SWS_BILINEAR, NULL, NULL, NULL);

    current_frame = av_frame_alloc();
    AVFrame *rgbFrame = av_frame_alloc();
    rgbFrame->format = AV_PIX_FMT_YUVJ420P;
    rgbFrame->width = videoDecCtx->width;
    rgbFrame->height = videoDecCtx->height;
    av_frame_get_buffer(rgbFrame, 32);

    // 准备JPEG编码器
    const AVCodec *jpegEnc = avcodec_find_encoder(AV_CODEC_ID_MJPEG);
    if (!jpegEnc) {
        fprintf(stderr, "无法找到JPEG编码器\n");
        return;
    }

    jpegCtx = avcodec_alloc_context3(jpegEnc);
    if (!jpegCtx) {
        fprintf(stderr, "无法分配JPEG编码器上下文\n");
        return;
    }

    jpegCtx->bit_rate = 400000;
    jpegCtx->width = videoDecCtx->width;
    jpegCtx->height = videoDecCtx->height;
    jpegCtx->pix_fmt = AV_PIX_FMT_YUVJ420P;
    jpegCtx->time_base = videoStream->time_base;

    if (avcodec_open2(jpegCtx, jpegEnc, NULL) < 0) {
        fprintf(stderr, "无法打开JPEG编码器\n");
        return;
    }

    AVPacket *pkt = av_packet_alloc();

    // 读取帧并解码
    while (av_read_frame(fmtCtx, pkt) >= 0) {
        if (pkt->stream_index == videoStreamIdx) {
            ret = avcodec_send_packet(videoDecCtx, pkt);
            if (ret < 0) {
                fprintf(stderr, "发送包到解码器时出错\n");
                break;
            }

            ret = avcodec_receive_frame(videoDecCtx, current_frame);
            if (ret >= 0) {
                // 转换帧格式
                sws_scale(swsCtx, (const uint8_t *const *)current_frame->data, current_frame->linesize,
                          0, videoDecCtx->height, rgbFrame->data, rgbFrame->linesize);

                // 写锁，保护 current_frame
                pthread_rwlock_wrlock(&frame_lock);
                av_frame_copy(current_frame, rgbFrame); // 将RGB数据复制到全局帧
                pthread_rwlock_unlock(&frame_lock);
            }
        }
        av_packet_unref(pkt);
    }

    // 释放资源
    av_frame_free(&rgbFrame);
    sws_freeContext(swsCtx);
    avcodec_free_context(&videoDecCtx);
    avformat_close_input(&fmtCtx);
    av_packet_free(&pkt);
}

void get_jpeg(const char *output_file) {
    if (!current_frame || !jpegCtx) {
        fprintf(stderr, "当前没有解码的帧或JPEG编码器未初始化\n");
        return;
    }

    AVPacket *pkt = av_packet_alloc();

    // 读锁，保护 current_frame
    pthread_rwlock_rdlock(&frame_lock);
    
    // 编码当前帧为JPEG
    int ret = avcodec_send_frame(jpegCtx, current_frame);
    pthread_rwlock_unlock(&frame_lock); // 解锁

    if (ret < 0) {
        fprintf(stderr, "发送帧到编码器时出错\n");
        return;
    }

    ret = avcodec_receive_packet(jpegCtx, pkt);
    if (ret < 0) {
        fprintf(stderr, "接收编码数据时出错\n");
        return;
    }

    // 将JPEG数据保存为文件
    FILE *f = fopen(output_file, "wb");
    if (!f) {
        fprintf(stderr, "无法打开文件 %s\n", output_file);
        av_packet_unref(pkt);
        return;
    }
    fwrite(pkt->data, 1, pkt->size, f);
    fclose(f);

    av_packet_unref(pkt);
    av_packet_free(&pkt);
}

int main() {
    const char *rtmp_url = "rtmp://42.177.94.206/SPu9pBgg2N2Q/GSPu9pBgg2N2Q-gOkj6qZaoE";

    // 初始化读写锁
    pthread_rwlock_init(&frame_lock, NULL);

    rtmp_pull(rtmp_url);
    get_jpeg("output.jpg");

    // 销毁读写锁
    pthread_rwlock_destroy(&frame_lock);

    return 0;
}


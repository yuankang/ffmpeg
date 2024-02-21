//gcc hi.c -lavcodec -lavformat -lavutil -lswscale -lavdevice
//gcc hi.c -lavcodec
#include <stdio.h>
#include <libavcodec/avcodec.h>

int main(int argc, char const *argv[])
{
    printf("%s\n", avcodec_configuration());
    av_log(NULL, AV_LOG_INFO, "the info line:%d, string:%s\n", __LINE__, "hello");
    printf("hi\n");
    return 0;
}

/*
--enable-shared --enable-gpl --enable-nonfree --enable-libfdk-aac --enable-libx264 --enable-libx265 --enable-libmp3lame --enable-libfreetype --pkg-config-flags=--static
the info line:8, string:hello
hi
*/

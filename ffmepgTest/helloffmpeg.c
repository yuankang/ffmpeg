// gcc helloffmpeg.c -lavcodec
#include <stdio.h>
#include <libavcodec/avcodec.h>

int main(int argc, char const *argv[])
{
    printf("%s\n", avcodec_configuration());
    printf("hello ffmpeg\n");
    return 0;
}

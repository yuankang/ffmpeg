// gcc ffmpegHi.c -lavcodec

// make examples

#include <stdio.h>
#include <libavcodec/avcodec.h>

int main(int argc, char const *argv[])
{
    printf("%s\n", avcodec_configuration());
    return 0;
}

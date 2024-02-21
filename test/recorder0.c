#include <iostream>

extern "C" 
{
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/avutil.h>
#include <libswscale/swscale.h>
#include <libavdevice/avdevice.h>
}

#pragma comment(lib, "avcodec.lib")
#pragma comment(lib, "avformat.lib")
#pragma comment(lib, "avutil.lib")
#pragma comment(lib, "swscale.lib")
#pragma comment(lib, "avdevice.lib")

using namespace std;

void showDevice()
{
    AVFormatContext* pFormat = avformat_alloc_context();
    AVDictionary *dict = NULL;
    av_dict_set(&dict, "list_devices", "true", 0);
    AVInputFormat* fmt = av_find_input_format("dshow");
    avformat_open_input(&pFormat, "video=dummy", fmt, &dict);
}

void showOption()
{
    AVFormatContext* pFormat = avformat_alloc_context();
    AVDictionary* dict = NULL;
    av_dict_set(&dict, "list_options", "true", 0);
    AVInputFormat* fmt = av_find_input_format("dshow");
    avformat_open_input(&pFormat, "video=Integrated Camera", fmt, &dict);

}


int main()
{
    av_register_all();
    avdevice_register_all();

    cout << "device list:" << endl;
    showDevice();

    cout << "device option:" << endl;
    showOption();

    return 0;
}

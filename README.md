# rtmpServerStudy
just for study golang and AV knowledge

纯golang直播服务器：
- 安装和使用非常简单；

- golang version >1.9

#### 支持的传输协议
- [x] RTMP
- [x] AMF
- [x] HLS
- [x] HTTP-FLV

#### 支持的容器格式
- [x] FLV
- [x] TS

### 支持推流传输协议
- [x] tcp
- [x] quic
- [x] kcp

#### 支持的编码格式
- [x] H264
- [x] AAC

#### 从源码编译
1. 下载源码 `git clone https://github.com/KouChongYang/rtmpServerStudy`
2. 去 main 目录中 执行 `go build`

## 使用
2. 启动服务：执行 `./main -c config.yaml -p ./ ` 二进制文件启动 rtmp server 服务；
3. 上行推流：通过 `RTMP` 协议把视频流推送到 `rtmp://test.uplive.com:1935/live/test`，
例如使用 `ffmpeg -re -i 4b.flv -c copy -f flv rtmp://127.0.0.1:1935/123?vhost=test.uplive.com/live` 推送；
或者绑定host test.uplive.com 127.0.0.1 直接通过以下命令推送`ffmpeg -re -i 4b.flv -c copy -f flv rtmp://test.uplive.com/live/123`
亦或直接通过obs推流
4. 下行播放：支持以下三种播放协议，播放地址如下：
    - `RTMP`:`rtmp://test.live.com:1935/live/123`
    - `FLV`:`http://test.live.com:8087/live/123.flv`
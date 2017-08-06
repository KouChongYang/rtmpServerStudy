## 1. RTMP

- 协议简介

```
RTMP协议是一个互联网TCP/IP五层体系结构中应用层的协议.
RTMP协议中基本的数据单元称为消息(Message).
当RTMP协议在互联网中传输数据的时候,消息会被拆分成更小的单元,称为消息块(Chunk)

RTMP有几个变种版本
1.原始版本
    基于TCP,默认端口1935
2.RTMPS     RTMP over TLS/SSL
3.RTMPE
4.RTMPT     封装于HTTP请求中
5.RTMFP     RTMP over UDP

RTMP 将传输的流(stream)分成片段(fragment),其大小由服务器和客户端之间动态协商,
默认的fragment大小为: 
    音频数据:   64 bytes
    视频数据:   128 bytes

```
- RTMP Chunk(RTMP消息块)

```
数据的发送并不是以message为单位的,而是将message拆分成chunk(>=1),chunk按序发送,接收端再根据 MsgStreamID 还原成1个message
在网络上传输数据时,消息需要被拆分成数据块(Chunk).如视频128 bytes
ChunkBasicHeader    |   ChunkMsgHeader  |   ExtendedTimeStamp   |   ChunkData 
```
```
    |                           ChunkHeader              |
    +------------------------------------------------------------------+
    |ChunkBasicHeader |ChunkMsgHeader |ExtendedTimeStamp |ChunkData |
    +------------------------------------------------------------------+

解释:
    ChunkBasicHeader:   块基本头, 包含 块流ID和块类型(4中类型)，每种类型的块**必须包含**
    ExtendedTimeStamp:  扩展时间戳（后面详细说明）
```
- chunkBasicHeader 介绍

```
+---------------------------------------+
|   fmt(2 bits)     |   csid(可变长度)   |
+---------------------------------------+

ftm: 表示块类型  （0，1，2）
csid: chunk stream id (小端表示法)   块流ID    范围[3, 65599] 
[0,2]为RTMP协议保留表示特殊信息

块基本头长度可能为1、2、3字节

1). ChunkBasicHeader 为 1 字节时, fmt占2 bits,csid占6 bits csid in [0, 63]
2). ChunkBasicHeader 为 2 字节时, 第一字节除去fmt外全置0, 剩下 1 字节用来表示csid
csid in [64, 2^8 + 64 = 319]
3). ChunkBasicHeader 为 3 字节时,第一字节除去fmt外全置1, 剩下2字节用来表示csid
            csid in [64, 2^16 + 64 = 65599]

            对于出现重叠的情况，应当使用使header**尽可能小**的实现

        代码逻辑上可以这样处理：
        1).先接收1字节，分别取高两位和低六位保存起来，
        2).判断低六位是全0还是全1，如果是全0，表示ChunkBasicHeader长度为2字节，那么再接收1个字节，
        并且csid需要加上64，如果全为1，则表示长度为3字节，需要在接收2字节

  
```
####    ChunkMsgHeader 介绍
```
    块消息头, 发送消息的信息,依据块基本头中的『块类型』分为4种:0,1,2,3
```
##### 块类型0:

- 11 bytes  流开始的第一个块必须使用这种类型,流时间戳回退时也必须使用这种类型头(backward seek)
timestamp（时间戳）：占用3个字节，因此它最多能表示到16777215=0xFFFFFF=2
24-1, 当它的值超过这个最大值时，这三个字节都置为1，这样实际的timestamp会转存到Extended Timestamp字段中，接受端在判断timestamp字段24个位都为1时就会去Extended timestamp中解析实际的时间戳。
- message length（消息数据的长度）：占用3个字节，表示实际发送的消息的数据如音频帧、视频帧等数据的长度，单位是字节。注意这里是Message的长度，也就是chunk属于的Message的总数据长度，而不是chunk本身Data的数据的长度。

- message type id(消息的类型id)：占用1个字节，表示实际发送的数据的类型，如8代表音频数据、9代表视频数据。

- msg stream id（消息的流id）：占用4个字节，表示该chunk所在的流的ID，和Basic Header的CSID一样，它采用小端存储的方式，
-- 结构如下：
```
+----------------------------------------------------------------------+
|   TimeStamp(3 bytes)  |   MsgLength(3 bytes)  |   MsgTypeID(1 byte)  |
+----------------------------------------------------------------------+
|   MsgStreamID(4 bytes)|
+-----------------------+
```

##### 块类型1: 

- 7 bytes   省去了MsgStreamID 表示此chunk和上一次发送的chunk属于同一个流,message大小变化的流的第一个消息块之后的每一个消息的第一个块应该使用这种头

- type=1时Message Header占用7个字节，省去了表示msg stream id的4个字节，表示此chunk和上一次发的chunk所在的流相同，如果在发送端只和对端有一个流链接的时候可以尽量去采取这种格式。
timestamp delta：占用3个字节，注意这里和type＝0时不同，存储的是和上一个chunk的时间差。类似上面提到的timestamp，当它的值超过3个字节所能表示的最大值时，三个字节都置为1，实际的时间戳差值就会转存到Extended Timestamp字段中，接受端在判断timestamp delta字段24个位都为1时就会去Extended timestamp中解析时机的与上次时间戳的差值。

- 结构如下:
```
+----------------------------------------------------------------------+
| TimeStampDelta(3 bytes) |   MsgLength(3 bytes)  |MsgTypeID(1 byte)   |
+----------------------------------------------------------------------+
```

##### 块类型2:

- 3 bytes 表示此chunk和上一次发送的chunk的 MsgTypeID, MsgLength, MsgStreamID 相同,
message大小不变的流的第一条message之后的每条message的第一个chunk

type=2时Message Header占用3个字节，相对于type＝1格式又省去了表示消息长度的3个字节和表示消息类型的1个字节，表示此chunk和上一次发送的chunk所在的流、消息的长度和消息的类型都相同。余下的这三个字节表示timestamp delta，使用同type＝1

- 结构如下:
```
+---------------------------+
| TimeStampDelta(3 bytes)   |
+---------------------------+
```

##### 块类型3:
- Type = 3时只有0字节！！！好吧，它表示这个chunk的Message Header和上一个是完全相同的，自然就不用再传输一遍了。当它跟在Type＝0的chunk后面时，表示和前一个chunk的时间戳都是相同的。什么时候连时间戳都相同呢？就是一个Message拆分成了多个chunk，这个chunk和上一个chunk同属于一个Message。而当它跟在Type＝1或者Type＝2的chunk后面时，表示和前一个chunk的时间戳的差是相同的。比如第一个chunk的Type＝0，timestamp＝100，第二个chunk的Type＝2，timestamp delta＝20，表示时间戳为100+20=120，第三个chunk的Type＝3，表示timestamp delta＝20，时间戳为120+20=140
3.3.3 Extended Timestamp（扩展时间戳）：
上面我们提到在chunk中会有时间戳timestamp和时间戳差timestamp delta，并且它们不会同时存在，只有这两者之一大于3个字节能表示的最大数值0xFFFFFF＝16777215时，才会用这个字段来表示真正的时间戳，否则这个字段为0。扩展时间戳占4个字节，能表示的最大数值就是0xFFFFFFFF＝4294967295。当扩展时间戳启用时，timestamp字段或者timestamp delta要全置为1，表示应该去扩展时间戳字段来提取真正的时间戳或者时间戳差。注意扩展时间戳存储的是完整值，而不是减去时间戳或者时间戳差的值。
http://blog.csdn.net/win_lin/article/details/13363699 扩展时间戳详细说明。
### chunk介绍

- chunk示例1

```
+-------+------------+---------+-------+-----+
|       |mesgstreamid|MesTyoeId|time |leng   |
+-------+------------+---------+-------+-----+
| mesg1 |  123456    |   8     |1000 |32     |
+-------+------------+---------+-------+-----+
| mesg2 |  123456    |   8     |1020 |32     |
+-------+------------+---------+-------+-----+
| mesg3 |  123456    |   8     |1040 |32     |
+-------+------------+---------+-------+-----+
| mesg4 |  123456    |   8     |1060 |32     |
+-------+------------+---------+-------+-----+
```
- 首先包含第一个Message的chunk的Chunk Type为0，因为它没有前面可参考的chunk，timestamp为1000，表示时间戳。type为0的header占用11个字节，假定chunkstreamId为3<127，因此Basic Header占用1个字节，再加上Data的32个字节，因此第一个chunk共44＝11+1+32个字节。
第二个chunk和第一个chunk的CSID，TypeId，Data的长度都相同，因此采用Chunk Type＝2，timestamp delta＝1020-1000＝20，因此第二个chunk占用36=3+1+32个字节。
第三个chunk和第二个chunk的CSID，TypeId，Data的长度和时间戳差都相同，因此采用Chunk Type＝3省去全部Message Header的信息，占用33=1+32个字节。
第四个chunk和第三个chunk情况相同，也占用33=1+32个字节。
- 实际chunk如下：

```
+-------+-----+------+-----------------------+------------+----------+
|       |fmt  |csid  |ChunkMsgHeader         |chunkdata   |all len   |
+-------+-----|------+-----------------------+------------+----------+
|       |     |      |1000(3bytes)32(3bytes) |            |          |
|chunk1 |0    | 3    |12345(4byte)8(1byte)   |32          +1+11+32=44|
|       |     |      |all(3+3+4+1=11)        |            |          |
+-------+-----+------+-----------------------+------------+----------+
|chunk2 |2    | 3    |20(3bytes)             |32          |1+3+32=36 |
+-------+-----+------+-----------------------+------------+----------+
|chunk3 |3    | 3    |0                      |32          |32+1 = 33 |
+-------+-----+------+-----------------------+------------+----------+
|chunk4 |3    | 3    |0                      |32          |32+1 = 33 |
+-------+-----+------+-----------------------+------------+----------+
```
- 示例二：

```
+-------+------------+---------+-------+-----+
|       |mesgstreamid|MesTyoeId|time |leng   |
+-------+------------+---------+-----+-----+
| mesg1 |  123457    |   9     |1000 |307     |
+-------+------------+---------+-------+-----+
````

- 注意到Data的Length＝307>128,因此这个Message要切分成几个chunk发送，第一个chunk的Type＝0，Timestamp＝1000，承担128个字节的Data，因此共占用140=11+1+128个字节。
第二个chunk也要发送128个字节，其他字段也同第一个chunk，因此采用Chunk Type＝3，此时时间戳也为1000，共占用129=1+128个字节。
第三个chunk要发送的Data的长度为307-128-128=51个字节，还是采用Type＝3，共占用1+51＝52个字节,由于视频一个关键帧的pkt比较大因此常采用0-3这种模式发送。

- 实际chunk如下
```
+-------+-----+------+-----------------------+------------+------------+
|       |fmt  |csid  |ChunkMsgHeader         |chunkdata   |all len     |
+-------+-----|------+-----------------------+------------+------------+
|       |     |      |1000(3bytes)307(3bytes) |            |            |
|chunk1 |0    | 4    |12345(4byte)9(1byte)   |128         |1+11+128=140|
|       |     |      |all(3+3+4+1=11)        |            |            |
+-------+-----+------+-----------------------+------------+------------+
|chunk3 |3    | 4    |0                      |128         |129         |
+-------+-----+------+-----------------------+------------+------------+
|chunk4 |3    | 4    |0                      |51          |52          |
+-------+-----+------+-----------------------+------------+------------+
```
- rtmp协议结构图
![rtmp协议结构图 模块](http://chuantu.biz/t5/162/1502001486x2890149494.png)

## 协议控制消息（Protocol Control Message）

- 在RTMP的chunk流会用一些特殊的值来代表协议的控制消息，它们的Message Stream ID必须为0（代表控制流信息），CSID必须为2，Message Type ID可以为1，2，3，5，6，具体代表的消息会在下面依次说明。控制消息的接受端会忽略掉chunk中的时间戳，收到后立即生效。

### Set Chunk Size(Message Type ID=1):

- 设置chunk中Data字段所能承载的最大字节数，默认为128B，通信过程中可以通过发送该消息来设置chunk Size的大小（不得小于128B），而且通信双方会各自维护一个chunkSize，两端的chunkSize是独立的。比如当A想向B发送一个200B的Message，但默认的chunkSize是128B，因此就要将该消息拆分为Data分别为128B和72B的两个chunk发送，如果此时先发送一个设置chunkSize为256B的消息，再发送Data为200B的chunk，本地不再划分Message，B接受到Set Chunk Size的协议控制消息时会调整的接受的chunk的Data的大小，也不用再将两个chunk组成为一个Message.

以下为代表Set Chunk Size消息的chunk的Data：
 
```
+-------+-----|------+------------------------+------------+----------+
|       |fmt  |csid  |ChunkMsgHeader          |chunkdata   |all len   |
+-------+-----|------+------------------------+------------+----------+
|       |     |      |0000(时间戳3bytes)                               |
|       |     |      |4(chuckdatalen3bytes)   |            |          |
|chunk1 |0    | 2    |0(messagestreamid4byte) |            |1+11+4=16 |
|       |     |      |1(mesg type 1byte)      |256         |          |
|       |     |      |all(3+3+4+1=11)         |            |          |
+-------+-----|------+------------------------+------------+----------+
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   timestamp                   |message length | 
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     message length (cont)     |message type id| msg stream id |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           message stream id (cont)            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 
02（basehead） 
   00 00 00    |  
   00 00 04    | msg header
   01          |
   00 00 00 00 |
00 00 01 00-- body 4byte（256）


```
![Set Chunk Size 模块](http://chuantu.biz/t5/162/1502001644x2890149494.png)

- 其中第一位必须为0，chunk Size占31个位，最大可代表2147483647＝0x7FFFFFFF＝231-1，但实际上所有大于16777215=0xFFFFFF的值都用不上，因为chunk size不能大于Message的长度，表示Message的长度字段是用3个字节表示的，最大只能为0xFFFFFF。

### Abort Message(Message Type ID=2)
- 当一个Message被切分为多个chunk，接受端只接收到了部分chunk时，发送该控制消息表示发送端不再传输同Message的chunk，接受端接收到这个消息后要丢弃这些不完整的chunk。Data数据中只需要一个CSID，表示丢弃该CSID的所有已接收到的chunk。

```
+-------+-----|------+------------------------+------------+----------+
|       |fmt  |csid  |ChunkMsgHeader          |chunkdata   |all len   |
+-------+-----|------+------------------------+------------+----------+
|       |     |      |0000(时间戳3bytes)                               |
|       |     |      |4(chuckdatalen3bytes)   |            |          |
|chunk1 |0    | 2    |0(messagestreamid4byte) |            |1+11+4=16 |
|       |     |      |2(mesg type 1byte)      |04          |          |
|       |     |      |all(3+3+4+1=11)         |            |          |
+-------+-----|------+------------------------+------------+----------+
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   timestamp                   |message length | 
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     message length (cont)     |message type id| msg stream id |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           message stream id (cont)            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
02（basehead） 
   00 00 00    |  
   00 00 04    | msg header
   02          |
   00 00 00 00 |
00 00 00 04 -- body 4byte（4 丢弃csid为4的trunk的所有消息）
```
### Acknowledgement(Message Type ID=3)

- 当收到对端的消息大小等于窗口大小（Window Size）时接受端要回馈一个ACK给发送端告知对方可以继续发送数据。窗口大小就是指收到接受端返回的ACK前最多可以发送的字节数量，返回的ACK中会带有从发送上一个ACK后接收到的字节数。

```
+-------+-----|------+------------------------+------------+----------+
|       |fmt  |csid  |ChunkMsgHeader          |chunkdata   |all len   |
+-------+-----|------+------------------------+------------+----------+
|       |     |      |0000(时间戳3bytes)                               |
|       |     |      |4(chuckdatalen3bytes)   |            |          |
|chunk1 |0    | 2    |0(messagestreamid4byte) |            |1+11+4=16 |
|       |     |      |3(mesg type 1byte)      |256         |          |
|       |     |      |all(3+3+4+1=11)         |            |          |
+-------+-----|------+------------------------+------------+----------+
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   timestamp                   |message length | 
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     message length (cont)     |message type id| msg stream id |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           message stream id (cont)            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 
02（basehead） 
   00 00 00    |  
   00 00 04    | msg header
   03          |
   00 00 00 00 |
00 4C 4B 41 -- body 4byte（5000001 ack）
```
### Window Acknowledgement Size(Message Type ID=5)
- 发送端在接收到接受端返回的两个ACK间最多可以发送的字节数。
 
```
+-------+-----|------+------------------------+------------+----------+
|       |fmt  |csid  |ChunkMsgHeader          |chunkdata   |all len   |
+-------+-----|------+------------------------+------------+----------+
|       |     |      |0000(时间戳3bytes)                               |
|       |     |      |4(chuckdatalen3bytes)   |            |          |
|chunk1 |0    | 2    |0(messagestreamid4byte) |            |1+11+4=16 |
|       |     |      |5(mesg type 1byte)      |256         |          |
|       |     |      |all(3+3+4+1=11)         |            |          |
+-------+-----|------+------------------------+------------+----------+
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   timestamp                   |message length | 
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     message length (cont)     |message type id| msg stream id |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           message stream id (cont)            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
02（basehead） 
   00 00 00    |  
   00 00 04    | msg header
   05          |
   00 00 00 00 |
00 4C 4B 41 -- body 4byte（5000001 ack）
```

![Set Chunk Size 模块](http://chuantu.biz/t5/162/1502002029x2890149494.png)

### Set Peer Bandwidth(Message Type ID=6)
- 限制对端的输出带宽。接受端接收到该消息后会通过设置消息中的Window ACK Size来限制已发送但未接受到反馈的消息的大小来限制发送端的发送带宽。如果消息中的Window ACK Size与上一次发送给发送端的size不同的话要回馈一个Window Acknowledgement Size的控制消息。
- Hard(Limit Type＝0):接受端应该将Window Ack Size设置为消息中的值
- Soft(Limit Type=1):接受端可以讲Window Ack Size设为消息中的值，也可以保存原来的值（前提是原来的Size小与该控制消息中的Window Ack Size）
- Dynamic(Limit Type=2):如果上次的Set Peer Bandwidth消息中的Limit Type为0，本次也按Hard处理，否则忽略本消息，不去设置Window Ack Size。

![Set Peer Bandwidth 模块](http://chuantu.biz/t5/162/1502002997x2890149494.png)

## User Control Message（用户控制协议）：

```
它的 Type ID 只能为 4。它主要是发送一些对视频的控制信息。其发送的条件也有一定的限制：
msg stream ID 为 0
chunk stream ID 为 2
它的 Body 部分的基本格式为：
+----------------+----------+
|EventType(16bit)|EventData |
+----------------+----------+

 Event Type 的不同，对流进行不同的设置。它的 Event Type 一共有 6 种格式 Stream Begin(0)，Stream EOF(1)，StreamDry(2)，SetBuffer Length(3)，StreamIs Recorded(4)，PingRequest(6)，PingResponse(7)。
```
### Stream Begin: Event Type  0
- 当客户端和服务端成功 connect 后发送。Event Data 为 4B，内容是已经可以正式用来传输数据的 Stream ID

![Stream Begin](http://chuantu.biz/t5/162/1502006120x1929325700.png)

### Stream EOF: Event Type  1
它常常出现在，当音视频流已经全部传输完时。 Event Data 为 4byte，用来表示已经发送完音视频流的 Stream ID

## Command Msg
- Command Msg 里面的内容，其 type id 涵盖了 8~22 之间的值。具体内容，可以参考下表

![Stream Begin](http://chuantu.biz/t5/162/1502007182x1929325700.png)
### Video Msg
- rtmp 内部使用的格式是flv格式的。video Msg 的streamtypeid为9.

![vdieo msg](http://chuantu.biz/t5/162/1502007944x2890149494.png)
- 如上图 
- 这是 FLV Video 的协议格式 rtmp 在解析FrameType 的时候，我们只需要支持 1/2 即可。因为，视频中最重要的是 I 帧，它对应的 FrameType 就是 1。而 B/P 则是剩下的 2。我们只要针对 1/2 进行软解，即可实现视频所有信息的获取。所以，在 RTMP 中，也主要（或者大部分）都是传输上面两种 FrameType。我们通过实际抓包来讲解一下。

这是 KeyFrame 的包，注意 Buffer 开头的 17 数字。大家可以找到上面的 FrameType 对应找一找，看结果是不是一致的：
![vdieo msg](http://chuantu.biz/t5/162/1502008632x1929325700.png)
-- sps pps 解析
![vdieo msg](http://chuantu.biz/t5/162/1502014308x1929325700.png)
![vdieo msg](http://chuantu.biz/t5/162/1502014437x1929325700.png)


### Audio Tag
![vdieo msg](http://chuantu.biz/t5/162/1502008738x1929325700.png)
- 如上图 为音频格式
- ![Audio msg](http://chuantu.biz/t5/162/1502008846x1929325700.png)
- AudioConf
-  ![Audio msg](http://chuantu.biz/t5/162/1502014549x1929325700.png)
## Command Msg

- Command Msg 是 RTMP 里面的一个主要信息传递工具。常常用在 RTMP 前期和后期处理。Command Msg 是通过 AMF 的格式进行传输的（其实就是类似 JSON 的二进制编码规则）。Command Msg 主要分为 net connect 和 net stream 两大块。它的交流方式是双向的，即，你发送一次 net connect 或者 stream 之后，另外一端都必须返回一个 _result 或者 _error 以表示收到信息。
- 详细结构可以参考下图：
-  ![cmmand msg](http://chuantu.biz/t5/162/1502014884x1929325700.png)

### netConnection

- netConnection 可以分为 4 种 Msg，connect，call，createStream，close。
#### connect
- connect 是客户端向 Server 端发送播放请求的。里面的字段内容如下
```
1.Command Name[String]: 默认为 connect。表示信息名称
2.Transaction ID[Number]: 默认为 1。
3.Command Object: 键值对的形式存放相关信息。
4.Optional: 可选值。一般没有
Command Object 信息：
1.app[String]: 服务端连接应用的名字。这个主要根据你的 RTMP 服务器设定来设置。比如：live。
2.flashver[String]: Flash Player 的版本号。一般根据自己设备上的型号来确定即可。也可以设置为默认值：LNX 9,0,124,2。
3.tcUrl[String]: 服务端的 URL 地址。简单来说，就是 protocol://host/path。比如：rtmp://6521.liveplay.myqcloud.com/live。

4：fpad[Boolean]: 表示是否使用代理。一般为 false。
5：audioCodecs[Number]: 客户端支持的音频解码。后续会介绍。默认可以设置为 4071
6：videoCodecs[Number]: 客户端支持的视频解码。有自定义的标准。默认可以设置为 252
7：videoFunction[Number]: 表明在服务端上调用那种特别的视频函数。默认可以设置为 1
```
 ![cmmand msg](http://chuantu.biz/t5/162/1502015314x1929325700.png)
- connect响应
```
1.Command Name[String]: 为 _result 或者 _error。
2.Transaction ID[Number]: 默认为 1。
3.Command Object: 键值对的形式存放相关信息。
4.Information[Object]: 键值对的形式，来描述相关的 response 信息。里面存在的字段有：level,code,description
```
 ![cmmand msg](http://chuantu.biz/t5/162/1502015485x1929325700.png)
#### createStream

| 字段    |类型     | 说明|
| :----- | ------ | --- |
| commandName（命令名） | string    |  “createStream”      |
| TransactionID | numer |  是       |
| Command Object | Object |  命令参数       |
| Optional Arguments | Object | 用户自定义参数       |
- 例子说明：
 ![cmmand msg](http://chuantu.biz/t5/162/1502016953x1929325700.png)
---
### NetStream Commands(流连接上的命令)

- Netstream建立在NetConnection之上，通过NetConnection的createStream命令创建，用于传输具体的音频、视频等信息。在传输层协议之上只能连接一个NetConnection，但一个NetConnection可以建立多个NetStream来建立不同的流通道传输数据。
以下会列出一些常用的NetStream Commands，服务端收到命令后会通过onStatus的命令来响应客户端，表示当前NetStream的状态。

#### onStatus命令的消息结构如下：
- 
| 字段    |类型     | 说明|
| :----- | ------ | --- |
| commandName（命令名） | string    |  “createStream”      |
| TransactionID | numer |  恒为0      |
| Command Object | NUll |  onstatus 命令不需要      |
| info Object | Object | AMF类型的Object，至少包含以下三个属性：1，“level”，String类型，可以为“warning”、”status”、”error”中的一种；2，”code”,String类型，代表具体状态的关键字,比如”NetStream.Play.Start”表示开始播流；3，”description”，String类型，代表对当前状态的描述，提供对当前状态可读性更好的解释，除了这三种必要信息，用户还可以自己增加自定义的键值对       |

- 例子如下：
  ![cmmand msg](http://chuantu.biz/t5/162/1502017303x1929325700.png)

#### play：

| 字段    |类型     | 说明|
| :----- | ------ | --- |
| commandName（命令名） | string    |  “play”      |
| TransactionID | numer |  恒为0       |
|命令参数对象 | Null |  不需要此字段，设为空     |
| 流名称 | string |  要播放的流的名称     |
|开始位置 | numer |  可选参数，表示从何时开始播流，以秒为单位。默认为－2，代表选取对应该流名称的直播流，即当前正在推送的流开始播放，如果对应该名称的直播流不存在，就选取该名称的流的录播版本，如果这也没有，当前播流端要等待直到对端开始该名称的流的直播。如果传值－1，那么只会选取直播流进行播放，即使有录播流也不会播放；如果传值或者正数，就代表从该流的该时间点开始播放，如果流不存在的话就会自动播放播放列表中的下一个流     |
| 周期 | numer |  可选参数，表示回退的最小间隔单位，以秒为单位计数。默认值为－1，代表直到直播流不再可用或者录播流停止后才能回退播放；如果传值为0，代表从当前帧开始播放       |
|重置 |Boolean | 可选参数，true代表清除之前的流，重新开始一路播放，false代表保留原来的流，向本地的播放列表中再添加一条播放流       |
- 例子

![cmmand msg](http://chuantu.biz/t5/162/1502017894x1929325700.png)

#### deleteStream(删除流)
- 结构如下

| 字段    |类型     | 说明|
| :----- | ------ | --- |
| commandName（命令名） | string    |  deleteStream”      |
| TransactionID | numer |  恒为0       |
| Command Object | Object |  NULL,对deleteStream命令来说不需要这个字段      |
|Stream ID（流ID)| Number | 本地已删除，不再需要服务器传输的流的ID       |
- 例子：
![cmmand msg](http://chuantu.biz/t5/162/1502018385x1929325700.png)

####  publish(推送数据)
- 由客户端向服务器发起请求推流到服务器
![cmmand msg](http://chuantu.biz/t5/162/1502018575x1929325700.png)

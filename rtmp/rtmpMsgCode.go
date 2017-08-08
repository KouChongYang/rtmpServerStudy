package rtmp

const  (
	RtmpMsgChunkSize =  1
	RtmpMsgAbort = 2
	RtmpMsgAck = 3
	RtmpMsgUser = 4
	RtmpMsgAckSize = 5
	RtmpMsgBandwidth = 6
	RtmpMsgEdge = 7
	RtmpMsgAudio = 8
	RtmpMsgVideo = 9
	RtmpMsgAmf3Meta = 15
	RtmpMsgAmf3Shared = 16
	RtmpMsgAmf3CMD = 17
	RtmpMsgAmfMeta = 18
	RtmpMsgAmfShared = 19
	RtmpMsgAmfCMD = 20
	RrmpMsgAggregate = 22
	NGX_RTMP_MSG_MAX =  22
)

/*#define NGX_RTMP_CONNECT                NGX_RTMP_MSG_MAX + 1
#define NGX_RTMP_DISCONNECT             NGX_RTMP_MSG_MAX + 2
#define NGX_RTMP_HANDSHAKE_DONE         NGX_RTMP_MSG_MAX + 3
#define NGX_RTMP_CONNECT_DONE           NGX_RTMP_MSG_MAX + 4
#define NGX_RTMP_PLAY_DONE              NGX_RTMP_MSG_MAX + 5
#define NGX_RTMP_NOTIFY_LATENCY         NGX_RTMP_MSG_MAX + 6
#define NGX_RTMP_ON_MESSAGE             NGX_RTMP_MSG_MAX + 7
#define NGX_RTMP_MAX_EVENT              NGX_RTMP_MSG_MAX + 8
*/

/* RMTP control message types */
const(
	RtmpUserStreamBegin = 0
	RtmpUserStreamEof = 1
	RtmpUserStreamDRY = 2
	RtmpUserSetBufLen  = 3
	RtmpUserRecorded = 4
	RtmpUserPingRequest = 6
	RtmpUserPingResponse = 7
	RtmpUserUnknown = 8
	RtmpUserBufferEnd = 32
)

type Handler func (sesion *Session,timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error)
type RtmpMsgHandles map[int]Handler
var RtmpMsgHandles  RtmpMsgHandles
type RtmpControlMsgHandles map[int]Handler
var RtmpControlMsgHandles RtmpControlMsgHandles

func init(){
	RtmpControlMsgHandles = make(RtmpControlMsgHandles)
	RtmpMsgHandles = make(RtmpMsgHandles)
	RtmpMsgHandles[RtmpMsgChunkSize] = RtmpMsgChunkSizeHandler
	RtmpMsgHandles[RtmpMsgAbort] = RtmpMsgAbortHandler
	RtmpMsgHandles[RtmpMsgAck] = RtmpMsgAckHanldler
	RtmpMsgHandles[RtmpMsgUser] = RtmpMsgUserEventHandler
	RtmpMsgHandles[RtmpMsgAckSize] = RtmpMsgAckSizeHandler
	RtmpMsgHandles[RtmpMsgBandwidth] = RtmpMsgBandwidthHandler
	RtmpMsgHandles[RtmpMsgEdge] = RtmpMsgEdgeHandler
	RtmpMsgHandles[RtmpMsgAudio] = RtmpMsgAudioHandler
	RtmpMsgHandles[RtmpMsgVideo] = RtmpMsgVideoHandler
	RtmpMsgHandles[RtmpMsgAmf3Meta] = RtmpMsgAmf3MetaHandler
	RtmpMsgHandles[RtmpMsgAmf3Shared] = RtmpMsgAmf3SharedHandler
	RtmpMsgHandles[RtmpMsgAmf3CMD] = RtmpMsgAmf3CMDHandler
	RtmpMsgHandles[RtmpMsgAmfMeta] = RtmpMsgAmfMetaHandler
	RtmpMsgHandles[RtmpMsgAmfShared] =RtmpMsgAmfSharedHandler
	RtmpMsgHandles[RtmpMsgAmfCMD] = RtmpMsgAmfCMDHandler
	RtmpMsgHandles[RrmpMsgAggregate] = RrmpMsgAggregateHandler


	RtmpControlMsgHandles[RtmpUserStreamBegin] = RtmpUserStreamBeginHandler
	RtmpControlMsgHandles[RtmpUserStreamEof] = RtmpUserStreamEofHandler
	RtmpControlMsgHandles[RtmpUserStreamDRY] =RtmpUserStreamDRYHandler
	RtmpControlMsgHandles[RtmpUserSetBufLen]  = RtmpUserSetBufLenHandler
	RtmpControlMsgHandles[RtmpUserRecorded] = RtmpUserRecordedHandler
	RtmpControlMsgHandles[RtmpUserPingRequest] = RtmpUserPingRequestHandler
	RtmpControlMsgHandles[RtmpUserPingResponse] = RtmpUserPingResponseHandler
	RtmpControlMsgHandles[RtmpUserUnknown] = RtmpUserUnknownHandler
}
package rtmp

import (
	"fmt"
	"rtmpServerStudy/amf"
	"context"
	"rtmpServerStudy/AvQue"
)

/*
   { ngx_string("connect"),            ngx_rtmp_cmd_connect_init           },
   { ngx_string("createStream"),       ngx_rtmp_cmd_create_stream_init     },
   { ngx_string("closeStream"),        ngx_rtmp_cmd_close_stream_init      },
   { ngx_string("deleteStream"),       ngx_rtmp_cmd_delete_stream_init     },
   { ngx_string("publish"),            ngx_rtmp_cmd_publish_init           },
   { ngx_string("play"),               ngx_rtmp_cmd_play_init              },
   { ngx_string("play2"),              ngx_rtmp_cmd_play2_init             },
   { ngx_string("seek"),               ngx_rtmp_cmd_seek_init              },
   { ngx_string("pause"),              ngx_rtmp_cmd_pause_init             },
   { ngx_string("pauseraw"),           ngx_rtmp_cmd_pause_init
*/

type cmdHandler func(sesion *Session, b []byte) (n int, err error)
type RtmpCmdHandle map[string]cmdHandler

var RtmpCmdHandles RtmpCmdHandle

func RtmpConnectCmdHandler(session *Session, b []byte) (n int, err error) {
	var transid, obj interface{}
	var size int
	if transid, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
		return
	}
	n += size
	if obj, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
		return
	}
	n += size

	session.commandtransid, _ = transid.(float64)
	commandobj, _ := obj.(amf.AMFMap)
	commandparams := []interface{}{}

	for n < len(b) {
		if obj, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
			return
		}
		n += size
		commandparams = append(commandparams, obj)
	}
	if n < len(b) {
		err = fmt.Errorf("rtmp: CommandMsgAMF0 left bytes=%d", len(b)-n)
		return
	}

	if commandobj == nil {
		err = fmt.Errorf("rtmp: connect command params invalid")
		return
	}

	var ok bool
	var _app, _tcurl interface{}
	if _app, ok = commandobj["app"]; !ok {
		err = fmt.Errorf("rtmp: `connect` params missing `app`")
		return
	}
	app, _ := _app.(string)
	session.App = &app

	var tcurl string
	if _tcurl, ok = commandobj["tcUrl"]; !ok {
	}
	if ok {
		tcurl, _ = _tcurl.(string)
	}
	session.TcUrl = &tcurl

	if err = session.writeBasicConf(); err != nil {
		return
	}

	// > _result("NetConnection.Connect.Success")
	if err = session.writeCommandMsg(3, 0, "_result", session.commandtransid,
		amf.AMFMap{
			"fmtVer":       "FMS/3,0,1,123",
			"capabilities": 31,
		},
		amf.AMFMap{
			"level":          "status",
			"code":           "NetConnection.Connect.Success",
			"description":    "Connection succeeded.",
			"objectEncoding": 3,
		},
	); err != nil {
		return
	}

	if err = session.flushWrite(); err != nil {
		return
	}
	return
}

func RtmpCreateStreamCmdHandler(session *Session, b []byte) (n int, err error) {

	session.avmsgsid = uint32(1)
	var transid interface{}

	if transid, _, err = amf.ParseAMF0Val(b[n:]); err != nil {
		return
	}
	session.commandtransid, _ = transid.(float64)
	// > _result(streamid)
	if err = session.writeCommandMsg(3, 0, "_result", session.commandtransid, nil, session.avmsgsid); err != nil {
		return
	}
	if err = session.flushWrite(); err != nil {
		return
	}
	return
}

func RtmpCloseStreamCmdHandler(sesion *Session, b []byte) (n int, err error) {
	return
}

func RtmpDeleteStreamCmdHandler(sesion *Session, b []byte) (n int, err error) {
	err = fmt.Errorf("rtmp: delateStream")
	return
}

func RtmpPublishCmdHandler(session *Session, b []byte) (n int, err error) {
	if Debug {
		fmt.Println("rtmp: < publish")
	}
	var transid, obj interface{}
	var size int
	if transid, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
		return
	}
	n += size
	if obj, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
		return
	}
	n += size

	session.commandtransid, _ = transid.(float64)
	commandparams := []interface{}{}

	for n < len(b) {
		if obj, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
			return
		}
		n += size
		commandparams = append(commandparams, obj)
	}
	if n < len(b) {
		err = fmt.Errorf("rtmp: CommandMsgAMF0 left bytes=%d", len(b)-n)
		return
	}

	if len(commandparams) < 1 {
		err = fmt.Errorf("rtmp: publish params invalid")
		return
	}
	publishpath, _ := commandparams[0].(string)
	fmt.Println(publishpath)

	// here must do something
	/*if session.OnPlayOrPublish != nil {
		cberr = self.OnPlayOrPublish("publish", commandparams)
	}*/

	var code , level,desc string
	session.URL = createURL(*session.TcUrl, *session.App, publishpath)
	session.context, session.cancel = context.WithCancel(context.Background())
	session.GopCache = AvQue.RingBufferCreate(8)
	ok := RtmpSessionPush(session)
	if !ok {
		code ,level,desc = "NetStream.Publish.BadName","status","Already publishing"
	}else {
		code ,level,desc = "NetStream.Publish.Start","status","Start publishing"
		//play register channel
		session.RegisterChannel = make(chan *Session, MAXREGISTERCHANNEL)
	}

	if err = session.writeRtmpStatus(code , level,desc);err != nil{
		return
	}
	if err = session.flushWrite(); err != nil {
		return
	}
	session.publishing = true

	if FlvRecord == true {
	}
	if HlsRecord == true {
	}

	session.stage = stageCommandDone
	return
}

func RtmpPlayCmdHandler(session *Session, b []byte) (n int, err error) {
	if Debug {
		fmt.Println("rtmp: < play")
	}
	var transid, obj interface{}
	var size int
	if transid, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
		return
	}
	n += size
	if obj, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
		return
	}
	n += size

	session.commandtransid, _ = transid.(float64)
	commandparams := []interface{}{}

	for n < len(b) {
		if obj, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
			return
		}
		n += size
		commandparams = append(commandparams, obj)
	}
	if n < len(b) {
		err = fmt.Errorf("rtmp: CommandMsgAMF0 left bytes=%d", len(b)-n)
		return
	}

	if len(commandparams) < 1 {
		err = fmt.Errorf("rtmp: publish params invalid")
		return
	}
	if len(commandparams) < 1 {
		err = fmt.Errorf("rtmp: command play params invalid")
		return
	}
	playpath, _ := commandparams[0].(string)
	fmt.Println(playpath)
	// > onStatus()
	if err = session.writeRtmpStatus("NetStream.Play.Start" , "status","Start live");err != nil{
		return
	}
	// > streamBegin(streamid)
	if err = session.writeStreamBegin(session.avmsgsid); err != nil {
		return
	}

	// > |RtmpSampleAccess()
	if err = session.writeDataMsg(5, session.avmsgsid,
		"|RtmpSampleAccess", true, true,
	); err != nil {
		return
	}
	if err = session.flushWrite(); err != nil {
		return
	}
	session.URL = createURL(*session.TcUrl, *session.App, playpath)
	session.playing = true
	session.stage = stageCommandDone
	return
}

func init() {
	RtmpCmdHandles = make(RtmpCmdHandle)
	RtmpCmdHandles["connect"] = RtmpConnectCmdHandler
	RtmpCmdHandles["createStream"] = RtmpCreateStreamCmdHandler
	RtmpCmdHandles["closeStream"] = RtmpCloseStreamCmdHandler
	RtmpCmdHandles["deleteStream"] = RtmpDeleteStreamCmdHandler
	RtmpCmdHandles["publish"] = RtmpPublishCmdHandler
	RtmpCmdHandles["play"] = RtmpPlayCmdHandler
}

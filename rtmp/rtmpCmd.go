package rtmp

import (
	"fmt"
	"rtmpServerStudy/amf"
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

type cmdHandler func (sesion *Session,obj amf.AMFMap,amfParams[]interface{}) (err error)
type RtmpCmdHandle map[string]cmdHandler

var RtmpCmdHandles RtmpCmdHandle

func RtmpConnectCmdHandler(Session *Session,CmdObj amf.AMFMap,amfParams[]interface{})(err error){

	if CmdObj == nil {
		err = fmt.Errorf("rtmp: connect command params invalid")
		return
	}

	var ok bool
	var _app, _tcurl interface{}
	if _app, ok = CmdObj["app"]; !ok {
		err = fmt.Errorf("rtmp: `connect` params missing `app`")
		return
	}
	app, _ := _app.(string)
	Session.App = &app

	var tcurl string
	if _tcurl, ok = CmdObj["tcUrl"]; !ok {
	}
	if ok {
		tcurl, _ = _tcurl.(string)
	}
	Session.TcUrl = &tcurl

	if err = Session.writeBasicConf(); err != nil {
		return
	}

	// > _result("NetConnection.Connect.Success")
	if err = Session.writeCommandMsg(3, 0, "_result", Session.commandtransid,
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

	if err = Session.flushWrite(); err != nil {
		return
	}
	return err
}

func RtmpCreateStreamCmdHandler(session *Session,obj amf.AMFMap,amfParams[]interface{})(err error){

	session.avmsgsid = uint32(1)
	// > _result(streamid)
	if err = session.writeCommandMsg(3, 0, "_result", session.commandtransid, nil, session.avmsgsid); err != nil {
		return
	}
	if err = session.flushWrite(); err != nil {
		return
	}
	return err
}

func RtmpCloseStreamCmdHandler(sesion *Session,obj amf.AMFMap,amfParams[]interface{})(err error){
	return err
}

func RtmpDeleteStreamCmdHandler(sesion *Session,obj amf.AMFMap,amfParams[]interface{})(err error){
	err = fmt.Errorf("rtmp: delateStream")
	return err
}

func RtmpPublishCmdHandler(session *Session,obj amf.AMFMap,amfParams[]interface{})(err error){
	if Debug {
		fmt.Println("rtmp: < publish")
	}

	if len(amfParams) < 1 {
		err = fmt.Errorf("rtmp: publish params invalid")
		return
	}
	publishpath, _ := amfParams[0].(string)
	fmt.Println(publishpath)
	var cberr error
	// here must do something
	/*if session.OnPlayOrPublish != nil {
		cberr = self.OnPlayOrPublish("publish", amfParams)
	}*/

	// > onStatus()
	if err = session.writeCommandMsg(5, session.avmsgsid,
		"onStatus", session.commandtransid, nil,
		amf.AMFMap{
			"level":       "status",
			"code":        "NetStream.Publish.Start",
			"description": "Start publishing",
		},
	); err != nil {
		return
	}
	if err = session.flushWrite(); err != nil {
		return
	}

	if cberr != nil {
		err = fmt.Errorf("rtmp: OnPlayOrPublish check failed")
		return
	}
	session.URL = createURL(*session.TcUrl, *session.App, publishpath)
	session.publishing = true
	session.stage = stageCommandDone
	return
}

func RtmpPlayCmdHandler(session *Session,obj amf.AMFMap,amfParams[]interface{})(err error){
	if Debug {
		fmt.Println("rtmp: < play")
	}

	if len(amfParams) < 1 {
		err = fmt.Errorf("rtmp: command play params invalid")
		return
	}
	playpath, _ := amfParams[0].(string)
	fmt.Println(playpath)
	// > streamBegin(streamid)
	if err = session.writeStreamBegin(session.avmsgsid); err != nil {
		return
	}

	// > onStatus()
	if err = session.writeCommandMsg(5, session.avmsgsid,
		"onStatus", session.commandtransid, nil,
		amf.AMFMap{
			"level":       "status",
			"code":        "NetStream.Play.Start",
			"description": "Start live",
		},
	); err != nil {
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


func init(){
	RtmpCmdHandles = make(RtmpCmdHandle)
	RtmpCmdHandles["connect"] = RtmpConnectCmdHandler
	RtmpCmdHandles["createStream"] = RtmpCreateStreamCmdHandler
	RtmpCmdHandles["closeStream"] = RtmpCloseStreamCmdHandler
	RtmpCmdHandles["deleteStream"] = RtmpDeleteStreamCmdHandler
	RtmpCmdHandles["publish"] = RtmpPublishCmdHandler
	RtmpCmdHandles["play"] = RtmpPlayCmdHandler
}

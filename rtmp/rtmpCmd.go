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

type Handler func (sesion *Session,obj amf.AMFMap,amfParams[]interface{}) (err error)
type RtmpCmdHandles map[string]Handler
var RtmpCmdHandles RtmpCmdHandles

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
	if err = Session.writeCommandMsg(3, 0, "_result", self.commandtransid,
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

	if err = self.flushWrite(); err != nil {
		return
	}
	return err
}

func RtmpCreateStreamCmdHandler(sesion *Session,obj amf.AMFMap,amfParams[]interface{})(err error){
	return err
}

func RtmpCloseStreamCmdHandler(sesion *Session,obj amf.AMFMap,amfParams[]interface{})(err error){
	return err
}

func RtmpDeleteStreamCmdHandler(sesion *Session,obj amf.AMFMap,amfParams[]interface{})(err error){
	return err
}

func RtmpPublishCmdHandler(sesion *Session,obj amf.AMFMap,amfParams[]interface{})(err error){
	return err
}

func RtmpPlayCmdHandler(sesion *Session,obj amf.AMFMap,amfParams[]interface{})(err error){
	return err
}


func init(){
	RtmpCmdHandles = make(RtmpCmdHandles)
	RtmpCmdHandles["connect"] = RtmpConnectCmdHandler
	RtmpCmdHandles["createStream"] = RtmpCreateStreamCmdHandler
	RtmpCmdHandles["closeStream"] = RtmpCloseStreamCmdHandler
	RtmpCmdHandles["deleteStream"] = RtmpDeleteStreamCmdHandler
	RtmpCmdHandles["publish"] = RtmpPublishCmdHandler
	RtmpCmdHandles["play"] = RtmpPlayCmdHandler
}





func (self *Session) readConnect() (err error) {
	var connectpath string

	// < connect("app")
	if err = self.pollCommand(); err != nil {
		return
	}
	if self.commandname != "connect" {
		err = fmt.Errorf("rtmp: first command is not connect")
		return
	}
	if self.commandobj == nil {
		err = fmt.Errorf("rtmp: connect command params invalid")
		return
	}

	var ok bool
	var _app, _tcurl interface{}
	if _app, ok = self.commandobj["app"]; !ok {
		err = fmt.Errorf("rtmp: `connect` params missing `app`")
		return
	}
	connectpath, _ = _app.(string)

	var tcurl string
	if _tcurl, ok = self.commandobj["tcUrl"]; !ok {
		_tcurl, ok = self.commandobj["tcurl"]
	}
	if ok {
		tcurl, _ = _tcurl.(string)
	}
	connectparams := self.commandobj

	if err = self.writeBasicConf(); err != nil {
		return
	}

	// > _result("NetConnection.Connect.Success")
	if err = self.writeCommandMsg(3, 0, "_result", self.commandtransid,
		flvio.AMFMap{
			"fmtVer":       "FMS/3,0,1,123",
			"capabilities": 31,
		},
		flvio.AMFMap{
			"level":          "status",
			"code":           "NetConnection.Connect.Success",
			"description":    "Connection succeeded.",
			"objectEncoding": 3,
		},
	); err != nil {
		return
	}

	if err = self.flushWrite(); err != nil {
		return
	}

	for {
		if err = self.pollMsg(); err != nil {
			return
		}
		if self.gotcommand {
			switch self.commandname {

			// < createStream
			case "createStream":
				self.avmsgsid = uint32(1)
				// > _result(streamid)
				if err = self.writeCommandMsg(3, 0, "_result", self.commandtransid, nil, self.avmsgsid); err != nil {
					return
				}
				if err = self.flushWrite(); err != nil {
					return
				}

			// < publish("path")
			case "publish":
				if Debug {
					fmt.Println("rtmp: < publish")
				}

				if len(self.commandparams) < 1 {
					err = fmt.Errorf("rtmp: publish params invalid")
					return
				}
				publishpath, _ := self.commandparams[0].(string)

				var cberr error
				if self.OnPlayOrPublish != nil {
					cberr = self.OnPlayOrPublish(self.commandname, connectparams)
				}

				// > onStatus()
				if err = self.writeCommandMsg(5, self.avmsgsid,
					"onStatus", self.commandtransid, nil,
					flvio.AMFMap{
						"level":       "status",
						"code":        "NetStream.Publish.Start",
						"description": "Start publishing",
					},
				); err != nil {
					return
				}
				if err = self.flushWrite(); err != nil {
					return
				}

				if cberr != nil {
					err = fmt.Errorf("rtmp: OnPlayOrPublish check failed")
					return
				}

				self.URL = createURL(tcurl, connectpath, publishpath)
				self.publishing = true
				self.reading = true
				self.stage++
				return

			// < play("path")
			case "play":
				if Debug {
					fmt.Println("rtmp: < play")
				}

				if len(self.commandparams) < 1 {
					err = fmt.Errorf("rtmp: command play params invalid")
					return
				}
				playpath, _ := self.commandparams[0].(string)

				// > streamBegin(streamid)
				if err = self.writeStreamBegin(self.avmsgsid); err != nil {
					return
				}

				// > onStatus()
				if err = self.writeCommandMsg(5, self.avmsgsid,
					"onStatus", self.commandtransid, nil,
					flvio.AMFMap{
						"level":       "status",
						"code":        "NetStream.Play.Start",
						"description": "Start live",
					},
				); err != nil {
					return
				}

				// > |RtmpSampleAccess()
				if err = self.writeDataMsg(5, self.avmsgsid,
					"|RtmpSampleAccess", true, true,
				); err != nil {
					return
				}

				if err = self.flushWrite(); err != nil {
					return
				}

				self.URL = createURL(tcurl, connectpath, playpath)
				self.playing = true
				self.writing = true
				self.stage++
				return
			case "onMetaData":
				fmt.Println(self.commandparams)
				return
			}

		}
	}

	return
}
package rtmp

import (
	"fmt"
	"rtmpServerStudy/amf"
	"context"
	"rtmpServerStudy/AvQue"
	"net/url"

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

func (self *Session)RtmpcheckHost(host string,cmd string)(err error){

	code,level,desc:="","",""
	switch cmd {
	case "connect":
		pubOk:=false
		PlayOk:=false
		_,PlayOk=Gconfig.UserConf.PlayDomain[host]
		_,pubOk=Gconfig.UserConf.PublishDomain[host]
		if (!PlayOk) && (!pubOk){
			code ,level,desc = "NetStream.Connect.IllegalDomain","status","Illegal domain"
			err = fmt.Errorf("NetStream.Connect.IllegalDomain")
		}
	case "publish":
		_,pubOk:=Gconfig.UserConf.PublishDomain[host]
		if (!pubOk){
			code ,level,desc = "NetStream.Publish.IllegalDomain","status","Illegal publish domain"
			err = fmt.Errorf("NetStream.Publish.IllegalDomain")
		}
	case "play":
		_,pubOk:=Gconfig.UserConf.PlayDomain[host]
		if (!pubOk){
			code ,level,desc = "NetStream.Play.IllegalDomain","status","Illegal play domain"
			err = fmt.Errorf("NetStream.Play.IllegalDomain")
		}
	}
	errBak := err
	if err != nil {
		if err = self.writeRtmpStatus(code, level, desc); err != nil {
			return
		}
		self.flushWrite()
		err = errBak
	}
	return
}

func (self *Session)RtmpChckeApp(host ,app string)(err error){

	code,level,desc:="","",""
	pubOk:=false
	PlayOk:=false
	_,PlayOk=Gconfig.UserConf.PlayDomain[host].App[app]
	_,pubOk=Gconfig.UserConf.PublishDomain[host].App[app]
	if (!PlayOk) && (!pubOk){
		code ,level,desc = "NetStream.Connect.IllegalApplication","status","Illegal Application"
		errbak := fmt.Errorf("NetStream.Connect.IllegalApplication")
		if err = self.writeRtmpStatus(code, level, desc); err != nil {
			return
		}
		err = errbak
		self.flushWrite()
	}else{
		if PlayOk{
			self.UserCnf = Gconfig.UserConf.PlayDomain[host].App[app]
		}else{
			self.UserCnf = Gconfig.UserConf.PublishDomain[host].App[app]
		}
	}
	return
}

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
	var u *url.URL
	if u, err = url.Parse(app);err != nil {
		return
	}
	fmt.Println(u.Path)
	session.App = u.Path

	var tcurl string
	if _tcurl, ok = commandobj["tcUrl"]; !ok {
	}
	if ok {
		tcurl, _ = _tcurl.(string)
	}

	host := ""
	session.TcUrl = tcurl
	u, err = url.Parse(tcurl)
	if err != nil {
		code ,level,desc := "NetStream.Connect.IllegalDomain","status","Illegal domain"
		if err = session.writeRtmpStatus(code , level,desc);err != nil{
			return
		}
		session.flushWrite()
		err = fmt.Errorf("NetStream.Connect.IllegalDomain")
		return
	}else{
		host =u.Host
		m, _ := url.ParseQuery(u.RawQuery)
		if len(m["vhost"])>0{
			host = m["vhost"][0]
		}
		if err = session.RtmpcheckHost(host,"connect");err != nil {
			return
		}

	}
	if err = session.RtmpChckeApp(host,session.App);err != nil{
		return
	}

	session.Vhost = host
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

func (self *Session)RtmpCheckStreamIsSelf() bool{

	index:=hash(self.StreamAnchor)%uint32(len(Gconfig.RtmpServer.ClusterCnf))
	if Gconfig.RtmpServer.ClusterCnf[index] == Gconfig.RtmpServer.SelfIp{
		return true
	}else{
		self.pushIp = Gconfig.RtmpServer.ClusterCnf[index]
	}
	return false
}


func RtmpPublishCmdHandler(session *Session, b []byte) (n int, err error) {
	if Debug {
		fmt.Println("rtmp: < publish")
	}
	if err = session.RtmpcheckHost(session.Vhost,"publish");err !=nil{
		return
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
	var u *url.URL
	if u, err = url.Parse(publishpath);err != nil {
		return
	}else{
		session.StreamId = u.Path
		session.StreamAnchor = u.Path + ":" + Gconfig.UserConf.PublishDomain[session.Vhost].UniqueName + ":" + session.App
	}

	// here must do something
	/*if session.OnPlayOrPublish != nil {
		cberr = self.OnPlayOrPublish("publish", commandparams)
	}*/
	var code , level,desc string
	session.URL = createURL(session.TcUrl, session.App, publishpath)
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
	if session.selfPush == false {
		if session.UserCnf.RecodeFlv == 1 {
			//flv recode start
		}
		if session.UserCnf.RecodeHls == 1 {
			//hls recode start
		}
	}
	//
	if len(session.UserCnf.TurnHost) > 0{

	}

	if noSelf := session.RtmpCheckStreamIsSelf();noSelf != true{
		//push stream to the true server
		//rtmp://127.0.0.1/live?vhost=test.uplive.com/123
		url1:= "rtmp://" + session.pushIp + "/" + session.App +"?" + "vhost=" + session.Vhost + "/" + session.StreamId +"?hashpull=1"
		go rtmpClientPullProxy(session,"tcp",url1, session.pushIp, stageSessionDone,preparePullWriting)
	}
	session.stage = stageCommandDone
	return
}

func RtmpPlayCmdHandler(session *Session, b []byte) (n int, err error) {
	if Debug {
		fmt.Println("rtmp: < play")
	}
	if err = session.RtmpcheckHost(session.Vhost,"play");err !=nil{
		return
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
	var u *url.URL
	if u, err = url.Parse(playpath);err != nil {
		return
	}else{
		session.StreamId = u.Path
		session.StreamAnchor = u.Path + ":" + Gconfig.UserConf.PlayDomain[session.Vhost].UniqueName + ":" + session.App
	}
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
	session.URL = createURL(session.TcUrl, session.App, playpath)
	session.playing = true
	session.stage = stageCommandDone
	return
}

func CheckOnStatus(session *Session,b[]byte)(n int ,err error){
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


	objs, _ := commandparams[0].(amf.AMFMap)
	if objs == nil {
		err = fmt.Errorf("params[0] not object")
		return
	}

	_code, _:= objs["code"]
	if _code == nil {
		err = fmt.Errorf("code invalid")
		return
	}

	code, _ := _code.(string)
	switch session.OnStatusStage {
	case ConnectStage:
		err = fmt.Errorf("code != NetConnection.Connect.Success")
		return
	case PublishStage:
		if code != "NetStream.Publish.Start"{
			err = fmt.Errorf("code != NetConnection.Connect.Success")
			return
		}
	case PlayStage:
		if code != "NetStream.Play.Start"{
			err = fmt.Errorf("code != NetConnection.Connect.Success")
			return
		}

	}

	err = fmt.Errorf("NetConnection.Onstatus.Success")
	return

}

func CheckCreateStreamResult(session *Session,b []byte)(n int ,err error){
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

	_avmsgsid, _ := commandparams[0].(float64)
	session.avmsgsid = uint32(_avmsgsid)
	err = fmt.Errorf("NetConnection.CreateStream.Success")
	return
}

func CheckConnectResult(session *Session, b []byte) (n int, err error){
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
		err = fmt.Errorf("rtmp: connect result command params invalid")
		return
	}
	objs, _ := commandparams[0].(amf.AMFMap)
	if objs == nil {
		err = fmt.Errorf("params[0] not object")
		return
	}

	_code, _:= objs["code"]
	if _code == nil {
		err = fmt.Errorf("code invalid")
		return
	}

	code, _ := _code.(string)
	if code != "NetConnection.Connect.Success" {
		err = fmt.Errorf("code != NetConnection.Connect.Success")
		return
	}
	err = fmt.Errorf("NetConnection.Connect.Success")
	return
}


func newRtmpCmdHandler() (RtmpCmdHandles RtmpCmdHandle){
	RtmpCmdHandles = make(RtmpCmdHandle)
	RtmpCmdHandles["connect"] = RtmpConnectCmdHandler
	RtmpCmdHandles["createStream"] = RtmpCreateStreamCmdHandler
	RtmpCmdHandles["closeStream"] = RtmpCloseStreamCmdHandler
	RtmpCmdHandles["deleteStream"] = RtmpDeleteStreamCmdHandler
	RtmpCmdHandles["publish"] = RtmpPublishCmdHandler
	RtmpCmdHandles["play"] = RtmpPlayCmdHandler
	RtmpCmdHandles["onStatus"] =CheckOnStatus
	return
}


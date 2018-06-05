package rtmp

import (
	"fmt"
	"rtmpServerStudy/amf"
	"context"
	"rtmpServerStudy/AvQue"
	"net/url"
	"rtmpServerStudy/log"
	"time"
	"strings"
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
			err = fmt.Errorf("%s","NetStream.Connect.IllegalDomain")
		}
	case "publish":
		_,pubOk:=Gconfig.UserConf.PublishDomain[host]
		if (!pubOk){
			code ,level,desc = "NetStream.Publish.IllegalDomain","status","Illegal publish domain"
			err = fmt.Errorf("%s","NetStream.Publish.IllegalDomain")
		}
	case "play":
		_,pubOk:=Gconfig.UserConf.PlayDomain[host]
		if (!pubOk){
			code ,level,desc = "NetStream.Play.IllegalDomain","status","Illegal play domain"
			err = fmt.Errorf("%s","NetStream.Play.IllegalDomain")
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
		errbak := fmt.Errorf("%s","NetStream.Connect.IllegalApplication")
		if err = self.writeRtmpStatus(code, level, desc); err != nil {
			return
		}
		err = errbak
		self.flushWrite()
	}else{
		if PlayOk{
			if Gconfig.UserConf.PlayDomain[host].App[app] != nil {
				self.UserCnf = *(Gconfig.UserConf.PlayDomain[host].App[app])
			}
			self.uniqueName = Gconfig.UserConf.PlayDomain[host].UniqueName
		}else{
			if Gconfig.UserConf.PublishDomain[host].App[app] != nil {
				self.UserCnf = *(Gconfig.UserConf.PublishDomain[host].App[app])
			}
			self.uniqueName = Gconfig.UserConf.PublishDomain[host].UniqueName
		}
	}
	return
}

func RtmpConnectCmdHandler(session *Session, b []byte) (n int, err error) {
	//startTime:= time.Now()

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
		err = fmt.Errorf("Rtmp.Connect.CommandMsgAMF0(left bytes=%d)", len(b)-n)
		return
	}

	if commandobj == nil {
		err = fmt.Errorf("%s","Rtmp.Connect.Command.Params.Invalid")
		return
	}

	var ok bool
	var _app, _tcurl interface{}
	if _app, ok = commandobj["app"]; !ok {
		err = fmt.Errorf("%s","Rtmp.Connect.Params.Missing.App")
		return
	}

	app, _ := _app.(string)
	var u *url.URL
	if u, err = url.Parse(app);err != nil {
		return
	}
	/*if GDefaultPath[len(GDefaultPath)-1] != '/' {
		GDefaultPath = GDefaultPath + "/"
	}*/
	if u.Path[len(u.Path)-1] == '/'{
		session.App = u.Path[0:len(u.Path)-1]
	}else {
		session.App = u.Path
	}


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

		log.Log.Info(fmt.Sprintf("%s rtmp server parse tcurl err :%s",
							session.LogFormat(),tcurl))

		err = fmt.Errorf("%s","Rtmp.NetStream.Connect.IllegalDomain")
		return
	}else{
		host =u.Host
		m, _ := url.ParseQuery(u.RawQuery)
		if len(m["vhost"])>0{
			host = m["vhost"][0]
		}
		h := strings.Split(host, ":")
		if  len(h)>0{
			host = h[0]
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

	log.Log.Info(fmt.Sprintf("%s rtmp parse connect ok the host:%s tcurl:%s ",
						session.LogFormat(),host,tcurl))
	//dis := time.Now().Sub(startTime).Seconds()

	return
}

func RtmpCreateStreamCmdHandler(session *Session, b []byte) (n int, err error) {

	session.avmsgsid = uint32(1)
	var transid interface{}

	if transid, _, err = amf.ParseAMF0Val(b[n:]); err != nil {
		log.Log.Info(fmt.Sprintf("%s rtmp create stream cmd parse tranid err",
						session.LogFormat()))
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

	log.Log.Info(fmt.Sprintf("%s rtmp create stream ok!",
						session.LogFormat()))
	return
}

func RtmpCloseStreamCmdHandler(session *Session, b []byte) (n int, err error) {
	log.Log.Info(fmt.Sprintf("%s rtmp close stream cmd!",
						session.LogFormat()))
	return
}

func RtmpDeleteStreamCmdHandler(session *Session, b []byte) (n int, err error) {
	err = fmt.Errorf("%s","Rtmp.Delete.Stream")
	log.Log.Info(fmt.Sprintf("%s rtmp delete stream cmd handler!",
		session.LogFormat()))
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

	log.Log.Debug(fmt.Sprintf("%s rtmp publish cmd handler",
					session.LogFormat()))

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
		err = fmt.Errorf("Rtmp.Publish.CommandMsgAMF0.Left(bytes=%d)", len(b)-n)
		return
	}

	if len(commandparams) < 1 {
		err = fmt.Errorf("%s","Rtmp.Publish.CommandMsg.Params.Invalid")
		return
	}
	publishpath, _ := commandparams[0].(string)

	var u *url.URL
	if u, err = url.Parse(publishpath);err != nil {
		log.Log.Info(fmt.Sprintf("%s rtmp client publish parse publish path err:%s",
			session.LogFormat(),publishpath))
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

	log.Log.Info(fmt.Sprintf("%s rtmp client publish:%s desc:%s",
		session.LogFormat(),session.StreamAnchor,desc))

	if err = session.writeRtmpStatus(code , level,desc);err != nil{
		return
	}
	if err = session.flushWrite(); err != nil {
		return
	}
	session.publishing = true

	session.recordTime = time.Now()

	//转推逻辑
	if len(session.UserCnf.TurnHost) > 0{

	}
	session.IsSelf = false
	if session.IsSelf = session.RtmpCheckStreamIsSelf(); session.IsSelf != true{
		//push stream to the true server
		//rtmp://127.0.0.1/live?vhost=test.uplive.com/123

		url1:= "rtmp://" + session.pushIp + "/" + session.App +"?" + "vhost=" + session.Vhost + "/" + session.StreamId +"?hashpull=1"


		log.Log.Info(fmt.Sprintf("%s rtmp publish must hash push: push url:%s",
			session.LogFormat(),url1))
		go rtmpClientPullProxy(session,"tcp", session.pushIp,url1,stageSessionDone)
	}

	if Gconfig.RtmpServer.QuicPush == 1 {
		url1 := "rtmp://" + session.Vhost + ":8888" + "/" + session.App + "/" + session.StreamId
		go rtmpClientPullProxy(session,"udp",session.Vhost + ":8888",url1,stageSessionDone)
	}

	//just hash self record
	if session.IsSelf == true {
		RecordPublishHandler(session)
	}
	log.Log.Info(fmt.Sprintf("%s rtmp publish ok!",
					session.LogFormat()))
	session.stage = stageCommandDone
	return
}

func RtmpPlayCmdHandler(session *Session, b []byte) (n int, err error) {

	log.Log.Debug(fmt.Sprintf("%s rtmp play cmd handler",
		session.LogFormat()))

	if err = session.RtmpcheckHost(session.Vhost, "play"); err != nil {
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
		err = fmt.Errorf("Rtmp.Play.CommandMsgAMF0.Left(bytes=%d)", len(b) - n)
		return
	}

	if len(commandparams) < 1 {
		err = fmt.Errorf("%s", "Rtmp.Play.CommandMsg.Params.Invalid")
		return
	}

	playpath, _ := commandparams[0].(string)

	var u *url.URL
	if u, err = url.Parse(playpath); err != nil {
		log.Log.Info(fmt.Sprintf("%s rtmp play parse playPath err playPath:%s err:%s !",
			session.LogFormat(), playpath, err.Error()))
		return
	} else {
		session.StreamId = u.Path
		session.StreamAnchor = u.Path + ":" + Gconfig.UserConf.PlayDomain[session.Vhost].UniqueName + ":" + session.App
	}

	//Onplay_handler{}
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

	log.Log.Info(fmt.Sprintf("%s rtmp play cmd ok!",
					session.LogFormat()))

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
		err = fmt.Errorf("Rtmp.OnStatus.CommandMsgAMF0.Left(bytes=%d)", len(b)-n)
		return
	}


	objs, _ := commandparams[0].(amf.AMFMap)
	if objs == nil {
		err = fmt.Errorf("%s","Rtmp.OnStatus.Params[0].Not.Object")
		return
	}

	_code, _:= objs["code"]
	if _code == nil {
		err = fmt.Errorf("%s","Rtmp.OnStatus.Code.Invalid")
		return
	}

	code, _ := _code.(string)
	switch session.OnStatusStage {
	case ConnectStage:
		err = fmt.Errorf("%s","NetStream.Connect.Bad")
		return
	case PublishStage:
		if code != "NetStream.Publish.Start"{
			err = fmt.Errorf("%s","NetStream.Publish.Bad")
			return
		}
	case PlayStage:
		if code != "NetStream.Play.Start"{
			err = fmt.Errorf("%s","NetStream.Play.Bad")
			return
		}

	}

	session.resultCheckStage = StageOnstatusChecked
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
		err = fmt.Errorf("Rtmp.CreateStreamResult.CommandMsgAMF0.Left(Bytes=%d)", len(b)-n)
		return
	}

	_avmsgsid, _ := commandparams[0].(float64)
	session.avmsgsid = uint32(_avmsgsid)
	session.resultCheckStage = StageCreateStreamResultChecked
	//err = fmt.Errorf("NetConnection.CreateStream.Success")
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
		err = fmt.Errorf("Rtmp.Connect.CommandMsgAMF0.Left.Bytes=%d", len(b)-n)
		return
	}

	if commandobj == nil {
		err = fmt.Errorf("%s","Rtmp.Connect.Result.Command.Params.Invalid")
		return
	}
	objs, _ := commandparams[0].(amf.AMFMap)
	if objs == nil {
		err = fmt.Errorf("%s","Rtmp.Connect.Params[0].Not.Object")
		return
	}

	_code, _:= objs["code"]
	if _code == nil {
		err = fmt.Errorf("%s","Rtmp.Connect.Amf.Code.Invalid")
		return
	}

	code, _ := _code.(string)
	if code != "NetConnection.Connect.Success" {
		err = fmt.Errorf("%s","Rtmp.Connect.Code != NetConnection.Connect.Success")
		return
	}
	session.resultCheckStage = StageConnectResultChecked
	//err = fmt.Errorf("%s","NetConnection.Connect.Success")
	return
}

func setDtaFrameHandler(session *Session, b []byte) (n int, err error) {

	if _, err = session.handleCommandMsgAMF0(b,session.rtmpCmdHandler); err != nil {
		return
	}

	return
}

func onMetaDataHandler(session *Session, b []byte) (n int, err error) {
	var size int
	n = 0
	session.metaData = amf.AMFMap{}
	var metaDatas []amf.AMFMap

	for n < len(b) {
		var  obj interface{}
		if obj, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
			//some log
		}else{
			switch value1 := obj.(type) {
			case amf.AMFMap:
				metaDatas = append(metaDatas, value1)
			}
		}
		n += size
	}
	if len(metaDatas) >0{
		for i:=range metaDatas{
			for k,_:=range metaDatas[i]{
				session.metaData[k] = (metaDatas[i])[k]
			}
		}
	}
	session.metaData["create"] = "kouyang"
	session.metaversion++
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
	RtmpCmdHandles["@setDataFrame"] =setDtaFrameHandler
	RtmpCmdHandles["onMetaData"] =onMetaDataHandler
	return
}


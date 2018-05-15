package config

import (
"fmt"
"gopkg.in/yaml.v2"
"path/filepath"
"io/ioutil"
)

type App struct {
	GopCacheNum int `yaml:"GopCacheNum"`
	ExtTimeSend int `yaml:"ExtTimeSend"`
	RecodeFlv int `yaml:"RecodeFlv"`
	RecodeHls int `yaml:"RecodeHls"`
	HlsFragment   string `yaml:"HlsFragment"`
	RecodeFlvPath string `yaml:"RecodeFlvPath"`
	RecodeHlsPath string `yaml:"RecodeHlsPath"`
	RecodePicture int `yaml:"RecodePicture"`
	RecodePicPath string `yaml:"RecodePicPath"`
	RecidePicFragment string `yaml:"RecidePicFragment"`
	TurnHost []string `yaml:"TurnHost"`
}


type playDomain struct {
	UniqueName string `yaml:"UniqueName"`
	App map[string]*App `yaml:"App"`
}

type publishdomain struct {
	UniqueName string `yaml:"UniqueName"`
	App map[string]*App `yaml:"App"`
}

type Rtmpserver struct{
	RtmpListen []string `yaml:"RtmpListen"`
	ClusterCnf []string `yaml:"ClusterCnf"`
	SelfIp string `yaml:"SelfIp"`
	HttpListen []string `yaml:"HttpListen"`
	QuicListen string `yaml:"QuicListen"`
	KcpListen string `yaml:"KcpListen"`
}
type UserConf struct {
	PublishDomain map[string]publishdomain `yaml:"PublishDomain"`
	PlayDomain map[string]playDomain `yaml:"PlayDomain"`
}

type RtmpServerCnf struct{
	LogInfo LogInfo 	 `yaml:"LogInfo"`
	RtmpServer Rtmpserver   `yaml:"RtmpServer"`
	UserConf UserConf 	 `yaml:"UserConf"`
}

/*
 Level: "DEBUG" #DEBUG,ERROR, INFO
  OutPaths: "./rtmp.log"
  LogFileCutInterval: "1h" #1s,1m,1h
*/

type LogInfo struct {
	Level string    		`yaml:"Level"`
	OutPaths string			`yaml:"OutPaths"`
	LogFileCutInterval string       `yaml:"LogFileCutInterval"`
}

func ParseConfig(file string) (err error,cnf *RtmpServerCnf){

	filename, _ := filepath.Abs(file)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	cnf = new(RtmpServerCnf)

	err = yaml.Unmarshal(yamlFile, cnf)
	if err != nil {
		fmt.Println("parse conf err please check")
		panic(err)
	}
	return err,cnf
}


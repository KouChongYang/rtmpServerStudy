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
	RecodeFlvPath string `yaml:"RecodeFlvPath"`
	RecodeHlsPath string `yaml:"RecodeHlsPath"`
	RecodePicture int `yaml:"RecodePicture"`
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
}
type UserConf struct {
	PublishDomain map[string]publishdomain `yaml:"PublishDomain"`
	PlayDomain map[string]playDomain `yaml:"PlayDomain"`
}

type RtmpServerCnf struct{
	RtmpServer Rtmpserver `yaml:"RtmpServer"`
	UserConf UserConf `yaml:"UserConf"`
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


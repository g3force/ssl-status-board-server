package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"log"
)

type AuthCredential struct {
	Username string `yaml:"Username"`
	Password string `yaml:"Password"`
}

type ProxyConfig struct {
	ListenAddress   string           `yaml:"ListenAddress"`
	SubscribePath   string           `yaml:"SubscribePath"`
	PublishPath     string           `yaml:"PublishPath"`
	AuthCredentials []AuthCredential `yaml:"AuthCredentials"`
}

func ReadProxyConfig(fileName string) ProxyConfig {
	config := ProxyConfig{}
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	d, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalln(err)
	}
	err = yaml.Unmarshal(d, &config)
	if err != nil {
		log.Fatalln(err)
	}
	return config
}

package app

import (
	"encoding/json"
	"os"
)

// Config The config struct
type Config struct {
	ServerAddr string `json:"server_addr"`
	Key        string `json:"key"`

	WSPath    string `json:"path"`
	Timeout   int    `json:"timeout"`
	MixinFunc string `json:"mixin_func"`
}

type nativeConfig Config

var DefaultConfig = nativeConfig{
	ServerAddr: ":3001",
	Key:        "fuck_key",
	WSPath:     "/freedom",
	Timeout:    30,
	MixinFunc:  "xor",
}

func (c *Config) UnmarshalJSON(data []byte) error {
	_ = json.Unmarshal(data, &DefaultConfig)
	*c = Config(DefaultConfig)
	return nil
}

func (c *Config) LoadConfig(configFile string) (err error) {
	file, err := os.Open(configFile)
	if err != nil {
		return
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(c)
	if err != nil {
		return
	}
	return
}

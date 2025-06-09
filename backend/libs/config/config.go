package config

type Config struct {
	Name string `yaml:"name" json:"name"`
	Addr string `yaml:"addr" json:"addr"`
	Mask string `yaml:"mask" json:"mask"`
	Port int    `yaml:"port" json:"port"`
	Host string `yaml:"host" json:"host"`
}

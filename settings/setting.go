package settings

import (
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var Conf = new(AppConfig)

type AppConfig struct {
	Name         string `mapstructure:"name"`
	Port         int    `mapstructure:"port"`
	HTTPPort     int    `mapstructure:"http_port"`
	*MySQLConfig `mapstructure:"mysql"`
}

type MySQLConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	DbName      string `mapstructure:"dbname"`
	MaxOpenConn int    `mapstructure:"max_open_conns"`
	MaxIdleConn int    `mapstructure:"max_idle_conns"`
}

// Validate 校验启动配置。
func (c *AppConfig) Validate() error {
	if c.MySQLConfig == nil {
		return fmt.Errorf("缺少 mysql 配置")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("gRPC 端口必须在 1～65535 之间")
	}
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("HTTP 端口必须在 1～65535 之间")
	}
	if c.Port == c.HTTPPort {
		return fmt.Errorf("gRPC 和 HTTP 端口不能相同")
	}
	if c.MySQLConfig.Port < 1 || c.MySQLConfig.Port > 65535 {
		return fmt.Errorf("MySQL 端口必须在 1～65535 之间")
	}
	return nil
}

func Init() (err error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err != nil {
		log.Printf("viper.ReadInConfig() failed , err:%v\n", err)
		return err
	}
	if err = viper.Unmarshal(Conf); err != nil {
		log.Printf("=viper.Unmarshal(Conf) failed , err:%v\n", err)
		return
	}
	if err = Conf.Validate(); err != nil {
		return err
	}
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("config file chaged .......")
		if err := viper.Unmarshal(Conf); err != nil {
			log.Printf("=viper.Unmarshal(Conf) failed , err:%v\n", err)
			return
		}
	})
	return
}

package main

import (
	"fmt"
	"github.com/programmfabrik/fylr-apitest/lib/filesystem"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
	"time"
)

type FylrConfigStruct struct {
	Fylr struct {
		Log struct {
			File   string `mapstructure:"file"`
			Level  string `mapstructure:"level"`
			Header bool   `mapstructure:"header"`
		} `mapstructure:"log"`
	}
	Apitest struct {
		Server    string                 `mapstructure:"server"`
		DBName    string                 `mapstructure:"db-name"`
		StoreInit map[string]interface{} `mapstructure:"store"`
		Report    struct {
			File   string `mapstructure:"file"`
			Format string `mapstructure:"format"`
		} `mapstructure:"report"`
	}
}

var FylrConfig FylrConfigStruct

var startTime time.Time

func LoadConfig(cfgFile string) {
	startTime = time.Now()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		log.Fatalf("Must provide a config file")
	}

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Can't read config \"%s\": %s", cfgFile, err)
	}

	viper.Unmarshal(&FylrConfig)

}

// TestToolConfig gives us the basic testtool infos
type TestToolConfig struct {
	ServerURL       string
	DataBaseName    string
	rootDirectorys  []string
	TestDirectories []string
}

// NewTestToolConfig is mostly used for testing purpose. We can setup our config with this function
func NewTestToolConfig(serverURL, dataBaseName string, rootDirectory []string) (config TestToolConfig, err error) {
	config = TestToolConfig{
		ServerURL:      serverURL,
		DataBaseName:   dataBaseName,
		rootDirectorys: rootDirectory,
	}
	err = config.extractTestDirectories()
	return config, err
}

func (config *TestToolConfig) extractTestDirectories() error {
	for _, rootDirectory := range config.rootDirectorys {
		if _, err := filesystem.Fs.Stat(rootDirectory); err != nil {
			return fmt.Errorf("The given root directory '%s' is not valid", rootDirectory)
		}
	}

	for _, rootDirectory := range config.rootDirectorys {
		err := afero.Walk(filesystem.Fs, rootDirectory, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				//Skip directories not containing a manifest
				if _, err := filesystem.Fs.Stat(filepath.Join(path, "manifest.json")); err == nil {
					config.TestDirectories = append(config.TestDirectories, path)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

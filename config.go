package main

import (
	"fmt"
	"github.com/programmfabrik/fylr-apitest/lib/filesystem"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
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
		StoreInit map[string]interface{} `mapstructure:"store"`
		Limit     struct {
			Request  int `mapstructure:"request"`
			Response int `mapstructure:"response"`
		} `mapstructure:"limit"`
		Report struct {
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

	// Set default values
	if FylrConfig.Apitest.Limit.Response == 0 {
		FylrConfig.Apitest.Limit.Response = 20
	}

}

// TestToolConfig gives us the basic testtool infos
type TestToolConfig struct {
	ServerURL       string
	rootDirectorys  []string
	TestDirectories []string
	LogNetwork      bool
	LogVerbose      bool
}

// NewTestToolConfig is mostly used for testing purpose. We can setup our config with this function
func NewTestToolConfig(serverURL string, rootDirectory []string, logNetwork bool, logVerbose bool) (config TestToolConfig, err error) {
	config = TestToolConfig{
		ServerURL:      serverURL,
		rootDirectorys: rootDirectory,
		LogNetwork:     logNetwork,
		LogVerbose:     logVerbose,
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

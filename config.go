package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/programmfabrik/apitest/pkg/lib/filesystem"
	"github.com/programmfabrik/apitest/pkg/lib/util"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type ConfigStruct struct {
	Apitest struct {
		Server    string         `mapstructure:"server"`
		StoreInit map[string]any `mapstructure:"store"`
		Limit     struct {
			Request  int `mapstructure:"request"`
			Response int `mapstructure:"response"`
		} `mapstructure:"limit"`
		Log struct {
			Short bool `mapstructure:"short"`
		} `mapstructure:"log"`
		Report struct {
			File   string `mapstructure:"file"`
			Format string `mapstructure:"format"`
		} `mapstructure:"report"`
		OAuthClient util.OAuthClientsConfig `mapstructure:"oauth_client"`
	}
}

var Config ConfigStruct

var startTime time.Time

func LoadConfig(cfgFile string) {
	startTime = time.Now()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		logrus.Infof("No config file provided (will only use command line parameters)")
	}

	if err := viper.ReadInConfig(); err != nil {
		logrus.Infof("No config %q read (will only use command line parameters): %s", cfgFile, err.Error())
	}

	viper.Unmarshal(&Config)
}

// TestToolConfig gives us the basic testtool infos
type TestToolConfig struct {
	ServerURL       string
	rootDirectorys  []string
	TestDirectories []string
	LogNetwork      bool
	LogVerbose      bool
	LogShort        bool
	OAuthClient     util.OAuthClientsConfig
}

// NewTestToolConfig is mostly used for testing purpose. We can setup our config with this function
func NewTestToolConfig(serverURL string, rootDirectory []string, logNetwork bool, logVerbose bool, logShort bool) (config TestToolConfig, err error) {
	config = TestToolConfig{
		ServerURL:      serverURL,
		rootDirectorys: rootDirectory,
		LogNetwork:     logNetwork,
		LogVerbose:     logVerbose,
		LogShort:       logShort,
		OAuthClient:    Config.Apitest.OAuthClient,
	}

	config.fillInOAuthClientNames()

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
				// Skip directories starting with "_"
				if strings.Contains(path, "/_") {
					// logrus.Infof("Skipping: %s", path)
					return filepath.SkipDir
				}
				//Skip directories not containing a manifest
				_, err := filesystem.Fs.Stat(filepath.Join(path, "manifest.json"))
				if err != nil {
					return nil
				}
				config.TestDirectories = append(config.TestDirectories, path)
				dirRel, err := filepath.Rel(rootDirectory, path)
				if err != nil {
					dirRel = path
				}
				if dirRel == "." {
					dirRel = filepath.Base(path)
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

// fillInOAuthClientNames fills in the Client field of loaded OAuthClientConfig
// structs, which the user may have left unset in the config yaml file.
func (config *TestToolConfig) fillInOAuthClientNames() {
	for key, clientConfig := range config.OAuthClient {
		if clientConfig.Client == "" {
			clientConfig.Client = key
			config.OAuthClient[key] = clientConfig
		}
	}
}

package configuro

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	ens "github.com/go-playground/validator/translations/en"
	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"gopkg.in/go-playground/validator.v9"
)

//Config Loads and WithValidateByTags Arbitrary structs based on options (set at constructing)
type Config struct {
	envLoad                 bool
	envPrefix               string
	envDotFileLoad          bool
	envDotFilePath          string
	configFileLoad          bool
	configFileErrIfNotFound bool
	configFilepath          string
	configFilepathEnv       bool
	configFilepathEnvName   string
	configEnvExpand         bool
	validateStopOnFirstErr  bool
	validateRecursive       bool
	validateUsingTags       bool
	validateTag             string
	tag                     string
	viper                   *viper.Viper
	validator               *validator.Validate
	validatorTrans          ut.Translator
	decodeHook              viper.DecoderConfigOption
}

//NewConfig Create config Loader/Validator according to options.
func NewConfig(opts ...ConfigOptions) (*Config, error) {
	var err error

	config := &Config{}

	options := DefaultOptions()

	options = append(options, opts...)

	// Loop through each option
	for _, opt := range options {
		// Call the option giving the instantiated
		// *House as the argument
		err = opt(config)
		if err != nil {
			return nil, err
		}

	}

	err = config.initialize()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func DefaultOptions() []ConfigOptions {
	return []ConfigOptions{
		WithLoadFromEnvVars("CONFIG"),
		WithLoadFromConfigFile("./config.yml", false),
		WithEnvConfigPathOverload("CONFIG_DIR"),
		WithLoadDotEnv("./env"),
		WithExpandEnvVars(),
		WithValidateByTags(true),
		Tag("config", "validate"),
	}
}

func (c *Config) initialize() error {

	// Init Viper
	c.viper = viper.New()

	if c.envDotFileLoad {
		// load .env vars
		if _, err := os.Stat(c.envDotFilePath); err == nil || !os.IsNotExist(err) {
			err := godotenv.Load(c.envDotFilePath)
			if err != nil {
				return fmt.Errorf("error loading .env envvars from \"%s\": %s", c.envDotFilePath, err.Error())
			}
		}
	}

	if c.envLoad {
		c.enableEnvLoad()
	}

	if c.configFileLoad {
		err := c.enableConfigFileLoad()
		if err != nil {
			return err
		}
	}

	// decoder config
	c.addDecoderConfig()

	// Tag validator
	if c.validateUsingTags {
		c.enableValidateUsingTag()
	}

	return nil
}

// ---------------------------------------------------------------------------------------------------------------------

//ConfigOptions Modify Config Options Accordingly
type ConfigOptions func(*Config) error

//WithLoadFromEnvVars Load Configuration from Environment Variables if they're set.
// 	- Prefix Require Environment Variables to prefixed with the set prefix (All CAPS)
// 	- For Nested fields replace `.` with `_` and if key itself has any `_` or `-` replace with `__` (e.g `config.host` to be `CONFIG_HOST`)
//	- Arrays can be declared in environment variables using
//		1. comma separated list.
//		2. json encoded array in a string.
//	- Maps and objects can be declared in environment using a json encoded object in a string.
func WithLoadFromEnvVars(EnvPrefix string) ConfigOptions {
	return func(h *Config) error {
		if EnvPrefix == "" {
			return fmt.Errorf("env prefix must be declared")
		}

		h.envLoad = true
		h.envPrefix = strings.ToUpper(EnvPrefix)

		return nil
	}
}

func WithoutLoadFromEnvVars() ConfigOptions {
	return func(h *Config) error {
		h.envLoad = false
		h.envPrefix = ""
		return nil
	}
}

//WithLoadDotEnv Allow loading .env file (notice that this is application global not to this config instance only)
func WithLoadDotEnv(envDotFilePath string) ConfigOptions {
	return func(h *Config) error {
		h.envDotFileLoad = true
		h.envDotFilePath = envDotFilePath
		return nil
	}
}

//WithLoadDotEnv Allow loading .env file (notice that this is application global not to this config instance only)
func WithoutLoadDotEnv() ConfigOptions {
	return func(h *Config) error {
		h.envDotFileLoad = false
		h.envDotFilePath = ""
		return nil
	}
}

//WithLoadFromConfigFile Load Config from file (notice that file doesn't have an extension as any file with supported extension should work)
func WithLoadFromConfigFile(Filepath string, ErrIfFileNotFound bool) ConfigOptions {
	return func(h *Config) error {
		h.configFileLoad = true
		h.configFileErrIfNotFound = ErrIfFileNotFound
		return h.setConfigFilepath(Filepath)
	}
}

func WithoutLoadFromConfigFile() ConfigOptions {
	return func(h *Config) error {
		h.configFileLoad = false
		h.configFileErrIfNotFound = false
		h.configFilepath = ""
		return nil
	}
}

//WithEnvConfigPathOverload Allow to override Config Dir Path with an Env Variable
func WithEnvConfigPathOverload(configFilepathENV string) ConfigOptions {
	return func(h *Config) error {
		h.configFilepathEnv = true
		h.configFilepathEnvName = strings.ToUpper(configFilepathENV)
		return nil
	}
}

//WithEnvConfigPathOverload Allow to override Config Dir Path with an Env Variable
func WithoutEnvConfigPathOverload() ConfigOptions {
	return func(h *Config) error {
		h.configFilepathEnv = false
		h.configFilepathEnvName = ""
		return nil
	}
}

//WithExpandEnvVars Expand config values with ${ENVVAR} with the value of ENVVAR in environment variables.
// You can set default if ENVVAR is not set using the following ${ENVVAR|defaultValue}
func WithExpandEnvVars() ConfigOptions {
	return func(h *Config) error {
		h.configEnvExpand = true
		return nil
	}
}
func WithoutExpandEnvVars() ConfigOptions {
	return func(h *Config) error {
		h.configEnvExpand = false
		return nil
	}
}

//WithValidateByTags Control WithValidateByTags function behavior.
func WithValidateByTags(StopOnFirstErr bool) ConfigOptions {
	return func(h *Config) error {
		h.validateStopOnFirstErr = StopOnFirstErr
		h.validateUsingTags = true
		return nil
	}
}

func WithoutValidateByTags() ConfigOptions {
	return func(h *Config) error {
		h.validateStopOnFirstErr = false
		h.validateUsingTags = false
		return nil
	}
}

//Tag Change default tag.
func Tag(structTag, validateTag string) ConfigOptions {
	return func(h *Config) error {
		h.tag = structTag
		h.validateTag = validateTag
		return nil
	}
}

var supportedExt = []string{".json", ".toml", ".yaml", ".yml"}

func isSupportedExtension(ext string) bool {
	found := false
	for _, supportedExt := range supportedExt {
		if ext == supportedExt {
			found = true
		}
	}
	return found
}

func (c *Config) setConfigFilepath(path string) error {

	// Turn into ABS filepath
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Check extension
	ext := filepath.Ext(path)
	if ext == "" {
		return fmt.Errorf("config file has no extension")
	}

	isSupported := isSupportedExtension(ext)

	if !isSupported {
		return fmt.Errorf("file with extension %s is not supported", ext)
	}

	c.configFilepath = path
	return nil
}

func (c *Config) enableValidateUsingTag() {
	c.validator = validator.New()
	c.validator.SetTagName(c.validateTag)
	// Get English Errors
	uni := ut.New(en.New(), en.New())
	c.validatorTrans, _ = uni.GetTranslator("en")
	_ = ens.RegisterDefaultTranslations(c.validator, c.validatorTrans)
}

func (c *Config) addDecoderConfig() {
	DefaultDecodeHookFuncs := []mapstructure.DecodeHookFunc{
		stringJSONArrayToSlice(),
		stringJSONObjToMap(),
		stringJSONObjToStruct(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToIPHookFunc(),
	}
	if c.configEnvExpand {
		DefaultDecodeHookFuncs = append([]mapstructure.DecodeHookFunc{expandEnvVariablesWithDefaults()}, DefaultDecodeHookFuncs...)
	}
	c.decodeHook = viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		DefaultDecodeHookFuncs...,
	))
}

func (c *Config) enableConfigFileLoad() error {

	if c.configFilepathEnv {
		configDirEnvValue, isSet := os.LookupEnv(c.configFilepathEnvName)
		if isSet {
			err := c.setConfigFilepath(configDirEnvValue)
			if err != nil {
				return err
			}
		}
	}

	c.viper.SetConfigFile(c.configFilepath)
	return nil
}

func (c *Config) enableEnvLoad() {
	c.viper.SetEnvPrefix(c.envPrefix)
	// Viper add the `prefix` + '_' to the Key *before* passing it to Key Replacer,causing the replacer to replace the '_' with '__' when it shouldn't.
	// by adding the Prefix to the replacer twice, this will let the replacer escapes the prefix as it scans through the string.
	c.viper.SetEnvKeyReplacer(strings.NewReplacer(c.envPrefix+"_", c.envPrefix+"_", "_", "__", ".", "_"))
	c.viper.AutomaticEnv()
}

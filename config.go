package bamboo

import (
	"io"
	"io/ioutil"
)

type Config struct {
	// the url for bamboo
	URL string
	// the output for debug logging
	LogOutput io.Writer
}

func NewDefaultConfig() Config {
	return Config{
		URL:       "http://127.0.0.1:8000",
		LogOutput: ioutil.Discard,
	}
}

package core

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const File50MB = "http://212.183.159.230/50MB.zip"

func TestNewDownloader(t *testing.T) {
	d := NewDownloader(File50MB, 2<<3)
	assert.Nil(t, d.Process())
}

package core

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const File50MB = "https://nodejs.org/dist/v18.17.1/node-v18.17.1-x64.msi"

func TestNewDownloader(t *testing.T) {
	d := NewDownloader(File50MB, 1<<4)
	assert.Nil(t, d.Process())
}

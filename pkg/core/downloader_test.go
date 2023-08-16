package core

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const File50MB = "http://ipv4.download.thinkbroadband.com/50MB.zip"

func TestDownloader_getFileHeader(t *testing.T) {
	d := NewDownloader(File50MB, 1)
	err := d.getFileHeader()
	assert.Nil(t, err)
	assert.Equal(t, 5242880, d.fileHeader.ContentLength)
}

func TestDownloader_download(t *testing.T) {
	checkpoint := time.Now()
	d := NewDownloader(File50MB, 16)
	assert.Nil(t, d.Download())
	fmt.Println("16 thread take", time.Now().Sub(checkpoint))
	//checkpoint = time.Now()
	//d = NewDownloader(File50MB, 1)
	//assert.Nil(t, d.Download())
	//fmt.Println("1 thread take", time.Now().Sub(checkpoint))

}

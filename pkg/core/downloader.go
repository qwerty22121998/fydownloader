package core

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"
)

type Downloader struct {
	url        string
	client     *http.Client
	fileInfo   *FileInfo
	totalChunk int
	chunks     map[int]*Chunk
	errChan    chan error
	wg         sync.WaitGroup
}

func NewDownloader(url string, totalParts int) *Downloader {
	return &Downloader{
		url:        url,
		client:     &http.Client{},
		fileInfo:   nil,
		totalChunk: totalParts,
		chunks:     make(map[int]*Chunk),
		errChan:    make(chan error),
	}
}

type Chunk struct {
	index int
	from  int
	to    int
	out   *os.File
}

func (c *Chunk) Range() string {
	return fmt.Sprintf("bytes=%d-%d", c.from, c.to)
}

func (d *Downloader) Process() error {
	if err := d.fetchFileInfo(); err != nil {
		return err
	}
	d.splitPart()
	for _, chunk := range d.chunks {
		chunk := chunk
		d.wg.Add(1)
		go d.processChunk(chunk)
	}
	go func() {
		for err := range d.errChan {
			log.Print(err)
		}
	}()
	d.wg.Wait()
	return nil
}

func (d *Downloader) processChunk(c *Chunk) {
	defer d.wg.Done()
	// create temp file
	tempFilePath := fmt.Sprintf("temp/%v.part%v", d.fileInfo.FileName, c.index)
	file, err := CreateFile(tempFilePath)
	if err != nil {
		d.errChan <- err
		return
	}

	req, err := http.NewRequest(http.MethodGet, d.url, nil)
	if err != nil {
		d.errChan <- err
		return
	}
	req.Header.Add("Range", c.Range())
	resp, err := d.client.Do(req)
	if err != nil {
		d.errChan <- err
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		d.errChan <- fmt.Errorf("request failed with status code %v", resp.StatusCode)
		return
	}
	if _, err := io.Copy(file, resp.Body); err != nil {
		d.errChan <- err
	}
}

func (d *Downloader) splitPart() {
	if !d.fileInfo.RangeSupported {
		chunk := &Chunk{
			index: 0,
			from:  0,
			to:    d.fileInfo.FileSize,
			out:   nil,
		}
		d.chunks[chunk.index] = chunk
		return
	}
	chunkSize := d.fileInfo.FileSize / d.totalChunk
	from := 0
	for i := 0; i < d.totalChunk; i++ {
		chunk := &Chunk{
			index: i,
			from:  from,
			to:    from + chunkSize - 1,
			out:   nil,
		}
		if i == d.totalChunk-1 {
			chunk.to += d.fileInfo.FileSize % d.totalChunk
		}
		from += chunkSize
		d.chunks[chunk.index] = chunk
	}
}

func (d *Downloader) fetchFileInfo() error {
	req, err := http.NewRequest(http.MethodHead, d.url, nil)
	if err != nil {
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("request failed with status code %v", resp.StatusCode)
	}
	// check if accept range
	acceptRange := resp.Header.Get("Accept-Ranges")
	// check file size
	contentLengthStr := resp.Header.Get("Content-Length")
	if len(contentLengthStr) == 0 {
		return errors.New("cannot get content length")
	}
	contentLength, err := strconv.Atoi(contentLengthStr)
	if err != nil {
		return err
	}
	// guess file name
	fileName := ""
	contentDisposition := resp.Header.Get("Content-Disposition")
	if len(contentDisposition) != 0 {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err != nil {

		} else {
			fileName = params["filename"]
		}
	}
	// get nothing from header
	if fileName == "" {
		fileURL, err := url.Parse(d.url)
		if err != nil {
			return err
		}
		fileURL.RawQuery = ""
		fileName = path.Base(fileURL.String())
	}

	d.fileInfo = &FileInfo{
		FileName:       fileName,
		RangeSupported: acceptRange == "bytes",
		FileSize:       contentLength,
	}
	return nil
}

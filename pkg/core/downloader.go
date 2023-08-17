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

const DefaultBufferSize = 1 << 9

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
		url: url,
		client: &http.Client{
			Timeout: 0,
		},
		fileInfo:   nil,
		totalChunk: totalParts,
		chunks:     make(map[int]*Chunk),
		errChan:    make(chan error),
	}
}

type Chunk struct {
	index     int
	from      int
	to        int
	outPath   string
	totalSize int
	download  int
	errChan   chan error
	out       *os.File
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

	return d.combineFile()
}

func (d *Downloader) processChunk(c *Chunk) {
	defer d.wg.Done()
	// create temp file
	tempFilePath := fmt.Sprintf("temp/%v.part%v", d.fileInfo.FileName, c.index)
	c.outPath = tempFilePath
	file, err := CreateFile(tempFilePath)
	if err != nil {
		d.errChan <- err
		return
	}
	c.out = file

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
	buf := make([]byte, DefaultBufferSize)
	for {
		r, err := resp.Body.Read(buf)
		if err != nil {
			if err == io.EOF {
				return
			}
			d.errChan <- err
			return
		}
		if _, err := file.Write(buf[:r]); err != nil {
			d.errChan <- err
			return
		}
		c.download += r
	}
}

func (d *Downloader) combineFile() error {
	file, err := CreateFile(d.fileInfo.FileName)
	if err != nil {
		return err
	}
	defer file.Close()
	for i := 0; i < d.totalChunk; i++ {
		func() {
			chunk := d.chunks[i]
			_, err := chunk.out.Seek(0, 0)
			if err != nil {
				d.errChan <- err
				return
			}
			defer os.Remove(chunk.outPath)
			defer chunk.out.Close()

			writeAt := io.NewOffsetWriter(file, int64(chunk.from))
			if _, err := io.Copy(writeAt, chunk.out); err != nil {
				d.errChan <- err
				return
			}
		}()
	}
	return nil
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
		currSize := chunkSize
		if i == d.totalChunk-1 {
			currSize += d.fileInfo.FileSize % d.totalChunk
		}
		chunk := &Chunk{
			index:     i,
			from:      from,
			to:        from + currSize - 1,
			totalSize: currSize,
			out:       nil,
			errChan:   make(chan error),
		}

		from += currSize
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

package core

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"
)

type Downloader struct {
	url         string
	client      *http.Client
	chunkNumber int
	chunkChan   chan ChunkProcessor
	errChan     chan error
	wg          sync.WaitGroup
	done        chan any
	file        *os.File
}

func NewDownloader(url string, chunkNumber int) *Downloader {
	return &Downloader{
		url:         url,
		client:      &http.Client{},
		chunkNumber: chunkNumber,
		chunkChan:   make(chan ChunkProcessor),
		errChan:     make(chan error),
		wg:          sync.WaitGroup{},
		done:        make(chan any),
	}
}

func (d *Downloader) Download() error {
	info, err := d.getFileHeader()
	if err != nil {
		return err
	}
	dst, err := os.Create(info.FileName)
	if err != nil {
		return err
	}
	defer dst.Close()
	d.file = dst

	numThread := d.chunkNumber
	if info.AcceptRanges != "bytes" {
		numThread = 1
	}
	ranges := SplitChunk(info.ContentLength, numThread)
	d.wg.Add(len(ranges))
	go func() {
		d.wg.Wait()
		d.done <- nil
	}()
	for _, r := range ranges {
		r := r
		go d.downloadChunk(r)
	}
	for {
		select {
		case chunk := <-d.chunkChan:
			go d.processChunk(chunk)
		case err := <-d.errChan:
			return err
		case <-d.done:
			fmt.Println("done")
			return nil
		}
	}
}

func (d *Downloader) processChunk(c ChunkProcessor) {
	defer d.wg.Done()
	defer c.Close()

	writer := io.NewOffsetWriter(d.file, int64(c.Chunk.From))

	_, err := io.Copy(writer, c.Reader)
	if err != nil {
		d.errChan <- err
	}
}

func (d *Downloader) downloadChunk(chunk FileChunk) {
	req, err := http.NewRequest(http.MethodGet, d.url, nil)
	req.Header.Add("Range", chunk.String())
	resp, err := d.client.Do(req)
	if err != nil {
		d.errChan <- err
		return
	}
	d.chunkChan <- ChunkProcessor{
		Chunk:  chunk,
		Index:  chunk.Index,
		Reader: resp.Body,
	}
}

func (d *Downloader) getFileHeader() (*FileHeader, error) {
	req, err := http.NewRequest(http.MethodHead, d.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	contentLengthStr := resp.Header.Get("Content-Length")
	if len(contentLengthStr) == 0 {
		return nil, errors.New("file size is zero")
	}
	contentLength, err := strconv.Atoi(contentLengthStr)
	if err != nil {
		return nil, err
	}
	acceptRange := resp.Header.Get("Accept-Ranges")
	if len(acceptRange) == 0 {
		acceptRange = "none"
	}
	// guess file name
	fileName := ""
	// from header
	contentDisposition := resp.Header.Get("Content-Disposition")
	if len(contentDisposition) != 0 {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err != nil {
			//TODO log
		} else {
			fileName = params["filename"]
		}
	}
	// get nothing from header
	if fileName == "" {
		fileUrl, err := url.Parse(d.url)
		if err != nil {
			return nil, err
		}
		fileUrl.RawQuery = ""
		fileName = path.Base(fileUrl.String())
	}

	return &FileHeader{
		ContentLength: contentLength,
		AcceptRanges:  acceptRange,
		FileName:      fileName,
	}, nil
}

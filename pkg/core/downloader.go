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
	errChan     chan error
	wg          sync.WaitGroup
	done        chan any
	file        *os.File
	fileHeader  *FileHeader
	tmpFile     map[int]*os.File
	*sync.RWMutex
}

func NewDownloader(url string, chunkNumber int) *Downloader {
	return &Downloader{
		url:         url,
		client:      &http.Client{},
		chunkNumber: chunkNumber,
		errChan:     make(chan error),
		wg:          sync.WaitGroup{},
		done:        make(chan any),
		tmpFile:     make(map[int]*os.File),
	}
}

func (d *Downloader) Download() error {
	if err := d.getFileHeader(); err != nil {
		return err
	}
	dst, err := os.Create(d.fileHeader.FileName)
	if err != nil {
		return err
	}
	defer dst.Close()
	d.file = dst

	numThread := d.chunkNumber
	if d.fileHeader.AcceptRanges != "bytes" {
		numThread = 1
	}
	ranges := SplitChunk(d.fileHeader.ContentLength, numThread)
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
		case err := <-d.errChan:
			return err
		case <-d.done:
			fmt.Println("done")
			return nil
		}
	}
}

func (d *Downloader) downloadChunk(chunk FileChunk) {
	defer d.wg.Done()
	req, err := http.NewRequest(http.MethodGet, d.url, nil)
	req.Header.Add("Range", chunk.String())
	resp, err := d.client.Do(req)
	if err != nil {
		d.errChan <- err
		return
	}
	defer resp.Body.Close()

	tmpFile, err := os.Create(fmt.Sprintf("%v.part%v", d.fileHeader.FileName, chunk.Index))
	if err != nil {
		d.errChan <- err
		return
	}
	defer tmpFile.Close()
	d.tmpFile[chunk.Index] = tmpFile
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		d.errChan <- err
		return
	}

	//writer := io.NewOffsetWriter(d.file, int64(chunk.From))
	//_, err = io.Copy(writer, resp.Body)
	//if err != nil {
	//	d.errChan <- err
	//	return
	//}

}

func (d *Downloader) getFileHeader() error {
	req, err := http.NewRequest(http.MethodHead, d.url, nil)
	if err != nil {
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return fmt.Errorf("status code is %v", resp.StatusCode)
	}
	contentLengthStr := resp.Header.Get("Content-Length")
	if len(contentLengthStr) == 0 {
		return errors.New("file size is zero")
	}
	contentLength, err := strconv.Atoi(contentLengthStr)
	if err != nil {
		return err
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
			return err
		}
		fileUrl.RawQuery = ""
		fileName = path.Base(fileUrl.String())
	}

	d.fileHeader = &FileHeader{
		ContentLength: contentLength,
		AcceptRanges:  acceptRange,
		FileName:      fileName,
	}

	return nil
}

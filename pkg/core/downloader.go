package core

import (
	"errors"
	"fmt"
	"fydownloader/pkg/log"
	"io"
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
	URL        string
	client     *http.Client
	FileInfo   *FileInfo
	TotalChunk int
	chunks     map[int]*Chunk
	errChan    chan error
	wg         sync.WaitGroup
}

func NewDownloader(url string, totalParts int) (*Downloader, error) {
	d := &Downloader{
		URL: url,
		client: &http.Client{
			Timeout: 0,
		},
		FileInfo:   nil,
		TotalChunk: totalParts,
		chunks:     make(map[int]*Chunk),
		errChan:    make(chan error),
	}

	if err := d.fetchFileInfo(); err != nil {
		log.S().Errorw("error when fetch file info", "url", d.URL, "error", err)
		return nil, err
	}
	return d, nil
}

type Chunk struct {
	chunkIndex  int
	from        int
	to          int
	downloaded  int
	totalSize   int
	stop        chan any
	tmpFilePath string
	tmpFile     *os.File
}

func (c *Chunk) Range() string {
	return fmt.Sprintf("bytes=%d-%d", c.from, c.to)
}

func (d *Downloader) Process() error {

	d.splitPart()
	for _, chunk := range d.chunks {
		chunk := chunk
		d.wg.Add(1)
		go d.processChunk(chunk)
	}
	go func() {
		for err := range d.errChan {
			log.S().Errorw("error received", "url", d.URL, "error", err)
			for _, chunk := range d.chunks {
				chunk.stop <- true
			}
			return
		}
	}()
	d.wg.Wait()

	return d.combineFile()
}

func (d *Downloader) processChunk(c *Chunk) {
	defer d.wg.Done()
	// create temp file
	tempFilePath := fmt.Sprintf("temp/%v.part%v", d.FileInfo.FileName, c.chunkIndex)
	c.tmpFilePath = tempFilePath
	file, err := CreateFile(tempFilePath)
	if err != nil {
		d.errChan <- err
		return
	}
	c.tmpFile = file

	req, err := http.NewRequest(http.MethodGet, d.URL, nil)
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
		select {
		case <-c.stop:
			return
		default:
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
			c.downloaded += r
		}

	}
}

func (d *Downloader) combineFile() error {
	file, err := CreateFile(d.FileInfo.FileName)
	if err != nil {
		return err
	}
	defer file.Close()
	for i := 0; i < d.TotalChunk; i++ {
		func() {
			chunk := d.chunks[i]
			_, err := chunk.tmpFile.Seek(0, 0)
			if err != nil {
				d.errChan <- err
				return
			}
			defer os.Remove(chunk.tmpFilePath)
			defer chunk.tmpFile.Close()

			writeAt := io.NewOffsetWriter(file, int64(chunk.from))
			if _, err := io.Copy(writeAt, chunk.tmpFile); err != nil {
				d.errChan <- err
				return
			}
		}()
	}
	return nil
}

func (d *Downloader) splitPart() {
	if !d.FileInfo.RangeSupported {
		chunk := &Chunk{
			chunkIndex: 0,
			from:       0,
			to:         d.FileInfo.FileSize,
			tmpFile:    nil,
		}
		d.chunks[chunk.chunkIndex] = chunk
		return
	}
	chunkSize := d.FileInfo.FileSize / d.TotalChunk
	from := 0
	for i := 0; i < d.TotalChunk; i++ {
		currSize := chunkSize
		if i == d.TotalChunk-1 {
			currSize += d.FileInfo.FileSize % d.TotalChunk
		}
		chunk := &Chunk{
			chunkIndex: i,
			from:       from,
			to:         from + currSize - 1,
			totalSize:  currSize,
			tmpFile:    nil,
			stop:       make(chan any),
		}

		from += currSize
		d.chunks[chunk.chunkIndex] = chunk
	}
}

func (d *Downloader) fetchFileInfo() error {
	req, err := http.NewRequest(http.MethodHead, d.URL, nil)
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
		fileURL, err := url.Parse(d.URL)
		if err != nil {
			return err
		}
		fileURL.RawQuery = ""
		fileName = path.Base(fileURL.String())
	}

	d.FileInfo = &FileInfo{
		FileName:       fileName,
		RangeSupported: acceptRange == "bytes",
		FileSize:       contentLength,
	}
	log.S().Infow("file info", "url", d.URL, "file_info", d.FileInfo)
	return nil
}

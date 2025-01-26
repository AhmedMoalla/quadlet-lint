package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	podmanGithubTagsURL        = "https://raw.githubusercontent.com/containers/podman/refs/tags/%s"
	quadletFileLocation        = podmanGithubTagsURL + "/pkg/systemd/quadlet/quadlet.go"
	unitfileParserFileLocation = podmanGithubTagsURL + "/pkg/systemd/parser/unitfile.go"
)

type DownloadableFile struct {
	name     string
	location string
}

var (
	Quadlet  = DownloadableFile{name: "quadlet.go", location: quadletFileLocation}
	Unitfile = DownloadableFile{name: "unitfile.go", location: unitfileParserFileLocation}
)

func (d DownloadableFile) WithVersion(version string) DownloadableFile {
	return DownloadableFile{name: d.name, location: fmt.Sprintf(d.location, version)}
}

type Downloader interface {
	Download(file DownloadableFile) (*os.File, error)
}

func NewDownloader(version string, fallbackLocations map[DownloadableFile]string) Downloader {
	fallback := make(map[string]string, len(fallbackLocations))
	for file, location := range fallbackLocations {
		fallback[file.name] = location
	}
	return FallbackDownloader{version: version, fallbackLocations: fallback}
}

type FallbackDownloader struct {
	version           string
	fallbackLocations map[string]string
}

func (f FallbackDownloader) Download(downloadableFile DownloadableFile) (*os.File, error) {
	_, hasFallback := f.fallbackLocations[downloadableFile.name]
	file, err := f.download(downloadableFile.WithVersion(f.version))
	if err == nil {
		return file, nil
	}

	if hasFallback {
		dlErr := err
		file, err := f.getFileFromFallbackLocation(downloadableFile)
		if err != nil {
			return nil, errors.Join(dlErr, err)
		}

		return file, nil
	}

	return nil, err
}

func (f FallbackDownloader) getFileFromFallbackLocation(file DownloadableFile) (*os.File, error) {
	fallback, ok := f.fallbackLocations[file.name]
	if !ok {
		return nil, fmt.Errorf("no fallback location for file %s", file.location)
	}

	if url, err := url.Parse(fallback); err == nil && url.Scheme != "" {
		return f.download(DownloadableFile{name: file.name + "fallback", location: url.String()})
	}

	localFile, err := os.Open(fallback)
	if err != nil {
		return nil, err
	}

	fileName := file.name
	tempFile, err := copyFileToTemp(fileName, localFile)
	if err != nil {
		return nil, err
	}

	return tempFile, nil
}

func (f FallbackDownloader) download(downloadableFile DownloadableFile) (*os.File, error) {
	if f.version == "" {
		return nil, errors.New("version not provided")
	}

	url := downloadableFile.location
	timeout, cancel := context.WithTimeoutCause(context.Background(), time.Second, errors.New("client timeout"))
	defer cancel()
	req, err := http.NewRequestWithContext(timeout, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download from '%s': %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status '%s' when downloading from '%s'", response.Status, url)
	}

	fileName := filepath.Base(downloadableFile.location)
	file, err := copyFileToTemp(fileName, response.Body)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func copyFileToTemp(fileName string, src io.Reader) (*os.File, error) {
	ext := filepath.Ext(fileName)
	fileName = strings.TrimSuffix(fileName, ext)
	tempFile, err := os.CreateTemp("", fileName+"-*"+ext)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file to copy the content of quadlet file: %w", err)
	}

	_, err = io.Copy(tempFile, src)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file contents: %w", err)
	}

	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	return tempFile, nil
}

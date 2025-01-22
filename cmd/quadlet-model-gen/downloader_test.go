package main

import (
	"crypto/md5"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFallbackDownloader_Download(t *testing.T) {
	server := httptest.NewServer(returnTestData(t))
	defer server.Close()

	downloader := NewDownloader("v5.3.1", nil)

	downloaded, err := downloader.Download(DownloadableFile{
		name:     "quadlet.go",
		location: server.URL + "/testdata/%s/quadlet.go",
	})
	require.NoError(t, err)

	local, err := os.Open("testdata/v5.3.1/quadlet.go")
	require.NoError(t, err)

	assertFileContentEqual(t, local, downloaded)
}

func TestFallbackDownloader_Fallback_Local(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(fail))
	defer server.Close()

	fallbackLocations := map[DownloadableFile]string{
		Quadlet: "testdata/v5.3.1/quadlet.go",
	}

	downloader := NewDownloader("v5.3.1", fallbackLocations)
	downloaded, err := downloader.Download(DownloadableFile{
		name:     "quadlet.go",
		location: server.URL + "/testdata/%s/quadlet.go",
	})
	require.NoError(t, err)

	local, err := os.Open("testdata/v5.3.1/quadlet.go")
	require.NoError(t, err)

	assertFileContentEqual(t, local, downloaded)
}

func TestFallbackDownloader_Fallback_Download(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(fail))
	defer server.Close()

	fallbackServer := httptest.NewServer(returnTestData(t))
	defer fallbackServer.Close()

	fallbackLocations := map[DownloadableFile]string{
		Quadlet: fallbackServer.URL + "/testdata/v5.3.1/quadlet.go",
	}

	downloader := NewDownloader("v5.3.1", fallbackLocations)
	downloaded, err := downloader.Download(DownloadableFile{
		name:     "quadlet.go",
		location: server.URL + "/testdata/%s/quadlet.go",
	})
	require.NoError(t, err)

	local, err := os.Open("testdata/v5.3.1/quadlet.go")
	require.NoError(t, err)

	assertFileContentEqual(t, local, downloaded)
}

func TestFallbackDownloader_Fallback_Fail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(fail))
	defer server.Close()

	fallbackLocations := map[DownloadableFile]string{
		Quadlet: "not exist",
	}

	downloader := NewDownloader("v5.3.1", fallbackLocations)
	downloaded, err := downloader.Download(DownloadableFile{
		name:     "quadlet.go",
		location: server.URL + "/testdata/%s/quadlet.go",
	})
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.Nil(t, downloaded)
}

func TestFallbackDownloader_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(timeout))
	defer server.Close()

	fallbackLocations := map[DownloadableFile]string{
		Quadlet: "testdata/v5.3.1/quadlet.go",
	}

	downloader := NewDownloader("v5.3.1", fallbackLocations)
	downloaded, err := downloader.Download(DownloadableFile{
		name:     "quadlet.go",
		location: server.URL + "/testdata/%s/quadlet.go",
	})
	require.NoError(t, err)

	local, err := os.Open("testdata/v5.3.1/quadlet.go")
	require.NoError(t, err)

	assertFileContentEqual(t, local, downloaded)
}

func assertFileContentEqual(t *testing.T, file1 *os.File, file2 *os.File) {
	require.NotNil(t, file1)
	checksum1, err := hashMD5(file1)
	require.NoError(t, err)

	require.NotNil(t, file2)
	checksum2, err := hashMD5(file2)
	require.NoError(t, err)
	assert.Equal(t, checksum1, checksum2)
}

func hashMD5(file *os.File) ([]byte, error) {
	hasher := md5.New()
	_, err := io.Copy(hasher, file)
	if err != nil {
		return nil, err
	}

	return hasher.Sum(nil), nil
}

func timeout(_ http.ResponseWriter, _ *http.Request) {
	time.Sleep(1 * time.Second)
}

func fail(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func returnTestData(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		file, err := os.Open(req.URL.Path[1:])
		require.NoError(t, err)

		w.WriteHeader(200)
		written, err := io.Copy(w, file)
		require.NoError(t, err)

		stat, err := file.Stat()
		require.NoError(t, err)

		assert.Equal(t, stat.Size(), written)
	}
}

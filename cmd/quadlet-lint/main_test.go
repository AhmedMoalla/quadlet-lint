package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/AhmedMoalla/quadlet-lint/pkg/validator/quadlet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDataDir = "testdata"

func TestUnitFiles(t *testing.T) {
	t.Parallel()

	pod := filepath.Join(testDataDir, "test.pod")

	files := findUnitFiles(testDataDir)
	assert.Len(t, files, 3)
	assert.Contains(t, files, filepath.Join(testDataDir, "test.container"))
	assert.Contains(t, files, pod)

	files = findUnitFiles(pod)
	assert.Len(t, files, 1)
	assert.Equal(t, files[0], pod)

	assert.Panics(t, func() { findUnitFiles("not-exists") })
}

func TestParseUnitFiles(t *testing.T) {
	t.Parallel()

	paths := findUnitFiles(testDataDir)

	units, errs := parseUnitFiles(paths)
	assert.Len(t, units, 2)
	assert.Len(t, errs, 1)
}

func TestValidateUnitFiles(t *testing.T) {
	t.Parallel()

	paths := findUnitFiles(testDataDir)
	units, _ := parseUnitFiles(paths)
	assert.Len(t, units, 2)

	errs := validateUnitFiles(units, *checkReferences)
	assert.Len(t, errs, 2)
	assert.Len(t, errs["test.container"], 1)
	assert.Equal(t, errs["test.container"][0].ErrorCategory, quadlet.AmbiguousImageName)
}

func TestReadInputPath(t *testing.T) {
	t.Parallel()

	executablePath, err := os.Executable()
	require.NoError(t, err)
	assert.Equal(t, filepath.Dir(executablePath), readInputPath())

	inputDir := "/my/dir"
	os.Args = []string{executablePath, inputDir}
	flag.Parse()
	assert.Equal(t, inputDir, readInputPath())
}

func TestLogSummary(t *testing.T) {
	t.Parallel()

	stdout := os.Stdout
	defer func() { os.Stdout = stdout }()

	temp := t.TempDir()
	fileStdout, err := os.Create(filepath.Join(temp, "stdout.txt"))
	require.NoError(t, err)

	os.Stdout = fileStdout

	paths := findUnitFiles(testDataDir)
	units, _ := parseUnitFiles(paths)
	assert.Len(t, units, 2)
	errs := validateUnitFiles(units, *checkReferences)
	logSummary(paths, errs)

	content, err := os.ReadFile(fileStdout.Name())
	require.NoError(t, err)
	assert.Equal(t, "Failed: 0 error(s), 1 warning(s) on 3 files.\n", string(content))
}

package quadlet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AhmedMoalla/quadlet-lint/pkg/testutils"
	"github.com/AhmedMoalla/quadlet-lint/pkg/testutils/assertions"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"github.com/stretchr/testify/require"
)

const testsDir = "testdata/tests"

func TestContainerValidator_Validate(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(testsDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(testsDir, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			t.Parallel()

			unit, asserts, err := assertions.ParseAndReadAssertions(path)
			if err != nil {
				require.NoError(t, err)
			}
			errs := Validator(append(testutils.IncludedTestUnits, unit), V.Options{CheckReferences: true}).Validate(unit)
			asserts.RunAssertions(t, errs)
		})
	}
}

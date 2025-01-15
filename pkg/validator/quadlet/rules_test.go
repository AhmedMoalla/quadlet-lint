package quadlet

import (
	"fmt"
	"testing"

	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/container"
	"github.com/AhmedMoalla/quadlet-lint/pkg/testutils"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validator = testutils.NewTestValidator(V.Options{})

func TestNetworkFormat(t *testing.T) {
	t.Parallel()

	err := NetworkFormat.ParseAndValidate("value.container:opt1=val1")
	require.ErrorIs(t, err, rules.ErrInvalidOptions)
}

func TestImageNotAmbiguous(t *testing.T) {
	t.Parallel()

	tests := []struct {
		image string
		res   bool
	}{
		// Ambiguous names
		{"fedora", false},
		{"fedora:latest", false},
		{"library/fedora", false},
		{"library/fedora:latest", false},
		{"busybox@sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a", false},
		{"busybox:latest@sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a", false},
		{"d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05", false},
		{"d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05aa", false},

		// Unambiguous names
		{"quay.io/fedora", true},
		{"docker.io/fedora", true},
		{"docker.io/library/fedora:latest", true},
		{"localhost/fedora", true},
		{"localhost:5000/fedora:latest", true},
		{"example.foo.this.may.be.garbage.but.maybe.not:1234/fedora:latest", true},
		{"docker.io/library/busybox@sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a", true},
		{"docker.io/library/busybox:latest@sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a", true},
		{"docker.io/fedora@sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a", true},
		{"sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a", true},
		{"d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a", true},
	}

	for _, test := range tests {
		t.Run(test.image, func(t *testing.T) {
			t.Parallel()

			unit := testutils.ParseString(t, "[Container]\nImage="+test.image)
			errors := ImageNotAmbiguous(validator, unit, container.Image)
			assert.Equal(t, len(errors) == 0, test.res)

			if len(errors) == 1 {
				err := errors[0]
				assert.Equal(t, validator.Name(), err.ValidatorName)
				assert.Equal(t, AmbiguousImageName, err.ErrorCategory)
				assert.Equal(t, 2, err.Line)
				assert.Equal(t, 6, err.Column)
			} else if len(errors) > 1 {
				require.FailNow(t, fmt.Sprintf("Unexpected errors: %v", errors))
			}
		})
	}
}

func TestImageNotAmbiguousArgumentAssertion(t *testing.T) {
	t.Parallel()

	unit := testutils.ParseString(t, "[Container]\nImage=test")
	assert.Nil(t, ImageNotAmbiguous(validator, unit, container.Group))
	unit = testutils.ParseString(t, "[Container]")
	assert.Nil(t, ImageNotAmbiguous(validator, unit, container.Image))
	unit = testutils.ParseString(t, "[Container]\nImage=test.build")
	assert.Nil(t, ImageNotAmbiguous(validator, unit, container.Image))
	unit = testutils.ParseString(t, "[Container]\nImage=test.image")
	assert.Nil(t, ImageNotAmbiguous(validator, unit, container.Image))
}

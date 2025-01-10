package common

import (
	"fmt"
	"testing"

	"github.com/AhmedMoalla/quadlet-lint/pkg/testutils"
)

const unitFileToTest = `
[Container]
Name=my-container
# Unknown key
Test=bla
# Empty value
Image=

# Unknown key
[Pod]
Name=my-pod
Other=bla

# Ignored group
[Service]
dazdaz=dadazdazd
`

func TestCommonValidator_Validate(t *testing.T) {
	unit := testutils.ParseString(t, unitFileToTest)

	validator := Validator()
	errs := validator.Validate(unit)
	fmt.Println(errs)
}

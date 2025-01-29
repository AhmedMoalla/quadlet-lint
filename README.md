# quadlet-lint

[![Build Status](https://github.com/AhmedMoalla/quadlet-lint/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/AhmedMoalla/quadlet-lint/actions/workflows/build.yml)
[![Test Coverage](https://raw.githubusercontent.com/AhmedMoalla/quadlet-lint/badges/.badges/main/coverage.svg)](https://github.com/AhmedMoalla/quadlet-lint/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/AhmedMoalla/quadlet-lint)](https://goreportcard.com/report/github.com/AhmedMoalla/quadlet-lint)

Podman Quadlet unit file linter

## Linting errors

## Disable linting
Disabling linting can either be done by excluding a file or can be done on different levels (file, group, key) using the 
directive `#nolint:quadlet`.

### File exclusion
To exclude a file from being linted, you can use the `-exclude` flag which takes a comma-separated list of glob patterns
as defined by Go's [`filepath.Glob`](https://pkg.go.dev/path/filepath#Glob).
#### Examples:
```bash
# Exclude a file by name
quadlet-lint -exclude file.container /dir/to/lint/recursively
# Exclude all `.container` and `.pod` files
quadlet-lint -exclude *.container,*.pod /dir/to/lint/recursively
# Exclude files under /dir/test
quadlet-lint -exclude /dir/test/* /dir/to/lint/recursively
```

### The `nolint:quadlet` directive
The directive always acts on the line that comes **_exactly_** after it. Apart from [file-level](#disable-on-file-level)
directives, any directive put before an empty line is reported as an error.
```ini
[Group]
#nolint:quadlet <- This reports an error

Key=value
```
By default, the directive will disable all error reporting on the Key or Group it has been put before, but, it is 
possible to disable only specific errors by listing them after the directive like this: `//nolint:quadlet:err1,err2`
#### Disable on file level
To disable linting on the file level, the directive needs to appear before any other line in the file and an empty line
must separate it with the beginning of the file. Listing specific errors also works.
```ini
#nolint:quadlet:err1 <- This disables 'err1' from being reported on the whole file
                     <- Notice the empty line. Without it the directive would apply to [Group]
[Group]
Key=value
```

#### Disable on key and group level
Disabling a specific error or all errors for a key or a group can be done by adding the directive before the line.
```ini
#nolint:quadlet <- Disables error reporting for the group
[Group]
Key=value

[Group2]
#nolint:quadlet <- Disables error reporting for the key
Key=value
#nolint:quadlet:err1 <- Disables 'err1' from being reported for the key
Other=value
```
#### Error format
Errors specified in the directive should be specified in the following formats:
- `validator.category.name`
- `category.name`
- `name`
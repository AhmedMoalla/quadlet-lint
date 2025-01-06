//go:generate ./quadletmodelgen_ref.sh ".generated-ref"
package main

import (
	"bufio"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	generatedRefDirName = ".generated-ref"
)

func TestQuadletModelGen(t *testing.T) {
	t.Parallel()

	generatedRefDir, err := os.Open(generatedRefDirName)
	if err != nil && os.IsNotExist(err) {
		t.Fatal(errors.Join(err, errors.New("'go generate' was not run before starting the test")))
	}

	podmanVersion, err := getPodmanVersionFromGeneratedComment(generatedRefDirName + "/groups.go")
	if err != nil {
		t.Fatal(err)
	}

	unitfile, err := os.Open("testdata/" + podmanVersion + "/unitfile.go")
	if err != nil {
		t.Fatal(err)
	}

	quadlet, err := os.Open("testdata/" + podmanVersion + "/quadlet.go")
	if err != nil {
		t.Fatal(err)
	}

	parseAndGenerateFiles(quadlet, unitfile)
	defer os.RemoveAll(generatedDirName)
	generatedDir, err := os.Open(generatedRefDirName)
	if err != nil && os.IsNotExist(err) {
		t.Fatal(err)
	}

	err = compareFiles(t, generatedRefDir, generatedDir)
	if err != nil {
		t.Fatal(err)
	}
}

type structDecl struct {
	Name   string
	Fields map[string]string
}

type structInstance struct {
	StructType string
	Fields     map[string]any
}

type testInspectionResult struct {
	StructDecls []structDecl
	Variables   map[string]any
}

func compareFiles(t *testing.T, generatedRefDir *os.File, generatedDir *os.File) error {
	t.Helper()

	refResult, err := inspectDir(t, generatedRefDir)
	if err != nil {
		return err
	}

	result, err := inspectDir(t, generatedDir)
	if err != nil {
		return err
	}

	assert.Equal(t, refResult, result)
	return nil
}

func inspectDir(t *testing.T, dir *os.File) (map[string]testInspectionResult, error) {
	t.Helper()

	files, err := listAllFiles(dir.Name())
	if err != nil {
		return nil, err
	}

	result := make(map[string]testInspectionResult, len(files))
	for _, file := range files {
		fileResult, err := inspectFile(t, file)
		if err != nil {
			return nil, err
		}

		_, filename := filepath.Split(file)
		result[filename] = fileResult
	}

	return result, nil
}

func inspectFile(t *testing.T, file string) (testInspectionResult, error) {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	if err != nil {
		return testInspectionResult{}, err
	}

	result := testInspectionResult{
		StructDecls: make([]structDecl, 0, 1),
		Variables:   make(map[string]any, 20),
	}
	ast.Inspect(parsed, func(n ast.Node) bool {
		node, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}

		switch node.Tok {
		case token.TYPE:
			for _, spec := range node.Specs {
				if structDecl, ok := extractStructDecl(spec); ok {
					result.StructDecls = append(result.StructDecls, structDecl)
				}
			}
		case token.VAR:
			for _, spec := range node.Specs {
				if varName, varValue, err := extractVariable(spec); err == nil {
					result.Variables[varName] = varValue
				} else {
					t.Fatal(err)
				}
			}
		default:
			return true
		}

		return true
	})

	return result, nil
}

func extractVariable(spec ast.Spec) (string, any, error) {
	valueSpec, ok := spec.(*ast.ValueSpec)
	if !ok {
		return "", nil, fmt.Errorf(
			"inspection does not support VARs specs other then *ast.ValueSpec. Found %T", spec)
	}

	compositeSpec, ok := valueSpec.Values[0].(*ast.CompositeLit)
	if !ok {
		return "", nil, fmt.Errorf(
			"inspection does not support VARs value specs other then *ast.CompositeLit. Found %T", valueSpec.Values[0])
	}

	if _, ok := compositeSpec.Type.(*ast.MapType); ok {
		mapValue := computeMapField(compositeSpec)
		if mapValue != nil {
			return valueSpec.Names[0].Name, mapValue, nil
		}
	}

	fields := make(map[string]any)
	for _, elt := range compositeSpec.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return "", nil, fmt.Errorf(
				"inspection does not support VARs composite literal specs other then *ast.KeyValueExpr. Found %T", elt)
		}

		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			return "", nil, fmt.Errorf("expected key to be of type *ast.Ident. Found %T", kv.Key)
		}

		fields[key.Name] = types.ExprString(kv.Value)
	}

	return valueSpec.Names[0].Name, structInstance{
		StructType: types.ExprString(compositeSpec.Type),
		Fields:     fields,
	}, nil
}

func extractStructDecl(spec ast.Spec) (structDecl, bool) {
	typeSpec, _ := spec.(*ast.TypeSpec)
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return structDecl{}, false
	}

	fields := make(map[string]string, len(structType.Fields.List))
	for _, field := range structType.Fields.List {
		fields[field.Names[0].Name] = types.ExprString(field.Type)
	}

	return structDecl{Name: typeSpec.Name.Name, Fields: fields}, true
}

// computeMapField can handle nested maps with string keys
func computeMapField(spec *ast.CompositeLit) any {
	result := make(map[string]any, len(spec.Elts))
	for _, elt := range spec.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return nil
		}

		key, ok := kv.Key.(*ast.BasicLit)
		if !ok {
			return nil
		}

		if value, ok := kv.Value.(*ast.CompositeLit); ok {
			result[key.Value] = computeMapField(value)
		} else if value, ok := kv.Value.(*ast.SelectorExpr); ok {
			result[key.Value] = types.ExprString(value)
		} else {
			return nil
		}
	}

	return result
}

func listAllFiles(dir string) ([]string, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(dirEntries))
	for _, entry := range dirEntries {
		if entry.IsDir() {
			subDirFiles, err := listAllFiles(filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, err
			}

			files = append(files, subDirFiles...)
		} else {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

func getPodmanVersionFromGeneratedComment(generatedFile string) (string, error) {
	file, err := os.Open(generatedFile)
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(file)
	headComment, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	commentSections := strings.SplitN(headComment, ";", 3)
	if len(commentSections) != 3 {
		return "", fmt.Errorf("could not find Podman version. invalid head comment: '%s'", headComment)
	}

	podmanVersionKV := strings.SplitN(commentSections[1], "=", 2)
	if len(podmanVersionKV) != 2 {
		return "", fmt.Errorf("could not find Podman version. invalid head comment: '%s'", commentSections[1])
	}

	return podmanVersionKV[1], nil
}

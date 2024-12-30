package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

const (
	generatedMarkerComment = "// Code generated by \"quadlet-model-gen\"; PodmanVersion=%s; DO NOT EDIT.\n"
	generatedFilesPerm     = 0777
)

func generateSourceFiles(data sourceFileData) error {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	outputDir := filepath.Join(workingDir, "generated")
	err = os.Mkdir(outputDir, generatedFilesPerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	err = generateFile(outputDir, data, "groups.go", groupsFile)
	if err != nil {
		return err
	}

	for group := range data.fieldsByGroup {
		groupLower := strings.ToLower(group)
		path := fmt.Sprintf("%s/%s.go", groupLower, groupLower)
		err = generateFile(outputDir, data, path, groupFile(group))
		if err != nil {
			return err
		}
	}

	err = generateFile(outputDir, data, "lookup/lookup.go", lookupFuncFile)
	if err != nil {
		return err
	}

	return nil
}

type FileGenerator = func(*bytes.Buffer, sourceFileData)

func generateFile(outputDir string, data sourceFileData, filename string, generateFileContent FileGenerator) error {
	dir, filename := filepath.Split(filename)
	fileDir := filepath.Join(outputDir, dir)
	if len(dir) > 0 {
		err := os.MkdirAll(fileDir, generatedFilesPerm)
		if err != nil && !os.IsExist(err) {
			return err
		}
	}

	path := filepath.Join(fileDir, filename)
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	sb := bytes.Buffer{}
	sb.WriteString(fmt.Sprintf(generatedMarkerComment, *podmanVersion))
	generateFileContent(&sb, data)

	formatted, err := format.Source(sb.Bytes())
	if err != nil {
		return err
	}

	_, err = file.Write(formatted)
	if err != nil {
		return err
	}

	return nil
}

func groupsFile(b *bytes.Buffer, data sourceFileData) {
	fieldsByGroup := data.fieldsByGroup
	b.WriteString("package model\n\n")
	b.WriteString("import (\n")
	for group := range fieldsByGroup {
		b.WriteString(fmt.Sprintf("\t\"%s/%s\"\n", baseModelPackageName, strings.ToLower(group)))
	}
	b.WriteString("\tM \"github.com/AhmedMoalla/quadlet-lint/pkg/model\"\n")
	b.WriteString(")\n\n")

	b.WriteString("type Groups struct {\n")
	for group := range fieldsByGroup {
		b.WriteString(fmt.Sprintf("\t%s %s.G%s\n", group, strings.ToLower(group), group))
	}
	b.WriteString("}\n\n")

	b.WriteString("var Fields =  map[string]map[string]M.Field{\n")
	for group, fields := range fieldsByGroup {
		b.WriteString(fmt.Sprintf("\t\"%s\": {\n", group))
		for _, field := range fields {
			b.WriteString(fmt.Sprintf("\t\t\"%s\": %s.%s,\n", field.Key, strings.ToLower(group), field.Key))
		}
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n")
}

func groupFile(group string) FileGenerator {
	return func(b *bytes.Buffer, data sourceFileData) {
		fieldsByGroup := data.fieldsByGroup
		b.WriteString(fmt.Sprintf("package %s\n\n", strings.ToLower(group)))
		if len(fieldsByGroup[group]) > 0 {
			b.WriteString("import (\n")
			b.WriteString("\tM \"github.com/AhmedMoalla/quadlet-lint/pkg/model\"\n")
			b.WriteString("\tV \"github.com/AhmedMoalla/quadlet-lint/pkg/validator\"\n")
			b.WriteString("\t  \"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/lookup\"\n")
			b.WriteString(")\n\n")
		}

		b.WriteString(fmt.Sprintf("type G%s struct {\n", group))
		for _, field := range fieldsByGroup[group] {
			b.WriteString(fmt.Sprintf("\t%s []V.Rule\n", field.Key))
		}
		b.WriteString("}\n\n")

		b.WriteString("var (\n")
		for _, field := range fieldsByGroup[group] {
			fieldStr := fmt.Sprintf("M.Field{Group: \"%s\", Key: \"%s\", LookupFunc: lookup.%s }",
				field.Group, field.Key, field.LookupFunc.Name)
			b.WriteString(fmt.Sprintf("\t%s = %s\n", field.Key, fieldStr))
		}
		b.WriteString(")\n")
	}
}

func lookupFuncFile(b *bytes.Buffer, data sourceFileData) {
	lookupFuncs := data.lookupFuncs
	b.WriteString("package lookup\n\n")
	b.WriteString("type LookupFunc struct {\n")
	b.WriteString("\tName     string\n")
	b.WriteString("\tMultiple bool\n")
	b.WriteString("}\n\n")

	b.WriteString("var (\n")
	b.WriteString("\nUnsupportedLookup = LookupFunc{Name: \"UnsupportedLookup\"}\n")
	for name, f := range lookupFuncs {
		b.WriteString(fmt.Sprintf("\t%s = LookupFunc{Name: \"%s\", Multiple: %t}\n", name, name, f.Multiple))
	}
	b.WriteString(")\n")
}

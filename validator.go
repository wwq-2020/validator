package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

const (
	suffix = "_validator.go"
)

var (
	src          string
	dst          string
	mod          string
	recursive    bool
	curStruct    string
	curPkg       string
	srcPkg       string
	curColumns   []string
	curValues    []string
	curFields    []field
	curBuf       = bytes.NewBuffer(nil)
	curImported  bool
	tableNameReg = regexp.MustCompile(`table:(\w+)`)
)

type field struct {
	IsEmpty string
	Error   string
}

type tpl struct {
	Name   string
	Fields []field
}

type walker func(ast.Node) bool

func (w walker) Visit(node ast.Node) ast.Visitor {
	if w(node) {
		return w
	}
	return nil
}

func init() {
	flag.StringVar(&src, "src", ".", "-src=.")
	flag.BoolVar(&recursive, "recursive", false, "-recursive=true")
	flag.Parse()
}

func main() {
	if src == "" {
		flag.PrintDefaults()
		return
	}
	if dst != "" && mod == "" {
		flag.PrintDefaults()
		return
	}
	info, err := os.Stat(src)
	if err != nil {
		log.Fatalf("failed to stat src:%s, err:%#v", src, err)
	}

	switch info.IsDir() {
	case true:
		handleDirMode(src)
	default:
		handleFileMode()
	}
}

func handleDirMode(src string) {
	srcPkg = ""
	curPkg = ""
	p, err := build.ImportDir(src, 0)
	if err != nil {
		return
	}
	srcPkg = p.Name
	curPkg = p.Name
	if err := filepath.Walk(src, func(curPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if recursive {
				return nil
			}
			return filepath.SkipDir
		}
		name := info.Name()

		if strings.HasSuffix(name, suffix) || !strings.HasSuffix(name, ".go") {
			return nil
		}

		fs := token.NewFileSet()
		file, err := parser.ParseFile(fs, curPath, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		handleFile(curPath, file)
		return nil
	}); err != nil {
		log.Fatalf("fail to walk path:%s, err:%#v", src, err)
	}
}

func handleFileMode() {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, src, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("failed to parse src:%s, err:%#v", src, err)
	}
	baseDir := path.Dir(src)
	p, err := build.ImportDir(baseDir, 0)
	if err != nil {
		return
	}
	srcPkg = p.Name
	curPkg = p.Name
	handleFile(src, file)
}

func handleFile(curPath string, file *ast.File) {
	baseDir := dst
	curDir := path.Dir(curPath)
	name := path.Base(curPath)
	if baseDir == "" {
		baseDir = curDir
	}
	fullPath := filepath.Join(baseDir, strings.Replace(name, ".go", suffix, -1))
	addImports(curDir)
	ast.Walk(walker(func(node ast.Node) bool {
		switch v := node.(type) {
		case *ast.GenDecl:
			if v.Tok == token.IMPORT {
				return false
			}
			return true
		case *ast.FuncDecl:
			return false
		case *ast.TypeSpec:
			handleStruct(v)
			reset()
			return false
		case *ast.ValueSpec:
			return false
		}
		return true
	}), file)
	if curBuf.Len() != 0 {
		ioutil.WriteFile(fullPath, curBuf.Bytes(), 0644)
	}
	curBuf.Reset()
}

func handleStruct(node *ast.TypeSpec) {
	st, ok := node.Type.(*ast.StructType)
	if !ok {
		return
	}

	if st.Fields == nil || len(st.Fields.List) == 0 {
		return
	}
	curStruct = node.Name.Name
	handleField(st.Fields.List)
	execTpl()
}

func handleField(fields []*ast.Field) {
	for _, each := range fields {
		if each.Tag == nil {
			continue
		}
		trimedValue := strings.Trim(each.Tag.Value, "`")
		tagValue := reflect.StructTag(trimedValue).Get("validator")
		tagValues := strings.Split(tagValue, ",")
		if len(tagValues) == 0 {
			continue
		}
		ident, ok := each.Type.(*ast.Ident)
		if !ok {
			continue
		}
		if len(tagValues) > 0 && tagValues[0] == "required" {
			switch ident.Name {
			case "string":
				curFields = append(curFields, field{IsEmpty: fmt.Sprintf(`src.%s == ""`, each.Names[0].Name), Error: fmt.Sprintf(`errors.New("field %s can't be empty")`, each.Names[0].Name)})
			case "int", "int8", "int32", "int64", "uint", "uint32", "uint64", "uint8", "float32", "float64":
				curFields = append(curFields, field{IsEmpty: fmt.Sprintf(`src.%s == 0`, each.Names[0].Name), Error: fmt.Sprintf(`errors.New("field %s can't be empty")`, each.Names[0].Name)})
			}
		}
	}
}

func reset() {
	curStruct = ""
	curColumns = nil
	curValues = nil
	curFields = nil

}

func execTpl() error {
	tpl := tpl{Name: curStruct, Fields: curFields}
	t, err := template.New("sqlutil").Funcs(template.FuncMap{
		"raw":   raw,
		"title": title,
	}).Parse(tplStr)
	if err != nil {
		return err
	}
	if err := t.Execute(curBuf, tpl); err != nil {
		return err
	}
	return nil
}

func raw(prev string) template.HTML {
	return template.HTML(prev)
}

func raw2(prev string) template.HTML {
	return template.HTML(prev)
}

func title(prev string) string {
	return strings.Title(strings.Replace(prev, "id", "ID", -1))
}

func addImports(curDir string) {
	prefix := fmt.Sprintf(`package %s
	import(
		"errors"`, curPkg)
	var mid string
	if dst != "" {
		mid = fmt.Sprintf(`
				%s "%s"
			`, curPkg, path.Join(mod, curDir))
	}
	suffix := `
	)`
	io.WriteString(curBuf, fmt.Sprintf("%s%s%s", prefix, mid, suffix))
}

func isBuiltIn(typ string) bool {
	switch typ {
	case "string", "int", "int8", "int32", "int64", "uint", "uint32", "uint64", "uint8", "float32", "float64":
		return true
	default:
		return false
	}
}

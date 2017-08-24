package main

import (
	"path/filepath"
	"os"
	"flag"
	"strings"
	"log"
	"go/token"
	"go/parser"
	"github.com/govend/govend/imports"
	"io/ioutil"
	"go/ast"
	"gopkg.in/libgit2/git2go.v24"
)

var oldRepo, newRepo string
var step int

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func visit(path string, file os.FileInfo, err error) error {
	if file.IsDir() || !strings.HasSuffix(path, ".go") {
		return nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return err
	}

	matchedImports := []*ast.ImportSpec{}

	for _, i := range f.Imports {
		if !imports.Valid(i.Path.Value) { //Pass invalid imports
			continue
		} else if strings.HasPrefix(i.Path.Value, `"`+oldRepo) {
			matchedImports = append(matchedImports, i)
		}
	}

	if len(matchedImports) == 0 {
		return nil
	}

	cont, _ := ioutil.ReadFile(path)
	text := string(cont)
	offset := 0

	for _, v := range matchedImports {
		s := int(v.Path.ValuePos) + offset
		f := s + len(v.Path.Value) - 2
		text = text[:s] + strings.Replace(text[s:f], oldRepo, newRepo, 1) + text[f:]

		offset += step
	}

	err = ioutil.WriteFile(path, []byte(text), file.Mode())
	if err != nil {
		panic(err)
	}

	return nil
}

func main() {
	var path string
	var err error

	flag.Parse()
	if len(flag.Args()) == 3 {
		path = flag.Arg(0)
		if path == "" {
			path = "./"
		}
		oldRepo = flag.Arg(1)
		newRepo = flag.Arg(2)
	} else if len(flag.Args()) == 2 {
		path = "./"
		oldRepo = flag.Arg(0)
		newRepo = flag.Arg(1)
	} else {
		log.Fatalln("Invalid args count")
	}

	// set char offset step
	step = len(newRepo) - len(oldRepo)

	repo, err := git.OpenRepository(path)
	failOnError(err, "Failed to open git repository")

	config, err := repo.Config()
	failOnError(err, "Failed to read repository config")

	url, err := config.LookupString("remote.origin.url")
	failOnError(err, "Failed to load repository url")

	// log url
	log.Println(url)

	err = filepath.Walk(path, visit)
	failOnError(err, "Failed to walk directory")
}

package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/einbert-xeride/goreexport/reexport"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/modfile"
)

var (
	Package   string
	Directory string
	Output    string
)

func GoPath() string {
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		fmt.Fprintf(os.Stderr, "Error: GOPATH not provided\n")
		os.Exit(255)
	}
	return goPath
}

func guessPackage(dir string) (string, error) {
	modPath := path.Join(dir, "go.mod")
	file, err := os.Open(modPath)
	if err != nil {
		return guessPackageInGoPath(dir)
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return guessPackageInGoPath(dir)
	}
	mod, err := modfile.Parse(modPath, data, nil)
	if err != nil {
		return "", errors.WithMessage(err, "module file parse failed")
	}
	return mod.Module.Mod.Path, nil
}

func guessPackageInGoPath(dir string) (string, error) {
	srcPath := path.Join(GoPath(), "src")
	if strings.HasPrefix(dir, srcPath) {
		pkg := dir[len(srcPath):]
		if pkg == "" {
			return "", errors.New("dir is just GOPATH")
		}
		return pkg, nil
	}
	return "", errors.New("dir not in GOPATH, and has no readable go.mod in it")
}

func main() {
	flag.StringVar(&Directory, "dir", "", "Directory of the package")
	flag.StringVar(&Package, "pkg", "", "Enforce package import path, will be generated according to GOPATH and go.mod")
	flag.StringVar(&Output, "out", "", "Output file, or stdout if not provided")
	flag.Parse()

	if Directory == "" && Package == "" {
		fmt.Fprintf(os.Stderr, "Error: both -dir and -pkg are not provided\n")
		os.Exit(255)
	}

	if Directory == "" && Package != "" {
		Directory = path.Join(GoPath(), "src", Package)
	}

	if Directory != "" && Package == "" {
		var err error
		Package, err = guessPackage(Directory)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			os.Exit(255)
		}
	}

	pkgs, err := parser.ParseDir(token.NewFileSet(), Directory, func(info os.FileInfo) bool {
		return info.IsDir() || !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.AllErrors)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Package parse failed: %s\n", err.Error())
		os.Exit(255)
	}

	if len(pkgs) != 1 {
		fmt.Fprintf(os.Stderr, "Error: %d packages in %s, unexpected\n", len(pkgs), Directory)
		os.Exit(255)
	}

	var pkg *ast.Package
	for _, pkg = range pkgs {
		break
	}

	re := reexport.New(pkg, Package)
	out, err := re.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(255)
	}

	file := os.Stdout
	if Output != "" && Output != "-" {
		file, err = os.Create(Output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			os.Exit(255)
		}
		defer file.Close()
	}
	_, err = fmt.Fprintf(file, "%s", out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(255)
	}
}

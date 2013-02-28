package main

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	// "strings"
)

func init() {
	cmdVacuum.Run = runVacuum
	addVacuumFlags(cmdVacuum)
}

var vacuumL bool

var cmdVacuum = &Command{
	UsageLine: "vacuum [-l]",
}

func runVacuum(cmd *Command, args []string) {
	if len(config.Entrances) == 0 {
		errorfExit("Entrances packages required")
	}
	for _, pkgPath := range config.Entrances {
		buildpkg, err := build.Default.Import(pkgPath, "", 0)
		if err != nil {
			errorfExit("can not import %s, %s", buildpkg, err)
		}

		parsePkg(buildpkg.Dir)
	}

}

func addVacuumFlags(cmd *Command) {
	cmd.Flag.BoolVar(&vacuumL, "l", false, "")
}

func parsePkg(dir string) {
	fset := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		errorfExit("can not parse package %s, %s", dir, err)
	}

	w := &walker{fset: fset}
	for _, pkg := range astPkgs {
		ast.Print(fset, pkg)
		ast.Walk(w, pkg)
	}
}

type walker struct {
	fset *token.FileSet
}

func (w *walker) Visit(node ast.Node) ast.Visitor {
	return w
}

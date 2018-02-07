package main

import (
	"strings"

	"github.com/flosch/pongo2"
	"github.com/gosimple/slug"
)

func init() {
	pongo2.RegisterFilter("groovy_save", tplGroovyFileSave)
	pongo2.RegisterFilter("slugify", tplSlugify)
}

func tplSlugify(in, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(slug.Make(in.String())), nil
}

func tplGroovyFileSave(in, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(strings.Replace(in.String(), "-", "_", -1)), nil
}

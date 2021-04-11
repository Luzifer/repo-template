package main

import (
	"errors"
	"strings"

	"github.com/flosch/pongo2/v4"
	"github.com/google/go-github/v34/github"
	"github.com/gosimple/slug"
)

func init() {
	pongo2.RegisterFilter("has_topic", tplRepoHasTopic)
	pongo2.RegisterFilter("groovy_save", tplGroovyFileSave)
	pongo2.RegisterFilter("slugify", tplSlugify)
}

func tplSlugify(in, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(slug.Make(in.String())), nil
}

func tplGroovyFileSave(in, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(strings.Replace(in.String(), "-", "_", -1)), nil
}

func tplRepoHasTopic(in, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	repo, ok := in.Interface().(*github.Repository)
	if !ok {
		return nil, &pongo2.Error{
			Sender:    "filter:has_topic",
			OrigError: errors.New("Input was no github.Repository"),
		}
	}

	topic := param.String()

	for _, t := range repo.Topics {
		if t == topic {
			return pongo2.AsValue(true), nil
		}
	}

	return pongo2.AsValue(false), nil
}

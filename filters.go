package main

import (
	"context"
	"strings"

	"github.com/Luzifer/go_helpers/str"
	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
)

type filterFunc func(*github.Repository) bool

var filters = map[string]filterFunc{
	"archived":     filterArchived,
	"dockerfile":   filterDockerfile,
	"fork":         filterFork,
	"make-jenkins": filterMakeJenkins,
	"public":       filterPublic,
}

func filterArchived(repo *github.Repository) bool { return repo.Archived != nil && *repo.Archived }

func filterDockerfile(repo *github.Repository) bool {
	ctx := context.Background()
	_, _, resp, err := client.Repositories.GetContents(ctx, *repo.Owner.Login, *repo.Name, "Dockerfile", nil)
	if err != nil {
		if resp.StatusCode == 404 {
			return false
		}

		log.WithError(err).Error("Error while looking for Dockerfile")
		return false
	}

	return true
}

func filterFork(repo *github.Repository) bool { return repo.Fork != nil && *repo.Fork }

func filterMakeJenkins(repo *github.Repository) bool {
	ctx := context.Background()
	fc, _, resp, err := client.Repositories.GetContents(ctx, *repo.Owner.Login, *repo.Name, "Makefile", nil)
	if err != nil {
		if resp.StatusCode == 404 {
			return false
		}

		log.WithError(err).Error("Error while looking for Dockerfile")
		return false
	}

	if fc.Content == nil {
		log.Error("File content had no content")
		return false
	}

	if str.StringInSlice("jenkins:", strings.Split(*fc.Content, "\n")) {
		return true
	}

	return false
}

func filterPublic(repo *github.Repository) bool { return repo.Private != nil && !*repo.Private }

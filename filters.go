package main

import (
	"context"

	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
)

type filterFunc func(*github.Repository) bool

var filters = map[string]filterFunc{
	"fork":       filterFork,
	"dockerfile": filterDockerfile,
	"public":     filterPublic,
}

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

func filterFork(repo *github.Repository) bool   { return repo.Fork != nil && *repo.Fork }
func filterPublic(repo *github.Repository) bool { return repo.Private != nil && !*repo.Private }

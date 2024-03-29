package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"strings"

	"github.com/google/go-github/v34/github"
	log "github.com/sirupsen/logrus"
)

type filterFunc func(*github.Repository) bool

var filters = map[string]filterFunc{
	"archived":     filterArchived,
	"dockerfile":   filterDockerfile,
	"fork":         filterFork,
	"has-file":     filterHasFile,
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

func filterHasFile(repo *github.Repository) bool {
	ctx := context.Background()
	_, _, resp, err := client.Repositories.GetContents(ctx, *repo.Owner.Login, *repo.Name, cfg.FilterHasFile, nil)
	if err != nil {
		if resp.StatusCode == 404 {
			return false
		}

		log.WithError(err).Error("Error while looking for file")
		return false
	}

	return true
}

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

	scanner := bufio.NewScanner(base64.NewDecoder(base64.StdEncoding, strings.NewReader(*fc.Content)))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "jenkins:") {
			return true
		}
	}

	return false
}

func filterPublic(repo *github.Repository) bool { return repo.Private != nil && !*repo.Private }

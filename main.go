package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"golang.org/x/oauth2"

	"github.com/Luzifer/rconfig"
	"github.com/flosch/pongo2"
	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
)

var (
	cfg = struct {
		Filters        []string `flag:"filter,f" default:"" description:"Filters to match the repos against"`
		GithubToken    string   `flag:"token" default:"" env:"GITHUB_TOKEN" description:"Token to access Github API"`
		LogLevel       string   `flag:"log-level" default:"info" description:"Log level for output (debug, info, warn, error, fatal)"`
		NameRegex      string   `flag:"name-regex" default:".*" description:"Regex to match the name against"`
		Output         string   `flag:"out,o" default:"-" description:"File to write to (- = stdout)"`
		Template       string   `flag:"template" default:"" description:"Template file to use for rendering" validate:"nonzero"`
		VersionAndExit bool     `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	client *github.Client

	version = "dev"
)

func init() {
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline options: %s", err)
	}

	if l, err := log.ParseLevel(cfg.LogLevel); err != nil {
		log.WithError(err).Fatalf("Could not parse log level")
	} else {
		log.SetLevel(l)
	}

	if cfg.VersionAndExit {
		fmt.Printf("repo-template %s\n", version)
		os.Exit(0)
	}
}

func main() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.GithubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client = github.NewClient(tc)

	repos, err := fetchRepos()
	if err != nil {
		log.WithError(err).Fatal("Error while fetching repos")
	}

	for _, repo := range repos {
		if !regexp.MustCompile(cfg.NameRegex).MatchString(*repo.FullName) {
			continue
		}

		skip := false

		for _, f := range cfg.Filters {
			if f == "" {
				continue
			}

			var (
				inverse = false
				filter  = f
			)

			if strings.HasPrefix(filter, "no-") {
				inverse = true
				filter = filter[3:]
			}

			if filters[filter](repo) == inverse {
				log.WithFields(log.Fields{
					"filter": filter,
					"repo":   *repo.FullName,
				}).Debug("Repo was filtered")
				skip = true
			}
		}

		if skip {
			continue
		}

		log.WithFields(log.Fields{
			"repo":    *repo.FullName,
			"private": *repo.Private,
			"fork":    *repo.Fork,
		}).Print("Found repo")

		if err := render(repo); err != nil {
			log.WithError(err).Error("Unable to render output")
			continue
		}
	}
}

func fetchRepos() ([]*github.Repository, error) {
	var (
		ctx   = context.Background()
		repos = []*github.Repository{}
	)

	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		rs, res, err := client.Repositories.List(ctx, "", opt)

		if err != nil {
			return nil, err
		}

		repos = append(repos, rs...)

		if res.NextPage == 0 {
			break
		}
		opt.Page = res.NextPage
	}

	return repos, nil
}

func render(repo *github.Repository) error {
	var outFile io.Writer
	if cfg.Output == "-" {
		outFile = os.Stdout
	} else {
		outName, err := getOutfile(repo)
		if err != nil {
			return err
		}

		log.WithFields(log.Fields{
			"repo":    *repo.FullName,
			"outName": outName,
		}).Debug("")

		f, err := os.Create(outName)
		if err != nil {
			return err
		}
		defer f.Close()
		outFile = f
	}

	tplRaw, err := ioutil.ReadFile(cfg.Template)
	if err != nil {
		return err
	}

	tpl, err := pongo2.FromString(string(tplRaw))
	if err != nil {
		return err
	}

	return tpl.ExecuteWriter(pongo2.Context{
		"repo": repo,
	}, outFile)
}

func getOutfile(repo *github.Repository) (string, error) {
	tpl, err := pongo2.FromString(cfg.Output)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)

	err = tpl.ExecuteWriter(pongo2.Context{
		"repo": repo,
	}, buf)
	return buf.String(), err
}

func simpleReplace(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}

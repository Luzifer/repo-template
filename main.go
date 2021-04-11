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

	"github.com/Luzifer/go_helpers/v2/str"
	"github.com/Luzifer/rconfig/v2"
	"github.com/flosch/pongo2/v4"
	"github.com/google/go-github/v34/github"
	log "github.com/sirupsen/logrus"
)

var (
	cfg = struct {
		Blacklist      []string `flag:"blacklist,b" default:"" description:"Repos to ignore even when matched through filters"`
		ExpandMatches  bool     `flag:"expand-matches" default:"false" description:"Replace matched repos with their full version"`
		Filters        []string `flag:"filter,f" default:"" description:"Filters to match the repos against"`
		GithubToken    string   `flag:"token" default:"" env:"GITHUB_TOKEN" description:"Token to access Github API"`
		LogLevel       string   `flag:"log-level" default:"info" description:"Log level for output (debug, info, warn, error, fatal)"`
		NameRegex      string   `flag:"name-regex" default:".*" description:"Regex to match the name against"`
		Output         string   `flag:"out,o" default:"-" description:"File to write to (- = stdout)"`
		Template       string   `flag:"template" default:"" description:"Template file to use for rendering" validate:"nonzero"`
		TopicFilter    []string `flag:"topic,t" default:"" description:"Filter by topic (Format: 'topic' to include, '-topic' to exclude)"`
		VersionAndExit bool     `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	client *github.Client

	version = "dev"
)

func init() {
	rconfig.AutoEnv(true)
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

		if str.StringInSlice(*repo.FullName, cfg.Blacklist) {
			continue
		}

		if !matchTopicFilter(repo) {
			continue
		}

		if !matchFilters(repo) {
			continue
		}

		if cfg.ExpandMatches {
			if err := expandRepo(repo); err != nil {
				log.WithError(err).Error("Unable to expand repo")
			}
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

func expandRepo(repo *github.Repository) error {
	ctx := context.Background()
	r, _, err := client.Repositories.Get(ctx, *repo.Owner.Login, *repo.Name)
	if err != nil {
		return err
	}

	*repo = *r
	return nil
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

func matchFilters(repo *github.Repository) bool {
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
			return false
		}
	}

	return true
}

func matchTopicFilter(repo *github.Repository) bool {
	for _, topic := range cfg.TopicFilter {
		if topic == "" {
			continue
		}

		negate := topic[0] == '-'
		if negate {
			topic = topic[1:]
		}

		if str.StringInSlice(topic, repo.Topics) != negate {
			return true
		}
	}

	return false
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

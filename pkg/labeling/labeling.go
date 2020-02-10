package labeling

import (
	"github.com/google/go-github/v29/github"
	log "github.com/sirupsen/logrus"
)

type Repository interface {
	OpenPullRequests() ([]*github.PullRequest, error)
	PullRequestModifiedFiles(number int) ([]*github.CommitFile, error)
	AddLabelsToPullRequest(number int, labels []string) error
	Owner() string
	Name() string
}

type Mappings interface {
	MatchedLabels([]*github.CommitFile) (labels []string)
}

type Labeler struct {
	DryRun bool
	Repository
	Mappings
}

func New(r Repository, m Mappings) *Labeler {
	return &Labeler{
		Repository: r,
		Mappings:   m,
	}
}

func (l *Labeler) ApplyLabels() (err error) {
	pulls, err := l.OpenPullRequests()
	if err != nil {
		return err
	}
	return l.applyLabels(pulls)
}

func (l *Labeler) applyLabels(pulls []*github.PullRequest) error {
	for _, pull := range pulls {
		if pull.Number == nil {
			continue
		}

		expected, err := l.expectedLabels(pull)
		if err != nil {
			return err
		}

		if !shouldApplyLabels(expected, pull.Labels) {
			continue
		}

		log.Infof("PR %s/%s#%d should have following labels: %v (%s)", l.Owner(), l.Name(), *pull.Number, expected, safeString(pull.Title))
		if l.DryRun {
			continue
		}
		if err := l.AddLabelsToPullRequest(*pull.Number, expected); err != nil {
			return err
		}
	}
	return nil
}

func (l *Labeler) expectedLabels(pull *github.PullRequest) ([]string, error) {
	files, err := l.PullRequestModifiedFiles(*pull.Number)
	if err != nil {
		return nil, err
	}
	return l.MatchedLabels(files), nil
}

func shouldApplyLabels(expected []string, existing []*github.Label) bool {
	switch {
	case len(expected) == 0:
		return false
	case len(expected) > len(existing):
		return true
	}
	return len(difference(expected, existing)) > 0
}

func difference(expected []string, existing []*github.Label) []string {
	existingSet := make(map[string]struct{}, len(existing))
	for _, v := range existing {
		existingSet[safeString(v.Name)] = struct{}{}
	}
	var diff []string
	for _, v := range expected {
		if _, ok := existingSet[v]; !ok {
			diff = append(diff, v)
		}
	}
	return diff
}

func safeString(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

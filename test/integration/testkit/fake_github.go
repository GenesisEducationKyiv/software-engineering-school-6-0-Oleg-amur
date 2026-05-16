//go:build integration

package testkit

import "context"

type FakeGithubClient struct {
	Exists   map[string]bool
	Tags     map[string]string
	CheckErr map[string]error
	TagErr   map[string]error
}

func NewFakeGithubClient() *FakeGithubClient {
	return &FakeGithubClient{
		Exists: map[string]bool{
			"owner/repo": true,
		},
		Tags: map[string]string{
			"owner/repo": "v1.0.0",
		},
		CheckErr: map[string]error{},
		TagErr:   map[string]error{},
	}
}

func (f *FakeGithubClient) CheckIfRepoExists(ctx context.Context, repoAddr string) (bool, error) {
	if err := f.CheckErr[repoAddr]; err != nil {
		return false, err
	}
	return f.Exists[repoAddr], nil
}

func (f *FakeGithubClient) GetRepositoryLatestTag(ctx context.Context, repoAddr string) (string, error) {
	if err := f.TagErr[repoAddr]; err != nil {
		return "", err
	}
	return f.Tags[repoAddr], nil
}

package external

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/horalstvo/ghs/util"
	"golang.org/x/oauth2"
	"os"
	"strconv"
	"time"
)

var (
	FromDate = time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC)
)

func GetClient(ctx context.Context, apiToken string) *github.Client {

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: apiToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}

func GetPullRequests(org string, repo string, ctx context.Context, client *github.Client) []*github.PullRequest {
	file, err := os.Create(fmt.Sprintf("%v-%v-%v-%v-%v.csv", repo, FromDate.Month(), FromDate.Day(), time.Now().Month(), time.Now().Day()))
	util.Check(err)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	err = writer.Write([]string{"ID", "Number", "CreatedAt", "ClosedAt", "MergedAt", "Author", "ChangedFiles", "NumberReviewers", "baseBranch", "headBranch", "firstReview", "approveReview", "firstApprover", "totalReviews"})

	allPRs := []*github.PullRequest{}
	pageId := 0
	for {
		prs, resp, err := client.PullRequests.List(ctx, org, repo, &github.PullRequestListOptions{
			Sort:      "created",
			State:     "all",
			Direction: "desc",
			ListOptions: github.ListOptions{
				Page:    pageId,
				PerPage: 100,
			},
		})
		util.Check(err)
		pageId = resp.NextPage
		fmt.Printf("Page: %v, Date: %v-%v\n", pageId, prs[0].GetCreatedAt().Month(), prs[0].GetCreatedAt().Day())

		util.Check(err)
		for _, pr := range prs {
			if pr.CreatedAt.After(FromDate) {
				entry := []string{strconv.Itoa(int(pr.GetID())),
					strconv.Itoa(pr.GetNumber()),
					pr.GetCreatedAt().String(),
					pr.GetClosedAt().String(),
					pr.GetMergedAt().String(),
					pr.GetUser().GetLogin(),
					strconv.Itoa(pr.GetChangedFiles()),
					strconv.Itoa(len(pr.RequestedReviewers)),
					pr.GetBase().GetRef(),
					pr.GetHead().GetRef()}

				reviews := GetReviews(org, repo, pr.GetNumber(), ctx, client)
				if len(reviews) > 0 {
					// First review comment
					entry = append(entry, reviews[0].SubmittedAt.String())
					for _, rev := range reviews {
						if rev.GetState() == "APPROVED" {
							entry = append(entry, rev.SubmittedAt.String())
							entry = append(entry, rev.GetUser().GetLogin())
							entry = append(entry, strconv.Itoa(len(reviews)))
						}
					}
				} else {
					entry = append(entry, "", "", "", "0")
				}
				err = writer.Write(entry)
				if err != nil {
					fmt.Printf("failed to write PR %v\n", pr.GetNumber())
				}
				allPRs = append(allPRs, pr)
			} else {
				return allPRs
			}
		}
		writer.Flush()
	}
}



func GetReviews(org string, repo string, number int, ctx context.Context,
	client *github.Client) []*github.PullRequestReview {
	reviews, _, err := client.PullRequests.ListReviews(ctx, org, repo, number, &github.ListOptions{})
	util.Check(err)
	return reviews
}

func GetTeamRepos(org string, team string, ctx context.Context, client *github.Client) []*github.Repository {
	teamId, getTeamErr := getTeamId(org, team, ctx, client)
	util.Check(getTeamErr)

	repos, _, err := client.Teams.ListTeamRepos(ctx, *teamId, &github.ListOptions{})
	util.Check(err)
	return repos
}

func getTeamId(org string, team string, ctx context.Context, client *github.Client) (*int64, error) {
	teams, _, err := client.Teams.ListTeams(ctx, org, &github.ListOptions{})
	util.Check(err)

	for _, t := range teams {
		fmt.Printf("team: %v\n", *t.Name)
	}

	for _, t := range teams {
		if *t.Name == team {
			return t.ID, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("Team not found for %s", team));
}

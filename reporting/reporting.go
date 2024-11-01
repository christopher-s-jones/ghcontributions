package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/shurcooL/githubv4"
)

// A QueryResult represents a Github GraphQL query result that returns select high level fields
// The User.Login field tag ($login) is configurable with a variables map, as are the
// User.ContributionCollection field tags ($from and $to).
type QueryResult struct {
	User struct {
		Login                   githubv4.String
		ContributionsCollection struct {
			HasAnyContributions                                githubv4.Boolean
			HasActivityInThePast                               githubv4.Boolean
			RestrictedContributionsCount                       githubv4.Int
			TotalCommitContributions                           githubv4.Int
			TotalIssueContributions                            githubv4.Int
			TotalPullRequestContributions                      githubv4.Int
			TotalPullRequestReviewContributions                githubv4.Int
			TotalRepositoriesWithContributedIssues             githubv4.Int
			TotalRepositoriesWithContributedCommits            githubv4.Int
			TotalRepositoriesWithContributedPullRequests       githubv4.Int
			TotalRepositoriesWithContributedPullRequestReviews githubv4.Int
			CommitContributionsByRepository                    []struct {
				Repository struct {
					Name githubv4.String
					URL  githubv4.String
				}
				Contributions struct {
					TotalCount githubv4.Int
				}
			}
			IssueContributionsByRepository []struct {
				Repository struct {
					Name githubv4.String
					URL  githubv4.String
				}
				Contributions struct {
					TotalCount githubv4.Int
				}
			}
			PullRequestContributionsByRepository []struct {
				Repository struct {
					Name githubv4.String
					URL  githubv4.String
				}
				Contributions struct {
					TotalCount githubv4.Int
				}
			}
			PullRequestReviewContributionsByRepository []struct {
				Repository struct {
					Name githubv4.String
					URL  githubv4.String
				}
				Contributions struct {
					TotalCount githubv4.Int
				}
			}
		} `graphql:"contributionsCollection(from: $from, to: $to)"`
	} `graphql:"user(login: $login)"`
}

// AggregatedResults stores the aggregated results in three fields: totalCommitContributions,
// totalRepositories, and totalOtherContributions
type AggregatedResults struct {
	Timestamp                int `json:"timestamp"`
	TotalCommitContributions int `json:"totalCommitContributions"`
	TotalRepositories        int `json:"totalRepositories"`
	TotalOtherContributions  int `json:"totalOtherContributions"`
}

// Represents a Github username and its associated API token string
type Credential struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

// Represents a list of credential objects
type Credentials []Credential

// A Reporter collects high level statistics for one or more github
// usernames, aggregates the results, and reports the results
// as three simple metrics: “totalCodeCommits“, across “totalRepositories“, and
// “totalOtherContributions“ (an aggregate of total TotalIssueContributions,
// “totalPullRequestContributions“, and “totalPullRequestReviewContributions“)
type Reporter struct {
	// An authenticated Github Client using an OAuth token
	Client *githubv4.Client
	// The Github username login
	User string
	// The last year to report statistics (defaults to the current year)
	LastYear int
	// The first year to report statistics (defaults to 2000)
	FirstYear int
}

// Constructs a new Reporter object
// The client is a pointer to a githubv4.Client object
// The user is a github username string
// The firstYear is the first year in the sequence to report
// The lastYear is the last year in the sequence to report
func (r *Reporter) NewReporter(client *githubv4.Client, user string, firstYear int, lastYear int) (reporter Reporter, err error) {

	if user == "" {
		err = fmt.Errorf("user %s cannot be blank in constructing a query", user)
		return Reporter{}, err
	}

	// Start with the current thisYear in UTC
	thisYear := time.Now().UTC().Year()

	// Validate the start year, default to 2000
	if firstYear < DefaultFirstContributionYear || firstYear > thisYear {
		firstYear = DefaultFirstContributionYear
	}

	if lastYear > thisYear {
		lastYear = thisYear
		log.Printf("the last reporting year can't be in the future. Using %d", thisYear)
	}

	if lastYear == 0 || lastYear < firstYear {
		lastYear = thisYear
		log.Printf("the last reporting year can't be earlier than the first year. Using %d", thisYear)

	}

	return Reporter{
		Client:    client,
		User:      user,
		LastYear:  lastYear,
		FirstYear: firstYear,
	}, err
}

// Returned query results by username-year
var queryResults = make(map[string]QueryResult)

// Collects Github contribution statistics via the GraphQL service
// Returns the results as map of user-year strings to Query objects, and a nil error on success
func (r *Reporter) Collect() (err error) {

	var queryResult = QueryResult{}

	log.Printf("fetching repository statistics...")

	// run the queries
	for targetYear := r.LastYear; targetYear >= r.FirstYear; targetYear-- {

		from := time.Date(targetYear, time.January, 1, 0, 0, 0, 0, time.UTC) // {year}-01-01T00:00:00
		to := from.AddDate(1, 0, 0).Add(-time.Second)                        // {year}-12-31T11:59:59

		// Build a map of variable values
		var variables = map[string]interface{}{
			"login": githubv4.String(r.User),
			"from":  githubv4.DateTime{Time: from},
			"to":    githubv4.DateTime{Time: to},
		}

		err := r.Client.Query(context.Background(), &queryResult, variables)
		if err != nil {
			log.Fatal(err)
			return err
		}
		if githubv4.String(queryResult.User.Login) != "" {
			userYear := r.User + "-" + strconv.Itoa(targetYear)
			log.Println(userYear)
			queryResults[userYear] = queryResult // Store a copy of the user-year results
		}
		hasActivityInThePast := queryResult.User.ContributionsCollection.HasActivityInThePast
		if !hasActivityInThePast {
			break
		}
	}
	return
}

// report the final results
// TODO: change this to the aggregated results
func (r *Reporter) Report() (aggregatedResultsJSON string, err error) {

	aggregatedResults, err := r.Aggregate(&queryResults)
	if err != nil {
		return aggregatedResultsJSON, err
	}

	b, err := json.MarshalIndent(aggregatedResults, "", "  ")
	if err != nil {
		return aggregatedResultsJSON, err
	}
	aggregatedResultsJSON = string(b[:])
	return
}

// Aggregates the results of each user over each year into:
//   - totalCommitContributions: The count of all commits across all users in the results.
//   - totalRepositories: The count of unique list of repository names committed to and contributed to
//     in other ways (issues, pull requests, and pull request reviews).
//   - totalOtherContributions: The count of all other contributions across all users, including
//     all issues, pull requests, and pull request reviews.
func (r *Reporter) Aggregate(queryResults *map[string]QueryResult) (aggregatedResults AggregatedResults, err error) {

	aggregatedResults = AggregatedResults{}
	var uniqueRepositories = make(map[string]int)

	for userYear, queryResult := range *queryResults {
		log.Println(userYear)
		// Aggregate total commits
		aggregatedResults.TotalCommitContributions +=
			int(queryResult.User.ContributionsCollection.TotalCommitContributions)
		// Aggregate other contributions
		aggregatedResults.TotalOtherContributions +=
			(int(queryResult.User.ContributionsCollection.TotalIssueContributions) +
				int(queryResult.User.ContributionsCollection.TotalPullRequestContributions) +
				int(queryResult.User.ContributionsCollection.TotalPullRequestReviewContributions))
		// Aggregate total repositories
		for _, repository := range queryResult.User.ContributionsCollection.CommitContributionsByRepository {
			uniqueRepositories[string(repository.Repository.Name)] = uniqueRepositories[string(repository.Repository.Name)] + 1
		}
		for _, repository := range queryResult.User.ContributionsCollection.IssueContributionsByRepository {
			uniqueRepositories[string(repository.Repository.Name)] = uniqueRepositories[string(repository.Repository.Name)] + 1
		}
		for _, repository := range queryResult.User.ContributionsCollection.PullRequestContributionsByRepository {
			uniqueRepositories[string(repository.Repository.Name)] = uniqueRepositories[string(repository.Repository.Name)] + 1
		}
		for _, repository := range queryResult.User.ContributionsCollection.PullRequestReviewContributionsByRepository {
			uniqueRepositories[string(repository.Repository.Name)] = uniqueRepositories[string(repository.Repository.Name)] + 1
		}
	}
	aggregatedResults.TotalRepositories = len(uniqueRepositories)
	aggregatedResults.Timestamp = int(time.Now().Unix())
	return
}

func Poll() {
	// Periodically poll and cache github statistics
	ticker := time.NewTicker(time.Minute * PollingIntervalInMinutes)
	done := make(chan bool)

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			// TODO: poll github here
			log.Printf("Tick at: %s", t)
		}
	}
}

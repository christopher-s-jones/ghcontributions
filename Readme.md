# Github Contributions

Do you have multiple Github accounts for different purposes?
Work? Personal? Nym?  If so, you may want to collate your
contributions in order to see your total statistics.

This tool aggregates Github repository contributions for a given
list of users, across a range of years, and reports grand totals.

The statistics include:

- Total Commit Contributions
- Total Issue Contributions
- Total Pull Request Contributions
- Total Pull Request Review Contributions
- Total Repositories With Contributed Issues
- Total Repositories With Contributed Commits

## Usage

```
./ghcontributions -h
Github Summary Contributions reporter

	Aggregates contributions across a list of Github accounts, 
  and across a range of years, including total commits, 
  total count of public repositories contributed to, 
  and total other contributions, including pull requests, 
  merges, and issues.

Usage:
 ./ghcontributions [options]

  -credentials string
    	The name of the file containing Github usernames
    	and API token values (default "gh-tokens.json")
  -encrypted
    	Whether the credentials file is PGP encrypted.
  -firstyear int
    	The first year to summarize. (default 2000)
  -lastyear int
    	The last year to summarize (default 2024)

----------------------------------------

1. Create a JSON file with a list of credentials:

[
  {
    "username": "your-github-username",
    "token": "your-github-api-token"
  },
  {
    "username": "your-next-github-username",
    "token": "your-next-github-api-token"
  }
]

2. Optionally encrypt the file using PGP/GPG,
   and use the -encrypted flag if it is encrypted.

3. Optionally set the -firstyear and -lastyear flags with four digit years.

3. Pass the path to the file as the argument to the -credentials flag.
```


Running the `ghcontributions` command produces a JSON object, for example:

```json
{
  "timestamp": 1730498450,
  "totalCommitContributions": 1234,
  "totalRepositories": 56,
  "totalOtherContributions": 789
}
```
## GraphQL reporter

The `ghcontributions` command uses the included reporting module
to query the Github GraphQL API for contributions.  See the query below to understand exactly what it is returning.

```graphql
{
  user(login: "your-gh-username") {
    login
    contributionsCollection(from: "2000-01-01T00:00:00", to: "2024-10-31T11:59:59") {
      hasAnyContributions
      hasActivityInThePast
      restrictedContributionsCount
      totalCommitContributions
      totalIssueContributions
      totalPullRequestContributions
      totalPullRequestReviewContributions
      totalRepositoriesWithContributedIssues
      totalRepositoriesWithContributedCommits
      totalRepositoriesWithContributedPullRequests
      totalRepositoriesWithContributedPullRequestReviews
      commitContributionsByRepository {
        repository {
          name
          url
        }
        contributions {
          totalCount
        }
      }
      issueContributionsByRepository {
        repository {
          name
          url
        }
        contributions {
          totalCount
        }
      }
      pullRequestContributionsByRepository {
        repository {
          name
          url
        }
        contributions {
          totalCount
        }
      }
      pullRequestReviewContributionsByRepository {
        repository {
          name
          url
        }
        contributions {
          totalCount
        }
      }
    }
  }
}
```
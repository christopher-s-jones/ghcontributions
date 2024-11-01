module github.com/christopher-s-jones/ghcontributions

go 1.23.0

require github.com/shurcooL/githubv4 v0.0.0-20240727222349-48295856cce7

require (
	github.com/christopher-s-jones/ghcontributions/reporting v0.0.0-00010101000000-000000000000
	github.com/shurcooL/graphql v0.0.0-20230722043721-ed46e5a46466 // indirect
	golang.org/x/oauth2 v0.30.0
)

replace github.com/christopher-s-jones/ghcontributions => ./ghcontributions
replace github.com/christopher-s-jones/ghcontributions/reporting => ./reporting

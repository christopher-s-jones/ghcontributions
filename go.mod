module github.com/christopher-s-jones/ghcontributions

go 1.26.3

require (
	github.com/shurcooL/githubv4 v0.0.0-20260209031235-2402fdf4a9ed
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/shurcooL/graphql v0.0.0-20240915155400-7ee5256398cf // indirect
	golang.org/x/oauth2 v0.36.0
)

replace github.com/christopher-s-jones/ghcontributions => ./ghcontributions

replace github.com/christopher-s-jones/ghcontributions/reporting => ./reporting

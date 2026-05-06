package reporting_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	rpt "github.com/christopher-s-jones/ghcontributions/reporting"
	"github.com/shurcooL/githubv4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// A mock client for the Github GraphQL API client
type MockGraphQLClient struct {
	// A Mock object instance
	mock.Mock
	// Pre-canned fake API responses for each query
	Responses map[string]rpt.QueryResult
	// Keep track of API calls and call variables for testing
	Calls []struct {
		Variables map[string]interface{}
	}
}

// Query returns queryResults from the MockGraphQLClient
func (m *MockGraphQLClient) Query(ctx context.Context, q interface{}, variables map[string]interface{}) error {
	args := m.Called(ctx, q, variables)
	fromTime := variables["from"].(githubv4.DateTime).Time
	year := strconv.Itoa(fromTime.Year())

	if resultPtr, ok := q.(*rpt.QueryResult); ok {
		if fixture, exists := m.Responses[year]; exists {
			*resultPtr = fixture
		}
	}
	return args.Error(0)
}

// getFixturePath returns the absolute path to a fixture file
func getFixturePath(filename string) string {
	// Reliably locate fixture files from where we are called
	// Ignore the program_counter, line_number, and success_flag
	_, source_path, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(source_path)
	return filepath.Join(basepath, "fixtures", filename)
}

// loadFixture loads a JSON fixture into the provided interface
func loadFixture(filename string, v interface{}) error {
	data, err := os.ReadFile(getFixturePath(filename))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// loadQueryResultsMap loads a fixture into a map of QueryResult objects
func loadQueryResultsMap(filename string) (map[string]rpt.QueryResult, error) {
	var results map[string]rpt.QueryResult
	err := loadFixture(filename, &results)
	return results, err
}

// loadSingleFixture loads a specific year's data from a fixture file
func loadSingleFixture(filename string, user string, year string) (rpt.QueryResult, error) {
	queryResultsMap, err := loadQueryResultsMap(filename)
	if err != nil {
		return rpt.QueryResult{}, err
	}

	// Find the exact match for the user and year in the retrned fixture map
	key := user + "-" + year
	if queryResultsByUserYear, ok := queryResultsMap[key]; ok {
		return queryResultsByUserYear, nil
	}

	// Fallback: return first entry if exact key not found
	for _, queryResult := range queryResultsMap {
		return queryResult, nil
	}

	return rpt.QueryResult{}, fmt.Errorf("no data found in the fixture %s", filename)
}

// Test the NewReporter constructor
func TestNewReporter(t *testing.T) {
	tests := []struct {
		name      string
		user      string
		firstYear int
		lastYear  int
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid reporter with default years",
			user:      "testuser",
			firstYear: 0,
			lastYear:  0,
			wantErr:   false,
		},
		{
			name:      "valid reporter with custom years",
			user:      "testuser",
			firstYear: 2020,
			lastYear:  2023,
			wantErr:   false,
		},
		{
			name:      "invalid reporter with empty user",
			user:      "",
			firstYear: 2020,
			lastYear:  2023,
			wantErr:   true,
			errMsg:    "user cannot be blank in constructing a query",
		},
		{
			name:      "first year too early defaults to default",
			user:      "testuser",
			firstYear: 1900,
			lastYear:  2023,
			wantErr:   false,
		},
		{
			name:      "last year in future defaults to current year",
			user:      "testuser",
			firstYear: 2020,
			lastYear:  2099,
			wantErr:   false,
		},
		{
			name:      "last year before first year defaults to current year",
			user:      "testuser",
			firstYear: 2023,
			lastYear:  2020,
			wantErr:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reporter, err := rpt.NewReporter(nil, test.user, test.firstYear, test.lastYear)

			if test.wantErr {
				assert.Error(t, err)
				if test.errMsg != "" {
					assert.Contains(t, err.Error(), test.errMsg)
				}
				return
			}

			assert.NoError(t, err)

			// Ensure the user is set correctly
			assert.Equal(t, test.user, reporter.User)

			// Ensure firstYear is set correctly
			currentYear := time.Now().UTC().Year()
			if test.firstYear < rpt.DefaultFirstContributionYear || test.firstYear > currentYear {
				assert.Equal(t, rpt.DefaultFirstContributionYear, reporter.FirstYear)
			} else {
				assert.Equal(t, test.firstYear, reporter.FirstYear)
			}

			// Ensure lastYear is set correctly
			if test.lastYear > currentYear || test.lastYear == 0 || test.lastYear < test.firstYear {
				assert.Equal(t, currentYear, reporter.LastYear)
			} else {
				assert.Equal(t, test.lastYear, reporter.LastYear)
			}
		})
	}
}

// Test Aggregate function with fixtures
func TestAggregate(t *testing.T) {
	tests := []struct {
		name            string
		fixtureFile     string
		expectedCommits int
		expectedOther   int
		expectedRepos   int
	}{
		{
			name:            "empty results",
			fixtureFile:     "empty_results.json",
			expectedCommits: 0,
			expectedOther:   0,
			expectedRepos:   0,
		},
		{
			name:            "single user single year",
			fixtureFile:     "single_user_single_year.json",
			expectedCommits: 10,
			expectedOther:   10,
			expectedRepos:   1,
		},
		{
			name:            "multiple years deduplicates repositories",
			fixtureFile:     "multiple_years_deduplicated.json",
			expectedCommits: 8,
			expectedOther:   6,
			expectedRepos:   2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			queryResults, err := loadQueryResultsMap(test.fixtureFile)
			assert.NoError(t, err, "Failed to load fixture: %s", test.fixtureFile)

			r := &rpt.Reporter{}
			result, err := r.Aggregate(queryResults)

			assert.NoError(t, err)
			// Ensure the three main aggregated stats are reported correctly
			assert.Equal(t, test.expectedCommits, result.TotalCommitContributions)
			assert.Equal(t, test.expectedOther, result.TotalOtherContributions)
			assert.Equal(t, test.expectedRepos, result.TotalRepositories)

			// Ensure the aggregated result timestamps are in range
			assert.True(t, result.Timestamp > 0)
			assert.True(t, result.Timestamp <= int(time.Now().Unix()))
		})
	}
}

// Test the Report function
func TestReport(t *testing.T) {
	queryResults, err := loadQueryResultsMap("report_test_data.json")
	assert.NoError(t, err, "Failed to load report test fixture")

	r := &rpt.Reporter{}
	jsonStr, err := r.Report(queryResults)

	assert.NoError(t, err)
	assert.NotEmpty(t, jsonStr)

	var result rpt.AggregatedResults
	err = json.Unmarshal([]byte(jsonStr), &result)
	assert.NoError(t, err)
	assert.Equal(t, 5, result.TotalCommitContributions)
}

// Test the Collect function
func TestCollect(t *testing.T) {
	// Reset global state before each test
	// queryResults = make(map[string]QueryResult)

	tests := []struct {
		name          string
		fixtureFiles  map[string]string // Maps year -> fixture filename
		reporter      rpt.Reporter
		expectedCalls int
		expectedError bool
	}{
		{
			name: "single year collection",
			fixtureFiles: map[string]string{
				"2023": "single_user_single_year.json",
			},
			reporter: rpt.Reporter{
				User:      "user1",
				FirstYear: 2023,
				LastYear:  2023,
			},
			expectedCalls: 1,
		},
		{
			name: "multiple years with early termination",
			fixtureFiles: map[string]string{
				"2023": "multiple_years_deduplicated_2023.json",
				"2022": "multiple_years_deduplicated_2022.json",
			},
			reporter: rpt.Reporter{
				User:      "user1",
				FirstYear: 2022,
				LastYear:  2023,
			},
			expectedCalls: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Use the helper to load fixtures
			fixtureData := make(map[string]rpt.QueryResult)
			for year, filename := range test.fixtureFiles {
				result, err := loadSingleFixture(filename, test.reporter.User, year)
				if err != nil {
					t.Fatalf("Failed to load fixture %s: %v", filename, err)
				}
				fixtureData[year] = result
			}

			// Create mock with loaded fixtures
			mockClient := &MockGraphQLClient{
				Responses: fixtureData,
			}

			// Configure mock to return fixture data based on year in variables
			mockClient.On("Query", mock.Anything, mock.Anything, mock.MatchedBy(func(vars map[string]interface{}) bool {
				login, ok := vars["login"].(githubv4.String)
				return ok && string(login) == test.reporter.User
			})).Return(nil).Run(func(args mock.Arguments) {
				vars := args.Get(2).(map[string]interface{})
				fromTime := vars["from"].(githubv4.DateTime).Time
				year := strconv.Itoa(fromTime.Year())

				resultPtr := args.Get(1).(*rpt.QueryResult)
				if fixture, ok := fixtureData[year]; ok {
					*resultPtr = fixture
				}
			})

			// Setup reporter and execute
			test.reporter.Client = mockClient
			// queryResults, err := test.reporter.Collect()
			_, err := test.reporter.Collect()

			// Verify
			if test.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockClient.AssertNumberOfCalls(t, "Query", test.expectedCalls)
		})
	}
}

// Test the Credential struct
func TestCredential(t *testing.T) {
	cred := rpt.Credential{
		Username: "testuser",
		Token:    "testtoken123",
	}

	assert.Equal(t, "testuser", cred.Username)
	assert.Equal(t, "testtoken123", cred.Token)
}

// Test Credentials slice
func TestCredentials(t *testing.T) {
	creds := rpt.Credentials{
		{Username: "user1", Token: "token1"},
		{Username: "user2", Token: "token2"},
	}

	assert.Len(t, creds, 2)
	assert.Equal(t, "user1", creds[0].Username)
	assert.Equal(t, "token2", creds[1].Token)
}

// Test Repository struct
func TestRepository(t *testing.T) {
	repo := rpt.Repository{
		Name: "myrepo",
		URL:  "https://github.com/user/myrepo",
	}

	jsonData, err := json.Marshal(repo)
	assert.NoError(t, err)

	expected := `{"name":"myrepo","url":"https://github.com/user/myrepo"}`
	assert.Equal(t, expected, string(jsonData))
}

// Test QueryResult structure
func TestQueryResultStructure(t *testing.T) {
	result := rpt.QueryResult{}
	result.User.Login = "testuser"
	result.User.ContributionsCollection.TotalCommitContributions = 42

	assert.Equal(t, githubv4.String("testuser"), result.User.Login)
	assert.Equal(t, githubv4.Int(42), result.User.ContributionsCollection.TotalCommitContributions)
}

// Benchmark Aggregate function
func BenchmarkAggregate(b *testing.B) {
	queryResults, err := loadQueryResultsMap("benchmark_data.json")
	if err != nil {
		b.Fatal(err)
	}

	r := &rpt.Reporter{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := r.Aggregate(queryResults)
		if err != nil {
			b.Fatal(err)
		}
	}
}

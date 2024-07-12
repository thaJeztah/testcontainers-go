// This test is testing very internal logic that should not be exported away from this package. We'll
// leave it in the main testcontainers package. Do not use for user facing examples.
package testcontainers

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestParseDockerIgnore(t *testing.T) {
	testCases := []struct {
		filePath         string
		exists           bool
		expectedErr      error
		expectedExcluded []string
	}{
		{
			filePath:         "./testdata/dockerignore",
			expectedErr:      nil,
			exists:           true,
			expectedExcluded: []string{"vendor", "foo", "bar"},
		},
		{
			filePath:         "./testdata",
			expectedErr:      nil,
			exists:           true,
			expectedExcluded: []string{"Dockerfile", "echo.Dockerfile"},
		},
		{
			filePath:         "./testdata/data",
			expectedErr:      nil,
			expectedExcluded: nil, // it's nil because the parseDockerIgnore function uses the zero value of a slice
		},
	}

	for _, testCase := range testCases {
		exists, excluded, err := parseDockerIgnore(testCase.filePath)
		assert.Check(t, is.Equal(testCase.exists, exists))
		assert.Check(t, is.DeepEqual(testCase.expectedErr, err))
		assert.Check(t, is.DeepEqual(testCase.expectedExcluded, excluded))
	}
}

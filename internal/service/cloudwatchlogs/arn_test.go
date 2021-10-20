package cloudwatchlogs_test

import (
	"testing"

	tfcloudwatchlogs "github.com/hashicorp/terraform-provider-aws/internal/service/cloudwatchlogs"
)

func TestTrimLogGroupARNWildcardSuffix(t *testing.T) {
	testCases := []struct {
		TestName    string
		InputARN    string
		ExpectedARN string
	}{
		{
			TestName: "Empty string",
		},
		{
			TestName:    "No suffix",
			InputARN:    "arn:aws-us-gov:logs:us-gov-west-1:123456789012:log-group:tf-acc-test-6899758375212691725/1", //lintignore:AWSAT003,AWSAT005
			ExpectedARN: "arn:aws-us-gov:logs:us-gov-west-1:123456789012:log-group:tf-acc-test-6899758375212691725/1", //lintignore:AWSAT003,AWSAT005
		},
		{
			TestName:    "With suffix",
			InputARN:    "arn:aws-us-gov:logs:us-gov-west-1:123456789012:log-group:tf-acc-test-6899758375212691725/1:*", //lintignore:AWSAT003,AWSAT005
			ExpectedARN: "arn:aws-us-gov:logs:us-gov-west-1:123456789012:log-group:tf-acc-test-6899758375212691725/1",   //lintignore:AWSAT003,AWSAT005
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.TestName, func(t *testing.T) {
			got := tfcloudwatchlogs.TrimLogGroupARNWildcardSuffix(testCase.InputARN)

			if got != testCase.ExpectedARN {
				t.Errorf("got %s, expected %s", got, testCase.ExpectedARN)
			}
		})
	}
}

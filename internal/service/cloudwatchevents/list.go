package cloudwatchevents

import (
	"github.com/aws/aws-sdk-go/aws"
	events "github.com/aws/aws-sdk-go/service/cloudwatchevents"
)

func ListAllTargetsForRulePages(conn *events.CloudWatchEvents, busName, ruleName string, fn func(*events.ListTargetsByRuleOutput, bool) bool) error {
	input := &events.ListTargetsByRuleInput{
		Rule:  aws.String(ruleName),
		Limit: aws.Int64(100), // Set limit to allowed maximum to prevent API throttling
	}

	if busName != "" {
		input.EventBusName = aws.String(busName)
	}

	return listTargetsByRulePages(conn, input, fn)
}

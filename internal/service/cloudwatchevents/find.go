package cloudwatchevents

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	events "github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func FindConnectionByName(conn *events.CloudWatchEvents, name string) (*events.DescribeConnectionOutput, error) {
	input := &events.DescribeConnectionInput{
		Name: aws.String(name),
	}

	output, err := conn.DescribeConnection(input)

	if tfawserr.ErrCodeEquals(err, events.ErrCodeResourceNotFoundException) {
		return nil, &resource.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil {
		return nil, &resource.NotFoundError{
			Message:     "Empty result",
			LastRequest: input,
		}
	}

	return output, nil
}

func FindRuleByEventBusAndRuleNames(conn *events.CloudWatchEvents, eventBusName, ruleName string) (*events.DescribeRuleOutput, error) {
	input := events.DescribeRuleInput{
		Name: aws.String(ruleName),
	}

	if eventBusName != "" {
		input.EventBusName = aws.String(eventBusName)
	}

	output, err := conn.DescribeRule(&input)

	if tfawserr.ErrCodeEquals(err, events.ErrCodeResourceNotFoundException) {
		return nil, &resource.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil {
		return nil, &resource.NotFoundError{
			Message:     "Empty result",
			LastRequest: input,
		}
	}

	return output, nil
}

func FindRuleByResourceID(conn *events.CloudWatchEvents, id string) (*events.DescribeRuleOutput, error) {
	eventBusName, ruleName, err := RuleParseResourceID(id)

	if err != nil {
		return nil, err
	}

	return FindRuleByEventBusAndRuleNames(conn, eventBusName, ruleName)
}

func FindTarget(conn *events.CloudWatchEvents, busName, ruleName, targetId string) (*events.Target, error) {
	var result *events.Target
	err := ListAllTargetsForRulePages(conn, busName, ruleName, func(page *events.ListTargetsByRuleOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, t := range page.Targets {
			if targetId == aws.StringValue(t.Id) {
				result = t
				return false
			}
		}

		return !lastPage
	})
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, fmt.Errorf("CloudWatch Event FindTarget %q (\"%s/%s\") not found", targetId, busName, ruleName)
	}
	return result, nil
}

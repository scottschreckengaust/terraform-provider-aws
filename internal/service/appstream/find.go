package appstream

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appstream"
)

// FindStackByName Retrieve a appstream stack by name
func FindStackByName(ctx context.Context, conn *appstream.AppStream, name string) (*appstream.Stack, error) {
	input := &appstream.DescribeStacksInput{
		Names: []*string{aws.String(name)},
	}

	var stack *appstream.Stack
	resp, err := conn.DescribeStacksWithContext(ctx, input)

	if err != nil {
		return nil, err
	}

	if len(resp.Stacks) > 1 {
		return nil, fmt.Errorf("got more than one stack with the name %s", name)
	}

	if len(resp.Stacks) == 1 {
		stack = resp.Stacks[0]
	}

	return stack, nil
}

// FindFleetByName Retrieve a appstream fleet by name
func FindFleetByName(ctx context.Context, conn *appstream.AppStream, name string) (*appstream.Fleet, error) {
	input := &appstream.DescribeFleetsInput{
		Names: []*string{aws.String(name)},
	}

	var fleet *appstream.Fleet
	resp, err := conn.DescribeFleetsWithContext(ctx, input)

	if err != nil {
		return nil, err
	}

	if len(resp.Fleets) > 1 {
		return nil, fmt.Errorf("got more than one fleet with the name %s", name)
	}

	if len(resp.Fleets) == 1 {
		fleet = resp.Fleets[0]
	}

	return fleet, nil
}

// FindImageBuilderByName Retrieve a appstream ImageBuilder by name
func FindImageBuilderByName(ctx context.Context, conn *appstream.AppStream, name string) (*appstream.ImageBuilder, error) {
	input := &appstream.DescribeImageBuildersInput{
		Names: []*string{aws.String(name)},
	}

	var result *appstream.ImageBuilder

	err := describeImageBuildersPagesWithContext(ctx, conn, input, func(page *appstream.DescribeImageBuildersOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, imageBuilder := range page.ImageBuilders {
			if imageBuilder == nil {
				continue
			}
			if aws.StringValue(imageBuilder.Name) == name {
				result = imageBuilder
				return false
			}
		}

		return !lastPage
	})

	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return result, nil
}

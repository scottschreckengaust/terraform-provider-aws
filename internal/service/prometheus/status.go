package prometheus

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/prometheusservice"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	resourceStatusFailed  = "Failed"
	resourceStatusUnknown = "Unknown"
	resourceStatusDeleted = "Deleted"
)

// statusWorkspaceCreated fetches the Workspace and its Status.
func statusWorkspaceCreated(ctx context.Context, conn *prometheusservice.PrometheusService, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &prometheusservice.DescribeWorkspaceInput{
			WorkspaceId: aws.String(id),
		}

		output, err := conn.DescribeWorkspaceWithContext(ctx, input)

		if err != nil {
			return output, resourceStatusFailed, err
		}

		if output == nil || output.Workspace == nil {
			return output, resourceStatusUnknown, nil
		}

		return output.Workspace, aws.StringValue(output.Workspace.Status.StatusCode), nil
	}
}

// statusWorkspaceDeleted fetches the Workspace and its Status
func statusWorkspaceDeleted(ctx context.Context, conn *prometheusservice.PrometheusService, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &prometheusservice.DescribeWorkspaceInput{
			WorkspaceId: aws.String(id),
		}

		output, err := conn.DescribeWorkspaceWithContext(ctx, input)

		if tfawserr.ErrCodeEquals(err, prometheusservice.ErrCodeResourceNotFoundException) {
			return output, resourceStatusDeleted, nil
		}

		if err != nil {
			return output, resourceStatusUnknown, err
		}

		if output == nil || output.Workspace == nil {
			return output, resourceStatusUnknown, nil
		}

		return output.Workspace, aws.StringValue(output.Workspace.Status.StatusCode), nil
	}
}

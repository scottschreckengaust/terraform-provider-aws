package cloudwatchevents

import (
	"github.com/aws/aws-sdk-go/aws"
	events "github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

func statusConnectionState(conn *events.CloudWatchEvents, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := FindConnectionByName(conn, name)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, aws.StringValue(output.ConnectionState), nil
	}
}

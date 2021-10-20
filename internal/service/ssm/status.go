package ssm

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	documentStatusUnknown = "Unknown"
)

// statusDocument fetches the Document and its Status
func statusDocument(conn *ssm.SSM, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := FindDocumentByName(conn, name)

		if err != nil {
			return nil, ssm.DocumentStatusFailed, err
		}

		if output == nil {
			return output, documentStatusUnknown, nil
		}

		return output, aws.StringValue(output.Status), nil
	}
}

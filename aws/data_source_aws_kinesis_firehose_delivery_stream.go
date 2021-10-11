package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/firehose/finder"
)

func dataSourceAwsKinesisFirehoseDeliveryStream() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsKinesisFirehoseDeliveryStreamRead,
		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceAwsKinesisFirehoseDeliveryStreamRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).firehoseconn

	sn := d.Get("name").(string)
	output, err := finder.DeliveryStreamByName(conn, sn)

	if err != nil {
		return fmt.Errorf("error reading Kinesis Firehose Delivery Stream (%s): %w", sn, err)
	}

	d.SetId(aws.StringValue(output.DeliveryStreamARN))
	d.Set("arn", output.DeliveryStreamARN)
	d.Set("name", output.DeliveryStreamName)

	return nil
}

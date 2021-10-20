package ec2

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func DataSourceInternetGateway() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceInternetGatewayRead,
		Schema: map[string]*schema.Schema{
			"internet_gateway_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"filter": CustomFiltersSchema(),
			"tags":   tftags.TagsSchemaComputed(),
			"attachments": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"state": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"vpc_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceInternetGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	req := &ec2.DescribeInternetGatewaysInput{}
	internetGatewayId, internetGatewayIdOk := d.GetOk("internet_gateway_id")
	tags, tagsOk := d.GetOk("tags")
	filter, filterOk := d.GetOk("filter")

	if !internetGatewayIdOk && !filterOk && !tagsOk {
		return fmt.Errorf("One of internet_gateway_id or filter or tags must be assigned")
	}

	req.Filters = BuildAttributeFilterList(map[string]string{
		"internet-gateway-id": internetGatewayId.(string),
	})
	req.Filters = append(req.Filters, BuildTagFilterList(
		Tags(tftags.New(tags.(map[string]interface{}))),
	)...)
	req.Filters = append(req.Filters, BuildCustomFilterList(
		filter.(*schema.Set),
	)...)

	log.Printf("[DEBUG] Reading Internet Gateway: %s", req)
	resp, err := conn.DescribeInternetGateways(req)

	if err != nil {
		return err
	}
	if resp == nil || len(resp.InternetGateways) == 0 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}
	if len(resp.InternetGateways) > 1 {
		return fmt.Errorf("Multiple Internet Gateways matched; use additional constraints to reduce matches to a single Internet Gateway")
	}

	igw := resp.InternetGateways[0]
	d.SetId(aws.StringValue(igw.InternetGatewayId))

	if err := d.Set("tags", KeyValueTags(igw.Tags).IgnoreAWS().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	d.Set("owner_id", igw.OwnerId)
	d.Set("internet_gateway_id", igw.InternetGatewayId)

	arn := arn.ARN{
		Partition: meta.(*conns.AWSClient).Partition,
		Service:   ec2.ServiceName,
		Region:    meta.(*conns.AWSClient).Region,
		AccountID: aws.StringValue(igw.OwnerId),
		Resource:  fmt.Sprintf("internet-gateway/%s", d.Id()),
	}.String()

	d.Set("arn", arn)

	err1 := d.Set("attachments", dataSourceAttachmentsRead(igw.Attachments))
	return err1

}

func dataSourceAttachmentsRead(igwAttachments []*ec2.InternetGatewayAttachment) []map[string]interface{} {
	attachments := make([]map[string]interface{}, 0, len(igwAttachments))
	for _, a := range igwAttachments {
		m := make(map[string]interface{})
		m["state"] = aws.StringValue(a.State)
		m["vpc_id"] = aws.StringValue(a.VpcId)
		attachments = append(attachments, m)
	}

	return attachments
}

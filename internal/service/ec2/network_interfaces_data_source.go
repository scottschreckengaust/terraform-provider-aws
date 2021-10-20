package ec2

import (
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func DataSourceNetworkInterfaces() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkInterfacesRead,
		Schema: map[string]*schema.Schema{

			"filter": CustomFiltersSchema(),

			"tags": tftags.TagsSchemaComputed(),

			"ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func dataSourceNetworkInterfacesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn

	req := &ec2.DescribeNetworkInterfacesInput{}

	filters, filtersOk := d.GetOk("filter")
	tags, tagsOk := d.GetOk("tags")

	if tagsOk {
		req.Filters = BuildTagFilterList(
			Tags(tftags.New(tags.(map[string]interface{}))),
		)
	}

	if filtersOk {
		req.Filters = append(req.Filters, BuildCustomFilterList(
			filters.(*schema.Set),
		)...)
	}

	if len(req.Filters) == 0 {
		req.Filters = nil
	}

	log.Printf("[DEBUG] DescribeNetworkInterfaces %s\n", req)
	resp, err := conn.DescribeNetworkInterfaces(req)
	if err != nil {
		return err
	}

	if resp == nil || len(resp.NetworkInterfaces) == 0 {
		return errors.New("no matching network interfaces found")
	}

	networkInterfaces := make([]string, 0)

	for _, networkInterface := range resp.NetworkInterfaces {
		networkInterfaces = append(networkInterfaces, aws.StringValue(networkInterface.NetworkInterfaceId))
	}

	d.SetId(meta.(*conns.AWSClient).Region)

	if err := d.Set("ids", networkInterfaces); err != nil {
		return fmt.Errorf("Error setting network interfaces ids: %w", err)
	}

	return nil
}

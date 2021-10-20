package ec2

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func DataSourceLocalGatewayVirtualInterfaceGroups() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceLocalGatewayVirtualInterfaceGroupsRead,

		Schema: map[string]*schema.Schema{
			"filter": CustomFiltersSchema(),
			"ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"local_gateway_virtual_interface_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tags": tftags.TagsSchemaComputed(),
		},
	}
}

func dataSourceLocalGatewayVirtualInterfaceGroupsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn

	input := &ec2.DescribeLocalGatewayVirtualInterfaceGroupsInput{}

	input.Filters = append(input.Filters, BuildTagFilterList(
		Tags(tftags.New(d.Get("tags").(map[string]interface{}))),
	)...)

	input.Filters = append(input.Filters, BuildCustomFilterList(
		d.Get("filter").(*schema.Set),
	)...)

	if len(input.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		input.Filters = nil
	}

	var localGatewayVirtualInterfaceGroups []*ec2.LocalGatewayVirtualInterfaceGroup

	err := conn.DescribeLocalGatewayVirtualInterfaceGroupsPages(input, func(page *ec2.DescribeLocalGatewayVirtualInterfaceGroupsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		localGatewayVirtualInterfaceGroups = append(localGatewayVirtualInterfaceGroups, page.LocalGatewayVirtualInterfaceGroups...)

		return !lastPage
	})

	if err != nil {
		return fmt.Errorf("error describing EC2 Local Gateway Virtual Interface Groups: %w", err)
	}

	if len(localGatewayVirtualInterfaceGroups) == 0 {
		return fmt.Errorf("no matching EC2 Local Gateway Virtual Interface Groups found")
	}

	var ids, localGatewayVirtualInterfaceIds []*string

	for _, group := range localGatewayVirtualInterfaceGroups {
		if group == nil {
			continue
		}

		ids = append(ids, group.LocalGatewayVirtualInterfaceGroupId)
		localGatewayVirtualInterfaceIds = append(localGatewayVirtualInterfaceIds, group.LocalGatewayVirtualInterfaceIds...)
	}

	d.SetId(meta.(*conns.AWSClient).Region)

	if err := d.Set("ids", ids); err != nil {
		return fmt.Errorf("error setting ids: %w", err)
	}

	if err := d.Set("local_gateway_virtual_interface_ids", localGatewayVirtualInterfaceIds); err != nil {
		return fmt.Errorf("error setting local_gateway_virtual_interface_ids: %w", err)
	}

	return nil
}

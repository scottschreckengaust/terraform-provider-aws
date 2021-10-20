package ec2

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func DataSourceCoIPPools() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCoIPPoolsRead,
		Schema: map[string]*schema.Schema{
			"filter": CustomFiltersSchema(),

			"tags": tftags.TagsSchemaComputed(),

			"pool_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func dataSourceCoIPPoolsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn

	req := &ec2.DescribeCoipPoolsInput{}

	if tags, tagsOk := d.GetOk("tags"); tagsOk {
		req.Filters = append(req.Filters, BuildTagFilterList(
			Tags(tftags.New(tags.(map[string]interface{}))),
		)...)
	}

	if filters, filtersOk := d.GetOk("filter"); filtersOk {
		req.Filters = append(req.Filters, BuildCustomFilterList(
			filters.(*schema.Set),
		)...)
	}
	if len(req.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filters = nil
	}

	var coipPools []*ec2.CoipPool

	err := conn.DescribeCoipPoolsPages(req, func(page *ec2.DescribeCoipPoolsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		coipPools = append(coipPools, page.CoipPools...)

		return !lastPage
	})

	if err != nil {
		return fmt.Errorf("error describing EC2 COIP Pools: %w", err)
	}

	if len(coipPools) == 0 {
		return fmt.Errorf("no matching EC2 COIP Pools found")
	}

	var poolIDs []string

	for _, coipPool := range coipPools {
		if coipPool == nil {
			continue
		}

		poolIDs = append(poolIDs, aws.StringValue(coipPool.PoolId))
	}

	d.SetId(meta.(*conns.AWSClient).Region)

	if err := d.Set("pool_ids", poolIDs); err != nil {
		return fmt.Errorf("error setting pool_ids: %w", err)
	}

	return nil
}

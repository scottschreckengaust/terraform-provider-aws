package ec2

import (
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func DataSourceEBSVolumes() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceEBSVolumesRead,
		Schema: map[string]*schema.Schema{
			"filter": CustomFiltersSchema(),

			"tags": tftags.TagsSchema(),

			"ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func dataSourceEBSVolumesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn

	req := &ec2.DescribeVolumesInput{}

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
		req.Filters = nil
	}

	log.Printf("[DEBUG] DescribeVolumes %s\n", req)
	resp, err := conn.DescribeVolumes(req)
	if err != nil {
		return fmt.Errorf("error describing EC2 Volumes: %w", err)
	}

	if resp == nil || len(resp.Volumes) == 0 {
		return errors.New("no matching volumes found")
	}

	volumes := make([]string, 0)

	for _, volume := range resp.Volumes {
		volumes = append(volumes, *volume.VolumeId)
	}

	d.SetId(meta.(*conns.AWSClient).Region)

	if err := d.Set("ids", volumes); err != nil {
		return fmt.Errorf("error setting ids: %w", err)
	}

	return nil
}

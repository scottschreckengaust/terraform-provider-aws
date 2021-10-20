package kinesis

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func DataSourceStream() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceStreamRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"creation_timestamp": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"retention_period": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"open_shards": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"closed_shards": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"shard_level_metrics": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"tags": tftags.TagsSchemaComputed(),
		},
	}
}

func dataSourceStreamRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).KinesisConn
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	sn := d.Get("name").(string)

	state, err := readKinesisStreamState(conn, sn)
	if err != nil {
		return err
	}
	d.SetId(state.arn)
	d.Set("arn", state.arn)
	d.Set("name", sn)
	d.Set("open_shards", state.openShards)
	d.Set("closed_shards", state.closedShards)
	d.Set("status", state.status)
	d.Set("creation_timestamp", state.creationTimestamp)
	d.Set("retention_period", state.retentionPeriod)
	d.Set("shard_level_metrics", state.shardLevelMetrics)

	tags, err := ListTags(conn, sn)

	if err != nil {
		return fmt.Errorf("error listing tags for Kinesis Stream (%s): %w", sn, err)
	}

	if err := d.Set("tags", tags.IgnoreAWS().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	return nil
}

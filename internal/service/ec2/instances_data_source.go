package ec2

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func DataSourceInstances() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceInstancesRead,

		Schema: map[string]*schema.Schema{
			"filter":        DataSourceFiltersSchema(),
			"instance_tags": tftags.TagsSchemaComputed(),
			"instance_state_names": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						ec2.InstanceStateNamePending,
						ec2.InstanceStateNameRunning,
						ec2.InstanceStateNameShuttingDown,
						ec2.InstanceStateNameStopped,
						ec2.InstanceStateNameStopping,
						ec2.InstanceStateNameTerminated,
					}, false),
				},
			},

			"ids": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"private_ips": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"public_ips": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceInstancesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn

	filters, filtersOk := d.GetOk("filter")
	tags, tagsOk := d.GetOk("instance_tags")

	if !filtersOk && !tagsOk {
		return fmt.Errorf("One of filters or instance_tags must be assigned")
	}

	instanceStateNames := []*string{aws.String(ec2.InstanceStateNameRunning)}
	if v, ok := d.GetOk("instance_state_names"); ok && len(v.(*schema.Set).List()) > 0 {
		instanceStateNames = flex.ExpandStringSet(v.(*schema.Set))
	}
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: instanceStateNames,
			},
		},
	}

	if filtersOk {
		params.Filters = append(params.Filters,
			BuildFiltersDataSource(filters.(*schema.Set))...)
	}
	if tagsOk {
		params.Filters = append(params.Filters, BuildTagFilterList(
			Tags(tftags.New(tags.(map[string]interface{}))),
		)...)
	}

	log.Printf("[DEBUG] Reading EC2 instances: %s", params)

	var instanceIds, privateIps, publicIps []string
	err := conn.DescribeInstancesPages(params, func(resp *ec2.DescribeInstancesOutput, lastPage bool) bool {
		for _, res := range resp.Reservations {
			for _, instance := range res.Instances {
				instanceIds = append(instanceIds, *instance.InstanceId)
				if instance.PrivateIpAddress != nil {
					privateIps = append(privateIps, *instance.PrivateIpAddress)
				}
				if instance.PublicIpAddress != nil {
					publicIps = append(publicIps, *instance.PublicIpAddress)
				}
			}
		}
		return !lastPage
	})
	if err != nil {
		return err
	}

	if len(instanceIds) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	log.Printf("[DEBUG] Found %d instances via given filter", len(instanceIds))

	d.SetId(meta.(*conns.AWSClient).Region)

	err = d.Set("ids", instanceIds)
	if err != nil {
		return err
	}

	err = d.Set("private_ips", privateIps)
	if err != nil {
		return err
	}

	err = d.Set("public_ips", publicIps)
	return err
}

package ec2

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func DataSourceNatGateway() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNatGatewayRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"allocation_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"connectivity_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"network_interface_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"public_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"private_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags":   tftags.TagsSchemaComputed(),
			"filter": CustomFiltersSchema(),
		},
	}
}

func dataSourceNatGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	req := &ec2.DescribeNatGatewaysInput{}

	if id, ok := d.GetOk("id"); ok {
		req.NatGatewayIds = aws.StringSlice([]string{id.(string)})
	}

	if vpc_id, ok := d.GetOk("vpc_id"); ok {
		req.Filter = append(req.Filter, BuildAttributeFilterList(
			map[string]string{
				"vpc-id": vpc_id.(string),
			},
		)...)
	}

	if state, ok := d.GetOk("state"); ok {
		req.Filter = append(req.Filter, BuildAttributeFilterList(
			map[string]string{
				"state": state.(string),
			},
		)...)
	}

	if subnet_id, ok := d.GetOk("subnet_id"); ok {
		req.Filter = append(req.Filter, BuildAttributeFilterList(
			map[string]string{
				"subnet-id": subnet_id.(string),
			},
		)...)
	}

	if tags, ok := d.GetOk("tags"); ok {
		req.Filter = append(req.Filter, BuildTagFilterList(
			Tags(tftags.New(tags.(map[string]interface{}))),
		)...)
	}

	req.Filter = append(req.Filter, BuildCustomFilterList(
		d.Get("filter").(*schema.Set),
	)...)
	if len(req.Filter) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filter = nil
	}
	log.Printf("[DEBUG] Reading NAT Gateway: %s", req)
	resp, err := conn.DescribeNatGateways(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.NatGateways) == 0 {
		return fmt.Errorf("no matching NAT gateway found: %s", req)
	}
	if len(resp.NatGateways) > 1 {
		return fmt.Errorf("multiple NAT gateways matched; use additional constraints to reduce matches to a single NAT gateway")
	}

	ngw := resp.NatGateways[0]

	log.Printf("[DEBUG] NAT Gateway response: %s", ngw)

	d.SetId(aws.StringValue(ngw.NatGatewayId))
	d.Set("connectivity_type", ngw.ConnectivityType)
	d.Set("state", ngw.State)
	d.Set("subnet_id", ngw.SubnetId)
	d.Set("vpc_id", ngw.VpcId)

	if err := d.Set("tags", KeyValueTags(ngw.Tags).IgnoreAWS().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	for _, address := range ngw.NatGatewayAddresses {
		if aws.StringValue(address.AllocationId) != "" {
			d.Set("allocation_id", address.AllocationId)
			d.Set("network_interface_id", address.NetworkInterfaceId)
			d.Set("private_ip", address.PrivateIp)
			d.Set("public_ip", address.PublicIp)
			break
		}
	}

	return nil
}

package ec2

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func DataSourceVPCDHCPOptions() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVPCDHCPOptionsRead,

		Schema: map[string]*schema.Schema{
			"dhcp_options_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"domain_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"domain_name_servers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"filter": CustomFiltersSchema(),
			"netbios_name_servers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"netbios_node_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ntp_servers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tags": tftags.TagsSchemaComputed(),
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

func dataSourceVPCDHCPOptionsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	input := &ec2.DescribeDhcpOptionsInput{}

	if v, ok := d.GetOk("dhcp_options_id"); ok {
		input.DhcpOptionsIds = []*string{aws.String(v.(string))}
	}

	input.Filters = append(input.Filters, BuildCustomFilterList(
		d.Get("filter").(*schema.Set),
	)...)
	if len(input.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		input.Filters = nil
	}

	log.Printf("[DEBUG] Reading EC2 DHCP Options: %s", input)
	output, err := conn.DescribeDhcpOptions(input)
	if err != nil {
		if isNoSuchDhcpOptionIDErr(err) {
			return errors.New("No matching EC2 DHCP Options found")
		}
		return fmt.Errorf("error reading EC2 DHCP Options: %w", err)
	}

	if len(output.DhcpOptions) == 0 {
		return errors.New("No matching EC2 DHCP Options found")
	}

	if len(output.DhcpOptions) > 1 {
		return errors.New("Multiple matching EC2 DHCP Options found")
	}

	dhcpOptionID := aws.StringValue(output.DhcpOptions[0].DhcpOptionsId)
	d.SetId(dhcpOptionID)
	d.Set("dhcp_options_id", dhcpOptionID)

	dhcpConfigurations := output.DhcpOptions[0].DhcpConfigurations

	for _, dhcpConfiguration := range dhcpConfigurations {
		key := aws.StringValue(dhcpConfiguration.Key)
		tfKey := strings.Replace(key, "-", "_", -1)

		if len(dhcpConfiguration.Values) == 0 {
			continue
		}

		switch key {
		case "domain-name":
			d.Set(tfKey, dhcpConfiguration.Values[0].Value)
		case "domain-name-servers":
			if err := d.Set(tfKey, flattenAttributeValues(dhcpConfiguration.Values)); err != nil {
				return fmt.Errorf("error setting %s: %w", tfKey, err)
			}
		case "netbios-name-servers":
			if err := d.Set(tfKey, flattenAttributeValues(dhcpConfiguration.Values)); err != nil {
				return fmt.Errorf("error setting %s: %w", tfKey, err)
			}
		case "netbios-node-type":
			d.Set(tfKey, dhcpConfiguration.Values[0].Value)
		case "ntp-servers":
			if err := d.Set(tfKey, flattenAttributeValues(dhcpConfiguration.Values)); err != nil {
				return fmt.Errorf("error setting %s: %w", tfKey, err)
			}
		}
	}

	if err := d.Set("tags", KeyValueTags(output.DhcpOptions[0].Tags).IgnoreAWS().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}
	d.Set("owner_id", output.DhcpOptions[0].OwnerId)

	arn := arn.ARN{
		Partition: meta.(*conns.AWSClient).Partition,
		Service:   ec2.ServiceName,
		Region:    meta.(*conns.AWSClient).Region,
		AccountID: aws.StringValue(output.DhcpOptions[0].OwnerId),
		Resource:  fmt.Sprintf("dhcp-options/%s", d.Id()),
	}.String()

	d.Set("arn", arn)

	return nil
}

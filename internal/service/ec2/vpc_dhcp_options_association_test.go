package ec2_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
)

func TestAccEC2VPCDHCPOptionsAssociation_basic(t *testing.T) {
	var v ec2.Vpc
	var d ec2.DhcpOptions
	resourceName := "aws_vpc_dhcp_options_association.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckDHCPOptionsAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDHCPOptionsAssociationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDHCPOptionsExists("aws_vpc_dhcp_options.test", &d),
					acctest.CheckVPCExists("aws_vpc.test", &v),
					testAccCheckDHCPOptionsAssociationExist(resourceName, &v),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateIdFunc: testAccDHCPOptionsAssociationVPCImportIdFunc(resourceName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEC2VPCDHCPOptionsAssociation_Disappears_vpc(t *testing.T) {
	var v ec2.Vpc
	var d ec2.DhcpOptions
	resourceName := "aws_vpc_dhcp_options_association.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckDHCPOptionsAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDHCPOptionsAssociationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDHCPOptionsExists("aws_vpc_dhcp_options.test", &d),
					acctest.CheckVPCExists("aws_vpc.test", &v),
					testAccCheckDHCPOptionsAssociationExist(resourceName, &v),
					acctest.CheckResourceDisappears(acctest.Provider, tfec2.ResourceVPC(), "aws_vpc.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccEC2VPCDHCPOptionsAssociation_Disappears_dhcp(t *testing.T) {
	var v ec2.Vpc
	var d ec2.DhcpOptions
	resourceName := "aws_vpc_dhcp_options_association.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckDHCPOptionsAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDHCPOptionsAssociationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDHCPOptionsExists("aws_vpc_dhcp_options.test", &d),
					acctest.CheckVPCExists("aws_vpc.test", &v),
					testAccCheckDHCPOptionsAssociationExist(resourceName, &v),
					acctest.CheckResourceDisappears(acctest.Provider, tfec2.ResourceVPCDHCPOptions(), "aws_vpc_dhcp_options.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccEC2VPCDHCPOptionsAssociation_disappears(t *testing.T) {
	var v ec2.Vpc
	var d ec2.DhcpOptions
	resourceName := "aws_vpc_dhcp_options_association.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckDHCPOptionsAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDHCPOptionsAssociationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDHCPOptionsExists("aws_vpc_dhcp_options.test", &d),
					acctest.CheckVPCExists("aws_vpc.test", &v),
					testAccCheckDHCPOptionsAssociationExist(resourceName, &v),
					acctest.CheckResourceDisappears(acctest.Provider, tfec2.ResourceVPCDHCPOptionsAssociation(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccDHCPOptionsAssociationVPCImportIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}

		return rs.Primary.Attributes["vpc_id"], nil
	}
}

func testAccCheckDHCPOptionsAssociationDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpc_dhcp_options_association" {
			continue
		}

		// Try to find the VPC associated to the DHCP Options set
		vpcs, err := tfec2.FindVPCsByDHCPOptionsID(conn, rs.Primary.Attributes["dhcp_options_id"])
		if err != nil {
			return err
		}

		if rs.Primary.Attributes["dhcp_options_id"] != tfec2.VPCDefaultOptionsID && len(vpcs) > 0 {
			return fmt.Errorf("vpc_dhcp_options_association (%s) is still associated to %d VPCs.", rs.Primary.Attributes["dhcp_options_id"], len(vpcs))
		}
	}

	return nil
}

func testAccCheckDHCPOptionsAssociationExist(n string, vpc *ec2.Vpc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No DHCP Options Set association ID is set")
		}

		if aws.StringValue(vpc.DhcpOptionsId) != rs.Primary.Attributes["dhcp_options_id"] {
			return fmt.Errorf("VPC %s does not have DHCP Options Set %s associated",
				aws.StringValue(vpc.VpcId), rs.Primary.Attributes["dhcp_options_id"])
		}

		if aws.StringValue(vpc.VpcId) != rs.Primary.Attributes["vpc_id"] {
			return fmt.Errorf("DHCP Options Set %s is not associated with VPC %s",
				rs.Primary.Attributes["dhcp_options_id"], aws.StringValue(vpc.VpcId))
		}

		return nil
	}
}

const testAccDHCPOptionsAssociationConfig = `
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = "terraform-testacc-vpc-dhcp-options-association"
  }
}

resource "aws_vpc_dhcp_options" "test" {
  domain_name          = "service.consul"
  domain_name_servers  = ["127.0.0.1", "10.0.0.2"]
  ntp_servers          = ["127.0.0.1"]
  netbios_name_servers = ["127.0.0.1"]
  netbios_node_type    = 2

  tags = {
    Name = "terraform-testacc-vpc-dhcp-options-association"
  }
}

resource "aws_vpc_dhcp_options_association" "test" {
  vpc_id          = aws_vpc.test.id
  dhcp_options_id = aws_vpc_dhcp_options.test.id
}
`

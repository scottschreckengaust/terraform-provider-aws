package networkfirewall_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfnetworkfirewall "github.com/hashicorp/terraform-provider-aws/internal/service/networkfirewall"
)

func TestAccNetworkFirewallResourcePolicy_firewallPolicy(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_networkfirewall_resource_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, networkfirewall.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckResourcePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFirewallResourcePolicy_firewallPolicy(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourcePolicyExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "resource_arn", "aws_networkfirewall_firewall_policy.test", "arn"),
					resource.TestMatchResourceAttr(resourceName, "policy", regexp.MustCompile(`"Action":\["network-firewall:CreateFirewall","network-firewall:UpdateFirewall","network-firewall:AssociateFirewallPolicy","network-firewall:ListFirewallPolicies"\]`)),
				),
			},
			{
				// Update the policy's Actions
				Config: testAccNetworkFirewallResourcePolicy_firewallPolicy_updatePolicy(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourcePolicyExists(resourceName),
					resource.TestMatchResourceAttr(resourceName, "policy", regexp.MustCompile(`"Action":\["network-firewall:UpdateFirewall","network-firewall:AssociateFirewallPolicy","network-firewall:ListFirewallPolicies","network-firewall:CreateFirewall"\]`)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNetworkFirewallResourcePolicy_ruleGroup(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_networkfirewall_resource_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, networkfirewall.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckResourcePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFirewallResourcePolicy_ruleGroup(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourcePolicyExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "resource_arn", "aws_networkfirewall_rule_group.test", "arn"),
					resource.TestMatchResourceAttr(resourceName, "policy", regexp.MustCompile(`"Action":\["network-firewall:CreateFirewallPolicy","network-firewall:UpdateFirewallPolicy","network-firewall:ListRuleGroups"\]`)),
				),
			},
			{
				// Update the policy's Actions
				Config: testAccNetworkFirewallResourcePolicy_ruleGroup_updatePolicy(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourcePolicyExists(resourceName),
					resource.TestMatchResourceAttr(resourceName, "policy", regexp.MustCompile(`"Action":\["network-firewall:UpdateFirewallPolicy","network-firewall:ListRuleGroups","network-firewall:CreateFirewallPolicy"\]`)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNetworkFirewallResourcePolicy_disappears(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_networkfirewall_resource_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, networkfirewall.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckResourcePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFirewallResourcePolicy_firewallPolicy(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourcePolicyExists(resourceName),
					acctest.CheckResourceDisappears(acctest.Provider, tfnetworkfirewall.ResourceResourcePolicy(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccNetworkFirewallResourcePolicy_Disappears_firewallPolicy(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_networkfirewall_resource_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, networkfirewall.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckResourcePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFirewallResourcePolicy_firewallPolicy(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourcePolicyExists(resourceName),
					acctest.CheckResourceDisappears(acctest.Provider, tfnetworkfirewall.ResourceFirewallPolicy(), "aws_networkfirewall_firewall_policy.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccNetworkFirewallResourcePolicy_Disappears_ruleGroup(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_networkfirewall_resource_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, networkfirewall.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckResourcePolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFirewallResourcePolicy_ruleGroup(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourcePolicyExists(resourceName),
					acctest.CheckResourceDisappears(acctest.Provider, tfnetworkfirewall.ResourceRuleGroup(), "aws_networkfirewall_rule_group.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckResourcePolicyDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_networkfirewall_resource_policy" {
			continue
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).NetworkFirewallConn
		policy, err := tfnetworkfirewall.FindResourcePolicy(context.Background(), conn, rs.Primary.ID)
		if tfawserr.ErrCodeEquals(err, networkfirewall.ErrCodeResourceNotFoundException) {
			continue
		}
		if err != nil {
			return err
		}
		if policy != nil {
			return fmt.Errorf("NetworkFirewall Resource Policy (for resource: %s) still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckResourcePolicyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No NetworkFirewall Resource Policy ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).NetworkFirewallConn
		policy, err := tfnetworkfirewall.FindResourcePolicy(context.Background(), conn, rs.Primary.ID)
		if err != nil {
			return err
		}

		if policy == nil {
			return fmt.Errorf("NetworkFirewall Resource Policy (for resource: %s) not found", rs.Primary.ID)
		}

		return nil

	}
}

func testAccNetworkFirewallResourcePolicyFirewallPolicyBaseConfig(rName string) string {
	return fmt.Sprintf(`
data "aws_partition" "current" {}

data "aws_caller_identity" "current" {}

resource "aws_networkfirewall_firewall_policy" "test" {
  name = %q
  firewall_policy {
    stateless_fragment_default_actions = ["aws:drop"]
    stateless_default_actions          = ["aws:pass"]
  }
}
`, rName)
}

func testAccNetworkFirewallResourcePolicy_firewallPolicy(rName string) string {
	return acctest.ConfigCompose(
		testAccNetworkFirewallResourcePolicyFirewallPolicyBaseConfig(rName), `
resource "aws_networkfirewall_resource_policy" "test" {
  resource_arn = aws_networkfirewall_firewall_policy.test.arn
  # policy's Action element must include all of the following operations
  policy = jsonencode({
    Statement = [{
      Action = [
        "network-firewall:CreateFirewall",
        "network-firewall:UpdateFirewall",
        "network-firewall:AssociateFirewallPolicy",
        "network-firewall:ListFirewallPolicies"
      ]
      Effect   = "Allow"
      Resource = aws_networkfirewall_firewall_policy.test.arn
      Principal = {
        AWS = "arn:${data.aws_partition.current.partition}:iam::${data.aws_caller_identity.current.account_id}:root"
      }
    }]
    Version = "2012-10-17"
  })
}
`)
}

func testAccNetworkFirewallResourcePolicy_firewallPolicy_updatePolicy(rName string) string {
	return acctest.ConfigCompose(
		testAccNetworkFirewallResourcePolicyFirewallPolicyBaseConfig(rName), `
resource "aws_networkfirewall_resource_policy" "test" {
  resource_arn = aws_networkfirewall_firewall_policy.test.arn
  # policy's Action element must include all of the following operations
  policy = jsonencode({
    Statement = [{
      Action = [
        "network-firewall:UpdateFirewall",
        "network-firewall:AssociateFirewallPolicy",
        "network-firewall:ListFirewallPolicies",
        "network-firewall:CreateFirewall"
      ]
      Effect   = "Allow"
      Resource = aws_networkfirewall_firewall_policy.test.arn
      Principal = {
        AWS = "arn:${data.aws_partition.current.partition}:iam::${data.aws_caller_identity.current.account_id}:root"
      }
    }]
    Version = "2012-10-17"
  })
}
`)
}

func testAccNetworkFirewallResourcePolicyRuleGroupBaseConfig(rName string) string {
	return fmt.Sprintf(`
data "aws_partition" "current" {}

data "aws_caller_identity" "current" {}

resource "aws_networkfirewall_rule_group" "test" {
  capacity = 100
  name     = %q
  type     = "STATEFUL"
  rule_group {
    rules_source {
      rules_source_list {
        generated_rules_type = "ALLOWLIST"
        target_types         = ["HTTP_HOST"]
        targets              = ["test.example.com"]
      }
    }
  }
}
`, rName)
}

func testAccNetworkFirewallResourcePolicy_ruleGroup(rName string) string {
	return acctest.ConfigCompose(
		testAccNetworkFirewallResourcePolicyRuleGroupBaseConfig(rName), `
resource "aws_networkfirewall_resource_policy" "test" {
  resource_arn = aws_networkfirewall_rule_group.test.arn
  # policy's Action element must include all of the following operations
  policy = jsonencode({
    Statement = [{
      Action = [
        "network-firewall:CreateFirewallPolicy",
        "network-firewall:UpdateFirewallPolicy",
        "network-firewall:ListRuleGroups"
      ]
      Effect   = "Allow"
      Resource = aws_networkfirewall_rule_group.test.arn
      Principal = {
        AWS = "arn:${data.aws_partition.current.partition}:iam::${data.aws_caller_identity.current.account_id}:root"
      }
    }]
    Version = "2012-10-17"
  })
}
`)
}

func testAccNetworkFirewallResourcePolicy_ruleGroup_updatePolicy(rName string) string {
	return acctest.ConfigCompose(
		testAccNetworkFirewallResourcePolicyRuleGroupBaseConfig(rName), `
resource "aws_networkfirewall_resource_policy" "test" {
  resource_arn = aws_networkfirewall_rule_group.test.arn
  # policy's Action element must include all of the following operations
  policy = jsonencode({
    Statement = [{
      Action = [
        "network-firewall:UpdateFirewallPolicy",
        "network-firewall:ListRuleGroups",
        "network-firewall:CreateFirewallPolicy"
      ]
      Effect   = "Allow"
      Resource = aws_networkfirewall_rule_group.test.arn
      Principal = {
        AWS = "arn:${data.aws_partition.current.partition}:iam::${data.aws_caller_identity.current.account_id}:root"
      }
    }]
    Version = "2012-10-17"
  })
}
`)
}

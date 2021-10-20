package ecr_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
)

func TestAccECRReplicationConfiguration_basic(t *testing.T) {
	resourceName := "aws_ecr_replication_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ecr.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckReplicationConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccReplicationConfiguration(acctest.AlternateRegion()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckReplicationConfigurationExists(resourceName),
					acctest.CheckResourceAttrAccountID(resourceName, "registry_id"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.0.rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.0.rule.0.destination.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.0.rule.0.destination.0.region", acctest.AlternateRegion()),
					acctest.CheckResourceAttrAccountID(resourceName, "replication_configuration.0.rule.0.destination.0.registry_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccReplicationMultipleRegionConfiguration(acctest.AlternateRegion(), acctest.ThirdRegion()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckReplicationConfigurationExists(resourceName),
					acctest.CheckResourceAttrAccountID(resourceName, "registry_id"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.0.rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.0.rule.0.destination.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.0.rule.0.destination.0.region", acctest.AlternateRegion()),
					acctest.CheckResourceAttrAccountID(resourceName, "replication_configuration.0.rule.0.destination.0.registry_id"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.0.rule.0.destination.1.region", acctest.ThirdRegion()),
					acctest.CheckResourceAttrAccountID(resourceName, "replication_configuration.0.rule.0.destination.1.registry_id"),
				),
			},
			{
				Config: testAccReplicationConfiguration(acctest.AlternateRegion()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckReplicationConfigurationExists(resourceName),
					acctest.CheckResourceAttrAccountID(resourceName, "registry_id"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.0.rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.0.rule.0.destination.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "replication_configuration.0.rule.0.destination.0.region", acctest.AlternateRegion()),
					acctest.CheckResourceAttrAccountID(resourceName, "replication_configuration.0.rule.0.destination.0.registry_id"),
				),
			},
		},
	})
}

func testAccCheckReplicationConfigurationExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).ECRConn
		out, err := conn.DescribeRegistry(&ecr.DescribeRegistryInput{})
		if err != nil {
			return fmt.Errorf("ECR replication rules not found: %w", err)
		}

		if len(out.ReplicationConfiguration.Rules) == 0 {
			return fmt.Errorf("ECR replication rules not found")
		}

		return nil
	}
}

func testAccCheckReplicationConfigurationDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).ECRConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ecr_replication_configuration" {
			continue
		}

		out, err := conn.DescribeRegistry(&ecr.DescribeRegistryInput{})
		if err != nil {
			return err
		}

		if len(out.ReplicationConfiguration.Rules) != 0 {
			return fmt.Errorf("ECR replication rules found")
		}
	}

	return nil
}

func testAccReplicationConfiguration(region string) string {
	return fmt.Sprintf(`
data "aws_caller_identity" "current" {}

resource "aws_ecr_replication_configuration" "test" {
  replication_configuration {
    rule {
      destination {
        region      = %[1]q
        registry_id = data.aws_caller_identity.current.account_id
      }
    }
  }
}
`, region)
}

func testAccReplicationMultipleRegionConfiguration(region1, region2 string) string {
	return fmt.Sprintf(`
data "aws_caller_identity" "current" {}

resource "aws_ecr_replication_configuration" "test" {
  replication_configuration {
    rule {
      destination {
        region      = %[1]q
        registry_id = data.aws_caller_identity.current.account_id
      }


      destination {
        region      = %[2]q
        registry_id = data.aws_caller_identity.current.account_id
      }
    }
  }
}
`, region1, region2)
}

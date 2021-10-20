package ec2_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
)

func TestAccEC2AMILaunchPermission_basic(t *testing.T) {
	resourceName := "aws_ami_launch_permission.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAMILaunchPermissionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAMILaunchPermissionConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAMILaunchPermissionExists(resourceName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccAMILaunchPermissionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEC2AMILaunchPermission_Disappears_launchPermission(t *testing.T) {
	resourceName := "aws_ami_launch_permission.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAMILaunchPermissionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAMILaunchPermissionConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAMILaunchPermissionExists(resourceName),
					testAccCheckAMILaunchPermissionDisappears(resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// Bug reference: https://github.com/hashicorp/terraform-provider-aws/issues/6222
// Images with <group>all</group> will not have <userId> and can cause a panic
func TestAccEC2AMILaunchPermission_DisappearsLaunchPermission_public(t *testing.T) {
	resourceName := "aws_ami_launch_permission.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAMILaunchPermissionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAMILaunchPermissionConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAMILaunchPermissionExists(resourceName),
					testAccCheckAMILaunchPermissionAddPublic(resourceName),
					testAccCheckAMILaunchPermissionDisappears(resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccEC2AMILaunchPermission_Disappears_ami(t *testing.T) {
	imageID := ""
	resourceName := "aws_ami_launch_permission.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAMILaunchPermissionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAMILaunchPermissionConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAMILaunchPermissionExists(resourceName),
				),
			},
			// Here we delete the AMI to verify the follow-on refresh after this step
			// should not error.
			{
				Config: testAccAMILaunchPermissionConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGetAttr("aws_ami_copy.test", "id", &imageID),
					testAccAMIDisappears(&imageID),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testCheckResourceGetAttr(name, key string, value *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("No primary instance: %s", name)
		}

		*value = is.Attributes[key]
		return nil
	}
}

func testAccCheckAMILaunchPermissionExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No resource ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn
		accountID := rs.Primary.Attributes["account_id"]
		imageID := rs.Primary.Attributes["image_id"]

		if has, err := tfec2.HasLaunchPermission(conn, imageID, accountID); err != nil {
			return err
		} else if !has {
			return fmt.Errorf("launch permission does not exist for '%s' on '%s'", accountID, imageID)
		}
		return nil
	}
}

func testAccCheckAMILaunchPermissionDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ami_launch_permission" {
			continue
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn
		accountID := rs.Primary.Attributes["account_id"]
		imageID := rs.Primary.Attributes["image_id"]

		if has, err := tfec2.HasLaunchPermission(conn, imageID, accountID); err != nil {
			return err
		} else if has {
			return fmt.Errorf("launch permission still exists for '%s' on '%s'", accountID, imageID)
		}
	}

	return nil
}

func testAccCheckAMILaunchPermissionAddPublic(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No resource ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn
		imageID := rs.Primary.Attributes["image_id"]

		input := &ec2.ModifyImageAttributeInput{
			ImageId:   aws.String(imageID),
			Attribute: aws.String(ec2.ImageAttributeNameLaunchPermission),
			LaunchPermission: &ec2.LaunchPermissionModifications{
				Add: []*ec2.LaunchPermission{
					{Group: aws.String(ec2.PermissionGroupAll)},
				},
			},
		}

		_, err := conn.ModifyImageAttribute(input)

		return err
	}
}

func testAccCheckAMILaunchPermissionDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No resource ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn
		accountID := rs.Primary.Attributes["account_id"]
		imageID := rs.Primary.Attributes["image_id"]

		input := &ec2.ModifyImageAttributeInput{
			ImageId:   aws.String(imageID),
			Attribute: aws.String(ec2.ImageAttributeNameLaunchPermission),
			LaunchPermission: &ec2.LaunchPermissionModifications{
				Remove: []*ec2.LaunchPermission{
					{UserId: aws.String(accountID)},
				},
			},
		}

		_, err := conn.ModifyImageAttribute(input)

		return err
	}
}

// testAccAMIDisappears is technically a "test check function" but really it
// exists to perform a side effect of deleting an AMI out from under a resource
// so we can test that Terraform will react properly
func testAccAMIDisappears(imageID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn
		req := &ec2.DeregisterImageInput{
			ImageId: aws.String(*imageID),
		}

		_, err := conn.DeregisterImage(req)
		if err != nil {
			return err
		}

		if err := tfec2.AMIWaitForDestroy(tfec2.AMIDeleteRetryTimeout, *imageID, conn); err != nil {
			return err
		}
		return nil
	}
}

func testAccAMILaunchPermissionConfig(rName string) string {
	return fmt.Sprintf(`
data "aws_ami" "amzn-ami-minimal-hvm" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["amzn-ami-minimal-hvm-*"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }
}

data "aws_caller_identity" "current" {}

data "aws_region" "current" {}

resource "aws_ami_copy" "test" {
  description       = %q
  name              = %q
  source_ami_id     = data.aws_ami.amzn-ami-minimal-hvm.id
  source_ami_region = data.aws_region.current.name
}

resource "aws_ami_launch_permission" "test" {
  account_id = data.aws_caller_identity.current.account_id
  image_id   = aws_ami_copy.test.id
}
`, rName, rName)
}

func testAccAMILaunchPermissionImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}

		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["account_id"], rs.Primary.Attributes["image_id"]), nil
	}
}

package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/directconnect/finder"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

func TestAccAWSDxConnectionAssociation_basic(t *testing.T) {
	resourceName := "aws_dx_connection_association.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, directconnect.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsDxConnectionAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDxConnectionAssociationConfigBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsDxConnectionAssociationExists(resourceName),
				),
			},
		},
	})
}

func TestAccAWSDxConnectionAssociation_LAGOnConnection(t *testing.T) {
	resourceName := "aws_dx_connection_association.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, directconnect.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsDxConnectionAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDxConnectionAssociationConfigLAGOnConnection(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsDxConnectionAssociationExists(resourceName),
				),
			},
		},
	})
}

func TestAccAWSDxConnectionAssociation_Multiple(t *testing.T) {
	resourceName1 := "aws_dx_connection_association.test1"
	resourceName2 := "aws_dx_connection_association.test2"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, directconnect.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsDxConnectionAssociationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDxConnectionAssociationConfigMultiple(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsDxConnectionAssociationExists(resourceName1),
					testAccCheckAwsDxConnectionAssociationExists(resourceName2),
				),
			},
		},
	})
}

func testAccCheckAwsDxConnectionAssociationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).dxconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_dx_connection_association" {
			continue
		}

		err := finder.ConnectionAssociationExists(conn, rs.Primary.ID, rs.Primary.Attributes["lag_id"])

		if tfresource.NotFound(err) {
			continue
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("Direct Connect Connection (%s) LAG (%s) Association still exists", rs.Primary.ID, rs.Primary.Attributes["lag_id"])
	}

	return nil
}

func testAccCheckAwsDxConnectionAssociationExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).dxconn

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		err := finder.ConnectionAssociationExists(conn, rs.Primary.ID, rs.Primary.Attributes["lag_id"])

		if err != nil {
			return err
		}

		return nil
	}
}

func testAccDxConnectionAssociationConfigBasic(rName string) string {
	return fmt.Sprintf(`
data "aws_dx_locations" "test" {}

resource "aws_dx_connection" "test" {
  name      = %[1]q
  bandwidth = "1Gbps"
  location  = tolist(data.aws_dx_locations.test.location_codes)[0]
}

resource "aws_dx_lag" "test" {
  name                  = %[1]q
  connections_bandwidth = "1Gbps"
  location              = tolist(data.aws_dx_locations.test.location_codes)[0]
  force_destroy         = true
}

resource "aws_dx_connection_association" "test" {
  connection_id = aws_dx_connection.test.id
  lag_id        = aws_dx_lag.test.id
}
`, rName)
}

func testAccDxConnectionAssociationConfigLAGOnConnection(rName string) string {
	return fmt.Sprintf(`
data "aws_dx_locations" "test" {}

resource "aws_dx_connection" "test1" {
  name      = "%[1]s-1"
  bandwidth = "1Gbps"
  location  = tolist(data.aws_dx_locations.test.location_codes)[0]
}

resource "aws_dx_connection" "test2" {
  name      = "%[1]s-2"
  bandwidth = "1Gbps"
  location  = tolist(data.aws_dx_locations.test.location_codes)[0]
}

resource "aws_dx_lag" "test" {
  name                  = %[1]q
  connection_id         = aws_dx_connection.test1.id
  connections_bandwidth = "1Gbps"
  location              = tolist(data.aws_dx_locations.test.location_codes)[0]
}

resource "aws_dx_connection_association" "test" {
  connection_id = aws_dx_connection.test2.id
  lag_id        = aws_dx_lag.test.id
}
`, rName)
}

func testAccDxConnectionAssociationConfigMultiple(rName string) string {
	return fmt.Sprintf(`
data "aws_dx_locations" "test" {}

resource "aws_dx_connection" "test1" {
  name      = "%[1]s-1"
  bandwidth = "1Gbps"
  location  = tolist(data.aws_dx_locations.test.location_codes)[0]
}

resource "aws_dx_connection" "test2" {
  name      = "%[1]s-2"
  bandwidth = "1Gbps"
  location  = tolist(data.aws_dx_locations.test.location_codes)[0]
}

resource "aws_dx_lag" "test" {
  name                  = %[1]q
  connections_bandwidth = "1Gbps"
  location              = tolist(data.aws_dx_locations.test.location_codes)[0]
  force_destroy         = true
}

resource "aws_dx_connection_association" "test1" {
  connection_id = aws_dx_connection.test1.id
  lag_id        = aws_dx_lag.test.id
}

resource "aws_dx_connection_association" "test2" {
  connection_id = aws_dx_connection.test2.id
  lag_id        = aws_dx_lag.test.id
}
`, rName)
}

package ec2_test

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
)

func TestAccEC2CapacityReservation_basic(t *testing.T) {
	var cr ec2.CapacityReservation
	availabilityZonesDataSourceName := "data.aws_availability_zones.available"
	resourceName := "aws_ec2_capacity_reservation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckCapacityReservation(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckEc2CapacityReservationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEc2CapacityReservationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					acctest.MatchResourceAttrRegionalARN(resourceName, "arn", "ec2", regexp.MustCompile(`capacity-reservation/cr-.+`)),
					resource.TestCheckResourceAttrPair(resourceName, "availability_zone", availabilityZonesDataSourceName, "names.0"),
					resource.TestCheckResourceAttr(resourceName, "ebs_optimized", "false"),
					resource.TestCheckResourceAttr(resourceName, "end_date", ""),
					resource.TestCheckResourceAttr(resourceName, "end_date_type", "unlimited"),
					resource.TestCheckResourceAttr(resourceName, "ephemeral_storage", "false"),
					resource.TestCheckResourceAttr(resourceName, "instance_count", "1"),
					resource.TestCheckResourceAttr(resourceName, "instance_match_criteria", "open"),
					resource.TestCheckResourceAttr(resourceName, "instance_platform", "Linux/UNIX"),
					resource.TestCheckResourceAttr(resourceName, "instance_type", "t2.micro"),
					resource.TestCheckResourceAttr(resourceName, "outpost_arn", ""),
					acctest.CheckResourceAttrAccountID(resourceName, "owner_id"),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "tenancy", "default"),
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

func TestAccEC2CapacityReservation_ebsOptimized(t *testing.T) {
	var cr ec2.CapacityReservation
	resourceName := "aws_ec2_capacity_reservation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckCapacityReservation(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckEc2CapacityReservationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEc2CapacityReservationConfig_ebsOptimized(true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "ebs_optimized", "true"),
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

func TestAccEC2CapacityReservation_endDate(t *testing.T) {
	var cr ec2.CapacityReservation
	endDate1 := time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)
	endDate2 := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	resourceName := "aws_ec2_capacity_reservation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckCapacityReservation(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckEc2CapacityReservationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEc2CapacityReservationConfig_endDate(endDate1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "end_date", endDate1),
					resource.TestCheckResourceAttr(resourceName, "end_date_type", "limited"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEc2CapacityReservationConfig_endDate(endDate2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "end_date", endDate2),
					resource.TestCheckResourceAttr(resourceName, "end_date_type", "limited"),
				),
			},
		},
	})
}

func TestAccEC2CapacityReservation_endDateType(t *testing.T) {
	var cr ec2.CapacityReservation
	endDate := time.Now().UTC().Add(12 * time.Hour).Format(time.RFC3339)
	resourceName := "aws_ec2_capacity_reservation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckCapacityReservation(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckEc2CapacityReservationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEc2CapacityReservationConfig_endDateType("unlimited"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "end_date_type", "unlimited"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEc2CapacityReservationConfig_endDate(endDate),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "end_date", endDate),
					resource.TestCheckResourceAttr(resourceName, "end_date_type", "limited"),
				),
			},
			{
				Config: testAccEc2CapacityReservationConfig_endDateType("unlimited"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "end_date_type", "unlimited"),
				),
			},
		},
	})
}

func TestAccEC2CapacityReservation_ephemeralStorage(t *testing.T) {
	var cr ec2.CapacityReservation
	resourceName := "aws_ec2_capacity_reservation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckCapacityReservation(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckEc2CapacityReservationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEc2CapacityReservationConfig_ephemeralStorage(true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "ephemeral_storage", "true"),
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

func TestAccEC2CapacityReservation_instanceCount(t *testing.T) {
	var cr ec2.CapacityReservation
	resourceName := "aws_ec2_capacity_reservation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckCapacityReservation(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckEc2CapacityReservationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEc2CapacityReservationConfig_instanceCount(1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "instance_count", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEc2CapacityReservationConfig_instanceCount(2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "instance_count", "2"),
				),
			},
		},
	})
}

func TestAccEC2CapacityReservation_instanceMatchCriteria(t *testing.T) {
	var cr ec2.CapacityReservation
	resourceName := "aws_ec2_capacity_reservation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckCapacityReservation(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckEc2CapacityReservationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEc2CapacityReservationConfig_instanceMatchCriteria("targeted"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "instance_match_criteria", "targeted"),
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

func TestAccEC2CapacityReservation_instanceType(t *testing.T) {
	var cr ec2.CapacityReservation
	resourceName := "aws_ec2_capacity_reservation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckCapacityReservation(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckEc2CapacityReservationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEc2CapacityReservationConfig_instanceType("t2.micro"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "instance_type", "t2.micro"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEc2CapacityReservationConfig_instanceType("t2.small"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "instance_type", "t2.small"),
				),
			},
		},
	})
}

func TestAccEC2CapacityReservation_tags(t *testing.T) {
	var cr ec2.CapacityReservation
	resourceName := "aws_ec2_capacity_reservation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckCapacityReservation(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckEc2CapacityReservationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEc2CapacityReservationConfig_tags_single("key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEc2CapacityReservationConfig_tags_multiple("key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccEc2CapacityReservationConfig_tags_single("key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func TestAccEC2CapacityReservation_disappears(t *testing.T) {
	var cr ec2.CapacityReservation
	resourceName := "aws_ec2_capacity_reservation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckCapacityReservation(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckEc2CapacityReservationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEc2CapacityReservationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					acctest.CheckResourceDisappears(acctest.Provider, tfec2.ResourceCapacityReservation(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccEC2CapacityReservation_tenancy(t *testing.T) {
	// Error creating EC2 Capacity Reservation: Unsupported: The requested configuration is currently not supported. Please check the documentation for supported configurations.
	acctest.Skip(t, "EC2 Capacity Reservations do not currently support dedicated tenancy.")
	var cr ec2.CapacityReservation
	resourceName := "aws_ec2_capacity_reservation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t); testAccPreCheckCapacityReservation(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckEc2CapacityReservationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEc2CapacityReservationConfig_tenancy("dedicated"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEc2CapacityReservationExists(resourceName, &cr),
					resource.TestCheckResourceAttr(resourceName, "tenancy", "dedicated"),
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

func testAccCheckEc2CapacityReservationExists(resourceName string, cr *ec2.CapacityReservation) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn
		resp, err := conn.DescribeCapacityReservations(&ec2.DescribeCapacityReservationsInput{
			CapacityReservationIds: []*string{aws.String(rs.Primary.ID)},
		})

		if err != nil {
			return fmt.Errorf("Error retrieving EC2 Capacity Reservations: %s", err)
		}

		if len(resp.CapacityReservations) == 0 {
			return fmt.Errorf("EC2 Capacity Reservation (%s) not found", rs.Primary.ID)
		}

		reservation := resp.CapacityReservations[0]

		if aws.StringValue(reservation.State) != ec2.CapacityReservationStateActive && aws.StringValue(reservation.State) != ec2.CapacityReservationStatePending {
			return fmt.Errorf("EC2 Capacity Reservation (%s) found in unexpected state: %s", rs.Primary.ID, aws.StringValue(reservation.State))
		}

		*cr = *reservation
		return nil
	}
}

func testAccCheckEc2CapacityReservationDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ec2_capacity_reservation" {
			continue
		}

		// Try to find the resource
		resp, err := conn.DescribeCapacityReservations(&ec2.DescribeCapacityReservationsInput{
			CapacityReservationIds: []*string{aws.String(rs.Primary.ID)},
		})
		if err == nil {
			for _, r := range resp.CapacityReservations {
				if aws.StringValue(r.State) != ec2.CapacityReservationStateCancelled && aws.StringValue(r.State) != ec2.CapacityReservationStateExpired {
					return fmt.Errorf("Found uncancelled EC2 Capacity Reservation: %s", r)
				}
			}
		}

		return err
	}

	return nil

}

func testAccPreCheckCapacityReservation(t *testing.T) {
	conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

	input := &ec2.DescribeCapacityReservationsInput{
		MaxResults: aws.Int64(1),
	}

	_, err := conn.DescribeCapacityReservations(input)

	if acctest.PreCheckSkipError(err) {
		t.Skipf("skipping acceptance testing: %s", err)
	}

	if err != nil {
		t.Fatalf("unexpected PreCheck error: %s", err)
	}
}

const testAccEc2CapacityReservationConfig = `
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_ec2_capacity_reservation" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  instance_count    = 1
  instance_platform = "Linux/UNIX"
  instance_type     = "t2.micro"
}
`

func testAccEc2CapacityReservationConfig_ebsOptimized(ebsOptimized bool) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_ec2_capacity_reservation" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  ebs_optimized     = %t
  instance_count    = 1
  instance_platform = "Linux/UNIX"
  instance_type     = "m4.large"
}
`, ebsOptimized)
}

func testAccEc2CapacityReservationConfig_endDate(endDate string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_ec2_capacity_reservation" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  end_date          = %q
  end_date_type     = "limited"
  instance_count    = 1
  instance_platform = "Linux/UNIX"
  instance_type     = "t2.micro"
}
`, endDate)
}

func testAccEc2CapacityReservationConfig_endDateType(endDateType string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_ec2_capacity_reservation" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  end_date_type     = %q
  instance_count    = 1
  instance_platform = "Linux/UNIX"
  instance_type     = "t2.micro"
}
`, endDateType)
}

func testAccEc2CapacityReservationConfig_ephemeralStorage(ephemeralStorage bool) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_ec2_capacity_reservation" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  ephemeral_storage = %t
  instance_count    = 1
  instance_platform = "Linux/UNIX"
  instance_type     = "m3.medium"
}
`, ephemeralStorage)
}

func testAccEc2CapacityReservationConfig_instanceCount(instanceCount int) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_ec2_capacity_reservation" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  instance_count    = %d
  instance_platform = "Linux/UNIX"
  instance_type     = "t2.micro"
}
`, instanceCount)
}

func testAccEc2CapacityReservationConfig_instanceMatchCriteria(instanceMatchCriteria string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_ec2_capacity_reservation" "test" {
  availability_zone       = data.aws_availability_zones.available.names[0]
  instance_count          = 1
  instance_platform       = "Linux/UNIX"
  instance_match_criteria = %q
  instance_type           = "t2.micro"
}
`, instanceMatchCriteria)
}

func testAccEc2CapacityReservationConfig_instanceType(instanceType string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_ec2_capacity_reservation" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  instance_count    = 1
  instance_platform = "Linux/UNIX"
  instance_type     = %q
}
`, instanceType)
}

func testAccEc2CapacityReservationConfig_tags_single(tag1Key, tag1Value string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_ec2_capacity_reservation" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  instance_count    = 1
  instance_platform = "Linux/UNIX"
  instance_type     = "t2.micro"

  tags = {
    %q = %q
  }
}
`, tag1Key, tag1Value)
}

func testAccEc2CapacityReservationConfig_tags_multiple(tag1Key, tag1Value, tag2Key, tag2Value string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_ec2_capacity_reservation" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  instance_count    = 1
  instance_platform = "Linux/UNIX"
  instance_type     = "t2.micro"

  tags = {
    %q = %q
    %q = %q
  }
}
`, tag1Key, tag1Value, tag2Key, tag2Value)
}

func testAccEc2CapacityReservationConfig_tenancy(tenancy string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_ec2_capacity_reservation" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  instance_count    = 1
  instance_platform = "Linux/UNIX"
  instance_type     = "t2.micro"
  tenancy           = %q
}
`, tenancy)
}

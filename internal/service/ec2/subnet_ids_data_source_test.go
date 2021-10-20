package ec2_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func TestAccEC2SubnetIDsDataSource_basic(t *testing.T) {
	rInt := sdkacctest.RandIntRange(0, 256)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetIDsDataSourceConfig(rInt),
			},
			{
				Config: testAccSubnetIDsWithDataSourceDataSourceConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_subnet_ids.selected", "ids.#", "3"),
					resource.TestCheckResourceAttr("data.aws_subnet_ids.private", "ids.#", "2"),
				),
			},
		},
	})
}

func TestAccEC2SubnetIDsDataSource_filter(t *testing.T) {
	rInt := sdkacctest.RandIntRange(0, 256)
	rName := "data.aws_subnet_ids.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckVpcDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetIDsDataSource_filter(rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "ids.#", "2"),
				),
			},
		},
	})
}

func testAccSubnetIDsWithDataSourceDataSourceConfig(rInt int) string {
	return acctest.ConfigCompose(acctest.ConfigAvailableAZsNoOptIn(), fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "172.%[1]d.0.0/16"

  tags = {
    Name = "terraform-testacc-subnet-ids-data-source"
  }
}

resource "aws_subnet" "test_public_a" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "172.%[1]d.123.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "tf-acc-subnet-ids-data-source-public-a"
    Tier = "Public"
  }
}

resource "aws_subnet" "test_private_a" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "172.%[1]d.125.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "tf-acc-subnet-ids-data-source-private-a"
    Tier = "Private"
  }
}

resource "aws_subnet" "test_private_b" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "172.%[1]d.126.0/24"
  availability_zone = data.aws_availability_zones.available.names[1]

  tags = {
    Name = "tf-acc-subnet-ids-data-source-private-b"
    Tier = "Private"
  }
}

data "aws_subnet_ids" "selected" {
  vpc_id = aws_vpc.test.id
}

data "aws_subnet_ids" "private" {
  vpc_id = aws_vpc.test.id

  tags = {
    Tier = "Private"
  }
}
`, rInt))
}

func testAccSubnetIDsDataSourceConfig(rInt int) string {
	return acctest.ConfigCompose(acctest.ConfigAvailableAZsNoOptIn(), fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "172.%[1]d.0.0/16"

  tags = {
    Name = "terraform-testacc-subnet-ids-data-source"
  }
}

resource "aws_subnet" "test_public_a" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "172.%[1]d.123.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "tf-acc-subnet-ids-data-source-public-a"
    Tier = "Public"
  }
}

resource "aws_subnet" "test_private_a" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "172.%[1]d.125.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "tf-acc-subnet-ids-data-source-private-a"
    Tier = "Private"
  }
}

resource "aws_subnet" "test_private_b" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "172.%[1]d.126.0/24"
  availability_zone = data.aws_availability_zones.available.names[1]

  tags = {
    Name = "tf-acc-subnet-ids-data-source-private-b"
    Tier = "Private"
  }
}
`, rInt))
}

func testAccSubnetIDsDataSource_filter(rInt int) string {
	return acctest.ConfigCompose(acctest.ConfigAvailableAZsNoOptIn(), fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "172.%[1]d.0.0/16"

  tags = {
    Name = "terraform-testacc-subnet-ids-data-source"
  }
}

resource "aws_subnet" "test_a_one" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "172.%[1]d.1.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]
}

resource "aws_subnet" "test_a_two" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "172.%[1]d.2.0/24"
  availability_zone = data.aws_availability_zones.available.names[0]
}

resource "aws_subnet" "test_b" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = "172.%[1]d.3.0/24"
  availability_zone = data.aws_availability_zones.available.names[1]
}

data "aws_subnet_ids" "test" {
  vpc_id = aws_subnet.test_a_two.vpc_id

  filter {
    name   = "availabilityZone"
    values = [aws_subnet.test_a_one.availability_zone]
  }
}
`, rInt))
}

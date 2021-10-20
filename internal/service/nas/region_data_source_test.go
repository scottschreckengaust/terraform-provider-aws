package nas_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	tfnas "github.com/hashicorp/terraform-provider-aws/internal/service/nas"
)

func TestFindRegionByEc2Endpoint(t *testing.T) {
	var testCases = []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "does-not-exist",
			ErrCount: 1,
		},
		{
			Value:    "ec2.does-not-exist.amazonaws.com",
			ErrCount: 1,
		},
		{
			Value:    "us-east-1", // lintignore:AWSAT003
			ErrCount: 1,
		},
		{
			Value:    "ec2.us-east-1.amazonaws.com", // lintignore:AWSAT003
			ErrCount: 0,
		},
	}

	for _, tc := range testCases {
		_, err := tfnas.FindRegionByEndpoint(tc.Value)
		if tc.ErrCount == 0 && err != nil {
			t.Fatalf("expected %q not to trigger an error, received: %s", tc.Value, err)
		}
		if tc.ErrCount > 0 && err == nil {
			t.Fatalf("expected %q to trigger an error", tc.Value)
		}
	}
}

func TestFindRegionByName(t *testing.T) {
	var testCases = []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "does-not-exist",
			ErrCount: 1,
		},
		{
			Value:    "ec2.us-east-1.amazonaws.com", // lintignore:AWSAT003
			ErrCount: 1,
		},
		{
			Value:    "us-east-1", // lintignore:AWSAT003
			ErrCount: 0,
		},
	}

	for _, tc := range testCases {
		_, err := tfnas.FindRegionByName(tc.Value)
		if tc.ErrCount == 0 && err != nil {
			t.Fatalf("expected %q not to trigger an error, received: %s", tc.Value, err)
		}
		if tc.ErrCount > 0 && err == nil {
			t.Fatalf("expected %q to trigger an error", tc.Value)
		}
	}
}

func TestAccNASRegionDataSource_basic(t *testing.T) {
	dataSourceName := "data.aws_region.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { acctest.PreCheck(t) },
		ErrorCheck: acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:  acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccRegionDataSourceConfig_empty,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(dataSourceName, "description", regexp.MustCompile(`^.+$`)),
					acctest.CheckResourceAttrRegionalHostnameService(dataSourceName, "endpoint", ec2.EndpointsID),
					resource.TestCheckResourceAttr(dataSourceName, "name", acctest.Region()),
				),
			},
		},
	})
}

func TestAccNASRegionDataSource_endpoint(t *testing.T) {
	dataSourceName := "data.aws_region.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { acctest.PreCheck(t) },
		ErrorCheck: acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:  acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccRegionDataSourceConfig_endpoint(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(dataSourceName, "description", regexp.MustCompile(`^.+$`)),
					resource.TestMatchResourceAttr(dataSourceName, "endpoint", regexp.MustCompile(fmt.Sprintf("^ec2\\.[^.]+\\.%s$", acctest.PartitionDNSSuffix()))),
					resource.TestMatchResourceAttr(dataSourceName, "name", regexp.MustCompile(`^.+$`)),
				),
			},
		},
	})
}

func TestAccNASRegionDataSource_endpointAndName(t *testing.T) {
	dataSourceName := "data.aws_region.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { acctest.PreCheck(t) },
		ErrorCheck: acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:  acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccRegionDataSourceConfig_endpointAndName(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(dataSourceName, "description", regexp.MustCompile(`^.+$`)),
					resource.TestMatchResourceAttr(dataSourceName, "endpoint", regexp.MustCompile(fmt.Sprintf("^ec2\\.[^.]+\\.%s$", acctest.PartitionDNSSuffix()))),
					resource.TestMatchResourceAttr(dataSourceName, "name", regexp.MustCompile(`^.+$`)),
				),
			},
		},
	})
}

func TestAccNASRegionDataSource_name(t *testing.T) {
	dataSourceName := "data.aws_region.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { acctest.PreCheck(t) },
		ErrorCheck: acctest.ErrorCheck(t, ec2.EndpointsID),
		Providers:  acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccRegionDataSourceConfig_name(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(dataSourceName, "description", regexp.MustCompile(`^.+$`)),
					resource.TestMatchResourceAttr(dataSourceName, "endpoint", regexp.MustCompile(fmt.Sprintf("^ec2\\.[^.]+\\.%s$", acctest.PartitionDNSSuffix()))),
					resource.TestMatchResourceAttr(dataSourceName, "name", regexp.MustCompile(`^.+$`)),
				),
			},
		},
	})
}

const testAccRegionDataSourceConfig_empty = `
data "aws_region" "test" {}
`

func testAccRegionDataSourceConfig_endpoint() string {
	return `
data "aws_partition" "test" {}

data "aws_regions" "test" {
}

data "aws_region" "test" {
  endpoint = "ec2.${tolist(data.aws_regions.test.names)[0]}.${data.aws_partition.test.dns_suffix}"
}
`
}

func testAccRegionDataSourceConfig_endpointAndName() string {
	return `
data "aws_partition" "test" {}

data "aws_regions" "test" {
}

data "aws_region" "test" {
  endpoint = "ec2.${tolist(data.aws_regions.test.names)[0]}.${data.aws_partition.test.dns_suffix}"
  name     = tolist(data.aws_regions.test.names)[0]
}
`
}

func testAccRegionDataSourceConfig_name() string {
	return `
data "aws_regions" "test" {
}

data "aws_region" "test" {
  name = tolist(data.aws_regions.test.names)[0]
}
`
}

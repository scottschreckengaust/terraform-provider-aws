package codestarconnections_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/codestarconnections"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func TestAccCodeStarConnectionsConnectionDataSource_basic(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	dataSourceName := "data.aws_codestarconnections_connection.test"
	resourceName := "aws_codestarconnections_connection.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(codestarconnections.EndpointsID, t) },
		ErrorCheck: acctest.ErrorCheck(t, codestarconnections.EndpointsID),
		Providers:  acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionBasicDataSourceConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "id", dataSourceName, "id"),
					resource.TestCheckResourceAttrPair(resourceName, "arn", dataSourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "provider_type", dataSourceName, "provider_type"),
					resource.TestCheckResourceAttrPair(resourceName, "name", dataSourceName, "name"),
					resource.TestCheckResourceAttrPair(resourceName, "connection_status", dataSourceName, "connection_status"),
					resource.TestCheckResourceAttrPair(resourceName, "tags.%", dataSourceName, "tags.%"),
				),
			},
		},
	})
}

func TestAccCodeStarConnectionsConnectionDataSource_tags(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	dataSourceName := "data.aws_codestarconnections_connection.test"
	resourceName := "aws_codestarconnections_connection.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(codestarconnections.EndpointsID, t) },
		ErrorCheck: acctest.ErrorCheck(t, codestarconnections.EndpointsID),
		Providers:  acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectionTagsDataSourceConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "tags.%", dataSourceName, "tags.%"),
				),
			},
		},
	})
}

func testAccConnectionBasicDataSourceConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_codestarconnections_connection" "test" {
  name          = %[1]q
  provider_type = "Bitbucket"
}

data "aws_codestarconnections_connection" "test" {
  arn = aws_codestarconnections_connection.test.arn
}
`, rName)
}

func testAccConnectionTagsDataSourceConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_codestarconnections_connection" "test" {
  name          = %[1]q
  provider_type = "Bitbucket"

  tags = {
    "key1" = "value1"
    "key2" = "value2"
  }
}

data "aws_codestarconnections_connection" "test" {
  arn = aws_codestarconnections_connection.test.arn
}
`, rName)
}

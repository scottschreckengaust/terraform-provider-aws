package dynamodb_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func TestAccDynamoDBTableDataSource_basic(t *testing.T) {
	datasourceName := "data.aws_dynamodb_table.test"
	tableName := fmt.Sprintf("testAccAWSdynamodbtable-basic-%s", sdkacctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { acctest.PreCheck(t) },
		ErrorCheck: acctest.ErrorCheck(t, dynamodb.EndpointsID),
		Providers:  acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccTableBasicDataSourceConfig(tableName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(datasourceName, "name", tableName),
					resource.TestCheckResourceAttr(datasourceName, "read_capacity", "20"),
					resource.TestCheckResourceAttr(datasourceName, "write_capacity", "20"),
					resource.TestCheckResourceAttr(datasourceName, "hash_key", "UserId"),
					resource.TestCheckResourceAttr(datasourceName, "range_key", "GameTitle"),
					resource.TestCheckResourceAttr(datasourceName, "attribute.#", "3"),
					resource.TestCheckResourceAttr(datasourceName, "global_secondary_index.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "ttl.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(datasourceName, "tags.Name", "dynamodb-table-1"),
					resource.TestCheckResourceAttr(datasourceName, "tags.Environment", "test"),
					resource.TestCheckResourceAttr(datasourceName, "server_side_encryption.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "billing_mode", "PROVISIONED"),
					resource.TestCheckResourceAttr(datasourceName, "point_in_time_recovery.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "point_in_time_recovery.0.enabled", "false"),
				),
			},
		},
	})
}

func testAccTableBasicDataSourceConfig(tableName string) string {
	return fmt.Sprintf(`
resource "aws_dynamodb_table" "test" {
  name           = "%s"
  read_capacity  = 20
  write_capacity = 20
  hash_key       = "UserId"
  range_key      = "GameTitle"

  attribute {
    name = "UserId"
    type = "S"
  }

  attribute {
    name = "GameTitle"
    type = "S"
  }

  attribute {
    name = "TopScore"
    type = "N"
  }

  global_secondary_index {
    name               = "GameTitleIndex"
    hash_key           = "GameTitle"
    range_key          = "TopScore"
    write_capacity     = 10
    read_capacity      = 10
    projection_type    = "INCLUDE"
    non_key_attributes = ["UserId"]
  }

  tags = {
    Name        = "dynamodb-table-1"
    Environment = "test"
  }
}

data "aws_dynamodb_table" "test" {
  name = aws_dynamodb_table.test.name
}
`, tableName)
}

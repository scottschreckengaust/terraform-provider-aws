package qldb_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/qldb"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func TestAccQLDBLedgerDataSource_basic(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-test-%s", sdkacctest.RandString(7)) // QLDB name cannot be longer than 32 characters

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(qldb.EndpointsID, t) },
		ErrorCheck: acctest.ErrorCheck(t, qldb.EndpointsID),
		Providers:  acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccLedgerDataSourceConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.aws_qldb_ledger.by_name", "arn", "aws_qldb_ledger.tf_test", "arn"),
					resource.TestCheckResourceAttrPair("data.aws_qldb_ledger.by_name", "deletion_protection", "aws_qldb_ledger.tf_test", "deletion_protection"),
					resource.TestCheckResourceAttrPair("data.aws_qldb_ledger.by_name", "name", "aws_qldb_ledger.tf_test", "name"),
					resource.TestCheckResourceAttrPair("data.aws_qldb_ledger.by_name", "permissions_mode", "aws_qldb_ledger.tf_test", "permissions_mode"),
				),
			},
		},
	})
}

func testAccLedgerDataSourceConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_qldb_ledger" "tf_wrong1" {
  name                = "%[1]s1"
  permissions_mode    = "STANDARD"
  deletion_protection = false
}

resource "aws_qldb_ledger" "tf_test" {
  name                = "%[1]s2"
  permissions_mode    = "STANDARD"
  deletion_protection = false
}

resource "aws_qldb_ledger" "tf_wrong2" {
  name                = "%[1]s3"
  permissions_mode    = "STANDARD"
  deletion_protection = false
}

data "aws_qldb_ledger" "by_name" {
  name = aws_qldb_ledger.tf_test.name
}
`, rName)
}

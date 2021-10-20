package rds_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/rds"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func TestAccRDSSnapshotDataSource_basic(t *testing.T) {
	rInt := sdkacctest.RandInt()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:   func() { acctest.PreCheck(t) },
		ErrorCheck: acctest.ErrorCheck(t, rds.EndpointsID),
		Providers:  acctest.Providers,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSnapshotDataSourceConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSnapshotIDDataSource("data.aws_db_snapshot.snapshot"),
				),
			},
		},
	})
}

func testAccCheckSnapshotIDDataSource(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find Volume data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Snapshot data source ID not set")
		}
		return nil
	}
}

func testAccCheckSnapshotDataSourceConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_db_instance" "bar" {
  allocated_storage   = 10
  engine              = "mysql"
  engine_version      = "5.6.35"
  instance_class      = "db.t2.micro"
  name                = "baz"
  password            = "barbarbarbar"
  username            = "foo"
  skip_final_snapshot = true

  # Maintenance Window is stored in lower case in the API, though not strictly
  # documented. Terraform will downcase this to match (as opposed to throw a
  # validation error).
  maintenance_window = "Fri:09:00-Fri:09:30"

  backup_retention_period = 0

  parameter_group_name = "default.mysql5.6"
}

data "aws_db_snapshot" "snapshot" {
  most_recent            = "true"
  db_snapshot_identifier = aws_db_snapshot.test.id
}

resource "aws_db_snapshot" "test" {
  db_instance_identifier = aws_db_instance.bar.id
  db_snapshot_identifier = "testsnapshot%d"
}
`, rInt)
}

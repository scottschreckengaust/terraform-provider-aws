package codecommit_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codecommit"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
)

func TestAccCodeCommitTrigger_basic(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_codecommit_trigger.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, codecommit.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckCodeCommitTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCodeCommitTrigger_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitTriggerExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "trigger.#", "1"),
				),
			},
		},
	})
}

func testAccCheckCodeCommitTriggerDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).CodeCommitConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_codecommit_trigger" {
			continue
		}

		_, err := conn.GetRepositoryTriggers(&codecommit.GetRepositoryTriggersInput{
			RepositoryName: aws.String(rs.Primary.ID),
		})

		if ae, ok := err.(awserr.Error); ok && ae.Code() == "RepositoryDoesNotExistException" {
			continue
		}
		if err == nil {
			return fmt.Errorf("Trigger still exists: %s", rs.Primary.ID)
		}
		return err
	}

	return nil
}

func testAccCheckCodeCommitTriggerExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).CodeCommitConn
		out, err := conn.GetRepositoryTriggers(&codecommit.GetRepositoryTriggersInput{
			RepositoryName: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if len(out.Triggers) == 0 {
			return fmt.Errorf("CodeCommit Trigger Failed: %q", out)
		}

		return nil
	}
}

func testAccCodeCommitTrigger_basic(rName string) string {
	return fmt.Sprintf(`
resource "aws_sns_topic" "test" {
  name = %[1]q
}

resource "aws_codecommit_repository" "test" {
  repository_name = %[1]q
}

resource "aws_codecommit_trigger" "test" {
  repository_name = aws_codecommit_repository.test.id

  trigger {
    name            = %[1]q
    events          = ["all"]
    destination_arn = aws_sns_topic.test.arn
  }
}
`, rName)
}

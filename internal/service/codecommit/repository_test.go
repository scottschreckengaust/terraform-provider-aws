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

func TestAccCodeCommitRepository_basic(t *testing.T) {
	rInt := sdkacctest.RandInt()
	resourceName := "aws_codecommit_repository.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, codecommit.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCodeCommitRepository_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
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

func TestAccCodeCommitRepository_withChanges(t *testing.T) {
	rInt := sdkacctest.RandInt()
	resourceName := "aws_codecommit_repository.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, codecommit.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCodeCommitRepository_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
					resource.TestCheckResourceAttr(
						"aws_codecommit_repository.test", "description", "This is a test description"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCodeCommitRepository_withChanges(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
					resource.TestCheckResourceAttr(
						"aws_codecommit_repository.test", "description", "This is a test description - with changes"),
				),
			},
		},
	})
}

func TestAccCodeCommitRepository_CreateDefault_branch(t *testing.T) {
	rInt := sdkacctest.RandInt()
	resourceName := "aws_codecommit_repository.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, codecommit.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCodeCommitRepository_with_default_branch(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
					resource.TestCheckResourceAttr(
						"aws_codecommit_repository.test", "default_branch", "master"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"default_branch"},
			},
		},
	})
}

func TestAccCodeCommitRepository_CreateAndUpdateDefault_branch(t *testing.T) {
	rInt := sdkacctest.RandInt()
	resourceName := "aws_codecommit_repository.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, codecommit.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCodeCommitRepository_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
					resource.TestCheckNoResourceAttr(
						"aws_codecommit_repository.test", "default_branch"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCodeCommitRepository_with_default_branch(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists("aws_codecommit_repository.test"),
					resource.TestCheckResourceAttr(
						"aws_codecommit_repository.test", "default_branch", "master"),
				),
			},
		},
	})
}

func TestAccCodeCommitRepository_tags(t *testing.T) {
	rName := sdkacctest.RandString(10)
	resourceName := "aws_codecommit_repository.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, codecommit.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryTags1Config(rName, "key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists(resourceName),
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
				Config: testAccRepositoryTags2Config(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccRepositoryTags1Config(rName, "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCodeCommitRepositoryExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func testAccCheckCodeCommitRepositoryExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).CodeCommitConn
		out, err := conn.GetRepository(&codecommit.GetRepositoryInput{
			RepositoryName: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if out.RepositoryMetadata.Arn == nil {
			return fmt.Errorf("No CodeCommit Repository Vault Found")
		}

		if *out.RepositoryMetadata.RepositoryName != rs.Primary.ID {
			return fmt.Errorf("CodeCommit Repository Mismatch - existing: %q, state: %q",
				*out.RepositoryMetadata.RepositoryName, rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckCodeCommitRepositoryDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).CodeCommitConn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_codecommit_repository" {
			continue
		}

		_, err := conn.GetRepository(&codecommit.GetRepositoryInput{
			RepositoryName: aws.String(rs.Primary.ID),
		})

		if ae, ok := err.(awserr.Error); ok && ae.Code() == "RepositoryDoesNotExistException" {
			continue
		}
		if err == nil {
			return fmt.Errorf("Repository still exists: %s", rs.Primary.ID)
		}
		return err
	}

	return nil
}

func testAccCodeCommitRepository_basic(rInt int) string {
	return fmt.Sprintf(`
resource "aws_codecommit_repository" "test" {
  repository_name = "test_repository_%d"
  description     = "This is a test description"
}
`, rInt)
}

func testAccCodeCommitRepository_withChanges(rInt int) string {
	return fmt.Sprintf(`
resource "aws_codecommit_repository" "test" {
  repository_name = "test_repository_%d"
  description     = "This is a test description - with changes"
}
`, rInt)
}

func testAccCodeCommitRepository_with_default_branch(rInt int) string {
	return fmt.Sprintf(`
resource "aws_codecommit_repository" "test" {
  repository_name = "test_repository_%d"
  description     = "This is a test description"
  default_branch  = "master"
}
`, rInt)
}

func testAccRepositoryTags1Config(r, tag1Key, tag1Value string) string {
	return fmt.Sprintf(`
resource "aws_codecommit_repository" "test" {
  repository_name = "terraform-test-%s"

  tags = {
    %q = %q
  }
}
`, r, tag1Key, tag1Value)
}

func testAccRepositoryTags2Config(r, tag1Key, tag1Value, tag2Key, tag2Value string) string {
	return fmt.Sprintf(`
resource "aws_codecommit_repository" "test" {
  repository_name = "terraform-test-%s"

  tags = {
    %q = %q
    %q = %q
  }
}
`, r, tag1Key, tag1Value, tag2Key, tag2Value)
}

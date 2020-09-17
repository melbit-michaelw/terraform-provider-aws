package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAWSSSMPatchGroup_basic(t *testing.T) {
	name := acctest.RandString(10)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMPatchGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMPatchGroupBasicConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMPatchGroupExists("aws_ssm_patch_group.patchgroup"),
				),
			},
		},
	})
}

func testAccCheckAWSSSMPatchGroupExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SSM Patch Baseline ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ssmconn

		params := &ssm.DescribePatchGroupsInput{}

		found := false
		err := conn.DescribePatchGroupsPages(params, func(page *ssm.DescribePatchGroupsOutput, lastPage bool) bool {
			for _, t := range page.Mappings {
				if testAccAWSSSMPatchGroupStateCompare(t, rs) {
					found = true
					break
				}
			}
			return true
		})
		if err != nil {
			return err
		}

		if found {
			return nil
		}

		return fmt.Errorf("No AWS SSM Patch Group found")
	}
}

func testAccCheckAWSSSMPatchGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ssmconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ssm_patch_group" {
			continue
		}

		found := false

		params := &ssm.DescribePatchGroupsInput{}

		err := conn.DescribePatchGroupsPages(params, func(page *ssm.DescribePatchGroupsOutput, lastPage bool) bool {
			for _, t := range page.Mappings {
				if testAccAWSSSMPatchGroupStateCompare(t, rs) {
					found = true
					break
				}
			}
			return true
		})
		if err != nil {
			// Verify the error is what we want
			if ae, ok := err.(awserr.Error); ok && ae.Code() == "DoesNotExistException" {
				continue
			}
			return err
		}

		if found {
			return fmt.Errorf("Expected AWS SSM Patch Group to be gone, but was still found")
		}

		return nil
	}

	return nil
}

func testAccAWSSSMPatchGroupBasicConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_patch_baseline" "foo" {
  name             = "patch-baseline-%s"
  approved_patches = ["KB123456"]
}

resource "aws_ssm_patch_group" "patchgroup" {
  baseline_id = aws_ssm_patch_baseline.foo.id
  patch_group = "patch-group"
}
`, rName)
}

func testAccAWSSSMPatchGroupStateCompare(mapping *ssm.PatchGroupPatchBaselineMapping, rs *terraform.ResourceState) bool {
	id := fmt.Sprintf("%s:%s", *mapping.PatchGroup, *mapping.BaselineIdentity.BaselineId)

	if *mapping.BaselineIdentity.BaselineId == rs.Primary.Attributes["baseline_id"] && id == rs.Primary.ID {

		return true
	}

	return false
}

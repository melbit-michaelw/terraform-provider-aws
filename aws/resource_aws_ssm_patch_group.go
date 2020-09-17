package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAwsSsmPatchGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmPatchGroupCreate,
		Read:   resourceAwsSsmPatchGroupRead,
		Delete: resourceAwsSsmPatchGroupDelete,

		Schema: map[string]*schema.Schema{
			"baseline_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"patch_group": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsSsmPatchGroupCreate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.RegisterPatchBaselineForPatchGroupInput{
		BaselineId: aws.String(d.Get("baseline_id").(string)),
		PatchGroup: aws.String(d.Get("patch_group").(string)),
	}

	resp, err := ssmconn.RegisterPatchBaselineForPatchGroup(params)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%s:%s", *resp.PatchGroup, *resp.BaselineId))
	return resourceAwsSsmPatchGroupRead(d, meta)
}

func resourceAwsSsmPatchGroupRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.DescribePatchGroupsInput{}

	found := false
	err := ssmconn.DescribePatchGroupsPages(params, func(page *ssm.DescribePatchGroupsOutput, lastPage bool) bool {
		for _, t := range page.Mappings {
			if fmt.Sprintf("%s:%s", *t.PatchGroup, *t.BaselineIdentity.BaselineId) == d.Id() {
				found = true

				d.Set("patch_group", t.PatchGroup)
				d.Set("baseline_id", t.BaselineIdentity.BaselineId)
				break
			}
		}
		return true
	})

	if err != nil {
		return err
	}

	if !found {
		log.Printf("[INFO] Patch Group not found. Removing from state")
		d.SetId("")
		return nil
	}

	return nil

}

func resourceAwsSsmPatchGroupDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Deleting SSM Patch Group: %s", d.Id())

	params := &ssm.DeregisterPatchBaselineForPatchGroupInput{
		BaselineId: aws.String(d.Get("baseline_id").(string)),
		PatchGroup: aws.String(d.Get("patch_group").(string)),
	}

	_, err := ssmconn.DeregisterPatchBaselineForPatchGroup(params)
	if err != nil {
		return fmt.Errorf("error deregistering SSM Patch Group (%s): %s", d.Id(), err)
	}

	return nil
}

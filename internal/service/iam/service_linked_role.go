package iam

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
)

func ResourceServiceLinkedRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceServiceLinkedRoleCreate,
		Read:   resourceServiceLinkedRoleRead,
		Update: resourceServiceLinkedRoleUpdate,
		Delete: resourceServiceLinkedRoleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"aws_service_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`\.`), "must be a full service hostname e.g. elasticbeanstalk.amazonaws.com"),
			},

			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"path": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"create_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"unique_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"custom_suffix": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if strings.Contains(d.Get("aws_service_name").(string), ".application-autoscaling.") && new == "" {
						return true
					}
					return false
				},
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceServiceLinkedRoleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn

	serviceName := d.Get("aws_service_name").(string)

	params := &iam.CreateServiceLinkedRoleInput{
		AWSServiceName: aws.String(serviceName),
	}

	if v, ok := d.GetOk("custom_suffix"); ok {
		params.CustomSuffix = aws.String(v.(string))
	}

	if v, ok := d.GetOk("description"); ok {
		params.Description = aws.String(v.(string))
	}

	resp, err := conn.CreateServiceLinkedRole(params)

	if err != nil {
		return fmt.Errorf("Error creating service-linked role with name %s: %s", serviceName, err)
	}
	d.SetId(aws.StringValue(resp.Role.Arn))

	return resourceServiceLinkedRoleRead(d, meta)
}

func resourceServiceLinkedRoleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn

	serviceName, roleName, customSuffix, err := DecodeServiceLinkedRoleID(d.Id())
	if err != nil {
		return err
	}

	params := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}

	resp, err := conn.GetRole(params)

	if err != nil {
		if tfawserr.ErrMessageContains(err, iam.ErrCodeNoSuchEntityException, "") {
			log.Printf("[WARN] IAM service linked role %s not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	role := resp.Role

	d.Set("arn", role.Arn)
	d.Set("aws_service_name", serviceName)
	d.Set("create_date", aws.TimeValue(role.CreateDate).Format(time.RFC3339))
	d.Set("custom_suffix", customSuffix)
	d.Set("description", role.Description)
	d.Set("name", role.RoleName)
	d.Set("path", role.Path)
	d.Set("unique_id", role.RoleId)

	return nil
}

func resourceServiceLinkedRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn

	_, roleName, _, err := DecodeServiceLinkedRoleID(d.Id())
	if err != nil {
		return err
	}

	params := &iam.UpdateRoleInput{
		Description: aws.String(d.Get("description").(string)),
		RoleName:    aws.String(roleName),
	}

	_, err = conn.UpdateRole(params)

	if err != nil {
		return fmt.Errorf("Error updating service-linked role %s: %s", d.Id(), err)
	}

	return resourceServiceLinkedRoleRead(d, meta)
}

func resourceServiceLinkedRoleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn

	_, roleName, _, err := DecodeServiceLinkedRoleID(d.Id())
	if err != nil {
		return err
	}

	deletionID, err := DeleteServiceLinkedRole(conn, roleName)
	if err != nil {
		return fmt.Errorf("Error deleting service-linked role %s: %s", d.Id(), err)
	}
	if deletionID == "" {
		return nil
	}

	err = DeleteServiceLinkedRoleWaiter(conn, deletionID)
	if err != nil {
		return fmt.Errorf("Error waiting for role (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}

func DecodeServiceLinkedRoleID(id string) (serviceName, roleName, customSuffix string, err error) {
	idArn, err := arn.Parse(id)
	if err != nil {
		return "", "", "", err
	}

	resourceParts := strings.Split(idArn.Resource, "/")
	if len(resourceParts) != 4 {
		return "", "", "", fmt.Errorf("expected IAM Service Role ARN (arn:PARTITION:iam::ACCOUNTID:role/aws-service-role/SERVICENAME/ROLENAME), received: %s", id)
	}

	serviceName = resourceParts[2]
	roleName = resourceParts[3]

	roleNameParts := strings.Split(roleName, "_")
	if len(roleNameParts) == 2 {
		customSuffix = roleNameParts[1]
	}

	return
}

func DeleteServiceLinkedRole(conn *iam.IAM, roleName string) (string, error) {
	params := &iam.DeleteServiceLinkedRoleInput{
		RoleName: aws.String(roleName),
	}

	resp, err := conn.DeleteServiceLinkedRole(params)

	if err != nil {
		if tfawserr.ErrMessageContains(err, iam.ErrCodeNoSuchEntityException, "") {
			return "", nil
		}
		return "", err
	}

	return aws.StringValue(resp.DeletionTaskId), nil
}

func DeleteServiceLinkedRoleWaiter(conn *iam.IAM, deletionTaskID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{iam.DeletionTaskStatusTypeInProgress, iam.DeletionTaskStatusTypeNotStarted},
		Target:  []string{iam.DeletionTaskStatusTypeSucceeded},
		Refresh: deleteIamServiceLinkedRoleRefreshFunc(conn, deletionTaskID),
		Timeout: 5 * time.Minute,
		Delay:   10 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		if tfawserr.ErrMessageContains(err, iam.ErrCodeNoSuchEntityException, "") {
			return nil
		}
		return err
	}

	return nil
}

func deleteIamServiceLinkedRoleRefreshFunc(conn *iam.IAM, deletionTaskId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		params := &iam.GetServiceLinkedRoleDeletionStatusInput{
			DeletionTaskId: aws.String(deletionTaskId),
		}

		resp, err := conn.GetServiceLinkedRoleDeletionStatus(params)
		if err != nil {
			return nil, "", err
		}

		return resp, aws.StringValue(resp.Status), nil
	}
}

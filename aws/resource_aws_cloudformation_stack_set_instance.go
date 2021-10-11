package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	tfcloudformation "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/cloudformation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/cloudformation/finder"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/cloudformation/waiter"
	iamwaiter "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/iam/waiter"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

func resourceAwsCloudFormationStackSetInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudFormationStackSetInstanceCreate,
		Read:   resourceAwsCloudFormationStackSetInstanceRead,
		Update: resourceAwsCloudFormationStackSetInstanceUpdate,
		Delete: resourceAwsCloudFormationStackSetInstanceDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(waiter.StackSetInstanceCreatedDefaultTimeout),
			Update: schema.DefaultTimeout(waiter.StackSetInstanceUpdatedDefaultTimeout),
			Delete: schema.DefaultTimeout(waiter.StackSetInstanceDeletedDefaultTimeout),
		},

		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ValidateFunc:  validateAwsAccountId,
				ConflictsWith: []string{"deployment_targets"},
			},
			"deployment_targets": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"organizational_unit_ids": {
							Type:     schema.TypeSet,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringMatch(regexp.MustCompile(`^(ou-[a-z0-9]{4,32}-[a-z0-9]{8,32}|r-[a-z0-9]{4,32})$`), ""),
							},
						},
					},
				},
				ConflictsWith: []string{"account_id"},
			},
			"organizational_unit_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"parameter_overrides": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"retain_stack": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"stack_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"stack_set_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
		},
	}
}

func resourceAwsCloudFormationStackSetInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cfconn

	region := meta.(*AWSClient).region
	if v, ok := d.GetOk("region"); ok {
		region = v.(string)
	}

	stackSetName := d.Get("stack_set_name").(string)
	input := &cloudformation.CreateStackInstancesInput{
		Regions:      aws.StringSlice([]string{region}),
		StackSetName: aws.String(stackSetName),
	}

	accountID := meta.(*AWSClient).accountid
	if v, ok := d.GetOk("account_id"); ok {
		accountID = v.(string)
	}

	if v, ok := d.GetOk("deployment_targets"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		dt := expandCloudFormationDeploymentTargets(v.([]interface{}))
		// temporarily set the accountId to the DeploymentTarget IDs
		// to later inform the Read CRUD operation if the true accountID needs to be determined
		accountID = strings.Join(aws.StringValueSlice(dt.OrganizationalUnitIds), "/")
		input.DeploymentTargets = dt
	} else {
		input.Accounts = aws.StringSlice([]string{accountID})
	}

	if v, ok := d.GetOk("parameter_overrides"); ok {
		input.ParameterOverrides = expandCloudFormationParameters(v.(map[string]interface{}))
	}

	log.Printf("[DEBUG] Creating CloudFormation StackSet Instance: %s", input)
	_, err := tfresource.RetryWhen(
		iamwaiter.PropagationTimeout,
		func() (interface{}, error) {
			input.OperationId = aws.String(resource.UniqueId())

			output, err := conn.CreateStackInstances(input)

			if err != nil {
				return nil, fmt.Errorf("error creating CloudFormation StackSet (%s) Instance: %w", stackSetName, err)
			}

			d.SetId(tfcloudformation.StackSetInstanceCreateResourceID(stackSetName, accountID, region))

			return waiter.StackSetOperationSucceeded(conn, stackSetName, aws.StringValue(output.OperationId), d.Timeout(schema.TimeoutCreate))
		},
		func(err error) (bool, error) {
			if err == nil {
				return false, nil
			}

			message := err.Error()

			// IAM eventual consistency
			if strings.Contains(message, "AccountGate check failed") {
				return true, err
			}

			// IAM eventual consistency
			// User: XXX is not authorized to perform: cloudformation:CreateStack on resource: YYY
			if strings.Contains(message, "is not authorized") {
				return true, err
			}

			// IAM eventual consistency
			// XXX role has insufficient YYY permissions
			if strings.Contains(message, "role has insufficient") {
				return true, err
			}

			// IAM eventual consistency
			// Account XXX should have YYY role with trust relationship to Role ZZZ
			if strings.Contains(message, "role with trust relationship") {
				return true, err
			}

			// IAM eventual consistency
			if strings.Contains(message, "The security token included in the request is invalid") {
				return true, err
			}

			return false, fmt.Errorf("error waiting for CloudFormation StackSet Instance (%s) creation: %w", d.Id(), err)
		},
	)

	if err != nil {
		return err
	}

	return resourceAwsCloudFormationStackSetInstanceRead(d, meta)
}

func resourceAwsCloudFormationStackSetInstanceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cfconn

	stackSetName, accountID, region, err := tfcloudformation.StackSetInstanceParseResourceID(d.Id())

	if err != nil {
		return err
	}

	// Determine correct account ID for the Instance if created with deployment targets;
	// we only expect the accountID to be the organization root ID or organizational unit (OU) IDs
	// separated by a slash after creation.
	if regexp.MustCompile(`(ou-[a-z0-9]{4,32}-[a-z0-9]{8,32}|r-[a-z0-9]{4,32})`).MatchString(accountID) {
		orgIDs := strings.Split(accountID, "/")
		accountID, err = finder.StackInstanceAccountIdByOrgIds(conn, stackSetName, region, orgIDs)

		if err != nil {
			return fmt.Errorf("error finding CloudFormation StackSet Instance (%s) Account: %w", d.Id(), err)
		}

		d.SetId(tfcloudformation.StackSetInstanceCreateResourceID(stackSetName, accountID, region))
	}

	stackInstance, err := finder.StackInstanceByName(conn, stackSetName, accountID, region)

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] CloudFormation StackSet Instance (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading CloudFormation StackSet Instance (%s): %w", d.Id(), err)
	}

	d.Set("account_id", stackInstance.Account)
	d.Set("organizational_unit_id", stackInstance.OrganizationalUnitId)
	if err := d.Set("parameter_overrides", flattenAllCloudFormationParameters(stackInstance.ParameterOverrides)); err != nil {
		return fmt.Errorf("error setting parameters: %w", err)
	}

	d.Set("region", stackInstance.Region)
	d.Set("stack_id", stackInstance.StackId)
	d.Set("stack_set_name", stackSetName)

	return nil
}

func resourceAwsCloudFormationStackSetInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cfconn

	if d.HasChanges("deployment_targets", "parameter_overrides") {
		stackSetName, accountID, region, err := tfcloudformation.StackSetInstanceParseResourceID(d.Id())

		if err != nil {
			return err
		}

		input := &cloudformation.UpdateStackInstancesInput{
			Accounts:           aws.StringSlice([]string{accountID}),
			OperationId:        aws.String(resource.UniqueId()),
			ParameterOverrides: []*cloudformation.Parameter{},
			Regions:            aws.StringSlice([]string{region}),
			StackSetName:       aws.String(stackSetName),
		}

		if v, ok := d.GetOk("deployment_targets"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
			// reset input Accounts as the API accepts only 1 of Accounts and DeploymentTargets
			input.Accounts = nil
			input.DeploymentTargets = expandCloudFormationDeploymentTargets(v.([]interface{}))
		}

		if v, ok := d.GetOk("parameter_overrides"); ok {
			input.ParameterOverrides = expandCloudFormationParameters(v.(map[string]interface{}))
		}

		log.Printf("[DEBUG] Updating CloudFormation StackSet Instance: %s", input)
		output, err := conn.UpdateStackInstances(input)

		if err != nil {
			return fmt.Errorf("error updating CloudFormation StackSet Instance (%s): %w", d.Id(), err)
		}

		if _, err := waiter.StackSetOperationSucceeded(conn, stackSetName, aws.StringValue(output.OperationId), d.Timeout(schema.TimeoutUpdate)); err != nil {
			return fmt.Errorf("error waiting for CloudFormation StackSet Instance (%s) update: %s", d.Id(), err)
		}
	}

	return resourceAwsCloudFormationStackSetInstanceRead(d, meta)
}

func resourceAwsCloudFormationStackSetInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cfconn

	stackSetName, accountID, region, err := tfcloudformation.StackSetInstanceParseResourceID(d.Id())

	if err != nil {
		return err
	}

	input := &cloudformation.DeleteStackInstancesInput{
		Accounts:     aws.StringSlice([]string{accountID}),
		OperationId:  aws.String(resource.UniqueId()),
		Regions:      aws.StringSlice([]string{region}),
		RetainStacks: aws.Bool(d.Get("retain_stack").(bool)),
		StackSetName: aws.String(stackSetName),
	}

	if v, ok := d.GetOk("organizational_unit_id"); ok {
		// For instances associated with stack sets that use a self-managed permission model,
		// the organizational unit must be provided;
		input.Accounts = nil
		input.DeploymentTargets = &cloudformation.DeploymentTargets{
			OrganizationalUnitIds: aws.StringSlice([]string{v.(string)}),
		}
	}

	log.Printf("[DEBUG] Deleting CloudFormation StackSet Instance: %s", d.Id())
	output, err := conn.DeleteStackInstances(input)

	if tfawserr.ErrCodeEquals(err, cloudformation.ErrCodeStackInstanceNotFoundException) || tfawserr.ErrCodeEquals(err, cloudformation.ErrCodeStackSetNotFoundException) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting CloudFormation StackSet Instance (%s): %s", d.Id(), err)
	}

	if _, err := waiter.StackSetOperationSucceeded(conn, stackSetName, aws.StringValue(output.OperationId), d.Timeout(schema.TimeoutDelete)); err != nil {
		return fmt.Errorf("error waiting for CloudFormation StackSet Instance (%s) deletion: %s", d.Id(), err)
	}

	return nil
}

func expandCloudFormationDeploymentTargets(l []interface{}) *cloudformation.DeploymentTargets {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})

	if !ok {
		return nil
	}

	dt := &cloudformation.DeploymentTargets{}

	if v, ok := tfMap["organizational_unit_ids"].(*schema.Set); ok && v.Len() > 0 {
		dt.OrganizationalUnitIds = expandStringSet(v)
	}

	return dt
}

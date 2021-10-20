package iam

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceRoleCreate,
		Read:   resourceRoleRead,
		Update: resourceRoleUpdate,
		Delete: resourceRoleDelete,
		Importer: &schema.ResourceImporter{
			State: resourceRoleImport,
		},
		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"unique_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 64),
					validation.StringMatch(regexp.MustCompile(`^[\w+=,.@-]*$`), "must match [\\w+=,.@-]"),
				),
			},

			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 64-resource.UniqueIDSuffixLength),
					validation.StringMatch(regexp.MustCompile(`^[\w+=,.@-]*$`), "must match [\\w+=,.@-]"),
				),
			},

			"path": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "/",
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(0, 512),
			},

			"permissions_boundary": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(0, 1000),
					validation.StringDoesNotMatch(regexp.MustCompile("[“‘]"), "cannot contain specially formatted single or double quotes: [“‘]"),
					validation.StringMatch(regexp.MustCompile(`[\p{L}\p{M}\p{Z}\p{S}\p{N}\p{P}]*`), `must satisfy regular expression pattern: [\p{L}\p{M}\p{Z}\p{S}\p{N}\p{P}]*)`),
				),
			},

			"assume_role_policy": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: verify.SuppressEquivalentPolicyDiffs,
				ValidateFunc:     validation.StringIsJSON,
			},

			"force_detach_policies": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"create_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"max_session_duration": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      3600,
				ValidateFunc: validation.IntBetween(3600, 43200),
			},

			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),

			"inline_policy": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.All(
								validation.StringIsNotEmpty,
								validRolePolicyName,
							),
						},
						"policy": {
							Type:             schema.TypeString,
							Optional:         true,
							ValidateFunc:     verify.ValidIAMPolicyJSON,
							DiffSuppressFunc: verify.SuppressEquivalentPolicyDiffs,
						},
					},
				},
			},

			"managed_policy_arns": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: verify.ValidARN,
				},
			},
		},

		CustomizeDiff: verify.SetTagsDiff,
	}
}

func resourceRoleImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.Set("force_detach_policies", false)
	return []*schema.ResourceData{d}, nil
}

func resourceRoleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(d.Get("tags").(map[string]interface{})))

	name := create.Name(d.Get("name").(string), d.Get("name_prefix").(string))
	request := &iam.CreateRoleInput{
		Path:                     aws.String(d.Get("path").(string)),
		RoleName:                 aws.String(name),
		AssumeRolePolicyDocument: aws.String(d.Get("assume_role_policy").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		request.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("max_session_duration"); ok {
		request.MaxSessionDuration = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("permissions_boundary"); ok {
		request.PermissionsBoundary = aws.String(v.(string))
	}

	if len(tags) > 0 {
		request.Tags = Tags(tags.IgnoreAWS())
	}

	outputRaw, err := tfresource.RetryWhen(
		PropagationTimeout,
		func() (interface{}, error) {
			return conn.CreateRole(request)
		},
		func(err error) (bool, error) {
			if tfawserr.ErrMessageContains(err, iam.ErrCodeMalformedPolicyDocumentException, "Invalid principal in policy") {
				return true, err
			}

			return false, err
		},
	)

	if err != nil {
		return fmt.Errorf("error creating IAM Role (%s): %w", name, err)
	}

	roleName := aws.StringValue(outputRaw.(*iam.CreateRoleOutput).Role.RoleName)

	if v, ok := d.GetOk("inline_policy"); ok && v.(*schema.Set).Len() > 0 {
		policies := expandIamInlinePolicies(roleName, v.(*schema.Set).List())
		if err := addIamInlinePolicies(policies, meta); err != nil {
			return err
		}
	}

	if v, ok := d.GetOk("managed_policy_arns"); ok && v.(*schema.Set).Len() > 0 {
		managedPolicies := flex.ExpandStringSet(v.(*schema.Set))
		if err := addIamManagedPolicies(roleName, managedPolicies, meta); err != nil {
			return err
		}
	}

	d.SetId(roleName)
	return resourceRoleRead(d, meta)
}

func resourceRoleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	outputRaw, err := tfresource.RetryWhenNewResourceNotFound(PropagationTimeout, func() (interface{}, error) {
		return FindRoleByName(conn, d.Id())
	}, d.IsNewResource())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] IAM Role (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading IAM Role (%s): %w", d.Id(), err)
	}

	role := outputRaw.(*iam.Role)

	d.Set("arn", role.Arn)
	if err := d.Set("create_date", role.CreateDate.Format(time.RFC3339)); err != nil {
		return err
	}
	d.Set("description", role.Description)
	d.Set("max_session_duration", role.MaxSessionDuration)
	d.Set("name", role.RoleName)
	d.Set("name_prefix", create.NamePrefixFromName(aws.StringValue(role.RoleName)))
	d.Set("path", role.Path)
	if role.PermissionsBoundary != nil {
		d.Set("permissions_boundary", role.PermissionsBoundary.PermissionsBoundaryArn)
	}
	d.Set("unique_id", role.RoleId)

	tags := KeyValueTags(role.Tags).IgnoreAWS().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	assumeRolePolicy, err := url.QueryUnescape(*role.AssumeRolePolicyDocument)
	if err != nil {
		return err
	}
	if err := d.Set("assume_role_policy", assumeRolePolicy); err != nil {
		return err
	}

	inlinePolicies, err := readIamInlinePolicies(aws.StringValue(role.RoleName), meta)
	if err != nil {
		return fmt.Errorf("reading inline policies for IAM role %s, error: %s", d.Id(), err)
	}
	if err := d.Set("inline_policy", flattenIamInlinePolicies(inlinePolicies)); err != nil {
		return fmt.Errorf("error setting inline_policy: %w", err)
	}

	managedPolicies, err := readIamRolePolicyAttachments(conn, aws.StringValue(role.RoleName))
	if err != nil {
		return fmt.Errorf("reading managed policies for IAM role %s, error: %s", d.Id(), err)
	}
	d.Set("managed_policy_arns", managedPolicies)

	return nil
}

func resourceRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn

	if d.HasChange("assume_role_policy") {
		assumeRolePolicyInput := &iam.UpdateAssumeRolePolicyInput{
			RoleName:       aws.String(d.Id()),
			PolicyDocument: aws.String(d.Get("assume_role_policy").(string)),
		}

		_, err := tfresource.RetryWhen(
			PropagationTimeout,
			func() (interface{}, error) {
				return conn.UpdateAssumeRolePolicy(assumeRolePolicyInput)
			},
			func(err error) (bool, error) {
				if tfawserr.ErrMessageContains(err, iam.ErrCodeMalformedPolicyDocumentException, "Invalid principal in policy") {
					return true, err
				}

				return false, err
			},
		)

		if err != nil {
			return fmt.Errorf("error updating IAM Role (%s) assume role policy: %w", d.Id(), err)
		}
	}

	if d.HasChange("description") {
		roleDescriptionInput := &iam.UpdateRoleDescriptionInput{
			RoleName:    aws.String(d.Id()),
			Description: aws.String(d.Get("description").(string)),
		}

		_, err := conn.UpdateRoleDescription(roleDescriptionInput)

		if err != nil {
			return fmt.Errorf("error updating IAM Role (%s) description: %w", d.Id(), err)
		}
	}

	if d.HasChange("max_session_duration") {
		roleMaxDurationInput := &iam.UpdateRoleInput{
			RoleName:           aws.String(d.Id()),
			MaxSessionDuration: aws.Int64(int64(d.Get("max_session_duration").(int))),
		}

		_, err := conn.UpdateRole(roleMaxDurationInput)

		if err != nil {
			return fmt.Errorf("error updating IAM Role (%s) MaxSessionDuration: %s", d.Id(), err)
		}
	}

	if d.HasChange("permissions_boundary") {
		permissionsBoundary := d.Get("permissions_boundary").(string)
		if permissionsBoundary != "" {
			input := &iam.PutRolePermissionsBoundaryInput{
				PermissionsBoundary: aws.String(permissionsBoundary),
				RoleName:            aws.String(d.Id()),
			}

			_, err := conn.PutRolePermissionsBoundary(input)

			if err != nil {
				return fmt.Errorf("error updating IAM Role (%s) permissions boundary: %w", d.Id(), err)
			}
		} else {
			input := &iam.DeleteRolePermissionsBoundaryInput{
				RoleName: aws.String(d.Id()),
			}

			_, err := conn.DeleteRolePermissionsBoundary(input)

			if err != nil {
				return fmt.Errorf("error deleting IAM Role (%s) permissions boundary: %w", d.Id(), err)
			}
		}
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := roleUpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating IAM Role (%s) tags: %s", d.Id(), err)
		}
	}

	if d.HasChange("inline_policy") {
		roleName := d.Get("name").(string)
		o, n := d.GetChange("inline_policy")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		remove := os.Difference(ns).List()
		add := ns.Difference(os).List()

		var policyNames []*string
		for _, policy := range remove {
			tfMap, ok := policy.(map[string]interface{})

			if !ok {
				continue
			}

			if v, ok := tfMap["name"].(string); ok && v != "" {
				policyNames = append(policyNames, aws.String(tfMap["name"].(string)))
			}
		}
		if err := deleteIamRolePolicies(conn, roleName, policyNames); err != nil {
			return fmt.Errorf("unable to delete inline policies: %w", err)
		}

		policies := expandIamInlinePolicies(roleName, add)
		if err := addIamInlinePolicies(policies, meta); err != nil {
			return err
		}
	}

	if d.HasChange("managed_policy_arns") {
		roleName := d.Get("name").(string)

		o, n := d.GetChange("managed_policy_arns")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		remove := flex.ExpandStringSet(os.Difference(ns))
		add := flex.ExpandStringSet(ns.Difference(os))

		if err := deleteIamRolePolicyAttachments(conn, roleName, remove); err != nil {
			return fmt.Errorf("unable to detach policies: %w", err)
		}

		if err := addIamManagedPolicies(roleName, add, meta); err != nil {
			return err
		}
	}

	return resourceRoleRead(d, meta)
}

func resourceRoleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn

	hasInline := false
	if v, ok := d.GetOk("inline_policy"); ok && v.(*schema.Set).Len() > 0 {
		hasInline = true
	}

	hasManaged := false
	if v, ok := d.GetOk("managed_policy_arns"); ok && v.(*schema.Set).Len() > 0 {
		hasManaged = true
	}

	err := DeleteRole(conn, d.Id(), d.Get("force_detach_policies").(bool), hasInline, hasManaged)

	if tfawserr.ErrCodeEquals(err, iam.ErrCodeNoSuchEntityException) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting IAM Role (%s): %w", d.Id(), err)
	}

	return nil
}

func DeleteRole(conn *iam.IAM, roleName string, forceDetach, hasInline, hasManaged bool) error {
	if err := deleteIamRoleInstanceProfiles(conn, roleName); err != nil {
		return fmt.Errorf("unable to detach instance profiles: %w", err)
	}

	if forceDetach || hasManaged {
		managedPolicies, err := readIamRolePolicyAttachments(conn, roleName)
		if err != nil {
			return err
		}

		if err := deleteIamRolePolicyAttachments(conn, roleName, managedPolicies); err != nil {
			return fmt.Errorf("unable to detach policies: %w", err)
		}
	}

	if forceDetach || hasInline {
		inlinePolicies, err := readIamRolePolicyNames(conn, roleName)
		if err != nil {
			return err
		}

		if err := deleteIamRolePolicies(conn, roleName, inlinePolicies); err != nil {
			return fmt.Errorf("unable to delete inline policies: %w", err)
		}
	}

	deleteRoleInput := &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	}
	err := resource.Retry(PropagationTimeout, func() *resource.RetryError {
		_, err := conn.DeleteRole(deleteRoleInput)
		if err != nil {
			if tfawserr.ErrCodeEquals(err, iam.ErrCodeDeleteConflictException) {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}
		return nil
	})
	if tfresource.TimedOut(err) {
		_, err = conn.DeleteRole(deleteRoleInput)
	}

	return err
}

func deleteIamRoleInstanceProfiles(conn *iam.IAM, roleName string) error {
	resp, err := conn.ListInstanceProfilesForRole(&iam.ListInstanceProfilesForRoleInput{
		RoleName: aws.String(roleName),
	})
	if tfawserr.ErrCodeEquals(err, iam.ErrCodeNoSuchEntityException) {
		return nil
	}
	if err != nil {
		return err
	}

	// Loop and remove this Role from any Profiles
	for _, i := range resp.InstanceProfiles {
		input := &iam.RemoveRoleFromInstanceProfileInput{
			InstanceProfileName: i.InstanceProfileName,
			RoleName:            aws.String(roleName),
		}

		_, err := conn.RemoveRoleFromInstanceProfile(input)
		if tfawserr.ErrCodeEquals(err, iam.ErrCodeNoSuchEntityException) {
			continue
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func readIamRolePolicyAttachments(conn *iam.IAM, roleName string) ([]*string, error) {
	managedPolicies := make([]*string, 0)
	input := &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	}

	err := conn.ListAttachedRolePoliciesPages(input, func(page *iam.ListAttachedRolePoliciesOutput, lastPage bool) bool {
		for _, v := range page.AttachedPolicies {
			managedPolicies = append(managedPolicies, v.PolicyArn)
		}
		return !lastPage
	})
	if err != nil && !tfawserr.ErrCodeEquals(err, iam.ErrCodeNoSuchEntityException) {
		return nil, err
	}

	return managedPolicies, nil
}

func deleteIamRolePolicyAttachments(conn *iam.IAM, roleName string, managedPolicies []*string) error {
	for _, parn := range managedPolicies {
		input := &iam.DetachRolePolicyInput{
			PolicyArn: parn,
			RoleName:  aws.String(roleName),
		}

		_, err := conn.DetachRolePolicy(input)
		if tfawserr.ErrCodeEquals(err, iam.ErrCodeNoSuchEntityException) {
			continue
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func readIamRolePolicyNames(conn *iam.IAM, roleName string) ([]*string, error) {
	inlinePolicies := make([]*string, 0)
	input := &iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	}

	err := conn.ListRolePoliciesPages(input, func(page *iam.ListRolePoliciesOutput, lastPage bool) bool {
		inlinePolicies = append(inlinePolicies, page.PolicyNames...)
		return !lastPage
	})

	if err != nil && !tfawserr.ErrCodeEquals(err, iam.ErrCodeNoSuchEntityException) {
		return nil, err
	}

	return inlinePolicies, nil
}

func deleteIamRolePolicies(conn *iam.IAM, roleName string, policyNames []*string) error {
	for _, name := range policyNames {
		if len(aws.StringValue(name)) == 0 {
			continue
		}

		input := &iam.DeleteRolePolicyInput{
			PolicyName: name,
			RoleName:   aws.String(roleName),
		}

		_, err := conn.DeleteRolePolicy(input)
		if tfawserr.ErrCodeEquals(err, iam.ErrCodeNoSuchEntityException) {
			return nil
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func flattenIamInlinePolicy(apiObject *iam.PutRolePolicyInput) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	tfMap["name"] = aws.StringValue(apiObject.PolicyName)
	tfMap["policy"] = aws.StringValue(apiObject.PolicyDocument)

	return tfMap
}

func flattenIamInlinePolicies(apiObjects []*iam.PutRolePolicyInput) []interface{} {
	if len(apiObjects) == 0 {
		return nil
	}

	var tfList []interface{}

	for _, apiObject := range apiObjects {
		if apiObject == nil {
			continue
		}

		tfList = append(tfList, flattenIamInlinePolicy(apiObject))
	}

	return tfList
}

func expandIamInlinePolicy(roleName string, tfMap map[string]interface{}) *iam.PutRolePolicyInput {
	if tfMap == nil {
		return nil
	}

	apiObject := &iam.PutRolePolicyInput{
		RoleName: aws.String(roleName),
	}

	if v, ok := tfMap["name"].(string); ok && v != "" {
		apiObject.PolicyName = aws.String(v)
	}

	if v, ok := tfMap["policy"].(string); ok && v != "" {
		apiObject.PolicyDocument = aws.String(v)
	}

	return apiObject
}

func expandIamInlinePolicies(roleName string, tfList []interface{}) []*iam.PutRolePolicyInput {
	if len(tfList) == 0 {
		return nil
	}

	var apiObjects []*iam.PutRolePolicyInput

	for _, tfMapRaw := range tfList {
		tfMap, ok := tfMapRaw.(map[string]interface{})

		if !ok {
			continue
		}

		apiObject := expandIamInlinePolicy(roleName, tfMap)

		if apiObject == nil {
			continue
		}

		apiObjects = append(apiObjects, apiObject)
	}

	return apiObjects
}

func addIamInlinePolicies(policies []*iam.PutRolePolicyInput, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn

	var errs *multierror.Error
	for _, policy := range policies {
		if len(aws.StringValue(policy.PolicyName)) == 0 || len(aws.StringValue(policy.PolicyDocument)) == 0 {
			continue
		}

		if _, err := conn.PutRolePolicy(policy); err != nil {
			newErr := fmt.Errorf("creating inline policy (%s): %w", aws.StringValue(policy.PolicyName), err)
			log.Printf("[ERROR] %s", newErr)
			errs = multierror.Append(errs, newErr)
		}
	}

	return errs.ErrorOrNil()
}

func addIamManagedPolicies(roleName string, policies []*string, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn

	var errs *multierror.Error
	for _, arn := range policies {
		if err := attachPolicyToRole(conn, roleName, aws.StringValue(arn)); err != nil {
			newErr := fmt.Errorf("attaching managed policy (%s): %w", aws.StringValue(arn), err)
			log.Printf("[ERROR] %s", newErr)
			errs = multierror.Append(errs, newErr)
		}
	}

	return errs.ErrorOrNil()
}

func readIamInlinePolicies(roleName string, meta interface{}) ([]*iam.PutRolePolicyInput, error) {
	conn := meta.(*conns.AWSClient).IAMConn

	policyNames, err := readIamRolePolicyNames(conn, roleName)
	if err != nil {
		return nil, err
	}

	var apiObjects []*iam.PutRolePolicyInput
	for _, policyName := range policyNames {
		policyResp, err := conn.GetRolePolicy(&iam.GetRolePolicyInput{
			RoleName:   aws.String(roleName),
			PolicyName: policyName,
		})
		if err != nil {
			return nil, err
		}

		policy, err := url.QueryUnescape(*policyResp.PolicyDocument)
		if err != nil {
			return nil, err
		}

		apiObject := &iam.PutRolePolicyInput{
			RoleName:       aws.String(roleName),
			PolicyDocument: aws.String(policy),
			PolicyName:     policyName,
		}

		apiObjects = append(apiObjects, apiObject)
	}

	if len(apiObjects) == 0 {
		apiObjects = append(apiObjects, &iam.PutRolePolicyInput{
			PolicyDocument: aws.String(""),
			PolicyName:     aws.String(""),
		})
	}

	return apiObjects, nil
}

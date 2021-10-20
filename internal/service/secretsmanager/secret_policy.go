package secretsmanager

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfiam "github.com/hashicorp/terraform-provider-aws/internal/service/iam"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceSecretPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecretPolicyCreate,
		Read:   resourceSecretPolicyRead,
		Update: resourceSecretPolicyUpdate,
		Delete: resourceSecretPolicyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"secret_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidARN,
			},
			"policy": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: verify.SuppressEquivalentPolicyDiffs,
			},
			"block_public_policy": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceSecretPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SecretsManagerConn

	input := &secretsmanager.PutResourcePolicyInput{
		ResourcePolicy: aws.String(d.Get("policy").(string)),
		SecretId:       aws.String(d.Get("secret_arn").(string)),
	}

	if v, ok := d.GetOk("block_public_policy"); ok {
		input.BlockPublicPolicy = aws.Bool(v.(bool))
	}

	log.Printf("[DEBUG] Setting Secrets Manager Secret resource policy; %#v", input)
	var res *secretsmanager.PutResourcePolicyOutput

	err := resource.Retry(tfiam.PropagationTimeout, func() *resource.RetryError {
		var err error
		res, err = conn.PutResourcePolicy(input)
		if tfawserr.ErrMessageContains(err, secretsmanager.ErrCodeMalformedPolicyDocumentException,
			"This resource policy contains an unsupported principal") {
			return resource.RetryableError(err)
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if tfresource.TimedOut(err) {
		res, err = conn.PutResourcePolicy(input)
	}
	if err != nil {
		return fmt.Errorf("error setting Secrets Manager Secret %q policy: %w", d.Id(), err)
	}

	d.SetId(aws.StringValue(res.ARN))

	return resourceSecretPolicyRead(d, meta)
}

func resourceSecretPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SecretsManagerConn

	input := &secretsmanager.GetResourcePolicyInput{
		SecretId: aws.String(d.Id()),
	}

	var res *secretsmanager.GetResourcePolicyOutput

	err := resource.Retry(PropagationTimeout, func() *resource.RetryError {
		var err error

		res, err = conn.GetResourcePolicy(input)

		if d.IsNewResource() && tfawserr.ErrCodeEquals(err, secretsmanager.ErrCodeResourceNotFoundException) {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if tfresource.TimedOut(err) {
		res, err = conn.GetResourcePolicy(input)
	}

	if !d.IsNewResource() && tfawserr.ErrCodeEquals(err, secretsmanager.ErrCodeResourceNotFoundException) {
		log.Printf("[WARN] Secrets Manager Secret Policy (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading Secrets Manager Secret Policy (%s): %w", d.Id(), err)
	}

	if res == nil {
		return fmt.Errorf("error reading Secrets Manager Secret Policy (%s): empty response", d.Id())
	}

	if res.ResourcePolicy != nil {
		policy, err := structure.NormalizeJsonString(aws.StringValue(res.ResourcePolicy))
		if err != nil {
			return fmt.Errorf("policy contains an invalid JSON: %w", err)
		}
		d.Set("policy", policy)
	} else {
		d.Set("policy", "")
	}
	d.Set("secret_arn", d.Id())

	return nil
}

func resourceSecretPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SecretsManagerConn

	if d.HasChanges("policy", "block_public_policy") {
		policy, err := structure.NormalizeJsonString(d.Get("policy").(string))
		if err != nil {
			return fmt.Errorf("policy contains an invalid JSON: %s", err)
		}
		input := &secretsmanager.PutResourcePolicyInput{
			ResourcePolicy:    aws.String(policy),
			SecretId:          aws.String(d.Id()),
			BlockPublicPolicy: aws.Bool(d.Get("block_public_policy").(bool)),
		}

		log.Printf("[DEBUG] Setting Secrets Manager Secret resource policy; %#v", input)
		err = resource.Retry(tfiam.PropagationTimeout, func() *resource.RetryError {
			_, err := conn.PutResourcePolicy(input)
			if tfawserr.ErrMessageContains(err, secretsmanager.ErrCodeMalformedPolicyDocumentException,
				"This resource policy contains an unsupported principal") {
				return resource.RetryableError(err)
			}
			if err != nil {
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if tfresource.TimedOut(err) {
			_, err = conn.PutResourcePolicy(input)
		}
		if err != nil {
			return fmt.Errorf("error setting Secrets Manager Secret %q policy: %w", d.Id(), err)
		}
	}

	return resourceSecretPolicyRead(d, meta)
}

func resourceSecretPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SecretsManagerConn

	input := &secretsmanager.DeleteResourcePolicyInput{
		SecretId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Removing Secrets Manager Secret policy: %#v", input)
	_, err := conn.DeleteResourcePolicy(input)
	if err != nil {
		if tfawserr.ErrMessageContains(err, secretsmanager.ErrCodeResourceNotFoundException, "") {
			return nil
		}
		return fmt.Errorf("error removing Secrets Manager Secret %q policy: %w", d.Id(), err)
	}

	return nil
}

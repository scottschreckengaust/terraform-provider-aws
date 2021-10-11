package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/iot/finder"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

func resourceAwsIoTAuthorizer() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIotAuthorizerCreate,
		Read:   resourceAwsIotAuthorizerRead,
		Update: resourceAwsIotAuthorizerUpdate,
		Delete: resourceAwsIotAuthorizerDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		CustomizeDiff: resourceAwsIotAuthorizerCustomizeDiff,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"authorizer_function_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 128),
					validation.StringMatch(regexp.MustCompile(`^[\w=,@-]+`), "must contain only alphanumeric characters, underscores, and hyphens"),
				),
			},
			"signing_disabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"status": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      iot.AuthorizerStatusActive,
				ValidateFunc: validation.StringInSlice(iot.AuthorizerStatus_Values(), false),
			},
			"token_key_name": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 128),
					validation.StringMatch(regexp.MustCompile(`^[A-Za-z0-9_-]+`), "must contain only alphanumeric characters, underscores, and hyphens"),
				),
			},
			"token_signing_public_keys": {
				Type:      schema.TypeMap,
				Optional:  true,
				Elem:      &schema.Schema{Type: schema.TypeString},
				Sensitive: true,
			},
		},
	}
}

func resourceAwsIotAuthorizerCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	name := d.Get("name").(string)
	input := &iot.CreateAuthorizerInput{
		AuthorizerFunctionArn: aws.String(d.Get("authorizer_function_arn").(string)),
		AuthorizerName:        aws.String(name),
		SigningDisabled:       aws.Bool(d.Get("signing_disabled").(bool)),
		Status:                aws.String(d.Get("status").(string)),
	}

	if v, ok := d.GetOk("token_key_name"); ok {
		input.TokenKeyName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("token_signing_public_keys"); ok {
		input.TokenSigningPublicKeys = expandStringMap(v.(map[string]interface{}))
	}

	log.Printf("[INFO] Creating IoT Authorizer: %s", input)
	output, err := conn.CreateAuthorizer(input)

	if err != nil {
		return fmt.Errorf("error creating IoT Authorizer (%s): %w", name, err)
	}

	d.SetId(aws.StringValue(output.AuthorizerName))

	return resourceAwsIotAuthorizerRead(d, meta)
}

func resourceAwsIotAuthorizerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	authorizer, err := finder.AuthorizerByName(conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] IoT Authorizer (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading IoT Authorizer (%s): %w", d.Id(), err)
	}

	d.Set("arn", authorizer.AuthorizerArn)
	d.Set("authorizer_function_arn", authorizer.AuthorizerFunctionArn)
	d.Set("name", authorizer.AuthorizerName)
	d.Set("signing_disabled", authorizer.SigningDisabled)
	d.Set("status", authorizer.Status)
	d.Set("token_key_name", authorizer.TokenKeyName)
	d.Set("token_signing_public_keys", aws.StringValueMap(authorizer.TokenSigningPublicKeys))

	return nil
}

func resourceAwsIotAuthorizerUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	input := iot.UpdateAuthorizerInput{
		AuthorizerName: aws.String(d.Id()),
	}

	if d.HasChange("authorizer_function_arn") {
		input.AuthorizerFunctionArn = aws.String(d.Get("authorizer_function_arn").(string))
	}

	if d.HasChange("status") {
		input.Status = aws.String(d.Get("status").(string))
	}

	if d.HasChange("token_key_name") {
		input.TokenKeyName = aws.String(d.Get("token_key_name").(string))
	}

	if d.HasChange("token_signing_public_keys") {
		input.TokenSigningPublicKeys = expandStringMap(d.Get("token_signing_public_keys").(map[string]interface{}))
	}

	log.Printf("[INFO] Updating IoT Authorizer: %s", input)
	_, err := conn.UpdateAuthorizer(&input)

	if err != nil {
		return fmt.Errorf("error updating IoT Authorizer (%s): %w", d.Id(), err)
	}

	return resourceAwsIotAuthorizerRead(d, meta)
}

func resourceAwsIotAuthorizerDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	// In order to delete an IoT Authorizer, you must set it inactive first.
	if d.Get("status").(string) == iot.AuthorizerStatusActive {
		log.Printf("[INFO] Deactivating IoT Authorizer: %s", d.Id())
		_, err := conn.UpdateAuthorizer(&iot.UpdateAuthorizerInput{
			AuthorizerName: aws.String(d.Id()),
			Status:         aws.String(iot.AuthorizerStatusInactive),
		})

		if err != nil {
			return fmt.Errorf("error deactivating IoT Authorizer (%s): %w", d.Id(), err)
		}
	}

	log.Printf("[INFO] Deleting IoT Authorizer: %s", d.Id())
	_, err := conn.DeleteAuthorizer(&iot.DeleteAuthorizerInput{
		AuthorizerName: aws.String(d.Id()),
	})

	if tfawserr.ErrCodeEquals(err, iot.ErrCodeResourceNotFoundException) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting IOT Authorizer (%s): %w", d.Id(), err)
	}

	return nil
}

func resourceAwsIotAuthorizerCustomizeDiff(_ context.Context, diff *schema.ResourceDiff, v interface{}) error {
	if !diff.Get("signing_disabled").(bool) {
		if _, ok := diff.GetOk("token_key_name"); !ok {
			return errors.New(`"token_key_name" is required when signing is enabled`)
		}
		if _, ok := diff.GetOk("token_signing_public_keys"); !ok {
			return errors.New(`"token_signing_public_keys" is required when signing is enabled`)
		}
	}

	return nil
}

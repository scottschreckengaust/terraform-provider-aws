package aws

import (
	"regexp"

	"github.com/aws/aws-sdk-go/service/healthlake"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAwsHealthLakeFHIRDatastore() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsHealthLakeFHIRDatasourceCreate,
		Read:   resourceAwsHealthLakeFHIRDatasourceRead,
		// NOTE: No update in AWS SDK Go for Amazon HealthLake
		Delete: resourceAwsHealthLakeFHIRDatasourceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"created_at": {
				Type: schema.TypeString,
				// TODO: if API says "No" is `Required: false`? or since `Required` is default `false` is left out?
				Computed: true,
			},
			"datastore_arn": {
				Type:     schema.TypeString,
				Required: true, // TODO: if API says "Yes", then set `Required: true`?
				Computed: true,
				// TODO: Pattern validation?
			},
			"datastore_endpoint": {
				Type:     schema.TypeString,
				Required: true, // API says "Yes"
				Computed: true,
				// TODO: Pattern length and validation?
			},
			"datastore_id": {
				Type:     schema.TypeString,
				Required: true, // API says "Yes"
				Computed: true,
				// TODO: Pattern validation?
			},
			"datastore_name": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 256),
					validation.StringMatch(regexp.MustCompile(`^([\p{L}\p{Z}\p{N}_.:/=+\-%@]*)$`), "Only letters, numbers, separators, or these symbols: underscore '_', period '.', forward slash '/', equals '=', plus '+', minus '-', percentage '%', at sign '@' permitted."),
				),
			},
			"datastore_status": {
				Type:     schema.TypeString,
				Required: true, // API says "Yes"
				Computed: true,
				// ExactlyOneOf: healthlake.DatastoreStatus_Values(),
			},
			"datastore_type_version": {
				Type:         schema.TypeString,
				Required:     true, // Can it be optional and required and defaulted?
				Optional:     true,
				ExactlyOneOf: healthlake.FHIRVersion_Values(), // Okay to use the AWS SDK Go string arrays? I do not see this pattern anywhere, but why?
				Default:      healthlake.FHIRVersionR4,        // "R4" if there is only one value, why not default it? Can use the SDK value too?
			},
			"preload_data_config": {
				Type:     schema.TypeSet,
				Required: false, // API says "No"
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"preload_data_type": {
							Type:         schema.TypeString,
							Required:     true,
							ExactlyOneOf: healthlake.PreloadDataType_Values(), // Okay to use SDK array?
							Default:      healthlake.PreloadDataTypeSynthea,   // "SYNTHEA" is the only one allowed and use SDK?
						},
					},
				},
			},
			"sse_configuration": {
				Type:     schema.TypeSet,
				Required: false, // API says "No"
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kms_encryption_config": {
							Type:     schema.TypeSet,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cmk_type": {
										Type:         schema.TypeString,
										Required:     true,
										ExactlyOneOf: healthlake.CmkType_Values(), // Okay to use the SDK values?
									},
									"kms_key_id": {
										Type:     schema.TypeString,
										Required: false, // API Question: How is CmkType required, and not the KmsKeyId???
										ValidateFunc: validation.All(
											validation.StringLenBetween(1, 400),
											validation.StringMatch(regexp.MustCompile(`(arn:aws((-us-gov)|(-iso)|(-iso-b)|(-cn))?:kms:)?([a-z]{2}-[a-z]+(-[a-z]+)?-\d:)?(\d{12}:)?(((key/)?[a-zA-Z0-9-_]+)|(alias/[a-zA-Z0-9:/_-]+))`), "Key does not match a 'key' or 'alias' regular expression"),
										),
									},
								},
							},
						},
					},
				},
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsHealthLakeFHIRDatasourceCreate(d *schema.ResourceData, meta interface{}) error {
	// healthlakeconn := meta.(*AWSClient).healthlakeconn
	return resourceAwsHealthLakeFHIRDatasourceRead(d, meta)
}

func resourceAwsHealthLakeFHIRDatasourceRead(d *schema.ResourceData, meta interface{}) error {
	// healthlakeconn := meta.(*AWSClient).healthlakeconn
	return nil
}

func resourceAwsHealthLakeFHIRDatasourceDelete(d *schema.ResourceData, meta interface{}) error {
	// healthlakeconn := meta.(*AWSClient).healthlakeconn
	return nil
}

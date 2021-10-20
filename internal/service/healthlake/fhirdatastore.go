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
				Type:     schema.TypeString,
				Computed: true,
			},
			"datastore_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datastore_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datastore_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datastore_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 256),
					//TODO: This differs from the console. Open issue with the service team?
					validation.StringMatch(regexp.MustCompile(`^([\p{L}\p{Z}\p{N}_.:/=+\-%@]*)$`), "Only letters, numbers, separators, or these symbols: underscore '_', period '.', forward slash '/', equals '=', plus '+', minus '-', percentage '%', at sign '@' permitted."),
				),
			},
			"datastore_status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datastore_type_version": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(healthlake.FHIRVersion_Values(), false),
				Default:      healthlake.FHIRVersionR4,
			},
			"preload_data_config": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"preload_data_type": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringInSlice(healthlake.PreloadDataType_Values(), false),
						},
					},
				},
			},
			"sse_configuration": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kms_encryption_config": {
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cmk_type": {
										Type:         schema.TypeString,
										Required:     true,
										ForceNew:     true,
										ValidateFunc: validation.StringInSlice(healthlake.CmkType_Values(), false),
									},
									"kms_key_id": {
										Type:     schema.TypeString,
										ForceNew: true,
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
			"tags": tagsSchemaForceNew(),
		},
	}
}

func resourceHealthLakeFHIRDatasourceCreate(d *schema.ResourceData, meta interface{}) error {
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

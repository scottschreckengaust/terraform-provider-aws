package aws

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/healthlake"

	//	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	//	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func ResourceFHIRDatastore() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFHIRDatasourceCreate,
		ReadContext:   resourceFHIRDatasourceRead,
		// NOTE: No update in AWS SDK Go for Amazon HealthLake
		DeleteContext: resourceFHIRDatasourceDelete,
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
			"tags": tftags.TagsSchemaForceNew(),
		},
	}
}

func resourceFHIRDatasourceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).HealthLakeConn

	//QUESTION: If there is an optional default, will it be "filled"?
	input := &healthlake.CreateFHIRDatastoreInput{
		ClientToken:          aws.String(resource.UniqueId()),
		DatastoreTypeVersion: aws.String(d.Get("data_type_version").(string)),
	}

	var err error
	var resp *healthlake.CreateFHIRDatastoreOutput

	resp, err = conn.CreateFHIRDatastoreWithContext(ctx, input)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating Amazon HealthLake FHIRDatastore (%s): %w", d.Id(), err))
	}

	d.SetId(aws.StringValue(resp.DatastoreId))
	return resourceFHIRDatasourceRead(ctx, d, meta)
}

func resourceFHIRDatasourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).HealthLakeConn
	input := &healthlake.DescribeFHIRDatastoreInput{
		DatastoreId: aws.String(d.Id()),
	}
	resp, err := conn.DescribeFHIRDatastoreWithContext(ctx, input)

	//QUESTION: Are there errors that we want to not report failure?
	// if tfawserr.ErrCodeEquals(err, healthlake.ErrCodeResourceNotFoundException) {
	// 	//|| tfawserr.ErrMessageContains(err, healthlake.ErrCodeAccessDeniedException, "HealthLake is not enabled")
	// 	log.Printf("[WARN] Amazon HealthLake FHIRDatastore (%s) not found, removing from state", d.Id())
	// 	d.SetId("")
	// 	return nil
	// }
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading Amazon HealthLake FHIRDatastore (%s): %w", d.Id(), err))
	}

	d.Set("created_at", resp.DatastoreProperties.CreatedAt.Format(time.RFC3339))
	d.Set("datastore_arn", resp.DatastoreProperties.DatastoreArn)
	d.Set("datastore_endpoint", resp.DatastoreProperties.DatastoreEndpoint)
	d.Set("datastore_id", resp.DatastoreProperties.DatastoreId)
	d.Set("datastore_name", resp.DatastoreProperties.DatastoreName)
	d.Set("datastore_status", resp.DatastoreProperties.DatastoreStatus)
	d.Set("datastore_type_version", resp.DatastoreProperties.DatastoreTypeVersion)
	//resp.DatastoreProperties.PreloadDataConfig
	//resp.DatastoreProperties.SseConfiguration
	return nil
}

func resourceFHIRDatasourceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).HealthLakeConn
	input := &healthlake.DeleteFHIRDatastoreInput{
		DatastoreId: aws.String(d.Id()),
	}
	_, err := conn.DeleteFHIRDatastoreWithContext(ctx, input)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting Amazon HealthLake FHIRDatastore (%s): %w", d.Id(), err))
	}
	return nil
}

package timestreamwrite

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/timestreamwrite"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceTable() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceTableCreate,
		ReadWithoutTimeout:   resourceTableRead,
		UpdateWithoutTimeout: resourceTableUpdate,
		DeleteWithoutTimeout: resourceTableDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"database_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(3, 64),
					validation.StringMatch(regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`), "must only include alphanumeric, underscore, period, or hyphen characters"),
				),
			},

			"retention_properties": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"magnetic_store_retention_period_in_days": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 73000),
						},

						"memory_store_retention_period_in_hours": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 8766),
						},
					},
				},
			},

			"table_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(3, 64),
					validation.StringMatch(regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`), "must only include alphanumeric, underscore, period, or hyphen characters"),
				),
			},

			"tags": tftags.TagsSchema(),

			"tags_all": tftags.TagsSchemaComputed(),
		},

		CustomizeDiff: verify.SetTagsDiff,
	}
}

func resourceTableCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).TimestreamWriteConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(d.Get("tags").(map[string]interface{})))

	tableName := d.Get("table_name").(string)
	input := &timestreamwrite.CreateTableInput{
		DatabaseName: aws.String(d.Get("database_name").(string)),
		TableName:    aws.String(tableName),
	}

	if v, ok := d.GetOk("retention_properties"); ok && len(v.([]interface{})) > 0 && v.([]interface{}) != nil {
		input.RetentionProperties = expandTimestreamWriteRetentionProperties(v.([]interface{}))
	}

	if len(tags) > 0 {
		input.Tags = Tags(tags.IgnoreAWS())
	}

	output, err := conn.CreateTableWithContext(ctx, input)

	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating Timestream Table (%s): %w", tableName, err))
	}

	if output == nil || output.Table == nil {
		return diag.FromErr(fmt.Errorf("error creating Timestream Table (%s): empty output", tableName))
	}

	d.SetId(fmt.Sprintf("%s:%s", aws.StringValue(output.Table.TableName), aws.StringValue(output.Table.DatabaseName)))

	return resourceTableRead(ctx, d, meta)
}

func resourceTableRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).TimestreamWriteConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	tableName, databaseName, err := TableParseID(d.Id())

	if err != nil {
		return diag.FromErr(err)
	}

	input := &timestreamwrite.DescribeTableInput{
		DatabaseName: aws.String(databaseName),
		TableName:    aws.String(tableName),
	}

	output, err := conn.DescribeTableWithContext(ctx, input)

	if !d.IsNewResource() && tfawserr.ErrCodeEquals(err, timestreamwrite.ErrCodeResourceNotFoundException) {
		log.Printf("[WARN] Timestream Table %s not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if output == nil || output.Table == nil {
		return diag.FromErr(fmt.Errorf("error reading Timestream Table (%s): empty output", d.Id()))
	}

	table := output.Table
	arn := aws.StringValue(table.Arn)

	d.Set("arn", arn)
	d.Set("database_name", table.DatabaseName)

	if err := d.Set("retention_properties", flattenTimestreamWriteRetentionProperties(table.RetentionProperties)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting retention_properties: %w", err))
	}

	d.Set("table_name", table.TableName)

	tags, err := ListTags(conn, arn)

	if err != nil {
		return diag.FromErr(fmt.Errorf("error listing tags for Timestream Table (%s): %w", arn, err))
	}

	tags = tags.IgnoreAWS().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return diag.FromErr(fmt.Errorf("error setting tags: %w", err))
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return diag.FromErr(fmt.Errorf("error setting tags_all: %w", err))
	}

	return nil
}

func resourceTableUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).TimestreamWriteConn

	if d.HasChange("retention_properties") {
		tableName, databaseName, err := TableParseID(d.Id())

		if err != nil {
			return diag.FromErr(err)
		}

		input := &timestreamwrite.UpdateTableInput{
			DatabaseName:        aws.String(databaseName),
			RetentionProperties: expandTimestreamWriteRetentionProperties(d.Get("retention_properties").([]interface{})),
			TableName:           aws.String(tableName),
		}

		_, err = conn.UpdateTableWithContext(ctx, input)

		if err != nil {
			return diag.FromErr(fmt.Errorf("error updating Timestream Table (%s): %w", d.Id(), err))
		}
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := UpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return diag.FromErr(fmt.Errorf("error updating Timestream Table (%s) tags: %w", d.Get("arn").(string), err))
		}
	}

	return resourceTableRead(ctx, d, meta)
}

func resourceTableDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).TimestreamWriteConn

	tableName, databaseName, err := TableParseID(d.Id())

	if err != nil {
		return diag.FromErr(err)
	}

	input := &timestreamwrite.DeleteTableInput{
		DatabaseName: aws.String(databaseName),
		TableName:    aws.String(tableName),
	}

	_, err = conn.DeleteTableWithContext(ctx, input)

	if tfawserr.ErrCodeEquals(err, timestreamwrite.ErrCodeResourceNotFoundException) {
		return nil
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting Timestream Table (%s): %w", d.Id(), err))
	}

	return nil
}

func expandTimestreamWriteRetentionProperties(l []interface{}) *timestreamwrite.RetentionProperties {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})

	if !ok {
		return nil
	}

	rp := &timestreamwrite.RetentionProperties{}

	if v, ok := tfMap["magnetic_store_retention_period_in_days"].(int); ok {
		rp.MagneticStoreRetentionPeriodInDays = aws.Int64(int64(v))
	}

	if v, ok := tfMap["memory_store_retention_period_in_hours"].(int); ok {
		rp.MemoryStoreRetentionPeriodInHours = aws.Int64(int64(v))
	}

	return rp
}

func flattenTimestreamWriteRetentionProperties(rp *timestreamwrite.RetentionProperties) []interface{} {
	if rp == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"magnetic_store_retention_period_in_days": aws.Int64Value(rp.MagneticStoreRetentionPeriodInDays),
		"memory_store_retention_period_in_hours":  aws.Int64Value(rp.MemoryStoreRetentionPeriodInHours),
	}

	return []interface{}{m}
}

func TableParseID(id string) (string, string, error) {
	idParts := strings.SplitN(id, ":", 2)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected table_name:database_name", id)
	}
	return idParts[0], idParts[1], nil
}

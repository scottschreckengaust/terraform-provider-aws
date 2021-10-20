package sns

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceTopic() *schema.Resource {
	return &schema.Resource{
		Create: resourceTopicCreate,
		Read:   resourceTopicRead,
		Update: resourceTopicUpdate,
		Delete: resourceTopicDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		CustomizeDiff: customdiff.Sequence(
			resourceTopicCustomizeDiff,
			verify.SetTagsDiff,
		),

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
			},
			"display_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"policy": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: verify.SuppressEquivalentPolicyDiffs,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},
			"delivery_policy": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         false,
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: verify.SuppressEquivalentJSONDiffs,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},
			"application_success_feedback_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"application_success_feedback_sample_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 100),
			},
			"application_failure_feedback_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"http_success_feedback_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"http_success_feedback_sample_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 100),
			},
			"http_failure_feedback_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"kms_master_key_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"fifo_topic": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"content_based_deduplication": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"firehose_success_feedback_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"firehose_success_feedback_sample_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 100),
			},
			"firehose_failure_feedback_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"lambda_success_feedback_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"lambda_success_feedback_sample_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 100),
			},
			"lambda_failure_feedback_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"sqs_success_feedback_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"sqs_success_feedback_sample_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 100),
			},
			"sqs_failure_feedback_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),
		},
	}
}

func resourceTopicCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SNSConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(d.Get("tags").(map[string]interface{})))

	var name string
	fifoTopic := d.Get("fifo_topic").(bool)

	if fifoTopic {
		name = create.NameWithSuffix(d.Get("name").(string), d.Get("name_prefix").(string), FIFOTopicNameSuffix)
	} else {
		name = create.Name(d.Get("name").(string), d.Get("name_prefix").(string))
	}

	attributes := make(map[string]*string)
	// If FifoTopic is true, then the attribute must be passed into the call to CreateTopic
	if fifoTopic {
		attributes["FifoTopic"] = aws.String(strconv.FormatBool(fifoTopic))
	}

	log.Printf("[DEBUG] SNS create topic: %s", name)

	req := &sns.CreateTopicInput{
		Name: aws.String(name),
		Tags: Tags(tags.IgnoreAWS()),
	}

	if len(attributes) > 0 {
		req.Attributes = attributes
	}

	output, err := conn.CreateTopic(req)
	if err != nil {
		return fmt.Errorf("error creating SNS Topic (%s): %w", name, err)
	}

	d.SetId(aws.StringValue(output.TopicArn))

	// update mutable attributes
	if d.HasChange("application_failure_feedback_role_arn") {
		_, v := d.GetChange("application_failure_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "ApplicationFailureFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("application_success_feedback_role_arn") {
		_, v := d.GetChange("application_success_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "ApplicationSuccessFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("arn") {
		_, v := d.GetChange("arn")
		if err := updateTopicAttribute(d.Id(), "TopicArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("delivery_policy") {
		_, v := d.GetChange("delivery_policy")
		if err := updateTopicAttribute(d.Id(), "DeliveryPolicy", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("display_name") {
		_, v := d.GetChange("display_name")
		if err := updateTopicAttribute(d.Id(), "DisplayName", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("http_failure_feedback_role_arn") {
		_, v := d.GetChange("http_failure_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "HTTPFailureFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("http_success_feedback_role_arn") {
		_, v := d.GetChange("http_success_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "HTTPSuccessFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("kms_master_key_id") {
		_, v := d.GetChange("kms_master_key_id")
		if err := updateTopicAttribute(d.Id(), "KmsMasterKeyId", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("content_based_deduplication") {
		_, v := d.GetChange("content_based_deduplication")
		if err := updateTopicAttribute(d.Id(), "ContentBasedDeduplication", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("lambda_failure_feedback_role_arn") {
		_, v := d.GetChange("lambda_failure_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "LambdaFailureFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("lambda_success_feedback_role_arn") {
		_, v := d.GetChange("lambda_success_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "LambdaSuccessFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("policy") {
		_, v := d.GetChange("policy")
		if err := updateTopicAttribute(d.Id(), "Policy", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("sqs_failure_feedback_role_arn") {
		_, v := d.GetChange("sqs_failure_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "SQSFailureFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("sqs_success_feedback_role_arn") {
		_, v := d.GetChange("sqs_success_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "SQSSuccessFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("application_success_feedback_sample_rate") {
		_, v := d.GetChange("application_success_feedback_sample_rate")
		if err := updateTopicAttribute(d.Id(), "ApplicationSuccessFeedbackSampleRate", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("http_success_feedback_sample_rate") {
		_, v := d.GetChange("http_success_feedback_sample_rate")
		if err := updateTopicAttribute(d.Id(), "HTTPSuccessFeedbackSampleRate", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("lambda_success_feedback_sample_rate") {
		_, v := d.GetChange("lambda_success_feedback_sample_rate")
		if err := updateTopicAttribute(d.Id(), "LambdaSuccessFeedbackSampleRate", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("sqs_success_feedback_sample_rate") {
		_, v := d.GetChange("sqs_success_feedback_sample_rate")
		if err := updateTopicAttribute(d.Id(), "SQSSuccessFeedbackSampleRate", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("firehose_failure_feedback_role_arn") {
		_, v := d.GetChange("firehose_failure_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "FirehoseFailureFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("firehose_success_feedback_role_arn") {
		_, v := d.GetChange("firehose_success_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "FirehoseSuccessFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("firehose_success_feedback_sample_rate") {
		_, v := d.GetChange("firehose_success_feedback_sample_rate")
		if err := updateTopicAttribute(d.Id(), "FirehoseSuccessFeedbackSampleRate", v, conn); err != nil {
			return err
		}
	}

	return resourceTopicRead(d, meta)
}

func resourceTopicUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SNSConn

	// update mutable attributes
	if d.HasChange("application_failure_feedback_role_arn") {
		_, v := d.GetChange("application_failure_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "ApplicationFailureFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("application_success_feedback_role_arn") {
		_, v := d.GetChange("application_success_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "ApplicationSuccessFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("arn") {
		_, v := d.GetChange("arn")
		if err := updateTopicAttribute(d.Id(), "TopicArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("delivery_policy") {
		_, v := d.GetChange("delivery_policy")
		if err := updateTopicAttribute(d.Id(), "DeliveryPolicy", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("display_name") {
		_, v := d.GetChange("display_name")
		if err := updateTopicAttribute(d.Id(), "DisplayName", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("http_failure_feedback_role_arn") {
		_, v := d.GetChange("http_failure_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "HTTPFailureFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("http_success_feedback_role_arn") {
		_, v := d.GetChange("http_success_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "HTTPSuccessFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("kms_master_key_id") {
		_, v := d.GetChange("kms_master_key_id")
		if err := updateTopicAttribute(d.Id(), "KmsMasterKeyId", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("content_based_deduplication") {
		_, v := d.GetChange("content_based_deduplication")
		if err := updateTopicAttribute(d.Id(), "ContentBasedDeduplication", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("lambda_failure_feedback_role_arn") {
		_, v := d.GetChange("lambda_failure_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "LambdaFailureFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("lambda_success_feedback_role_arn") {
		_, v := d.GetChange("lambda_success_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "LambdaSuccessFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("policy") {
		_, v := d.GetChange("policy")
		if err := updateTopicAttribute(d.Id(), "Policy", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("sqs_failure_feedback_role_arn") {
		_, v := d.GetChange("sqs_failure_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "SQSFailureFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("sqs_success_feedback_role_arn") {
		_, v := d.GetChange("sqs_success_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "SQSSuccessFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("application_success_feedback_sample_rate") {
		_, v := d.GetChange("application_success_feedback_sample_rate")
		if err := updateTopicAttribute(d.Id(), "ApplicationSuccessFeedbackSampleRate", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("http_success_feedback_sample_rate") {
		_, v := d.GetChange("http_success_feedback_sample_rate")
		if err := updateTopicAttribute(d.Id(), "HTTPSuccessFeedbackSampleRate", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("lambda_success_feedback_sample_rate") {
		_, v := d.GetChange("lambda_success_feedback_sample_rate")
		if err := updateTopicAttribute(d.Id(), "LambdaSuccessFeedbackSampleRate", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("sqs_success_feedback_sample_rate") {
		_, v := d.GetChange("sqs_success_feedback_sample_rate")
		if err := updateTopicAttribute(d.Id(), "SQSSuccessFeedbackSampleRate", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("firehose_failure_feedback_role_arn") {
		_, v := d.GetChange("firehose_failure_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "FirehoseFailureFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("firehose_success_feedback_role_arn") {
		_, v := d.GetChange("firehose_success_feedback_role_arn")
		if err := updateTopicAttribute(d.Id(), "FirehoseSuccessFeedbackRoleArn", v, conn); err != nil {
			return err
		}
	}
	if d.HasChange("firehose_success_feedback_sample_rate") {
		_, v := d.GetChange("firehose_success_feedback_sample_rate")
		if err := updateTopicAttribute(d.Id(), "FirehoseSuccessFeedbackSampleRate", v, conn); err != nil {
			return err
		}
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")
		if err := UpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating tags: %w", err)
		}
	}

	return resourceTopicRead(d, meta)
}

func resourceTopicRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SNSConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	log.Printf("[DEBUG] Reading SNS Topic Attributes for %s", d.Id())
	attributeOutput, err := conn.GetTopicAttributes(&sns.GetTopicAttributesInput{
		TopicArn: aws.String(d.Id()),
	})

	if tfawserr.ErrMessageContains(err, sns.ErrCodeNotFoundException, "") {
		log.Printf("[WARN] SNS Topic (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading SNS Topic (%s): %w", d.Id(), err)
	}

	fifoTopic := false

	// set the mutable attributes
	if attributeOutput.Attributes != nil && len(attributeOutput.Attributes) > 0 {
		// set the string values
		d.Set("application_failure_feedback_role_arn", attributeOutput.Attributes["ApplicationFailureFeedbackRoleArn"])
		d.Set("application_success_feedback_role_arn", attributeOutput.Attributes["ApplicationSuccessFeedbackRoleArn"])
		d.Set("arn", attributeOutput.Attributes["TopicArn"])
		d.Set("delivery_policy", attributeOutput.Attributes["DeliveryPolicy"])
		d.Set("display_name", attributeOutput.Attributes["DisplayName"])
		d.Set("http_failure_feedback_role_arn", attributeOutput.Attributes["HTTPFailureFeedbackRoleArn"])
		d.Set("http_success_feedback_role_arn", attributeOutput.Attributes["HTTPSuccessFeedbackRoleArn"])
		d.Set("kms_master_key_id", attributeOutput.Attributes["KmsMasterKeyId"])
		d.Set("lambda_failure_feedback_role_arn", attributeOutput.Attributes["LambdaFailureFeedbackRoleArn"])
		d.Set("lambda_success_feedback_role_arn", attributeOutput.Attributes["LambdaSuccessFeedbackRoleArn"])
		d.Set("policy", attributeOutput.Attributes["Policy"])
		d.Set("sqs_failure_feedback_role_arn", attributeOutput.Attributes["SQSFailureFeedbackRoleArn"])
		d.Set("sqs_success_feedback_role_arn", attributeOutput.Attributes["SQSSuccessFeedbackRoleArn"])
		d.Set("firehose_success_feedback_role_arn", attributeOutput.Attributes["FirehoseSuccessFeedbackRoleArn"])
		d.Set("firehose_failure_feedback_role_arn", attributeOutput.Attributes["FirehoseFailureFeedbackRoleArn"])
		d.Set("owner", attributeOutput.Attributes["Owner"])

		// set the boolean values
		if v, ok := attributeOutput.Attributes["FifoTopic"]; ok && aws.StringValue(v) == "true" {
			fifoTopic = true
		}
		d.Set("content_based_deduplication", false)
		if v, ok := attributeOutput.Attributes["ContentBasedDeduplication"]; ok && aws.StringValue(v) == "true" {
			d.Set("content_based_deduplication", true)
		}

		// set the number values
		var vStr string
		var v int64
		var err error

		vStr = aws.StringValue(attributeOutput.Attributes["ApplicationSuccessFeedbackSampleRate"])
		if vStr != "" {
			v, err = strconv.ParseInt(vStr, 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing integer attribute 'ApplicationSuccessFeedbackSampleRate': %w", err)
			}
			d.Set("application_success_feedback_sample_rate", v)
		}

		vStr = aws.StringValue(attributeOutput.Attributes["HTTPSuccessFeedbackSampleRate"])
		if vStr != "" {
			v, err = strconv.ParseInt(vStr, 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing integer attribute 'HTTPSuccessFeedbackSampleRate': %w", err)
			}
			d.Set("http_success_feedback_sample_rate", v)
		}

		vStr = aws.StringValue(attributeOutput.Attributes["LambdaSuccessFeedbackSampleRate"])
		if vStr != "" {
			v, err = strconv.ParseInt(vStr, 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing integer attribute 'LambdaSuccessFeedbackSampleRate': %w", err)
			}
			d.Set("lambda_success_feedback_sample_rate", v)
		}

		vStr = aws.StringValue(attributeOutput.Attributes["SQSSuccessFeedbackSampleRate"])
		if vStr != "" {
			v, err = strconv.ParseInt(vStr, 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing integer attribute 'SQSSuccessFeedbackSampleRate': %w", err)
			}
			d.Set("sqs_success_feedback_sample_rate", v)
		}

		vStr = aws.StringValue(attributeOutput.Attributes["FirehoseSuccessFeedbackSampleRate"])
		if vStr != "" {
			v, err = strconv.ParseInt(vStr, 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing integer attribute 'FirehoseSuccessFeedbackSampleRate': %w", err)
			}
			d.Set("firehose_success_feedback_sample_rate", v)
		}
	}

	d.Set("fifo_topic", fifoTopic)

	arn, err := arn.Parse(d.Id())

	if err != nil {
		return fmt.Errorf("error parsing ARN (%s): %w", d.Id(), err)
	}

	name := arn.Resource
	d.Set("name", name)
	if fifoTopic {
		d.Set("name_prefix", create.NamePrefixFromNameWithSuffix(name, FIFOTopicNameSuffix))
	} else {
		d.Set("name_prefix", create.NamePrefixFromName(name))
	}

	tags, err := ListTags(conn, d.Id())

	if err != nil {
		return fmt.Errorf("error listing tags for SNS Topic (%s): %w", d.Id(), err)
	}

	tags = tags.IgnoreAWS().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	return nil
}

func resourceTopicDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SNSConn

	log.Printf("[DEBUG] SNS Delete Topic: %s", d.Id())
	_, err := conn.DeleteTopic(&sns.DeleteTopicInput{
		TopicArn: aws.String(d.Id()),
	})

	if err != nil {
		if tfawserr.ErrMessageContains(err, sns.ErrCodeNotFoundException, "") {
			return nil
		}
		return fmt.Errorf("error deleting SNS Topic (%s): %w", d.Id(), err)
	}

	return nil
}

func resourceTopicCustomizeDiff(_ context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	fifoTopic := diff.Get("fifo_topic").(bool)
	contentBasedDeduplication := diff.Get("content_based_deduplication").(bool)

	if diff.Id() == "" {
		// Create.

		var name string

		if fifoTopic {
			name = create.NameWithSuffix(diff.Get("name").(string), diff.Get("name_prefix").(string), FIFOTopicNameSuffix)
		} else {
			name = create.Name(diff.Get("name").(string), diff.Get("name_prefix").(string))
		}

		var re *regexp.Regexp

		if fifoTopic {
			re = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,251}\.fifo$`)
		} else {
			re = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,256}$`)
		}

		if !re.MatchString(name) {
			return fmt.Errorf("invalid topic name: %s", name)
		}

	}

	if !fifoTopic && contentBasedDeduplication {
		return fmt.Errorf("content-based deduplication can only be set for FIFO topics")
	}

	return nil
}

func updateTopicAttribute(topicArn, name string, value interface{}, conn *sns.SNS) error {
	// Ignore an empty policy
	if name == "Policy" && value == "" {
		return nil
	}
	log.Printf("[DEBUG] Updating SNS Topic Attribute: %s", name)

	// Make API call to update attributes
	req := sns.SetTopicAttributesInput{
		TopicArn:       aws.String(topicArn),
		AttributeName:  aws.String(name),
		AttributeValue: aws.String(fmt.Sprintf("%v", value)),
	}

	// Retry the update in the event of an eventually consistent style of
	// error, where say an IAM resource is successfully created but not
	// actually available. See https://github.com/hashicorp/terraform/issues/3660
	_, err := verify.RetryOnAWSCode(sns.ErrCodeInvalidParameterException, func() (interface{}, error) {
		return conn.SetTopicAttributes(&req)
	})

	if err != nil {
		return fmt.Errorf("error setting SNS Topic (%s) attributes: %w", topicArn, err)
	}

	return nil
}

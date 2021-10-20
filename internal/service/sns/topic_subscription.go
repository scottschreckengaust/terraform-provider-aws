package sns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceTopicSubscription() *schema.Resource {
	return &schema.Resource{
		Create: resourceTopicSubscriptionCreate,
		Read:   resourceTopicSubscriptionRead,
		Update: resourceTopicSubscriptionUpdate,
		Delete: resourceTopicSubscriptionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"confirmation_timeout_in_minutes": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},
			"confirmation_was_authenticated": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"delivery_policy": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: SuppressEquivalentTopicSubscriptionDeliveryPolicy,
			},
			"endpoint": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"endpoint_auto_confirms": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"filter_policy": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: verify.SuppressEquivalentJSONDiffs,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},
			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"pending_confirmation": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"protocol": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"application",
					"email-json",
					"email",
					"firehose",
					"http",
					"https",
					"lambda",
					"sms",
					"sqs",
				}, false),
			},
			"raw_message_delivery": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"redrive_policy": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: verify.SuppressEquivalentJSONDiffs,
			},
			"subscription_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: verify.ValidARN,
			},
			"topic_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidARN,
			},
		},
	}
}

func resourceTopicSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SNSConn

	input := &sns.SubscribeInput{
		Attributes:            expandSNSSubscriptionAttributes(d),
		Endpoint:              aws.String(d.Get("endpoint").(string)),
		Protocol:              aws.String(d.Get("protocol").(string)),
		ReturnSubscriptionArn: aws.Bool(true), // even if not confirmed, will get ARN
		TopicArn:              aws.String(d.Get("topic_arn").(string)),
	}

	output, err := conn.Subscribe(input)

	if err != nil {
		return fmt.Errorf("error creating SNS topic subscription: %w", err)
	}

	if output == nil || output.SubscriptionArn == nil || aws.StringValue(output.SubscriptionArn) == "" {
		return fmt.Errorf("error creating SNS topic subscription: empty response")
	}

	d.SetId(aws.StringValue(output.SubscriptionArn))

	waitForConfirmation := true

	if !d.Get("endpoint_auto_confirms").(bool) && strings.Contains(d.Get("protocol").(string), "http") {
		waitForConfirmation = false
	}

	if strings.Contains(d.Get("protocol").(string), "email") {
		waitForConfirmation = false
	}

	timeout := subscriptionPendingConfirmationTimeout
	if strings.Contains(d.Get("protocol").(string), "http") {
		timeout = time.Duration(d.Get("confirmation_timeout_in_minutes").(int)) * time.Minute
	}

	if waitForConfirmation {
		if _, err := waitSubscriptionConfirmed(conn, d.Id(), "false", timeout); err != nil {
			return fmt.Errorf("waiting for SNS topic subscription (%s) confirmation: %w", d.Id(), err)
		}
	}

	return resourceTopicSubscriptionRead(d, meta)
}

func resourceTopicSubscriptionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SNSConn

	var output *sns.GetSubscriptionAttributesOutput

	err := resource.Retry(subscriptionCreateTimeout, func() *resource.RetryError {
		var err error

		output, err = FindSubscriptionByARN(conn, d.Id())

		if err != nil {
			return resource.NonRetryableError(err)
		}

		if d.IsNewResource() && output == nil {
			return resource.RetryableError(&resource.NotFoundError{
				LastError: fmt.Errorf("SNS Topic Subscription Attributes (%s) not found", d.Id()),
			})
		}

		return nil
	})

	if tfresource.TimedOut(err) {
		output, err = FindSubscriptionByARN(conn, d.Id())
	}

	if err != nil {
		return fmt.Errorf("error getting SNS Topic Subscription Attributes (%s): %w", d.Id(), err)
	}

	if output == nil {
		if d.IsNewResource() {
			return fmt.Errorf("error getting SNS Topic Subscription Attributes (%s): not found after creation", d.Id())
		}

		log.Printf("[WARN] SNS Topic Subscription Attributes (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	attributes := output.Attributes

	d.Set("arn", attributes["SubscriptionArn"])
	d.Set("delivery_policy", attributes["DeliveryPolicy"])
	d.Set("endpoint", attributes["Endpoint"])
	d.Set("filter_policy", attributes["FilterPolicy"])
	d.Set("owner_id", attributes["Owner"])
	d.Set("protocol", attributes["Protocol"])
	d.Set("redrive_policy", attributes["RedrivePolicy"])
	d.Set("subscription_role_arn", attributes["SubscriptionRoleArn"])
	d.Set("topic_arn", attributes["TopicArn"])

	d.Set("confirmation_was_authenticated", false)
	if v, ok := attributes["ConfirmationWasAuthenticated"]; ok && aws.StringValue(v) == "true" {
		d.Set("confirmation_was_authenticated", true)
	}

	d.Set("pending_confirmation", false)
	if v, ok := attributes["PendingConfirmation"]; ok && aws.StringValue(v) == "true" {
		d.Set("pending_confirmation", true)
	}

	d.Set("raw_message_delivery", false)
	if v, ok := attributes["RawMessageDelivery"]; ok && aws.StringValue(v) == "true" {
		d.Set("raw_message_delivery", true)
	}

	return nil
}

func resourceTopicSubscriptionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SNSConn

	if d.HasChange("raw_message_delivery") {
		if err := snsSubscriptionAttributeUpdate(conn, d.Id(), "RawMessageDelivery", fmt.Sprintf("%t", d.Get("raw_message_delivery").(bool))); err != nil {
			return err
		}
	}

	if d.HasChange("filter_policy") {
		filterPolicy := d.Get("filter_policy").(string)

		// https://docs.aws.amazon.com/sns/latest/dg/message-filtering.html#message-filtering-policy-remove
		if filterPolicy == "" {
			filterPolicy = "{}"
		}

		if err := snsSubscriptionAttributeUpdate(conn, d.Id(), "FilterPolicy", filterPolicy); err != nil {
			return err
		}
	}

	if d.HasChange("delivery_policy") {
		if err := snsSubscriptionAttributeUpdate(conn, d.Id(), "DeliveryPolicy", d.Get("delivery_policy").(string)); err != nil {
			return err
		}
	}

	if d.HasChange("subscription_role_arn") {
		protocol := d.Get("protocol").(string)
		subscriptionRoleARN := d.Get("subscription_role_arn").(string)
		if strings.Contains(protocol, "firehose") && subscriptionRoleARN == "" {
			return fmt.Errorf("protocol firehose must contain subscription_role_arn!")
		}
		if !strings.Contains(protocol, "firehose") && subscriptionRoleARN != "" {
			return fmt.Errorf("only protocol firehose supports subscription_role_arn!")
		}

		if err := snsSubscriptionAttributeUpdate(conn, d.Id(), "SubscriptionRoleArn", subscriptionRoleARN); err != nil {
			return err
		}
	}

	if d.HasChange("redrive_policy") {
		if err := snsSubscriptionAttributeUpdate(conn, d.Id(), "RedrivePolicy", d.Get("redrive_policy").(string)); err != nil {
			return err
		}
	}

	return resourceTopicSubscriptionRead(d, meta)
}

func resourceTopicSubscriptionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SNSConn

	log.Printf("[DEBUG] SNS delete topic subscription: %s", d.Id())
	_, err := conn.Unsubscribe(&sns.UnsubscribeInput{
		SubscriptionArn: aws.String(d.Id()),
	})

	if tfawserr.ErrMessageContains(err, sns.ErrCodeInvalidParameterException, "Cannot unsubscribe a subscription that is pending confirmation") {
		log.Printf("[WARN] Removing unconfirmed SNS topic subscription (%s) from Terraform state but failed to remove it from AWS!", d.Id())
		d.SetId("")
		return nil
	}

	if _, err := waitSubscriptionDeleted(conn, d.Id()); err != nil {
		if tfawserr.ErrCodeEquals(err, sns.ErrCodeNotFoundException) {
			return nil
		}
		return fmt.Errorf("error waiting for SNS topic subscription (%s) deletion: %w", d.Id(), err)
	}

	return err
}

// Assembles supplied attributes into a single map - empty/default values are excluded from the map
func expandSNSSubscriptionAttributes(d *schema.ResourceData) (output map[string]*string) {
	deliveryPolicy := d.Get("delivery_policy").(string)
	filterPolicy := d.Get("filter_policy").(string)
	rawMessageDelivery := d.Get("raw_message_delivery").(bool)
	redrivePolicy := d.Get("redrive_policy").(string)
	subscriptionRoleARN := d.Get("subscription_role_arn").(string)

	// Collect attributes if available
	attributes := map[string]*string{}

	if deliveryPolicy != "" {
		attributes["DeliveryPolicy"] = aws.String(deliveryPolicy)
	}

	if filterPolicy != "" {
		attributes["FilterPolicy"] = aws.String(filterPolicy)
	}

	if rawMessageDelivery {
		attributes["RawMessageDelivery"] = aws.String(fmt.Sprintf("%t", rawMessageDelivery))
	}

	if subscriptionRoleARN != "" {
		attributes["SubscriptionRoleArn"] = aws.String(subscriptionRoleARN)
	}

	if redrivePolicy != "" {
		attributes["RedrivePolicy"] = aws.String(redrivePolicy)
	}

	return attributes
}

func snsSubscriptionAttributeUpdate(conn *sns.SNS, subscriptionArn, attributeName, attributeValue string) error {
	req := &sns.SetSubscriptionAttributesInput{
		SubscriptionArn: aws.String(subscriptionArn),
		AttributeName:   aws.String(attributeName),
		AttributeValue:  aws.String(attributeValue),
	}

	// The AWS API requires a non-empty string value or nil for the RedrivePolicy attribute,
	// else throws an InvalidParameter error
	if attributeName == "RedrivePolicy" && attributeValue == "" {
		req.AttributeValue = nil
	}

	_, err := conn.SetSubscriptionAttributes(req)

	if err != nil {
		return fmt.Errorf("error setting subscription (%s) attribute (%s): %s", subscriptionArn, attributeName, err)
	}
	return nil
}

type TopicSubscriptionDeliveryPolicy struct {
	Guaranteed         bool                                                 `json:"guaranteed,omitempty"`
	HealthyRetryPolicy *TopicSubscriptionDeliveryPolicyHealthyRetryPolicy   `json:"healthyRetryPolicy,omitempty"`
	SicklyRetryPolicy  *snsTopicSubscriptionDeliveryPolicySicklyRetryPolicy `json:"sicklyRetryPolicy,omitempty"`
	ThrottlePolicy     *snsTopicSubscriptionDeliveryPolicyThrottlePolicy    `json:"throttlePolicy,omitempty"`
}

func (s TopicSubscriptionDeliveryPolicy) String() string {
	return awsutil.Prettify(s)
}

func (s TopicSubscriptionDeliveryPolicy) GoString() string {
	return s.String()
}

type TopicSubscriptionDeliveryPolicyHealthyRetryPolicy struct {
	BackoffFunction    string `json:"backoffFunction,omitempty"`
	MaxDelayTarget     int    `json:"maxDelayTarget,omitempty"`
	MinDelayTarget     int    `json:"minDelayTarget,omitempty"`
	NumMaxDelayRetries int    `json:"numMaxDelayRetries,omitempty"`
	NumMinDelayRetries int    `json:"numMinDelayRetries,omitempty"`
	NumNoDelayRetries  int    `json:"numNoDelayRetries,omitempty"`
	NumRetries         int    `json:"numRetries,omitempty"`
}

func (s TopicSubscriptionDeliveryPolicyHealthyRetryPolicy) String() string {
	return awsutil.Prettify(s)
}

func (s TopicSubscriptionDeliveryPolicyHealthyRetryPolicy) GoString() string {
	return s.String()
}

type snsTopicSubscriptionDeliveryPolicySicklyRetryPolicy struct {
	BackoffFunction    string `json:"backoffFunction,omitempty"`
	MaxDelayTarget     int    `json:"maxDelayTarget,omitempty"`
	MinDelayTarget     int    `json:"minDelayTarget,omitempty"`
	NumMaxDelayRetries int    `json:"numMaxDelayRetries,omitempty"`
	NumMinDelayRetries int    `json:"numMinDelayRetries,omitempty"`
	NumNoDelayRetries  int    `json:"numNoDelayRetries,omitempty"`
	NumRetries         int    `json:"numRetries,omitempty"`
}

func (s snsTopicSubscriptionDeliveryPolicySicklyRetryPolicy) String() string {
	return awsutil.Prettify(s)
}

func (s snsTopicSubscriptionDeliveryPolicySicklyRetryPolicy) GoString() string {
	return s.String()
}

type snsTopicSubscriptionDeliveryPolicyThrottlePolicy struct {
	MaxReceivesPerSecond int `json:"maxReceivesPerSecond,omitempty"`
}

func (s snsTopicSubscriptionDeliveryPolicyThrottlePolicy) String() string {
	return awsutil.Prettify(s)
}

func (s snsTopicSubscriptionDeliveryPolicyThrottlePolicy) GoString() string {
	return s.String()
}

type TopicSubscriptionRedrivePolicy struct {
	DeadLetterTargetArn string `json:"deadLetterTargetArn,omitempty"`
}

func SuppressEquivalentTopicSubscriptionDeliveryPolicy(k, old, new string, d *schema.ResourceData) bool {
	ob, err := normalizeSnsTopicSubscriptionDeliveryPolicy(old)
	if err != nil {
		log.Print(err)
		return false
	}

	nb, err := normalizeSnsTopicSubscriptionDeliveryPolicy(new)
	if err != nil {
		log.Print(err)
		return false
	}

	return verify.JSONBytesEqual(ob, nb)
}

func normalizeSnsTopicSubscriptionDeliveryPolicy(policy string) ([]byte, error) {
	var deliveryPolicy TopicSubscriptionDeliveryPolicy

	if err := json.Unmarshal([]byte(policy), &deliveryPolicy); err != nil {
		return nil, fmt.Errorf("[WARN] Unable to unmarshal SNS Topic Subscription delivery policy JSON: %s", err)
	}

	normalizedDeliveryPolicy, err := json.Marshal(deliveryPolicy)

	if err != nil {
		return nil, fmt.Errorf("[WARN] Unable to marshal SNS Topic Subscription delivery policy back to JSON: %s", err)
	}

	b := bytes.NewBufferString("")
	if err := json.Compact(b, normalizedDeliveryPolicy); err != nil {
		return nil, fmt.Errorf("[WARN] Unable to marshal SNS Topic Subscription delivery policy back to JSON: %s", err)
	}

	return b.Bytes(), nil
}

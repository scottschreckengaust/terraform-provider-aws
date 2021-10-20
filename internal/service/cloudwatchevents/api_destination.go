package cloudwatchevents

import (
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	events "github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceAPIDestination() *schema.Resource {
	return &schema.Resource{
		Create: resourceAPIDestinationCreate,
		Read:   resourceAPIDestinationRead,
		Update: resourceAPIDestinationUpdate,
		Delete: resourceAPIDestinationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 64),
					validation.StringMatch(regexp.MustCompile(`^[\.\-_A-Za-z0-9]+`), ""),
				),
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 512),
			},
			"invocation_endpoint": {
				Type:     schema.TypeString,
				Required: true,
			},
			"invocation_rate_limit_per_second": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 300),
				Default:      300,
			},
			"http_method": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(events.ApiDestinationHttpMethod_Values(), true),
			},
			"connection_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: verify.ValidARN,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAPIDestinationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).CloudWatchEventsConn

	input := &events.CreateApiDestinationInput{}

	if name, ok := d.GetOk("name"); ok {
		input.Name = aws.String(name.(string))
	}
	if description, ok := d.GetOk("description"); ok {
		input.Description = aws.String(description.(string))
	}
	if invocationEndpoint, ok := d.GetOk("invocation_endpoint"); ok {
		input.InvocationEndpoint = aws.String(invocationEndpoint.(string))
	}
	if invocationRateLimitPerSecond, ok := d.GetOk("invocation_rate_limit_per_second"); ok {
		input.InvocationRateLimitPerSecond = aws.Int64(int64(invocationRateLimitPerSecond.(int)))
	}
	if httpMethod, ok := d.GetOk("http_method"); ok {
		input.HttpMethod = aws.String(httpMethod.(string))
	}
	if connectionArn, ok := d.GetOk("connection_arn"); ok {
		input.ConnectionArn = aws.String(connectionArn.(string))
	}

	log.Printf("[DEBUG] Creating CloudWatchEvent API Destination: %v", input)

	_, err := conn.CreateApiDestination(input)
	if err != nil {
		return fmt.Errorf("Creating CloudWatchEvent API Destination (%s) failed: %w", *input.Name, err)
	}

	d.SetId(aws.StringValue(input.Name))

	log.Printf("[INFO] CloudWatchEvent API Destination (%s) created", d.Id())

	return resourceAPIDestinationRead(d, meta)
}

func resourceAPIDestinationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).CloudWatchEventsConn

	input := &events.DescribeApiDestinationInput{
		Name: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading CloudWatchEvent API Destination (%s)", d.Id())
	output, err := conn.DescribeApiDestination(input)
	if tfawserr.ErrMessageContains(err, events.ErrCodeResourceNotFoundException, "") {
		log.Printf("[WARN] CloudWatchEvent API Destination (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error reading CloudWatchEvent API Destination: %w", err)
	}

	log.Printf("[DEBUG] Found CloudWatchEvent API Destination: %#v", *output)

	d.Set("arn", output.ApiDestinationArn)
	d.Set("name", output.Name)
	d.Set("description", output.Description)
	d.Set("invocation_endpoint", output.InvocationEndpoint)
	d.Set("invocation_rate_limit_per_second", output.InvocationRateLimitPerSecond)
	d.Set("http_method", output.HttpMethod)
	d.Set("connection_arn", output.ConnectionArn)

	return nil
}

func resourceAPIDestinationUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).CloudWatchEventsConn

	input := &events.UpdateApiDestinationInput{}

	if name, ok := d.GetOk("name"); ok {
		input.Name = aws.String(name.(string))
	}
	if description, ok := d.GetOk("description"); ok {
		input.Description = aws.String(description.(string))
	}
	if invocationEndpoint, ok := d.GetOk("invocation_endpoint"); ok {
		input.InvocationEndpoint = aws.String(invocationEndpoint.(string))
	}
	if invocationRateLimitPerSecond, ok := d.GetOk("invocation_rate_limit_per_second"); ok {
		input.InvocationRateLimitPerSecond = aws.Int64(int64(invocationRateLimitPerSecond.(int)))
	}
	if httpMethod, ok := d.GetOk("http_method"); ok {
		input.HttpMethod = aws.String(httpMethod.(string))
	}
	if connectionArn, ok := d.GetOk("connection_arn"); ok {
		input.ConnectionArn = aws.String(connectionArn.(string))
	}

	log.Printf("[DEBUG] Updating CloudWatchEvent API Destination: %s", input)
	_, err := conn.UpdateApiDestination(input)
	if err != nil {
		return fmt.Errorf("error updating CloudWatchEvent API Destination (%s): %w", d.Id(), err)
	}
	return resourceAPIDestinationRead(d, meta)
}

func resourceAPIDestinationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).CloudWatchEventsConn

	log.Printf("[INFO] Deleting CloudWatchEvent API Destination (%s)", d.Id())
	input := &events.DeleteApiDestinationInput{
		Name: aws.String(d.Id()),
	}

	_, err := conn.DeleteApiDestination(input)

	if tfawserr.ErrMessageContains(err, events.ErrCodeResourceNotFoundException, "") {
		log.Printf("[WARN] CloudWatchEvent API Destination (%s) not found", d.Id())
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error deleting CloudWatchEvent API Destination (%s): %w", d.Id(), err)
	}
	log.Printf("[INFO] CloudWatchEvent API Destination (%s) deleted", d.Id())

	return nil
}

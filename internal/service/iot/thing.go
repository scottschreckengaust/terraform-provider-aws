package iot

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
)

func ResourceThing() *schema.Resource {
	return &schema.Resource{
		Create: resourceThingCreate,
		Read:   resourceThingRead,
		Update: resourceThingUpdate,
		Delete: resourceThingDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 128),
			},
			"attributes": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"thing_type_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(1, 128),
			},
			"default_client_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceThingCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IoTConn

	params := &iot.CreateThingInput{
		ThingName: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("thing_type_name"); ok {
		params.ThingTypeName = aws.String(v.(string))
	}
	if v, ok := d.GetOk("attributes"); ok {
		params.AttributePayload = &iot.AttributePayload{
			Attributes: flex.ExpandStringMap(v.(map[string]interface{})),
		}
	}

	log.Printf("[DEBUG] Creating IoT Thing: %s", params)
	out, err := conn.CreateThing(params)
	if err != nil {
		return err
	}

	d.SetId(aws.StringValue(out.ThingName))

	return resourceThingRead(d, meta)
}

func resourceThingRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IoTConn

	params := &iot.DescribeThingInput{
		ThingName: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Reading IoT Thing: %s", params)
	out, err := conn.DescribeThing(params)

	if err != nil {
		if tfawserr.ErrMessageContains(err, iot.ErrCodeResourceNotFoundException, "") {
			log.Printf("[WARN] IoT Thing %q not found, removing from state", d.Id())
			d.SetId("")
		}
		return err
	}

	log.Printf("[DEBUG] Received IoT Thing: %s", out)

	d.Set("arn", out.ThingArn)
	d.Set("name", out.ThingName)
	d.Set("attributes", aws.StringValueMap(out.Attributes))
	d.Set("default_client_id", out.DefaultClientId)
	d.Set("thing_type_name", out.ThingTypeName)
	d.Set("version", out.Version)

	return nil
}

func resourceThingUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IoTConn

	params := &iot.UpdateThingInput{
		ThingName: aws.String(d.Get("name").(string)),
	}
	if d.HasChange("thing_type_name") {
		if v, ok := d.GetOk("thing_type_name"); ok {
			params.ThingTypeName = aws.String(v.(string))
		} else {
			params.RemoveThingType = aws.Bool(true)
		}
	}
	if d.HasChange("attributes") {
		attributes := map[string]*string{}

		if v, ok := d.GetOk("attributes"); ok {
			if m, ok := v.(map[string]interface{}); ok {
				attributes = flex.ExpandStringMap(m)
			}
		}
		params.AttributePayload = &iot.AttributePayload{
			Attributes: attributes,
		}
	}

	_, err := conn.UpdateThing(params)
	if err != nil {
		return err
	}

	return resourceThingRead(d, meta)
}

func resourceThingDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IoTConn

	params := &iot.DeleteThingInput{
		ThingName: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Deleting IoT Thing: %s", params)

	_, err := conn.DeleteThing(params)
	if err != nil {
		if tfawserr.ErrMessageContains(err, iot.ErrCodeResourceNotFoundException, "") {
			return nil
		}
		return err
	}

	return nil
}

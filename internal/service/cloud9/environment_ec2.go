package cloud9

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloud9"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfiam "github.com/hashicorp/terraform-provider-aws/internal/service/iam"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceEnvironmentEC2() *schema.Resource {
	return &schema.Resource{
		Create: resourceEnvironmentEC2Create,
		Read:   resourceEnvironmentEC2Read,
		Update: resourceEnvironmentEC2Update,
		Delete: resourceEnvironmentEC2Delete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"instance_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"automatic_stop_time_minutes": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"owner_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidARN,
			},
			"subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),
		},

		CustomizeDiff: verify.SetTagsDiff,
	}
}

func resourceEnvironmentEC2Create(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).Cloud9Conn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(d.Get("tags").(map[string]interface{})))

	params := &cloud9.CreateEnvironmentEC2Input{
		InstanceType:       aws.String(d.Get("instance_type").(string)),
		Name:               aws.String(d.Get("name").(string)),
		ClientRequestToken: aws.String(resource.UniqueId()),
		Tags:               Tags(tags.IgnoreAWS()),
	}

	if v, ok := d.GetOk("automatic_stop_time_minutes"); ok {
		params.AutomaticStopTimeMinutes = aws.Int64(int64(v.(int)))
	}
	if v, ok := d.GetOk("description"); ok {
		params.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("owner_arn"); ok {
		params.OwnerArn = aws.String(v.(string))
	}
	if v, ok := d.GetOk("subnet_id"); ok {
		params.SubnetId = aws.String(v.(string))
	}

	var out *cloud9.CreateEnvironmentEC2Output
	err := resource.Retry(tfiam.PropagationTimeout, func() *resource.RetryError {
		var err error
		out, err = conn.CreateEnvironmentEC2(params)
		if err != nil {
			// NotFoundException: User arn:aws:iam::*******:user/****** does not exist.
			if tfawserr.ErrMessageContains(err, cloud9.ErrCodeNotFoundException, "User") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if tfresource.TimedOut(err) {
		out, err = conn.CreateEnvironmentEC2(params)
	}

	if err != nil {
		return fmt.Errorf("Error creating Cloud9 EC2 Environment: %s", err)
	}
	d.SetId(aws.StringValue(out.EnvironmentId))

	stateConf := resource.StateChangeConf{
		Pending: []string{
			cloud9.EnvironmentStatusConnecting,
			cloud9.EnvironmentStatusCreating,
		},
		Target: []string{
			cloud9.EnvironmentStatusReady,
		},
		Timeout: 10 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeEnvironmentStatus(&cloud9.DescribeEnvironmentStatusInput{
				EnvironmentId: aws.String(d.Id()),
			})
			if err != nil {
				return 42, "", err
			}

			status := aws.StringValue(out.Status)

			if status == cloud9.EnvironmentStatusError && out.Message != nil {
				return out, status, fmt.Errorf("Reason: %s", aws.StringValue(out.Message))
			}

			return out, status, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceEnvironmentEC2Read(d, meta)
}

func resourceEnvironmentEC2Read(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).Cloud9Conn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	log.Printf("[INFO] Reading Cloud9 Environment EC2 %s", d.Id())

	out, err := conn.DescribeEnvironments(&cloud9.DescribeEnvironmentsInput{
		EnvironmentIds: []*string{aws.String(d.Id())},
	})
	if err != nil {
		if tfawserr.ErrMessageContains(err, cloud9.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] Cloud9 Environment EC2 (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	if len(out.Environments) == 0 {
		log.Printf("[WARN] Cloud9 Environment EC2 (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	env := out.Environments[0]

	arn := aws.StringValue(env.Arn)
	d.Set("arn", arn)
	d.Set("description", env.Description)
	d.Set("name", env.Name)
	d.Set("owner_arn", env.OwnerArn)
	d.Set("type", env.Type)

	tags, err := ListTags(conn, arn)

	if err != nil {
		return fmt.Errorf("error listing tags for Cloud9 EC2 Environment (%s): %s", arn, err)
	}

	tags = tags.IgnoreAWS().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	log.Printf("[DEBUG] Received Cloud9 Environment EC2: %s", env)

	return nil
}

func resourceEnvironmentEC2Update(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).Cloud9Conn

	input := cloud9.UpdateEnvironmentInput{
		Description:   aws.String(d.Get("description").(string)),
		EnvironmentId: aws.String(d.Id()),
		Name:          aws.String(d.Get("name").(string)),
	}

	log.Printf("[INFO] Updating Cloud9 Environment EC2: %s", input)

	out, err := conn.UpdateEnvironment(&input)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Cloud9 Environment EC2 updated: %s", out)

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")
		arn := d.Get("arn").(string)

		if err := UpdateTags(conn, arn, o, n); err != nil {
			return fmt.Errorf("error updating Cloud9 EC2 Environment (%s) tags: %s", arn, err)
		}
	}

	return resourceEnvironmentEC2Read(d, meta)
}

func resourceEnvironmentEC2Delete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).Cloud9Conn

	_, err := conn.DeleteEnvironment(&cloud9.DeleteEnvironmentInput{
		EnvironmentId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	input := &cloud9.DescribeEnvironmentsInput{
		EnvironmentIds: []*string{aws.String(d.Id())},
	}
	var out *cloud9.DescribeEnvironmentsOutput
	err = resource.Retry(20*time.Minute, func() *resource.RetryError { // Deleting instances can take a long time
		out, err = conn.DescribeEnvironments(input)
		if err != nil {
			if tfawserr.ErrMessageContains(err, cloud9.ErrCodeNotFoundException, "") {
				return nil
			}
			// :'-(
			if tfawserr.ErrMessageContains(err, "AccessDeniedException", "is not authorized to access this resource") {
				return nil
			}
			return resource.NonRetryableError(err)
		}
		if len(out.Environments) == 0 {
			return nil
		}
		return resource.RetryableError(fmt.Errorf("Cloud9 EC2 Environment %q still exists", d.Id()))
	})
	if tfresource.TimedOut(err) {
		out, err = conn.DescribeEnvironments(input)
		if tfawserr.ErrMessageContains(err, cloud9.ErrCodeNotFoundException, "") {
			return nil
		}
		if tfawserr.ErrMessageContains(err, "AccessDeniedException", "is not authorized to access this resource") {
			return nil
		}
	}
	if err != nil {
		return fmt.Errorf("Error deleting Cloud9 EC2 Environment: %s", err)
	}
	return nil
}

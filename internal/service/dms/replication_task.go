package dms

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	dms "github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceReplicationTask() *schema.Resource {
	return &schema.Resource{
		Create: resourceReplicationTaskCreate,
		Read:   resourceReplicationTaskRead,
		Update: resourceReplicationTaskUpdate,
		Delete: resourceReplicationTaskDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"cdc_start_position": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"cdc_start_time"},
			},
			"cdc_start_time": {
				Type:     schema.TypeString,
				Optional: true,
				// Requires a Unix timestamp in seconds. Example 1484346880
				ConflictsWith: []string{"cdc_start_position"},
			},
			"migration_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					dms.MigrationTypeValueFullLoad,
					dms.MigrationTypeValueCdc,
					dms.MigrationTypeValueFullLoadAndCdc,
				}, false),
			},
			"replication_instance_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidARN,
			},
			"replication_task_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"replication_task_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validReplicationTaskID,
			},
			"replication_task_settings": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: verify.SuppressEquivalentJSONDiffs,
			},
			"source_endpoint_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidARN,
			},
			"table_mappings": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: verify.SuppressEquivalentJSONDiffs,
			},
			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),
			"target_endpoint_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidARN,
			},
		},

		CustomizeDiff: verify.SetTagsDiff,
	}
}

func resourceReplicationTaskCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).DMSConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(d.Get("tags").(map[string]interface{})))

	request := &dms.CreateReplicationTaskInput{
		MigrationType:             aws.String(d.Get("migration_type").(string)),
		ReplicationInstanceArn:    aws.String(d.Get("replication_instance_arn").(string)),
		ReplicationTaskIdentifier: aws.String(d.Get("replication_task_id").(string)),
		SourceEndpointArn:         aws.String(d.Get("source_endpoint_arn").(string)),
		TableMappings:             aws.String(d.Get("table_mappings").(string)),
		Tags:                      Tags(tags.IgnoreAWS()),
		TargetEndpointArn:         aws.String(d.Get("target_endpoint_arn").(string)),
	}

	if v, ok := d.GetOk("cdc_start_position"); ok {
		request.CdcStartPosition = aws.String(v.(string))
	}

	if v, ok := d.GetOk("cdc_start_time"); ok {
		seconds, err := strconv.ParseInt(v.(string), 10, 64)
		if err != nil {
			return fmt.Errorf("DMS create replication task. Invalid CDC Unix timestamp: %s", err)
		}
		request.CdcStartTime = aws.Time(time.Unix(seconds, 0))
	}

	if v, ok := d.GetOk("replication_task_settings"); ok {
		request.ReplicationTaskSettings = aws.String(v.(string))
	}

	log.Println("[DEBUG] DMS create replication task:", request)

	_, err := conn.CreateReplicationTask(request)
	if err != nil {
		return err
	}

	taskId := d.Get("replication_task_id").(string)
	d.SetId(taskId)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating"},
		Target:     []string{"ready"},
		Refresh:    resourceReplicationTaskStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceReplicationTaskRead(d, meta)
}

func resourceReplicationTaskRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).DMSConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	response, err := conn.DescribeReplicationTasks(&dms.DescribeReplicationTasksInput{
		Filters: []*dms.Filter{
			{
				Name:   aws.String("replication-task-id"),
				Values: []*string{aws.String(d.Id())}, // Must use d.Id() to work with import.
			},
		},
	})
	if err != nil {
		if dmserr, ok := err.(awserr.Error); ok && dmserr.Code() == "ResourceNotFoundFault" {
			log.Printf("[DEBUG] DMS Replication Task %q Not Found", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	err = resourceReplicationTaskSetState(d, response.ReplicationTasks[0])
	if err != nil {
		return err
	}

	tags, err := ListTags(conn, d.Get("replication_task_arn").(string))

	if err != nil {
		return fmt.Errorf("error listing tags for DMS Replication Task (%s): %s", d.Get("replication_task_arn").(string), err)
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

func resourceReplicationTaskUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).DMSConn

	request := &dms.ModifyReplicationTaskInput{
		ReplicationTaskArn: aws.String(d.Get("replication_task_arn").(string)),
	}
	hasChanges := false

	if d.HasChange("cdc_start_position") {
		request.CdcStartPosition = aws.String(d.Get("cdc_start_position").(string))
		hasChanges = true
	}

	if d.HasChange("cdc_start_time") {
		seconds, err := strconv.ParseInt(d.Get("cdc_start_time").(string), 10, 64)
		if err != nil {
			return fmt.Errorf("DMS update replication task. Invalid CRC Unix timestamp: %s", err)
		}
		request.CdcStartTime = aws.Time(time.Unix(seconds, 0))
		hasChanges = true
	}

	if d.HasChange("migration_type") {
		request.MigrationType = aws.String(d.Get("migration_type").(string))
		hasChanges = true
	}

	if d.HasChange("replication_task_settings") {
		request.ReplicationTaskSettings = aws.String(d.Get("replication_task_settings").(string))
		hasChanges = true
	}

	if d.HasChange("table_mappings") {
		request.TableMappings = aws.String(d.Get("table_mappings").(string))
		hasChanges = true
	}

	if d.HasChange("tags_all") {
		arn := d.Get("replication_task_arn").(string)
		o, n := d.GetChange("tags_all")

		if err := UpdateTags(conn, arn, o, n); err != nil {
			return fmt.Errorf("error updating DMS Replication Task (%s) tags: %s", arn, err)
		}
	}

	if hasChanges {
		log.Println("[DEBUG] DMS update replication task:", request)

		_, err := conn.ModifyReplicationTask(request)
		if err != nil {
			return err
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"modifying"},
			Target:     []string{"ready", "stopped", "failed"},
			Refresh:    resourceReplicationTaskStateRefreshFunc(d, meta),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			MinTimeout: 10 * time.Second,
			Delay:      30 * time.Second, // Wait 30 secs before starting
		}

		// Wait, catching any errors
		_, err = stateConf.WaitForState()
		if err != nil {
			return err
		}

		return resourceReplicationTaskRead(d, meta)
	}

	return nil
}

func resourceReplicationTaskDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).DMSConn

	request := &dms.DeleteReplicationTaskInput{
		ReplicationTaskArn: aws.String(d.Get("replication_task_arn").(string)),
	}

	log.Printf("[DEBUG] DMS delete replication task: %#v", request)

	_, err := conn.DeleteReplicationTask(request)
	if err != nil {
		if dmserr, ok := err.(awserr.Error); ok && dmserr.Code() == "ResourceNotFoundFault" {
			log.Printf("[DEBUG] DMS Replication Task %q Not Found", d.Id())
			return nil
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting"},
		Target:     []string{},
		Refresh:    resourceReplicationTaskStateRefreshFunc(d, meta),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()

	return err
}

func resourceReplicationTaskSetState(d *schema.ResourceData, task *dms.ReplicationTask) error {
	d.SetId(aws.StringValue(task.ReplicationTaskIdentifier))

	d.Set("cdc_start_position", task.CdcStartPosition)
	d.Set("migration_type", task.MigrationType)
	d.Set("replication_instance_arn", task.ReplicationInstanceArn)
	d.Set("replication_task_arn", task.ReplicationTaskArn)
	d.Set("replication_task_id", task.ReplicationTaskIdentifier)
	d.Set("source_endpoint_arn", task.SourceEndpointArn)
	d.Set("table_mappings", task.TableMappings)
	d.Set("target_endpoint_arn", task.TargetEndpointArn)

	settings, err := dmsReplicationTaskRemoveReadOnlySettings(*task.ReplicationTaskSettings)
	if err != nil {
		return err
	}
	d.Set("replication_task_settings", settings)

	return nil
}

func resourceReplicationTaskStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*conns.AWSClient).DMSConn

		v, err := conn.DescribeReplicationTasks(&dms.DescribeReplicationTasksInput{
			Filters: []*dms.Filter{
				{
					Name:   aws.String("replication-task-id"),
					Values: []*string{aws.String(d.Id())}, // Must use d.Id() to work with import.
				},
			},
		})
		if err != nil {
			if dmserr, ok := err.(awserr.Error); ok && dmserr.Code() == "ResourceNotFoundFault" {
				return nil, "", nil
			}
			log.Printf("Error on retrieving DMS Replication Task when waiting: %s", err)
			return nil, "", err
		}

		if v == nil {
			return nil, "", nil
		}

		if v.ReplicationTasks != nil {
			log.Printf("[DEBUG] DMS Replication Task status for instance %s: %s", d.Id(), *v.ReplicationTasks[0].Status)
		}

		return v, *v.ReplicationTasks[0].Status, nil
	}
}

func dmsReplicationTaskRemoveReadOnlySettings(settings string) (*string, error) {
	var settingsData map[string]interface{}
	if err := json.Unmarshal([]byte(settings), &settingsData); err != nil {
		return nil, err
	}

	controlTablesSettings, ok := settingsData["ControlTablesSettings"].(map[string]interface{})
	if ok {
		delete(controlTablesSettings, "historyTimeslotInMinutes")
	}

	logging, ok := settingsData["Logging"].(map[string]interface{})
	if ok {
		delete(logging, "CloudWatchLogGroup")
		delete(logging, "CloudWatchLogStream")
	}

	cleanedSettings, err := json.Marshal(settingsData)
	if err != nil {
		return nil, err
	}

	cleanedSettingsString := string(cleanedSettings)
	return &cleanedSettingsString, nil
}

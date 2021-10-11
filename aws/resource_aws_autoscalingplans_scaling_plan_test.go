package aws

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscalingplans"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/autoscalingplans/finder"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/autoscalingplans/lister"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

func init() {
	resource.AddTestSweepers("aws_autoscalingplans_scaling_plan", &resource.Sweeper{
		Name: "aws_autoscalingplans_scaling_plan",
		F:    testSweepAutoScalingPlansScalingPlans,
	})
}

func testSweepAutoScalingPlansScalingPlans(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*AWSClient).autoscalingplansconn
	input := &autoscalingplans.DescribeScalingPlansInput{}
	sweepResources := make([]*testSweepResource, 0)

	err = lister.DescribeScalingPlansPages(conn, input, func(page *autoscalingplans.DescribeScalingPlansOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, scalingPlan := range page.ScalingPlans {
			scalingPlanName := aws.StringValue(scalingPlan.ScalingPlanName)
			scalingPlanVersion := int(aws.Int64Value(scalingPlan.ScalingPlanVersion))

			r := resourceAwsAutoScalingPlansScalingPlan()
			d := r.Data(nil)
			d.SetId("????????????????") // ID not used in Delete.
			d.Set("name", scalingPlanName)
			d.Set("scaling_plan_version", scalingPlanVersion)

			sweepResources = append(sweepResources, NewTestSweepResource(r, d, client))
		}

		return !lastPage
	})

	if testSweepSkipSweepError(err) {
		log.Printf("[WARN] Skipping Auto Scaling Scaling Plan sweep for %s: %s", region, err)
		return nil
	}

	if err != nil {
		return fmt.Errorf("error listing Auto Scaling Scaling Plans (%s): %w", region, err)
	}

	err = testSweepResourceOrchestrator(sweepResources)

	if err != nil {
		return fmt.Errorf("error sweeping Auto Scaling Scaling Plans (%s): %w", region, err)
	}

	return nil
}

func TestAccAwsAutoScalingPlansScalingPlan_basicDynamicScaling(t *testing.T) {
	var scalingPlan autoscalingplans.ScalingPlan
	resourceName := "aws_autoscalingplans_scaling_plan.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, autoscalingplans.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAutoScalingPlansScalingPlanDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAutoScalingPlansScalingPlanConfigBasicDynamicScaling(rName, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAutoScalingPlansScalingPlanExists(resourceName, &scalingPlan),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "scaling_plan_version", "1"),
					resource.TestCheckResourceAttr(resourceName, "application_source.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "application_source.0.cloudformation_stack_arn", ""),
					testAccCheckAutoScalingPlansApplicationSourceTags(&scalingPlan, map[string][]string{
						rName: {rName},
					}),
					resource.TestCheckResourceAttr(resourceName, "scaling_instruction.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "scaling_instruction.*", map[string]string{
						"customized_load_metric_specification.#": "0",
						"disable_dynamic_scaling":                "false",
						"max_capacity":                           "3",
						"min_capacity":                           "0",
						"predefined_load_metric_specification.#": "0",
						"resource_id":                            fmt.Sprintf("autoScalingGroup/%s", rName),
						"scalable_dimension":                     "autoscaling:autoScalingGroup:DesiredCapacity",
						"scaling_policy_update_behavior":         "KeepExternalPolicies",
						"service_namespace":                      "autoscaling",
						"target_tracking_configuration.#":        "1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "scaling_instruction.*.target_tracking_configuration.*", map[string]string{
						"customized_scaling_metric_specification.#":                                "0",
						"disable_scale_in":                                                         "false",
						"predefined_scaling_metric_specification.#":                                "1",
						"predefined_scaling_metric_specification.0.predefined_scaling_metric_type": "ASGAverageCPUUtilization",
						"target_value": "75",
					}),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return rName, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsAutoScalingPlansScalingPlan_basicPredictiveScaling(t *testing.T) {
	var scalingPlan autoscalingplans.ScalingPlan
	resourceName := "aws_autoscalingplans_scaling_plan.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIamServiceLinkedRole(t, "/aws-service-role/autoscaling-plans")
		},
		ErrorCheck:   testAccErrorCheck(t, autoscalingplans.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAutoScalingPlansScalingPlanDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAutoScalingPlansScalingPlanConfigBasicPredictiveScaling(rName, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAutoScalingPlansScalingPlanExists(resourceName, &scalingPlan),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "scaling_plan_version", "1"),
					resource.TestCheckResourceAttr(resourceName, "application_source.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "application_source.0.cloudformation_stack_arn", ""),
					testAccCheckAutoScalingPlansApplicationSourceTags(&scalingPlan, map[string][]string{
						rName: {rName},
					}),
					resource.TestCheckResourceAttr(resourceName, "scaling_instruction.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "scaling_instruction.*", map[string]string{
						"customized_load_metric_specification.#": "0",
						"disable_dynamic_scaling":                "true",
						"max_capacity":                           "3",
						"min_capacity":                           "0",
						"predefined_load_metric_specification.#": "1",
						"predefined_load_metric_specification.0.predefined_load_metric_type": "ASGTotalCPUUtilization",
						"predictive_scaling_max_capacity_behavior":                           "SetForecastCapacityToMaxCapacity",
						"predictive_scaling_mode":                                            "ForecastOnly",
						"resource_id":                                                        fmt.Sprintf("autoScalingGroup/%s", rName),
						"scalable_dimension":                                                 "autoscaling:autoScalingGroup:DesiredCapacity",
						"scaling_policy_update_behavior":                                     "KeepExternalPolicies",
						"service_namespace":                                                  "autoscaling",
						"target_tracking_configuration.#":                                    "1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "scaling_instruction.*.target_tracking_configuration.*", map[string]string{
						"customized_scaling_metric_specification.#":                                "0",
						"disable_scale_in":                                                         "false",
						"predefined_scaling_metric_specification.#":                                "1",
						"predefined_scaling_metric_specification.0.predefined_scaling_metric_type": "ASGAverageCPUUtilization",
						"target_value": "75",
					}),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return rName, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsAutoScalingPlansScalingPlan_basicUpdate(t *testing.T) {
	var scalingPlan autoscalingplans.ScalingPlan
	resourceName := "aws_autoscalingplans_scaling_plan.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")
	rNameUpdated := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIamServiceLinkedRole(t, "/aws-service-role/autoscaling-plans")
		},
		ErrorCheck:   testAccErrorCheck(t, autoscalingplans.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAutoScalingPlansScalingPlanDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAutoScalingPlansScalingPlanConfigBasicDynamicScaling(rName, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAutoScalingPlansScalingPlanExists(resourceName, &scalingPlan),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "scaling_plan_version", "1"),
					resource.TestCheckResourceAttr(resourceName, "application_source.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "application_source.0.cloudformation_stack_arn", ""),
					testAccCheckAutoScalingPlansApplicationSourceTags(&scalingPlan, map[string][]string{
						rName: {rName},
					}),
					resource.TestCheckResourceAttr(resourceName, "scaling_instruction.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "scaling_instruction.*", map[string]string{
						"customized_load_metric_specification.#": "0",
						"disable_dynamic_scaling":                "false",
						"max_capacity":                           "3",
						"min_capacity":                           "0",
						"predefined_load_metric_specification.#": "0",
						"resource_id":                            fmt.Sprintf("autoScalingGroup/%s", rName),
						"scalable_dimension":                     "autoscaling:autoScalingGroup:DesiredCapacity",
						"scaling_policy_update_behavior":         "KeepExternalPolicies",
						"service_namespace":                      "autoscaling",
						"target_tracking_configuration.#":        "1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "scaling_instruction.*.target_tracking_configuration.*", map[string]string{
						"customized_scaling_metric_specification.#":                                "0",
						"disable_scale_in":                                                         "false",
						"predefined_scaling_metric_specification.#":                                "1",
						"predefined_scaling_metric_specification.0.predefined_scaling_metric_type": "ASGAverageCPUUtilization",
						"target_value": "75",
					}),
				),
			},
			{
				Config: testAccAutoScalingPlansScalingPlanConfigBasicPredictiveScaling(rName, rNameUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAutoScalingPlansScalingPlanExists(resourceName, &scalingPlan),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "scaling_plan_version", "1"),
					resource.TestCheckResourceAttr(resourceName, "application_source.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "application_source.0.cloudformation_stack_arn", ""),
					testAccCheckAutoScalingPlansApplicationSourceTags(&scalingPlan, map[string][]string{
						rNameUpdated: {rNameUpdated},
					}),
					resource.TestCheckResourceAttr(resourceName, "scaling_instruction.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "scaling_instruction.*", map[string]string{
						"customized_load_metric_specification.#": "0",
						"disable_dynamic_scaling":                "true",
						"max_capacity":                           "3",
						"min_capacity":                           "0",
						"predefined_load_metric_specification.#": "1",
						"predefined_load_metric_specification.0.predefined_load_metric_type": "ASGTotalCPUUtilization",
						"predictive_scaling_max_capacity_behavior":                           "SetForecastCapacityToMaxCapacity",
						"predictive_scaling_mode":                                            "ForecastOnly",
						"resource_id":                                                        fmt.Sprintf("autoScalingGroup/%s", rName),
						"scalable_dimension":                                                 "autoscaling:autoScalingGroup:DesiredCapacity",
						"scaling_policy_update_behavior":                                     "KeepExternalPolicies",
						"service_namespace":                                                  "autoscaling",
						"target_tracking_configuration.#":                                    "1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "scaling_instruction.*.target_tracking_configuration.*", map[string]string{
						"customized_scaling_metric_specification.#":                                "0",
						"disable_scale_in":                                                         "false",
						"predefined_scaling_metric_specification.#":                                "1",
						"predefined_scaling_metric_specification.0.predefined_scaling_metric_type": "ASGAverageCPUUtilization",
						"target_value": "75",
					}),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return rName, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAwsAutoScalingPlansScalingPlan_disappears(t *testing.T) {
	var scalingPlan autoscalingplans.ScalingPlan
	resourceName := "aws_autoscalingplans_scaling_plan.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, autoscalingplans.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAutoScalingPlansScalingPlanDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAutoScalingPlansScalingPlanConfigBasicDynamicScaling(rName, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAutoScalingPlansScalingPlanExists(resourceName, &scalingPlan),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsAutoScalingPlansScalingPlan(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAwsAutoScalingPlansScalingPlan_dynamicScaling_CustomizedScalingMetricSpecification(t *testing.T) {
	var scalingPlan autoscalingplans.ScalingPlan
	resourceName := "aws_autoscalingplans_scaling_plan.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, autoscalingplans.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAutoScalingPlansScalingPlanDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAutoScalingPlansScalingPlanConfigDynamicScalingCustomizedScalingMetricSpecification(rName, rName, 90),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAutoScalingPlansScalingPlanExists(resourceName, &scalingPlan),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "scaling_plan_version", "1"),
					resource.TestCheckResourceAttr(resourceName, "application_source.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "application_source.0.cloudformation_stack_arn", ""),
					testAccCheckAutoScalingPlansApplicationSourceTags(&scalingPlan, map[string][]string{
						rName: {rName},
					}),
					resource.TestCheckResourceAttr(resourceName, "scaling_instruction.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "scaling_instruction.*", map[string]string{
						"customized_load_metric_specification.#": "0",
						"disable_dynamic_scaling":                "false",
						"max_capacity":                           "3",
						"min_capacity":                           "0",
						"predefined_load_metric_specification.#": "0",
						"resource_id":                            fmt.Sprintf("autoScalingGroup/%s", rName),
						"scalable_dimension":                     "autoscaling:autoScalingGroup:DesiredCapacity",
						"scaling_policy_update_behavior":         "KeepExternalPolicies",
						"service_namespace":                      "autoscaling",
						"target_tracking_configuration.#":        "1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "scaling_instruction.*.target_tracking_configuration.*", map[string]string{
						"customized_scaling_metric_specification.#":                                 "1",
						"customized_scaling_metric_specification.0.dimensions.%":                    "2",
						"customized_scaling_metric_specification.0.dimensions.AutoScalingGroupName": rName,
						"customized_scaling_metric_specification.0.dimensions.objectname":           "Memory",
						"customized_scaling_metric_specification.0.metric_name":                     "Memory % Committed Bytes In Use",
						"customized_scaling_metric_specification.0.namespace":                       "test",
						"customized_scaling_metric_specification.0.statistic":                       "Average",
						"customized_scaling_metric_specification.0.unit":                            "",
						"disable_scale_in":                          "false",
						"predefined_scaling_metric_specification.#": "0",
						"target_value":                              "90",
					}),
				),
			},
			{
				Config: testAccAutoScalingPlansScalingPlanConfigDynamicScalingCustomizedScalingMetricSpecification(rName, rName, 75),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAutoScalingPlansScalingPlanExists(resourceName, &scalingPlan),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "scaling_plan_version", "1"),
					resource.TestCheckResourceAttr(resourceName, "application_source.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "application_source.0.cloudformation_stack_arn", ""),
					testAccCheckAutoScalingPlansApplicationSourceTags(&scalingPlan, map[string][]string{
						rName: {rName},
					}),
					resource.TestCheckResourceAttr(resourceName, "scaling_instruction.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "scaling_instruction.*", map[string]string{
						"customized_load_metric_specification.#": "0",
						"disable_dynamic_scaling":                "false",
						"max_capacity":                           "3",
						"min_capacity":                           "0",
						"predefined_load_metric_specification.#": "0",
						"resource_id":                            fmt.Sprintf("autoScalingGroup/%s", rName),
						"scalable_dimension":                     "autoscaling:autoScalingGroup:DesiredCapacity",
						"scaling_policy_update_behavior":         "KeepExternalPolicies",
						"service_namespace":                      "autoscaling",
						"target_tracking_configuration.#":        "1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "scaling_instruction.*.target_tracking_configuration.*", map[string]string{
						"customized_scaling_metric_specification.#":                                 "1",
						"customized_scaling_metric_specification.0.dimensions.%":                    "2",
						"customized_scaling_metric_specification.0.dimensions.AutoScalingGroupName": rName,
						"customized_scaling_metric_specification.0.dimensions.objectname":           "Memory",
						"customized_scaling_metric_specification.0.metric_name":                     "Memory % Committed Bytes In Use",
						"customized_scaling_metric_specification.0.namespace":                       "test",
						"customized_scaling_metric_specification.0.statistic":                       "Average",
						"customized_scaling_metric_specification.0.unit":                            "",
						"disable_scale_in":                          "false",
						"predefined_scaling_metric_specification.#": "0",
						"target_value":                              "75",
					}),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return rName, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckAutoScalingPlansScalingPlanDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).autoscalingplansconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_autoscalingplans_scaling_plan" {
			continue
		}

		scalingPlanVersion, err := strconv.Atoi(rs.Primary.Attributes["scaling_plan_version"])

		if err != nil {
			return err
		}

		_, err = finder.ScalingPlanByNameAndVersion(conn, rs.Primary.Attributes["name"], scalingPlanVersion)

		if tfresource.NotFound(err) {
			continue
		}

		return fmt.Errorf("Auto Scaling Scaling Plan %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccCheckAutoScalingPlansScalingPlanExists(name string, v *autoscalingplans.ScalingPlan) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).autoscalingplansconn

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Auto Scaling Scaling Plan ID is set")
		}

		scalingPlanVersion, err := strconv.Atoi(rs.Primary.Attributes["scaling_plan_version"])

		if err != nil {
			return err
		}

		scalingPlan, err := finder.ScalingPlanByNameAndVersion(conn, rs.Primary.Attributes["name"], scalingPlanVersion)

		if err != nil {
			return err
		}

		*v = *scalingPlan

		return nil
	}
}

func testAccCheckAutoScalingPlansApplicationSourceTags(scalingPlan *autoscalingplans.ScalingPlan, expectedTagFilters map[string][]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, tagFilter := range scalingPlan.ApplicationSource.TagFilters {
			key := aws.StringValue(tagFilter.Key)
			values := aws.StringValueSlice(tagFilter.Values)

			expectedValues, ok := expectedTagFilters[key]
			if !ok {
				return fmt.Errorf("Scaling plan application source tag filter key %q not expected", key)
			}

			sort.Strings(values)
			sort.Strings(expectedValues)
			if !reflect.DeepEqual(values, expectedValues) {
				return fmt.Errorf("Scaling plan application source tag filter values %q, expected %q", values, expectedValues)
			}
		}

		return nil
	}
}

func testAccAutoScalingPlansScalingPlanConfigBase(rName, tagName string) string {
	return composeConfig(
		testAccLatestAmazonLinuxHvmEbsAmiConfig(),
		testAccAvailableAZsNoOptInDefaultExcludeConfig(),
		testAccAvailableEc2InstanceTypeForRegion("t3.micro", "t2.micro"),
		fmt.Sprintf(`
resource "aws_launch_configuration" "test" {
  image_id      = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type = data.aws_ec2_instance_type_offering.available.instance_type
}

resource "aws_autoscaling_group" "test" {
  name = %[1]q

  launch_configuration = aws_launch_configuration.test.name
  availability_zones   = [data.aws_availability_zones.available.names[0]]

  min_size         = 0
  max_size         = 3
  desired_capacity = 0

  tags = [
    {
      key                 = %[2]q
      value               = %[2]q
      propagate_at_launch = true
    },
  ]
}
`, rName, tagName))
}

func testAccAutoScalingPlansScalingPlanConfigBasicDynamicScaling(rName, tagName string) string {
	return composeConfig(
		testAccAutoScalingPlansScalingPlanConfigBase(rName, tagName),
		fmt.Sprintf(`
resource "aws_autoscalingplans_scaling_plan" "test" {
  name = %[1]q

  application_source {
    tag_filter {
      key    = %[2]q
      values = [%[2]q]
    }
  }

  scaling_instruction {
    max_capacity       = aws_autoscaling_group.test.max_size
    min_capacity       = aws_autoscaling_group.test.min_size
    resource_id        = format("autoScalingGroup/%%s", aws_autoscaling_group.test.name)
    scalable_dimension = "autoscaling:autoScalingGroup:DesiredCapacity"
    service_namespace  = "autoscaling"

    target_tracking_configuration {
      predefined_scaling_metric_specification {
        predefined_scaling_metric_type = "ASGAverageCPUUtilization"
      }

      target_value = 75
    }
  }
}
`, rName, tagName))
}

func testAccAutoScalingPlansScalingPlanConfigBasicPredictiveScaling(rName, tagName string) string {
	return composeConfig(
		testAccAutoScalingPlansScalingPlanConfigBase(rName, tagName),
		fmt.Sprintf(`
resource "aws_autoscalingplans_scaling_plan" "test" {
  name = %[1]q

  application_source {
    tag_filter {
      key    = %[2]q
      values = [%[2]q]
    }
  }

  scaling_instruction {
    disable_dynamic_scaling = true

    max_capacity       = aws_autoscaling_group.test.max_size
    min_capacity       = aws_autoscaling_group.test.min_size
    resource_id        = format("autoScalingGroup/%%s", aws_autoscaling_group.test.name)
    scalable_dimension = "autoscaling:autoScalingGroup:DesiredCapacity"
    service_namespace  = "autoscaling"

    target_tracking_configuration {
      predefined_scaling_metric_specification {
        predefined_scaling_metric_type = "ASGAverageCPUUtilization"
      }

      target_value = 75
    }

    predictive_scaling_max_capacity_behavior = "SetForecastCapacityToMaxCapacity"
    predictive_scaling_mode                  = "ForecastOnly"

    predefined_load_metric_specification {
      predefined_load_metric_type = "ASGTotalCPUUtilization"
    }
  }
}
`, rName, tagName))
}

func testAccAutoScalingPlansScalingPlanConfigDynamicScalingCustomizedScalingMetricSpecification(rName, tagName string, targetValue int) string {
	return composeConfig(
		testAccAutoScalingPlansScalingPlanConfigBase(rName, tagName),
		fmt.Sprintf(`
resource "aws_autoscalingplans_scaling_plan" "test" {
  name = %[1]q

  application_source {
    tag_filter {
      key    = %[2]q
      values = [%[2]q]
    }
  }

  scaling_instruction {
    max_capacity       = aws_autoscaling_group.test.max_size
    min_capacity       = aws_autoscaling_group.test.min_size
    resource_id        = format("autoScalingGroup/%%s", aws_autoscaling_group.test.name)
    scalable_dimension = "autoscaling:autoScalingGroup:DesiredCapacity"
    service_namespace  = "autoscaling"

    target_tracking_configuration {
      customized_scaling_metric_specification {
        metric_name = "Memory %% Committed Bytes In Use"
        namespace   = "test"
        statistic   = "Average"
        dimensions = {
          "AutoScalingGroupName" = aws_autoscaling_group.test.name,
          "objectname"           = "Memory"
        }
      }

      target_value = %[3]d
    }
  }
}
`, rName, tagName, targetValue))
}

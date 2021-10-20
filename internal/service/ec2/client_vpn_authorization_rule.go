package ec2

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceClientVPNAuthorizationRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceClientVPNAuthorizationRuleCreate,
		Read:   resourceClientVPNAuthorizationRuleRead,
		Delete: resourceClientVPNAuthorizationRuleDelete,
		Importer: &schema.ResourceImporter{
			State: resourceClientVPNAuthorizationRuleImport,
		},

		Schema: map[string]*schema.Schema{
			"client_vpn_endpoint_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"target_network_cidr": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidCIDRNetworkAddress,
			},
			"access_group_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"access_group_id", "authorize_all_groups"},
			},
			"authorize_all_groups": {
				Type:         schema.TypeBool,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"access_group_id", "authorize_all_groups"},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceClientVPNAuthorizationRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn

	endpointID := d.Get("client_vpn_endpoint_id").(string)
	targetNetworkCidr := d.Get("target_network_cidr").(string)

	input := &ec2.AuthorizeClientVpnIngressInput{
		ClientVpnEndpointId: aws.String(endpointID),
		TargetNetworkCidr:   aws.String(targetNetworkCidr),
	}

	var accessGroupID string
	if v, ok := d.GetOk("access_group_id"); ok {
		accessGroupID = v.(string)
		input.AccessGroupId = aws.String(accessGroupID)
	}

	if v, ok := d.GetOk("authorize_all_groups"); ok {
		input.AuthorizeAllGroups = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	id := ClientVPNAuthorizationRuleCreateID(endpointID, targetNetworkCidr, accessGroupID)

	log.Printf("[DEBUG] Creating Client VPN authorization rule: %#v", input)
	_, err := conn.AuthorizeClientVpnIngress(input)
	if err != nil {
		return fmt.Errorf("error creating Client VPN authorization rule %q: %w", id, err)
	}

	_, err = WaitClientVPNAuthorizationRuleAuthorized(conn, id)
	if err != nil {
		return fmt.Errorf("error waiting for Client VPN authorization rule %q to be active: %w", id, err)
	}

	d.SetId(id)

	return resourceClientVPNAuthorizationRuleRead(d, meta)
}

func resourceClientVPNAuthorizationRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn

	result, err := FindClientVPNAuthorizationRule(conn,
		d.Get("client_vpn_endpoint_id").(string),
		d.Get("target_network_cidr").(string),
		d.Get("access_group_id").(string),
	)

	if tfawserr.ErrMessageContains(err, ErrCodeClientVPNAuthorizationRuleNotFound, "") {
		log.Printf("[WARN] EC2 Client VPN authorization rule (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error reading Client VPN authorization rule: %w", err)
	}

	if result == nil || len(result.AuthorizationRules) == 0 || result.AuthorizationRules[0] == nil {
		log.Printf("[WARN] EC2 Client VPN authorization rule (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	rule := result.AuthorizationRules[0]
	d.Set("client_vpn_endpoint_id", rule.ClientVpnEndpointId)
	d.Set("target_network_cidr", rule.DestinationCidr)
	d.Set("access_group_id", rule.GroupId)
	d.Set("authorize_all_groups", rule.AccessAll)
	d.Set("description", rule.Description)

	return nil
}

func resourceClientVPNAuthorizationRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn

	input := &ec2.RevokeClientVpnIngressInput{
		ClientVpnEndpointId: aws.String(d.Get("client_vpn_endpoint_id").(string)),
		TargetNetworkCidr:   aws.String(d.Get("target_network_cidr").(string)),
		RevokeAllGroups:     aws.Bool(d.Get("authorize_all_groups").(bool)),
	}
	if v, ok := d.GetOk("access_group_id"); ok {
		input.AccessGroupId = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Revoking Client VPN authorization rule %q", d.Id())
	err := deleteClientVpnAuthorizationRule(conn, input)
	if err != nil {
		return fmt.Errorf("error revoking Client VPN authorization rule %q: %w", d.Id(), err)
	}

	return nil
}

func resourceClientVPNAuthorizationRuleImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	endpointID, targetNetworkCidr, accessGroupID, err := ClientVPNAuthorizationRuleParseID(d.Id())
	if err != nil {
		return nil, err
	}

	d.Set("client_vpn_endpoint_id", endpointID)
	d.Set("target_network_cidr", targetNetworkCidr)
	d.Set("access_group_id", accessGroupID)
	return []*schema.ResourceData{d}, nil
}

func deleteClientVpnAuthorizationRule(conn *ec2.EC2, input *ec2.RevokeClientVpnIngressInput) error {
	id := ClientVPNAuthorizationRuleCreateID(
		aws.StringValue(input.ClientVpnEndpointId),
		aws.StringValue(input.TargetNetworkCidr),
		aws.StringValue(input.AccessGroupId))

	_, err := conn.RevokeClientVpnIngress(input)
	if tfawserr.ErrMessageContains(err, ErrCodeClientVPNAuthorizationRuleNotFound, "") {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = WaitClientVPNAuthorizationRuleRevoked(conn, id)

	return err
}

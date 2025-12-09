package bcc_terraform

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceFirewallRule() *schema.Resource {
	args := Defaults()
	args.injectContextFirewallTemplateById()
	args.injectCreateFirewallRule()

	return &schema.Resource{
		CreateContext: resourceFirewallRuleCreate,
		ReadContext:   resourceFirewallRuleRead,
		UpdateContext: resourceFirewallRuleUpdate,
		DeleteContext: resourceFirewallRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceFirewallImport,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: args,
	}
}

func resourceFirewallRuleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	firewallId := d.Get("firewall_id").(string)
	firewall, err := manager.GetFirewallTemplate(firewallId)
	if err != nil {
		return diag.Errorf("[ERROR-048]: crash via getting FirewallTemplate by id=%s: %s", firewallId, err)
	}

	var newFirewallRule bcc.FirewallRule
	newFirewallRule.Name = d.Get("name").(string)
	newFirewallRule.Direction = d.Get("direction").(string)
	newFirewallRule.Protocol = d.Get("protocol").(string)
	newFirewallRule.DestinationIp = d.Get("destination_ip").(string)

	if newFirewallRule.Protocol == "tcp" || newFirewallRule.Protocol == "udp" {
		err = setUpRule(&newFirewallRule, d)
		if err != nil {
			return diag.Errorf("[ERROR-048]: crash vid setup FirewallRule: %s", err)
		}
	}

	if err = firewall.CreateFirewallRule(&newFirewallRule); err != nil {
		return diag.Errorf("[ERROR-048]: crash via creating FirewallRule: %s", err)
	}

	d.SetId(newFirewallRule.ID)
	log.Printf("[INFO-048]: firewall Rule created, ID: %s", d.Id())

	return resourceFirewallRuleRead(ctx, d, meta)
}

func resourceFirewallRuleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	firewallId := d.Get("firewall_id").(string)
	firewall, err := manager.GetFirewallTemplate(firewallId)
	if err != nil {
		return diag.Errorf("[ERROR-048}: crash via getting Firewall Template by id=%s: %s", firewallId, err)
	}

	firewallRuleId := d.Id()
	firewallRule, err := firewall.GetRuleById(firewallRuleId)
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("[ERROR-048]: crash via getting fierwall Rule by id=%s: %s", firewallRuleId, err)
		}
	}

	fields := map[string]interface{}{
		"firewall_id":    firewall.ID,
		"name":           firewallRule.Name,
		"direction":      firewallRule.Direction,
		"protocol":       firewallRule.Protocol,
		"destination_ip": firewallRule.DestinationIp,
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.FromErr(err)
	}

	if firewallRule.DstPortRangeMin != nil {
		if err = d.Set("port_range", fmt.Sprintf("%d", *firewallRule.DstPortRangeMin)); err != nil {
			return diag.Errorf("[ERROR-048]: error setting MIN port_range: %s", err)
		}
	}
	if firewallRule.DstPortRangeMax != nil {
		if err = d.Set("port_range", fmt.Sprintf("%s:%d", d.Get("port_range").(string), *firewallRule.DstPortRangeMax)); err != nil {
			return diag.Errorf("[ERROR-049]: error setting MAX port_range: %s", err)
		}
	}

	return nil
}

func resourceFirewallRuleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	firewallId := d.Get("firewallId").(string)
	firewall, err := manager.GetFirewallTemplate(firewallId)
	if err != nil {
		return diag.Errorf("[ERROR-048}: crash via getting Firewall Template by id=%s: %s", firewallId, err)
	}

	firewallRuleId := d.Id()
	firewallRule, err := firewall.GetRuleById(firewallRuleId)
	if err != nil {
		return diag.Errorf("[ERROR-048]: crash via getting fierwall Rule by id=%S: %s", firewallRuleId, err)
	}

	firewallRule.Name = d.Get("name").(string)
	firewallRule.DestinationIp = d.Get("destination_ip").(string)
	firewallRule.Protocol = d.Get("protocol").(string)
	if firewallRule.Protocol == "tcp" || firewallRule.Protocol == "udp" {
		err = setUpRule(firewallRule, d)
		if err != nil {
			return diag.Errorf("[ERROR-048]: crash via setting up FirewallRule: %s", err)
		}
	}
	if err = firewallRule.Update(); err != nil {
		return diag.Errorf("[ERROR-048]: crash via updating Fierwall rule: %s", err)
	}

	return resourceFirewallRuleRead(ctx, d, meta)
}

func resourceFirewallRuleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	firewallId := d.Get("firewallId").(string)
	firewall, err := manager.GetFirewallTemplate(firewallId)
	if err != nil {
		return diag.Errorf("[ERROR-048}: crash via getting Firewall Template by id=%s: %s", firewallId, err)
	}

	firewallRuleId := d.Id()
	firewallRule, err := firewall.GetRuleById(firewallRuleId)
	if err != nil {
		return diag.Errorf("[ERROR-048]: crash via getting fierwall Rule by id=%S: %s", firewallRuleId, err)
	}

	err = firewallRule.Delete()
	if err != nil {
		return diag.Errorf("[ERROR-048]: crash via deleting Fierwall rule id=%s: %s", firewallId, err)
	}

	d.SetId("")
	log.Printf("[INFO-048] Fierwall rule deleted, ID: %s", firewallRuleId)
	return nil
}

func setUpRule(rule *bcc.FirewallRule, d *schema.ResourceData) (err error) {
	rule.DstPortRangeMax = nil
	rule.DstPortRangeMin = nil
	portRange := d.Get("port_range").(string)

	if portRange == "" {
		return nil
	}
	var min, max int
	var re_for_port_range = regexp.MustCompile(`(?m)^(\d+:\d+)$`)
	var re_for_port = regexp.MustCompile(`(?m)^(\d+)$`)
	if len(re_for_port_range.FindStringIndex(portRange)) > 0 {
		fmt.Sscanf(portRange, "%d:%d", &min, &max)
		rule.DstPortRangeMax = &max
		rule.DstPortRangeMin = &min
	} else if len(re_for_port.FindStringIndex(portRange)) > 0 {
		fmt.Sscanf(portRange, "%d", &min)
		rule.DstPortRangeMin = &min
	} else {
		return errors.New("PORT RANGE UNSUPPORTED FORMAT, " +
			"should be `val:val` or `val`")
	}

	return nil
}

func resourceFirewallImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	id := d.Id()
	ids := strings.Split(id, ",")

	firewall, err := manager.GetFirewallTemplate(ids[0])
	if err != nil {
		return nil, fmt.Errorf("[ERROR-048}: crash via getting Firewall Template by id=%s: %s", ids[0], err)
	}

	firewallRule, err := firewall.GetRuleById(ids[1])
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil, nil
		} else {
			return nil, fmt.Errorf("[ERROR-048]: crash via getting fierwall Rule by id=%s: %s", ids[1], err)
		}
	}

	d.SetId(firewallRule.ID)
	if err = d.Set("firewall_id", firewall.ID); err != nil {
		return nil, fmt.Errorf("[ERROR-048]: error setting firewall_id: %s", err)
	}

	return []*schema.ResourceData{d}, nil
}

package bcc_terraform

import (
	"context"
	"log"

	"time"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePort() *schema.Resource {
	args := Defaults()
	args.injectContextOptionalVdcById()
	args.injectCreatePort()

	return &schema.Resource{
		CreateContext: resourcePortCreate,
		ReadContext:   resourcePortRead,
		UpdateContext: resourcePortUpdate,
		DeleteContext: resourcePortDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourcePortImport,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: args,
	}
}

func resourcePortCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	portNetwork, err := GetNetworkById(d, manager, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	targetVdc, err := GetVdcByVal(portNetwork.Vdc.Id, manager)
	if err != nil {
		return diag.FromErr(err)
	}

	firewallsCount := d.Get("firewall_templates.#").(int)
	firewalls := make([]*bcc.FirewallTemplate, firewallsCount)
	firewallsResourceData := d.Get("firewall_templates").(*schema.Set).List()
	for j, firewallId := range firewallsResourceData {
		portFirewall, err := manager.GetFirewallTemplate(firewallId.(string))
		if err != nil {
			return diag.Errorf("firewall_templates: Error getting Firewall Template: %s", err)
		}
		firewalls[j] = portFirewall
	}

	ipAddressInterface, ok := d.GetOk("ip_address")
	var ipAddressStr string
	if ok {
		ipAddressStr = ipAddressInterface.(string)
	} else {
		ipAddressStr = "0.0.0.0"
	}

	log.Printf("[DEBUG] subnetInfo: %#v", targetVdc)
	newPort := bcc.NewPort(portNetwork, firewalls, ipAddressStr)
	newPort.Tags = unmarshalTagNames(d.Get("tags"))

	if err = targetVdc.WaitLock(); err != nil {
		return diag.FromErr(err)
	}
	if err = targetVdc.CreateEmptyPort(&newPort); err != nil {
		return diag.FromErr(err)
	}
	if err = newPort.WaitLock(); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(newPort.ID)
	log.Printf("[INFO] Port created, ID: %s", d.Id())

	return resourcePortRead(ctx, d, meta)
}

func resourcePortRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	port, err := manager.GetPort(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("id: Error getting port: %s", err)
		}
	}
	firewalls := make([]*string, len(port.FirewallTemplates))
	for i, firewall := range port.FirewallTemplates {
		firewalls[i] = &firewall.ID
	}

	fields := map[string]interface{}{
		"ip_address":         port.IpAddress,
		"network_id":         port.Network.ID,
		"vdc_id":             port.Network.Vdc.Id,
		"tags":               marshalTagNames(port.Tags),
		"firewall_templates": firewalls,
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("crash via setting resource data: %s", err)
	}

	return nil
}

func resourcePortUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	portId := d.Id()
	port, err := manager.GetPort(portId)
	if err != nil {
		return diag.Errorf("[ERROR-053]: %s", err)
	}
	if d.HasChange("tags") {
		port.Tags = unmarshalTagNames(d.Get("tags"))
	}
	ipAddress := d.Get("ip_address").(string)
	if d.HasChange("ip_address") {
		port.IpAddress = &ipAddress
	}

	if d.HasChange("firewall_templates") {
		firewallsCount := d.Get("firewall_templates.#").(int)
		firewalls := make([]*bcc.FirewallTemplate, firewallsCount)
		firewallsResourceData := d.Get("firewall_templates").(*schema.Set).List()
		for j, firewallId := range firewallsResourceData {
			portFirewall, err := manager.GetFirewallTemplate(firewallId.(string))
			if err != nil {
				return diag.Errorf("firewall_templates: Error updating Firewall Template: %s", err)
			}
			firewalls[j] = portFirewall
		}

		port.FirewallTemplates = firewalls
	}
	if err = port.Update(); err != nil {
		return diag.FromErr(err)
	}
	if err = port.WaitLock(); err != nil {
		return diag.FromErr(err)
	}
	return resourcePortRead(ctx, d, meta)
}

func resourcePortDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	portId := d.Id()

	port, err := manager.GetPort(portId)
	if err != nil {
		return diag.Errorf("id: Error getting port: %s", err)
	}

	err = port.ForceDelete()
	if err != nil {
		return diag.Errorf("Error deleting port: %s", err)
	}
	if err = port.WaitLock(); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Printf("[INFO] Port deleted, ID: %s", portId)
	return nil
}

func resourcePortImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()
	port, err := manager.GetPort(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil, err
		} else {
			return nil, err
		}
	}

	d.SetId(port.ID)

	return []*schema.ResourceData{d}, nil
}

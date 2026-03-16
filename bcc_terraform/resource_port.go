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
	args.injectContextResourcePort()

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
	fields := struct {
		ipAddressStr          string
		vdcId                 string
		networkId             string
		firewallsCount        int
		firewallsResourceData []interface{}
	}{
		ipAddressStr:          d.Get("ip_address").(string),
		vdcId:                 d.Get("vdc_id").(string),
		networkId:             d.Get("network_id").(string),
		firewallsCount:        d.Get("firewall_templates.#").(int),
		firewallsResourceData: d.Get("firewall_templates").(*schema.Set).List(),
	}
	_, ipOk := d.GetOk("ip_address")
	if !ipOk {
		fields.ipAddressStr = "0.0.0.0"
	}

	network, err := manager.GetNetwork(fields.networkId)
	if err != nil {
		return diag.Errorf("[ERROR-045] crash via get network: %s", err)
	}

	_, vdcOk := d.GetOk("vdc_id")
	if !vdcOk {
		fields.vdcId = network.Vdc.Id
	}

	vdc, err := manager.GetVdc(fields.vdcId)
	if err != nil {
		return diag.Errorf("[ERROR-045] crash via get vdc: %s", err)
	}

	firewalls := make([]*bcc.FirewallTemplate, fields.firewallsCount)
	for j, firewallId := range fields.firewallsResourceData {
		portFirewall, err := manager.GetFirewallTemplate(firewallId.(string))
		if err != nil {
			return diag.Errorf("[ERROR-045] crash via getting Firewall Template: %s", err)
		}
		firewalls[j] = portFirewall
	}

	port := bcc.NewPort(network, firewalls, fields.ipAddressStr)
	if vdcOk {
		port.Vdc = vdc
	}
	port.Tags = unmarshalTagNames(d.Get("tags"))

	if err = vdc.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-045] crash via wait lock for vdc: %s", err)
	}
	if err = vdc.CreateEmptyPort(&port); err != nil {
		return diag.Errorf("[ERROR-045] crash via creating empty port: %s", err)
	}
	if err = port.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-045] crash via wait lock for port: %s", err)
	}

	d.SetId(port.ID)
	log.Printf("[INFO] Port created, ID: %s", d.Id())

	return resourcePortRead(ctx, d, meta)
}

func resourcePortRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	port, err := manager.GetPort(d.Id())
	if err != nil {
		return resourceReadCheck(d, err, "[ERROR-045]:")
	}

	firewallTemplates := make([]*string, len(port.FirewallTemplates))
	for i, firewall := range port.FirewallTemplates {
		firewallTemplates[i] = &firewall.ID
	}

	fields := map[string]interface{}{
		"ip_address":         port.IpAddress,
		"network_id":         port.Network.ID,
		"vdc_id":             port.Vdc.ID,
		"tags":               marshalTagNames(port.Tags),
		"firewall_templates": firewallTemplates,
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-045] crash via set attrs: %s", err)
	}

	return nil
}

func resourcePortUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	portId := d.Id()
	port, err := manager.GetPort(portId)
	if err != nil {
		return diag.Errorf("[ERROR-045] crash via get port: %s", err)
	}

	if d.HasChange("tags") {
		port.Tags = unmarshalTagNames(d.Get("tags"))
	}

	if d.HasChange("ip_address") {
		ipAddress := d.Get("ip_address").(string)
		port.IpAddress = &ipAddress
	}

	if d.HasChange("firewall_templates") {
		firewallsCount := d.Get("firewall_templates.#").(int)
		firewalls := make([]*bcc.FirewallTemplate, firewallsCount)
		firewallsResourceData := d.Get("firewall_templates").(*schema.Set).List()
		for j, firewallId := range firewallsResourceData {
			portFirewall, err := manager.GetFirewallTemplate(firewallId.(string))
			if err != nil {
				return diag.Errorf("[ERROR-045] crash via updating Firewall Template: %s", err)
			}
			firewalls[j] = portFirewall
		}

		port.FirewallTemplates = firewalls
	}
	if err = port.Update(); err != nil {
		return diag.Errorf("[ERROR-045] crash via updating port: %s", err)
	}
	if err = port.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-045] crash via port waitlock: %s", err)
	}
	return resourcePortRead(ctx, d, meta)
}

func resourcePortDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	portId := d.Id()

	port, err := manager.GetPort(portId)
	if err != nil {
		return diag.Errorf("[ERROR-045] crash via getting port: %s", err)
	}

	err = port.ForceDelete()
	if err != nil {
		return diag.Errorf("[ERROR-045] crash via deleting port: %s", err)
	}
	port.WaitLock()

	return nil
}

func resourcePortImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()
	port, err := manager.GetPort(d.Id())
	if err != nil {
		log.Printf("[ERROR-045] crash via getting port by id='%s': %s", d.Id(), err)
		return nil, err
	}

	d.SetId(port.ID)

	return []*schema.ResourceData{d}, nil
}

package bcc_terraform

import (
	"context"
	"fmt"
	"log"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLbaas() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextResourceLbaas()

	return &schema.Resource{
		CreateContext: resourceLbaasCreate,
		ReadContext:   resourceLbaasRead,
		UpdateContext: resourceLbaasUpdate,
		DeleteContext: resourceLbaasDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceLbaasImport,
		},
		Schema: args,
	}
}

func resourceLbaasCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	portPrefix := "port.0"
	fields := struct {
		name         string
		port         map[string]interface{}
		floating     bool
		ipAddressStr string
		floatingIp   *bcc.Port
	}{
		name:         d.Get("name").(string),
		port:         d.Get("port.0").(map[string]interface{}),
		floating:     d.Get("floating").(bool),
		ipAddressStr: d.Get(MakePrefix(&portPrefix, "ip_address")).(string),
		floatingIp:   nil,
	}
	if fields.floating {
		fields.floatingIp = &bcc.Port{ID: "RANDOM_FIP"}
	}
	if fields.ipAddressStr == "" {
		fields.ipAddressStr = "0.0.0.0"
	}

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-049]: crash via getting VDC: %s", err)
	}

	network, err := manager.GetNetwork(fields.port["network_id"].(string))
	if err != nil {
		return diag.Errorf("[ERROR-049]: crash via getting network by id=%s: %s", fields.port["network_id"].(string), err)
	}
	if err = network.WaitLock(); err != nil {
		diag.Errorf("[ERROR-049]: crash via wait lock for network")
	}

	firewalls := make([]*bcc.FirewallTemplate, 0)
	_port := bcc.NewPort(network, firewalls, fields.ipAddressStr)
	lbaas := bcc.NewLoadBalancer(fields.name, vdc, &_port, fields.floatingIp)
	lbaas.Tags = unmarshalTagNames(d.Get("tags"))

	if err = vdc.CreateLoadBalancer(&lbaas); err != nil {
		return diag.Errorf("[ERROR-049]: crash via creating Lbaas: %s", err)
	}
	if err = lbaas.WaitLock(); err != nil {
		diag.Errorf("[ERROR-049]: crash via wait lock for lbaas")
	}

	d.SetId(lbaas.ID)
	log.Printf("[INFO] Lbaas created, ID: %s", d.Id())

	return resourceLbaasRead(ctx, d, meta)
}

func resourceLbaasRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	lbaas, err := manager.GetLoadBalancer(d.Id())
	if err != nil {
		return resourceReadCheck(d, err, "[ERROR-049]:")
	}

	lbaasPort := make([]interface{}, 1)
	lbaasPort[0] = map[string]interface{}{
		"ip_address": lbaas.Port.IpAddress,
		"network_id": lbaas.Port.Network.ID,
	}

	fields := map[string]interface{}{
		"name":        lbaas.Name,
		"floating":    false,
		"floating_ip": "",
		"port":        lbaasPort,
		"vdc_id":      lbaas.Vdc.ID,
		"tags":        marshalTagNames(lbaas.Tags),
	}
	if lbaas.Floating != nil {
		fields["floating"] = true
		fields["floating_ip"] = lbaas.Floating.IpAddress
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-049]: crash via set attrs: %s", err)
	}

	return nil
}

func resourceLbaasUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	lbaas, err := manager.GetLoadBalancer(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-049]: crash via getting Lbaas by 'id'=%s: %s", d.Id(), err)
	}

	if d.HasChange("name") {
		lbaas.Name = d.Get("name").(string)
	}
	if d.HasChange("floating") {
		if !d.Get("floating").(bool) {
			lbaas.Floating = &bcc.Port{IpAddress: nil}
		} else {
			lbaas.Floating = &bcc.Port{ID: "RANDOM_FIP"}
		}
	}
	if d.HasChange("tags") {
		lbaas.Tags = unmarshalTagNames(d.Get("tags"))
	}
	lbaasPort := d.Get("port.0").(map[string]interface{})

	if ipAddress := lbaasPort["ipAddress"]; ipAddress != nil {
		if val, ok := ipAddress.(string); ok {
			if lbaas.Port.IpAddress == nil || *lbaas.Port.IpAddress != val {
				lbaas.Port.IpAddress = &val
			}
		}
	}

	if err := repeatOnError(lbaas.Update, lbaas); err != nil {
		return diag.Errorf("[ERROR-049]: crash via updating lbaas: %s", err)
	}
	if err = lbaas.WaitLock(); err != nil {
		diag.Errorf("[ERROR-049]: crash via wait lock for lbaas")
	}

	return resourceLbaasRead(ctx, d, meta)
}

func resourceLbaasDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	lbaasId := d.Id()
	lbaas, err := manager.GetLoadBalancer(lbaasId)
	if err != nil {
		return diag.Errorf("[ERROR-049]: crash via getting Lbaas by 'id'=%s: %s", d.Id(), err)
	}

	if err = lbaas.Delete(); err != nil {
		return diag.Errorf("[ERROR-049]: crash via deleting lbaas: %s", err)
	}
	lbaas.WaitLock()

	return nil
}

func resourceLbaasImport(_ context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()
	lbaas, err := manager.GetLoadBalancer(d.Id())
	if err != nil {
		return nil, fmt.Errorf("[ERROR-049]: crash via getting Lbaas by 'id'=%s: %s", d.Id(), err)
	}

	d.SetId(lbaas.ID)
	return []*schema.ResourceData{d}, nil
}

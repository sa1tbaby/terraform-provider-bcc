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
	args.injectContextVdcById()
	args.injectCreateLbaas()

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

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-049]: crash via getting VDC: %s", err)
	}

	config := struct {
		Name       string
		Port       map[string]interface{}
		Floating   bool
		FloatingIp *bcc.Port
		NetworkId  string
		VdcId      string
		Tags       []bcc.Tag
	}{
		Name:       d.Get("name").(string),
		Port:       d.Get("port.0").(map[string]interface{}),
		Floating:   d.Get("floating").(bool),
		FloatingIp: nil,
		NetworkId:  d.Get("network_id").(string),
		VdcId:      d.Get("vdc_id").(string),
		Tags:       unmarshalTagNames(d.Get("tags")),
	}

	portPrefix := "port.0"
	ipAddressStr := d.Get(MakePrefix(&portPrefix, "ip_address")).(string)

	// create port
	if config.Floating {
		config.FloatingIp = &bcc.Port{ID: "RANDOM_FIP"}
	}

	network, err := manager.GetNetwork(config.Port["network_id"].(string))
	if err != nil {
		return diag.Errorf("[ERROR-049]: crash via getting network by id=%s: %s", config.Port["network_id"].(string), err)
	}
	if err = network.WaitLock(); err != nil {
		diag.Errorf("[ERROR-049]: crash via wait lock for network")
	}

	firewalls := make([]*bcc.FirewallTemplate, 0)
	if ipAddressStr == "" {
		ipAddressStr = "0.0.0.0"
	}
	port := bcc.NewPort(network, firewalls, ipAddressStr)

	newLbaas := bcc.NewLoadBalancer(config.Name, vdc, &port, config.FloatingIp)
	newLbaas.Tags = config.Tags

	err = vdc.CreateLoadBalancer(&newLbaas)
	if err != nil {
		return diag.Errorf("[ERROR-049]: crash via creating Lbaas: %s", err)
	}
	if err = newLbaas.WaitLock(); err != nil {
		diag.Errorf("[ERROR-049]: crash via wait lock for lbaas")
	}

	d.SetId(newLbaas.ID)
	return resourceLbaasRead(ctx, d, meta)
}

func resourceLbaasRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	lbaas, err := manager.GetLoadBalancer(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("[ERROR-049]: crash via getting Lbaas by 'id'=%s: %s", d.Id(), err)
		}
	}

	lbaasPort := make([]interface{}, 1)
	lbaasPort[0] = map[string]interface{}{
		"ip_address": lbaas.Port.IpAddress,
		"network_id": lbaas.Port.Network.ID,
	}

	fields := map[string]interface{}{
		"name":        lbaas.Name,
		"floating":    lbaas.Floating != nil,
		"floating_ip": "",
		"port":        lbaasPort,
		"vdc_id":      lbaas.Vdc.ID,
		"tags":        marshalTagNames(lbaas.Tags),
	}
	if lbaas.Floating != nil {
		fields["floating_ip"] = lbaas.Floating.IpAddress
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.FromErr(err)
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
		d.Set("floating", lbaas.Floating != nil)
	}
	if d.HasChange("tags") {
		lbaas.Tags = unmarshalTagNames(d.Get("tags"))
	}
	lbaasPort := d.Get("port.0").(map[string]interface{})
	ipAddress := lbaasPort["ipAddress"].(string)
	if ipAddress != *lbaas.Port.IpAddress {
		lbaas.Port.IpAddress = &ipAddress
	}
	if err := repeatOnError(lbaas.Update, lbaas); err != nil {
		return diag.Errorf("[ERROR-049]: crash via updating lbaas: %s", err)
	}
	if err = lbaas.WaitLock(); err != nil {
		diag.Errorf("[ERROR-049]: crash via wait lock for lbaas")
	}

	return resourceLbaasRead(ctx, d, meta)
}

func resourceLbaasDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	d.SetId("")
	log.Printf("[INFO-049] Lbaas deleted, ID: %s", lbaasId)

	return nil
}

func resourceLbaasImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()
	lbaas, err := manager.GetLoadBalancer(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil, fmt.Errorf("[ERROR-049]: Lbaas not found")
		} else {
			return nil, fmt.Errorf("[ERROR-049]: crash via getting Lbaas by 'id'=%s: %s", d.Id(), err)
		}
	}

	d.SetId(lbaas.ID)
	return []*schema.ResourceData{d}, nil
}

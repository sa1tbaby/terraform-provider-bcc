package bcc_terraform

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVdc() *schema.Resource {
	args := Defaults()
	args.injectCreateVdc()
	args.injectContextHypervisorById()

	return &schema.Resource{
		CreateContext: resourceVdcCreate,
		ReadContext:   resourceVdcRead,
		UpdateContext: resourceVdcUpdate,
		DeleteContext: resourceVdcDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVdcImport,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: args,
		CustomizeDiff: func(ctx context.Context, rd *schema.ResourceDiff, i interface{}) error {
			if rd.Id() != "" && !rd.HasChange("project_id") {
				rd.Clear("id")
				rd.Clear("default_network_id")
			}
			return nil
		},
	}
}

func resourceVdcCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	targetProject, err := manager.GetProject(d.Get("project_id").(string))
	if err != nil {
		return diag.Errorf("[ERROR-006]: %s", err)
	}

	targetHypervisor, err := GetHypervisorById(d, manager, targetProject)
	if err != nil {
		return diag.Errorf("[ERROR-006]: %s", err)
	}

	vdc := bcc.NewVdc(d.Get("name").(string), targetHypervisor)
	vdc.Tags = unmarshalTagNames(d.Get("tags"))

	// if we creating multiple vdc at once, there are need some time to get new vnid
	f := func() error { return targetProject.CreateVdc(&vdc) }
	if err = repeatOnError(f, targetProject); err != nil {
		return diag.Errorf("[ERROR-006]: crash via creating vdc: %s", err)
	}

	if err = vdc.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-006]: %s", err)
	}
	if mtu, ok := d.GetOk("default_network_mtu"); ok {
		networks, err := vdc.GetNetworks(bcc.Arguments{"defaults_only": "true"})
		if err != nil {
			return diag.Errorf("[ERROR-006]: %s", err)
		}
		if len(networks) != 1 {
			return diag.Errorf("[ERROR-006]: expected 1 network, got %d networks", len(networks))
		}
		network := networks[0]
		mtuValue := mtu.(int)
		network.Mtu = &mtuValue
		if err = network.Update(); err != nil {
			return diag.Errorf("[ERROR-006]: %s", err)
		}
	}

	//vdc.GetNetworks()
	d.SetId(vdc.ID)
	log.Printf("[INFO-006] VDC created, ID: %s", d.Id())

	return resourceVdcRead(ctx, d, meta)
}

func resourceVdcRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	vdc, err := manager.GetVdc(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("[ERROR-006] id: Error getting vdc: %s", err)
		}
	}
	networks, err := vdc.GetNetworks(bcc.Arguments{"defaults_only": "true"})
	if err != nil {
		return diag.Errorf("[ERROR-006]: %s", err)
	}
	if len(networks) != 1 {
		return diag.Errorf("[ERROR-006]: expected 1 network, got %d networks", len(networks))
	}
	network := networks[0]

	subnets, err := network.GetSubnets()
	if err != nil {
		return diag.Errorf("[ERROR-006]: %s", err)
	}

	flattenedSubnets := make([]map[string]interface{}, len(subnets))
	for i, subnet := range subnets {
		dnsStrings := make([]string, len(subnet.DnsServers))
		for i2, dns := range subnet.DnsServers {
			dnsStrings[i2] = dns.DNSServer
		}
		flattenedSubnets[i] = map[string]interface{}{
			"id":       subnet.ID,
			"cidr":     subnet.CIDR,
			"dhcp":     subnet.IsDHCP,
			"gateway":  subnet.Gateway,
			"start_ip": subnet.StartIp,
			"end_ip":   subnet.EndIp,
			"dns":      dnsStrings,
		}
	}
	fields := map[string]interface{}{
		"name":                    vdc.Name,
		"project_id":              vdc.Project.ID,
		"hypervisor_id":           vdc.Hypervisor.ID,
		"default_network_id":      network.ID,
		"default_network_name":    network.Name,
		"default_network_subnets": flattenedSubnets,
		"default_network_mtu":     network.Mtu,
		"tags":                    marshalTagNames(vdc.Tags),
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(vdc.ID)
	return nil
}

func resourceVdcUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := manager.GetVdc(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-006] id: Error getting vdc: %s", err)
	}
	if d.HasChange("hypervisor_id") {
		return diag.Errorf("[ERROR-006] hypervisor_id: you can`t change hypervisor type on created vdc")
	}
	if d.HasChange("name") {
		vdc.Name = d.Get("name").(string)
	}
	if d.HasChange("tags") {
		vdc.Tags = unmarshalTagNames(d.Get("tags"))
	}
	err = vdc.Update()
	if err != nil {
		return diag.Errorf("[ERROR-006] name: Error rename vdc: %s", err)
	}
	if d.HasChange("default_network_mtu") {
		networks, err := vdc.GetNetworks(bcc.Arguments{"defaults_only": "true"})
		if err != nil {
			return diag.Errorf("[ERROR-006] Error getting vdc networks: %s", err)
		}
		if len(networks) != 1 {
			return diag.Errorf("[ERROR-006] Expected 1 network, got %d networks", len(networks))
		}
		network := networks[0]

		if mtu, ok := d.GetOk("default_network_mtu"); ok {
			mtuValue := mtu.(int)
			network.Mtu = &mtuValue
		} else {
			network.Mtu = nil
		}
		err = network.Update()
		if err != nil {
			return diag.Errorf("[ERROR-006] Error updating vdc default network: %s", err)
		}
	}

	if err = vdc.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-006] Error locking vdc: %s", err)
	}

	return resourceVdcRead(ctx, d, meta)
}

func resourceVdcDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	vdc, err := manager.GetVdc(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-006]: %s", err)
	}

	if err = vdc.Delete(); err != nil {
		return diag.Errorf("[ERROR-006]: %s", err)
	}
	vdc.WaitLock()

	return nil
}

func resourceVdcImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := manager.GetVdc(d.Id())
	if err != nil {
		d.SetId("")
		return nil, fmt.Errorf("[ERROR-006] id: Error getting vdc: %s", err)
	}

	d.SetId(vdc.ID)
	return []*schema.ResourceData{d}, nil
}

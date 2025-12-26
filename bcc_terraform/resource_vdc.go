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

	fields := map[string]interface{}{
		"name":          vdc.Name,
		"project_id":    vdc.Project.ID,
		"hypervisor_id": vdc.Hypervisor.ID,
		"tags":          marshalTagNames(vdc.Tags),
	}

	if len(networks) != 0 {
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

		fields["default_network_subnets"] = flattenedSubnets
		fields["default_network_mtu"] = network.Mtu
		fields["default_network_name"] = network.Name
		fields["default_network_id"] = network.ID
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

package bcc_terraform

import (
	"context"
	"fmt"
	"log"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceNetwork() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextResourceNetwork()

	return &schema.Resource{
		CreateContext: resourceNetworkCreate,
		UpdateContext: resourceNetworkUpdate,
		ReadContext:   resourceNetworkRead,
		DeleteContext: resourceNetworkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceNetworkImport,
		},
		Schema: args,
		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
			if d.Id() != "" {
				if d.HasChange("subnets.0.cidr") {
					oldNet, newNet := d.GetChange("subnets.0.cidr")
					if oldNet.(string) != newNet.(string) {
						return fmt.Errorf("[ERROR-009]: changing 'cidr' attribute is not supported")
					}
				}
			}
			return nil
		},
	}
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-009]: %s", err)
	}

	network := bcc.NewNetwork(d.Get("name").(string))
	network.Tags = unmarshalTagNames(d.Get("tags"))

	if mtu, ok := d.GetOk("mtu"); ok {
		mtuValue := mtu.(int)
		network.Mtu = &mtuValue
	} else {
		network.Mtu = nil
	}

	if err = vdc.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-009]: crash via wait lock %s", err)
	}
	if err = vdc.CreateNetwork(&network); err != nil {
		return diag.Errorf("[ERROR-009]: crash via creating %s", err)
	}
	if err = vdc.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-009]: crash via wait lock %s", err)
	}

	if err = createSubnet(d, manager, &network); err != nil {
		return diag.Errorf("[ERROR-009]: crash via creating gsub nets%s", err)
	}
	if err = network.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-009]: crash via waitlock %s", err)
	}

	d.SetId(network.ID)
	log.Printf("[INFO] Network created, ID: %s", d.Id())

	return resourceNetworkRead(ctx, d, meta)
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	manager := meta.(*CombinedConfig).Manager()

	network, err := manager.GetNetwork(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-009]: %s", err)
	}
	needUpdate := false

	if d.HasChange("tags") {
		network.Tags = unmarshalTagNames(d.Get("tags"))
		needUpdate = true
	}

	if d.HasChange("name") {
		network.Name = d.Get("name").(string)
		needUpdate = true
	}
	if d.HasChange("mtu") {
		if mtu, ok := d.GetOk("mtu"); ok {
			mtuValue := mtu.(int)
			network.Mtu = &mtuValue
		} else {
			network.Mtu = nil
		}
		needUpdate = true
	}
	if needUpdate {
		err = network.Update()
		if err != nil {
			return diag.Errorf("[ERROR-009]: %s", err)
		}
	}

	if d.HasChange("subnets") {
		diagErr := updateSubnet(d, manager)
		if diagErr != nil {
			return diagErr
		}
	}
	if err = network.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-009]: %s", err)
	}

	return resourceNetworkRead(ctx, d, meta)
}

func resourceNetworkRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	network, err := manager.GetNetwork(d.Id())
	if err != nil {
		return resourceReadCheck(d, err, "[ERROR-009]:")
	}

	subnets, err := network.GetSubnets()
	if err != nil {
		return diag.Errorf("[ERROR-009]: %s", err)
	}

	subnetsMap := make([]map[string]interface{}, len(subnets))
	for i, subnet := range subnets {
		dnsStrings := make([]string, len(subnet.DnsServers))
		for i2, dns := range subnet.DnsServers {
			dnsStrings[i2] = dns.DNSServer
		}
		subnetsMap[i] = map[string]interface{}{
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
		"name":     network.Name,
		"tags":     marshalTagNames(network.Tags),
		"mtu":      network.Mtu,
		"subnets":  subnetsMap,
		"vdc_id":   network.Vdc.Id,
		"external": network.External,
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-009]: %s", err)
	}

	return nil
}

func resourceNetworkDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	network, err := manager.GetNetwork(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-009]: %s", err)
	}

	if err = repeatOnError(network.Delete, network); err != nil {
		return diag.Errorf("[ERROR-009]: crash via deleting network-%s: %s", d.Id(), err)
	}
	network.WaitLock()

	return nil
}

func resourceNetworkImport(_ context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()
	network, err := manager.GetNetwork(d.Id())
	if err != nil {
		return nil, fmt.Errorf("[ERROR-009]: crash via getting network-%s: %s", d.Id(), err)
	}

	d.SetId(network.ID)
	return []*schema.ResourceData{d}, nil
}

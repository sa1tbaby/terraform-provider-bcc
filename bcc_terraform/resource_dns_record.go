package bcc_terraform

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDnsRecord() *schema.Resource {
	args := Defaults()
	args.injectContextDnsById()
	args.injectCreateDnsRecord()

	return &schema.Resource{
		CreateContext: resourceDnsRecordCreate,
		ReadContext:   resourceDnsRecordRead,
		UpdateContext: resourceDnsRecordUpdate,
		DeleteContext: resourceDnsRecordDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: args,
	}
}

func resourceDnsRecordCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	dnsId := d.Get("dns_id").(string)
	dns, err := manager.GetDns(dnsId)
	if err != nil {
		return diag.Errorf("vdc_id: Error getting Dns: %s", err)
	}

	host := d.Get("host").(string)
	if !strings.HasSuffix(host, dns.Name) {
		return diag.Errorf("host: must be ending by '%s'", dns.Name)
	}

	newDnsRecord := bcc.NewDnsRecord(
		d.Get("data").(string),
		d.Get("flag").(int),
		host,
		d.Get("port").(int),
		d.Get("priority").(int),
		d.Get("tag").(string),
		d.Get("ttl").(int),
		d.Get("type").(string),
		d.Get("weight").(int),
	)

	err = dns.CreateDnsRecord(&newDnsRecord)
	if err != nil {
		return diag.Errorf("Error creating Dns record: %s", err)
	}

	d.SetId(newDnsRecord.ID)
	log.Printf("[INFO] Dns record created, ID: %s", d.Id())

	return resourceDnsRecordRead(ctx, d, meta)
}

func resourceDnsRecordUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	dnsId := d.Get("dns_id").(string)
	dns, err := manager.GetDns(dnsId)

	if err != nil {
		return diag.Errorf("id: Error getting Dns: %s", err)
	}
	dnsRecord, err := dns.GetDnsRecord(d.Id())
	if err != nil {
		return diag.Errorf("id: Error getting Dns record: %s", err)
	}

	if d.HasChange("data") {
		dnsRecord.Data = d.Get("data").(string)
	}
	if d.HasChange("host") {
		dnsRecord.Host = d.Get("host").(string)
	}
	if d.HasChange("ttl") {
		dnsRecord.Ttl = d.Get("ttl").(int)
	}
	if d.HasChange("type") {
		dnsRecord.Type = d.Get("type").(string)
	}
	if d.HasChange("weight") {
		dnsRecord.Weight = d.Get("weight").(int)
	}
	if d.HasChange("flag") {
		dnsRecord.Flag = d.Get("flag").(int)
	}
	if d.HasChange("tag") {
		dnsRecord.Tag = d.Get("tag").(string)
	}
	if d.HasChange("priority") {
		dnsRecord.Priority = d.Get("priority").(int)
	}
	if d.HasChange("port") {
		dnsRecord.Port = d.Get("port").(int)
	}

	if err = dnsRecord.Update(); err != nil {
		return diag.FromErr(err)
	}

	return resourceDnsRecordRead(ctx, d, meta)
}

func resourceDnsRecordRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	dnsId := d.Get("dns_id").(string)
	dns, err := manager.GetDns(dnsId)
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("id: Error getting Dns: %s", err)
		}
	}

	dnsRecord, err := dns.GetDnsRecord(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("id: Error getting Dns record: %s", err)
		}
	}

	fields := map[string]interface{}{
		"dns_id":   d.Get("dns_id").(string),
		"data":     dnsRecord.Data,
		"flag":     dnsRecord.Flag,
		"host":     dnsRecord.Host,
		"port":     dnsRecord.Port,
		"priority": dnsRecord.Priority,
		"tag":      dnsRecord.Tag,
		"ttl":      dnsRecord.Ttl,
		"type":     dnsRecord.Type,
		"weight":   dnsRecord.Weight,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceDnsRecordDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	dnsId := d.Get("dns_id").(string)
	dns, err := manager.GetDns(dnsId)
	if err != nil {
		return diag.Errorf("id: Error getting Dns: %s", err)
	}
	dnsRecord, err := dns.GetDnsRecord(d.Id())
	if err != nil {
		return diag.Errorf("id: Error getting Dns record: %s", err)
	}

	err = dnsRecord.Delete()
	if err != nil {
		return diag.Errorf("Error deleting Dns: %s", err)
	}

	return nil
}

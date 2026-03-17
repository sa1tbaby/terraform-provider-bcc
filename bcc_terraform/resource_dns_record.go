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
	args.injectContextRequiredDns()
	args.injectContextResourceDnsRecord()

	return &schema.Resource{
		CreateContext: resourceDnsRecordCreate,
		UpdateContext: resourceDnsRecordUpdate,
		ReadContext:   resourceDnsRecordRead,
		DeleteContext: resourceDnsRecordDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceDnsRecordImport,
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

	fields := struct {
		DnsId    string
		Data     string
		Flag     int
		Host     string
		Port     int
		Priority int
		Tag      string
		Ttl      int
		Type     string
		Weight   int
	}{
		d.Get("dns_id").(string),
		d.Get("data").(string),
		d.Get("flag").(int),
		d.Get("host").(string),
		d.Get("port").(int),
		d.Get("priority").(int),
		d.Get("tag").(string),
		d.Get("ttl").(int),
		d.Get("type").(string),
		d.Get("weight").(int),
	}

	dns, err := manager.GetDns(fields.DnsId)
	if err != nil {
		return diag.Errorf("[ERROR-047] crash via get dns: %s", err)
	}

	if !strings.HasSuffix(fields.Host, dns.Name) {
		return diag.Errorf("[ERROR-047] host must be ending by '%s'", dns.Name)
	}

	newDnsRecord := bcc.NewDnsRecord(
		fields.Data, fields.Flag, fields.Host, fields.Port, fields.Priority,
		fields.Tag, fields.Ttl, fields.Type, fields.Weight,
	)

	err = dns.CreateDnsRecord(&newDnsRecord)
	if err != nil {
		return diag.Errorf("[ERROR-047] crash via creating Dns record: %s", err)
	}

	d.SetId(newDnsRecord.ID)
	log.Printf("[INFO] Dns record created, ID: %s", d.Id())

	return resourceDnsRecordRead(ctx, d, meta)
}

func resourceDnsRecordUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	fields := struct {
		ID       string
		dnsId    string
		Data     string
		Flag     int
		Host     string
		Port     int
		Priority int
		Tag      string
		Ttl      int
		Type     string
		Weight   int
	}{
		ID:       d.Id(),
		dnsId:    d.Get("dns_id").(string),
		Data:     d.Get("data").(string),
		Flag:     d.Get("flag").(int),
		Host:     d.Get("host").(string),
		Port:     d.Get("port").(int),
		Priority: d.Get("priority").(int),
		Tag:      d.Get("tag").(string),
		Ttl:      d.Get("ttl").(int),
		Type:     d.Get("type").(string),
		Weight:   d.Get("weight").(int),
	}

	dns, err := manager.GetDns(fields.dnsId)
	if err != nil {
		return diag.Errorf("[ERROR-047] crash via getting Dns: %s", err)
	}
	dnsRecord, err := dns.GetDnsRecord(fields.ID)
	if err != nil {
		return diag.Errorf("[ERROR-047] crash via getting Dns record: %s", err)
	}

	if d.HasChange("data") {
		dnsRecord.Data = fields.Data
	}
	if d.HasChange("host") {
		dnsRecord.Host = fields.Host
	}
	if d.HasChange("ttl") {
		dnsRecord.Ttl = fields.Ttl
	}
	if d.HasChange("type") {
		dnsRecord.Type = fields.Type
	}
	if d.HasChange("weight") {
		dnsRecord.Weight = fields.Weight
	}
	if d.HasChange("flag") {
		dnsRecord.Flag = fields.Flag
	}
	if d.HasChange("tag") {
		dnsRecord.Tag = fields.Tag
	}
	if d.HasChange("priority") {
		dnsRecord.Priority = fields.Priority
	}
	if d.HasChange("port") {
		dnsRecord.Port = fields.Port
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
			return diag.Errorf("[ERROR-047] crash via get dns: %s", err)
		}
	}

	dnsRecord, err := dns.GetDnsRecord(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("[ERROR-047] crash via get Dns record: %s", err)
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
		return diag.Errorf("[ERROR-047] crash via set attrs: %s", err)
	}

	return nil
}

func resourceDnsRecordDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	dnsId := d.Get("dns_id").(string)
	dns, err := manager.GetDns(dnsId)
	if err != nil {
		return diag.Errorf("[ERROR-047] crash via get Dns: %s", err)
	}
	dnsRecord, err := dns.GetDnsRecord(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-047] crash via get Dns record: %s", err)
	}

	err = dnsRecord.Delete()
	if err != nil {
		return diag.Errorf("[ERROR-047] crash via delete Dns: %s", err)
	}

	return nil
}

func resourceDnsRecordImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	id := d.Id()
	ids := strings.Split(id, ",")

	dns, err := manager.GetDns(ids[0])
	if err != nil {
		return nil, fmt.Errorf("[ERROR-047] crash via get Dns: %s", err)
	}

	dnsRecord, err := dns.GetDnsRecord(ids[1])
	if err != nil {
		d.SetId("")
		return nil, fmt.Errorf("[ERROR-047] crash via get Dns record: %s", err)
	}

	d.SetId(dnsRecord.ID)
	if err := d.Set("dns_id", dns.ID); err != nil {
		return nil, fmt.Errorf("[ERROR-047] crash via set 'dns_id': %s", err)
	}

	return []*schema.ResourceData{d}, nil
}

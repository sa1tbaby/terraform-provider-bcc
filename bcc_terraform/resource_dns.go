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

func resourceDns() *schema.Resource {
	args := Defaults()
	args.injectCreateDns()
	args.injectContextProjectById()

	return &schema.Resource{
		CreateContext: resourceDnsCreate,
		ReadContext:   resourceDnsRead,
		DeleteContext: resourceDnsDelete,
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

func resourceDnsCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	project, err := GetProjectById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-046]: crash via getting Projecct by 'id': %s", err)
	}

	name := d.Get("name").(string)
	if !strings.HasSuffix(name, ".") {
		return diag.Errorf("[ERROR-046]: 'name' must be ending by '.'")
	}

	newDns := bcc.NewDns(name)
	newDns.Tags = unmarshalTagNames(d.Get("tags"))

	err = project.CreateDns(&newDns)
	if err != nil {
		return diag.Errorf("[ERROR-046]: creating Dns: %s", err)
	}

	d.SetId(newDns.ID)
	log.Printf("[INFO-046]: Dns created, ID: %s", d.Id())

	return resourceDnsRead(ctx, d, meta)
}

func resourceDnsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	dns, err := manager.GetDns(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("[ERROR-046]: crash via getting Dns: %s", err)
		}
	}

	fields := map[string]interface{}{
		"name":       dns.Name,
		"project_id": dns.Project.ID,
		"tags":       unmarshalTagNames(dns.Tags),
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-046]: crash via setting resource data: %s", err)
	}

	return nil
}

func resourceDnsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	dns, err := manager.GetDns(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-046]: crash via getting Dns by 'id': %s", err)
	}

	err = dns.Delete()
	if err != nil {
		return diag.Errorf("[ERROR-046]: crash via deleting Dns: %s", err)
	}

	return nil
}

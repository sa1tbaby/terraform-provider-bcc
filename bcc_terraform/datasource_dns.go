package bcc_terraform

import (
	"context"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDns() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredProject()
	args.injectContextDataDns()
	args.injectContextGetDns()

	return &schema.Resource{
		ReadContext: dataSourceDnsRead,
		Schema:      args,
	}
}

func dataSourceDnsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-028] crash via chose the target for get: %s", err)
	}
	var dns *bcc.Dns
	if strings.EqualFold(target, "id") {
		dnsId := d.Get("id").(string)
		dns, err = manager.GetDns(dnsId)
		if err != nil {
			return diag.Errorf("[ERROR-028] crash via getting dns by id=%s: %s", dnsId, err)
		}
	} else {
		dns, err = GetDnsByName(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-028] crash via getting dns by name: %s", err)
		}
	}

	fields := map[string]interface{}{
		"id":         dns.ID,
		"name":       dns.Name,
		"tags":       marshalTagNames(dns.Tags),
		"project_id": dns.Project.ID,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-028] crash via set attrs: %s", err)
	}

	return nil
}

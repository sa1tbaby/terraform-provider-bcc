package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceDnss() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredProject()
	args.injectContextDataDnsList()

	return &schema.Resource{
		ReadContext: dataSourceDnssRead,
		Schema:      args,
	}
}

func dataSourceDnssRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	project, err := GetProjectById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-029] crash via getting project: %s", err)
	}

	dnsList, err := project.GetDnss()
	if err != nil {
		return diag.Errorf("[ERROR-029] crash via retrieving dnss: %s", err)
	}

	dnsMap := make([]map[string]interface{}, len(dnsList))
	for i, dns := range dnsList {
		dnsMap[i] = map[string]interface{}{
			"id":   dns.ID,
			"name": dns.Name,
			"tags": marshalTagNames(dns.Tags),
		}
	}

	hash, err := hashstructure.Hash(dnsList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-029] crash via computing hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":         fmt.Sprintf("dnss/%d", hash),
		"project_id": project.Name,
		"dnss":       dnsMap,
	}
	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-029] crash via set attrs: %s", err)
	}

	return nil
}

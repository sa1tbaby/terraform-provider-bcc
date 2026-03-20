package bcc_terraform

import (
	"context"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceKubernetesTemplate() *schema.Resource {
	args := Defaults()
	args.injectResultKubernetesTemplate()
	args.injectContextRequiredVdc()
	args.injectContextGetKubernetesTemplate() // override name

	return &schema.Resource{
		ReadContext: dataSourceKubernetesTemplateRead,
		Schema:      args,
	}
}

func dataSourceKubernetesTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-036] crash via  getting KubernetesTemplate: %s", err)
	}

	var k8sTemplate *bcc.KubernetesTemplate
	if strings.EqualFold(target, "id") {
		k8sTemplateId := d.Get("id").(string)
		k8sTemplate, err = manager.GetKubernetesTemplate(k8sTemplateId)
		if err != nil {
			return diag.Errorf("[ERROR-036] crash via getting KubernetesTemplate: %s", err)
		}
	} else {
		vdc, err := GetVdcById(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-036] crash via getting vdc: %s", err)
		}

		k8sTemplate, err = GetKubernetesTemplateByName(d, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-036] crash via getting KubernetesTemplate: %s", err)
		}
	}

	fields := map[string]interface{}{
		"id":           k8sTemplate.ID,
		"name":         k8sTemplate.Name,
		"min_node_cpu": k8sTemplate.MinNodeCpu,
		"min_node_ram": k8sTemplate.MinNodeRam,
		"min_node_hdd": k8sTemplate.MinNodeHdd,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-036] crash via set attrs: %s", err)
	}
	return nil
}

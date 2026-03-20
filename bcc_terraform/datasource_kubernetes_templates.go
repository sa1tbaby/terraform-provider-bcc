package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceKubernetesTemplates() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectResultListKubernetesTemplate()

	return &schema.Resource{
		ReadContext: dataSourceKubernetesTemplateReadRead,
		Schema:      args,
	}
}

func dataSourceKubernetesTemplateReadRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-037] crash via getting vdc: %s", err)
	}

	k8sTmpList, err := vdc.GetKubernetesTemplates()
	if err != nil {
		return diag.Errorf("[ERROR-037] crash via retrieving KubernetesTemplateRead: %s", err)
	}

	k8sTmpMap := make([]map[string]interface{}, len(k8sTmpList))
	for i, KubernetesTemplateRead := range k8sTmpList {
		k8sTmpMap[i] = map[string]interface{}{
			"id":           KubernetesTemplateRead.ID,
			"name":         KubernetesTemplateRead.Name,
			"min_node_cpu": KubernetesTemplateRead.MinNodeCpu,
			"min_node_ram": KubernetesTemplateRead.MinNodeRam,
			"min_node_hdd": KubernetesTemplateRead.MinNodeHdd,
		}
	}

	hash, err := hashstructure.Hash(k8sTmpList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-037] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":                   fmt.Sprintf("kubernetes_templates/%d", hash),
		"vdc_id":               vdc.ID,
		"kubernetes_templates": k8sTmpMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-037] crash via set attrs: %s", err)
	}

	return nil
}

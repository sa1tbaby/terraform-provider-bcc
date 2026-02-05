package bcc_terraform

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func (args *Arguments) injectContextKubernetesTemplateById() {
	args.merge(Arguments{
		"template_id": {
			Type:        schema.TypeString,
			ForceNew:    true,
			Required:    true,
			Description: "id of the Kubernetes Template",
		},
	})
}

func (args *Arguments) injectContextGetKubernetesTemplate() {
	args.merge(Arguments{
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Kubernetes Template name",
		},
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Kubernetes Template identifier",
		},
	})
}

func (args *Arguments) injectResultKubernetesTemplate() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Kubernetes Template identifier",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Kubernetes Template name",
		},
		"min_node_cpu": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "minimum required number of virtual CPUs for the template",
		},
		"min_node_ram": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "minimum required amount of RAM for the template, in GB",
		},
		"min_node_hdd": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "minimum required disk size for the template, in GB",
		},
	})
}

func (args *Arguments) injectResultListKubernetesTemplate() {
	s := Defaults()
	s.injectResultKubernetesTemplate()

	args.merge(Arguments{
		"kubernetes_templates": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: s,
			},
		},
	})
}

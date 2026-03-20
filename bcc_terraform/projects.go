package bcc_terraform

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func (args *Arguments) injectContextGetProject() {
	args.merge(Arguments{
		"name": {
			Type:        schema.TypeString,
			Description: "Project name",
			Optional:    true,
			Computed:    true,
		},
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Project identifier",
		},
	})
}

func (args *Arguments) injectContextRequiredProject() {
	args.merge(Arguments{
		"project_id": {
			Type:        schema.TypeString,
			Description: "Project identifier",
			Required:    true,
			ForceNew:    true,
		},
	})
}

func (args *Arguments) injectContextOptionalProject() {
	args.merge(Arguments{
		"project_id": {
			Type:        schema.TypeString,
			Description: "id of the Project",
			Optional:    true,
		},
	})
}

func (args *Arguments) injectContextResourceProject() {
	args.merge(Arguments{
		"name": {
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: validation.All(
				validation.NoZeroValues,
				validation.StringLenBetween(1, 100),
			),
			Description: "name of the Project",
		},
		"tags": newTagNamesResourceSchema("tags of the Project"),
	})
}

func (args *Arguments) injectContextDataProject() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Project identifier",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Project name",
		},
		"tags": newTagNamesDataSchema("tags of the router"),
	})
}

func (args *Arguments) injectContextDataProjectList() {
	s := Defaults()
	s.injectContextDataProject()

	args.merge(Arguments{
		"projects": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: s,
			},
		},
	})
}

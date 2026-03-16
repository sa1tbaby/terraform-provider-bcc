package bcc_terraform

import (
	"context"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceProject() *schema.Resource {
	args := Defaults()
	args.injectContextDataProject()
	args.injectContextGetProject()

	return &schema.Resource{
		ReadContext: dataSourceProjectRead,
		Schema:      args,
	}
}

func dataSourceProjectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-002] crash via chose target: %s", err)
	}

	var project *bcc.Project
	if target == "id" {
		projectId := d.Get("id").(string)
		project, err = manager.GetProject(projectId)
		if err != nil {
			return diag.Errorf("[ERROR-002] crash via getting project by id=%s: %s", projectId, err)
		}
	} else {
		project, err = GetProjectByName(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-002] crash via getting project by name: %s", err)
		}
	}

	config := map[string]interface{}{
		"id":   project.ID,
		"name": project.Name,
		"tags": marshalTagNames(project.Tags),
	}

	if err := setResourceDataFromMap(d, config); err != nil {
		return diag.Errorf("[ERROR-002] crash via set attrs: %s", err)
	}

	return nil
}

package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceProjects() *schema.Resource {
	args := Defaults()
	args.injectContextDataProjectList()

	return &schema.Resource{
		ReadContext: dataSourceProjectsRead,
		Schema:      args,
	}
}

func dataSourceProjectsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	projectList, err := manager.GetProjects()
	if err != nil {
		return diag.Errorf("[ERROR-003] crash via getting projects: %s", err)
	}

	projectMap := make([]map[string]interface{}, len(projectList))
	for i, project := range projectList {
		projectMap[i] = map[string]interface{}{
			"id":   project.ID,
			"name": project.Name,
			"tags": marshalTagNames(project.Tags),
		}
	}

	hash, err := hashstructure.Hash(projectList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-003] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":       fmt.Sprintf("projects/%d", hash),
		"projects": projectMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-003] crash via set attrs: %s", err)
	}

	return nil
}

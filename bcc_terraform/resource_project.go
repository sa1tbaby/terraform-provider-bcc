package bcc_terraform

import (
	"context"
	"fmt"
	"log"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceProject() *schema.Resource {
	args := Defaults()
	args.injectContextResourceProject()

	return &schema.Resource{
		CreateContext: resourceProjectCreate,
		ReadContext:   resourceProjectRead,
		UpdateContext: resourceProjectUpdate,
		DeleteContext: resourceProjectDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceProjectImport,
		},
		Schema: args,
	}
}

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	var err error

	fields := struct {
		name     string
		clientId string
		client   *bcc.Client
	}{
		name:     d.Get("name").(string),
		clientId: manager.ClientID,
		client:   nil,
	}

	if fields.clientId != "" {
		fields.client, err = manager.GetClient(fields.clientId)
		if err != nil {
			return diag.Errorf("[ERROR-002] crash via getting client: %s", err)
		}
	} else {
		allClients, err := manager.GetClients()
		if err != nil {
			return diag.Errorf("[ERROR-002] crash via getting client: %s", err)
		}
		if len(allClients) == 0 {
			return diag.Errorf("[ERROR-002] there are no available clients")
		}
		if len(allClients) > 1 {
			return diag.Errorf("[ERROR-002] more than one client available for you")
		}

		fields.client = allClients[0]
	}

	project := bcc.NewProject(fields.name)
	project.Tags = unmarshalTagNames(d.Get("tags"))
	log.Printf("[DEBUG] Project create request: %#v", project)

	if err = fields.client.CreateProject(&project); err != nil {
		return diag.Errorf("[ERROR-002] crash via creating project: %s", err)
	}
	project.WaitLock()

	d.SetId(project.ID)
	log.Printf("[INFO] Project created, ID: %s", d.Id())

	return resourceProjectRead(ctx, d, meta)
}

func resourceProjectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	project, err := manager.GetProject(d.Id())
	if err != nil {
		resourceReadCheck(d, err, "[ERROR-002]:")
	}

	fields := map[string]interface{}{
		"name": project.Name,
		"tags": marshalTagNames(project.Tags),
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-002] crash via set attrs: %s", err)
	}

	return nil
}

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	project, err := manager.GetProject(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-002] crash via getting project: %s", err)
	}

	if d.HasChange("name") {
		project.Name = d.Get("name").(string)
	}
	if d.HasChange("tags") {
		project.Tags = unmarshalTagNames(d.Get("tags"))
	}
	if err = project.Update(); err != nil {
		return diag.Errorf("[ERROR-002] crash via update project: %s", err)
	}
	project.WaitLock()

	return resourceProjectRead(ctx, d, meta)
}

func resourceProjectDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	project, err := manager.GetProject(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-002] crash via getting project: %s", err)
	}

	if err = project.Delete(); err != nil {
		return diag.Errorf("[ERROR-002] crash via deleting project: %s", err)
	}
	project.WaitLock()

	return nil
}

func resourceProjectImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()
	project, err := manager.GetProject(d.Id())
	if err != nil {
		return nil, fmt.Errorf("[ERROR-002] crash via getting project: %s", err)
	}

	d.SetId(project.ID)

	return []*schema.ResourceData{d}, nil
}

package bcc_terraform

import (
	"context"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRouter() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextDataRouter()
	args.injectContextGetRouter()

	return &schema.Resource{
		ReadContext: dataSourceRouterRead,
		Schema:      args,
	}
}

func dataSourceRouterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-024] crash via getting router: %s", err)
	}

	var router *bcc.Router
	if target == "id" {
		routerId := d.Get("id").(string)
		router, err = manager.GetRouter(routerId)
		if err != nil {
			return diag.Errorf("[ERROR-024] crash via getting router by id: %s", err)
		}
	} else {
		router, err = GetRouterByName(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-024] crash via getting router by name: %s", err)
		}
	}

	ports := make([]*string, len(router.Ports))
	for i, port := range router.Ports {
		ports[i] = &port.ID
	}

	routes := make([]map[string]interface{}, len(router.Routes))
	for i, route := range router.Routes {
		routes[i] = map[string]interface{}{
			"destination": route.Destination,
			"next_hop":    route.NextHop,
		}
	}

	fields := map[string]interface{}{
		"id":          router.ID,
		"name":        router.Name,
		"is_default":  router.IsDefault,
		"routes":      routes,
		"ports":       ports,
		"vdc_id":      router.Vdc.ID,
		"floating":    false,
		"floating_id": "",
		"tags":        marshalTagNames(router.Tags),
	}

	if router.Floating != nil {
		fields["floating"] = true
		fields["floating_id"] = router.Floating.ID
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-024] crash via set attrs: %s", err)
	}

	return nil
}

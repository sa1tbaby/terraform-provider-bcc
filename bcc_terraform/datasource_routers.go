package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceRouters() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextDataRouterList()

	return &schema.Resource{
		ReadContext: dataSourceRoutersRead,
		Schema:      args,
	}
}

func dataSourceRoutersRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-025] crash via get vdc: %s", err)
	}

	routerList, err := vdc.GetRouters()
	if err != nil {
		return diag.Errorf("[ERROR-025] crash via getting routers: %s", err)
	}

	routersMap := make([]map[string]interface{}, len(routerList))
	for i, router := range routerList {

		ports := make([]string, len(router.Ports))
		for j, port := range router.Ports {
			ports[j] = port.ID
		}

		routes := make([]map[string]interface{}, len(router.Routes))
		for j, route := range router.Routes {
			routes[j] = map[string]interface{}{
				"destination": route.Destination,
				"next_hop":    route.NextHop,
			}
		}

		routersMap[i] = map[string]interface{}{
			"id":          router.ID,
			"name":        router.Name,
			"is_default":  router.IsDefault,
			"ports":       ports,
			"routes":      routes,
			"floating":    false,
			"floating_id": "",
			"tags":        marshalTagNames(router.Tags),
		}

		if router.Floating != nil {
			routersMap[i]["floating"] = true
			routersMap[i]["floating_id"] = router.Floating.IpAddress
		}
	}

	hash, err := hashstructure.Hash(routerList, hashstructure.FormatV2, nil)
	if err != nil {
		return diag.Errorf("[ERROR-025] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":      fmt.Sprintf("routers/%d", hash),
		"vdc_id":  vdc.ID,
		"routers": routersMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-025] crash via set attrs: %s", err)
	}

	return nil
}

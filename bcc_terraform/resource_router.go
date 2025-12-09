package bcc_terraform

import (
	"context"
	"fmt"
	"log"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRouter() *schema.Resource {
	args := Defaults()
	args.injectContextVdcById()
	args.injectCreateRouter()

	return &schema.Resource{
		CreateContext: resourceRouterCreate,
		ReadContext:   resourceRouterRead,
		UpdateContext: resourceRouterUpdate,
		DeleteContext: resourceRouterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceRouterImport,
		},
		Schema: args,
	}
}

func resourceRouterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("ports: Error getting Ports from vdc: %s", err)
	}
	if _, ok := d.GetOk("ports"); !ok {
		return diag.Errorf("ports: Error You should setup a port for non default routers")
	}

	config := struct {
		Name      string
		VdcId     string
		IsDefault bool
		System    bool
		Floating  bool
		portsIds  []string
		Tags      []bcc.Tag
	}{
		Name:      d.Get("name").(string),
		VdcId:     d.Get("vdc_id").(string),
		IsDefault: d.Get("is_default").(bool),
		System:    d.Get("system").(bool),
		Floating:  d.Get("floating").(bool),
		portsIds:  d.Get("ports").([]string),
		Tags:      unmarshalTagNames(d.Get("tags")),
	}

	router := bcc.NewRouter(config.Name, config.IsDefault)
	router.Tags = config.Tags
	router.Vdc.ID = vdc.ID

	if config.Floating {
		floatingIpStr := "RANDOM_FIP"
		router.Floating = &bcc.Port{IpAddress: &floatingIpStr}
	}

	for _, portId := range config.portsIds {
		port, err := manager.GetPort(portId)
		if err != nil {
			return diag.FromErr(err)
		}
		router.Ports = append(router.Ports, port)
	}

	log.Printf("[DEBUG] Router create request: %#v", router)

	if err = vdc.WaitLock(); err != nil {
		return diag.FromErr(err)
	}
	if err = vdc.CreateRouter(&router); err != nil {
		return diag.Errorf("Error creating Router: %s", err)
	}
	if err = router.WaitLock(); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(router.ID)
	d.Set("system", config.System)
	log.Printf("[INFO] Router created, ID: %s", router.ID)

	return resourceRouterRead(ctx, d, meta)
}

func resourceRouterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) (diagErr diag.Diagnostics) {
	manager := meta.(*CombinedConfig).Manager()
	router, err := manager.GetRouter(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("id: Error getting Router: %s", err)
		}
	}

	ports := make([]*string, len(router.Ports))
	for i, port := range router.Ports {
		ports[i] = &port.ID
	}

	fields := map[string]interface{}{
		"name":        router.Name,
		"floating":    router.Floating != nil,
		"is_default":  router.IsDefault,
		"floating_id": "",
		"ports":       ports,
		"vdc_id":      router.Vdc.ID,
		"tags":        marshalTagNames(router.Tags),
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.FromErr(err)
	}

	if router.Floating != nil {
		d.Set("floating_id", router.Floating.ID)
	}

	return nil
}

func resourceRouterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	router, err := manager.GetRouter(d.Id())
	if err != nil {
		return diag.Errorf("id: Error getting Router: %s", err)
	}
	shouldUpdate := false
	if d.HasChange("name") {
		router.Name = d.Get("name").(string)
		shouldUpdate = true
	}
	if d.HasChange("tags") {
		router.Tags = unmarshalTagNames(d.Get("tags"))
		shouldUpdate = true
	}
	if d.HasChange("is_default") {
		router.IsDefault = d.Get("is_default").(bool)
		shouldUpdate = true
	}
	if shouldUpdate {
		if err := router.Update(); err != nil {
			return diag.Errorf("error on router's update %s", err)
		}
	}

	if err := syncFloating(d, router); err != nil {
		return diag.FromErr(err)
	}

	// Disconnect ports and connect new
	err = syncRouterPorts(d, manager, router)
	if err != nil {
		return diag.FromErr(err)
	}
	router.WaitLock()

	return resourceRouterRead(ctx, d, meta)
}

func resourceRouterImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	router, err := manager.GetRouter(d.Id())
	if err != nil {
		d.SetId("")
		return nil, err
	}

	d.SetId(router.ID)

	return []*schema.ResourceData{d}, nil

}

func resourceRouterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	portsIds := d.Get("ports").(*schema.Set).List()
	routerId := d.Id()
	router, err := manager.GetRouter(routerId)
	if err != nil {
		return diag.Errorf("id: Error getting Router: %s", err)
	}

	// Disconnect custom ports from system router
	if d.Get("system").(bool) {

		for _, port := range router.Ports {
			network, err := manager.GetNetwork(port.Network.ID)
			if err != nil {
				return diag.FromErr(err)
			}
			if !network.IsDefault {
				err = router.DisconnectPort(port)
				if err != nil {
					return diag.FromErr(err)
				}
			}
			if router.Floating == nil {
				router.Floating = &bcc.Port{ID: "RANDOM_FIP"}
				if err = repeatOnError(router.Update, router); err != nil {
					return diag.Errorf("ERROR: Can't return router to default state: %s", err)
				}
			}
		}

		return nil
	}

	// Detach ports and delete custom router
	for _, portId := range portsIds {
		port, err := manager.GetPort(portId.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		err = router.DisconnectPort(port)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err = repeatOnError(router.Delete, router); err != nil {
		return diag.Errorf("Error deleting Router: %s", err)
	}
	router.WaitLock()

	d.SetId("")
	log.Printf("[INFO] Router deleted, ID: %s", routerId)

	return nil
}

func syncRouterPorts(d *schema.ResourceData, manager *bcc.Manager, router *bcc.Router) (err error) {
	portsIds := d.Get("ports").(*schema.Set).List()
	router_id := d.Id()

	for _, port := range router.Ports {
		found := false
		for _, portId := range portsIds {
			if portId == port.ID {
				found = true
				break
			}
		}

		if !found {
			if port.Connected != nil && port.Connected.ID == router_id {
				log.Printf("Port %s found on vm and not mentioned in the state."+
					" Port will be detached", port.ID)
				router.DisconnectPort(port)
				port.WaitLock()
			}
		}
	}

	for _, portId := range portsIds {
		found := false
		for _, port := range router.Ports {
			if port.ID == portId {
				found = true
				break
			}
		}

		if !found {
			port, err := manager.GetPort(portId.(string))
			if err != nil {
				return fmt.Errorf("ports: getting Port from vdc")
			}
			if port.Connected != nil && port.Connected.Type == "vm_int" {
				return fmt.Errorf("ports: Unable to bind a port that is already connected to the server")
			}
			if port.Connected != nil && port.Connected.ID != router_id {
				router.DisconnectPort(port)
				port.WaitLock()
			}
			port, err = manager.GetPort(portId.(string))
			if err != nil {
				return fmt.Errorf("ERROR: Cannot get port `%s`: %s", portId, err)
			}
			log.Printf("Port `%s` will be Attached", port.ID)
			if err := router.ConnectPort(port, true); err != nil {
				return fmt.Errorf("ERROR: Cannot attach port `%s`: %s", port.ID, err)
			}
		}
	}

	return
}

func syncFloating(d *schema.ResourceData, router *bcc.Router) (err error) {
	floating := d.Get("floating")

	if floating.(bool) && (router.Floating == nil) {
		// add floating if it was removed
		router.Floating = &bcc.Port{ID: "RANDOM_FIP"}
		if err = repeatOnError(router.Update, router); err != nil {
			return fmt.Errorf("ERROR: Can't update Router: %s", err)
		}
		d.Set("floating", true)
		d.Set("floating_id", router.Floating.ID)
	} else if !floating.(bool) && (router.Floating != nil) {
		// remove floating if needed
		router.Floating = nil

		if err = repeatOnError(router.Update, router); err != nil {
			return fmt.Errorf("ERROR: Can't update Router: %s", err)
		}
	} else if floating.(bool) && (router.Floating != nil) {
		d.Set("floating", true)
		d.Set("floating_id", router.Floating.ID)
	}
	return
}

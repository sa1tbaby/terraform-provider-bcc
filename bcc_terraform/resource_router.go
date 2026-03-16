package bcc_terraform

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRouter() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextResourceRouter()

	return &schema.Resource{
		CreateContext: resourceRouterCreate,
		ReadContext:   resourceRouterRead,
		UpdateContext: resourceRouterUpdate,
		DeleteContext: resourceRouterDelete,
		CustomizeDiff: resourceRouterCustomizeDiff,
		Importer: &schema.ResourceImporter{
			StateContext: resourceRouterImport,
		},
		Schema: args,
	}
}
func resourceRouterCustomizeDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if d.HasChange("floating") {
		d.SetNewComputed("floating_id")
	}
	return nil
}

func resourceRouterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-044] crash via getting vdc: %s", err)
	}
	if _, ok := d.GetOk("ports"); !ok {
		return diag.Errorf("[ERROR-044] You should setup a port for non default routers")
	}

	fields := struct {
		name       string
		isDefault  bool
		system     bool
		ports      []interface{}
		routes     []interface{}
		floating   bool
		floatingIp *string
	}{
		name:       d.Get("name").(string),
		isDefault:  d.Get("is_default").(bool),
		system:     d.Get("system").(bool),
		ports:      d.Get("ports").([]interface{}),
		routes:     d.Get("routes").([]interface{}),
		floating:   d.Get("floating").(bool),
		floatingIp: nil,
	}

	if d.Get("floating").(bool) {
		v := "RANDOM_FIP"
		fields.floatingIp = &v
	}

	router := bcc.NewRouter(fields.name, fields.floatingIp, vdc.ID)
	router.Tags = unmarshalTagNames(d.Get("tags"))
	router.IsDefault = fields.isDefault

	for _, portId := range fields.ports {
		port, err := manager.GetPort(portId.(string))
		if err != nil {
			return diag.Errorf("[ERROR-044] crash via getting port: %s", err)
		}
		router.Ports = append(router.Ports, port)
	}

	if strings.EqualFold(vdc.Hypervisor.Type, "Vmware") {
		for _, route := range fields.routes {
			r := route.(map[string]interface{})
			router.Routes = append(router.Routes, &bcc.Route{
				Destination: r["destination"].(string),
				NextHop:     r["next_hop"].(string),
			})
		}
	} else {
		if len(fields.routes) > 0 {
			return diag.Errorf("[ERROR-044] crash via Routes are not supported for %s hypervisor", vdc.Hypervisor.Type)
		}
	}

	log.Printf("[DEBUG] Router create request: %#v", router)

	if err = vdc.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-044] crash via waitlock for vdc: %s", err)
	}
	if err = vdc.CreateRouter(&router); err != nil {
		return diag.Errorf("[ERROR-044] crash via creating Router: %s", err)
	}
	if err = router.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-044] crash via waitlock for router: %s", err)
	}

	d.SetId(router.ID)
	d.Set("system", fields.system)
	log.Printf("[INFO] Router created, ID: %s", router.ID)

	return resourceRouterRead(ctx, d, meta)
}

func resourceRouterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) (diagErr diag.Diagnostics) {
	manager := meta.(*CombinedConfig).Manager()
	router, err := manager.GetRouter(d.Id())
	if err != nil {
		return resourceReadCheck(d, err, "[ERROR-044]")
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
		"name":        router.Name,
		"is_default":  router.IsDefault,
		"routes":      routes,
		"ports":       ports,
		"vdc_id":      router.Vdc.ID,
		"tags":        marshalTagNames(router.Tags),
		"floating":    false,
		"floating_id": "",
	}

	if router.Floating != nil {
		fields["floating"] = true
		fields["floating_id"] = router.Floating.ID
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-044] crash via set attrs: %s", err)
	}

	return nil
}

func resourceRouterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	router, err := manager.GetRouter(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-044] crash via getting Router: %s", err)
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
	if d.HasChange("routes") {
		shouldUpdate = true
		addRoutes := make(map[string]*bcc.Route)

		if strings.EqualFold(router.Vdc.Hypervisor.Type, "Vmware") {
			for _, route := range d.Get("routes").([]interface{}) {
				r := route.(map[string]interface{})
				addRoutes[r["next_hop"].(string)] = &bcc.Route{
					Destination: r["destination"].(string),
					NextHop:     r["next_hop"].(string),
				}
			}
			for _, route := range router.Routes {
				if _, exist := addRoutes[route.NextHop]; exist {
					delete(addRoutes, route.NextHop)
				} else {
					if err := route.Delete(); err != nil {
						return diag.Errorf("[ERROR-044] crash via deleting route by 'id':%s", route.ID)
					}
				}
			}
			for _, route := range addRoutes {
				if err := router.CreateRoute(route); err != nil {
					return diag.Errorf("[ERROR-044] crash via creating route, for router 'id':%s", router.ID)
				}
			}
		} else {
			if len(d.Get("routes").([]interface{})) > 0 {
				return diag.Errorf("[ERROR-044] crash via: Routes are not supported for %s hypervisor", router.Vdc.Hypervisor.Type)
			}
		}
	}

	if shouldUpdate {
		if err = repeatOnError(router.Update, router); err != nil {
			return diag.Errorf("[ERROR-044] crash via router's update %s", err)
		}
		router.WaitLock()
	}

	// Disconnect ports and connect new
	err = syncRouterPorts(d, manager, router)
	if err != nil {
		return diag.Errorf("[ERROR-044] %s", err)
	}
	router.WaitLock()

	if err := syncFloating(d, router); err != nil {
		return diag.Errorf("[ERROR-044] %s", err)
	}
	router.WaitLock()

	return resourceRouterRead(ctx, d, meta)
}

func resourceRouterImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	router, err := manager.GetRouter(d.Id())
	if err != nil {
		return nil, err
	}

	d.SetId(router.ID)

	return []*schema.ResourceData{d}, nil

}

func resourceRouterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	portsIds := d.Get("ports").([]interface{})
	routerId := d.Id()
	router, err := manager.GetRouter(routerId)
	if err != nil {
		return diag.Errorf("[ERROR-044] crash via getting Router: %s", err)
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
					return diag.Errorf("[ERROR-044] Can't return router to default state: %s", err)
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
		return diag.Errorf("[ERROR-044] crash via deleting Router: %s", err)
	}

	router.WaitLock()
	log.Printf("[INFO] Router deleted, ID: %s", routerId)

	return nil
}

func syncRouterPorts(d *schema.ResourceData, manager *bcc.Manager, router *bcc.Router) (err error) {
	portsIds := d.Get("ports").([]interface{})
	routerId := d.Id()

	for _, port := range router.Ports {
		found := false
		for _, portId := range portsIds {
			if strings.EqualFold(portId.(string), port.ID) {
				found = true
				break
			}
		}

		if !found {
			if port.Connected != nil && strings.EqualFold(port.Connected.ID, routerId) {
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
				return fmt.Errorf("crash via getting Port ")
			}
			if port.Connected != nil && port.Connected.Type == "vm_int" {
				return fmt.Errorf("unable to bind a port that is already connected to the server")
			}
			if port.Connected != nil && port.Connected.ID != routerId {
				router.DisconnectPort(port)
				port.WaitLock()
			}
			port, err = manager.GetPort(portId.(string))
			if err != nil {
				return fmt.Errorf("cannot get port `%s`: %s", portId, err)
			}
			log.Printf("Port `%s` will be Attached", port.ID)
			if err := router.ConnectPort(port, true); err != nil {
				return fmt.Errorf("cannot attach port `%s`: %s", port.ID, err)
			}
		}
	}

	return
}

func syncFloating(d *schema.ResourceData, router *bcc.Router) (err error) {
	oldFloating, newFloating := d.GetChange("floating")

	if !oldFloating.(bool) && newFloating.(bool) {
		router.Floating = &bcc.Port{ID: "RANDOM_FIP"}
	} else if oldFloating.(bool) && !newFloating.(bool) {
		router.Floating = nil
	}

	if err = repeatOnError(router.Update, router); err != nil {
		return fmt.Errorf("crash via updating floating for router: %s", err)
	}
	router.WaitLock()
	if err = d.Set("floating", router.Floating != nil); err != nil {
		return fmt.Errorf("crash via setting floating: %s", err)
	}

	return nil
}

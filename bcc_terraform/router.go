package bcc_terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func (args *Arguments) injectContextGetRouter() {
	args.merge(Arguments{
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Router name",
		},
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Router identifier",
		},
	})
}

func (args *Arguments) injectContextDataRouter() {
	routes := Defaults()
	routes.injectCmpRouterRoutes()

	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "id of the Router",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "name of the Router",
		},
		"is_default": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Set if this is default router",
		},
		"floating": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Enable floating ip for the Vm",
		},
		"floating_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Floating id address.",
		},
		"ports": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "List of Ports connected to the router",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"system": {
			Type:        schema.TypeBool,
			Computed:    true,
			Deprecated:  "param has been removed",
			Description: "Determinate if router is system.",
		},
		"routes": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "List of Routes connected to the router",
			Elem: &schema.Resource{
				Schema: routes,
			},
		},
		"tags": newTagNamesDataSchema("tags of the router"),
	})
}

func (args *Arguments) injectContextDataRouterList() {
	Router := Defaults()
	Router.injectContextDataRouter()

	args.merge(Arguments{
		"routers": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: Router,
			},
		},
	})
}

func (args *Arguments) injectReqRouterRoutes() {
	args.merge(Arguments{
		"destination": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "route destination",
		},
		"next_hop": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "next for before destination",
		},
	})
}

func (args *Arguments) injectCmpRouterRoutes() {
	args.merge(Arguments{
		"destination": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "route destination",
		},
		"next_hop": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "next for before destination",
		},
	})
}

func (args *Arguments) injectContextResourceRouter() {
	routes := Defaults()
	routes.injectReqRouterRoutes()

	args.merge(Arguments{
		"name": {
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: validation.All(
				validation.NoZeroValues,
				validation.StringLenBetween(1, 100),
			),
			Description: "Name of the Router",
		},
		"is_default": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Set if this is default router",
		},
		"floating": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Enable floating ip for the Vm",
		},
		"floating_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Floating id address.",
		},
		"ports": {
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			MinItems:    1,
			MaxItems:    10,
			Description: "List of Ports connected to the router",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"system": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Deprecated:  "param has been removed",
			Description: "Determinate if router is system.",
		},
		"routes": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "List of Routes connected to the router",
			Elem: &schema.Resource{
				Schema: routes,
			},
		},
		"tags": newTagNamesResourceSchema("tags of the router"),
	})
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

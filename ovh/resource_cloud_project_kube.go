package ovh

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/ovh/go-ovh/ovh"
	"github.com/ovh/terraform-provider-ovh/ovh/helpers"
)

const (
	kubeClusterNameKey                        = "name"
	kubeClusterPrivateNetworkIDKey            = "private_network_id"
	kubeClusterPrivateNetworkConfigurationKey = "private_network_configuration"
	kubeClusterUpdatePolicyKey                = "update_policy"
	kubeClusterVersionKey                     = "version"

	kubeClusterProxyModeKey = "kube_proxy_mode"

	kubeClusterCustomizationKey = "customization"

	kubeClusterCustomizationApiServerKey = "customization_apiserver"
	kubeClusterCustomizationKubeProxyKey = "customization_kube_proxy"
)

func resourceCloudProjectKube() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudProjectKubeCreate,
		Read:   resourceCloudProjectKubeRead,
		Delete: resourceCloudProjectKubeDelete,
		Update: resourceCloudProjectKubeUpdate,

		Importer: &schema.ResourceImporter{
			State: resourceCloudProjectKubeImportState,
		},

		Schema: map[string]*schema.Schema{
			"service_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_CLOUD_PROJECT_SERVICE", nil),
			},
			kubeClusterNameKey: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			kubeClusterVersionKey: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: false,
			},
			kubeClusterCustomizationApiServerKey: {
				Type:     schema.TypeSet,
				Computed: true,
				Optional: true,
				// Required: true,
				ForceNew: false,
				// MaxItems: 1,
				Set: CustomSchemaSetFunc(false),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admissionplugins": {
							Type:     schema.TypeSet,
							Computed: true,
							Optional: true,
							// Required: true,
							ForceNew: false,
							// MaxItems: 1,
							Set: CustomSchemaSetFunc(false),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeList,
										Computed: true,
										Optional: true,
										// Required: true,
										ForceNew: false,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"disabled": {
										Type:     schema.TypeList,
										Computed: true,
										Optional: true,
										// Required: true,
										ForceNew: false,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
					},
				},
			},

			kubeClusterCustomizationKubeProxyKey: {
				Type:     schema.TypeSet,
				Computed: false,
				Optional: true,
				ForceNew: false,
				MaxItems: 1,
				// Set:      CustomSchemaSetFunc(true),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"iptables": {
							Type:     schema.TypeSet,
							Computed: false,
							Optional: true,
							ForceNew: false,
							MaxItems: 1,
							Set:      CustomIPtablesSchemaSetFunc(false),

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"min_sync_period": {
										Type:     schema.TypeString,
										Computed: false,
										Optional: true,
										ForceNew: false,
									},
									"sync_period": {
										Type:     schema.TypeString,
										Computed: false,
										Optional: true,
										ForceNew: false,
									},
								},
							},
						},
						"ipvs": {
							Type:     schema.TypeSet,
							Computed: false,
							Optional: true,
							ForceNew: false,
							MaxItems: 1,
							Set:      CustomIPVSSchemaSetFunc(true),
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"min_sync_period": {
										Type:     schema.TypeString,
										Computed: false,
										Optional: true,
										ForceNew: false,
									},
									"sync_period": {
										Type:     schema.TypeString,
										Computed: false,
										Optional: true,
										ForceNew: false,
									},
									"scheduler": {
										Type:     schema.TypeString,
										Computed: false,
										Optional: true,
										ForceNew: false,
									},
									"tcp_fin_timeout": {
										Type:     schema.TypeString,
										Computed: false,
										Optional: true,
										ForceNew: false,
									},
									"tcp_timeout": {
										Type:     schema.TypeString,
										Computed: false,
										Optional: true,
										ForceNew: false,
									},
									"udp_timeout": {
										Type:             schema.TypeString,
										Computed:         false,
										Optional:         true,
										ForceNew:         false,
										DiffSuppressFunc: DiffDurationRfc3339,
									},
								},
							},
						},
					},
				},
			},

			kubeClusterPrivateNetworkIDKey: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			kubeClusterProxyModeKey: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			kubeClusterPrivateNetworkConfigurationKey: {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default_vrack_gateway": {
							Required:    true,
							Type:        schema.TypeString,
							Description: "If defined, all egress traffic will be routed towards this IP address, which should belong to the private network. Empty string means disabled.",
						},
						"private_network_routing_as_default": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Defines whether routing should default to using the nodes' private interface, instead of their public interface. Default is false.",
						},
					},
				},
			},
			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// Computed
			"control_plane_is_up_to_date": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"is_up_to_date": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"next_upgrade_versions": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"nodes_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			kubeClusterUpdatePolicyKey: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
			"url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"kubeconfig": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func CustomIPtablesSchemaSetFunc(output bool) schema.SchemaSetFunc {
	return func(i interface{}) int {
		if i.(map[string]interface{})["min_sync_period"] == "P0D" {
			i.(map[string]interface{})["min_sync_period"] = "PT0S"
		}
		if i.(map[string]interface{})["sync_period"] == "P0D" {
			i.(map[string]interface{})["sync_period"] = "PT0S"
		}

		out := fmt.Sprintf("%#v", i)
		hash := schema.HashString(out)
		if output {
			log.Printf(">>>>>>>%d %s\n", hash, out)
		}
		return hash
	}
}

func CustomIPVSSchemaSetFunc(output bool) schema.SchemaSetFunc {
	return func(i interface{}) int {
		if i.(map[string]interface{})["min_sync_period"] == "P0D" {
			i.(map[string]interface{})["min_sync_period"] = "PT0S"
		}
		if i.(map[string]interface{})["sync_period"] == "P0D" {
			i.(map[string]interface{})["sync_period"] = "PT0S"
		}
		if i.(map[string]interface{})["tcp_fin_timeout"] == "P0D" {
			i.(map[string]interface{})["tcp_fin_timeout"] = "PT0S"
		}
		if i.(map[string]interface{})["tcp_timeout"] == "P0D" {
			i.(map[string]interface{})["tcp_timeout"] = "PT0S"
		}
		if i.(map[string]interface{})["udp_timeout"] == "P0D" {
			i.(map[string]interface{})["udp_timeout"] = "PT0S"
		}

		out := fmt.Sprintf("%#v", i)
		hash := schema.HashString(out)
		if output {
			log.Printf(">>>>>>>%d %s\n", hash, out)
		}
		return hash
	}
}

func CustomSchemaSetFunc(output bool) schema.SchemaSetFunc {
	return func(i interface{}) int {
		out := fmt.Sprintf("%#v", i)
		hash := schema.HashString(out)
		if output {
			log.Printf(">>>>>>>%d %s\n", hash, out)
		}
		return hash
	}
}

func resourceCloudProjectKubeImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	givenId := d.Id()
	splitId := strings.SplitN(givenId, "/", 2)
	if len(splitId) != 2 {
		return nil, fmt.Errorf("import Id is not service_name/kubeid formatted")
	}
	serviceName := splitId[0]
	id := splitId[1]
	d.SetId(id)
	d.Set("service_name", serviceName)

	// add kubeconfig in state
	kubeConfig, err := getKubeconfig(meta.(*Config), serviceName, d.Id())
	if err != nil {
		return nil, err
	}
	d.Set("kubeconfig", kubeConfig)

	results := make([]*schema.ResourceData, 1)
	results[0] = d
	return results, nil
}

func resourceCloudProjectKubeCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceName := d.Get("service_name").(string)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube", serviceName)
	params := (&CloudProjectKubeCreateOpts{}).FromResource(d)
	res := &CloudProjectKubeResponse{}

	log.Printf("[DEBUG] Will create kube: %+v", params)
	err := config.OVHClient.Post(endpoint, params, res)
	if err != nil {
		return fmt.Errorf("calling Post %s with params %s:\n\t %w", endpoint, params, err)
	}

	// This is a fix for a weird bug where the kube is not immediately available on API
	log.Printf("[DEBUG] Waiting for kube %s to be available", res.Id)
	endpoint = fmt.Sprintf("/cloud/project/%s/kube/%s", serviceName, res.Id)
	err = helpers.WaitAvailable(config.OVHClient, endpoint, 2*time.Minute)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting for kube %s to be READY", res.Id)
	err = waitForCloudProjectKubeReady(config.OVHClient, serviceName, res.Id, []string{"INSTALLING"}, []string{"READY"})
	if err != nil {
		return fmt.Errorf("timeout while waiting kube %s to be READY: %w", res.Id, err)
	}
	log.Printf("[DEBUG] kube %s is READY", res.Id)

	d.SetId(res.Id)

	return resourceCloudProjectKubeRead(d, meta)
}

func resourceCloudProjectKubeRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceName := d.Get("service_name").(string)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s", serviceName, d.Id())
	res := &CloudProjectKubeResponse{}

	log.Printf("[DEBUG] Will read kube %s from project: %s", d.Id(), serviceName)
	if err := config.OVHClient.Get(endpoint, res); err != nil {
		return helpers.CheckDeleted(d, err, endpoint)
	}
	for k, v := range res.ToMap() {
		if k != "id" {
			d.Set(k, v)
		} else {
			d.SetId(fmt.Sprint(v))
		}
	}

	if d.IsNewResource() {
		kubeConfig, err := getKubeconfig(config, serviceName, res.Id)
		if err != nil {
			return err
		}
		d.Set("kubeconfig", kubeConfig)
	}

	log.Printf("[DEBUG] Read kube %+v", res)
	return nil
}

func resourceCloudProjectKubeDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceName := d.Get("service_name").(string)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s", serviceName, d.Id())

	log.Printf("[DEBUG] Will delete kube %s from project: %s", d.Id(), serviceName)
	err := config.OVHClient.Delete(endpoint, nil)
	if err != nil {
		return helpers.CheckDeleted(d, err, endpoint)
	}

	log.Printf("[DEBUG] Waiting for kube %s to be DELETED", d.Id())
	err = waitForCloudProjectKubeDeleted(config.OVHClient, serviceName, d.Id())
	if err != nil {
		return fmt.Errorf("timeout while waiting kube %s to be DELETED: %w", d.Id(), err)
	}
	log.Printf("[DEBUG] kube %s is DELETED", d.Id())

	d.SetId("")

	return nil
}

func resourceCloudProjectKubeUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceName := d.Get("service_name").(string)

	if d.HasChange(kubeClusterCustomizationApiServerKey) || d.HasChange(kubeClusterCustomizationKubeProxyKey) {
		_, apiServerAdmissionPlugins := d.GetChange(kubeClusterCustomizationApiServerKey)
		_, kubeProxyCustomization := d.GetChange(kubeClusterCustomizationKubeProxyKey)

		customization := loadCustomization(apiServerAdmissionPlugins, kubeProxyCustomization)

		params := &CloudProjectKubeUpdateCustomizationOpts{
			APIServer: customization.APIServer,
			KubeProxy: customization.KubeProxy,
		}

		endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/customization", serviceName, d.Id())
		err := config.OVHClient.Put(endpoint, params, nil)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] Waiting for kube %s to be READY", d.Id())
		err = waitForCloudProjectKubeReady(config.OVHClient, serviceName, d.Id(), []string{"REDEPLOYING", "RESETTING"}, []string{"READY"})
		if err != nil {
			return fmt.Errorf("timeout while waiting kube %s to be READY: %w", d.Id(), err)
		}
		log.Printf("[DEBUG] kube %s is READY", d.Id())
	}

	if d.HasChange(kubeClusterVersionKey) {
		oldValueI, newValueI := d.GetChange(kubeClusterVersionKey)

		oldValue := oldValueI.(string)
		newValue := newValueI.(string)

		log.Printf("[DEBUG] cluster version change from %s to %s", oldValue, newValue)

		oldVersion, err := version.NewVersion(oldValueI.(string))
		if err != nil {
			return fmt.Errorf("version %s does not match a semver", oldValue)
		}
		newVersion, err := version.NewVersion(newValueI.(string))
		if err != nil {
			return fmt.Errorf("version %s does not match a semver", newValue)
		}

		oldVersionSegments := oldVersion.Segments()
		newVersionSegments := newVersion.Segments()

		if oldVersionSegments[0] != 1 || newVersionSegments[0] != 1 {
			return fmt.Errorf("the only supported major version is 1")
		}
		if len(oldVersionSegments) < 2 || len(newVersionSegments) < 2 {
			log.Printf("[DEBUG] old version segments: %#v new version segments: %#v", oldVersionSegments, newVersionSegments)
			return fmt.Errorf("the version should only specify the major and minor versions (e.g. \\\"1.20\\\")")
		}

		if newVersion.LessThan(oldVersion) {
			return fmt.Errorf("cannot downgrade cluster from %s to %s", oldValue, newValue)
		}

		if oldVersionSegments[1]+1 != newVersionSegments[1] {
			return fmt.Errorf("cannot upgrade cluster from %s to %s, only next minor version is authorized", oldValue, newValue)
		}

		endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/update", serviceName, d.Id())
		err = config.OVHClient.Post(endpoint, CloudProjectKubeUpdateOpts{
			Strategy: "NEXT_MINOR",
		}, nil)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] Waiting for kube %s to be READY", d.Id())
		err = waitForCloudProjectKubeReady(config.OVHClient, serviceName, d.Id(), []string{"UPDATING", "REDEPLOYING", "RESETTING"}, []string{"READY"})
		if err != nil {
			return fmt.Errorf("timeout while waiting kube %s to be READY: %w", d.Id(), err)
		}
		log.Printf("[DEBUG] kube %s is READY", d.Id())
	}

	if d.HasChange(kubeClusterUpdatePolicyKey) {
		_, newValue := d.GetChange(kubeClusterUpdatePolicyKey)
		value := newValue.(string)

		endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/updatePolicy", serviceName, d.Id())
		err := config.OVHClient.Put(endpoint, CloudProjectKubeUpdatePolicyOpts{
			UpdatePolicy: value,
		}, nil)
		if err != nil {
			return err
		}
	}

	if d.HasChange(kubeClusterNameKey) {
		_, newValue := d.GetChange(kubeClusterNameKey)
		value := newValue.(string)

		endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s", serviceName, d.Id())
		err := config.OVHClient.Put(endpoint, CloudProjectKubePutOpts{
			Name: &value,
		}, nil)
		if err != nil {
			return err
		}
	}

	if d.HasChange(kubeClusterPrivateNetworkConfigurationKey) {
		_, newValue := d.GetChange(kubeClusterPrivateNetworkConfigurationKey)
		pncOutput := privateNetworkConfiguration{}

		pncSet := newValue.(*schema.Set).List()
		for _, pnc := range pncSet {
			mapping := pnc.(map[string]interface{})
			pncOutput.DefaultVrackGateway = mapping["default_vrack_gateway"].(string)
			pncOutput.PrivateNetworkRoutingAsDefault = mapping["private_network_routing_as_default"].(bool)
		}

		endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/privateNetworkConfiguration", serviceName, d.Id())
		err := config.OVHClient.Put(endpoint, CloudProjectKubeUpdatePNCOpts{
			DefaultVrackGateway:            pncOutput.DefaultVrackGateway,
			PrivateNetworkRoutingAsDefault: pncOutput.PrivateNetworkRoutingAsDefault,
		}, nil)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] Waiting for kube %s to be READY", d.Id())
		err = waitForCloudProjectKubeReady(config.OVHClient, serviceName, d.Id(), []string{"REDEPLOYING", "RESETTING"}, []string{"READY"})
		if err != nil {
			return fmt.Errorf("timeout while waiting kube %s to be READY: %w", d.Id(), err)
		}
		log.Printf("[DEBUG] kube %s is READY", d.Id())
	}

	return nil
}

func cloudProjectKubeExists(serviceName, id string, client *ovh.Client) error {
	res := &CloudProjectKubeResponse{}

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s", serviceName, id)
	return client.Get(endpoint, res)
}

func waitForCloudProjectKubeReady(client *ovh.Client, serviceName, kubeId string, pending []string, target []string) error {
	stateConf := &resource.StateChangeConf{
		Pending: pending,
		Target:  target,
		Refresh: func() (interface{}, string, error) {
			res := &CloudProjectKubeResponse{}
			endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s", serviceName, kubeId)
			err := client.Get(endpoint, res)
			if err != nil {
				return res, "", err
			}

			return res, res.Status, nil
		},
		Timeout:    20 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	return err
}

func waitForCloudProjectKubeDeleted(client *ovh.Client, serviceName, kubeId string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{"DELETING"},
		Target:  []string{"DELETED"},
		Refresh: func() (interface{}, string, error) {
			res := &CloudProjectKubeResponse{}
			endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s", serviceName, kubeId)
			err := client.Get(endpoint, res)
			if err != nil {
				if errOvh, ok := err.(*ovh.APIError); ok && errOvh.Code == 404 {
					return res, "DELETED", nil
				} else {
					return res, "", err
				}
			}

			return res, res.Status, nil
		},
		Timeout:    20 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	return err
}

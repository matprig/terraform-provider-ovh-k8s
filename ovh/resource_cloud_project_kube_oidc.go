package ovh

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudProjectKubeOIDC() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudProjectKubeOIDCCreate,
		Read:   resourceCloudProjectKubeOIDCRead,
		Delete: resourceCloudProjectKubeOIDCDelete,
		Update: resourceCloudProjectKubeOIDCUpdate,

		Schema: map[string]*schema.Schema{
			"service_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_CLOUD_PROJECT_SERVICE", nil),
			},
			"kube_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"client_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"issuer_url": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceCloudProjectKubeOIDCCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	serviceName := d.Get("service_name").(string)
	kubeID := d.Get("kube_id").(string)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/openIdConnect", serviceName, kubeID)
	params := (&CloudProjectKubeOIDCCreateOpts{}).FromResource(d)
	res := &CloudProjectKubeOIDCResponse{}

	log.Printf("[DEBUG] Will create OIDC: %+v", params)
	err := config.OVHClient.Post(endpoint, params, res)
	if err != nil {
		return fmt.Errorf("calling Post %s with params %s:\n\t %w", endpoint, params, err)
	}

	d.SetId(kubeID + "-" + params.ClientID + "-" + params.IssuerUrl)

	log.Printf("[DEBUG] Waiting for kube %s to be READY", kubeID)
	err = waitForCloudProjectKubeReady(config.OVHClient, serviceName, kubeID, []string{"REDEPLOYING"}, []string{"READY"})
	if err != nil {
		return fmt.Errorf("timeout while waiting kube %s to be READY: %v", kubeID, err)
	}
	log.Printf("[DEBUG] kube %s is READY", kubeID)

	return resourceCloudProjectKubeOIDCRead(d, meta)
}

func resourceCloudProjectKubeOIDCRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	serviceName := d.Get("service_name").(string)
	kubeID := d.Get("kube_id").(string)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/openIdConnect", serviceName, kubeID)
	res := &CloudProjectKubeOIDCResponse{}

	log.Printf("[DEBUG] Will read oidc from kube %s and project: %s", kubeID, serviceName)
	err := config.OVHClient.Get(endpoint, res)
	if err != nil {
		return fmt.Errorf("calling get %s %w", endpoint, err)
	}
	for k, v := range res.ToMap() {
		if k != "id" {
			d.Set(k, v)
		} else {
			d.SetId(kubeID + "-" + res.ClientID + "-" + res.IssuerUrl)
		}
	}

	log.Printf("[DEBUG] Read kube %+v", res)
	return nil
}

func resourceCloudProjectKubeOIDCUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	serviceName := d.Get("service_name").(string)
	kubeID := d.Get("kube_id").(string)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/openIdConnect", serviceName, kubeID)
	params := (&CloudProjectKubeOIDCUpdateOpts{}).FromResource(d)
	res := &CloudProjectKubeOIDCResponse{}

	log.Printf("[DEBUG] Will update OIDC: %+v", params)
	err := config.OVHClient.Put(endpoint, params, res)
	if err != nil {
		return fmt.Errorf("calling Put %s with params %s:\n\t %w", endpoint, params, err)
	}

	log.Printf("[DEBUG] Waiting for kube %s to be READY", kubeID)
	err = waitForCloudProjectKubeReady(config.OVHClient, serviceName, kubeID, []string{"REDEPLOYING"}, []string{"READY"})
	if err != nil {
		return fmt.Errorf("timeout while waiting kube %s to be READY: %v", kubeID, err)
	}
	log.Printf("[DEBUG] kube %s is READY", kubeID)

	return nil
}

func resourceCloudProjectKubeOIDCDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	serviceName := d.Get("service_name").(string)
	kubeID := d.Get("kube_id").(string)

	endpoint := fmt.Sprintf("/cloud/project/%s/kube/%s/openIdConnect", serviceName, kubeID)

	log.Printf("[DEBUG] Will delete OIDC")
	err := config.OVHClient.Delete(endpoint, nil)
	if err != nil {
		return fmt.Errorf("calling delete %s %w", endpoint, err)
	}

	log.Printf("[DEBUG] Waiting for kube %s to be READY", kubeID)
	err = waitForCloudProjectKubeReady(config.OVHClient, serviceName, kubeID, []string{"REDEPLOYING"}, []string{"READY"})
	if err != nil {
		return fmt.Errorf("timeout while waiting kube %s to be READY: %v", kubeID, err)
	}
	log.Printf("[DEBUG] kube %s is READY", kubeID)

	return nil
}
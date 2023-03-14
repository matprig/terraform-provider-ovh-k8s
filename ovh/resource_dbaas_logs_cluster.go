package ovh

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDbaasLogsCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceDbaasLogsClusterCreate,
		Update: resourceDbaasLogsClusterUpdate,
		Read:   resourceDbaasLogsClusterRead,
		Delete: resourceDbaasLogsClusterDelete,
		Importer: &schema.ResourceImporter{
			State: resourceDbaasLogsClusterImportState,
		},

		Schema: resourceDbaasLogsClusterSchema(),
	}
}

func resourceDbaasLogsClusterSchema() map[string]*schema.Schema {
	schema := map[string]*schema.Schema{
		"service_name": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"archive_allowed_networks": {
			Type: schema.TypeSet,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "Allowed networks for ARCHIVE flow type",
			Optional:    true,
		},
		"direct_input_allowed_networks": {
			Type: schema.TypeSet,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "Allowed networks for DIRECT_INPUT flow type",
			Optional:    true,
		},
		"query_allowed_networks": {
			Type: schema.TypeSet,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "Allowed networks for QUERY flow type",
			Optional:    true,
		},

		// Computed
		"cluster_type": {
			Type:        schema.TypeString,
			Description: "Cluster type",
			Computed:    true,
		},
		"dedicated_input_pem": {
			Type:        schema.TypeString,
			Description: "PEM for dedicated inputs",
			Computed:    true,
			Sensitive:   true,
		},
		"direct_input_pem": {
			Type:        schema.TypeString,
			Description: "PEM for direct inputs",
			Computed:    true,
			Sensitive:   true,
		},
		"hostname": {
			Type:        schema.TypeString,
			Description: "hostname",
			Computed:    true,
		},
		"is_default": {
			Type:        schema.TypeBool,
			Description: "All content generated by given service will be placed on this cluster",
			Computed:    true,
		},
		"is_unlocked": {
			Type:        schema.TypeBool,
			Description: "Allow given service to perform advanced operations on cluster",
			Computed:    true,
		},
		"region": {
			Type:        schema.TypeString,
			Description: "Data center localization",
			Computed:    true,
		},
		// Store ACL before the cluster was managed by terraform
		"initial_archive_allowed_networks": {
			Type: schema.TypeSet,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "Initial allowed networks for ARCHIVE flow type",
			Computed:    true,
			Sensitive:   true,
		},
		"initial_direct_input_allowed_networks": {
			Type: schema.TypeSet,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "Initial allowed networks for DIRECT_INPUT flow type",
			Computed:    true,
			Sensitive:   true,
		},
		"initial_query_allowed_networks": {
			Type: schema.TypeSet,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "Initial allowed networks for QUERY flow type",
			Computed:    true,
			Sensitive:   true,
		},
	}

	return schema
}

func resourceDbaasLogsClusterCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	serviceName := d.Get("service_name").(string)
	cluster_id, err := dbaasGetClusterID(config, serviceName)
	if err != nil {
		return fmt.Errorf("Error fetching info for %s:\n\t %q", serviceName, err)
	}
	d.SetId(cluster_id)

	// Fetch current ACL to restore them as-is when the resource is deleted
	endpoint := fmt.Sprintf(
		"/dbaas/logs/%s/cluster/%s",
		url.PathEscape(serviceName),
		url.PathEscape(cluster_id),
	)

	res := map[string]interface{}{}
	if err := config.OVHClient.Get(endpoint, &res); err != nil {
		return fmt.Errorf("Error calling GET %s:\n\t %q", endpoint, err)
	}

	d.Set("initial_archive_allowed_networks", res["archiveAllowedNetworks"])
	d.Set("initial_direct_input_allowed_networks", res["directInputAllowedNetworks"])
	d.Set("initial_query_allowed_networks", res["queryAllowedNetworks"])

	return resourceDbaasLogsClusterUpdate(d, meta)
}

func resourceDbaasLogsClusterDelete(d *schema.ResourceData, meta interface{}) error {
	// Restore ACL as they were before we managed the resource using terraform
	config := meta.(*Config)

	serviceName := d.Get("service_name").(string)
	cluster_id := d.Id()

	endpoint := fmt.Sprintf(
		"/dbaas/logs/%s/cluster/%s",
		url.PathEscape(serviceName),
		url.PathEscape(cluster_id),
	)

	opts := &DbaasLogsOpts{}
	ArchiveAllowedNetworks := d.Get("initial_archive_allowed_networks").(*schema.Set).List()
	opts.ArchiveAllowedNetworks = make([]string, len(ArchiveAllowedNetworks))
	for i, ipBlock := range ArchiveAllowedNetworks {
		opts.ArchiveAllowedNetworks[i] = ipBlock.(string)
	}
	DirectInputAllowedNetworks := d.Get("initial_direct_input_allowed_networks").(*schema.Set).List()
	opts.DirectInputAllowedNetworks = make([]string, len(DirectInputAllowedNetworks))
	for i, ipBlock := range DirectInputAllowedNetworks {
		opts.DirectInputAllowedNetworks[i] = ipBlock.(string)
	}
	QueryAllowedNetworks := d.Get("initial_query_allowed_networks").(*schema.Set).List()
	opts.QueryAllowedNetworks = make([]string, len(QueryAllowedNetworks))
	for i, ipBlock := range QueryAllowedNetworks {
		opts.QueryAllowedNetworks[i] = ipBlock.(string)
	}
	res := &DbaasLogsOperation{}
	if err := config.OVHClient.Put(endpoint, opts, res); err != nil {
		return fmt.Errorf("Error calling Put %s:\n\t %q", endpoint, err)
	}

	// Wait for operation status
	if _, err := waitForDbaasLogsOperation(config.OVHClient, serviceName, res.OperationId); err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourceDbaasLogsClusterImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	givenId := d.Id()
	splitId := strings.SplitN(givenId, "/", 2)
	if len(splitId) != 2 {
		return nil, fmt.Errorf("Import Id is not service_name/id formatted")
	}
	serviceName := splitId[0]
	id := splitId[1]
	d.SetId(id)
	d.Set("service_name", serviceName)

	results := make([]*schema.ResourceData, 1)
	results[0] = d
	return results, nil
}

func resourceDbaasLogsClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	serviceName := d.Get("service_name").(string)
	id := d.Id()

	log.Printf("[INFO] Will update dbaas logs cluster for: %s", serviceName)

	opts := (&DbaasLogsOpts{}).FromResource(d)
	res := &DbaasLogsOperation{}
	endpoint := fmt.Sprintf(
		"/dbaas/logs/%s/cluster/%s",
		url.PathEscape(serviceName),
		url.PathEscape(id),
	)

	if err := config.OVHClient.Put(endpoint, opts, res); err != nil {
		return fmt.Errorf("Error calling Put %s:\n\t %q", endpoint, err)
	}

	// Wait for operation status
	if _, err := waitForDbaasLogsOperation(config.OVHClient, serviceName, res.OperationId); err != nil {
		return err
	}

	return resourceDbaasLogsClusterRead(d, meta)
}

func resourceDbaasLogsClusterRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	serviceName := d.Get("service_name").(string)
	cluster_id := d.Id()

	log.Printf("[DEBUG] Will read dbaas logs cluster %s", serviceName)

	endpoint := fmt.Sprintf(
		"/dbaas/logs/%s/cluster/%s",
		url.PathEscape(serviceName),
		url.PathEscape(cluster_id),
	)

	res := map[string]interface{}{}
	if err := config.OVHClient.Get(endpoint, &res); err != nil {
		return fmt.Errorf("Error calling GET %s:\n\t %q", endpoint, err)
	}

	d.Set("archive_allowed_networks", res["archiveAllowedNetworks"])
	d.Set("cluster_type", res["clusterType"])
	d.Set("dedicated_input_pem", res["dedicatedInputPEM"])
	d.Set("direct_input_allowed_networks", res["directInputAllowedNetworks"])
	d.Set("direct_input_pem", res["directInputPEM"])
	d.Set("hostname", res["hostname"])
	d.Set("is_default", res["isDefault"])
	d.Set("is_unlocked", res["isUnlocked"])
	d.Set("query_allowed_networks", res["queryAllowedNetworks"])
	d.Set("region", res["region"])

	return nil
}
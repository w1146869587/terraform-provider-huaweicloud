package cluster

import "github.com/gophercloud/gophercloud"

const (
	resourcePath = "clusters"
)

func createURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL(resourcePath)
}

func updateURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL(resourcePath, id)
}

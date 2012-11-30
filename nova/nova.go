// The nova package provides a way to access the OpenStack Compute APIs.
// See http://docs.openstack.org/api/openstack-compute/2/content/.
package nova

import (
	"encoding/base64"
	"fmt"
	"launchpad.net/goose/client"
	goosehttp "launchpad.net/goose/http"
	"net/http"
)

const (
	apiFlavors            = "/flavors"
	apiFlavorsDetail      = "/flavors/detail"
	apiServers            = "/servers"
	apiServersDetail      = "/servers/detail"
	apiSecurityGroups     = "/os-security-groups"
	apiSecurityGroupRules = "/os-security-group-rules"
	apiFloatingIPs        = "/os-floating-ips"
)

// Client provides a means to access the OpenStack Compute Service.
type Client struct {
	client client.Client
}

func New(client client.Client) *Client {
	return &Client{client}
}

type Link struct {
	Href string
	Rel  string
	Type string
}

// Entity can describe a flavor, flavor detail or server.
// Contains a list of links.
type Entity struct {
	Id    string
	Links []Link
	Name  string
}

// ListFlavours lists IDs, names, and links for available flavors.
func (c *Client) ListFlavors() ([]Entity, error) {
	var resp struct {
		Flavors []Entity
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", apiFlavors, &requestData, "failed to get list of flavors")
	return resp.Flavors, err
}

type FlavorDetail struct {
	Name  string
	RAM   int
	VCPUs int
	Disk  int
	Id    string
	Swap  interface{} // Can be an empty string (?!)
}

// ListFlavorsDetail lists all details for available flavors.
func (c *Client) ListFlavorsDetail() ([]FlavorDetail, error) {
	var resp struct {
		Flavors []FlavorDetail
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", apiFlavorsDetail, &requestData,
		"failed to get list of flavors details")
	return resp.Flavors, err
}

// ListServers lists IDs, names, and links for all servers.
func (c *Client) ListServers() ([]Entity, error) {
	var resp struct {
		Servers []Entity
	}
	requestData := goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.GET, "compute", apiServers, &requestData,
		"failed to get list of servers")
	return resp.Servers, err
}

type ServerDetail struct {
	AddressIPv4 string
	AddressIPv6 string
	Created     string
	Flavor      Entity
	HostId      string
	Id          string
	Image       Entity
	Links       []Link
	Name        string
	Progress    int
	Status      string
	TenantId    string `json:"tenant_id"`
	Updated     string
	UserId      string `json:"user_id"`
}

// ListServersDetail lists all details for available servers.
func (c *Client) ListServersDetail() ([]ServerDetail, error) {
	var resp struct {
		Servers []ServerDetail
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", apiServersDetail, &requestData,
		"failed to get list of servers details")
	return resp.Servers, err
}

// GetServer lists details for the specified server.
func (c *Client) GetServer(serverId string) (ServerDetail, error) {
	var resp struct {
		Server ServerDetail
	}
	url := fmt.Sprintf("%s/%s", apiServers, serverId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", url, &requestData,
		"failed to get details for serverId=%s", serverId)
	return resp.Server, err
}

// DeleteServer terminates the specified server.
func (c *Client) DeleteServer(serverId string) error {
	var resp struct {
		Server ServerDetail
	}
	url := fmt.Sprintf("%s/%s", apiServers, serverId)
	requestData := goosehttp.RequestData{RespValue: &resp, ExpectedStatus: []int{http.StatusNoContent}}
	err := c.client.SendRequest(client.DELETE, "compute", url, &requestData,
		"failed to delete server with serverId=%s", serverId)
	return err
}

type RunServerOpts struct {
	Name               string  `json:"name"`
	FlavorId           string  `json:"flavorRef"`
	ImageId            string  `json:"imageRef"`
	UserData           *string `json:"user_data"`
	SecurityGroupNames []struct {
		Name string `json:"name"`
	} `json:"security_groups"`
}

// RunServer creates a new server.
func (c *Client) RunServer(opts RunServerOpts) error {
	var req struct {
		Server RunServerOpts `json:"server"`
	}
	req.Server = opts
	if opts.UserData != nil {
		data := []byte(*opts.UserData)
		encoded := base64.StdEncoding.EncodeToString(data)
		req.Server.UserData = &encoded
	}
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.POST, "compute", apiServers, &requestData,
		"failed to run a server with %#v", opts)
	return err
}

type SecurityGroupRule struct {
	FromPort      *int              `json:"from_port"`   // Can be nil
	IPProtocol    *string           `json:"ip_protocol"` // Can be nil
	ToPort        *int              `json:"to_port"`     // Can be nil
	ParentGroupId int               `json:"parent_group_id"`
	IPRange       map[string]string `json:"ip_range"` // Can be empty
	Id            int
	Group         map[string]string // Can be empty
}

type SecurityGroup struct {
	Rules       []SecurityGroupRule
	TenantId    string `json:"tenant_id"`
	Id          int
	Name        string
	Description string
}

// ListSecurityGroups lists IDs, names, and other details for all security groups.
func (c *Client) ListSecurityGroups() ([]SecurityGroup, error) {
	var resp struct {
		Groups []SecurityGroup `json:"security_groups"`
	}
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", apiSecurityGroups, &requestData,
		"failed to list security groups")
	return resp.Groups, err
}

// GetServerSecurityGroups list security groups for a specific server.
func (c *Client) GetServerSecurityGroups(serverId string) ([]SecurityGroup, error) {

	var resp struct {
		Groups []SecurityGroup `json:"security_groups"`
	}
	url := fmt.Sprintf("%s/%s/%s", apiServers, serverId, apiSecurityGroups)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", url, &requestData,
		"failed to list server (%s) security groups", serverId)
	return resp.Groups, err
}

// CreateSecurityGroup creates a new security group.
func (c *Client) CreateSecurityGroup(name, description string) (SecurityGroup, error) {
	var req struct {
		SecurityGroup struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"security_group"`
	}
	req.SecurityGroup.Name = name
	req.SecurityGroup.Description = description

	var resp struct {
		SecurityGroup SecurityGroup `json:"security_group"`
	}
	requestData := goosehttp.RequestData{ReqValue: req, RespValue: &resp, ExpectedStatus: []int{http.StatusOK}}
	err := c.client.SendRequest(client.POST, "compute", apiSecurityGroups, &requestData,
		"failed to create a security group with name=%s", name)
	return resp.SecurityGroup, err
}

// DeleteSecurityGroup deletes the specified security group.
func (c *Client) DeleteSecurityGroup(groupId int) error {
	url := fmt.Sprintf("%s/%d", apiSecurityGroups, groupId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.DELETE, "compute", url, &requestData,
		"failed to delete a security group with id=%d", groupId)
	return err
}

type RuleInfo struct {
	IPProtocol    string `json:"ip_protocol"`     // Required, if GroupId is nil
	FromPort      int    `json:"from_port"`       // Required, if GroupId is nil
	ToPort        int    `json:"to_port"`         // Required, if GroupId is nil
	Cidr          string `json:"cidr"`            // Required, if GroupId is nil
	GroupId       *int   `json:"group_id"`        // If nil, FromPort/ToPort/IPProtocol must be set
	ParentGroupId int    `json:"parent_group_id"` // Required always
}

// CreateSecurityGroupRule creates a security group rule.
func (c *Client) CreateSecurityGroupRule(ruleInfo RuleInfo) (SecurityGroupRule, error) {
	var req struct {
		SecurityGroupRule RuleInfo `json:"security_group_rule"`
	}
	req.SecurityGroupRule = ruleInfo

	var resp struct {
		SecurityGroupRule SecurityGroupRule `json:"security_group_rule"`
	}

	requestData := goosehttp.RequestData{ReqValue: req, RespValue: &resp}
	err := c.client.SendRequest(client.POST, "compute", apiSecurityGroupRules, &requestData,
		"failed to create a rule for the security group with id=%s", ruleInfo.GroupId)
	return resp.SecurityGroupRule, err
}

// DeleteSecurityGroupRule deletes the specified security group rule.
func (c *Client) DeleteSecurityGroupRule(ruleId int) error {
	url := fmt.Sprintf("%s/%d", apiSecurityGroupRules, ruleId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.DELETE, "compute", url, &requestData,
		"failed to delete a security group rule with id=%d", ruleId)
	return err
}

// AddServerSecurityGroup adds a security group to the specified server.
func (c *Client) AddServerSecurityGroup(serverId, groupName string) error {
	var req struct {
		AddSecurityGroup struct {
			Name string `json:"name"`
		} `json:"addSecurityGroup"`
	}
	req.AddSecurityGroup.Name = groupName

	url := fmt.Sprintf("%s/%s/action", apiServers, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.POST, "compute", url, &requestData,
		"failed to add security group '%s' to server with id=%s", groupName, serverId)
	return err
}

// RemoveServerSecurityGroup removes a security group from the specified server.
func (c *Client) RemoveServerSecurityGroup(serverId, groupName string) error {
	var req struct {
		RemoveSecurityGroup struct {
			Name string `json:"name"`
		} `json:"removeSecurityGroup"`
	}
	req.RemoveSecurityGroup.Name = groupName

	url := fmt.Sprintf("%s/%s/action", apiServers, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.POST, "compute", url, &requestData,
		"failed to remove security group '%s' from server with id=%s", groupName, serverId)
	return err
}

type FloatingIP struct {
	FixedIP    interface{} `json:"fixed_ip"` // Can be a string or null
	Id         int         `json:"id"`
	InstanceId interface{} `json:"instance_id"` // Can be a string or null
	IP         string      `json:"ip"`
	Pool       string      `json:"pool"`
}

// ListFloatingIPs lists floating IP addresses associated with the tenant or account.
func (c *Client) ListFloatingIPs() ([]FloatingIP, error) {
	var resp struct {
		FloatingIPs []FloatingIP `json:"floating_ips"`
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", apiFloatingIPs, &requestData,
		"failed to list floating ips")
	return resp.FloatingIPs, err
}

// GetFloatingIP lists details of the floating IP address associated with specified id.
func (c *Client) GetFloatingIP(ipId int) (FloatingIP, error) {
	var resp struct {
		FloatingIP FloatingIP `json:"floating_ip"`
	}

	url := fmt.Sprintf("%s/%d", apiFloatingIPs, ipId)
	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.GET, "compute", url, &requestData,
		"failed to get floating ip %d details", ipId)
	return resp.FloatingIP, err
}

// AllocateFloatingIP allocates a new floating IP address to a tenant or account.
func (c *Client) AllocateFloatingIP() (FloatingIP, error) {
	var resp struct {
		FloatingIP FloatingIP `json:"floating_ip"`
	}

	requestData := goosehttp.RequestData{RespValue: &resp}
	err := c.client.SendRequest(client.POST, "compute", apiFloatingIPs, &requestData,
		"failed to allocate a floating ip")
	return resp.FloatingIP, err
}

// DeleteFloatingIP deallocates the floating IP address associated with the specified id.
func (c *Client) DeleteFloatingIP(ipId int) error {
	url := fmt.Sprintf("%s/%d", apiFloatingIPs, ipId)
	requestData := goosehttp.RequestData{ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.DELETE, "compute", url, &requestData,
		"failed to delete floating ip %d details", ipId)
	return err
}

// AddServerFloatingIP assigns a floating IP addess to the specified server.
func (c *Client) AddServerFloatingIP(serverId, address string) error {
	var req struct {
		AddFloatingIP struct {
			Address string `json:"address"`
		} `json:"addFloatingIp"`
	}
	req.AddFloatingIP.Address = address

	url := fmt.Sprintf("%s/%s/action", apiServers, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.POST, "compute", url, &requestData,
		"failed to add floating ip %s to server %s", address, serverId)
	return err
}

// RemoveServerFloatingIP removes a floating IP address from the specified server.
func (c *Client) RemoveServerFloatingIP(serverId, address string) error {
	var req struct {
		RemoveFloatingIP struct {
			Address string `json:"address"`
		} `json:"removeFloatingIp"`
	}
	req.RemoveFloatingIP.Address = address

	url := fmt.Sprintf("%s/%s/action", apiServers, serverId)
	requestData := goosehttp.RequestData{ReqValue: req, ExpectedStatus: []int{http.StatusAccepted}}
	err := c.client.SendRequest(client.POST, "compute", url, &requestData,
		"failed to remove floating ip %s to server %s", address, serverId)
	return err
}
package fpoc

import (
	"maps"
	"regexp"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/mitchellh/mapstructure"
)

// ExtCreateOpts extended version of servers.CreateOpts
// nolint:revive
type ExtCreateOpts struct {
	servers.CreateOpts

	// fields absent in gophercloud
	Description string `json:"description,omitempty"`
	KeyName     string `json:"key_name,omitempty"`

	// annotation overrides
	Networks       []servers.Network `json:"networks,omitempty"`
	SecurityGroups []string          `json:"security_groups,omitempty"`
	UserData       string            `json:"user_data,omitempty"`
}

// ToServerCreateMap for extended opts
func (opts ExtCreateOpts) ToServerCreateMap() (map[string]interface{}, error) {
	if opts.Networks != nil {
		opts.CreateOpts.Networks = opts.Networks
	}

	if opts.SecurityGroups != nil {
		opts.CreateOpts.SecurityGroups = opts.SecurityGroups
	}

	if opts.UserData != "" {
		opts.CreateOpts.UserData = []byte(opts.UserData)
	}

	ob, err := opts.CreateOpts.ToServerCreateMap()
	if err != nil {
		return nil, err
	}

	/*
		b, err := gophercloud.BuildRequestBody(opts, "")
		if err != nil {
			return nil, err
		}

		delete(b, "user_data")
		delete(b, "security_groups")
		delete(b, "SchedulerHints")
	*/

	b := map[string]any{}
	if opts.Description != "" {
		b["description"] = opts.Description
	}
	if opts.KeyName != "" {
		b["key_name"] = opts.KeyName
	}

	sob := ob["server"].(map[string]any)
	maps.Copy(sob, b)

	return ob, nil
}

type Address struct {
	Version int    `json:"version"`
	Address string `json:"addr"`
	MACAddr string `json:"OS-EXT-IPS-MAC:mac_addr,omitempty"`
	Type    string `json:"OS-EXT-IPS:type,omitempty"`
}

func extractAddresses(srv *servers.Server) (map[string][]Address, error) {
	ret := make(map[string][]Address, len(srv.Addresses))

	for net, isv := range srv.Addresses {
		ism := isv.([]interface{})
		items := make([]Address, 0, len(ism))

		for _, iv := range ism {
			var out Address

			cfg := &mapstructure.DecoderConfig{
				Metadata: nil,
				Result:   &out,
				TagName:  "json",
			}
			decoder, _ := mapstructure.NewDecoder(cfg)
			err := decoder.Decode(iv)
			if err != nil {
				return nil, err
			}

			items = append(items, out)
		}

		ret[net] = items
	}

	return ret, nil
}

var initFinishedRe = regexp.MustCompile(`^.*Cloud-init\ v\.\ \d+\.\d+\.\d+\ finished\ at.*$`)

func IsCloudInitFinished(log string) bool {
	lines := strings.Split(log, "\n")

	for _, line := range lines {
		if initFinishedRe.MatchString(line) {
			return true
		}
	}
	return false
}

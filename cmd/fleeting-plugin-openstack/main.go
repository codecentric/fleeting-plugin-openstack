package main

import (
	osplugin "github.com/codecentric/fleeting-plugin-openstack"
	"gitlab.com/gitlab-org/fleeting/fleeting/plugin"
)

func main() {
	plugin.Serve(&osplugin.InstanceGroup{})
}

package sources

import (
//	"encoding/json"

	"github.com/golang/glog"
)


type ExternalSource struct {
	cadvisor *cadvisorSource
}

func (self *ExternalSource) getCadvisorHosts() (*CadvisorHosts, error) {
	var cadvisorHosts CadvisorHosts
	cadvisorHosts.Port= 31500
	cadvisorHosts.Hosts = map[string]string{
		"slave1": "172.31.2.11",
		"slave2": "172.31.2.12",
		"slave3": "172.31.2.13",
	}
	return &cadvisorHosts, nil
}

func (self *ExternalSource) GetInfo() (ContainerData, error) {
	hosts, err := self.getCadvisorHosts()
	if err != nil {
		return ContainerData{}, err
	}

	containers, nodes, err := self.cadvisor.fetchData(hosts)
	if err != nil {
		glog.Error(err)
		return ContainerData{}, nil
	}

	return ContainerData{
		Containers: containers,
		Machine:    nodes,
	}, nil
}

func newExternalSource() (Source, error) {
	cadvisorSource := newCadvisorSource()
	return &ExternalSource{
		cadvisor: cadvisorSource,
	}, nil
}

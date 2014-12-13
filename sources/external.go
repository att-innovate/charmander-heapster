package sources

import (
	"github.com/golang/glog"

	"github.com/att-innovate/charmander-heapster/charmander"
)

type ExternalSource struct {
	cadvisor *cadvisorSource
}

func (self *ExternalSource) GetInfo() (ContainerData, error) {
	var cadvisorHosts CadvisorHosts
	cadvisorHosts.Port = 31500
	cadvisorHosts.Hosts = *charmander.GetCadvisorHosts()

	containers, nodes, err := self.cadvisor.fetchData(&cadvisorHosts)
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

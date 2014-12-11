package sources

import (
	"flag"
	"net"
	"time"
	"strconv"
	"bufio"

//	"encoding/json"

	"github.com/golang/glog"
)

var redisHost = flag.String("source_redis_host", "127.0.0.1:6379", "Redis IP Address:Port")

type ExternalSource struct {
	cadvisor *cadvisorSource
}

func (self *ExternalSource) getCadvisorHosts() (*CadvisorHosts, error) {
	var cadvisorHosts CadvisorHosts
	cadvisorHosts.Port= 31500
	cadvisorHosts.Hosts = map[string]string{}

	if redis := redisAvailable(); redis != nil {
		sendCommand(redis, "KEYS", "charmander:nodes:*")
		hosts := *parseResult(redis)
		for _, host := range hosts {
			cadvisorHosts.Hosts[host] = host
		}
		redis.Close()
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

func redisAvailable() net.Conn {
	connection, error := net.DialTimeout("tcp", *redisHost, 2 * time.Second)
	if error != nil {
		return nil
	}

	return connection
}

func sendCommand(connection net.Conn, args ...string) {
	buffer := make([]byte, 0, 0)
	buffer = encodeReq(buffer, args)
	connection.Write(buffer)
}

func parseResult(connection net.Conn) *[]string {
	bufferedInput := bufio.NewReader(connection)
	line, _, err := bufferedInput.ReadLine()
	if err != nil {
		glog.Errorf("error parsing redis response %s\n", err)
		return &[]string {}
	}
	numberOfArgs, _ := strconv.ParseInt(string(line[1:]), 10, 64)
	args := make([]string, 0, numberOfArgs)
	for i := int64(0); i < numberOfArgs; i++ {
		line, _, _ = bufferedInput.ReadLine()
		argLen, _ := strconv.ParseInt(string(line[1:]), 10, 32)
		line, _, _ = bufferedInput.ReadLine()
		args = append(args, string(line[len("charmander:nodes:"):argLen]))
	}

	return &args
}

func encodeReq(buf []byte, args []string) []byte {
	buf = append(buf, '*')
	buf = strconv.AppendUint(buf, uint64(len(args)), 10)
	buf = append(buf, '\r', '\n')
	for _, arg := range args {
		buf = append(buf, '$')
		buf = strconv.AppendUint(buf, uint64(len(arg)), 10)
		buf = append(buf, '\r', '\n')
		buf = append(buf, []byte(arg)...)
		buf = append(buf, '\r', '\n')
	}
	return buf
}


func newExternalSource() (Source, error) {
	cadvisorSource := newCadvisorSource()
	return &ExternalSource{
		cadvisor: cadvisorSource,
	}, nil
}

package sinks

import (
	"flag"
	"fmt"
	"time"

	"github.com/att-innovate/charmander-heapster/sources"
	"github.com/att-innovate/charmander-heapster/charmander"
	"github.com/golang/glog"

	cadvisor "github.com/google/cadvisor/info"
	influxdb "github.com/influxdb/influxdb/client"

)

var (
	argBufferDuration = flag.Duration("sink_influxdb_buffer_duration", 10*time.Second, "Time duration for which stats should be buffered in influxdb sink before being written as a single transaction")
	argDbUsername     = flag.String("sink_influxdb_username", "root", "InfluxDB username")
	argDbPassword     = flag.String("sink_influxdb_password", "root", "InfluxDB password")
	argDbHost         = flag.String("sink_influxdb_host", "localhost:8086", "InfluxDB host:port")
	argDbName         = flag.String("sink_influxdb_name", "charmander", "Influxdb database name")
)

type InfluxdbSink struct {
	client         *influxdb.Client
	series         []*influxdb.Series
	dbName         string
	bufferDuration time.Duration
	lastWrite      time.Time
	containerIdMap map[string]string
	metered        map[string]bool
}

func (self *InfluxdbSink) containerStatsToValues(hostname, containerName string, spec cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) (columns []string, values []interface{}) {
	// Timestamp
	columns = append(columns, colTimestamp)
	values = append(values, stat.Timestamp.Unix())

	// Hostname
	columns = append(columns, colHostName)
	values = append(values, hostname)

	// Container name
	columns = append(columns, colContainerName)
	values = append(values, containerName)

	if spec.HasCpu {
		// Cumulative Cpu Usage
		columns = append(columns, colCpuCumulativeUsage)
		values = append(values, stat.Cpu.Usage.Total)
	}

	if spec.HasMemory {
		// Memory Usage
		columns = append(columns, colMemoryUsage)
		values = append(values, stat.Memory.Usage)

		// Memory Page Faults
		columns = append(columns, colMemoryPgFaults)
		values = append(values, stat.Memory.ContainerData.Pgfault)

		// Working set size
		columns = append(columns, colMemoryWorkingSet)
		values = append(values, stat.Memory.WorkingSet)
	}

	// Optional: Network stats.
	if spec.HasNetwork {
		columns = append(columns, colRxBytes)
		values = append(values, stat.Network.RxBytes)

		columns = append(columns, colRxErrors)
		values = append(values, stat.Network.RxErrors)

		columns = append(columns, colTxBytes)
		values = append(values, stat.Network.TxBytes)

		columns = append(columns, colTxErrors)
		values = append(values, stat.Network.TxErrors)
	}
	return
}

// Returns a new influxdb series.
func (self *InfluxdbSink) newSeries(tableName string, columns []string, points []interface{}) *influxdb.Series {
	out := &influxdb.Series{
		Name:    tableName,
		Columns: columns,
		// There's only one point for each stats
		Points: make([][]interface{}, 1),
	}
	out.Points[0] = points
	return out
}

func (self *InfluxdbSink) handleContainers(containers []sources.RawContainer, tableName string) {
	for _, container := range containers {
		containerName :=self.resolveContainerName(container.Name, container.Hostname)
		if len(containerName) == 0 { continue }
		if self.isMetered(containerName) == false { continue }

		for _, stat := range container.Stats {
			col, val := self.containerStatsToValues(container.Hostname, containerName, container.Spec, stat)
			self.series = append(self.series, self.newSeries(tableName, col, val))
		}
	}
}

func (self *InfluxdbSink) handleMachines(machines []sources.RawContainer, tableName string) {
	for _, machine := range machines {
		for _, stat := range machine.Stats {
			col, val := self.containerStatsToValues(machine.Hostname, machine.Name, machine.Spec, stat)
			self.series = append(self.series, self.newSeries(tableName, col, val))
		}
	}
}

func (self *InfluxdbSink) resolveContainerName(containerId string, hostname string) string {
	if containerId[0] == '/' { return containerId }
	if containerName := self.containerIdMap[containerId]; len(containerName) > 0 { return containerName }

	containerName := charmander.ResolveContainerName(containerId, hostname)
	self.containerIdMap[containerId] = containerName
	glog.Infof("Resolved containerId - [%s] [%s]", containerId, containerName)

	return containerName
}

func (self *InfluxdbSink) readyToFlush() bool {
	return time.Since(self.lastWrite) >= self.bufferDuration
}

func (self *InfluxdbSink) StoreData(ip Data) error {
	var seriesToFlush []*influxdb.Series
	if data, ok := ip.(sources.ContainerData); ok {
		self.handleContainers(data.Containers, statsTable)
		self.handleMachines(data.Machine, machineTable)
	} else {
		return fmt.Errorf("Requesting unrecognized type to be stored in InfluxDB")
	}
	if self.readyToFlush() {
		seriesToFlush = self.series
		self.series = make([]*influxdb.Series, 0)
		self.lastWrite = time.Now()
	}

	if len(seriesToFlush) > 0 {
		glog.V(2).Info("flushed data to influxdb sink")
		// TODO(vishh): Do writes in a separate thread.
		err := self.client.WriteSeriesWithTimePrecision(seriesToFlush, influxdb.Second)
		if err != nil {
			glog.Errorf("failed to write stats to influxDb - %s", err)
		}
	}

	return nil
}

func (self *InfluxdbSink) isMetered(containerName string) bool {
	if metered, ok := self.metered[containerName]; ok { return metered }
	if charmander.ContainerReady(containerName) == false { return false }

	result := false

	if containerName[0] != '/' {
		result = charmander.ContainerMetered(containerName)
		glog.Infof("Container metered %s %v", containerName, result)

	}

	self.metered[containerName] = result

	return result
}

func NewInfluxdbSink() (Sink, error) {
	config := &influxdb.ClientConfig{
		Host:     *argDbHost,
		Username: *argDbUsername,
		Password: *argDbPassword,
		Database: *argDbName,
		IsSecure: false,
	}
	client, err := influxdb.NewClient(config)
	if err != nil {
		return nil, err
	}
	client.DisableCompression()
	if err := client.CreateDatabase(*argDbName); err != nil {
		glog.Infof("Database creation failed - %s", err)
	}
	// Create the database if it does not already exist. Ignore errors.
	return &InfluxdbSink{
		client:         client,
		series:         make([]*influxdb.Series, 0),
		dbName:         *argDbName,
		bufferDuration: *argBufferDuration,
		lastWrite:      time.Now(),
		containerIdMap: make(map[string]string),
		metered:        make(map[string]bool),
	}, nil
}

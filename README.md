Heapster
===========

Note .. work in progress .. making it part of charmander


_Warning: Virtual Machines need to have at least 2 cores for InfluxDB to perform optimally._

Heapster enables monitoring of Clusters using [cAdvisor](https://github.com/google/cadvisor).


#####Hints
* Grafana's default username and password is 'admin'. You can change that by modifying the grafana container [here](influx-grafana/deploy/grafana-influxdb-pod.json)
* To enable memory and swap accounting on the minions follow the instructions [here](https://docs.docker.com/installation/ubuntulinux/#memory-and-swap-accounting)

#### Community

Contributions, questions, and comments are all welcomed and encouraged! Heapster and cAdvisor developers hang out in [#google-containers](http://webchat.freenode.net/?channels=google-containers) room on freenode.net.  We also have the [google-containers Google Groups mailing list](https://groups.google.com/forum/#!forum/google-containers).

package charmander

import (
	"net/http"
	"strings"
	"io/ioutil"

	"github.com/golang/glog"
)

func ResolveContainerName(containerId string, hostname string) string {
	if containerId[0] == '/' { return containerId }

	resp, err := http.Get("http://"+hostname+":31300/"+containerId)
	if err != nil {
		glog.Errorf("Failed to look up containerId - %s", err)
		return ""
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if len(body) == 0 { return "" }

	return strings.TrimSpace(string(body))
}

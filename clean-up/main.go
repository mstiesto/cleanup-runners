package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

var (
	gitlab_url string = "https://gitlab.example.com/api/v4/runners/"
	gitlab_token string = "xxxxx"
)

type DiskStatus struct {
	All         uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

func DiskUsage(path string) (disk DiskStatus) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return
	}
	disk.All = fs.Blocks
	disk.Free = fs.Bfree
	disk.Used = disk.All - disk.Free
	disk.UsedPercent = float64(disk.Used) / float64(disk.All) * 100
	return
}

func pauseRunner(ID, state string) (responseStatusCode int) {
	var (
		gitlab_url string = gitlab_url + ID
		active string= "true"
		client = &http.Client{}
		form = url.Values {}
	)
	fmt.Printf("--> %v the runner ...\n", state)
	if state == "pause" {
		active = "false"
	}
	form.Add("active", active)
    req, err := http.NewRequest("PUT", gitlab_url, strings.NewReader(form.Encode()))
	req.Header.Add("PRIVATE-TOKEN", gitlab_token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    if err != nil {
        log.Fatal(err)
    }
    resp, err := client.Do(req)
	if err != nil {
        log.Fatal(err)
    }
	fmt.Printf("--> Response: %d\n", resp.StatusCode)
	return resp.StatusCode
}
func sendMessage(runnerName, state string) (responseStatusCode int) {
	var (
		element_url string = "https://element.example.com"
		client = &http.Client{}
	)
	fmt.Printf("--> Sending %v message to the element ...\n", state)
	body := fmt.Sprintf(`{
		"text": "%v %vd",
		"displayName": "Go runners cleanup"
	}`, runnerName, state)
    req, err := http.NewRequest("POST", element_url, strings.NewReader(body))
	req.Header.Add("Content-Type", "application/json")
    if err != nil {
        log.Fatal(err)
    }
    resp, err := client.Do(req)
	if err != nil {
        log.Fatal(err)
    }
	fmt.Printf("--> Response: %d\n", resp.StatusCode)
	return resp.StatusCode
}
func getJobs(runnerID string) int {
	var (
		gitlab_url string = gitlab_url + runnerID
		url = gitlab_url + "/" + "/jobs?status=running"
		client = &http.Client{}
		jsonObjects interface{}
	)
	fmt.Printf("--> Waiting for running jobs to be finished ...\n")
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("PRIVATE-TOKEN", gitlab_token)
	if err != nil {
		log.Fatal(err)
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal([]byte(body), &jsonObjects)
	objectSlice, ok := jsonObjects.([]interface{})

	if !ok {
		fmt.Println("cannot convert the JSON objects")
		os.Exit(1)
	}
	fmt.Printf("--> jobs count: %d\n", len(objectSlice))
	return len(objectSlice)
}
func imagePrune() {
	var (
		ctx = context.Background()
		filter = filters.Arg("dangling", "true")
	)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}
	report, err := cli.ImagesPrune(ctx, filters.NewArgs(filter))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("--> Total reclamed space from imagePrune: %v MB\n", report.SpaceReclaimed/1048576)
}

func volumePrune() {
	var (
		ctx = context.Background()
		filter = filters.Arg("label", "!keep")
	)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}
	report, err := cli.VolumesPrune(ctx , filters.NewArgs(filter))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("--> Total reclamed space from volumePrune: %v MB\n", report.SpaceReclaimed/1048576)
}
func main() {
	var (
		runner string = os.Args[1]
		capacity uint64 = 70
		runner_id string
		runner_ids = map[string]string{"runner-01": "xx", "runner-02": "xx", "runner-03": "xx"}
		disk = DiskUsage("/")
	)
	for runnerName := range runner_ids {
		if runnerName == runner {
			runner_id = runner_ids[runnerName]
		}
	}
	if runner_id == "" {
		log.Fatal("%v is not a valid runner", runner)
	}
	fmt.Printf("--> %v\n--> Script started on %v with ID %v\n",time.Now().Format(time.UnixDate), runner, runner_id)
	if uint64(disk.UsedPercent) > capacity {
		response := 1
		for response != 200 {
			time.Sleep(5 * time.Second)
			response = pauseRunner(runner_id, "pause")
		}
		response = 1
		for response != 200 {
			time.Sleep(5 * time.Second)
			response = sendMessage(runner, "pause")
		}
		jobs := 1
		for jobs != 0 {
			time.Sleep(5 * time.Second)
			jobs = getJobs(runner_id)
		}
		imagePrune()
		volumePrune()
		response = 1
		for response != 200 {
			time.Sleep(5 * time.Second)
			response = pauseRunner(runner_id, "continue")
		}
		response = 1
		for response != 200 {
			time.Sleep(5 * time.Second)
			response = sendMessage(runner, "continue")
		}
		fmt.Printf("--> %v\n--> Finished\n",time.Now().Format(time.UnixDate))
	} else {
		fmt.Printf("################################### \nThe storage is not yet full enough \n###################################\n")
	}
}

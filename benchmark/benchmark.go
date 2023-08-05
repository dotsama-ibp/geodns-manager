package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
)

type Benchmarks struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Disk   string `json:"disk"`
}

type ComponentInfo struct {
	Make  string `json:"make"`
	Model string `json:"model"`
}

type ServerComponents struct {
	CPU    ComponentInfo `json:"cpu"`
	Memory ComponentInfo `json:"memory"`
	Disk   ComponentInfo `json:"disk"`
}

type BenchmarkReport struct {
	Benchmarks      Benchmarks      `json:"benchmarks"`
	ServerComponents ServerComponents `json:"server_components"`
}

func main() {
	submitFlag := flag.Bool("submit", false, "Set this flag to submit the report to the URL")
	flag.Parse()

	serverComponents := ServerComponents{
		CPU:    getCPUInfo(),
		Memory: getMemoryInfo(),
		Disk:   getDiskInfo(),
	}

	benchmarks := Benchmarks{
		CPU:    benchmarkCPU(),
		Memory: benchmarkMemory(),
		Disk:   benchmarkDisk(),
	}

	report := BenchmarkReport{
		Benchmarks:      benchmarks,
		ServerComponents: serverComponents,
	}

	fmt.Println("Report:")
	fmt.Printf("CPU:    Make: %s, Model: %s, Benchmark: %s\n", report.ServerComponents.CPU.Make, report.ServerComponents.CPU.Model, report.Benchmarks.CPU)
	fmt.Printf("Memory: Make: %s, Model: %s, Benchmark: %s\n", report.ServerComponents.Memory.Make, report.ServerComponents.Memory.Model, report.Benchmarks.Memory)
	fmt.Printf("Disk:   Make: %s, Model: %s, Benchmark: %s\n", report.ServerComponents.Disk.Make, report.ServerComponents.Disk.Model, report.Benchmarks.Disk)

	if *submitFlag {
		url := "YOUR_REST_URL_HERE"
		sendReport(report, url)
	}
}

func getCPUInfo() ComponentInfo {
	make, _ := exec.Command("sh", "-c", "lscpu | grep 'Vendor ID:'").Output()
	model, _ := exec.Command("sh", "-c", "lscpu | grep 'Model name:'").Output()
	return ComponentInfo{Make: string(make), Model: string(model)}
}

func getMemoryInfo() ComponentInfo {
	make, _ := exec.Command("sh", "-c", "sudo dmidecode -t memory | grep 'Manufacturer:' | head -n 1").Output()
	model, _ := exec.Command("sh", "-c", "sudo dmidecode -t memory | grep 'Part Number:' | head -n 1").Output()
	return ComponentInfo{Make: string(make), Model: string(model)}
}

func getDiskInfo() ComponentInfo {
	make, _ := exec.Command("sh", "-c", "sudo hdparm -I /dev/sda | grep 'Model Number:'").Output()
	model, _ := exec.Command("sh", "-c", "sudo hdparm -I /dev/sda | grep 'Serial Number:'").Output()
	return ComponentInfo{Make: string(make), Model: string(model)}
}

func benchmarkCPU() string {
	output, _ := exec.Command("sysbench", "cpu", "--time=60", "run").Output()
	return string(output)
}

func benchmarkMemory() string {
	output, _ := exec.Command("sysbench", "memory", "--time=60", "run").Output()
	return string(output)
}

func benchmarkDisk() string {
	output, _ := exec.Command("sysbench", "fileio", "--time=60", "run").Output()
	return string(output)
}

func sendReport(report BenchmarkReport, url string) {
	jsonData, err := json.Marshal(report)
	if err != nil {
		fmt.Println(err)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Response:", string(body))
}

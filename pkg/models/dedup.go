package models

import (
	"log"
	"os/exec"
	"strings"
)

// basically a code path which can be used to identify a line of code
// this can be in encoded form for different classes and files
// for example: pkg/models/dedup.go:Compute Class:1 or pkg/models/dedup.go:Compute Class:2
var line_path map[string]string = make(map[string]string)

// maintain an array for duplicate ids and then remove them from the map
var duplicate_ids []string
var not_duplicate_ids []string

type Dedup struct {
	Lines_covered   int    `json:"lines_covered" yaml:"lines_covered"`
	Branch_covered  int    `json:"branch_covered" yaml:"branch_covered"`
	Lines_total     int    `json:"lines_total" yaml:"lines_total"`
	Branch_total    int    `json:"branch_total" yaml:"branch_total"`
	Methods_covered int    `json:"methods_covered" yaml:"methods_covered"`
	Methods_total   int    `json:"methods_total" yaml:"methods_total"`
	Classes_covered int    `json:"classes_covered" yaml:"classes_covered"`
	Classes_total   int    `json:"classes_total" yaml:"classes_total"`
	Line_path       string `json:"line_path" yaml:"line_path"`
}

func (d *Dedup) Compute(id string) {
	if _, ok := line_path[d.Line_path]; ok {
		duplicate_ids = append(duplicate_ids, id)

	} else {
		if IsSuperSet(d.Line_path) {
			duplicate_ids = append(duplicate_ids, id)
			return
		}
		// println("not exists ")
		line_path[d.Line_path] = id
		not_duplicate_ids = append(not_duplicate_ids, id)
		// println(line_path[d.Line_path] + " " + id)
	}
}

func hasSuperset(sets [][]string, targetSet string) bool {
	targetElements := strings.Split(targetSet, ",")
	for _, set := range sets {
		isSuperset := true
		for _, element := range targetElements {
			if !contains(set, element) {
				isSuperset = false
				break
			}
		}
		if isSuperset {
			return true
		}
	}
	return false
}

func contains(set []string, element string) bool {
	for _, e := range set {
		if e == element {
			return true
		}
	}
	return false
}

// we have to find superset of remaining testcases and print them

func IsSuperSet(targetSet string) bool {
	var set [][]string
	for key := range line_path {
		// print(key + " " + v)
		k := strings.Split(key, ",")
		set = append(set, k)
	}
	result := hasSuperset(set, targetSet)
	return result
}

// take a list which contains all the line hits by different testcases
// then sort the testcases by the number of line hits
// then start removing individual hit for a testcase and check if it reduces the coverage dont remove it
// if it doesn't affect the current coverage remove it
func (d *Dedup) Deduplication() {
	// println("already exists!! Current duplicate testcases are with ids :")
	log.Printf("Keploy has detected %v duplicate testcases with ids: %v", len(duplicate_ids), duplicate_ids)
	log.Printf("run `keploy dedup` to remove duplicate testcases")
	println("------------------------------------")
	log.Printf("Keploy has detected %v non duplicate testcases with ids: %v", len(not_duplicate_ids), not_duplicate_ids)
	// for _, v := range duplicate_ids {
	// 	println("id: " + v)
	// }
}
func NewDedup() *Dedup {
	return &Dedup{}
}

// Run this during each test case run create a Nyc report check the path -> then extract the json , write it in a yaml
// read the yaml and then compute the coverage for each testcase and then remove the duplicates
func ProcessNycJs() {
	// exec.Command("nyc", "report", "--reporter=json")
	exec.Command("node", "server.js")
}

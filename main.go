package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	datadogApi "github.com/DataDog/go-datadog-api"
)

func main() {
	filePath := os.Getenv("INPUT_FILE_PATH")
	fBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("could not read file %s", err)
	}
	fCont := string(fBytes)
	graphs := extractGraphs(fCont)

	buf := bytes.Buffer{}
	for _, g := range graphs {
		for _, r := range g.Definition.Requests {
			buf.WriteString(r.Query)
			buf.WriteString("\n")
		}
	}

	ioutil.WriteFile("output.txt", []byte(buf.String()), 0644)
}

func extractGraphs(fCont string) []*datadogApi.Graph {
	var pos int
	graphs := make([]*datadogApi.Graph, 0)
	collect := false
	scanner := bufio.NewScanner(strings.NewReader(fCont))
	result := make([]string, 0)
	results := make([][]string, 0)

	for scanner.Scan() {
		line := scanner.Text()

		if collect == false {
			re := regexp.MustCompile(`\ *definition: \{ *`)
			if match := re.MatchString(line); match {
				collect = true

				pos = strings.Index(line, "definition")
				if pos == -1 {
					log.Fatal("could not find start of json object")
				}
			}
		}

		if collect == true {
			result = append(result, line)

			match, err := regexp.MatchString(".*}.*", line)
			if err == nil && match {
				endPos := strings.Index(line, "}")
				if endPos == -1 {
					log.Fatal("could not find position of closing bracket")
				}

				if endPos == (pos - 2) {
					collect = false
					pos = -1
					endPos = -1

					results = append(results, result)
					result = make([]string, 0)
				}
			}
		}
	}

	replacements := make([][]string, 0)
	for _, r := range results {
		buf := bytes.Buffer{}
		buf.WriteString("{")

		for _, line := range r {
			buf.WriteString(line)

			re := regexp.MustCompile(` *([a-zA-Z]+):.*`)
			match := re.MatchString(line)
			if match {
				group1 := re.FindStringSubmatch(line)[1]
				replace := []string{
					fmt.Sprintf("%s:", group1),
					fmt.Sprintf(`"%s":`, group1),
				}
				replacements = append(replacements, replace)
			}
		}

		rString := buf.String()
		for _, replace := range replacements {
			rString = strings.Replace(rString, replace[0], replace[1], 1)
		}

		graph := &datadogApi.Graph{}
		err := json.Unmarshal([]byte(rString), graph)
		if err != nil {
			log.Fatalf("could not unmarshal json: %v", err)
		}

		graphs = append(graphs, graph)
	}

	return graphs
}

package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"os/exec"
)

func readlink(master string, loc string) {
	inputFile, err := os.Open(master)
	if err != nil {
		fmt.Println(err)
	}
	defer inputFile.Close()
	scanner := bufio.NewScanner(inputFile)
	for scanner.Scan() {
		// Get the current line
		download(scanner.Text(), loc)

	}
	return
}

func download(link string, loc string) {
	// Make an HTTP GET request to the URL of the file you want to download
	response, err := http.Get(link)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("download:" + link + " completed!")
	}
	defer response.Body.Close()

	// Create a local file to save the downloaded file
	file, err := os.OpenFile(loc, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	// Use io.Copy to copy the response body to the local file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		fmt.Println(err)
	}
	return
}
func processing(loc string, output string, custom string) {
	i := 0 // Number of duplicated items
	j := 0 // Number of valid items
	outputFile, err := os.Create(output)
	if err != nil {
		fmt.Println(err)
	}
	linesWritten := make(map[string]bool)
	inputFile, err := os.Open(loc)
	if err != nil {
		fmt.Println(err)
	}
	defer inputFile.Close()
	scanner := bufio.NewScanner(inputFile)
	println("Creating " + output + " ......")
	for scanner.Scan() {
		// Get the current line
		line := scanner.Text()
		// Check if the line is started from "[blank] or [space] or # or !"
		match, _ := regexp.MatchString("^[ #!]|^$|^\\[", line)
		if match {
			linesWritten[line] = true
		}
		if !linesWritten[line] {
			// Write the line to the output file
			outputFile.WriteString(line + "\n")
			j++
			// Mark the line as written
			linesWritten[line] = true
		} else {
			i++
			fmt.Println("removed: "+line)	
		}
	}
	customFile, err := os.Open(custom)
	if err != nil {
		fmt.Println(err)
	}
	defer customFile.Close()
	scanner1 := bufio.NewScanner(customFile)
	for scanner1.Scan() {
		outputFile.WriteString(scanner1.Text() + "\n")
		j++
	}
	defer outputFile.Close()
	println(strconv.Itoa(i) + " of items are deleted")
	println(strconv.Itoa(j) + " of items are included")
	println(output + " is created with addition rules from " + custom)
	return

}
func remove(loc string) {
	err := os.Remove(loc)
	if err != nil {
		fmt.Println(err)
	}
	return
}

func main() {
	readlink(os.Args[1], "zjc")
	cmd := exec.Command("hostlist-compiler", "-c", "./setting/c.json", "-o", "ckc")
	out, err := cmd.CombinedOutput()
	if err != nil { 
                fmt.Println("Error: ", err)
        } else  {
		fmt.Println(string(out))
                fmt.Println("Data Cleansing completed")
   	}             
        processing("ckc", os.Args[2], os.Args[3])
	remove("ckc")
	remove("zjc")
}

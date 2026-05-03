package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func readlink(master string, loc string) {
	inputFile, err := os.Open(master)
	if err != nil {
		fmt.Println("Error opening rules file:", err)
		return
	}
	defer inputFile.Close()

	scanner := bufio.NewScanner(inputFile)
	for scanner.Scan() {
		link := strings.TrimSpace(scanner.Text())
		if link != "" && !strings.HasPrefix(link, "#") {
			download(link, loc)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading rules file:", err)
	}
}

func download(link string, loc string) {
	response, err := http.Get(link)
	if err != nil {
		fmt.Println("Download error for", link, ":", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Printf("Download failed for %s: status %d\n", link, response.StatusCode)
		return
	}

	file, err := os.OpenFile(loc, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file for writing:", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	fmt.Println("download:" + link + " completed!")
}

func processing(loc string, output string, custom string) (j int) {
	i := 0
	j = 0

	outputFile, err := os.Create(output)
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return 0
	}
	defer outputFile.Close()

	linesWritten := make(map[string]bool)

	inputFile, err := os.Open(loc)
	if err != nil {
		fmt.Println("Error opening input file:", err)
		return 0
	}
	defer inputFile.Close()

	scanner := bufio.NewScanner(inputFile)
	fmt.Println("Creating " + output + " ......")

	for scanner.Scan() {
		line := scanner.Text()
		match, _ := regexp.MatchString("^[ #!]|^$|^\\[", line)
		if match {
			linesWritten[line] = true
		}
		if !linesWritten[line] {
			outputFile.WriteString(line + "\n")
			j++
			linesWritten[line] = true
		} else {
			i++
			if i <= 10 {
				fmt.Println("removed: " + line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading input file:", err)
	}

	customFile, err := os.Open(custom)
	if err != nil {
		fmt.Println("Warning: Custom file not found:", custom)
		fmt.Println(strconv.Itoa(i) + " of items are deleted")
		fmt.Println(strconv.Itoa(j) + " of items are included")
		return j
	}
	defer customFile.Close()

	scanner1 := bufio.NewScanner(customFile)
	for scanner1.Scan() {
		line := scanner1.Text()
		if !linesWritten[line] {
			outputFile.WriteString(line + "\n")
			j++
			linesWritten[line] = true
		}
	}

	if err := scanner1.Err(); err != nil {
		fmt.Println("Error reading custom file:", err)
	}

	fmt.Println(strconv.Itoa(i) + " of items are deleted")
	fmt.Println(strconv.Itoa(j) + " of items are included")
	fmt.Println(output + " is created with addition rules from " + custom)
	return j
}

func remove(loc string) {
	err := os.Remove(loc)
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("Warning: Could not remove", loc, ":", err)
	}
}

func main() {
	args := os.Args

	skipDNS := false
	filteredArgs := []string{}
	for _, arg := range args {
		if arg == "--skip-dns" {
			skipDNS = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	args = filteredArgs

	if len(args) < 4 {
		fmt.Println("Usage: go run main.go [--skip-dns] <rules_file> <output_file> <custom_file> [tier]")
		fmt.Println("  --skip-dns: Skip DNS NXDOMAIN filtering")
		fmt.Println("  tier: super, full, medium, min (default: full)")
		os.Exit(1)
	}

	rulesFile := args[1]
	outputFile := args[2]
	customFile := args[3]
	tier := "full"
	if len(args) > 4 {
		tier = args[4]
	}

	fmt.Printf("Processing tier: %s\n", tier)
	fmt.Printf("Rules file: %s\n", rulesFile)
	fmt.Printf("Output file: %s\n", outputFile)
	fmt.Printf("Custom file: %s\n", customFile)

	readlink(rulesFile, "zjc")

	cmd := exec.Command("hostlist-compiler", "-c", "./setting/c.json", "-o", "ckc")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("hostlist-compiler error:", err)
		fmt.Println(string(out))
	} else {
		fmt.Println(string(out))
		fmt.Println("Data Cleansing completed")
	}

	number := processing("ckc", outputFile, customFile)

	if !skipDNS {
		fmt.Println("Starting DNS NXDOMAIN filtering...")
		dnsConfigPath := "./setting/dns_config.json"
		whitelistPath := "./setting/whitelist.txt"
		filteredOutput := outputFile + ".filtered"
		RunDNSFilter(outputFile, filteredOutput, dnsConfigPath, whitelistPath, tier)

		if _, err := os.Stat(filteredOutput); err == nil {
			remove(outputFile)
			if err := os.Rename(filteredOutput, outputFile); err != nil {
				fmt.Println("Error renaming filtered file:", err)
			} else {
				finalData, err := os.ReadFile(outputFile)
				if err == nil {
					finalLines := len(strings.Split(strings.TrimSpace(string(finalData)), "\n"))
					fmt.Printf("Final rule count after DNS filtering: %d\n", finalLines)
				}
			}
		} else {
			fmt.Println("Warning: Filtered file not created, using original")
		}
	} else {
		fmt.Println("DNS filtering skipped (--skip-dns flag)")
	}

	remove("ckc")
	remove("zjc")

	fmt.Println("###" + strconv.Itoa(number) + "###")

	fmt.Println("Generating format variants...")
	formatsConfigPath := "./setting/formats_config.json"
	config := LoadFormatsConfig(formatsConfigPath)

	outputDir := "./publish/"
	rulesDir := "./rules/"

	os.MkdirAll(outputDir, 0755)
	os.MkdirAll(rulesDir, 0755)

	ConvertFormat(outputFile, outputDir, tier, config)
	AddHeaders(outputDir, config, rulesDir, tier)

	for _, format := range config.Formats {
		var suffix string
		switch format {
		case "txt":
			suffix = ".txt"
		case "hosts":
			suffix = "_hosts.txt"
		case "dnsmasq":
			suffix = "_dnsmasq.txt"
		case "unbound":
			suffix = "_unbound.txt"
		case "ublock":
			suffix = "_ublock.txt"
		case "domains":
			suffix = "_domains.txt"
		default:
			continue
		}
		src := filepath.Join(outputDir, "output_"+tier+suffix)
		dst := filepath.Join(rulesDir, "output_"+tier+suffix)
		data, err := os.ReadFile(src)
		if err == nil {
			if err := os.WriteFile(dst, data, 0644); err != nil {
				fmt.Printf("Warning: Failed to copy %s to rules: %v\n", format, err)
			}
		}
	}

	fmt.Println("Format generation completed!")
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FormatsConfig struct {
	Formats       []string `json:"formats"`
	ReleaseAssets []string `json:"release_assets"`
	Tiers         []string `json:"tiers"`
}

func LoadFormatsConfig(path string) FormatsConfig {
	defaultConfig := FormatsConfig{
		Formats:       []string{"txt", "hosts", "dnsmasq", "unbound", "ublock", "domains"},
		ReleaseAssets: []string{"txt", "hosts", "dnsmasq"},
		Tiers:         []string{"super", "full", "medium", "min"},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("Formats config not found, using defaults:", err)
		return defaultConfig
	}

	err = json.Unmarshal(data, &defaultConfig)
	if err != nil {
		fmt.Println("Error parsing formats config, using defaults:", err)
		return defaultConfig
	}

	return defaultConfig
}

func ToHosts(domains []string) []string {
	result := make([]string, 0, len(domains))
	for _, d := range domains {
		result = append(result, "0.0.0.0 "+d)
	}
	return result
}

func ToDnsmasq(domains []string) []string {
	result := make([]string, 0, len(domains))
	for _, d := range domains {
		result = append(result, "address=/"+d+"/#")
	}
	return result
}

func ToUnbound(domains []string) []string {
	result := make([]string, 0, len(domains))
	for _, d := range domains {
		result = append(result, `local-zone: "`+d+`" NXDOMAIN`)
	}
	return result
}

func ToUBlock(domains []string) []string {
	result := make([]string, 0, len(domains))
	for _, d := range domains {
		result = append(result, "||"+d+"^")
	}
	return result
}

func ToDomainList(domains []string) []string {
	result := make([]string, len(domains))
	copy(result, domains)
	return result
}

func ConvertFormat(inputFile string, outputDir string, tier string, config FormatsConfig) {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Println("Error reading input file:", err)
		return
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Println("Error creating output directory:", err)
		return
	}

	lines := strings.Split(string(data), "\n")
	domains := ExtractDomains(lines)
	fmt.Printf("Extracted %d unique domains from %s\n", len(domains), inputFile)

	for _, format := range config.Formats {
		var converted []string
		var outputFile string

		switch format {
		case "txt":
			converted = lines
			outputFile = filepath.Join(outputDir, "output_"+tier+".txt")
		case "hosts":
			converted = ToHosts(domains)
			outputFile = filepath.Join(outputDir, "output_"+tier+"_hosts.txt")
		case "dnsmasq":
			converted = ToDnsmasq(domains)
			outputFile = filepath.Join(outputDir, "output_"+tier+"_dnsmasq.txt")
		case "unbound":
			converted = ToUnbound(domains)
			outputFile = filepath.Join(outputDir, "output_"+tier+"_unbound.txt")
		case "ublock":
			converted = ToUBlock(domains)
			outputFile = filepath.Join(outputDir, "output_"+tier+"_ublock.txt")
		case "domains":
			converted = ToDomainList(domains)
			outputFile = filepath.Join(outputDir, "output_"+tier+"_domains.txt")
		default:
			fmt.Println("Unknown format:", format)
			continue
		}

		output := strings.Join(converted, "\n")
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			fmt.Println("Error writing", outputFile, ":", err)
		} else {
			fmt.Printf("Created %s (%d lines)\n", outputFile, len(converted))
		}
	}
}

func AddHeaders(outputDir string, config FormatsConfig, rulesDir string, currentTier string) {
	tiers := config.Tiers
	if currentTier != "" {
		tiers = []string{currentTier}
	}
	for _, tier := range tiers {
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

			outputFile := filepath.Join(outputDir, "output_"+tier+suffix)
			rulesFile := filepath.Join(rulesDir, "output_"+tier+suffix)

			data, err := os.ReadFile(outputFile)
			if err != nil {
				continue
			}

			lines := strings.Split(string(data), "\n")
			lineCount := len(lines)

			rulesListFile := filepath.Join("setting", tier+"_rules.txt")
			var sourceLinks []string
			linkData, err := os.ReadFile(rulesListFile)
			if err == nil {
				for _, link := range strings.Split(string(linkData), "\n") {
					link = strings.TrimSpace(link)
					if link != "" && !strings.HasPrefix(link, "#") {
						sourceLinks = append(sourceLinks, link)
					}
				}
			}

			var header []string
			title := FormatTitle(tier, format)
			header = append(header, "# Title: "+title)
			header = append(header, "# Version: "+time.Now().Format("200601021504"))
			header = append(header, "# Expires: around 1 day")
			header = append(header, "# Total number of rules = "+fmt.Sprintf("%d", lineCount))

			if len(sourceLinks) > 0 {
				header = append(header, "# Below are the links of the adblock rules")
				for _, link := range sourceLinks {
					header = append(header, "# "+link)
				}
			}

			header = append(header, "####################################################################################")

			output := strings.Join(header, "\n") + "\n" + strings.Join(lines, "\n") + "\n####################################################################################\n"

			if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
				fmt.Printf("Warning: Failed to write %s: %v\n", outputFile, err)
			}

			if err := os.MkdirAll(rulesDir, 0755); err == nil {
				if err := os.WriteFile(rulesFile, []byte(output), 0644); err != nil {
					fmt.Printf("Warning: Failed to write %s: %v\n", rulesFile, err)
				}
			}
		}
	}
}

func FormatTitle(tier string, format string) string {
	titles := map[string]string{
		"super":  "5whys SUPER block",
		"full":   "5whys Full block",
		"medium": "5whys Medium",
		"min":    "5whys Min",
	}

	formats := map[string]string{
		"txt":     "Adguard Home Rules List",
		"hosts":   "Hosts Format Rules List",
		"dnsmasq": "dnsmasq Format Rules List",
		"unbound": "Unbound Format Rules List",
		"ublock":  "uBlock Origin Format Rules List",
		"domains": "Domain-only List",
	}

	title := titles[tier]
	if title == "" {
		title = "5whys " + tier
	}

	formatTitle := formats[format]
	if formatTitle == "" {
		formatTitle = format + " Rules List"
	}

	if format == "txt" && tier == "super" {
		return title + " " + formatTitle + " (Use with own risk)"
	}

	return title + " " + formatTitle
}
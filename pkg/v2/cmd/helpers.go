package cmd

import "strings"

// parseEndpoints parses comma-separated endpoints into a slice
func parseEndpoints(s string) []string {
	if s == "" {
		return nil
	}

	var endpoints []string
	for _, endpoint := range strings.Split(s, ",") {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint != "" {
			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints
}

// parseCSV parses a comma-separated string into a slice
// Alias for parseEndpoints for backward compatibility
func parseCSV(s string) []string {
	return parseEndpoints(s)
}

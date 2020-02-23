package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func textResponse(w http.ResponseWriter, code int, message string) error {
	w.WriteHeader(code)
	_, err := w.Write([]byte(message))
	return err
}

func errorResponse(w http.ResponseWriter, err error) error {
	return textResponse(w, 500, fmt.Sprintf("Error: %v", err))
}

func redirect(w http.ResponseWriter, location string) {
	w.Header().Add("location", location)
	w.WriteHeader(302)
}

func setContentTypeFromPath(w http.ResponseWriter, path string) {
	contentType := "text/plain"

	switch {
	case strings.HasSuffix(path, ".html"):
		contentType = "text/html"
	case strings.HasSuffix(path, ".svg"):
		contentType = "image/svg+xml"
	}
	w.Header().Set("content-type", contentType)
}

func requireParam(param string, vars url.Values) (string, error) {
	value := vars.Get(param)
	if value == "" {
		return "", fmt.Errorf("Expected parameter %q", param)
	}
	return value, nil
}

func handleAddPlot(vars url.Values) error {
	index, err := requireParam("index", vars)
	if err != nil {
		return err
	}
	metric, err := requireParam("metric", vars)
	if err != nil {
		return err
	}
	method, err := requireParam("method", vars)
	if err != nil {
		return err
	}

	periodString, err := requireParam("period", vars)
	if err != nil {
		return err
	}
	windowString, err := requireParam("window", vars)
	if err != nil {
		return err
	}

	period, err := time.ParseDuration(periodString)
	if err != nil {
		return fmt.Errorf("Failed to parse period: %w", err)
	}
	window, err := time.ParseDuration(windowString)
	if err != nil {
		return fmt.Errorf("Failed to parse window: %w", err)
	}

	plotType, err := requireParam("type", vars)
	if err != nil {
		return err
	}

	err = addPlot(index, metric, method, period, window, plotType)

	return err
}

func handleRemovePlot(vars url.Values) error {
	id, err := requireParam("id", vars)
	if err != nil {
		return err
	}
	err = removePlot(id)

	return err
}

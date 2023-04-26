package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/go-logr/stdr"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/example/namedtracer/foo"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	fooKey     = attribute.Key("ex.com/foo")
	barKey     = attribute.Key("ex.com/bar")
	anotherKey = attribute.Key("ex.com/another")
)

var tp *sdktrace.TracerProvider

// newExporter returns a console exporter.

func newExporter(w io.Writer) (sdktrace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		// Use human-readable output.
		stdouttrace.WithPrettyPrint(),
		// Do not print timestamps for the demo.
		stdouttrace.WithoutTimestamps(),
	)
}

// initTracer creates and registers trace provider instance.
func initTracer() error {
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return fmt.Errorf("failed to initialize stdouttrace exporter: %w", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tp)
	return nil
}

func main() {
	stdr.SetVerbosity(5)

	// Set up the logger
	logFile := &lumberjack.Logger{
		Filename:   "focusterms.log",
		MaxSize:    1, // In megabytes
		MaxBackups: 0,
		MaxAge:     365, // In days
	}
	defer logFile.Close()
	logger := log.New(logFile, "", log.Ldate|log.Ltime)

	// initialize trace provider.
	if err := initTracer(); err != nil {
		log.Panic(err)
	}

	// Create a named tracer with package path as its name.
	tracer := tp.Tracer("example/namedtracer/main")
	ctx := context.Background()
	defer func() { _ = tp.Shutdown(ctx) }()

	m0, _ := baggage.NewMember(string(fooKey), "foo1")
	m1, _ := baggage.NewMember(string(barKey), "bar1")
	b, _ := baggage.New(m0, m1)
	ctx = baggage.ContextWithBaggage(ctx, b)

	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting working directory:", err)
		return
	}

	dataPath := filepath.Join(wd, "ec2-instance-metadata.json")

	err = os.Remove(dataPath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println("Error deleting file:", err)
		}
	}
	logger.Printf("%s successfully deleted", dataPath)

	// Make the HTTP request to the metadata service
	url := "http://169.254.169.254/latest/dynamic/instance-identity/document"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Fatalf("Error creating HTTP request: %s", err)
	}

	logger.Printf("Fetching from url: %s", url)
	client := &http.Client{
		Timeout: time.Second * 2,
	}

	var span trace.Span
	ctx, span = tracer.Start(ctx, "operation")

	resp, err := client.Do(req)
	if err != nil {
		logger.Fatalf("Error making HTTP request: %s", err)
	}
	defer resp.Body.Close()

	defer span.End()
	span.AddEvent(
		"Nice operation!",
		trace.WithAttributes(attribute.Int("bogons", 100)),
	)
	span.SetAttributes(anotherKey.String("yes"))
	if err := foo.SubOperation(ctx); err != nil {
		panic(err)
	}

	// Read the response body and parse the JSON data
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Fatalf("Error reading response body: %s", err)
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Fatalf("Error parsing JSON data: %s", err)
	}

	// Pretty print the JSON and write it to a file
	jsonStr, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		logger.Fatalf("Error pretty-printing JSON: %s", err)
	}

	err = ioutil.WriteFile(dataPath, jsonStr, 0o644)
	if err != nil {
		logger.Fatalf("Error writing JSON to file: %s", err)
	}

	msg := "Successfully fetched instance metadata and wrote it to file"
	msg = fmt.Sprintf("%s %s", msg, dataPath)

	// Log a success message
	fmt.Printf(string(jsonStr))
	logger.Printf(msg)
}

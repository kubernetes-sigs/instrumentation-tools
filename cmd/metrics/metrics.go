/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"sort"
	"strings"
	"sync"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/fatih/color"
	"github.com/gdamore/tcell"
	_ "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	promtime "github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/promql"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/instrumentation-tools/cmd/cli"
	"sigs.k8s.io/instrumentation-tools/notstdlib/sets"
	"sigs.k8s.io/instrumentation-tools/promq/autocomplete/earley"
	"sigs.k8s.io/instrumentation-tools/promq/prom"
	"sigs.k8s.io/instrumentation-tools/promq/term"
	"sigs.k8s.io/instrumentation-tools/promq/term/plot"
)

type MetricsCommand struct {
	cli.PromQCommand
	Period       time.Duration
	Window       time.Duration
	outputFormat string
	sources      DataSources
}

const (
	maxSizeOfLabelContainer = 30
)

var (
	exitStrings = sets.NewString("q", "quit", "exit")
	yellow      = color.New(color.FgYellow).SprintFunc()
	cyan        = color.New(color.FgHiCyan).SprintFunc()
)

type DataSources struct {
	sources []prom.DataSource
}

func (d DataSources) ScrapePrometheusEndpoint(ctx context.Context, ts time.Time) ([]prom.ParsedSeries, error) {
	accumMetrics := make([]prom.ParsedSeries, 0)
	for _, src := range d.sources {
		m, err := src.ScrapePrometheusEndpoint(ctx, ts)
		if err != nil {
			return accumMetrics, err
		}
		accumMetrics = append(accumMetrics, m...)
	}

	return accumMetrics, nil
}

func (c *MetricsCommand) getClient() (*http.Client, error) {
	rt, err := rest.TransportFor(c.RestConfig)
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: rt}, nil
}

type httpSource struct {
	url    string
	client *http.Client
}

func (s *httpSource) getInstanceLabel() map[string]string {
	return map[string]string{
		labels.InstanceName: s.url,
	}
}

func (s *httpSource) ScrapePrometheusEndpoint(ctx context.Context, nowish time.Time) ([]prom.ParsedSeries, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.url, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to construct metrics HTTP request: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch raw metrics data: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read metrics response body: %w", err)
	}

	metrics, err := prom.ParseTextDataWithAdditionalLabels(body, nowish, s.getInstanceLabel())
	if err != nil {
		return nil, fmt.Errorf("unable to parse metrics: %w", err)
	}
	return metrics, nil
}

// this the the hook for the interactive prompt, if we detect an exit string
// we quit the program, otherwise we will invoke a function with the query
// string.
func promptExecutor(execFunc func(string)) prompt.Executor {
	return func(qs string) {
		if cli.ExitFunc(qs) {
			return
		}
		execFunc(qs)
	}
}
func (c *MetricsCommand) setupSources(flags cli.PromQFlags) error {
	client, err := c.getClient()
	if err != nil {
		return err
	}
	sources := make([]prom.DataSource, len(flags.HostNames))
	for i, url := range flags.HostNames {
		src := &httpSource{url: url, client: client}
		sources[i] = src
	}
	if len(sources) == 0 {
		kubeCfgHost := metricsURL(c.RestConfig.Host)
		sources = append(sources, &httpSource{url: kubeCfgHost, client: client})
	}
	c.sources = DataSources{
		sources: sources,
	}
	return nil
}

func (c *MetricsCommand) Run(flags cli.PromQFlags) error {
	c.outputFormat = flags.Output
	if err := c.setupSources(flags); err != nil {
		return err
	}

	metrics, err := c.sources.ScrapePrometheusEndpoint(context.Background(), time.Now())
	if err != nil {
		return err
	}

	if flags.List {
		return c.outputMetricNames(metrics)
	}
	query := flags.PromQuery
	timeoutDur := c.Period
	runner := prom.NewPeriodicData(c.sources, prom.DefaultEngineOptions(timeoutDur, 100000))

	ctx := context.Background()
	runner.Times = prom.Range{
		Window:   c.Window,
		Interval: c.Period,
		Instant:  !flags.Continuous,
	}

	// asyncronously trigger scrape
	go c.scrape(ctx, runner)

	//c := NewPromQLCompleter(index)
	if flags.Continuous {
		// it's valid, let's try drawing
		if err := c.runInteractiveChart(ctx, runner, query); err != nil {
			return err
		}
	} else {
		if query == "" {
			query = "{__name__=~\"..*\"}" // match everything
		}
		if err := runner.SetQuery(ctx, query); err != nil {
			return err
		}
		// set our callback for our prom engine runner
		// to an output format string, since we're only
		// going to return actual datapoints
		runner.Callback = func(res *promql.Result) error {
			if res.Err != nil {
				return res.Err
			}
			o, err := prom.ToPrettyFormat(res, c.outputFormat, true)
			if err != nil {
				return err
			}
			c.Fprintf("%s\n", *o)
			return nil
		}
		// trigger a scrape
		if err := runner.Scrape(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *MetricsCommand) triggerPrompt(ctx context.Context, runner *prom.PeriodicData, timeoutDur time.Duration, updateText chan string, comp func(prompt.Document) []prompt.Suggest) {
	p := prompt.New(
		// this is the thing that gets called when 'enter' is pressed
		promptExecutor(
			func(qs string) {
				if err := runner.SetQuery(ctx, qs); err != nil {
					c.triggerPrompt(ctx, runner, timeoutDur, updateText, comp)
				}
			}),
		comp,
		prompt.OptionTitle("pq: interactive cli querying against arbitrary prometheus endpoints"),
		prompt.OptionPrefix(">>> "),
		prompt.OptionCompletionWordSeparator(PromQLTokenSeparators),
		prompt.OptionPrefixTextColor(prompt.Cyan),
		prompt.OptionInputTextColor(prompt.Yellow))
	// wait for input
	p.Run()
}

func (c *MetricsCommand) outputMetricNames(metrics []prom.ParsedSeries) error {
	metricNames := sets.NewString()
	for _, m := range metrics {
		// get metric name
		if n, ok := m.Labels.Map()[labels.MetricName]; ok {
			// let's get the labels as a map
			labelsMap := m.Labels.Map()
			// delete the name entry since that's the metric name
			delete(labelsMap, labels.MetricName)
			// let's also use this opp to delete the instance name key, since we're populating this ourselves.
			delete(labelsMap, labels.InstanceName)
			// create a ordered list from the keys
			keys := getSortedKeys(labelsMap)
			// create our output string for this metric. use set for uniqueness
			metricNames.Insert(fmt.Sprintf("%s { %s }", cyan(n), yellow(strings.Join(keys, ", "))))
		}
	}
	// iterate through a sorted list of our set (our output is deterministic).
	for _, n := range metricNames.List() {
		c.Fprintf("--%v\n", n)
	}
	return nil
}

func getSortedKeys(m map[string]string) []string {
	keys := make([]string, 0)
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// we are going to assume that the query here is valid
func (c *MetricsCommand) runInteractiveChart(ctx context.Context, runner *prom.PeriodicData, qs string) error {
	ac := NewCompleter(earley.NewPromQLCompleter(runner.GetIndex()))
	comp := ac.Complete

	makeView := func(promptView term.View, keyView term.View, graph *plot.PlatonicGraph, keySize int) *term.SplitView {
		if keyView == nil {
			keyView = &term.TextBox{}
		}
		graphView := &term.GraphView{
			Graph: graph,
			RangeLabeler: func(v float64) string {
				return fmt.Sprintf("%5.5g", v)
			},
			DomainLabeler: func(v int64) string {
				// figure out a sane date format based on the window size
				// NB: time.Format uses a "canonical time" of 1 2 3 4 5 6 -7,
				// because this is clearly easier to read out of context than
				// mm:ss and such :-/
				switch {
				case c.Window >= 10*24*time.Hour:
					// span is in days, show month/day
					return promtime.Time(v).Format("Jan _2")
				case c.Window >= 24*time.Hour:
					// span is in short number of days, show day/hour
					return promtime.Time(v).Format("_2 15h")
				case c.Window >= 1*time.Hour:
					// span is in hours, show hours/minutes
					return promtime.Time(v).Format("15:04")
				case c.Window >= 1*time.Minute:
					// span is in minutes, show minutes/seconds
					return promtime.Time(v).Format("04:05")
				default:
					// otherwise show raw timestamp
					return fmt.Sprintf("%dms", v)
				}
			},
		}
		return &term.SplitView{
			DockSize: 9,
			Dock:     term.PosBelow,
			Docked:   promptView,
			Flexed: &term.SplitView{
				Docked: keyView,
				Flexed: graphView,

				Dock:           term.PosLeft,
				DockSize:       keySize,
				DockMaxPercent: 20,
			},
		}
	}

	var axesMu sync.Mutex
	lastAxes := plot.AutoAxes()

	promptView := &term.PromptView{
		SetupPrompt: func(requiredOpts ...prompt.Option) *prompt.Prompt {
			opts := []prompt.Option{
				prompt.OptionPrefix(">>> "),
				prompt.OptionCompletionWordSeparator(PromQLTokenSeparators),
				prompt.OptionPrefixTextColor(prompt.Cyan),
				prompt.OptionInputTextColor(prompt.Yellow),
			}
			opts = append(opts, requiredOpts...)

			return prompt.New(nil, comp, opts...)
		},
		HandleInput: func(input string) (*string, bool) {
			if input == "" {
				return nil, false
			}
			if input[0] == ':' {
				switch input {
				case ":quit", ":q":
					return nil, true
				default:
					msg := fmt.Sprintf("no known command %q (hint: try %q)\n", input, ":quit")
					return &msg, false
				}
			}

			if err := runner.SetQuery(ctx, input); err != nil {
				msg := fmt.Sprintf("Unable to set query: %v\n", err)
				return &msg, false
			}
			axesMu.Lock()
			lastAxes = plot.AutoAxes() // reset the axes when we change query
			axesMu.Unlock()

			msg := fmt.Sprintf("Plotting %q...\n", input)
			if input == "quit" {
				msg += "(hint: use \":quit\" to quit)\n"
			}

			// don't draw a graph now -- we'll draw it on the next update via the callback
			return &msg, false
		},
	}

	termRunner := &term.Runner{
		KeyHandler: promptView.HandleKey,
	}
	promptView.Screen = termRunner

	runner.Callback = func(res *promql.Result) error {
		// expecting a matrix
		_, err := res.Matrix()
		if err != nil {
			// TODO: signal to terminal
			return err
		}
		// transform data in a better structure.
		seriesSet, err := PromResultToPromSeriesSet(res)
		if err != nil {
			return err
		}

		for _, warning := range res.Warnings {
			// TODO: signal to terminal
			c.Fprintf("Warning running query: %v", warning)
		}

		// write to our lc object with all the label and chart information.
		axesMu.Lock()
		platGraph := plot.DataToPlatonicGraph(seriesSet, plot.AutoAxes().WithPreviousRange(lastAxes))
		lastAxes = platGraph.PlatonicAxes
		axesMu.Unlock()

		// size key
		maxSize := 1
		for _, series := range seriesSet {
			title := series.Title()
			if len(title)+3 > maxSize {
				maxSize = len(title) + 3
			}
		}
		// TODO(sollyross): cap this to a reasonable width, and wrap after

		keyView := &term.TextBox{}
		mainView := makeView(promptView, keyView, platGraph, maxSize)

		// set key
		for _, series := range seriesSet {
			title := series.Title()
			sty := tcell.StyleDefault.Foreground(tcell.Color(series.Id() % 256))
			keyView.WriteString("• ", sty)
			keyView.WriteString(title, sty)
			keyView.WriteString("\n\n", tcell.StyleDefault)
		}

		// and request that we redraw everything
		termRunner.RequestUpdate(mainView)

		return nil
	}

	ctx, stopScreen := context.WithCancel(ctx)
	go promptView.Run(ctx, &qs, stopScreen)

	if err := termRunner.Run(ctx, makeView(promptView, nil, nil, 10)); err != nil {
		return err
	}
	return nil
}

func (c *MetricsCommand) scrape(ctx context.Context, runner *prom.PeriodicData) error {
	ticker := time.NewTicker(c.Period)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := runner.Scrape(ctx); err != nil {
				// TODO: send to terminal
				// let's swallow this one, we can try next scrape.
				//return fmt.Errorf("unable to run periodic query: %w", err)
			}
		}
	}
}

func metricsURL(endpoint string) string {
	return fmt.Sprintf("%s/metrics", endpoint)
}

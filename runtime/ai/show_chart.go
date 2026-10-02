package ai

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"github.com/rilldata/rill/runtime"
	"github.com/rilldata/rill/runtime/metricsview"
)

// Bratrax: show_chart draws a chart inside Claude or ChatGPT.
//
// It is the external-client counterpart of create_chart. create_chart returns a
// spec the Bratrax app renders by querying the runtime from the browser; an MCP
// App iframe inside Claude has no session to do that with. So this tool runs
// the query itself and returns the rows inline, and the widget (the
// ui://bratrax/chart resource, see mcp_widget.go) draws them without making a
// single network call: no widget token to mint, no CSP domains to declare.
//
// The input is deliberately small and flat, not the full metrics query schema:
// one metrics view, up to four measures, and either a time series or a
// breakdown by one dimension. That covers the charts a merchant asks for
// ("ROAS by week", "spend by channel") and keeps the schema free of $ref.

const (
	ShowChartName          = "show_chart"
	showChartWidgetURI     = "ui://bratrax/chart"
	showChartMaxMeasures   = 4
	showChartMaxPoints     = 400
	showChartDefaultLimit  = 10
	showChartMaxCategories = 25
)

type ShowChart struct {
	Runtime *runtime.Runtime
}

var _ Tool[*ShowChartArgs, *ShowChartResult] = (*ShowChart)(nil)

type ShowChartArgs struct {
	MetricsView string   `json:"metrics_view" jsonschema:"Metrics view to chart, as returned by list_metrics_views."`
	Measures    []string `json:"measures" jsonschema:"One to four measure names from the metrics view. Each is drawn as its own panel."`
	Dimension   string   `json:"dimension,omitempty" jsonschema:"Optional dimension to break the measures down by, drawn as a bar chart of the top values. Omit it to draw a time series instead."`
	TimeGrain   string   `json:"time_grain,omitempty" jsonschema:"Time series grain: day, week or month. Default day. Ignored for a breakdown."`
	Period      string   `json:"period,omitempty" jsonschema:"ISO 8601 duration ending at the latest data, such as P7D, P30D, P3M or P1Y. Default P30D. Ignored when start and end are given."`
	Start       string   `json:"start,omitempty" jsonschema:"Optional inclusive start date, YYYY-MM-DD. Use together with end."`
	End         string   `json:"end,omitempty" jsonschema:"Optional exclusive end date, YYYY-MM-DD. Use together with start."`
	TimeZone    string   `json:"time_zone,omitempty" jsonschema:"IANA time zone for day boundaries, such as America/New_York. Use the store's time zone when you know it. Default UTC."`
	Limit       int      `json:"limit,omitempty" jsonschema:"For a breakdown: how many top values to show, 1 to 25. Default 10."`
	Title       string   `json:"title,omitempty" jsonschema:"Optional chart title. Defaults to the measure names."`
}

type ShowChartResult struct {
	Title       string            `json:"title"`
	ChartType   string            `json:"chart_type"`
	XField      string            `json:"x_field"`
	XType       string            `json:"x_type"`
	XLabel      string            `json:"x_label"`
	TimeGrain   string            `json:"time_grain,omitempty"`
	TimeZone    string            `json:"time_zone"`
	Series      []ShowChartSeries `json:"series"`
	Rows        []map[string]any  `json:"rows"`
	PeriodLabel string            `json:"period_label,omitempty"`
	OpenURL     string            `json:"open_url,omitempty"`
	Note        string            `json:"note,omitempty"`
}

type ShowChartSeries struct {
	Field  string `json:"field"`
	Label  string `json:"label"`
	Format string `json:"format,omitempty"`
}

func (t *ShowChart) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:  ShowChartName,
		Title: "Show chart",
		Description: "Draws an interactive chart of store and ad performance from a metrics view query. " +
			"Without a dimension it plots the measures as a time series; with a dimension it draws a bar chart of that dimension's top values. " +
			"Metrics view, measure and dimension names are the ones returned by list_metrics_views and get_metrics_view. " +
			"The result also contains the plotted rows.",
		Meta: map[string]any{
			// MCP Apps: the host renders this resource with the tool result.
			// The flat key is the legacy spelling some hosts still read.
			"ui":             map[string]any{"resourceUri": showChartWidgetURI},
			"ui/resourceUri": showChartWidgetURI,
		},
	}
}

func (t *ShowChart) CheckAccess(ctx context.Context) (bool, error) {
	s := GetSession(ctx)
	// External MCP clients only: inside the Bratrax app, create_chart renders
	// natively and this tool's widget has nowhere to render.
	if strings.HasPrefix(s.CatalogSession().UserAgent, "rill") {
		return false, nil
	}
	return s.Claims().Can(runtime.ReadMetrics), nil
}

func (t *ShowChart) Handler(ctx context.Context, args *ShowChartArgs) (*ShowChartResult, error) {
	session := GetSession(ctx)
	if args.MetricsView == "" {
		return nil, errors.New("metrics_view is required; call list_metrics_views to find one")
	}
	if len(args.Measures) == 0 || len(args.Measures) > showChartMaxMeasures {
		return nil, fmt.Errorf("pass between 1 and %d measures", showChartMaxMeasures)
	}
	tz := args.TimeZone
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("unknown time_zone %q; use an IANA name such as America/New_York", tz)
	}

	spec, err := t.metricsViewSpec(ctx, session, args.MetricsView)
	if err != nil {
		return nil, err
	}

	res := &ShowChartResult{TimeZone: tz}
	measures := make([]map[string]any, 0, len(args.Measures))
	for _, name := range args.Measures {
		m := findMeasure(spec, name)
		if m == nil {
			return nil, fmt.Errorf("measure %q not found in %q; available: %s", name, args.MetricsView, strings.Join(measureNames(spec), ", "))
		}
		label := m.GetDisplayName()
		if label == "" {
			label = name
		}
		res.Series = append(res.Series, ShowChartSeries{Field: name, Label: label, Format: m.GetFormatPreset()})
		measures = append(measures, map[string]any{"name": name})
	}

	query := map[string]any{
		"metrics_view": args.MetricsView,
		"measures":     measures,
		"time_zone":    tz,
	}
	timeRange, periodLabel, err := chartTimeRange(args, loc)
	if err != nil {
		return nil, err
	}
	query["time_range"] = timeRange
	res.PeriodLabel = periodLabel

	var limit int
	if args.Dimension == "" {
		timeDim := spec.GetTimeDimension()
		if timeDim == "" {
			return nil, fmt.Errorf("%q has no time dimension; pass a dimension to chart a breakdown instead", args.MetricsView)
		}
		grain := args.TimeGrain
		if grain == "" {
			grain = string(metricsview.TimeGrainDay)
		}
		if !slices.Contains([]string{"day", "week", "month"}, grain) {
			return nil, errors.New("time_grain must be day, week or month")
		}
		// time_floor buckets the time dimension to the grain; the output column
		// takes the dimension's own name.
		query["dimensions"] = []map[string]any{{
			"name":    timeDim,
			"compute": map[string]any{"time_floor": map[string]any{"dimension": timeDim, "grain": grain}},
		}}
		query["sort"] = []map[string]any{{"name": timeDim, "desc": false}}
		limit = showChartMaxPoints
		res.ChartType, res.XType, res.XField, res.XLabel, res.TimeGrain = "line", "time", timeDim, "Date", grain
	} else {
		d := findDimension(spec, args.Dimension)
		if d == nil {
			return nil, fmt.Errorf("dimension %q not found in %q; available: %s", args.Dimension, args.MetricsView, strings.Join(dimensionNames(spec), ", "))
		}
		limit = args.Limit
		if limit <= 0 {
			limit = showChartDefaultLimit
		}
		limit = min(limit, showChartMaxCategories)
		query["dimensions"] = []map[string]any{{"name": args.Dimension}}
		query["sort"] = []map[string]any{{"name": args.Measures[0], "desc": true}}
		label := d.GetDisplayName()
		if label == "" {
			label = args.Dimension
		}
		res.ChartType, res.XType, res.XField, res.XLabel = "bar", "category", args.Dimension, label
	}
	// One extra row tells us whether the result was cut off.
	query["limit"] = limit + 1

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	rr, err := t.Runtime.Resolve(ctx, &runtime.ResolveOptions{
		InstanceID:         session.InstanceID(),
		Resolver:           "metrics",
		ResolverProperties: query,
		Claims:             session.Claims(),
	})
	if err != nil {
		return nil, err
	}
	defer rr.Close()
	schema, data, err := resolverResultToTabular(rr)
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		data = data[:limit]
		if args.Dimension == "" {
			res.Note = fmt.Sprintf("Only the first %d points are shown; use a coarser time_grain or a shorter period.", limit)
		} else {
			res.Note = fmt.Sprintf("Showing the top %d values by %s.", limit, res.Series[0].Label)
		}
	}
	if len(data) == 0 {
		// An empty chart looks like a broken tool. Say when the data actually
		// ends, so the model can pick a period that has some: common for a
		// store whose extracts have stopped, and for demo workspaces.
		res.Note = "No data in this period."
		if latest := t.latestDataTime(ctx, session, args.MetricsView); !latest.IsZero() {
			res.Note = fmt.Sprintf("No data in this period. The most recent data is from %s; choose a period ending on or before that date.",
				latest.In(loc).Format(time.DateOnly))
		}
	}
	res.Rows = make([]map[string]any, 0, len(data))
	for _, row := range data {
		m := make(map[string]any, len(schema))
		for i, f := range schema {
			m[f.Name] = row[i]
		}
		res.Rows = append(res.Rows, m)
	}

	res.Title = args.Title
	if res.Title == "" {
		labels := make([]string, len(res.Series))
		for i, s := range res.Series {
			labels[i] = s.Label
		}
		res.Title = strings.Join(labels, ", ")
		if args.Dimension != "" {
			res.Title += " by " + res.XLabel
		}
	}
	if inst, err := t.Runtime.Instance(ctx, session.InstanceID()); err == nil && inst.FrontendURL != "" {
		res.OpenURL = strings.TrimSuffix(inst.FrontendURL, "/")
	}
	return res, nil
}

func (t *ShowChart) metricsViewSpec(ctx context.Context, session *Session, name string) (*runtimev1.MetricsViewSpec, error) {
	ctrl, err := t.Runtime.Controller(ctx, session.InstanceID())
	if err != nil {
		return nil, err
	}
	r, err := ctrl.Get(ctx, &runtimev1.ResourceName{Kind: runtime.ResourceKindMetricsView, Name: name}, false)
	if err != nil {
		return nil, fmt.Errorf("metrics view %q not found; call list_metrics_views to see what exists", name)
	}
	r, access, err := t.Runtime.ApplySecurityPolicy(ctx, session.InstanceID(), session.Claims(), r)
	if err != nil {
		return nil, err
	}
	if !access {
		return nil, fmt.Errorf("metrics view %q not found; call list_metrics_views to see what exists", name)
	}
	spec := r.GetMetricsView().GetState().GetValidSpec()
	if spec == nil {
		return nil, fmt.Errorf("metrics view %q is not available right now", name)
	}
	return spec, nil
}

// latestDataTime returns the newest timestamp in the metrics view, or zero if
// it can't be determined. Best effort: it only improves an empty result's note.
func (t *ShowChart) latestDataTime(ctx context.Context, session *Session, metricsView string) time.Time {
	rr, err := t.Runtime.Resolve(ctx, &runtime.ResolveOptions{
		InstanceID:         session.InstanceID(),
		Resolver:           "metrics_time_range",
		ResolverProperties: map[string]any{"metrics_view": metricsView},
		Claims:             session.Claims(),
	})
	if err != nil {
		return time.Time{}
	}
	defer rr.Close()
	row, err := rr.Next()
	if err != nil {
		return time.Time{}
	}
	// Cached resolver results come back JSON-decoded, so the time may be a string.
	switch v := row["max"].(type) {
	case time.Time:
		return v
	case string:
		latest, _ := time.Parse(time.RFC3339Nano, v)
		return latest
	}
	return time.Time{}
}

// chartTimeRange builds the query's time_range and a human label for it.
// Explicit dates are midnights in the chart's time zone, so a day matches the
// day the merchant sees in their own dashboards.
func chartTimeRange(args *ShowChartArgs, loc *time.Location) (map[string]any, string, error) {
	if args.Start != "" || args.End != "" {
		start, err := time.ParseInLocation(time.DateOnly, args.Start, loc)
		if err != nil {
			return nil, "", errors.New("start must be a YYYY-MM-DD date, given together with end")
		}
		end, err := time.ParseInLocation(time.DateOnly, args.End, loc)
		if err != nil {
			return nil, "", errors.New("end must be a YYYY-MM-DD date, given together with start")
		}
		if !end.After(start) {
			return nil, "", errors.New("end must be after start")
		}
		return map[string]any{"start": start.Format(time.RFC3339), "end": end.Format(time.RFC3339)},
			args.Start + " to " + end.AddDate(0, 0, -1).Format(time.DateOnly), nil
	}
	period := args.Period
	if period == "" {
		period = "P30D"
	}
	if !strings.HasPrefix(period, "P") {
		return nil, "", fmt.Errorf("period %q must be an ISO 8601 duration such as P30D", period)
	}
	return map[string]any{"iso_duration": period}, "Last " + humanizeISODuration(period), nil
}

func humanizeISODuration(p string) string {
	units := map[byte]string{'D': "day", 'W': "week", 'M': "month", 'Y': "year"}
	body := strings.TrimPrefix(p, "P")
	if len(body) < 2 {
		return p
	}
	unit, ok := units[body[len(body)-1]]
	if !ok {
		return p
	}
	n := body[:len(body)-1]
	if n == "1" {
		return unit
	}
	return n + " " + unit + "s"
}

func findMeasure(spec *runtimev1.MetricsViewSpec, name string) *runtimev1.MetricsViewSpec_Measure {
	for _, m := range spec.GetMeasures() {
		if m.GetName() == name {
			return m
		}
	}
	return nil
}

func findDimension(spec *runtimev1.MetricsViewSpec, name string) *runtimev1.MetricsViewSpec_Dimension {
	for _, d := range spec.GetDimensions() {
		if d.GetName() == name {
			return d
		}
	}
	return nil
}

func measureNames(spec *runtimev1.MetricsViewSpec) []string {
	out := make([]string, 0, len(spec.GetMeasures()))
	for _, m := range spec.GetMeasures() {
		out = append(out, m.GetName())
	}
	return out
}

func dimensionNames(spec *runtimev1.MetricsViewSpec) []string {
	out := make([]string, 0, len(spec.GetDimensions()))
	for _, d := range spec.GetDimensions() {
		out = append(out, d.GetName())
	}
	return out
}

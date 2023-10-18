//go:build darwin
// +build darwin

// https://github.com/kolide/launcher/blob/main/pkg/osquery/tables/systemprofiler/systemprofiler.go

package systemprofiler

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	// "github.com/go-kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/groob/plist"

	// "github.com/kolide/launcher/pkg/dataflatten" Need a dataflatten table?!
	"github.com/kolide/launcher/pkg/dataflatten"
	"github.com/kolide/launcher/pkg/osquery/tables/dataflattentable"
	"github.com/osquery/osquery-go/plugin/table"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("parentdatatype"),
		table.TextColumn("datatype"),
		table.TextColumn("detaillevel"),
	}
}

const systemprofilerPath = "/usr/sbin/system_profiler"

var knownDetailLevels = []string{
	"mini",  // short report (contains no identifying or personal information)
	"basic", // basic hardware and network information
	"full",  // all available information
}

// Generate is called to return the results for the table at query time.
//
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	requestedDatatypes := []string{}

	datatypeQ, ok := queryContext.Constraints["datatype"]
	if !ok || len(datatypeQ.Constraints) == 0 {
		return results, fmt.Errorf("The %s table requires that you specify a constraint for datatype", t.tableName)
	}

	for _, datatypeConstraint := range datatypeQ.Constraints {
		dt := datatypeConstraint.Expression

		// If the constraint is the magic "%", it's eqivlent to an `all` style
		if dt == "%" {
			requestedDatatypes = []string{}
			break
		}

		requestedDatatypes = append(requestedDatatypes, dt)
	}

	var detailLevel string
	if q, ok := queryContext.Constraints["detaillevel"]; ok && len(q.Constraints) != 0 {
		if len(q.Constraints) > 1 {
			level.Info(t.logger).Log("msg", "WARNING: Only using the first detaillevel request")
		}

		dl := q.Constraints[0].Expression
		for _, known := range knownDetailLevels {
			if known == dl {
				detailLevel = dl
			}
		}

	}

	systemProfilerOutput, err := t.execSystemProfiler(ctx, detailLevel, requestedDatatypes)
	if err != nil {
		return results, fmt.Errorf("exec: %w", err)
	}

	if q, ok := queryContext.Constraints["query"]; ok && len(q.Constraints) != 0 {
		for _, constraint := range q.Constraints {
			dataQuery := constraint.Expression
			results = append(results, t.getRowsFromOutput(dataQuery, detailLevel, systemProfilerOutput)...)
		}
	} else {
		results = append(results, t.getRowsFromOutput("", detailLevel, systemProfilerOutput)...)
	}

	return results, nil
}

func (t *Table) getRowsFromOutput(dataQuery, detailLevel string, systemProfilerOutput []byte) []map[string]string {
	var results []map[string]string

	flattenOpts := []dataflatten.FlattenOpts{
		dataflatten.WithLogger(t.logger),
		dataflatten.WithQuery(strings.Split(dataQuery, "/")),
	}

	var systemProfilerResults []Result
	if err := plist.Unmarshal(systemProfilerOutput, &systemProfilerResults); err != nil {
		level.Info(t.logger).Log("msg", "error unmarshalling system_profile output", "err", err)
		return nil
	}

	for _, systemProfilerResult := range systemProfilerResults {

		dataType := systemProfilerResult.DataType

		flatData, err := dataflatten.Flatten(systemProfilerResult.Items, flattenOpts...)
		if err != nil {
			level.Info(t.logger).Log("msg", "failure flattening system_profile output", "err", err)
			continue
		}

		rowData := map[string]string{
			"datatype":       dataType,
			"parentdatatype": systemProfilerResult.ParentDataType,
			"detaillevel":    detailLevel,
		}

		results = append(results, dataflattentable.ToMap(flatData, dataQuery, rowData)...)
	}

	return results
}

func (t *Table) execSystemProfiler(ctx context.Context, detailLevel string, subcommands []string) ([]byte, error) {
	timeout := 45 * time.Second
	if detailLevel == "full" {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	args := []string{"-xml"}

	if detailLevel != "" {
		args = append(args, "-detailLevel", detailLevel)
	}

	args = append(args, subcommands...)

	cmd := exec.CommandContext(ctx, systemprofilerPath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	level.Debug(t.logger).Log("msg", "calling system_profiler", "args", cmd.Args)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("calling system_profiler. Got: %s: %w", string(stderr.Bytes()), err)
	}

	return stdout.Bytes(), nil
}

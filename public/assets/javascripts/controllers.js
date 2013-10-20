angular.module('app').controller('QueryCtrl', ['$scope', 'QueryService', function QueryCtrl($scope, qs) {

    $scope.results = [];
    $scope.selectedResult = null;

    // TODO: get from query service
    $scope.connected = true;

    $scope.resultForms = {
        0: 'no results',
        one: '{} row',
        other: '{} rows'
    };

    $scope.addResult = function(result) {
        // TODO: remove old results
        $scope.results.push(result);
        if (!$scope.selectedResult) {
            $scope.selectResult(result);
        }
    };

    $scope.selectResult = function(result) {
	console.log("selecting result: " + result.id);
        if (result != $scope.selectedResult) {
            $scope.selectedResult = result
            $scope.$broadcast('resultSelected', result);
        }
    };

    qs.onQuery(function(query) {
	$scope.addResult(query);
    });

    qs.onClose(function(reason) {
	$scope.connected = false;
    });
}]);

angular.module('app').controller('ChartCtrl', [ '$scope', function ChartCtrl($scope) {
    // TODO: a fair amount of this should probably be pushed down into a directive
    $scope.registeredCharts = []

    $scope.selectedResult = null;
    $scope.selectedChart = null;

    $scope.$on('resultSelected', function(event, queryResult) {
	console.log("update selectedResult to: " + queryResult.id);
        $scope.selectedResult = queryResult;
        if (queryResult) {
	    // TODO: drop reloads, do sticky chart selection instead
            if (canReload(queryResult)) {
                $scope.currChart.reload(queryResult);
            } else {
                // We assume that at least one chart will always be available
                $scope.selectChart($scope.availableCharts()[0]);
            }
        }
    });

    function canReload(queryResult) {
        return $scope.selectedChart && $scope.selectedChart.accepts(queryResult) &&
            typeof($scope.currChart.loadData) == 'function';
    }

    $scope.availableCharts = function() {
        return $scope.registeredCharts.filter(function(chart) {
            return $scope.selectedResult && chart.accepts($scope.selectedResult);
        });
    }

    $scope.addChart = function(chartFn) {
        $scope.registeredCharts.push(chartFn);
    }

    $scope.selectChart = function(chart) {
        window.console.log("selected " + chart.chartName);
        $scope.selectedChart = chart;
        if ($scope.selectedResult) {
            loadChart(chart, $scope.selectedResult);
        }
    }

    // load the given chart
    function loadChart(chartFn, result) {
        // TODO: reload new query into old charts for efficiency; also
        // cache multiple chart types for quick navigation of most
        // recent items recent items
	container = $('#chart-container')
        if ($scope.currChart) {
	    console.log("cleaning up old chart");
            if (typeof($scope.currChart.dispose) == 'function') {
		console.log("chart has dispose method; calling it");
                $scope.currChart.dispose();
            }
	    console.log("emptying the container of children");
            container.empty();
        }

	console.log("creating new chart");
        $scope.currChart = new chartFn(container, result);

        window.console.log("Loaded " + chartFn.chartName +
                           " with result for " + result.query);
    }

    $scope.addChart(Table);
    $scope.addChart(LineChart);
    $scope.addChart(AreaChart);

}]);

function Table(target, queryResult) {
    function zpad(numstr, len) {
        return numstr.length == len ? numstr : '0' + zpad(numstr, len - 1);
    }
    function formatDate(d) {
	// TODO: timezone info
        return d.getFullYear() + '-' +
            zpad((d.getMonth() + 1).toString(), 2) + '-' +
            zpad((d.getDate()).toString(), 2) + '&nbsp;' +
            zpad((d.getHours()).toString(), 2) + ':' +
            zpad((d.getMinutes()).toString(), 2) + ':' +
            zpad((d.getSeconds()).toString(), 2) + '.' +
            zpad((d.getMilliseconds()).toString(), 3);
    }

    function massageColumns(queryResult) {
        return queryResult.columns.map(function(column) {
            const tableColumn = {
                "sTitle": column.name
            };
            if (isNumeric(column)) {
                tableColumn.sClass = 'right-align';
		const precision = isInteger(column) ? '0' : '3'
                tableColumn.fnRender = function(obj) {
                    return obj.aData[obj.iDataColumn].toFixed(precision);
                }
            } else if (isDate(column)) {
                tableColumn.fnRender = function(obj) {
                    return formatDate(new Date(obj.aData[obj.iDataColumn]));
                }
            }
            return tableColumn;
        });
    }

    var table = $('<table class="display"></table>').appendTo(target);
    table.dataTable({
        "bDestroy": true,
        "aaData": queryResult.data,
        "aoColumns": massageColumns(queryResult)
    });

    this.dispose = function() {
	console.log("destroying old data table");
	table.dataTable().fnDestroy();
    }
}
Table.chartName = 'TABLE'
Table.accepts = function(queryResult) {
    return true;
}

chart_template = '<div id="chart_wrapper">' +
  '<div id="y_axis"></div>' +
  '<div id="chart"></div>' +
  '<div id="legend"></div>' +
'</div>';

function RickshawChart(target, queryResult) {
    var palette = new Rickshaw.Color.Palette();
    var chart = $(chart_template).appendTo(target)

    var tsCol = firstIdx(queryResult.columns, 'time.Time')
    var numCols = allIdx(queryResult.columns, 'int32', 'int64', 'float32', 'float64')
    series = numCols.map(function(numCol) {
	var remapped = remapData(queryResult.data, tsCol, numCol)
	var parsed = parseTimes(remapped)
	return {
	    name: queryResult.columns[numCol].name,
	    data: parsed,
	    color: palette.color()	    
	}
    });

    var graph = new Rickshaw.Graph({
        element: document.querySelector("#chart"),
        renderer: this.renderer,
	offset: this.offset,
        series: series
    });

    // TODO: avoid all the DOM fetches by, e.g., building the template
    // piecemeal
    var x_axis = new Rickshaw.Graph.Axis.Time({ graph: graph });
    var y_axis = new Rickshaw.Graph.Axis.Y({
        graph: graph,
        orientation: 'left',
        tickFormat: Rickshaw.Fixtures.Number.formatKMBT,
        element: document.getElementById('y_axis'),
    });

    // TODO: directive-ify or at least angular-ify
    var legend = new Rickshaw.Graph.Legend({
        element: document.querySelector('#legend'),
        graph: graph
    });

    graph.render();

    this.dispose = function() {
	console.log("destroying chart")
    }
}

function LineChart(target, queryResult) {
    this.renderer = 'line';
    this.offset = 'zero';

    RickshawChart.call(this, target, queryResult);
}

LineChart.chartName = 'LINE'
LineChart.accepts = function(queryResult) {
    var cols = queryResult.columns;
    return hasCol(cols, 'time.Time') && hasCol(cols, 'int32', 'int64', 'float32', 'float64');
}

function AreaChart(target, queryResult) {
    this.renderer = 'stack';
    this.offset = 'stack';

    RickshawChart.call(this, target, queryResult);
}

AreaChart.chartName = 'AREA'
AreaChart.accepts = function(queryResult) {
    var cols = queryResult.columns;
    return hasCol(cols, 'time.Time') && hasCol(cols, 'int32', 'int64', 'float32', 'float64');
}

function remapData(data, xIdx, yIdx) {
    return data.map(function(item) {
        return { 'x': item[xIdx], 'y': item[yIdx] };
    });
}

function parseTimes(data) {
    return data.map(function(item) {
	oldX = item['x']
	newX = (new Date(oldX).getTime() / 1000)
	console.log("mapping " + oldX + " to " + newX)
	return {
	    'x': newX, 'y': item['y']
	}
    })
}

function hasCol(cols, types) {
    for (var i = 1; i < arguments.length; i++) {
        if (firstIdx(cols, arguments[i]) > -1) {
            return true;
        }
    }
    return false;
}

function firstIdx(cols, ofType) {
    for (var i = 0; i < cols.length; i++){
        for (var j = 1; j < arguments.length; j++) {
            if (cols[i].type == arguments[j]) {
                return i;
            }
        }
    }
    return -1;
}

function allIdx(cols, ofType) {
    var result = [];
    for (var i = 0; i < cols.length; i++) {
        for (var j = 1; j < arguments.length; j++) {
            if (cols[i].type == arguments[j]) {
                result.push(i);
                // No point checking the other types for this col
                continue;
            }
        }
    }
    return result;
}

function isNumeric(column) {
    return column.type == 'int32' || column.type == 'int64' ||
	column.type == 'float32' || column.type == 'float64';
}

function isInteger(column) {
    return column.type == 'int32' || column.type == 'int64';
}

function isDate(column) {
    return column.type == 'time.Time';
}

// chart types: table / bar (clustered/stacked) / line /
//  area (stacked) / pie / scatter plot / maps / single value

// TODO:
// ! add multiple series
// add stacked area charts
// UI cleanup
//  - styling
//  - sizing
//  - buffer for chart y range
//  - additional query list interactivity
//    * lightbox for full query text
//    * explain
//    * re-run
// add csv/pdf export
// share (!)
// add maps
// multiple axes where reasonable
// add ability to select "active" fields for charts
// add ability to turn on/off fields for multi-series charts
// add stacked/clustered bar chart (toggle? or two separate?)
// add pie
// add scatter plot
// add single value
// sort series by time if not sorted for timeseries

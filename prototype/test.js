function remapData(data, xIdx, yIdx) {
    return data.map(function(item) {
        return { 'x': item[xIdx], 'y': item[yIdx] };
    });
}

function LineChart(target, queryResult) {

    function getData(queryResult) {
        xColIdx = firstTypeIdx(queryResult.columns, 'date')
        yColIdx = firstTypeIdx(queryResult.columns, 'integer', 'float')

        result = {
            values: remapData(queryResult.data, xColIdx, yColIdx),
            key: "Sine Wave",
            color: "#ff7f0e"
        };

        return [ result ];
    }

    nv.addGraph(function() {
        window.console.log("Adding line chart to " + target);
        var chart = nv.models.lineChart();

        // chart sub-models (ie. xAxis, yAxis, etc) when
        // accessed directly, return themselves, not the
        // partent chart, so need to chain separately
        chart.xAxis 
            .tickFormat(function(d) { 
                window.console.log("Got " + d);
                return d3.time.format('%x') (new Date(d))
            });

        chart.yAxis
            .axisLabel('Voltage (v)')
            .tickFormat(d3.format(',.1f'));

        $(target).append('<svg></svg>');

        d3.select(target + ' svg')
            .datum(getData(queryResult))
            .transition().duration(500)
            .call(chart);

        //TODO: eventually nvd3 may do this automatically
        nv.utils.windowResize(chart.update);

        this.nvd3chart = chart;

        return chart;
    });

    this.dispose = function() {
        // This is a little ugly right now because nvd3 does not allow
        // for clean adding / removing of charts.
        for (var i = 0; i < nv.graphs.length; i++) {
            if (nv.graphs[i] = this.nvd3chart) {
                nv.graphs.splice(index, 1);
                break;
            }
        }
    }

}

LineChart.chartName = 'Line'
LineChart.accepts = function(queryResult) {
    var cols = queryResult.columns;
    return firstTypeIdx(cols, 'date') > -1 &&
        firstTypeIdx(cols, 'integer', 'float') > -1;
}

function BarChart(width, height, target) {}
BarChart.chartName = 'Bar'
BarChart.accepts = function(queryResult) { 
    return queryResult.query.length < 15;
}

function Table(width, height, target) {}
Table.chartName = 'Table'
Table.accepts = function(queryResult) {
    return true;
}

/*function RootCtrl($scope) {
    $scope.$on('resultSelected', function(event, queryResult) {

    });
}*/

function ChartCtrl($scope) {
    $scope.registeredCharts = []

    $scope.selectedResult = null;
    $scope.selectedChart = null;

    $scope.$on('resultSelected', function(event, queryResult) {
        $scope.selectedResult = queryResult;
        if (queryResult) {
            // if the last selected chart supports this
            // query result, reuse that; otherwise, reinitialize
            if (!$scope.selectedChart || !$scope.selectedChart.accepts(queryResult)) {
                // We assume that at least one chart will always be available
                $scope.selectChart($scope.availableCharts()[0]);
            }
        }
    });

    $scope.availableCharts = function() {
        return $scope.registeredCharts.filter(function(chart) {
            return $scope.selectedResult && chart.accepts($scope.selectedResult);
        });
    }

    $scope.addChart = function(chartFn) {
        $scope.registeredCharts.push(chartFn);
    }

    $scope.selectChart = function(chart) {
        if ($scope.selectedChart && $scope.selectedChart == chart) {
            return;
        }
        
        if (chart) {
            window.console.log("selected " + chart.chartName);
            $scope.selectedChart = chart;
            if ($scope.selectedResult) {
                loadChart(chart, $scope.selectedResult);
            }
        } else {
            // error or something
        }
    }

    var chartId = 0;

    // load the given chart
    function loadChart(chartFn, result) {
        // TODO: reload new query into old charts for efficiency; also
        // cache multiple chart types for quick navigation of most
        // recent items recent items
        if ($scope.currChart) {
            if (typeof($scope.currChart.dispose) == 'function') {
                $scope.currChart.dispose();
            }
            $('#chart-container').children().remove()
        }
        nextId  = 'chart' + chartId++;
        $('#chart-container').append('<div id="' + nextId + '">');
        $scope.currChart = new chartFn('#' + nextId, result);

        window.console.log("Loaded " + chartFn.chartName +
                           " with result for " + result.query);
    }

    $scope.addChart(LineChart);
    $scope.addChart(BarChart);
    $scope.addChart(Table);
}


// Query results: let's assume we get back objects that look like this:

var results = [ {
    id: 1,
    query: "select col1, col2, col3, col4 from table_foo where x > 30",
    runtime: 213.123 /* in millis */,
    columns: [
        { name: 'col1', type: 'text' },
        { name: 'col2', type: 'integer' },
        { name: 'col3', type: 'text' },
        { name: 'col4', type: 'float' }
    ],
    data: [
        [ 'a', 1, 'c', 2.3 ],
        [ 'a', 1, 'c', 2.3 ],
        [ 'a', 1, 'c', 2.3 ]
    ]
}, {
    id: 2,
    query: 'select now()',
    runtime: 0.323 /* in millis */,
    columns: [ { name: 'time', type: 'date' } ],
    data: [ [ 123.234 /* for now, we'll just send dates as floats */ ] ]
}, {
    id: 3,
    query: 'select date, amount from invoice order by date',
    runtime: 0.334,
    columns: [ { name: 'date', type: 'date' }, { name: 'amount', type: 'float' } ],
    data: [
        [ 1025409600000, 123.2234 ],
        [ 1025419600000, 93.2234 ],
        [ 1025429600000, 103.2234 ],
        [ 1025439600000, 113.2234 ],
        [ 1025449600000, 124.2234 ],
        [ 1025459600000, 120.2234 ],
        [ 1025469600000, 111.2234 ],
        [ 1025479600000, 97.2234 ],
        [ 1025489600000, 83.2234 ]
    ]
}, {
    id: 4,
    query: 'select g from generate_series(1,10) g',
    columns: [ { name: 'g', type: 'integer' } ],
    runtime: 0.082,
    data: [
        [ 1 ],
        [ 2 ],
        [ 3 ],
        [ 4 ],
        [ 5 ],
        [ 6 ],
        [ 7 ],
        [ 8 ],
        [ 9 ],
        [ 10 ]
    ]
}];

function QueryCtrl($scope) {
    $scope.results = [];
    $scope.selectedResult = null;

    $scope.addResult = function(result) {
        // TODO: remove old results
        $scope.results.push(result);
        if (!$scope.selectedResult) {
            $scope.selectResult(result);
        }
    }

    $scope.selectResult = function(result) {
        if (result != $scope.selectedResult) {
            $scope.selectedResult = result
            $scope.$broadcast('resultSelected', result);
        }
    }

    $scope.resultForms = {
        0: 'no results',
        one: '{} row',
        other: '{} rows'
    }

    //results.forEach($scope.addResult);
    setTimeout(function() {
        $scope.$apply(function() {
            results.forEach($scope.addResult);
        });
    }, 100);
}


/*
  charts
    - line/area/bar (distinct Date vs. Number+ | String + distinct Date vs. Number)
    - scatter (Number vs. Number+ | String + Number vs. Number)
    - horizontal bar (String vs. Number+ | String + String vs. Number)
    - pie (x, y) (String vs. Number, no more than five)
    - table (any) (could also get sparklines and shit in there)

  pivoting important, but v2

*/

function firstTypeIdx(cols, ofType) {
    for (var i = 0; i < cols.length; i++){
        for (var j = 1; j < arguments.length; j++) {
            if (cols[i].type == arguments[j]) {
                return i;
            }
        }
    }
    return -1;
}

function isDistinct(data, field) {
    var values = {};
    for (var row in data) {
        var value = row[field];
        if (value in values) {
            return false;
        } else {
            values[value] = true;
        }
    }
    return true;
}

function distinctValues(data, field) {
    var result = [];
    var existing = {};
    for (var row in data) {
        var value = row[field]
        if (!(value in existing)) {
            existing[value] = true;
            result.push(value)
        }
    }
    return result;
}


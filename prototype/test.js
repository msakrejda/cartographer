
function LineChart(width, height, target) {}
LineChart.chartName = 'Line'
LineChart.accepts = function(queryResult) { return true; }

function BarChart(width, height, target) {}
BarChart.chartName = 'Bar'
BarChart.accepts = function(queryResult) { return true; }

function Table(width, height, target) {}
Table.chartName = 'Table'
Table.accepts = function(queryResult) { return queryResult.query.length < 15; }

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
        // if the last selected chart supports this
        // query result, reuse that; otherwise, reinitialize
        if (!$scope.selectedChart || !$scope.selectedChart.accepts(queryResult)) {
            // We assume that at least one chart will always be available
            $scope.selectChart($scope.availableCharts()[0]);
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
            loadChart(chart);
        } else {
            // error or something
        }
    }


    // load the given chart
    function loadChart(chartFn) {
        result = $scope.selectedResult
        // Remove old chart
        // TODO: reload new query into old charts for efficiency; also
        // cache multiple chart types for quick navigation of most
        // recent items
        /*if ($scope.currChart) {
            $scope.currChart.dispose();
        }
        $scope.currChart = new chartFn();*/
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
    setTimeout(function() { $scope.$apply(function() { results.forEach(($scope.addResult)); }) }, 100);
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


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

}]);

function Table(target, queryResult) {
    function zpad(numstr, len) {
        return numstr.length == len ? numstr : '0' + zpad(numstr, len - 1);
    }
    function formatDate(d) {
        return d.getFullYear() + '-' +
            zpad((d.getMonth() + 1).toString(), 2) + '-' +
            zpad((d.getDate()).toString(), 2) + ' ' +
            zpad((d.getHours()).toString(), 2) + ':' +
            zpad((d.getMinutes()).toString(), 2) + ':' +
            zpad((d.getSeconds()).toString(), 2) + '.' +
            zpad((d.getMilliseconds()).toString(), 3);
    }

    function massageColumns(queryResult) {
        return queryResult.columns.map(function(column) {
            var tableColumn = {
                "sTitle": column.name
            };
            if (isNumeric(column)) {
                tableColumn.sClass = 'right-align';
                tableColumn.fnRender = function(obj) {
                    var result = obj.aData[obj.iDataColumn];
                    return result.toFixed(column.type == 'integer' ? 0 : 3);
                }
            } else if (isDate(column)) {
                tableColumn.fnRender = function(obj) {
                    return formatDate(new Date(obj.aData[obj.iDataColumn]));
                }
            }
            return tableColumn;
        });
    }

    table = $('<table class="display"></table>').appendTo(target);
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
Table.chartName = 'Table'
Table.accepts = function(queryResult) {
    return true;
}

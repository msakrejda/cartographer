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

    $scope.selectedResult = null;

    $scope.$on('resultSelected', function(event, queryResult) {
	console.log("update selectedResult to: " + queryResult.id);
        $scope.selectedResult = queryResult;
    });

}]);

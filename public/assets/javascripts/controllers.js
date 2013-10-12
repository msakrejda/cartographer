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
        if (result != $scope.selectedResult) {
            $scope.selectedResult = result
            $scope.$broadcast('resultSelected', result);
        }
    };

    qs.onQuery(function(query) {
	$scope.results.push(query);
    });

    qs.onClose(function(reason) {
	$scope.connected = false;
    });
}]);

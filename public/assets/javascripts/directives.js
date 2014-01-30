angular.module('app').directive('grid', function () {
    console.log("loading grid directive");

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

    return {
	restrict: 'E',
	template: '<table class="display" width="600" height="400"></table>',
	scope: {
	    resultSet: '='
	},
	link: function (scope, elem, attrs) {
	    console.log("linking grid");
	    var tableElem = elem[0].firstChild
	    var rebind = function() {
		console.log("scope.resultSet is: " + scope.resultSet);
		if (!scope.resultSet) {
		    return
		}
		try {
		    var data = scope.resultSet.data
		    var cols = massageColumns(scope.resultSet)
		    $(tableElem).dataTable({
			"aaData": data,
			"aoColumns": cols,
			"bDestroy": true
		    });
		} catch (e) {
		    console.log(e.name + ": " + e.message);
		}
	    };

	    scope.$watch('resultSet', function(oldVal, newVal) {
		if(newVal) {
		    rebind();
		}
	    });

	    console.log("linked");
	}
    }
});

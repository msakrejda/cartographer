var myAppModule = angular.module('app', []);

function isNumeric(column) {
    return column.type == 'int64' || column.type == 'float';
}

function isDate(column) {
    return column.type == 'time.Time';
}

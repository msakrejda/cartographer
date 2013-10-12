angular.module('app').factory('QueryService', ['$q', '$rootScope', function($q, $rootScope) {
    var conn;

    if (window["WebSocket"]) {
        var loc = window.location, wsUrl;
        if (loc.protocol === "https:") {
	    wsUrl = "wss:";
        } else {
	    wsUrl = "ws:";
        }
        wsUrl += "//" + loc.host + loc.pathname + "connect";
        conn = new WebSocket(wsUrl);
	conn.onopen = function(){
	    console.log("query service connected");
	}
	conn.onerror = function (error) {
	    console.log("query service error: " + error);
	};
    } else {
        // TODO: display "no websockets, yo" error message
    }

    return {
	onQuery: function(callback) {
	    conn.onmessage = function(message) {
		console.log("query service message received");
		$rootScope.$apply(function () {
		    callback(JSON.parse(message.data));
		});
	    }
	},

	onClose: function(callback) {
	    conn.onclose = function(evt) {
		console.log("query service closed: " + evt.reason);
		$rootScope.$apply(function () {
		    callback(evt.reason)
		});
	    }
	}
    }}])

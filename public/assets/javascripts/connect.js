$(function() {
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
        conn.onclose = function(evt) {
            appendLog($("<div><b>Connection closed.</b></div>"))
        }
        conn.onmessage = function(evt) {
            appendLog($("<div/>").text(evt.data))
        }
    } else {
        appendLog($("<div><b>Your browser does not support WebSockets.</b></div>"))
    }
});

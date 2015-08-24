var app = angular.module('netgraph', [
    'angular-websocket'
])
app.factory('netdata', function($websocket) {
    var dataStream = $websocket("ws://" + location.host + "/data");
    var streams = {};
    var reqs = [];
    dataStream.onMessage(function(message) {
        var e = JSON.parse(message.data)
        if (!(e.StreamSeq in streams)) {
            streams[e.StreamSeq] = []
        }
        var stream = streams[e.StreamSeq]
        e.Timestamp = e.Timestamp.toFixed(3)
        if (e.Type == "HttpRequest") {
            stream.push(e);
            reqs.push(e);
        } else if (e.Type == "HttpResponse") {
            if (stream.length > 0) {
                stream[stream.length-1].Response = e
                console.log(e.Code)
            }
        }
    });
    var data = {
        reqs: reqs,
        streams: streams,
        sync: function() {
            dataStream.send("sync");
        }
    };
    return data;
})
app.controller('HttpListCtrl', function ($scope, netdata) {
    console.log("called")
    $scope.reqs = netdata.reqs;
    $scope.showDetail = function(req) {
        var reqHeader = $("#request-head table")
        var respHeader = $("#response-head table")
        reqHeader.html("")
        respHeader.html("")
        $("#request-first-line").html("")
        $("#response-first-line").html("")
        
        
        $("#request-first-line").html(req.Method + " " + req.Uri + " " + req.Version)
        for (var i = 0; i < req.Headers.length; ++i) {
            var hi = req.Headers[i]
                reqHeader.append("<tr><td>"+hi.Name+"</td><td>"+hi.Value+"</td></tr>")
        }
        if (req.Response) {
            var resp = req.Response
            $("#response-first-line").html(resp.Version + " " + resp.Code + " " + resp.Reason)
            for (var i = 0; i < resp.Headers.length; ++i) {
                var hi = resp.Headers[i]
                    respHeader.append("<tr><td>"+hi.Name+"</td><td>"+hi.Value+"</td></tr>")
            }
        }
    }
    netdata.sync()
})

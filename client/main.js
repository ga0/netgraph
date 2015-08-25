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
    $scope.reqs = netdata.reqs;
    $scope.showDetail = function($event, req) {
        $scope.selectedReq = req
        var tr = $event.currentTarget
        if ($scope.selectedRow) {
            $($scope.selectedRow).attr("style", "")
        }
        $scope.selectedRow = tr
        $(tr).attr("style", "background-color: lightgreen")
    }
    $scope.getHost = function(req) {
        for (var i = 0; i < req.Headers.length; ++i) {
            var h = req.Headers[i]
            if (h.Name == "Host") {
                return h.Value
            }
        }
        return null
    }
    $scope.selectedRow = null
    netdata.sync()
})

angular.module('ngFilter', []).filter('reqFilter', function() {
    return function(items, filterType, pattern) {
        var result = [];
        if (!filterType || !pattern)
            return items;

        function getMatchFunction() {
            if (filterType == "URI") {
                return function(item) {
                    return item.URI.indexOf(pattern) != -1;
                };
            } else if (filterType == "RequestHeader") {
                return function(item) {
                    for (var i = 0; i < item.Headers.length; ++i) {
                        var h = item.Headers[i]
                        if (h.Name.indexOf(pattern) != -1)
                            return true;
                        if (h.Value.indexOf(pattern) != -1)
                            return true;
                    }
                };
            } else if (filterType == "ResponseHeader") {
                return function(item) {
                    if (!item.Response)
                        return false;
                    for (var i = 0; i < item.Response.Headers.length; ++i) {
                        var h = item.Response.Headers[i]
                        if (h.Name.indexOf(pattern) != -1)
                            return true;
                        if (h.Value.indexOf(pattern) != -1)
                            return true;
                    }
                };
            } else if (filterType == "Cookie") {
                return function(item) {
                    for (var i = 0; i < item.Headers.length; ++i) {
                        var h = item.Headers[i];
                        if (h.Name == "Cookie") {
                            return h.Value.indexOf(pattern) != -1;
                        }
                    }
                };
            } else if (filterType == "Code") {
                return function(item) {
                    if (!item.Response)
                        return false;
                    return item.Response.Code == parseInt(pattern)
                };
            } else if (filterType == "RequestBody") {
                return function(item) {
                    return item.Body.indexOf(pattern) != -1;
                };
            } else if (filterType == "ResponseBody") {
                return function(item) {
                    if (!item.Response)
                        return false;
                    return item.Response.Body.indexOf(pattern) != -1;
                };
            }
        };
        var matchFunc = getMatchFunction();
        for (var i = 0; i < items.length; i++) {
            var item = items[i];
            if (matchFunc(item)) {
                result.push(item);
            }
        }
        return result;
    };
});
var app = angular.module('netgraph', ['angular-websocket', 'ngFilter'])
app.factory('netdata', function($websocket) {
    var dataStream = $websocket("ws://" + location.host + "/data");
    var streams = {};
    var reqs = [];
    dataStream.onMessage(function(message) {
        var e = JSON.parse(message.data);
        if (!(e.StreamSeq in streams)) {
            streams[e.StreamSeq] = [];
        }
        var stream = streams[e.StreamSeq];
        if (e.Type == "HTTPRequest") {
            e.Start = new Date(e.Start)
            if (e.Body) {
                e.Body = Base64.decode(e.Body)
            }
            stream.push(e);
            reqs.push(e);
            //add Host
            for (var i = 0; i < e.Headers.length; ++i) {
                var h = e.Headers[i];
                if (h.Name == 'Host') {
                    e.Host = h.Value;
                    break;
                }
            }
        } else if (e.Type == "HTTPResponse") {
            if (e.Body) {
                e.Body = Base64.decode(e.Body)
            }
            
            if (stream.length > 0) {
                var req = stream[stream.length-1]
                if (req.Response) {
                    console.error("duplicate response in stream #" + e.StreamSeq + " URI:" + req.URI
                        + "\nold:", req.Response, "\nnew:", e)
                } else {
                    req.Response = e;
                    req.Duration = new Date(e.End) - req.Start;
                }
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
        $scope.selectedReq = req;
        var tr = $event.currentTarget;
        if ($scope.selectedRow) {
            $($scope.selectedRow).attr("style", "");
        }
        $scope.selectedRow = tr;
        $(tr).attr("style", "background-color: lightgreen");
    }
    $scope.getHost = function(req) {
        for (var i = 0; i < req.Headers.length; ++i) {
            var h = req.Headers[i];
            if (h.Name == "Host") {
                return h.Value;
            }
        }
        return null;
    }
    $scope.selectedRow = null;
    $scope.filterType = "URI";
    $scope.order = "Start";
    netdata.sync();
})

var Stream = function(surface, x, y) {
    this.events = []
    this.surface = surface
    this.lineY = y
    this.lineX = x
}

Stream.prototype.displayEventDetail = function(ev) {
    var reqEv, respEv
    var reqHeader = $("#request-head table")
    var respHeader = $("#response-head table")
    if (ev.Type == "HttpRequest") {
        reqEv = ev
        respEv = this.events[ev.relativeHttpEventId]
    } else if (ev.Type == "HttpResponse") {
        reqEv = this.events[ev.relativeHttpEventId]
        respEv = ev
    }
    reqHeader.html("")
    respHeader.html("")
    $("#request-first-line").html("")
    $("#response-first-line").html("")
    if (reqEv) {
        $("#request-first-line").html(reqEv.Method + " " + reqEv.Uri + " " + reqEv.Version)
        for (var i = 0; i < reqEv.Headers.length; ++i) {
        var hi = reqEv.Headers[i]
            reqHeader.append("<tr><td>"+hi.Name+"</td><td>"+hi.Value+"</td></tr>")
        }
    }
    if (respEv) {
        $("#response-first-line").html(respEv.Version + " " + respEv.Code + " " + respEv.Reason)
        for (var i = 0; i < respEv.Headers.length; ++i) {
            var hi = respEv.Headers[i]
            respHeader.append("<tr><td>"+hi.Name+"</td><td>"+hi.Value+"</td></tr>")
        }
    }  
}

Stream.prototype.addEvent = function(ev) {
    var g
    var x = this.lineX + parseFloat(ev.Timestamp) * 500
    var height = 8
    var width = this.lineX + (parseFloat(ev.EndTimestamp) - parseFloat(ev.Timestamp)) * 500
    if (width < 8)
        width = 8;
    if (this.surface.attr('width') < x + width + 10) {
        this.surface.attr({width: x + width + 10})
    }
    if (this.surface.attr('height') < this.lineY + height + 10) {
        this.surface.attr({height: this.lineY + height + 10})
    }
    g = this.surface.rect(
            x,
            this.lineY - 4, 
            width,
            height,
            4,
            4)
    ev.symbol = g
    this.events.push(ev)
    if (ev.Type == "HttpRequest") {
        ev.relativeHttpEventId = this.events.length
        g.attr({
            fill: "#00ff00"
        })
    } else if (ev.Type = "HttpResponse") {
        ev.relativeHttpEventId = this.events.length - 2
        g.attr({
            fill: "#ff0000"
        })
        var reqEv = this.events[ev.relativeHttpEventId]
        if (reqEv) {
            var reqG = reqEv.symbol
            var l = this.surface.line(
                parseFloat(reqG.attr("x")) + parseFloat(reqG.attr("width")),
                this.lineY,
                x,
                this.lineY).attr({
                    strokeWidth: 1,
                    stroke: "#000000"
                })
            reqEv.line = l
            ev.line = l
        }
    }
    var stream = this
    g.attr({
        strokeWidth: 1,
        stroke: "#000000",
        "fill-opacity": 0.5
    }).mouseover(function() {
        g.attr({
            strokeWidth: 3,
            stroke: "#0000ff"
        })
        var relativeEv = stream.events[ev.relativeHttpEventId]
        if (relativeEv) {
            relativeEv.symbol.attr({
                strokeWidth: 3,
                stroke: "#0000ff"
            })
        }
        if (ev.line) {
            ev.line.attr({
                strokeWidth: 3,
                stroke: "#0000ff"
            })
        }
        stream.displayEventDetail(ev)
    }).mouseout(function() {
        g.attr({
            strokeWidth: 1,
            stroke: "#000000"
        })
        var relativeEv = stream.events[ev.relativeHttpEventId]
        if (relativeEv) {
            relativeEv.symbol.attr({
                strokeWidth: 1,
                stroke: "#000000"
            })
        }
        if (ev.line) {
            ev.line.attr({
                strokeWidth: 1,
                stroke: "#000000"
            })
        }
    })
}

var StreamPool = function(surface) {
    this.streams = {}
    this.surface = surface
    this.streamCount = 0
}

StreamPool.prototype.addStream = function(seq) {
    var s = new Stream(this.surface, 0, 20 * this.streamCount + 10)
    pool.streams[seq] = s
    this.streamCount++
    return s
}

StreamPool.prototype.getStream = function(seq) {
    var s = pool.streams[seq]
    if (s == undefined) {
        s = this.addStream(seq)
    }
    return s
}

var pool
var eventCount = 0

function start() {
    var surface = Snap("#ng")
    pool = new StreamPool(surface)
    var ws = new WebSocket("ws://" + location.host + "/data");
    ws.onmessage = function(e) {
        recvEvent(e.data)
    };
    ws.onopen = function(evt) { 
        console.log("connected to " + location.host)
        ws.send("sync")
    };
}

function recvEvent(json) {
    var e = JSON.parse(json)
    if (e.StreamSeq == undefined) {
        return
    }
    //console.log(e)
    var stream = pool.getStream(e.StreamSeq)
    stream.addEvent(e)
    eventCount++
    document.title = "[" + eventCount + "] Captured"
}
// Minimal host polyfills for React 19 SSR inside slowjs (not Node, not a browser).
// Evaluated before the IIFE SSR bundle.
(function (g) {
  if (typeof g.console === "undefined") {
    var noop = function () {};
    g.console = { log: noop, info: noop, warn: noop, error: noop, debug: noop };
  }

  if (typeof g.setTimeout === "undefined") {
    g.setTimeout = function (fn) {
      Promise.resolve().then(function () {
        if (typeof fn === "function") fn();
      });
      return 0;
    };
    g.clearTimeout = function () {};
  }

  if (typeof g.TextEncoder === "undefined") {
    g.TextEncoder = function TextEncoder() {};
    g.TextEncoder.prototype.encode = function (str) {
      str = String(str == null ? "" : str);
      var out = new Uint8Array(unescape(encodeURIComponent(str)).split("").map(function (c) {
        return c.charCodeAt(0);
      }));
      return out;
    };
    g.TextEncoder.prototype.encodeInto = function () {
      throw new Error("TextEncoder.encodeInto is not polyfilled");
    };
  }

  if (typeof g.TextDecoder === "undefined") {
    g.TextDecoder = function TextDecoder() {};
    g.TextDecoder.prototype.decode = function (input) {
      if (input == null) return "";
      var bytes = input instanceof Uint8Array ? input : new Uint8Array(input);
      var s = "";
      for (var i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i]);
      return decodeURIComponent(escape(s));
    };
  }

  if (typeof g.FormData === "undefined") {
    g.FormData = function FormData() {
      this._entries = [];
    };
    g.FormData.prototype.append = function (name, value, filename) {
      this._entries.push([String(name), value, filename]);
    };
    g.FormData.prototype.delete = function (name) {
      name = String(name);
      this._entries = this._entries.filter(function (e) { return e[0] !== name; });
    };
    g.FormData.prototype.get = function (name) {
      name = String(name);
      for (var i = 0; i < this._entries.length; i++) {
        if (this._entries[i][0] === name) return this._entries[i][1];
      }
      return null;
    };
    g.FormData.prototype.getAll = function (name) {
      name = String(name);
      var out = [];
      for (var i = 0; i < this._entries.length; i++) {
        if (this._entries[i][0] === name) out.push(this._entries[i][1]);
      }
      return out;
    };
    g.FormData.prototype.has = function (name) {
      return this.get(name) !== null;
    };
    g.FormData.prototype.set = function (name, value, filename) {
      this.delete(name);
      this.append(name, value, filename);
    };
    g.FormData.prototype.forEach = function (cb, thisArg) {
      for (var i = 0; i < this._entries.length; i++) {
        cb.call(thisArg, this._entries[i][1], this._entries[i][0], this);
      }
    };
    g.FormData.prototype.entries = function () {
      var i = 0, entries = this._entries;
      return {
        next: function () {
          if (i >= entries.length) return { done: true, value: undefined };
          var e = entries[i++];
          return { done: false, value: [e[0], e[1]] };
        },
      };
    };
    g.FormData.prototype.keys = function () {
      var i = 0, entries = this._entries;
      return {
        next: function () {
          if (i >= entries.length) return { done: true, value: undefined };
          return { done: false, value: entries[i++][0] };
        },
      };
    };
    g.FormData.prototype.values = function () {
      var i = 0, entries = this._entries;
      return {
        next: function () {
          if (i >= entries.length) return { done: true, value: undefined };
          return { done: false, value: entries[i++][1] };
        },
      };
    };
    g.FormData.prototype[Symbol.iterator] = g.FormData.prototype.entries;
  }

  if (typeof g.MessageChannel === "undefined") {
    function Port() {
      this.onmessage = null;
      this._peer = null;
    }
    Port.prototype.postMessage = function (data) {
      var peer = this._peer;
      Promise.resolve().then(function () {
        if (peer && typeof peer.onmessage === "function") {
          peer.onmessage({ data: data });
        }
      });
    };
    Port.prototype.start = function () {};
    Port.prototype.close = function () {};
    Port.prototype.addEventListener = function () {};
    Port.prototype.removeEventListener = function () {};

    function MessageChannel() {
      this.port1 = new Port();
      this.port2 = new Port();
      this.port1._peer = this.port2;
      this.port2._peer = this.port1;
    }
    g.MessageChannel = MessageChannel;
  }
})(globalThis);

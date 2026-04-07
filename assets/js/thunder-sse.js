/*
 * Thunder SSE Client
 *
 * Connects to /__thunder/events and listens for server-sent events.
 * When an event arrives, finds all elements with a matching
 * data-t-sse attribute and re-fetches their content from
 * data-t-sse-url, then swaps the HTML using Idiomorph for
 * smooth DOM updates.
 *
 * SSE named events require explicit addEventListener calls.
 * On page load, we scan all [data-t-sse] elements, collect
 * unique event names, and register a listener for each.
 * A MutationObserver watches for dynamically added elements.
 */
(function () {
  "use strict";

  var source = null;
  var reconnectDelay = 1000;
  var maxReconnectDelay = 30000;
  var registeredEvents = {};

  function connect() {
    if (source) {
      source.close();
    }

    source = new EventSource("/__thunder/events");

    source.addEventListener("connected", function () {
      reconnectDelay = 1000;
    });

    source.onerror = function () {
      source.close();
      source = null;
      setTimeout(function () {
        reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
        connect();
      }, reconnectDelay);
    };

    // Re-register all known event listeners on the new source
    registeredEvents = {};
    scanAndRegister();
  }

  function registerEvent(eventName) {
    if (registeredEvents[eventName]) return;
    registeredEvents[eventName] = true;

    source.addEventListener(eventName, function () {
      var targets = document.querySelectorAll(
        '[data-t-sse="' + eventName + '"]'
      );
      for (var i = 0; i < targets.length; i++) {
        refreshElement(targets[i]);
      }
    });
  }

  function scanAndRegister() {
    if (!source) return;
    var els = document.querySelectorAll("[data-t-sse]");
    for (var i = 0; i < els.length; i++) {
      var name = els[i].getAttribute("data-t-sse");
      if (name) registerEvent(name);
    }
  }

  function refreshElement(el) {
    var url = el.getAttribute("data-t-sse-url");
    if (!url) return;

    fetch(url, {
      headers: { "HX-Request": "true" },
      credentials: "same-origin",
    })
      .then(function (res) {
        if (!res.ok) throw new Error("HTTP " + res.status);
        return res.text();
      })
      .then(function (html) {
        // Use Idiomorph if available for smooth morphing
        if (typeof Idiomorph !== "undefined" && Idiomorph.morph) {
          Idiomorph.morph(el, html, { morphStyle: "innerHTML" });
        } else {
          el.innerHTML = html;
        }
      })
      .catch(function (err) {
        console.error("[thunder-sse] refresh failed:", url, err);
      });
  }

  // Watch for dynamically added SSE elements
  var observer = new MutationObserver(function () {
    scanAndRegister();
  });

  function init() {
    connect();
    observer.observe(document.body, { childList: true, subtree: true });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  window.__thunderSSE = {
    refresh: refreshElement,
    reconnect: connect,
    scan: scanAndRegister,
  };
})();

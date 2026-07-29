// board-filter.js — the board's child-project filter (T-061).
//
// The filter bar lives outside #board, so it and its active button survive the
// 5s htmx poll that swaps #board. Only the .child blocks are re-rendered, so the
// selection is re-applied to them on htmx:afterSwap. Child names are dynamic
// (any registered project), so hiding is done here in JS rather than with a
// static per-name CSS rule.
(function () {
  "use strict";

  var page = document.querySelector(".page[data-filter]");
  if (!page) {
    return; // not the board page
  }

  // apply hides every child block that does not match the current filter.
  // "all" shows everything.
  function apply() {
    var filter = page.getAttribute("data-filter") || "all";
    var blocks = page.querySelectorAll(".child[data-child]");
    for (var i = 0; i < blocks.length; i++) {
      var child = blocks[i].getAttribute("data-child");
      blocks[i].hidden = filter !== "all" && child !== filter;
    }
  }

  // One delegated click handler: the buttons never get re-rendered (they live
  // outside #board), but delegation keeps this robust and cheap.
  page.addEventListener("click", function (e) {
    var btn = e.target.closest(".filter-btn");
    if (!btn) {
      return;
    }
    page.setAttribute("data-filter", btn.getAttribute("data-child"));
    var buttons = page.querySelectorAll(".filter-btn");
    for (var i = 0; i < buttons.length; i++) {
      buttons[i].classList.toggle("is-active", buttons[i] === btn);
    }
    apply();
  });

  // htmx replaces #board every 5s; re-apply the filter to the fresh rows.
  document.body.addEventListener("htmx:afterSwap", apply);

  apply();
})();

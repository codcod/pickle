// board-filter.js — the board's child-project filter (T-061) and its search box
// (T-104), composed in one script.
//
// Both controls live outside #board, so they and their state survive the 5s htmx
// poll that swaps #board. Only the .child blocks (section groups and lane rows
// alike) and [data-search] ticket rows are re-rendered, so both selections are
// re-applied to them on htmx:afterSwap. Child names are dynamic (any registered
// project), so hiding is done here in JS rather than with a static per-name CSS
// rule; the search query is likewise arbitrary text, so it is a substring test,
// not a CSS selector.
//
// A ticket is visible only if it passes BOTH filters (T-104 decision 4): the
// child filter hides whole .child[data-child] blocks (which, for the active-state
// lanes, is one child's entire row of columns); the search filter hides
// individual [data-search] ticket rows within whatever blocks remain visible.
(function () {
  "use strict";

  var page = document.querySelector(".page[data-filter]");
  if (!page) {
    return; // not the board page
  }

  var search = document.getElementById("board-search");

  // apply hides every child block that does not match the current child
  // filter, and every ticket row that does not match the current search query.
  function apply() {
    var filter = page.getAttribute("data-filter") || "all";
    var blocks = page.querySelectorAll(".child[data-child]");
    for (var i = 0; i < blocks.length; i++) {
      var child = blocks[i].getAttribute("data-child");
      blocks[i].hidden = filter !== "all" && child !== filter;
    }

    var query = (page.getAttribute("data-query") || "").trim().toLowerCase();
    var items = page.querySelectorAll("[data-search]");
    for (var j = 0; j < items.length; j++) {
      var hay = items[j].getAttribute("data-search") || "";
      items[j].classList.toggle("is-hidden", query !== "" && hay.indexOf(query) === -1);
    }
  }

  // One delegated click handler: the filter buttons never get re-rendered (they
  // live outside #board), but delegation keeps this robust and cheap.
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

  // The search input lives outside #board too (same reason as the filter bar):
  // its value is mirrored onto the page element so apply() has one place to
  // read either signal from, and so it needs no re-wiring after a swap.
  if (search) {
    search.addEventListener("input", function () {
      page.setAttribute("data-query", search.value);
      apply();
    });
  }

  // htmx replaces #board every 5s; re-apply both filters to the fresh rows.
  document.body.addEventListener("htmx:afterSwap", apply);

  apply();
})();

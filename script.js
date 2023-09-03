/* global openURL, quit, onReady */

const contentEl = document.getElementById("content");

// eslint-disable-next-line no-unused-vars
function setContent(s) {
  contentEl.innerHTML = s;
}

document.documentElement.addEventListener(
  "click",
  (event) => {
    var parent = event.target;
    while (parent !== null && parent.tagName !== "A") {
      parent = parent.parentNode;
    }
    if (parent != null) {
      event.preventDefault();
      openURL(parent.getAttribute("href"));
    }
  },
  false,
);

document.addEventListener(
  "keydown",
  (ev) => {
    if (ev.key === "q") {
      ev.preventDefault();
      quit();
      return;
    }
  },
  false,
);

onReady();

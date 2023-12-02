/* global openURL, quit, onReady */

const contentEl = document.getElementById("content");

function isElementInView(el) {
  var rect = el.getBoundingClientRect();
  return (
    rect.top >= 0 &&
    rect.left >= 0 &&
    rect.bottom <=
      (window.innerHeight || document.documentElement.clientHeight) &&
    rect.right <= (window.innerWidth || document.documentElement.clientWidth)
  );
}

// eslint-disable-next-line no-unused-vars
function setContent(s) {
  contentEl.innerHTML = s;
  const changed = document.querySelector(".changed");
  if (changed != null) {
    if (!isElementInView(changed)) {
      changed.scrollIntoView();
    }
  }
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

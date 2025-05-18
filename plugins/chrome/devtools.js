chrome.devtools.panels.create(
  "Agent Logger", // Title of the panel
  "images/icon16.png",     // Icon for the panel (optional)
  "panel.html",         // HTML page for the panel's content
  function(panel) {
    // Code that runs when the panel is created (optional)
    console.log("Agent Logger panel created");
  }
);
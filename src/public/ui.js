(() => {
  const composer = document.querySelector("[data-composer]");
  const composerTitle = document.querySelector("[data-composer-title]");
  const composerClose = document.querySelector("[data-composer-close]");
  const noteCards = Array.from(document.querySelectorAll("[data-note-card]"));

  function setComposerOpen(isOpen) {
    if (!composer) {
      return;
    }
    composer.dataset.composerOpen = String(isOpen);
    if (isOpen && composerTitle) {
      composerTitle.focus();
    }
  }

  function closeAllNotes(exceptCard) {
    noteCards.forEach((card) => {
      if (card !== exceptCard) {
        card.dataset.expanded = "false";
      }
    });
  }

  if (composer) {
    composer.addEventListener("click", () => {
      if (composer.dataset.composerOpen !== "true") {
        setComposerOpen(true);
      }
    });

    if (composerClose) {
      composerClose.addEventListener("click", (event) => {
        event.preventDefault();
        setComposerOpen(false);
      });
    }
  }

  noteCards.forEach((card) => {
    const preview = card.querySelector("[data-note-preview]");
    const closeButton = card.querySelector("[data-note-close]");

    if (preview) {
      preview.addEventListener("click", () => {
        closeAllNotes(card);
        card.dataset.expanded = "true";
        const firstInput = card.querySelector("input");
        if (firstInput) {
          firstInput.focus();
        }
      });
    }

    if (closeButton) {
      closeButton.addEventListener("click", (event) => {
        event.preventDefault();
        card.dataset.expanded = "false";
      });
    }
  });

  document.addEventListener("click", (event) => {
    if (composer && composer.dataset.composerOpen === "true" && !composer.contains(event.target)) {
      setComposerOpen(false);
    }

    noteCards.forEach((card) => {
      if (card.dataset.expanded === "true" && !card.contains(event.target)) {
        card.dataset.expanded = "false";
      }
    });
  });

  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") {
      return;
    }

    if (composer && composer.dataset.composerOpen === "true") {
      setComposerOpen(false);
    }

    closeAllNotes();
  });
})();

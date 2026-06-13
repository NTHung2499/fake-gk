(() => {
  const composer = document.querySelector("[data-composer]");
  const composerTitle = document.querySelector("[data-composer-title]");
  const composerClose = document.querySelector("[data-composer-close]");
  const searchInput = document.querySelector("[data-note-search]");
  const noteCards = Array.from(document.querySelectorAll("[data-note-card]"));
  const noteSections = Array.from(document.querySelectorAll("[data-note-section]"));

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

  function refreshSectionVisibility(query) {
    noteSections.forEach((section) => {
      const cards = Array.from(section.querySelectorAll("[data-note-card]"));
      const visibleCards = cards.filter((card) => !card.hidden);
      const empty = section.querySelector("[data-section-empty]");
      const count = section.querySelector(".section-header > span");

      if (count) {
        count.textContent = String(query ? visibleCards.length : cards.length);
      }

      if (empty) {
        empty.hidden = visibleCards.length > 0;
        empty.textContent = query ? "No matching notes here." : empty.dataset.defaultText;
      }
    });
  }

  function filterNotes() {
    const query = (searchInput ? searchInput.value : "").trim().toLowerCase();

    noteCards.forEach((card) => {
      const searchableText = card.textContent.toLowerCase();
      const matches = !query || searchableText.includes(query);
      card.hidden = !matches;
      if (!matches) {
        card.dataset.expanded = "false";
      }
    });

    refreshSectionVisibility(query);
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

  noteSections.forEach((section) => {
    const empty = section.querySelector("[data-section-empty]");
    if (empty) {
      empty.dataset.defaultText = empty.textContent;
    }
  });

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

  if (searchInput) {
    searchInput.addEventListener("input", filterNotes);
  }

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

    if (searchInput && searchInput.value) {
      searchInput.value = "";
      filterNotes();
    }

    closeAllNotes();
  });
})();

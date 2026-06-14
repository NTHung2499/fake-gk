(() => {
  const composer = document.querySelector("[data-composer]");
  const composerTitle = document.querySelector("[data-composer-title]");
  const composerClose = document.querySelector("[data-composer-close]");
  const searchInput = document.querySelector("[data-note-search]");
  const searchClear = document.querySelector("[data-search-clear]");
  const searchStatus = document.querySelector("[data-search-status]");
  const globalEmpty = document.querySelector("[data-global-empty]");
  const refreshButton = document.querySelector("[data-refresh-button]");
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

  function openNote(card) {
    closeAllNotes(card);
    card.dataset.expanded = "true";
    const firstInput = card.querySelector("input");
    if (firstInput) {
      firstInput.focus();
    }
  }

  function isInteractiveElement(element) {
    return Boolean(element.closest("button, input, textarea, select, a, form"));
  }

  function refreshSectionVisibility(query) {
    let totalVisible = 0;

    noteSections.forEach((section) => {
      const cards = Array.from(section.querySelectorAll("[data-note-card]"));
      const visibleCards = cards.filter((card) => !card.hidden);
      const empty = section.querySelector("[data-section-empty]");
      const count = section.querySelector(".section-header > span");
      totalVisible += visibleCards.length;

      if (count) {
        count.textContent = String(query ? visibleCards.length : cards.length);
      }

      if (empty) {
        const emptyTitle = empty.querySelector("strong");
        const emptyBody = empty.querySelector("p");
        empty.hidden = visibleCards.length > 0;
        if (query) {
          if (emptyTitle) {
            emptyTitle.textContent = "No matching notes here.";
          }
          if (emptyBody) {
            emptyBody.textContent = "This section has no result for your search.";
          }
        } else {
          if (emptyTitle) {
            emptyTitle.textContent = empty.dataset.defaultTitle || "";
          }
          if (emptyBody) {
            emptyBody.textContent = empty.dataset.defaultBody || "";
          }
        }
      }
    });

    if (searchClear) {
      searchClear.hidden = !query;
    }

    if (searchStatus) {
      searchStatus.hidden = !query;
      searchStatus.textContent = query ? `Showing ${totalVisible} ${totalVisible === 1 ? "note" : "notes"} for "${query}"` : "";
    }

    if (globalEmpty) {
      globalEmpty.hidden = !query || totalVisible > 0;
    }
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
      const title = empty.querySelector("strong");
      const body = empty.querySelector("p");
      empty.dataset.defaultTitle = title ? title.textContent : "";
      empty.dataset.defaultBody = body ? body.textContent : "";
    }
  });

  noteCards.forEach((card) => {
    const preview = card.querySelector("[data-note-preview]");
    const closeButton = card.querySelector("[data-note-close]");

    if (preview) {
      preview.addEventListener("click", () => {
        openNote(card);
      });
    }

    card.addEventListener("keydown", (event) => {
      if ((event.key !== "Enter" && event.key !== " ") || isInteractiveElement(event.target)) {
        return;
      }

      event.preventDefault();
      openNote(card);
    });

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

  if (searchClear && searchInput) {
    searchClear.addEventListener("click", () => {
      searchInput.value = "";
      filterNotes();
      searchInput.focus();
    });
  }

  if (refreshButton) {
    refreshButton.addEventListener("click", () => {
      window.location.reload();
    });
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

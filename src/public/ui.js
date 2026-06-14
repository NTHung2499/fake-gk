(() => {
  const app = document.querySelector("[data-chat-app]");
  if (!app) {
    return;
  }

  const sessionList = document.querySelector("[data-session-list]");
  const newSessionButton = document.querySelector("[data-new-session]");
  const messageList = document.querySelector("[data-message-list]");
  const chatForm = document.querySelector("[data-chat-form]");
  const messageInput = chatForm ? chatForm.querySelector("textarea") : null;
  const chatTitle = document.querySelector("[data-chat-title]");
  const keyForm = document.querySelector("[data-key-form]");
  const keyStatus = document.querySelector("[data-key-status]");
  const renameButton = document.querySelector("[data-rename-session]");
  const deleteButton = document.querySelector("[data-delete-session]");

  let activeSessionId = Number(app.dataset.activeSessionId || 0);
  let isStreaming = false;

  function requestJSON(url, options = {}) {
    return fetch(url, {
      headers: {"Content-Type": "application/json", ...(options.headers || {})},
      ...options,
    }).then(async (response) => {
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(payload.error || "Request failed");
      }
      return payload;
    });
  }

  function scrollToBottom() {
    if (messageList) {
      messageList.scrollTop = messageList.scrollHeight;
    }
  }

  function clearWelcome() {
    const welcome = messageList ? messageList.querySelector("[data-welcome-panel]") : null;
    if (welcome) {
      welcome.remove();
    }
  }

  function ensureWelcome() {
    if (!messageList || messageList.children.length > 0) {
      return;
    }
    const panel = document.createElement("div");
    panel.className = "welcome-panel";
    panel.dataset.welcomePanel = "true";
    panel.innerHTML = `
      <span class="welcome-mark">F</span>
      <h2>Start a focused conversation.</h2>
      <p>Create a session, ask a question, and FakeGK will keep the history tied to this chat.</p>
    `;
    messageList.append(panel);
  }

  function renderMessage(message, streaming = false) {
    clearWelcome();
    const article = document.createElement("article");
    article.className = `message message-${message.role || "assistant"}${message.status === "error" ? " is-error" : ""}`;
    if (message.id) {
      article.dataset.messageId = message.id;
    }

    const avatar = document.createElement("div");
    avatar.className = "message-avatar";
    avatar.textContent = message.role === "user" ? "U" : "F";

    const body = document.createElement("div");
    body.className = "message-body";

    const meta = document.createElement("div");
    meta.className = "message-meta";
    meta.textContent = message.role === "user" ? "You" : "FakeGK";

    const text = document.createElement("p");
    text.textContent = message.status === "error" ? (message.error || "Something went wrong.") : (message.content || "");
    if (streaming) {
      text.dataset.streamTarget = "true";
      text.textContent = "";
    }

    body.append(meta, text);
    article.append(avatar, body);
    messageList.append(article);
    scrollToBottom();
    return article;
  }

  function renderSession(session, active = false) {
    const empty = sessionList.querySelector("[data-session-empty]");
    if (empty) {
      empty.remove();
    }

    const button = document.createElement("button");
    button.className = `session-item${active ? " is-active" : ""}`;
    button.type = "button";
    button.dataset.sessionId = session.id;
    button.innerHTML = `<span></span><small>Chat</small>`;
    button.querySelector("span").textContent = session.title || "New chat";
    sessionList.prepend(button);
    bindSessionButton(button);
    return button;
  }

  function setActiveSession(sessionId, title) {
    activeSessionId = Number(sessionId || 0);
    app.dataset.activeSessionId = String(activeSessionId);
    document.querySelectorAll("[data-session-id]").forEach((button) => {
      button.classList.toggle("is-active", Number(button.dataset.sessionId) === activeSessionId);
    });
    if (chatTitle) {
      chatTitle.textContent = title || "New chat";
    }
  }

  async function createSession(title = "New chat") {
    const payload = await requestJSON("/api/sessions", {
      method: "POST",
      body: JSON.stringify({title}),
    });
    renderSession(payload.session, true);
    setActiveSession(payload.session.id, payload.session.title);
    if (messageList) {
      messageList.innerHTML = "";
      ensureWelcome();
    }
    return payload.session;
  }

  async function loadMessages(sessionId, title) {
    const payload = await requestJSON(`/api/sessions/${sessionId}/messages`);
    messageList.innerHTML = "";
    payload.messages.forEach((message) => renderMessage(message));
    ensureWelcome();
    setActiveSession(sessionId, title);
    scrollToBottom();
  }

  function bindSessionButton(button) {
    button.addEventListener("click", () => {
      const title = button.querySelector("span") ? button.querySelector("span").textContent : "New chat";
      loadMessages(Number(button.dataset.sessionId), title).catch((error) => {
        renderMessage({role: "assistant", status: "error", error: error.message});
      });
    });
  }

  function updateActiveSessionTitleFromMessage(message) {
    const active = document.querySelector(`.session-item[data-session-id="${activeSessionId}"] span`);
    if (!active || active.textContent !== "New chat") {
      return;
    }
    const chars = Array.from(message);
    const title = chars.length > 48 ? `${chars.slice(0, 48).join("")}...` : message;
    active.textContent = title;
    if (chatTitle) {
      chatTitle.textContent = title;
    }
  }

  async function sendMessage(message) {
    if (isStreaming || !message) {
      return;
    }
    if (!activeSessionId) {
      await createSession();
    }

    isStreaming = true;
    chatForm.querySelector("button").disabled = true;
    renderMessage({role: "user", content: message, status: "complete"});
    updateActiveSessionTitleFromMessage(message);
    const assistantNode = renderMessage({role: "assistant", content: "", status: "complete"}, true);
    const streamTarget = assistantNode.querySelector("[data-stream-target]");

    const source = new EventSource(`/api/sessions/${activeSessionId}/stream?message=${encodeURIComponent(message)}`);
    source.addEventListener("delta", (event) => {
      const payload = JSON.parse(event.data);
      streamTarget.textContent += payload.delta || "";
      scrollToBottom();
    });
    source.addEventListener("done", (event) => {
      const payload = JSON.parse(event.data);
      if (payload.message && payload.message.id) {
        assistantNode.dataset.messageId = payload.message.id;
      }
      source.close();
      isStreaming = false;
      chatForm.querySelector("button").disabled = false;
      messageInput.focus();
    });
    source.addEventListener("error", (event) => {
      let messageText = "Streaming failed. Please try again.";
      if (event.data) {
        try {
          const payload = JSON.parse(event.data);
          messageText = payload.error || messageText;
        } catch (_) {
          messageText = event.data;
        }
      }
      streamTarget.textContent = messageText;
      assistantNode.classList.add("is-error");
      source.close();
      isStreaming = false;
      chatForm.querySelector("button").disabled = false;
    });
  }

  document.querySelectorAll("[data-session-id]").forEach(bindSessionButton);

  if (newSessionButton) {
    newSessionButton.addEventListener("click", () => {
      createSession().catch((error) => {
        renderMessage({role: "assistant", status: "error", error: error.message});
      });
    });
  }

  if (keyForm) {
    keyForm.addEventListener("submit", (event) => {
      event.preventDefault();
      const apiKey = keyForm.elements.apiKey.value.trim();
      keyStatus.textContent = "";
      requestJSON("/api/key", {
        method: "POST",
        body: JSON.stringify({apiKey}),
      })
        .then(() => window.location.reload())
        .catch((error) => {
          keyStatus.textContent = error.message;
        });
    });
  }

  if (chatForm && messageInput) {
    messageInput.addEventListener("input", () => {
      messageInput.style.height = "auto";
      messageInput.style.height = `${Math.min(messageInput.scrollHeight, 180)}px`;
    });

    messageInput.addEventListener("keydown", (event) => {
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault();
        chatForm.requestSubmit();
      }
    });

    chatForm.addEventListener("submit", (event) => {
      event.preventDefault();
      const message = messageInput.value.trim();
      if (!message) {
        return;
      }
      messageInput.value = "";
      messageInput.style.height = "auto";
      sendMessage(message).catch((error) => {
        isStreaming = false;
        chatForm.querySelector("button").disabled = false;
        renderMessage({role: "assistant", status: "error", error: error.message});
      });
    });
  }

  if (renameButton) {
    renameButton.addEventListener("click", () => {
      if (!activeSessionId) {
        return;
      }
      const current = chatTitle ? chatTitle.textContent : "New chat";
      const title = window.prompt("Rename chat", current);
      if (!title) {
        return;
      }
      requestJSON(`/api/sessions/${activeSessionId}/rename`, {
        method: "POST",
        body: JSON.stringify({title}),
      }).then(() => {
        const active = document.querySelector(`.session-item[data-session-id="${activeSessionId}"] span`);
        if (active) {
          active.textContent = title;
        }
        if (chatTitle) {
          chatTitle.textContent = title;
        }
      }).catch((error) => {
        renderMessage({role: "assistant", status: "error", error: error.message});
      });
    });
  }

  if (deleteButton) {
    deleteButton.addEventListener("click", () => {
      if (!activeSessionId || !window.confirm("Delete this chat?")) {
        return;
      }
      requestJSON(`/api/sessions/${activeSessionId}/delete`, {method: "POST"})
        .then(() => {
          const active = document.querySelector(`.session-item[data-session-id="${activeSessionId}"]`);
          if (active) {
            active.remove();
          }
          const next = document.querySelector("[data-session-id]");
          if (next) {
            next.click();
          } else {
            activeSessionId = 0;
            if (chatTitle) {
              chatTitle.textContent = "New chat";
            }
            messageList.innerHTML = "";
            ensureWelcome();
          }
        })
        .catch((error) => {
          renderMessage({role: "assistant", status: "error", error: error.message});
        });
    });
  }

  scrollToBottom();
})();

const path = require("path");
const express = require("express");
const config = require("./config");
const db = require("./db");
const notes = require("./notes");

const app = express();

app.set("view engine", "ejs");
app.set("views", path.join(__dirname, "views"));

app.use(express.urlencoded({ extended: false }));
app.use(express.static(path.join(__dirname, "public"), {
  maxAge: "1h"
}));

function asyncRoute(handler) {
  return (req, res, next) => {
    Promise.resolve(handler(req, res, next)).catch(next);
  };
}

app.get("/healthz", (_req, res) => {
  res.status(200).json({ status: "ok" });
});

app.get("/readyz", async (_req, res) => {
  try {
    await db.ping();
    res.status(200).json({ status: "ready" });
  } catch (err) {
    res.status(503).json({ status: "not ready", error: err.code || err.message });
  }
});

app.get("/", asyncRoute(async (_req, res) => {
  const allNotes = await notes.listNotes();
  res.render("index", {
    notes: allNotes.filter((note) => !note.is_archived),
    archivedNotes: allNotes.filter((note) => note.is_archived),
    colors: Array.from(notes.allowedColors)
  });
}));

app.post("/notes", asyncRoute(async (req, res) => {
  await notes.createNote(req.body);
  res.redirect("/");
}));

app.post("/notes/:id", asyncRoute(async (req, res) => {
  await notes.updateNote(req.params.id, req.body);
  res.redirect("/");
}));

app.post("/notes/:id/pin", asyncRoute(async (req, res) => {
  await notes.togglePinned(req.params.id);
  res.redirect("/");
}));

app.post("/notes/:id/archive", asyncRoute(async (req, res) => {
  await notes.toggleArchived(req.params.id);
  res.redirect("/");
}));

app.post("/notes/:id/delete", asyncRoute(async (req, res) => {
  await notes.deleteNote(req.params.id);
  res.redirect("/");
}));

app.use((err, _req, res, _next) => {
  console.error(err);
  res.status(500).render("error", {
    message: "Fake GK hit a server error.",
    detail: process.env.NODE_ENV === "production" ? null : err.message
  });
});

async function start() {
  await db.migrate();
  app.listen(config.port, () => {
    console.log(`fake-gk listening on port ${config.port}`);
  });
}

if (require.main === module) {
  start().catch((err) => {
    console.error("Failed to start fake-gk");
    console.error(err);
    process.exit(1);
  });
}

module.exports = app;

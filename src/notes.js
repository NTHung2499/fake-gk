const { pool } = require("./db");

const allowedColors = new Set(["yellow", "green", "blue", "pink", "purple", "gray"]);

function normalizeColor(color) {
  return allowedColors.has(color) ? color : "yellow";
}

function normalizeText(value) {
  return String(value || "").trim();
}

async function listNotes() {
  const [rows] = await pool.execute(`
    SELECT id, title, body, color, is_pinned, is_archived, created_at, updated_at
    FROM notes
    ORDER BY is_archived ASC, is_pinned DESC, updated_at DESC, id DESC
  `);

  return rows.map((note) => ({
    ...note,
    is_pinned: Boolean(note.is_pinned),
    is_archived: Boolean(note.is_archived)
  }));
}

async function createNote(input) {
  const title = normalizeText(input.title);
  const body = normalizeText(input.body);
  const color = normalizeColor(input.color);

  if (!title && !body) {
    return;
  }

  await pool.execute(
    "INSERT INTO notes (title, body, color) VALUES (?, ?, ?)",
    [title, body, color]
  );
}

async function updateNote(id, input) {
  await pool.execute(
    "UPDATE notes SET title = ?, body = ?, color = ? WHERE id = ?",
    [normalizeText(input.title), normalizeText(input.body), normalizeColor(input.color), id]
  );
}

async function togglePinned(id) {
  await pool.execute("UPDATE notes SET is_pinned = NOT is_pinned WHERE id = ?", [id]);
}

async function toggleArchived(id) {
  await pool.execute(
    "UPDATE notes SET is_pinned = IF(is_archived = FALSE, FALSE, is_pinned), is_archived = NOT is_archived WHERE id = ?",
    [id]
  );
}

async function deleteNote(id) {
  await pool.execute("DELETE FROM notes WHERE id = ?", [id]);
}

module.exports = {
  allowedColors,
  createNote,
  deleteNote,
  listNotes,
  toggleArchived,
  togglePinned,
  updateNote
};

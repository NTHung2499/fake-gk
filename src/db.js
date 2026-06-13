const mysql = require("mysql2/promise");
const config = require("./config");

const pool = mysql.createPool(config.mysql);

async function migrate() {
  await pool.execute(`
    CREATE TABLE IF NOT EXISTS notes (
      id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
      title VARCHAR(255) NOT NULL DEFAULT '',
      body TEXT NOT NULL,
      color VARCHAR(32) NOT NULL DEFAULT 'yellow',
      is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
      is_archived BOOLEAN NOT NULL DEFAULT FALSE,
      created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      PRIMARY KEY (id),
      INDEX idx_notes_pinned_updated (is_pinned, updated_at),
      INDEX idx_notes_archived_updated (is_archived, updated_at)
    )
  `);
}

async function ping() {
  const connection = await pool.getConnection();
  try {
    await connection.ping();
  } finally {
    connection.release();
  }
}

module.exports = {
  migrate,
  ping,
  pool
};

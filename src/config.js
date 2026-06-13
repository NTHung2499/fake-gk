const config = {
  port: Number(process.env.PORT || 3000),
  mysql: {
    host: process.env.MYSQL_HOST || "mysql.database.svc.cluster.local",
    port: Number(process.env.MYSQL_PORT || 3306),
    database: process.env.MYSQL_DATABASE || "appdb",
    user: process.env.MYSQL_USER || "appuser",
    password: process.env.MYSQL_PASSWORD || "apppass123",
    waitForConnections: true,
    connectionLimit: Number(process.env.MYSQL_CONNECTION_LIMIT || 10),
    queueLimit: 0
  }
};

module.exports = config;

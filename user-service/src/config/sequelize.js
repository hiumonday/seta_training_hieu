import dbConfig from "./postgres.db.js";
import { Sequelize, DataTypes } from "sequelize";
import userModel from "../models/userModel.js";
import teamModel from "../models/teamModel.js";
import rosterModel from "../models/rosterModel.js";

import dotenv from "dotenv";
import { fileURLToPath } from "url";
import path from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

dotenv.config({ path: path.resolve(__dirname, "../../../.env") });

const conn =
  process.env.NODE_ENV === "development"
    ? dbConfig.development
    : dbConfig.production;

const sequelize = new Sequelize(conn.database, conn.username, conn.password, {
  host: conn.host,
  dialect: conn.dialect,
  pool: {
    max: conn.pool.max,
    min: conn.pool.min,
    acquire: conn.pool.acquire,
    idle: conn.pool.idle,
  },
  logging: (msg) => {
    const queryType = msg.split(" ")[0]; // extract the query type
    if (queryType === "SELECT" || queryType === "INSERT") {
      console.log(`Sequelize: ${msg.substring(0, 100)}...`); // log only specific types of queries
    }
  },
});

const db = {};

db.Sequelize = Sequelize; // refer to the library
db.sequelize = sequelize; // refer to an instance of Sequelize

db.User = userModel(sequelize, DataTypes);
db.Team = teamModel(sequelize, DataTypes);
db.Roster = rosterModel(sequelize, DataTypes);

// set up associations
Object.keys(db).forEach((modelName) => {
  if (db[modelName].associate) {
    db[modelName].associate(db);
  }
});

export default db;

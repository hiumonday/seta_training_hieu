import { ApolloServer } from "@apollo/server";
import { expressMiddleware } from "@apollo/server/express4";
import { ApolloServerPluginDrainHttpServer } from "@apollo/server/plugin/drainHttpServer";
import chalk from "chalk";
import express from "express";
import http from "http";
import cors from "cors";
import cookieParser from "cookie-parser";
import { fileURLToPath } from "url";
import path from "path";
import fs from "fs";
import resolvers from "./src/resolvers/resolvers.js";
import db from "./src/config/sequelize.js";
import dotenv from "dotenv";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

dotenv.config({ path: path.resolve(__dirname, "../.env") });

const port = process.env.PORT || 4000;
const host = process.env.HOST || "localhost";

if (process.env.NODE_ENV === "development") {
  db.sequelize
    .sync() // create if not exist
    //.sync({ alter: true })
    //.sync({ force: true })
    .then(() => {
      console.log("Synced postgres db.");
    })
    .catch((err) => {
      console.log(`Failed to sync db: ${err.message}`);
    });
}

const app = express();
const httpServer = http.createServer(app);

const server = new ApolloServer({
  typeDefs: fs.readFileSync(
    path.join(__dirname, "/src", "/schema", "schema.graphql"),
    "utf8"
  ),
  resolvers,
  plugins: [ApolloServerPluginDrainHttpServer({ httpServer })],
});
await server.start();

const allowedOrigins = ["http://localhost:5173"];

const corsOptions = {
  origin: allowedOrigins,
  credentials: true,
};

// to deal with Apollo Server's built-in Express middleware not being compatible with express 5
app.use((req, res, next) => {
  req.body = req.body || {};
  next();
});

app.use(
  "/users",
  cors(corsOptions),
  express.json(),
  express.urlencoded({ extended: true }),
  cookieParser(),
  expressMiddleware(server, {
    context: async ({ req, res }) => ({ req, res }),
  })
);

await new Promise((resolve) => httpServer.listen(port, host, resolve));
console.log(
  `${chalk.cyan("⚛️  User Service")} running in ${chalk.yellow(
    process.env.NODE_ENV
  )} environment and ready at http://${chalk.green(host)}:${chalk.green(
    port
  )}/${chalk.green("users")}`
);

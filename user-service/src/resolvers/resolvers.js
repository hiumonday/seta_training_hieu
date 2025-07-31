import db from "../config/sequelize.js";
import jwt from "jsonwebtoken";
import bcrypt from "bcryptjs";
import { DateTimeResolver } from "graphql-scalars";
import {
  generateAccessToken,
  generateRefreshToken,
} from "../utils/generateTokens.js";
import dotenv from "dotenv";
import { fileURLToPath } from "url";
import path from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

dotenv.config({ path: path.resolve(__dirname, "../../../.env") });

const user = db.User;
const team = db.Team;
const roster = db.Roster;

const resolvers = {
  DateTime: DateTimeResolver,
  Query: {
    users: async (_, { role }) => {
      return await user.findAll({
        where: { role },
      });
    },
    user: async (_, { userId }) => {
      return await user.findByPk(userId);
    },
    teams: async (_, { userId }) => {
      try {
        const teams = await team.findAll({
          attributes: [
            "teamId",
            "teamName",
            [
              db.sequelize.literal(
                `(SELECT COUNT(*) FROM "Rosters" WHERE "Rosters"."teamId" = "Team"."teamId")`
              ),
              "rosterCount",
            ],
          ],
          include: [
            {
              model: roster,
              as: "rosters",
              where: { userId },
              attributes: [],
            },
          ],
          // return plain JavaScript objects directly from query results
          raw: true,
        });

        return teams;
      } catch (err) {
        console.error(err);
        throw err;
      }
    },
    team: async (_, { teamId }) => {
      const query = await team.findOne({
        where: { teamId },
        include: [
          {
            model: roster,
            as: "rosters",
            include: [
              {
                model: user,
                as: "user",
                attributes: ["userId", "username", "role"],
              },
            ],
          },
          {
            model: user,
            as: "managers",
            attributes: ["userId", "username"],
            through: { attributes: [] },
            required: false,
            where: { role: "MANAGER" },
          },
          {
            model: user,
            as: "members",
            attributes: ["userId", "username"],
            through: { attributes: [] },
            required: false,
            where: { role: "MEMBER" },
          },
        ],
      });

      const teamInstance = query.get({ plain: true });
      console.log(JSON.stringify(teamInstance, null, 2));

      const managers = teamInstance.managers.map((manager) => ({
        managerId: manager.userId,
        managerName: manager.username,
      }));

      const members = teamInstance.members.map((member) => ({
        memberId: member.userId,
        memberName: member.username,
      }));

      return {
        teamId: teamInstance.teamId,
        teamName: teamInstance.teamName,
        managers,
        members,
        totalManagers: managers.length,
        totalMembers: members.length,
      };
    },
    myTeams: async (_, { userId }) => {
      const myTeams = await roster.findAll({
        where: {
          userId,
          isLeader: true,
        },
        include: [
          {
            model: team,
            as: "team",
            attributes: ["teamId", "teamName"],
          },
        ],
      });

      return myTeams.map((roster) => ({
        teamId: roster.team.teamId,
        teamName: roster.team.teamName,
      }));
    },
  },

  Mutation: {
    createUser: async (_, args) => {
      const { username, email, password, role } = args.input;
      try {
        const userRes = await user.create({
          username,
          email,
          password,
          role: role.toUpperCase(),
        });
        return {
          code: "200",
          success: true,
          message: `Welcome on board ${username}^^`,
          user: userRes,
        };
      } catch (err) {
        return {
          code:
            err.name === "SequelizeUniqueConstraintError" ||
            "SequelizeValidationError"
              ? "400"
              : "500",
          success: false,
          errors: err.errors ? err.errors.map((error) => error.message) : null,
          user: null,
        };
      }
    },

    updateUser: async (_, { userId, username, email }) => {
      try {
        await user.update(
          {
            username,
            email,
          },
          {
            where: {
              userId,
            },
          }
        );
        return {
          code: "200",
          success: true,
          message: `Updated ${username}'s profile`,
          user: null,
        };
      } catch (err) {
        return {
          code:
            err.name === "SequelizeUniqueConstraintError" ||
            "SequelizeValidationError"
              ? "400"
              : "500",
          success: false,
          errors: err.errors ? err.errors.map((error) => error.message) : null,
          user: null,
        };
      }
    },

    login: async (_, args, context) => {
      const { email, password } = args.input;
      try {
        const result = await user.findOne({ where: { email: email } });
        if (!result) {
          return {
            code: "400",
            success: false,
            message: "This user does not exist.",
            accessToken: null,
            refreshToken: null,
            user: null,
          };
        }

        const isVerified = await bcrypt.compare(password, result.password);

        if (result && isVerified) {
          const refreshToken = generateRefreshToken(result.userId);
          const accessToken = generateAccessToken(result.userId);
          context.res.cookie("refreshToken", refreshToken, {
            httpOnly: true,
            sameSite: "lax", // in prod use "none"
            secure: process.env.NODE_ENV === "production",
            expires: new Date(Date.now() + 24 * 60 * 60 * 1000), // 1d
          });
          // convert this instance of Sequelize model into a plain object
          const { password: _, ...safeUser } = result.get({ plain: true });
          return {
            code: "200",
            success: true,
            message: `Good to see you, ${result.username}`,
            accessToken: accessToken,
            refreshToken: refreshToken,
            user: safeUser,
          };
        } else {
          return {
            code: "400",
            success: false,
            message: "Invalid credentials",
            accessToken: null,
            refreshToken: null,
            user: null,
          };
        }
      } catch (err) {
        console.log(err);
        return {
          code:
            err.name === "SequelizeUniqueConstraintError" ||
            "SequelizeValidationError"
              ? "400"
              : "500",
          success: false,
          errors: err.errors ? err.errors.map((error) => error.message) : null,
          message: err.message,
          accessToken: null,
          refreshToken: null,
          user: null,
        };
      }
    },

    renewToken: async (_, { userId }, context) => {
      const refreshToken = context.req.cookies.refreshToken;
      //console.log(refreshToken);
      console.log(process.env.REFRESH_TOKEN_SECRET);
      if (!refreshToken) {
        return {
          code: "401",
          success: false,
          message: "Invalid refresh token",
          accessToken: null,
          refreshToken: null,
          user: null,
        };
      }

      try {
        const decoded = jwt.verify(
          refreshToken,
          process.env.REFRESH_TOKEN_SECRET
        );
        if (decoded.userId !== userId) {
          return {
            code: "401",
            success: false,
            message: "Not allowed to perform this action",
            accessToken: null,
            refreshToken: null,
            user: null,
          };
        }

        const newAccess = generateAccessToken(userId);
        return {
          code: "200",
          success: true,
          message: "Token renewed",
          errors: null,
          accessToken: newAccess,
          refreshToken: null,
          user: null,
        };
      } catch (err) {
        return {
          code: "403",
          success: false,
          message: "Renew token failed",
          errors: [err.message],
          accessToken: null,
          refreshToken: null,
          user: null,
        };
      }
    },
  },

  User: {
    // map DB fields to GraphQL fields
    userId: (user) => user.userId.toString(),
    username: (user) => user.username,
    email: (user) => user.email,
    role: (user) => user.role.toUpperCase(),
  },
};

export default resolvers;

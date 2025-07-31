import { hashPassword } from "../utils/hashPassword.js";

const userModel = (sequelize, DataTypes) => {
  const User = sequelize.define(
    "User",
    {
      userId: {
        type: DataTypes.UUID,
        defaultValue: DataTypes.UUIDV4,
        primaryKey: true,
        allowNull: false,
      },
      username: {
        type: DataTypes.STRING,
        allowNull: false,
      },
      email: {
        type: DataTypes.STRING,
        allowNull: false,
        unique: true,
        validate: { isEmail: true },
      },
      password: {
        type: DataTypes.STRING,
        allowNull: false,
        validate: {
          is: {
            args: /(?=(.*[0-9]))(?=.*[\!@#$%^&*()\\[\]{}\-_+=~`|:;"'<>,./?])(?=.*[a-z])(?=(.*[A-Z]))(?=(.*)).{8,}/,
            msg: "Password should have 1 lowercase letter, 1 uppercase letter, 1 number, 1 special character and be at least 8 characters long",
          },
        },
      },
      role: {
        type: DataTypes.ENUM("MANAGER", "MEMBER"),
        allowNull: false,
      },
    },
    {
      tableName: "Users",
      timestamps: true,
    }
  );

  User.associate = (models) => {
    // Many-To-Many relationship
    User.belongsToMany(models.Team, {
      through: models.Roster,
      foreignKey: "userId",
      as: "teams",
      onDelete: "CASCADE",
    });
    // One-To-Many relationship
    User.hasMany(models.Roster, {
      foreignKey: "userId",
      as: "rosters",
    });
  };

  // hook
  User.beforeCreate(async (user, options) => {
    const hashedPassword = await hashPassword(user.password);
    user.password = hashedPassword;
  });

  return User;
};

export default userModel;
